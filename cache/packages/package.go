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

package packages

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/blanketops/environments/cache"
	"github.com/blanketops/environments/core/cache"
	packagesResolution "github.com/blanketops/environments/resolution/packages"
)

// PackageCache provides domain-specific, field-level caching for Package resources.
type PackageCache struct {
	*bocache.ObjectCache
}

// NewPackageCache constructs a new PackageCache with the provided cache.Cache.
func NewPackageCache(c *cache.Cache) *PackageCache {
	return &PackageCache{ObjectCache: bocache.NewObjectCache(c, "package", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: Version
// -----------------------------------------------------------------------------

func (p *PackageCache) SetVersion(ctx context.Context, nn types.NamespacedName, gen int64, version string) error {
	return p.SetField(ctx, nn, gen, "version", version)
}

func (p *PackageCache) GetVersion(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var version string
	found, err := p.GetField(ctx, nn, gen, "version", &version)
	return version, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: PackageRepository
// -----------------------------------------------------------------------------

func (p *PackageCache) SetPackageRepository(ctx context.Context, nn types.NamespacedName, gen int64, repo any) error {
	return p.SetField(ctx, nn, gen, "packageRepository", repo)
}

func (p *PackageCache) GetPackageRepository(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return p.GetField(ctx, nn, gen, "packageRepository", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: StateRepo
// -----------------------------------------------------------------------------

func (p *PackageCache) SetStateRepo(ctx context.Context, nn types.NamespacedName, gen int64, stateRepo any) error {
	return p.SetField(ctx, nn, gen, "stateRepo", stateRepo)
}

func (p *PackageCache) GetStateRepo(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return p.GetField(ctx, nn, gen, "stateRepo", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: KappDiff
// -----------------------------------------------------------------------------

func (p *PackageCache) SetKappDiff(ctx context.Context, nn types.NamespacedName, gen int64, enabled bool) error {
	return p.SetField(ctx, nn, gen, "kappDiff", enabled)
}

func (p *PackageCache) GetKappDiff(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var enabled bool
	found, err := p.GetField(ctx, nn, gen, "kappDiff", &enabled)
	return enabled, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Enabled
// -----------------------------------------------------------------------------

func (p *PackageCache) SetEnabled(ctx context.Context, nn types.NamespacedName, gen int64, enabled bool) error {
	return p.SetField(ctx, nn, gen, "enabled", enabled)
}

func (p *PackageCache) GetEnabled(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var enabled bool
	found, err := p.GetField(ctx, nn, gen, "enabled", &enabled)
	return enabled, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Checksum (derived — written by the service layer after
// package content is fetched/verified, not by PublishResolved)
// -----------------------------------------------------------------------------

func (p *PackageCache) SetChecksum(ctx context.Context, nn types.NamespacedName, gen int64, checksum string) error {
	return p.SetField(ctx, nn, gen, "checksum", checksum)
}

func (p *PackageCache) GetChecksum(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var checksum string
	found, err := p.GetField(ctx, nn, gen, "checksum", &checksum)
	return checksum, found, err
}

// PublishResolved writes the resolved package contract as a
// generation-scoped, field-level projection. All writes are best-effort:
// failures cost queryability, never correctness. Returns the first error
// encountered for optional logging; callers should not fail
// reconciliation on it.
func (p *PackageCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *packagesResolution.ResolvedPackage) error {
	if r == nil || r.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	record(p.SetVersion(ctx, nn, gen, r.Spec.Version))
	record(p.SetEnabled(ctx, nn, gen, r.Spec.Enabled))
	var zeroRepo packagesResolution.ResolvedPackageRepository
	if r.Spec.PackageRepository != zeroRepo {
		record(p.SetPackageRepository(ctx, nn, gen, r.Spec.PackageRepository))
	}
	return firstErr
}
