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
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// BuildTriggerCache provides domain-specific, field-level caching for BuildTrigger resources.
type BuildTriggerCache struct {
	cache *core.Cache
}

// NewBuildTriggerCache creates a new BuildTriggerCache instance.
func NewBuildTriggerCache(c *core.Cache) *BuildTriggerCache {
	return &BuildTriggerCache{cache: c}
}

// key generates a specific cache key for a single field of a BuildTrigger.
func (b *BuildTriggerCache) key(name, field string) string {
	return fmt.Sprintf("buildtrigger:%s:%s", name, field)
}

// SetField stores an individual CR value in the external cache.
func (b *BuildTriggerCache) SetField(ctx context.Context, name, field string, val any) error {
	return b.cache.External.Set(ctx, b.key(name, field), val, 1*time.Hour)
}

// GetField retrieves an individual CR value from the external cache.
func (b *BuildTriggerCache) GetField(ctx context.Context, name, field string, into any) (bool, error) {
	return b.cache.External.Get(ctx, b.key(name, field), into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Source
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetSource(ctx context.Context, name string, source string) error {
	return b.SetField(ctx, name, "source", source)
}

func (b *BuildTriggerCache) GetSource(ctx context.Context, name string) (string, bool, error) {
	var src string
	found, err := b.GetField(ctx, name, "source", &src)
	return src, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: EventType
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetEventType(ctx context.Context, name string, eventType string) error {
	return b.SetField(ctx, name, "eventType", eventType)
}

func (b *BuildTriggerCache) GetEventType(ctx context.Context, name string) (string, bool, error) {
	var et string
	found, err := b.GetField(ctx, name, "eventType", &et)
	return et, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Repository
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetRepository(ctx context.Context, name string, repo any) error {
	return b.SetField(ctx, name, "repository", repo)
}

func (b *BuildTriggerCache) GetRepository(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "repository", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Ref
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetRef(ctx context.Context, name string, ref string) error {
	return b.SetField(ctx, name, "ref", ref)
}

func (b *BuildTriggerCache) GetRef(ctx context.Context, name string) (string, bool, error) {
	var r string
	found, err := b.GetField(ctx, name, "ref", &r)
	return r, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: BuildRef
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetBuildRef(ctx context.Context, name string, buildRef any) error {
	return b.SetField(ctx, name, "buildRef", buildRef)
}

func (b *BuildTriggerCache) GetBuildRef(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "buildRef", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: PayloadPolicy
// -----------------------------------------------------------------------------

func (b *BuildTriggerCache) SetPayloadPolicy(ctx context.Context, name string, policy any) error {
	return b.SetField(ctx, name, "payloadPolicy", policy)
}

func (b *BuildTriggerCache) GetPayloadPolicy(ctx context.Context, name string, into any) (bool, error) {
	return b.GetField(ctx, name, "payloadPolicy", into)
}
