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

package githubevent

import (
	"context"
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// GitHubEventCache provides domain-specific, field-level caching for GitHubEvent resources.
type GitHubEventCache struct {
	cache *core.Cache
}

// NewGitHubEventCache creates a new GitHubEventCache instance.
func NewGitHubEventCache(c *core.Cache) *GitHubEventCache {
	return &GitHubEventCache{cache: c}
}

// key generates a specific cache key for a single field of a GitHubEvent.
func (g *GitHubEventCache) key(name, field string) string {
	return fmt.Sprintf("githubevent:%s:%s", name, field)
}

// SetField stores an individual CR value in the external cache.
func (g *GitHubEventCache) SetField(ctx context.Context, name, field string, val any) error {
	return g.cache.External.Set(ctx, g.key(name, field), val, 1*time.Hour)
}

// GetField retrieves an individual CR value from the external cache.
func (g *GitHubEventCache) GetField(ctx context.Context, name, field string, into any) (bool, error) {
	return g.cache.External.Get(ctx, g.key(name, field), into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Repository
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetRepository(ctx context.Context, name string, repo string) error {
	return g.SetField(ctx, name, "repository", repo)
}

func (g *GitHubEventCache) GetRepository(ctx context.Context, name string) (string, bool, error) {
	var repo string
	found, err := g.GetField(ctx, name, "repository", &repo)
	return repo, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: EventType
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetEventType(ctx context.Context, name string, eventType string) error {
	return g.SetField(ctx, name, "eventType", eventType)
}

func (g *GitHubEventCache) GetEventType(ctx context.Context, name string) (string, bool, error) {
	var et string
	found, err := g.GetField(ctx, name, "eventType", &et)
	return et, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Ref
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetRef(ctx context.Context, name string, ref string) error {
	return g.SetField(ctx, name, "ref", ref)
}

func (g *GitHubEventCache) GetRef(ctx context.Context, name string) (string, bool, error) {
	var r string
	found, err := g.GetField(ctx, name, "ref", &r)
	return r, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Webhook
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetWebhook(ctx context.Context, name string, webhook any) error {
	return g.SetField(ctx, name, "webhook", webhook)
}

func (g *GitHubEventCache) GetWebhook(ctx context.Context, name string, into any) (bool, error) {
	return g.GetField(ctx, name, "webhook", into)
}
