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
Package packages implements resolution for the Package CR.

The Package CR stores its spec as a raw JSON contract (spec.contract) rather
than typed Kubernetes fields. ResolvePackage decodes this raw contract into
a fully typed ResolvedPackage — the authoritative runtime representation
consumed by all downstream domain and application logic.

Resolution is strict: required fields that are missing or malformed return
errors immediately. Optional fields are extracted with safe defaults. No
panics — all validation surfaces as errors that the domain layer can handle
and record as conditions on the CR.
*/
package packages

import (
	"encoding/json"
	"fmt"

	environmentv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
//
// ResolvedPackage is the single runtime representation of a Package CR.
// All downstream domain and application logic MUST use this type.
// Never re-read from the raw CR spec after resolution.
// -----------------------------------------------------------------------------

// ResolvedPackage is the fully resolved Package CR, pairing the original
// Kubernetes object with its decoded and validated spec.
type ResolvedPackage struct {
	Package *environmentv1alpha1.Package
	Spec    *ResolvedPackageSpec
}

// ResolvedPackageSpec is the decoded and validated Package spec, ready for
// domain and application layer consumption.
type ResolvedPackageSpec struct {
	// Enabled controls whether the package is active. Defaults to true.
	Enabled     bool
	Name        string
	Version     string
	Description string
	// DiffEnabled controls whether kapp diff is run before apply.
	DiffEnabled       bool
	PackageRepository ResolvedPackageRepository
	// StateRepository is optional — not all packages track state via GitOps.
	StateRepository *ResolvedStateRepository
	Maintainers     []ResolvedMaintainer
}

// ResolvedPackageRepository is the resolved OCI or Carvel package repository
// from which the package is sourced.
type ResolvedPackageRepository struct {
	URL               string
	CredentialsSecret string
}

// ResolvedStateRepository is the optional GitOps state repository where
// package deployment state is tracked by Carvel kapp.
type ResolvedStateRepository struct {
	URL         string
	Ref         Ref
	CloneSecret string
	Strategy    string
	Path        string
}

// Ref is a Git reference — exactly one of Branch, Tag, or Commit should
// be set. Resolution does not enforce mutual exclusivity; consumers
// should prefer Commit > Tag > Branch when multiple are set.
type Ref struct {
	Branch string
	Tag    string
	Commit string
}

// ResolvedMaintainer is a package maintainer contact.
type ResolvedMaintainer struct {
	Name  string
	Email string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolvePackage decodes and validates the raw JSON contract from the Package
// CR spec into a ResolvedPackage. Returns an error if the CR is nil, the
// contract is absent, or any required field is missing or malformed.
func ResolvePackage(pkg *environmentv1alpha1.Package) (*ResolvedPackage, error) {
	if pkg == nil {
		return nil, fmt.Errorf("package is nil")
	}

	if pkg.Spec.Contract.Raw == nil {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// Decode the raw contract into an untyped map for field extraction.
	// Typed field extraction follows below via the helper functions.
	var raw map[string]any
	if err := json.Unmarshal(pkg.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode contract: %w", err)
	}

	spec := &ResolvedPackageSpec{
		Enabled:     optionalBool(raw, "enabled", true),
		DiffEnabled: optionalBool(raw, "packageKappDiff", false),
		Description: optionalString(raw, "packageDescription"),
	}

	// Required string fields — errors propagate immediately.
	var err error
	if spec.Name, err = requiredString(raw, "packageName"); err != nil {
		return nil, err
	}
	if spec.Version, err = requiredString(raw, "packageVersion"); err != nil {
		return nil, err
	}

	// ------------------------------------------------
	// Package repository (REQUIRED)
	// ------------------------------------------------
	repoRaw, err := requiredMap(raw, "packageRepository")
	if err != nil {
		return nil, err
	}
	spec.PackageRepository, err = resolveRepository(repoRaw)
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------
	// State repository (OPTIONAL)
	// ------------------------------------------------
	if srRaw, ok := raw["stateRepo"]; ok {
		m, ok := srRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("spec.contract.stateRepo must be an object")
		}
		spec.StateRepository, err = resolveStateRepository(m)
		if err != nil {
			return nil, err
		}
	}

	// ------------------------------------------------
	// Maintainers (OPTIONAL)
	// ------------------------------------------------
	if msRaw, ok := raw["packageMaintainers"]; ok {
		spec.Maintainers, err = resolveMaintainers(msRaw)
		if err != nil {
			return nil, err
		}
	}

	return &ResolvedPackage{
		Package: pkg,
		Spec:    spec,
	}, nil
}

// -----------------------------------------------------------------------------
// Field resolvers
// -----------------------------------------------------------------------------

func resolveRepository(m map[string]any) (ResolvedPackageRepository, error) {
	url, err := requiredString(m, "url")
	if err != nil {
		return ResolvedPackageRepository{}, err
	}
	return ResolvedPackageRepository{
		URL:               url,
		CredentialsSecret: optionalString(m, "credentialsSecret"),
	}, nil
}

func resolveStateRepository(m map[string]any) (*ResolvedStateRepository, error) {
	url, err := requiredString(m, "url")
	if err != nil {
		return nil, err
	}

	ref := Ref{}
	if r, ok := m["ref"]; ok {
		rm, ok := r.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("stateRepo.ref must be an object")
		}
		ref = Ref{
			Branch: optionalString(rm, "branch"),
			Tag:    optionalString(rm, "tag"),
			Commit: optionalString(rm, "commit"),
		}
	}

	return &ResolvedStateRepository{
		URL:         url,
		Ref:         ref,
		CloneSecret: optionalString(m, "cloneSecret"),
		Strategy:    optionalString(m, "strategy"),
		Path:        optionalString(m, "path"),
	}, nil
}

func resolveMaintainers(v any) ([]ResolvedMaintainer, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("packageMaintainers must be an array")
	}

	out := make([]ResolvedMaintainer, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("packageMaintainers[%d] must be an object", i)
		}

		name, err := requiredString(m, "name")
		if err != nil {
			return nil, fmt.Errorf("packageMaintainers[%d].name: %w", i, err)
		}
		email, err := requiredString(m, "email")
		if err != nil {
			return nil, fmt.Errorf("packageMaintainers[%d].email: %w", i, err)
		}

		out = append(out, ResolvedMaintainer{Name: name, Email: email})
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
//
// requiredString returns an error rather than panicking — resolution must
// never crash the controller process. All required field failures are surfaced
// as conditions on the CR via the domain error handling path.
// -----------------------------------------------------------------------------

// requiredString extracts a non-empty string from m[key].
// Returns an error if the key is absent or the value is not a non-empty string.
func requiredString(m map[string]any, key string) (string, error) {
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

// requiredMap extracts a map[string]any from m[key].
// Returns an error if the key is absent or the value is not an object.
func requiredMap(m map[string]any, key string) (map[string]any, error) {
	v, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("missing required object %q", key)
	}
	out, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %q must be an object", key)
	}
	return out, nil
}

// optionalString extracts a string from m[key], returning "" if absent
// or not a string.
func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// optionalBool extracts a bool from m[key], returning def if absent or
// not a bool.
func optionalBool(m map[string]any, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
