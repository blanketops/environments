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

package build

import (
	"encoding/json"
	"fmt"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

//
// ==============================
// RUNTIME BUILD (AUTHORITATIVE)
// ==============================
//

type ResolvedBuild struct {
	Build *environmentv1.Build
	Spec  *ResolvedBuildSpec
}

type ResolvedBuildSpec struct {
	Image string

	Source   ResolvedSource
	Strategy ResolvedStrategy

	ServiceAccount *ResolvedServiceAccount
	Policy         *ResolvedBuildPolicy
}

type ResolvedSource struct {
	URL         string
	Revision    string
	ContextDir  string
	CloneSecret string
}

type ResolvedStrategy struct {
	Name string
	Kind contractv1.BuildStrategy_Kind
}

type ResolvedServiceAccount struct {
	Name   string
	Secret string
}

//
// ==============================
// POLICY (NEW)
// ==============================
//

type ResolvedBuildPolicy struct {
	Retry *ResolvedRetryPolicy
}

type ResolvedRetryPolicy struct {
	OnFailure   bool
	MaxAttempts uint32
}

//
// ==============================
// RESOLUTION ENTRY POINT
// ==============================
//

func ResolveBuild(build *environmentv1.Build) (*ResolvedBuild, error) {
	if build == nil {
		return nil, fmt.Errorf("build is nil")
	}
	if len(build.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// -----------------------------------------------------------------
	// Decode raw contract
	// -----------------------------------------------------------------
	var raw map[string]any
	if err := json.Unmarshal(build.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// -----------------------------------------------------------------
	// Normalize strategy.kind
	// -----------------------------------------------------------------
	var strategyKind contractv1.BuildStrategy_Kind
	var strategyName string

	if strat, ok := raw["strategy"].(map[string]any); ok {
		if rawKind, ok := strat["kind"]; ok {
			kind, err := resolveStrategyKind(rawKind)
			if err != nil {
				return nil, err
			}
			strategyKind = kind
		}
		if n, ok := strat["name"].(string); ok {
			strategyName = n
		}
	}

	// -----------------------------------------------------------------
	// Resolve source (MANDATORY)
	// -----------------------------------------------------------------
	srcRaw, ok := raw["source"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}

	source := ResolvedSource{
		URL:         mustString(srcRaw, "url"),
		Revision:    optionalString(srcRaw, "revision"),
		ContextDir:  optionalString(srcRaw, "contextDir"),
		CloneSecret: optionalString(srcRaw, "cloneSecret"),
	}

	if _, declared := srcRaw["cloneSecret"]; declared && source.CloneSecret == "" {
		return nil, fmt.Errorf("source.cloneSecret declared but resolved empty")
	}

	// -----------------------------------------------------------------
	// Resolve image (MANDATORY)
	// -----------------------------------------------------------------
	image := mustString(raw, "image")

	// -----------------------------------------------------------------
	// Resolve serviceAccount (OPTIONAL)
	// -----------------------------------------------------------------
	var sa *ResolvedServiceAccount
	if saRaw, ok := raw["serviceAccount"].(map[string]any); ok {
		sa = &ResolvedServiceAccount{
			Name:   optionalString(saRaw, "name"),
			Secret: optionalString(saRaw, "secret"),
		}
	}

	// -----------------------------------------------------------------
	// Resolve policy (OPTIONAL)
	// -----------------------------------------------------------------
	var policy *ResolvedBuildPolicy
	if polRaw, ok := raw["policy"].(map[string]any); ok {
		policy = &ResolvedBuildPolicy{}

		if retryRaw, ok := polRaw["retry"].(map[string]any); ok {
			retry := &ResolvedRetryPolicy{
				OnFailure:   optionalBool(retryRaw, "onFailure"),
				MaxAttempts: optionalUint32(retryRaw, "maxAttempts"),
			}

			// 🔥 HARD FAILS
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
				Name: strategyName,
				Kind: strategyKind,
			},
			ServiceAccount: sa,
			Policy:         policy,
		},
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

func optionalBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

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

//
// ==============================
// STRATEGY KIND NORMALIZATION
// ==============================
//

func resolveStrategyKind(raw any) (contractv1.BuildStrategy_Kind, error) {
	switch v := raw.(type) {

	case string:
		switch v {
		case "ClusterBuildStrategy":
			return contractv1.BuildStrategy_KIND_CLUSTER, nil
		default:
			return 0, fmt.Errorf(
				"unsupported strategy.kind %q (only ClusterBuildStrategy supported)",
				v,
			)
		}

	case float64:
		return contractv1.BuildStrategy_Kind(v), nil

	default:
		return 0, fmt.Errorf("invalid type for strategy.kind: %T", raw)
	}
}
