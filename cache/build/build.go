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
	"context"
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// BuildCache provides domain-specific, field-level caching for Build resources.
type BuildCache struct {
	cache *core.Cache
}

// NewBuildCache creates a new BuildCache instance.
func NewBuildCache(c *core.Cache) *BuildCache {
	return &BuildCache{cache: c}
}

// key generates a specific cache key for a single field of a Build.
func (b *BuildCache) key(name, field string) string {
	return fmt.Sprintf("build:%s:%s", name, field)
}

// SetField stores an individual CR value in the external cache.
func (b *BuildCache) SetField(ctx context.Context, name, field string, val any) error {
	// Set a reasonable TTL (e.g., 1 hour)
	return b.cache.External.Set(ctx, b.key(name, field), val, 1*time.Hour)
}

// GetField retrieves an individual CR value from the external cache.
func (b *BuildCache) GetField(ctx context.Context, name, field string, into any) (bool, error) {
	return b.cache.External.Get(ctx, b.key(name, field), into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Image
// -----------------------------------------------------------------------------

func (b *BuildCache) SetImage(ctx context.Context, name string, image string) error {
	return b.SetField(ctx, name, "image", image)
}

func (b *BuildCache) GetImage(ctx context.Context, name string) (string, bool, error) {
	var img string
	found, err := b.GetField(ctx, name, "image", &img)
	return img, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Strategy
// -----------------------------------------------------------------------------

func (b *BuildCache) SetStrategy(ctx context.Context, name string, strategy any) error {
	return b.SetField(ctx, name, "strategy", strategy)
}

func (b *BuildCache) GetStrategy(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "strategy", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Source
// -----------------------------------------------------------------------------

func (b *BuildCache) SetSource(ctx context.Context, name string, source any) error {
	return b.SetField(ctx, name, "source", source)
}

func (b *BuildCache) GetSource(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "source", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: ServiceAccount
// -----------------------------------------------------------------------------

func (b *BuildCache) SetServiceAccount(ctx context.Context, name string, sa any) error {
	return b.SetField(ctx, name, "serviceAccount", sa)
}

func (b *BuildCache) GetServiceAccount(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "serviceAccount", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Policy
// -----------------------------------------------------------------------------

func (b *BuildCache) SetPolicy(ctx context.Context, name string, policy any) error {
	return b.SetField(ctx, name, "policy", policy)
}

func (b *BuildCache) GetPolicy(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "policy", into)
}
