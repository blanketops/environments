package build

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"

// ToBuildContract converts a resolved runtime spec into a CONTRACT spec
// for legacy / infra-only consumers (hashing, comparison, etc).
//
// ⚠️ This is a ONE-WAY adapter.
// Controllers must NEVER consume the returned value.
func (s *ResolvedBuildSpec) ToBuildContract() *contractv1.BuildSpec {
	// Absolute guard: no spec, no contract
	if s == nil {
		return nil
	}

	out := &contractv1.BuildSpec{
		Image: s.Image,
	}

	// ------------------------------------------------
	// Source (Git)
	// ------------------------------------------------
	if s.Source.URL != "" {
		out.Source = &contractv1.GitSource{
			Url:         s.Source.URL,
			Revision:    s.Source.Revision,
			ContextDir:  s.Source.ContextDir,
			CloneSecret: s.Source.CloneSecret,
		}
	}

	// ------------------------------------------------
	// Strategy
	// ------------------------------------------------
	if s.Strategy.Name != "" {
		out.Strategy = &contractv1.BuildStrategy{
			Name: s.Strategy.Name,
			Kind: s.Strategy.Kind,
		}
	}

	// ------------------------------------------------
	// Service Account
	// ------------------------------------------------
	if s.ServiceAccount != nil {
		out.ServiceAccount = &contractv1.ServiceAccount{
			Name:   s.ServiceAccount.Name,
			Secret: s.ServiceAccount.Secret,
		}
	}

	// ------------------------------------------------
	// Build policy (retry)
	// ------------------------------------------------
	if s.Policy != nil && s.Policy.Retry != nil {
		out.Policy = &contractv1.BuildPolicy{
			Retry: &contractv1.RetryPolicy{
				OnFailure:   s.Policy.Retry.OnFailure,
				MaxAttempts: uint32(s.Policy.Retry.MaxAttempts),
			},
		}
	}

	return out
}
