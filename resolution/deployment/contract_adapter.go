package deployment

import (
	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

func (s *ResolvedDeploymentSpec) ToDeploymentContract() *contractv1.DeploymentSpec {

	if s == nil {
		return nil
	}

	out := &contractv1.DeploymentSpec{
		ServiceUnits:    s.ServiceUnits,
		ImageAutomation: s.ImageAutomation,
		Runtime:         toContractRuntime(s.Runtime),
		ReconciliationStrategy: toContractReconciliationStrategy(
			s.ReconciliationStrategy,
		),
	}

	// ------------------------------------------------
	// Manifests Repo (GitOps)
	// ------------------------------------------------
	if s.ManifestsRepo != nil {
		out.ManifestsRepo = &contractv1.DeploymentManifestsRepo{
			Url:         s.ManifestsRepo.URL,
			Ref:         s.ManifestsRepo.Ref,
			Path:        s.ManifestsRepo.Path,
			Strategy:    s.ManifestsRepo.Strategy,
			CloneSecret: s.ManifestsRepo.CloneSecret,
		}
	}

	return out
}

func toContractRuntime(rt Runtime) contractv1.DeploymentRuntime {

	switch rt {

	case RuntimeKubernetes:
		return contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER

	case RuntimeKnative:
		return contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE

	default:
		return contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_UNSPECIFIED
	}
}

func toContractReconciliationStrategy(
	rs ReconciliationStrategy,
) contractv1.DeploymentReconciliationStrategy {

	switch rs {

	case ReconciliationKustomize:
		return contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_KUSTOMIZE

	case ReconciliationHelm:
		return contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_HELM

	case ReconciliationImperative:
		return contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_UNSPECIFIED

	default:
		return contractv1.DeploymentReconciliationStrategy_DEPLOYMENT_RECONCILIATION_STRATEGY_UNSPECIFIED
	}
}
