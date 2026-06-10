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

package deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// DeploymentCache provides domain-specific, field-level caching for Deployment resources.
type DeploymentCache struct {
	cache *core.Cache
}

// NewDeploymentCache creates a new DeploymentCache instance.
func NewDeploymentCache(c *core.Cache) *DeploymentCache {
	return &DeploymentCache{cache: c}
}

// key generates a specific cache key for a single field of a Deployment.
func (d *DeploymentCache) key(name, field string) string {
	return fmt.Sprintf("deployment:%s:%s", name, field)
}

// SetField stores an individual CR value in the external cache.
func (d *DeploymentCache) SetField(ctx context.Context, name, field string, val any) error {
	return d.cache.External.Set(ctx, d.key(name, field), val, 1*time.Hour)
}

// GetField retrieves an individual CR value from the external cache.
func (d *DeploymentCache) GetField(ctx context.Context, name, field string, into any) (bool, error) {
	return d.cache.External.Get(ctx, d.key(name, field), into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: ServiceUnits
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetServiceUnits(ctx context.Context, name string, units []string) error {
	return d.SetField(ctx, name, "serviceUnits", units)
}

func (d *DeploymentCache) GetServiceUnits(ctx context.Context, name string) ([]string, bool, error) {
	var units []string
	found, err := d.GetField(ctx, name, "serviceUnits", &units)
	return units, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Runtime
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetRuntime(ctx context.Context, name string, runtime string) error {
	return d.SetField(ctx, name, "runtime", runtime)
}

func (d *DeploymentCache) GetRuntime(ctx context.Context, name string) (string, bool, error) {
	var r string
	found, err := d.GetField(ctx, name, "runtime", &r)
	return r, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Strategy
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetStrategy(ctx context.Context, name string, strategy string) error {
	return d.SetField(ctx, name, "strategy", strategy)
}

func (d *DeploymentCache) GetStrategy(ctx context.Context, name string) (string, bool, error) {
	var s string
	found, err := d.GetField(ctx, name, "strategy", &s)
	return s, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: ImageAutomation
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetImageAutomation(ctx context.Context, name string, enabled bool) error {
	return d.SetField(ctx, name, "imageAutomation", enabled)
}

func (d *DeploymentCache) GetImageAutomation(ctx context.Context, name string) (bool, bool, error) {
	var auto bool
	found, err := d.GetField(ctx, name, "imageAutomation", &auto)
	return auto, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: ReconciliationStrategy
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetReconciliationStrategy(ctx context.Context, name string, strategy string) error {
	return d.SetField(ctx, name, "reconciliationStrategy", strategy)
}

func (d *DeploymentCache) GetReconciliationStrategy(ctx context.Context, name string) (string, bool, error) {
	var rs string
	found, err := d.GetField(ctx, name, "reconciliationStrategy", &rs)
	return rs, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: GitOwner
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetGitOwner(ctx context.Context, name string, owner string) error {
	return d.SetField(ctx, name, "gitOwner", owner)
}

func (d *DeploymentCache) GetGitOwner(ctx context.Context, name string) (string, bool, error) {
	var owner string
	found, err := d.GetField(ctx, name, "gitOwner", &owner)
	return owner, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: ManifestsRepo
// -----------------------------------------------------------------------------

func (d *DeploymentCache) SetManifestsRepo(ctx context.Context, name string, repo any) error {
	return d.SetField(ctx, name, "manifestsRepo", repo)
}

func (d *DeploymentCache) GetManifestsRepo(ctx context.Context, name string, into any) (bool, error) {
	return d.GetField(ctx, name, "manifestsRepo", into)
}
