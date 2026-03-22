package deployment

import (
	"encoding/json"
	"fmt"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

//
// ==============================
// RUNTIME DEPLOYMENT (AUTHORITATIVE)
// ==============================
//

type ResolvedDeployment struct {
	Deployment *environmentv1.Deployment
	Spec       *ResolvedDeploymentSpec
}

type ResolvedDeploymentSpec struct {
	ServiceUnits []string

	Runtime contractv1.DeploymentRuntime

	GitOwner string

	ImageAutomation bool

	ManifestsRepo *ResolvedManifestsRepo

	ReconciliationStrategy contractv1.DeploymentReconciliationStrategy
}

type ResolvedManifestsRepo struct {
	URL         string
	Ref         string
	Path        string
	Strategy    string
	CloneSecret string
}

//
// ==============================
// RESOLUTION ENTRY POINT
// ==============================
//

func ResolveDeployment(depl *environmentv1.Deployment) (*ResolvedDeployment, error) {
	if depl == nil {
		return nil, fmt.Errorf("deployment is nil")
	}
	if len(depl.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// ------------------------------------------------------------
	// Decode RAW contract (DO NOT USE PROTO)
	// ------------------------------------------------------------
	var raw map[string]any
	if err := json.Unmarshal(depl.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode deployment contract: %w", err)
	}

	// ------------------------------------------------------------
	// Resolve serviceUnits (MANDATORY)
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
	// Resolve runtime (MANDATORY, NORMALIZED)
	// ------------------------------------------------------------
	runtime, err := resolveDeploymentRuntime(raw["runtime"])
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------------------
	// Resolve imageAutomation (OPTIONAL)
	// ------------------------------------------------------------
	imageAutomation := optionalBool(raw, "imageAutomation")

	// ------------------------------------------------------------
	// Resolve gitOwner (OPTIONAL but validated if present)
	// ------------------------------------------------------------
	// gitOwner := optionalString(raw, "gitOwner")
	// if _, declared := raw["gitOwner"]; declared && gitOwner == "" {
	// 	return nil, fmt.Errorf("gitOwner declared but empty")
	// }

	// ------------------------------------------------------------
	// Resolve reconciliationStrategy (MANDATORY)
	// ------------------------------------------------------------
	recon, err := resolveReconciliationStrategy(raw["reconciliationStrategy"])
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------------------
	// Resolve manifestsRepo (OPTIONAL)
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

	return &ResolvedDeployment{
		Deployment: depl,
		Spec: &ResolvedDeploymentSpec{
			ServiceUnits: serviceUnits,
			Runtime:      runtime,
			//GitOwner:               gitOwner,
			ImageAutomation:        imageAutomation,
			ManifestsRepo:          repo,
			ReconciliationStrategy: recon,
		},
	}, nil
}

//
// ==============================
// HELPERS (SAME PHILOSOPHY AS BUILD)
// ==============================
//

func resolveDeploymentRuntime(raw any) (contractv1.DeploymentRuntime, error) {
	switch v := raw.(type) {

	case string:
		switch v {

		// canonical
		case "kubernetes":
			return contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER, nil

		// aliases (🔥 intentional)
		case "kubernetes.io/container-runtime":
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
