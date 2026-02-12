package environment

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"

// ToEnvironmentContract converts a resolved runtime spec into a CONTRACT spec
// for legacy / infra-only consumers (hashing, comparison, etc).
//
// ⚠️ This is a ONE-WAY adapter.
// Controllers must NEVER consume the returned value.
func (s *ResolvedEnvironmentSpec) ToEnvironmentContract() *contractv1.EnvironmentSpec {
	// Absolute guard: no spec, no contract
	if s == nil {
		return nil
	}

	spec := &contractv1.EnvironmentSpec{
		ApplicationName: s.ApplicationName,
		Branch:          s.Branch,
		GitOwner:        s.GitOwner,
		Version:         s.Version,
		Description:     s.Description,
	}

	// -------------------------------------------------------------
	// Environment type (string → enum)
	// -------------------------------------------------------------
	switch s.EnvironmentType {
	case "ENVIRONMENT_TYPE_DEVELOPMENT":
		spec.EnvironmentType = contractv1.EnvironmentType_ENVIRONMENT_TYPE_DEVELOPMENT
	case "ENVIRONMENT_TYPE_STAGING":
		spec.EnvironmentType = contractv1.EnvironmentType_ENVIRONMENT_TYPE_STAGING
	case "ENVIRONMENT_TYPE_PRODUCTION":
		spec.EnvironmentType = contractv1.EnvironmentType_ENVIRONMENT_TYPE_PRODUCTION
	case "ENVIRONMENT_TYPE_TESTING":
		spec.EnvironmentType = contractv1.EnvironmentType_ENVIRONMENT_TYPE_TESTING
	default:
		spec.EnvironmentType = contractv1.EnvironmentType_ENVIRONMENT_TYPE_UNSPECIFIED
	}

	// -------------------------------------------------------------
	// Build
	// -------------------------------------------------------------
	if s.Build != "" {
		spec.Build = &contractv1.EnvironmentBuildRef{
			Name: s.Build,
		}
	}

	// -------------------------------------------------------------
	// Build Triggers
	// -------------------------------------------------------------
	for _, name := range s.BuildTriggers {
		spec.BuildTriggers = append(spec.BuildTriggers,
			&contractv1.EnvironmentBuildTriggerRef{
				Name: name,
			},
		)
	}

	// -------------------------------------------------------------
	// Service Units
	// -------------------------------------------------------------
	for _, name := range s.ServiceUnits {
		spec.ServiceUnits = append(spec.ServiceUnits,
			&contractv1.EnvironmentServiceUnitRef{
				Name: name,
			},
		)
	}

	// -------------------------------------------------------------
	// Deployment
	// -------------------------------------------------------------
	if s.Deployment != "" {
		spec.Deployment = &contractv1.EnvironmentDeploymentRef{
			Name: s.Deployment,
		}
	}

	// -------------------------------------------------------------
	// Route
	// -------------------------------------------------------------
	if s.Route != "" {
		spec.Route = &contractv1.EnvironmentRouteRef{
			Name: s.Route,
		}
	}

	// -------------------------------------------------------------
	// Package
	// -------------------------------------------------------------
	if s.Package != "" {
		spec.Package = &contractv1.EnvironmentPackageRef{
			Name: s.Package,
		}
	}

	return spec
}
