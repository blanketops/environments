package deployment

import (
	"encoding/json"
	"fmt"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

//
// ======================================================
// DOMAIN TYPES (AUTHORITATIVE)
// ======================================================
//

type Runtime string

const (
	RuntimeKubernetes Runtime = "kubernetes.io/container-runtime"
	RuntimeKnative    Runtime = "knative.dev/service-runtime"
	RuntimeWasm       Runtime = "blanketops.dev/wasm-runtime"
	RuntimeECS        Runtime = "blanketops.dev/aws-ecs"
	RuntimeAzure      Runtime = "blanketops.dev/azure-container"
)

type Strategy string

const (
	StrategyRolling   Strategy = "Rolling"
	StrategyBlueGreen Strategy = "BlueGreen"
	StrategyCanary    Strategy = "Canary"
)

type ReconciliationStrategy string

const (
	ReconciliationImperative ReconciliationStrategy = "Imperative"
	ReconciliationKustomize  ReconciliationStrategy = "Kustomize"
	ReconciliationHelm       ReconciliationStrategy = "Helm"
)

//
// ======================================================
// RESOLVED TYPES (DOMAIN SAFE)
// ======================================================
//

type ResolvedDeployment struct {
	Deployment *environmentv1.Deployment
	Spec       *ResolvedDeploymentSpec
}

type ResolvedDeploymentSpec struct {
	ServiceUnits []string
	Runtime      Runtime
	Strategy     Strategy

	GitOwner        string
	ImageAutomation bool

	ManifestsRepo *ResolvedManifestsRepo

	ReconciliationStrategy ReconciliationStrategy
}

type ResolvedManifestsRepo struct {
	URL         string
	Ref         string
	Path        string
	Strategy    string
	CloneSecret string
}

//
// ======================================================
// RESOLUTION ENTRY POINT
// ======================================================
//

func ResolveDeployment(depl *environmentv1.Deployment) (*ResolvedDeployment, error) {

	if depl == nil {
		return nil, fmt.Errorf("deployment is nil")
	}

	if len(depl.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(depl.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode deployment contract: %w", err)
	}

	// ------------------------------------------------------------
	// ServiceUnits (MANDATORY)
	// ------------------------------------------------------------

	suRaw, ok := raw["serviceUnits"].([]any)
	if !ok || len(suRaw) == 0 {
		return nil, fmt.Errorf("serviceUnits is required and must be non-empty")
	}

	serviceUnits := make([]string, 0, len(suRaw))
	for _, v := range suRaw {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("serviceUnits entries must be non-empty strings")
		}
		serviceUnits = append(serviceUnits, s)
	}

	// ------------------------------------------------------------
	// Runtime (MANDATORY)
	// ------------------------------------------------------------

	enumRuntime, err := resolveDeploymentRuntime(raw["runtime"])
	if err != nil {
		return nil, err
	}

	domainRuntime, err := normalizeRuntime(enumRuntime)
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------------------
	// Strategy (MANDATORY)
	// ------------------------------------------------------------

	domainStrategy, err := resolveDeploymentStrategy(raw["strategy"])
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------------------
	// Optional Fields
	// ------------------------------------------------------------

	imageAutomation := optionalBool(raw, "imageAutomation")

	gitOwner := optionalString(raw, "gitOwner")
	if _, declared := raw["gitOwner"]; declared && gitOwner == "" {
		return nil, fmt.Errorf("gitOwner declared but empty")
	}

	// ------------------------------------------------------------
	// ManifestsRepo
	// ------------------------------------------------------------

	var repo *ResolvedManifestsRepo

	if mrRaw, ok := raw["manifestsRepo"].(map[string]any); ok {

		repo = &ResolvedManifestsRepo{
			URL:         mustString(mrRaw, "url"),
			Ref:         optionalString(mrRaw, "ref"),
			Path:        optionalString(mrRaw, "path"),
			Strategy:    optionalString(mrRaw, "strategy"),
			CloneSecret: optionalString(mrRaw, "cloneSecret"),
		}

		if _, declared := mrRaw["cloneSecret"]; declared && repo.CloneSecret == "" {
			return nil, fmt.Errorf("manifestsRepo.cloneSecret declared but empty")
		}
	}

	// ------------------------------------------------------------
	// Reconciliation Strategy (DOMAIN SAFE)
	// ------------------------------------------------------------

	var recon ReconciliationStrategy

	rawRecon, hasRecon := raw["reconciliationStrategy"]

	if repo != nil {

		if !hasRecon {
			return nil, fmt.Errorf("reconciliationStrategy is required when manifestsRepo is defined")
		}

		enumRecon, err := resolveReconciliationStrategy(rawRecon)
		if err != nil {
			return nil, err
		}

		recon, err = normalizeReconciliationStrategy(enumRecon)
		if err != nil {
			return nil, err
		}

	} else {

		if hasRecon {
			return nil, fmt.Errorf("reconciliationStrategy cannot be set when manifestsRepo is not defined")
		}

		recon = ReconciliationImperative
	}

	// ------------------------------------------------------------
	// Final Domain Object
	// ------------------------------------------------------------

	return &ResolvedDeployment{
		Deployment: depl,
		Spec: &ResolvedDeploymentSpec{
			ServiceUnits:           serviceUnits,
			Runtime:                domainRuntime,
			Strategy:               domainStrategy,
			GitOwner:               gitOwner,
			ImageAutomation:        imageAutomation,
			ManifestsRepo:          repo,
			ReconciliationStrategy: recon,
		},
	}, nil
}

//
// ======================================================
// NORMALIZATION HELPERS
// ======================================================
//

func resolveDeploymentRuntime(raw any) (contractv1.DeploymentRuntime, error) {
	switch v := raw.(type) {

	case string:
		switch v {
		case "kubernetes", "kubernetes.io/container-runtime":
			return contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER, nil
		case "knative":
			return contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE, nil
		default:
			return 0, fmt.Errorf("unsupported deployment.runtime %q", v)
		}

	case float64:
		return contractv1.DeploymentRuntime(v), nil

	default:
		return 0, fmt.Errorf("deployment.runtime is required")
	}
}

func normalizeRuntime(rt contractv1.DeploymentRuntime) (Runtime, error) {

	switch rt {

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER:
		return RuntimeKubernetes, nil

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE:
		return RuntimeKnative, nil

	default:
		return "", fmt.Errorf("unsupported deployment runtime enum %v", rt)
	}
}

func resolveReconciliationStrategy(raw any) (contractv1.DeploymentReconciliationStrategy, error) {

	switch v := raw.(type) {

	case string:
		switch v {
		case "kustomize":
			return contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_KUSTOMIZE, nil
		case "helm":
			return contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_HELM, nil
		default:
			return 0, fmt.Errorf("unsupported reconciliationStrategy %q", v)
		}

	case float64:
		return contractv1.DeploymentReconciliationStrategy(v), nil

	default:
		return 0, fmt.Errorf("reconciliationStrategy is required")
	}
}

func normalizeReconciliationStrategy(
	rt contractv1.DeploymentReconciliationStrategy,
) (ReconciliationStrategy, error) {

	switch rt {

	case contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_KUSTOMIZE:
		return ReconciliationKustomize, nil

	case contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_HELM:
		return ReconciliationHelm, nil

	case contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_UNSPECIFIED:
		return ReconciliationImperative, nil

	default:
		return "", fmt.Errorf("unsupported reconciliation strategy enum %v", rt)
	}
}

//
// ======================================================
// GENERIC HELPERS
// ======================================================
//

func resolveDeploymentStrategy(raw any) (Strategy, error) {

	switch v := raw.(type) {

	case string:
		switch v {
		case "Rolling":
			return StrategyRolling, nil
		case "BlueGreen":
			return StrategyBlueGreen, nil
		case "Canary":
			return StrategyCanary, nil
		default:
			return "", fmt.Errorf("unsupported deployment.strategy %q", v)
		}

	default:
		return "", fmt.Errorf("deployment.strategy is required")
	}
}

func mustString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		panic(fmt.Sprintf("missing required field %q", key))
	}
	s, ok := v.(string)
	if !ok || s == "" {
		panic(fmt.Sprintf("field %q must be a non-empty string", key))
	}
	return s
}

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func optionalBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
