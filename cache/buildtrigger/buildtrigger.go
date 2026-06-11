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

package buildtrigger

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/ntlaletsi70/blanketops-environments/cache"
	"github.com/ntlaletsi70/blanketops-environments/core"
)

// BuildTriggerCache provides domain-specific, field-level caching for BuildTrigger resources.
type BuildTriggerCache struct {
	*bocache.ObjectCache
}

// NewBuildTriggerCache creates a new BuildTriggerCache instance.
func NewBuildTriggerCache(c *core.Cache) *BuildTriggerCache {
	return &BuildTriggerCache{ObjectCache: bocache.NewObjectCache(c, "buildtrigger", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: Source
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetSource(ctx context.Context, nn types.NamespacedName, gen int64, source string) error {
	return b.SetField(ctx, nn, gen, "source", source)

}

func (b *BuildTriggerCache) GetSource(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var src string
	found, err := b.GetField(ctx, nn, gen, "source", &src)
	return src, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: EventType
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetEventType(ctx context.Context, nn types.NamespacedName, gen int64, eventType string) error {
	return b.SetField(ctx, nn, gen, "eventType", eventType)
}

func (b *BuildTriggerCache) GetEventType(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var et string
	found, err := b.GetField(ctx, nn, gen, "eventType", &et)
	return et, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Repository
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetRepository(ctx context.Context, nn types.NamespacedName, gen int64, repo any) error {
	return b.SetField(ctx, nn, gen, "repository", repo)
}

func (b *BuildTriggerCache) GetRepository(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "repository", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Ref
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetRef(ctx context.Context, nn types.NamespacedName, gen int64, ref string) error {
	return b.SetField(ctx, nn, gen, "ref", ref)
}

func (b *BuildTriggerCache) GetRef(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var r string
	found, err := b.GetField(ctx, nn, gen, "ref", &r)
	return r, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: BuildRef
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetBuildRef(ctx context.Context, nn types.NamespacedName, gen int64, buildRef any) error {
	return b.SetField(ctx, nn, gen, "buildRef", buildRef)
}

func (b *BuildTriggerCache) GetBuildRef(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "buildRef", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: PayloadPolicy
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetPayloadPolicy(ctx context.Context, nn types.NamespacedName, gen int64, policy any) error {
	return b.SetField(ctx, nn, gen, "payloadPolicy", policy)
}

func (b *BuildTriggerCache) GetPayloadPolicy(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return b.GetField(ctx, nn, gen, "payloadPolicy", into)
}
