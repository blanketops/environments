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
Package deployment provides domain-specific, field-level caching for
Deployment resources. DeploymentCache embeds cache.ObjectCache and adds
typed Set* / Get* helpers for each resolved spec field (serviceUnits, runtime,
strategy, imageAutomation, reconciliationStrategy, gitOwner,
manifestsRepo), plus PublishResolved, which projects a
*deploymentResolution.ResolvedDeployment into those fields — mandatory
fields unconditionally, optional ones only when resolution produced a
non-empty value.
*/
package deployment

import (
	"context"

	"github.com/blanketops/environments/core/cache"
	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/blanketops/environments/cache"
	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"
)

// DeploymentCache provides domain-specific, field-level caching for Deployment resources.
type DeploymentCache struct {
	*bocache.ObjectCache
}

// NewDeploymentCache constructs a new DeploymentCache with the provided cache.Cache.
func NewDeploymentCache(c *cache.Cache) *DeploymentCache {
	return &DeploymentCache{ObjectCache: bocache.NewObjectCache(c, "deployment", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: ServiceUnits
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetServiceUnits(ctx context.Context, nn types.NamespacedName, gen int64, units []string) error {
	return d.SetField(ctx, nn, gen, "serviceUnits", units)

}

func (d *DeploymentCache) GetServiceUnits(ctx context.Context, nn types.NamespacedName, gen int64) ([]string, bool, error) {
	var units []string
	found, err := d.GetField(ctx, nn, gen, "serviceUnits", &units)
	return units, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Runtime
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetRuntime(ctx context.Context, nn types.NamespacedName, gen int64, runtime string) error {
	return d.SetField(ctx, nn, gen, "runtime", runtime)
}

func (d *DeploymentCache) GetRuntime(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var r string
	found, err := d.GetField(ctx, nn, gen, "runtime", &r)
	return r, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Strategy
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetStrategy(ctx context.Context, nn types.NamespacedName, gen int64, strategy string) error {
	return d.SetField(ctx, nn, gen, "strategy", strategy)
}

func (d *DeploymentCache) GetStrategy(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var s string
	found, err := d.GetField(ctx, nn, gen, "strategy", &s)
	return s, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: ImageAutomation
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetImageAutomation(ctx context.Context, nn types.NamespacedName, gen int64, enabled bool) error {
	return d.SetField(ctx, nn, gen, "imageAutomation", enabled)
}

func (d *DeploymentCache) GetImageAutomation(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var auto bool
	found, err := d.GetField(ctx, nn, gen, "imageAutomation", &auto)
	return auto, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: ReconciliationStrategy
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetReconciliationStrategy(ctx context.Context, nn types.NamespacedName, gen int64, strategy string) error {
	return d.SetField(ctx, nn, gen, "reconciliationStrategy", strategy)
}

func (d *DeploymentCache) GetReconciliationStrategy(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var rs string
	found, err := d.GetField(ctx, nn, gen, "reconciliationStrategy", &rs)
	return rs, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: GitOwner
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetGitOwner(ctx context.Context, nn types.NamespacedName, gen int64, owner string) error {
	return d.SetField(ctx, nn, gen, "gitOwner", owner)
}

func (d *DeploymentCache) GetGitOwner(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var owner string
	found, err := d.GetField(ctx, nn, gen, "gitOwner", &owner)
	return owner, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: ManifestsRepo
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetManifestsRepo(ctx context.Context, nn types.NamespacedName, gen int64, repo any) error {
	return d.SetField(ctx, nn, gen, "manifestsRepo", repo)
}

func (d *DeploymentCache) GetManifestsRepo(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return d.GetField(ctx, nn, gen, "manifestsRepo", into)
}

// PublishResolved writes the resolved deployment contract as a
// generation-scoped, field-level projection. All writes are best-effort:
// failures cost queryability, never correctness. Returns the first error
// encountered for optional logging; callers should not fail
// reconciliation on it.
func (d *DeploymentCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *deploymentResolution.ResolvedDeployment) error {
	if r == nil || r.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Mandatory: present whenever resolution succeeds.
	record(d.SetRuntime(ctx, nn, gen, string(r.Spec.Runtime)))
	record(d.SetStrategy(ctx, nn, gen, string(r.Spec.Strategy)))
	record(d.SetReconciliationStrategy(ctx, nn, gen, string(r.Spec.ReconciliationStrategy)))

	// Conditional: publish only when resolved to something meaningful.
	if len(r.Spec.ServiceUnits) > 0 {
		record(d.SetServiceUnits(ctx, nn, gen, r.Spec.ServiceUnits))
	}
	record(d.SetImageAutomation(ctx, nn, gen, r.Spec.ImageAutomation))
	if r.Spec.GitOwner != "" {
		record(d.SetGitOwner(ctx, nn, gen, r.Spec.GitOwner))
	}
	if r.Spec.ManifestsRepo != nil {
		record(d.SetManifestsRepo(ctx, nn, gen, r.Spec.ManifestsRepo))
	}

	return firstErr
}
