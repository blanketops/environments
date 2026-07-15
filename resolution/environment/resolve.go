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
Package environment implements resolution for the Environment CR.

The Environment CR stores its spec as a raw JSON contract (spec.contract).
ResolveEnvironment decodes this into a fully typed ResolvedEnvironment —
the authoritative runtime representation consumed by all downstream domain
and application logic.

Resolution is nil-tolerant for the CR itself but strict on the contract
payload — a missing or malformed contract is always an error. All failures
surface as errors. Resolution never panics.
*/
package environment

import (
	"encoding/json"
	"fmt"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedEnvironment pairs the original Kubernetes Environment object with
// its fully decoded and validated spec.
type ResolvedEnvironment struct {
	Environment *environmentv1alpha1.Environment
	Spec        *ResolvedEnvironmentSpec
}

// ResolvedEnvironmentSpec is the decoded Environment spec.
// All CR references are stored as name strings — cross-CR lookups are the
// responsibility of the domain layer.
type ResolvedEnvironmentSpec struct {
	ApplicationName string
	Branch          string
	GitOwner        string
	EnvironmentType string
	Version         string
	Description     string

	// Optional CR references — empty string means not declared.
	Build         string
	GitRepository string
	GitHubEvent   string
	Deployment    string
	Route         string
	Package       string

	// Slices are nil when not declared; empty slice is never returned.
	ServiceUnits []string

	// Platform-level bindings.
	Contract *ResolvedEnvironmentContract
}

// ResolvedEnvironmentContract holds platform-level bindings declared on the
// Environment — currently the ESO secret store provider.
type ResolvedEnvironmentContract struct {
	SecretStoreProvider string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveEnvironment decodes and validates the raw JSON contract from the
// Environment CR spec into a ResolvedEnvironment.
func ResolveEnvironment(environment *environmentv1alpha1.Environment) (*ResolvedEnvironment, error) {
	if environment == nil {
		return &ResolvedEnvironment{}, nil
	}

	if len(environment.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(environment.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Required scalar fields.
	// ------------------------------------------------
	appName, err := mustString(raw, "applicationName")
	if err != nil {
		return nil, err
	}

	branch, err := mustString(raw, "branch")
	if err != nil {
		return nil, err
	}

	gitOwner, err := mustString(raw, "gitOwner")
	if err != nil {
		return nil, err
	}

	envType, err := mustString(raw, "environmentType")
	if err != nil {
		return nil, err
	}

	version, err := mustString(raw, "version")
	if err != nil {
		return nil, err
	}

	spec := &ResolvedEnvironmentSpec{
		ApplicationName: appName,
		Branch:          branch,
		GitOwner:        gitOwner,
		EnvironmentType: envType,
		Version:         version,
		Description:     optionalString(raw, "description"),
	}

	// ------------------------------------------------
	// Optional CR reference: build.
	// ------------------------------------------------
	if v, ok := raw["build"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("build.name: %w", err)
		}
		spec.Build = n
	}

	// ------------------------------------------------
	// Optional CR reference: gitRepository.
	// The GitRepository CR owns branch, owner, and clone config.
	// ------------------------------------------------
	if v, ok := raw["gitRepository"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("gitRepository.name: %w", err)
		}
		spec.GitRepository = n
	}

	// ------------------------------------------------
	// Optional CR reference: gitHubEvent.
	// The GitRepository CR owns branch, owner, and clone config.
	// ------------------------------------------------
	if v, ok := raw["gitHubEvent"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("gitHubEvent.name: %w", err)
		}
		spec.GitHubEvent = n
	}

	// ------------------------------------------------
	// Optional CR reference list: serviceUnits.
	// ------------------------------------------------
	if list, ok := raw["serviceUnits"].([]any); ok {
		for i, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("serviceUnits[%d] must be an object", i)
			}
			n, err := mustString(m, "name")
			if err != nil {
				return nil, fmt.Errorf("serviceUnits[%d].name: %w", i, err)
			}
			spec.ServiceUnits = append(spec.ServiceUnits, n)
		}
	}

	// ------------------------------------------------
	// Optional CR references: deployment, route, package.
	// ------------------------------------------------
	if v, ok := raw["deployment"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("deployment.name: %w", err)
		}
		spec.Deployment = n
	}

	if v, ok := raw["route"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("route.name: %w", err)
		}
		spec.Route = n
	}

	if v, ok := raw["domain"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("domain.name: %w", err)
		}
		spec.Route = n
	}

	if v, ok := raw["package"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("package.name: %w", err)
		}
		spec.Package = n
	}

	// ------------------------------------------------
	// Optional platform contract: secretStore provider.
	// ------------------------------------------------
	// ------------------------------------------------
	// Platform contract: secretStore is required when contract is declared.
	// ------------------------------------------------
	if contractRaw, ok := raw["contract"].(map[string]any); ok {
		ssRaw, ok := contractRaw["secretStore"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("contract.secretStore is required")
		}
		provider, err := mustString(ssRaw, "provider")
		if err != nil {
			return nil, fmt.Errorf("contract.secretStore.provider: %w", err)
		}
		spec.Contract = &ResolvedEnvironmentContract{
			SecretStoreProvider: provider,
		}
	}

	return &ResolvedEnvironment{
		Environment: environment,
		Spec:        spec,
	}, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
// -----------------------------------------------------------------------------

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
