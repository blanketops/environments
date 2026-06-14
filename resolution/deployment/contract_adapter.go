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

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedDeploymentSpec) to the protobuf contract type
(contractv1.DeploymentSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for deployment deduplication
  - Spec comparison across GitOps reconciliation cycles
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.
The resolved runtime spec is the authoritative type for all reconciliation
decisions.

Unlike the BuildTrigger adapter, enum mapping here is non-fatal — unknown
runtime and reconciliation strategy values map to UNSPECIFIED rather than
returning an error. UNSPECIFIED is a valid contract value that downstream
consumers can handle gracefully. Unknown enums here indicate a configuration
gap, not a version skew requiring immediate failure.
*/
package deployment

import (
	contractv1alpha1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

// ToDeploymentContract projects the resolved runtime deployment spec into a
// protobuf contractv1.DeploymentSpec for infrastructure consumers
// (hashing, comparison, audit).
//
// ManifestsRepo is omitted from the output when nil — a Deployment without
// a manifests repo has not yet been fully configured and produces a partial
// contract suitable for hashing but not for GitOps delivery.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func (s *ResolvedDeploymentSpec) ToDeploymentContract() *contractv1alpha1.DeploymentSpec {
	if s == nil {
		return nil
	}

	out := &contractv1alpha1.DeploymentSpec{
		ServiceUnits:           s.ServiceUnits,
		ImageAutomation:        s.ImageAutomation,
		Runtime:                toContractRuntime(s.Runtime),
		ReconciliationStrategy: toContractReconciliationStrategy(s.ReconciliationStrategy),
	}

	// ------------------------------------------------
	// Manifests repo (GitOps delivery target).
	//
	// Omitted when nil — not all Deployments have a manifests repo
	// configured at resolution time (e.g. pre-GitOps bootstrap phase).
	// ------------------------------------------------
	if s.ManifestsRepo != nil {
		out.ManifestsRepo = &contractv1alpha1.DeploymentManifestsRepo{
			Url:         s.ManifestsRepo.URL,
			Ref:         s.ManifestsRepo.Ref,
			Path:        s.ManifestsRepo.Path,
			Strategy:    s.ManifestsRepo.Strategy,
			CloneSecret: s.ManifestsRepo.CloneSecret,
		}
	}

	return out
}

// toContractRuntime maps a domain Runtime to the corresponding contract enum.
// Unknown values map to UNSPECIFIED — a valid contract value indicating the
// runtime has not been configured rather than an error condition.
func toContractRuntime(rt Runtime) contractv1alpha1.DeploymentRuntime {
	switch rt {
	case RuntimeKubernetes:
		return contractv1alpha1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER
	case RuntimeKnative:
		return contractv1alpha1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE
	default:
		return contractv1alpha1.DeploymentRuntime_DEPLOYMENT_RUNTIME_UNSPECIFIED
	}
}

// toContractReconciliationStrategy maps a domain ReconciliationStrategy to
// the corresponding contract enum. Unknown values and ReconciliationImperative
// both map to UNSPECIFIED — imperative reconciliation has no contract
// representation since it operates outside the GitOps delivery model.
func toContractReconciliationStrategy(rs ReconciliationStrategy) contractv1alpha1.DeploymentReconciliationStrategy {
	switch rs {
	case ReconciliationKustomize:
		return contractv1alpha1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_KUSTOMIZE
	case ReconciliationHelm:
		return contractv1alpha1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_HELM
	case ReconciliationImperative:
		// Imperative reconciliation operates outside the GitOps delivery model
		// and has no contract representation. UNSPECIFIED is the correct signal.
		return contractv1alpha1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_UNSPECIFIED
	default:
		return contractv1alpha1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_UNSPECIFIED
	}
}
