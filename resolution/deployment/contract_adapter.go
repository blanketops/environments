package deployment

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"

// ToDeploymentContract converts a resolved runtime deployment spec into a
// canonical Deployment CONTRACT spec.
//
// ⚠️ ONE-WAY ADAPTER
// - For hashing, diffing, storage, and legacy consumers only
// - Controllers must NEVER consume the returned value
func (s *ResolvedDeploymentSpec) ToDeploymentContract() *contractv1.DeploymentSpec {
	// Absolute guard: no spec, no contract
	if s == nil {
		return nil
	}

	out := &contractv1.DeploymentSpec{
		ServiceUnits:           s.ServiceUnits,
		Runtime:                s.Runtime,
		ImageAutomation:        s.ImageAutomation,
		ReconciliationStrategy: s.ReconciliationStrategy,
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
