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
Package build implements resolution for the Build CR.

The Build CR stores its spec as a raw JSON contract (spec.contract) rather
than typed Kubernetes fields. ResolveBuild decodes this raw contract into a
fully typed ResolvedBuild — the authoritative runtime representation consumed
by all downstream domain and application logic.

Resolution validates required fields, normalises strategy kind, and enforces
policy invariants (e.g. maxAttempts must be > 0 when retry is enabled).
All failures surface as errors — resolution never panics. A panic in the
resolution layer would crash the controller process.
*/
package build

import (
	"encoding/json"
	"fmt"

	environmentv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
//
// ResolvedBuild is the single runtime representation of a Build CR.
// All downstream domain and application logic MUST use this type.
// Never re-read from the raw CR spec after resolution.
// -----------------------------------------------------------------------------

// ResolvedBuild pairs the original Kubernetes Build object with its fully
// decoded and validated spec.
type ResolvedBuild struct {
	Build *environmentv1alpha1.Build
	Spec  *ResolvedBuildSpec
}

// ResolvedBuildSpec is the decoded and validated Build spec.
type ResolvedBuildSpec struct {
	// Image is the fully qualified target image reference (registry/repo:tag).
	Image    string
	Source   ResolvedSource
	Strategy ResolvedStrategy
	// ServiceAccount is optional — omitted when the Build uses the default
	// Shipwright service account.
	ServiceAccount *ResolvedServiceAccount
	// Policy is optional — omitted when no retry or build policy is declared.
	Policy *ResolvedBuildPolicy
}

// ResolvedSource is the resolved Git source for the Build.
type ResolvedSource struct {
	URL         string
	Revision    string
	ContextDir  string
	CloneSecret string
}

// ResolvedStrategy is the resolved Shipwright build strategy reference.
// StrategyKind is stored as the raw string from the contract ("ClusterBuildStrategy",
// "NamespacedBuildStrategy") — the contract adapter is responsible for converting
// this to the appropriate *v1.BuildStrategyKind proto message.
type ResolvedStrategy struct {
	Name         string
	StrategyKind string // raw string: "ClusterBuildStrategy" | "NamespacedBuildStrategy"
}

// ResolvedServiceAccount carries the service account name and registry
// credentials secret for Shipwright to use when pushing the built image.
type ResolvedServiceAccount struct {
	Name   string
	Secret string
}

// ResolvedBuildPolicy is the optional build execution policy.
type ResolvedBuildPolicy struct {
	Retry *ResolvedRetryPolicy
}

// ResolvedRetryPolicy controls automatic retry on build failure.
// MaxAttempts must be > 0 when OnFailure is true — resolution enforces this.
type ResolvedRetryPolicy struct {
	OnFailure   bool
	MaxAttempts uint32
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveBuild decodes and validates the raw JSON contract from the Build CR
// spec into a ResolvedBuild. Returns an error if the CR is nil, the contract
// is absent, or any required field is missing or malformed.
func ResolveBuild(build *environmentv1alpha1.Build) (*ResolvedBuild, error) {
	if build == nil {
		return nil, fmt.Errorf("build is nil")
	}

	if len(build.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(build.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Strategy (OPTIONAL — defaults to zero value).
	//
	// StrategyKind is stored as a raw string — the contract adapter converts
	// it to *v1.BuildStrategyKind when projecting to the proto contract.
	// Only ClusterBuildStrategy is currently supported.
	// ------------------------------------------------
	var strategyName string
	var strategyKind string
	if strat, ok := raw["strategy"].(map[string]any); ok {
		if n, ok := strat["name"].(string); ok {
			strategyName = n
		}
		if k, ok := strat["kind"].(string); ok {
			switch k {
			case "ClusterBuildStrategy", "NamespacedBuildStrategy":
				strategyKind = k
			default:
				return nil, fmt.Errorf(
					"unsupported strategy.kind %q (supported: ClusterBuildStrategy, NamespacedBuildStrategy)",
					k,
				)
			}
		}
	}

	// ------------------------------------------------
	// Source (REQUIRED).
	//
	// cloneSecret is validated when declared — an empty secret name after
	// declaration indicates a misconfigured CR, not an optional field.
	// ------------------------------------------------
	srcRaw, ok := raw["source"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}

	sourceURL, err := mustString(srcRaw, "url")
	if err != nil {
		return nil, fmt.Errorf("source.url: %w", err)
	}

	source := ResolvedSource{
		URL:         sourceURL,
		Revision:    optionalString(srcRaw, "revision"),
		ContextDir:  optionalString(srcRaw, "contextDir"),
		CloneSecret: optionalString(srcRaw, "cloneSecret"),
	}

	if _, declared := srcRaw["cloneSecret"]; declared && source.CloneSecret == "" {
		return nil, fmt.Errorf("source.cloneSecret declared but resolved empty")
	}

	// ------------------------------------------------
	// Image (REQUIRED).
	// ------------------------------------------------
	image, err := mustString(raw, "image")
	if err != nil {
		return nil, fmt.Errorf("image: %w", err)
	}

	// ------------------------------------------------
	// Service account (OPTIONAL).
	// ------------------------------------------------
	var sa *ResolvedServiceAccount
	if saRaw, ok := raw["serviceAccount"].(map[string]any); ok {
		sa = &ResolvedServiceAccount{
			Name:   optionalString(saRaw, "name"),
			Secret: optionalString(saRaw, "secret"),
		}
	}

	// ------------------------------------------------
	// Policy (OPTIONAL).
	//
	// Hard invariant: maxAttempts must be > 0 when onFailure=true.
	// A retry policy with zero attempts is a misconfiguration that would
	// cause the retry observer to loop indefinitely.
	// ------------------------------------------------
	var policy *ResolvedBuildPolicy
	if polRaw, ok := raw["policy"].(map[string]any); ok {
		policy = &ResolvedBuildPolicy{}
		if retryRaw, ok := polRaw["retry"].(map[string]any); ok {
			retry := &ResolvedRetryPolicy{
				OnFailure:   optionalBool(retryRaw, "onFailure"),
				MaxAttempts: optionalUint32(retryRaw, "maxAttempts"),
			}
			if retry.OnFailure && retry.MaxAttempts == 0 {
				return nil, fmt.Errorf("policy.retry.maxAttempts must be > 0 when onFailure=true")
			}
			policy.Retry = retry
		}
	}

	return &ResolvedBuild{
		Build: build,
		Spec: &ResolvedBuildSpec{
			Image:  image,
			Source: source,
			Strategy: ResolvedStrategy{
				Name:         strategyName,
				StrategyKind: strategyKind,
			},
			ServiceAccount: sa,
			Policy:         policy,
		},
	}, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
//
// mustString returns an error rather than panicking — resolution must never
// crash the controller process. All required field failures are surfaced as
// errors that the domain layer records as conditions on the CR.
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

func optionalBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// optionalUint32 extracts a uint32 from m[key]. JSON numbers decode as
// float64 by default — both float64 and int are handled to support callers
// that pre-parse the JSON differently.
func optionalUint32(m map[string]any, key string) uint32 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return uint32(n)
		case int:
			return uint32(n)
		}
	}
	return 0
}
