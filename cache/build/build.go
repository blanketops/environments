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

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/blanketops/environments/cache"
	"github.com/blanketops/environments/core/cache"
	buildResolution "github.com/blanketops/environments/resolution/build"
)

// BuildCache provides domain-specific, field-level caching for Build resources.
type BuildCache struct {
	*bocache.ObjectCache
}

// NewBuildCache constructs a new BuildCache with the provided cache.Cache.
func NewBuildCache(c *cache.Cache) *BuildCache {
	return &BuildCache{ObjectCache: bocache.NewObjectCache(c, "build", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers
// -----------------------------------------------------------------------------

func (b *BuildCache) SetImage(ctx context.Context, nn types.NamespacedName, gen int64, image string) error {
	return b.SetField(ctx, nn, gen, "image", image)
}

func (b *BuildCache) GetImage(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var img string
	found, err := b.GetField(ctx, nn, gen, "image", &img)
	return img, found, err
}

func (b *BuildCache) SetStrategy(ctx context.Context, nn types.NamespacedName, gen int64, strategy any) error {
	return b.SetField(ctx, nn, gen, "strategy", strategy)
}

func (b *BuildCache) GetStrategy(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "strategy", into)
}

func (b *BuildCache) SetSource(ctx context.Context, nn types.NamespacedName, gen int64, source any) error {
	return b.SetField(ctx, nn, gen, "source", source)
}

func (b *BuildCache) GetSource(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "source", into)
}

func (b *BuildCache) SetServiceAccount(ctx context.Context, nn types.NamespacedName, gen int64, sa any) error {
	return b.SetField(ctx, nn, gen, "serviceAccount", sa)
}

func (b *BuildCache) GetServiceAccount(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "serviceAccount", into)
}

func (b *BuildCache) SetPolicy(ctx context.Context, nn types.NamespacedName, gen int64, policy any) error {
	return b.SetField(ctx, nn, gen, "policy", policy)
}

func (b *BuildCache) GetPolicy(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "policy", into)
}

// PublishResolved writes the resolved contract as a generation-scoped,
// field-level projection. All writes are best-effort: failures cost
// queryability, never correctness. Returns the first error encountered
// for optional logging; callers should not fail reconciliation on it.
func (b *BuildCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *buildResolution.ResolvedBuild) error {
	if r == nil || r.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	record(b.SetImage(ctx, nn, gen, r.Spec.Image))
	record(b.SetSource(ctx, nn, gen, r.Spec.Source))
	record(b.SetStrategy(ctx, nn, gen, r.Spec.Strategy))
	if r.Spec.ServiceAccount != nil {
		record(b.SetServiceAccount(ctx, nn, gen, r.Spec.ServiceAccount))
	}
	if r.Spec.Policy != nil {
		record(b.SetPolicy(ctx, nn, gen, r.Spec.Policy))
	}
	return firstErr
}
