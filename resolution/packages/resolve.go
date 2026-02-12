package packages

import (
	"encoding/json"
	"fmt"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

//
// ==============================
// RUNTIME Package (AUTHORITATIVE)
// ==============================
//

// ResolvedPackage is the SINGLE runtime representation of a Package.
// Everything downstream MUST use this.
type ResolvedPackage struct {
	Package *environmentv1.Package
	Spec    *ResolvedPackageSpec
}

type ResolvedPackageSpec struct {
	Enabled     bool
	Name        string
	Version     string
	Description string
	DiffEnabled bool

	PackageRepository ResolvedPackageRepository
	StateRepository   *ResolvedStateRepository

	Maintainers []ResolvedMaintainer
}

type ResolvedPackageRepository struct {
	URL               string
	CredentialsSecret string
}

type ResolvedStateRepository struct {
	URL         string
	Ref         Ref
	CloneSecret string
	Strategy    string
	Path        string
}

type Ref struct {
	Branch string
	Tag    string
	Commit string
}

type ResolvedMaintainer struct {
	Name  string
	Email string
}

//
// ==============================
// RESOLUTION ENTRY POINT
// ==============================
//

func ResolvePackage(pkg *environmentv1.Package) (*ResolvedPackage, error) {
	if pkg == nil {
		return nil, fmt.Errorf("package is nil")
	}
	if pkg.Spec.Contract.Raw == nil {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// Decode raw contract
	var raw map[string]any
	if err := json.Unmarshal(pkg.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode contract: %w", err)
	}

	spec := &ResolvedPackageSpec{
		Enabled:     optionalBool(raw, "enabled", true),
		Name:        requiredString(raw, "packageName"),
		Version:     requiredString(raw, "packageVersion"),
		Description: optionalString(raw, "packageDescription"),
		DiffEnabled: optionalBool(raw, "packageKappDiff", false),
	}

	// Package repository (REQUIRED)
	repoRaw, err := requiredMap(raw, "packageRepository")
	if err != nil {
		return nil, err
	}
	spec.PackageRepository, err = resolveRepository(repoRaw)
	if err != nil {
		return nil, err
	}

	// State repository (OPTIONAL)
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

	// Maintainers (OPTIONAL)
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

//
// ==============================
// RESOLVERS
// ==============================
//

func resolveRepository(m map[string]any) (ResolvedPackageRepository, error) {
	url := requiredString(m, "url")

	return ResolvedPackageRepository{
		URL:               url,
		CredentialsSecret: optionalString(m, "credentialsSecret"),
	}, nil
}

func resolveStateRepository(m map[string]any) (*ResolvedStateRepository, error) {
	url := requiredString(m, "url")

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
		out = append(out, ResolvedMaintainer{
			Name:  requiredString(m, "name"),
			Email: requiredString(m, "email"),
		})
	}
	return out, nil
}

//
// ==============================
// HELPERS (STRICT, NO PANICS)
// ==============================
//

func requiredString(m map[string]any, key string) string {
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

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func optionalBool(m map[string]any, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
