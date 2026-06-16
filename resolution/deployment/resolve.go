/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package deployment implements resolution for the Deployment CR.

The Deployment CR stores its spec as a raw JSON contract (spec.contract).
ResolveDeployment decodes this into a fully typed ResolvedDeployment — the
authoritative runtime representation consumed by all downstream domain and
application logic.

Resolution is strict: required fields that are missing or malformed return
errors immediately. All failures surface as errors — resolution never panics.
*/
package deployment

import (
	"encoding/json"
	"fmt"

	environmentv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/common/v1"
)

// -----------------------------------------------------------------------------
// Domain types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Resolved types (DOMAIN SAFE)
// -----------------------------------------------------------------------------

// ResolvedDeployment pairs the original Kubernetes Deployment object with
// its fully decoded and validated spec.
type ResolvedDeployment struct {
	Deployment *environmentv1alpha1.Deployment
	Spec       *ResolvedDeploymentSpec
}

// ResolvedDeploymentSpec is the decoded and validated Deployment spec.
type ResolvedDeploymentSpec struct {
	ServiceUnits           []string
	Runtime                Runtime
	Strategy               Strategy
	GitOwner               string
	ImageAutomation        bool
	ManifestsRepo          *ResolvedManifestsRepo
	ReconciliationStrategy ReconciliationStrategy
}

// ResolvedManifestsRepo is the resolved GitOps manifests repository.
type ResolvedManifestsRepo struct {
	URL         string
	Ref         string
	Path        string
	Strategy    string
	CloneSecret string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveDeployment decodes and validates the raw JSON contract from the
// Deployment CR spec into a ResolvedDeployment. Returns an error if the CR
// is nil, the contract is absent, or any required field is missing or malformed.
func ResolveDeployment(depl *environmentv1alpha1.Deployment) (*ResolvedDeployment, error) {
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

	// ------------------------------------------------
	// ServiceUnits (REQUIRED).
	// ------------------------------------------------
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

	// ------------------------------------------------
	// Runtime (REQUIRED).
	//
	// Resolved via the common contract enum then normalised to the
	// domain Runtime constant. The contract import is used only for
	// enum normalisation — the domain type is authoritative downstream.
	// ------------------------------------------------
	enumRuntime, err := resolveDeploymentRuntime(raw["runtime"])
	if err != nil {
		return nil, err
	}

	domainRuntime, err := normalizeRuntime(enumRuntime)
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------
	// Strategy (REQUIRED).
	// ------------------------------------------------
	domainStrategy, err := resolveDeploymentStrategy(raw["strategy"])
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------
	// Optional fields.
	// ------------------------------------------------
	imageAutomation := optionalBool(raw, "imageAutomation")

	gitOwner := optionalString(raw, "gitOwner")
	if _, declared := raw["gitOwner"]; declared && gitOwner == "" {
		return nil, fmt.Errorf("gitOwner declared but empty")
	}

	// ------------------------------------------------
	// ManifestsRepo (OPTIONAL).
	// ------------------------------------------------
	var repo *ResolvedManifestsRepo
	if mrRaw, ok := raw["manifestsRepo"].(map[string]any); ok {
		url, err := mustString(mrRaw, "url")
		if err != nil {
			return nil, fmt.Errorf("manifestsRepo.url: %w", err)
		}
		repo = &ResolvedManifestsRepo{
			URL:         url,
			Ref:         optionalString(mrRaw, "ref"),
			Path:        optionalString(mrRaw, "path"),
			Strategy:    optionalString(mrRaw, "strategy"),
			CloneSecret: optionalString(mrRaw, "cloneSecret"),
		}
		if _, declared := mrRaw["cloneSecret"]; declared && repo.CloneSecret == "" {
			return nil, fmt.Errorf("manifestsRepo.cloneSecret declared but empty")
		}
	}

	// ------------------------------------------------
	// ReconciliationStrategy.
	//
	// Required when manifestsRepo is set; must be absent otherwise.
	// Defaults to Imperative when no manifests repo is configured.
	// ------------------------------------------------
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

// -----------------------------------------------------------------------------
// Normalisation helpers
//
// Runtime and ReconciliationStrategy use the common contract enum types
// (commoncontractv1.DeploymentRuntime_DeploymentRuntime etc.) as the
// intermediate step before mapping to domain constants. This matches the
// generated proto pattern where enums are nested inside wrapper messages.
// -----------------------------------------------------------------------------

func resolveDeploymentRuntime(raw any) (commoncontractv1.DeploymentRuntime_DeploymentRuntime, error) {
	switch v := raw.(type) {
	case string:
		switch v {
		case "kubernetes", "kubernetes.io/container-runtime":
			return commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER, nil
		case "knative":
			return commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE, nil
		default:
			return 0, fmt.Errorf("unsupported deployment.runtime %q", v)
		}
	case float64:
		return commoncontractv1.DeploymentRuntime_DeploymentRuntime(v), nil
	default:
		return 0, fmt.Errorf("deployment.runtime is required")
	}
}

func normalizeRuntime(rt commoncontractv1.DeploymentRuntime_DeploymentRuntime) (Runtime, error) {
	switch rt {
	case commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER:
		return RuntimeKubernetes, nil
	case commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE:
		return RuntimeKnative, nil
	default:
		return "", fmt.Errorf("unsupported deployment runtime enum %v", rt)
	}
}

func resolveReconciliationStrategy(raw any) (commoncontractv1.ReconciliationStrategy_ReconciliationStrategy, error) {
	switch v := raw.(type) {
	case string:
		switch v {
		case "kustomize":
			return commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_KUSTOMIZE, nil
		case "helm":
			return commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_HELM, nil
		default:
			return 0, fmt.Errorf("unsupported reconciliationStrategy %q", v)
		}
	case float64:
		return commoncontractv1.ReconciliationStrategy_ReconciliationStrategy(v), nil
	default:
		return 0, fmt.Errorf("reconciliationStrategy is required")
	}
}

func normalizeReconciliationStrategy(rt commoncontractv1.ReconciliationStrategy_ReconciliationStrategy) (ReconciliationStrategy, error) {
	switch rt {
	case commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_KUSTOMIZE:
		return ReconciliationKustomize, nil
	case commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_HELM:
		return ReconciliationHelm, nil
	case commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_UNSPECIFIED:
		return ReconciliationImperative, nil
	default:
		return "", fmt.Errorf("unsupported reconciliation strategy enum %v", rt)
	}
}

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

// -----------------------------------------------------------------------------
// Extraction helpers — no panics
// -----------------------------------------------------------------------------

// mustString extracts a non-empty string from m[key].
// Returns an error if the key is absent or the value is not a non-empty string.
func mustString(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("field %q must be a non-empty string", key)
	}
	return s, nil
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
