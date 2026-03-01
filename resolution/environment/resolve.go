package environment

import (
	"encoding/json"
	"fmt"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

//
// ==============================
// RUNTIME Environment (AUTHORITATIVE)
// ==============================
//

// ResolvedEnvironment is a runtime-safe interpretation of an Environment CR.
// It is intentionally defensive and may represent partial or empty state.
type ResolvedEnvironment struct {
	Environment *environmentv1.Environment
	Spec        *ResolvedEnvironmentSpec
}

type ResolvedEnvironmentSpec struct {
	ApplicationName string
	Branch          string
	GitOwner        string
	EnvironmentType string
	Version         string
	Description     string

	Build         string
	BuildTriggers []string
	ServiceUnits  []string
	Deployment    string
	Route         string
	Package       string
}

func ResolveEnvironment(environment *environmentv1.Environment) (*ResolvedEnvironment, error) {
	// -----------------------------------------------------------------
	// Nil-tolerant by design
	// -----------------------------------------------------------------
	if environment == nil {
		return &ResolvedEnvironment{
			Environment: nil,
			Spec:        nil,
		}, nil
	}

	if len(environment.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// -----------------------------------------------------------------
	// Decode raw contract
	// -----------------------------------------------------------------
	var raw map[string]any
	if err := json.Unmarshal(environment.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	spec := &ResolvedEnvironmentSpec{
		ApplicationName: mustString(raw, "applicationName"),
		Branch:          mustString(raw, "branch"),
		GitOwner:        mustString(raw, "gitOwner"),
		EnvironmentType: mustString(raw, "environmentType"),
		Version:         mustString(raw, "version"),
		Description:     optionalString(raw, "description"),
	}

	// -----------------------------------------------------------------
	// build
	// -----------------------------------------------------------------
	if v, ok := raw["build"].(map[string]any); ok {
		spec.Build = mustString(v, "name")
	}

	// -----------------------------------------------------------------
	// buildTriggers
	// -----------------------------------------------------------------
	if list, ok := raw["buildTriggers"].([]any); ok {
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("buildTriggers entries must be objects")
			}
			spec.BuildTriggers = append(spec.BuildTriggers, mustString(m, "name"))
		}
	}

	// -----------------------------------------------------------------
	// serviceUnits
	// -----------------------------------------------------------------
	if list, ok := raw["serviceUnits"].([]any); ok {
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("serviceUnits entries must be objects")
			}
			spec.ServiceUnits = append(spec.ServiceUnits, mustString(m, "name"))
		}
	}

	// -----------------------------------------------------------------
	// deployment
	// -----------------------------------------------------------------
	if v, ok := raw["deployment"].(map[string]any); ok {
		spec.Deployment = mustString(v, "name")
	}

	// -----------------------------------------------------------------
	// route
	// -----------------------------------------------------------------
	if v, ok := raw["route"].(map[string]any); ok {
		spec.Route = mustString(v, "name")
	}

	// -----------------------------------------------------------------
	// package
	// -----------------------------------------------------------------
	if v, ok := raw["package"].(map[string]any); ok {
		spec.Package = mustString(v, "name")
	}

	return &ResolvedEnvironment{
		Environment: environment,
		Spec:        spec,
	}, nil
}

//
// ==============================
// HELPERS
// ==============================
//

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
