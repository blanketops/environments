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
Package contract implements the Deployment CR contract adapter — the one-way
projection from the resolved runtime spec (resolve.ResolvedDeploymentSpec)
to the protobuf contract type (environmentscontractv1alpha1.DeploymentSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for deployment deduplication
  - Spec comparison across GitOps reconciliation cycles
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.

Unlike the BuildTrigger adapter, enum mapping here is non-fatal — unknown
runtime and reconciliation strategy values map to UNSPECIFIED rather than
returning an error. UNSPECIFIED is a valid contract value that downstream
consumers can handle gracefully.

The common proto wrapper pattern applies: Runtime and ReconciliationStrategy
are wrapper messages (*v1.DeploymentRuntime, *v1.ReconciliationStrategy) with
the enum value carried on a named field.
*/
package contract

import (
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	environmentscontractv1alpha1 "github.com/blanketops/environments-contract/blanketops/environments/v1alpha1"

	"github.com/blanketops/environments/resolution/deployment/resolve"
)

// ToDeploymentContract projects the resolved runtime deployment spec into a
// protobuf environmentscontractv1alpha1.DeploymentSpec for infrastructure
// consumers (hashing, comparison, audit).
//
// Runtime and ReconciliationStrategy are projected as wrapper message
// instances carrying the enum value on a named field.
//
// ManifestsRepo is omitted when nil — a Deployment without a manifests repo
// has not yet been fully configured and produces a partial contract suitable
// for hashing but not for GitOps delivery.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func ToDeploymentContract(s *resolve.ResolvedDeploymentSpec) *environmentscontractv1alpha1.DeploymentSpec {
	if s == nil {
		return nil
	}

	out := &environmentscontractv1alpha1.DeploymentSpec{
		ServiceUnits:    s.ServiceUnits,
		ImageAutomation: s.ImageAutomation,
		Runtime:         toContractRuntime(s.Runtime),
		ReconciliationStrategy: toContractReconciliationStrategy(
			s.ReconciliationStrategy,
		),
	}

	// ------------------------------------------------
	// Manifests repo (GitOps delivery target).
	//
	// Omitted when nil — not all Deployments have a manifests repo
	// configured at resolution time (e.g. pre-GitOps bootstrap phase).
	// ManifestsRepo is a local message in environments v1alpha1, not a
	// common type.
	// ------------------------------------------------
	if s.ManifestsRepo != nil {
		out.ManifestsRepo = &environmentscontractv1alpha1.ManifestsRepo{
			Url:         s.ManifestsRepo.URL,
			Ref:         s.ManifestsRepo.Ref,
			CloneSecret: s.ManifestsRepo.CloneSecret,
			Strategy:    s.ManifestsRepo.Strategy,
			Path:        s.ManifestsRepo.Path,
		}
	}

	return out
}

// toContractRuntime maps a domain Runtime to the corresponding
// *v1.DeploymentRuntime wrapper message. Unknown values produce a wrapper
// with UNSPECIFIED — a valid contract value indicating the runtime has not
// been configured.
func toContractRuntime(rt resolve.Runtime) *commoncontractv1.DeploymentRuntime {
	switch rt {
	case resolve.RuntimeKubernetes:
		return &commoncontractv1.DeploymentRuntime{
			Runtime: commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER,
		}
	case resolve.RuntimeKnative:
		return &commoncontractv1.DeploymentRuntime{
			Runtime: commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE,
		}
	default:
		return &commoncontractv1.DeploymentRuntime{
			Runtime: commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_UNSPECIFIED,
		}
	}
}

// toContractReconciliationStrategy maps a domain ReconciliationStrategy to
// the corresponding *v1.ReconciliationStrategy wrapper message.
//
// ReconciliationImperative has no contract representation since it operates
// outside the GitOps delivery model — it maps to UNSPECIFIED.
func toContractReconciliationStrategy(
	rs resolve.ReconciliationStrategy,
) *commoncontractv1.ReconciliationStrategy {
	switch rs {
	case resolve.ReconciliationKustomize:
		return &commoncontractv1.ReconciliationStrategy{
			Strategy: commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_KUSTOMIZE,
		}
	case resolve.ReconciliationHelm:
		return &commoncontractv1.ReconciliationStrategy{
			Strategy: commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_HELM,
		}
	default:
		// ReconciliationImperative and unknown values — no GitOps representation.
		return &commoncontractv1.ReconciliationStrategy{
			Strategy: commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_UNSPECIFIED,
		}
	}
}
