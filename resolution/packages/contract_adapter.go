package packages

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"

// ToPackageContract converts a resolved runtime spec into a CONTRACT spec
// for legacy / infra-only consumers (hashing, diffing, comparison, etc).
//
// ⚠️ This is a ONE-WAY adapter.
// Controllers must NEVER consume the returned value.
func (s *ResolvedPackageSpec) ToPackageContract() *contractv1.PackageSpec {
	// Absolute guard: no spec, no contract
	if s == nil {
		return nil
	}

	out := &contractv1.PackageSpec{
		Enabled:     s.Enabled,
		Name:        s.Name,
		Version:     s.Version,
		Description: s.Description,
		DiffEnabled: s.DiffEnabled,
	}

	// ------------------------------------------------
	// Maintainers
	// ------------------------------------------------
	if len(s.Maintainers) > 0 {
		out.Maintainers = make([]*contractv1.Maintainer, 0, len(s.Maintainers))
		for _, m := range s.Maintainers {
			out.Maintainers = append(out.Maintainers, &contractv1.Maintainer{
				Name:  m.Name,
				Email: m.Email,
			})
		}
	}

	// ------------------------------------------------
	// Source repository (package definitions)
	// ------------------------------------------------
	if s.PackageRepository.URL != "" {
		out.Repository = &contractv1.PackageRepository{
			Url:               s.PackageRepository.URL,
			CredentialsSecret: s.PackageRepository.CredentialsSecret,
		}
	}

	// ------------------------------------------------
	// State repository (GitOps anchor)
	// ------------------------------------------------
	if s.StateRepository != nil && s.StateRepository.URL != "" {
		out.StateRepository = &contractv1.StateRepository{
			Url:         s.StateRepository.URL,
			Path:        s.StateRepository.Path,
			Strategy:    s.StateRepository.Strategy,
			CloneSecret: s.StateRepository.CloneSecret,
			//Ref:         s.StateRepository.Ref.Branch, // see note below
		}
	}

	return out
}
