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

package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/packages/domain"
	pkgResolution "github.com/ntlaletsi70/blanketops-environments/resolution/packages"
)

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved Package into a pure domain Package.
//
// CONTRACT:
// - Resolver guarantees presence of mandatory fields
// - Nil values indicate a resolver bug and MUST crash loudly
// - Optional fields must be preserved verbatim
// - Mapper must not invent defaults or hide intent
func (Mapper) MapResolvedToDomain(rp *pkgResolution.ResolvedPackage) *domain.PackageSpec {
	if rp == nil {
		panic("nil ResolvedPackage passed to Package mapper (resolver bug)")
	}

	spec := rp.Spec
	cr := rp.Package

	if spec == nil {
		panic(fmt.Sprintf(
			"resolved package %q has nil Spec (resolver bug)",
			cr.Name,
		))
	}

	// ---------------------------------------------------------------------
	// INVARIANTS (resolver-owned guarantees)
	// ---------------------------------------------------------------------

	if spec.Name == "" {
		panic(fmt.Sprintf(
			"resolved package %q has empty Name (resolver bug)",
			cr.Name,
		))
	}

	if spec.Version == "" {
		panic(fmt.Sprintf(
			"resolved package %q has empty Version (resolver bug)",
			cr.Name,
		))
	}

	// if spec.Repository.URL == "" {
	// 	panic(fmt.Sprintf(
	// 		"resolved package %q has invalid Repository (resolver bug)",
	// 		cr.Name,
	// 	))
	// }

	// ---------------------------------------------------------------------
	// Identity
	// ---------------------------------------------------------------------

	pkgID := domain.PackageID{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	}

	// Disabled packages are valid but inert
	if !spec.Enabled {
		return &domain.PackageSpec{
			ID:      pkgID,
			Name:    spec.Name,
			Enabled: false,
		}
	}

	// ---------------------------------------------------------------------
	// Maintainers (verbatim)
	// ---------------------------------------------------------------------

	maintainers := make([]domain.Maintainer, 0, len(spec.Maintainers))
	for _, m := range spec.Maintainers {
		maintainers = append(maintainers, domain.Maintainer{
			Name:  m.Name,
			Email: m.Email,
		})
	}

	// ---------------------------------------------------------------------
	// Source of truth (package repo)
	// ---------------------------------------------------------------------

	source := domain.PackageSource{
		RepositoryURL:     spec.PackageRepository.URL,
		CredentialsSecret: spec.PackageRepository.CredentialsSecret,
	}

	// ---------------------------------------------------------------------
	// State repository (GitOps anchor)
	// ---------------------------------------------------------------------

	var stateRepo domain.StateRepository
	if spec.StateRepository != nil {
		stateRepo = domain.StateRepository{
			URL:         spec.StateRepository.URL,
			CloneSecret: spec.StateRepository.CloneSecret,
			Strategy:    spec.StateRepository.Strategy,
			Path:        spec.StateRepository.Path,
			//Ref:         spec.StateRepository.Ref,
		}
	}

	// ---------------------------------------------------------------------
	// Apply strategy (explicit mapping, no guessing)
	// ---------------------------------------------------------------------

	var applyStrategy domain.ApplyStrategy
	switch spec.StateRepository.Strategy {
	case "kustomization", "kustomize":
		applyStrategy = domain.StrategyKustomize
	case "", "plain":
		applyStrategy = domain.StrategyPlainYAML
	default:
		panic(fmt.Sprintf(
			"resolved package %q has unsupported apply strategy %q (resolver bug)",
			cr.Name,
			spec.StateRepository.Strategy,
		))
	}

	// ---------------------------------------------------------------------
	// Final domain package
	// ---------------------------------------------------------------------

	return &domain.PackageSpec{
		ID:          pkgID,
		Name:        spec.Name,
		Version:     spec.Version,
		Enabled:     true,
		Description: spec.Description,

		Maintainers: maintainers,
		Source:      source,
		StateRepo:   stateRepo,

		DiffEnabled: spec.DiffEnabled,
		Strategy:    applyStrategy,
	}
}
