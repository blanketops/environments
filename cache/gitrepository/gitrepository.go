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

package gitrepository

import (
	"context"
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// GitRepositoryCache provides domain-specific, field-level caching for GitRepository resources.
type GitRepositoryCache struct {
	cache *core.Cache
}

// NewGitRepositoryCache creates a new GitRepositoryCache instance.
func NewGitRepositoryCache(c *core.Cache) *GitRepositoryCache {
	return &GitRepositoryCache{cache: c}
}

// key generates a specific cache key for a single field of a GitRepository.
func (g *GitRepositoryCache) key(name, field string) string {
	return fmt.Sprintf("gitrepository:%s:%s", name, field)
}

// SetField stores an individual CR value in the external cache.
func (g *GitRepositoryCache) SetField(ctx context.Context, name, field string, val any) error {
	return g.cache.External.Set(ctx, g.key(name, field), val, 1*time.Hour)
}

// GetField retrieves an individual CR value from the external cache.
func (g *GitRepositoryCache) GetField(ctx context.Context, name, field string, into any) (bool, error) {
	return g.cache.External.Get(ctx, g.key(name, field), into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Provider
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetProvider(ctx context.Context, name string, provider string) error {
	return g.SetField(ctx, name, "provider", provider)
}

func (g *GitRepositoryCache) GetProvider(ctx context.Context, name string) (string, bool, error) {
	var p string
	found, err := g.GetField(ctx, name, "provider", &p)
	return p, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: HookUrl
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetHookUrl(ctx context.Context, name string, url string) error {
	return g.SetField(ctx, name, "hookUrl", url)
}

func (g *GitRepositoryCache) GetHookUrl(ctx context.Context, name string) (string, bool, error) {
	var u string
	found, err := g.GetField(ctx, name, "hookUrl", &u)
	return u, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Repository
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetRepository(ctx context.Context, name string, repo any) error {
	return g.SetField(ctx, name, "repository", repo)
}

func (g *GitRepositoryCache) GetRepository(ctx context.Context, name string, into any) (bool, error) {
	return g.GetField(ctx, name, "repository", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Webhooks
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetWebhooks(ctx context.Context, name string, webhooks any) error {
	return g.SetField(ctx, name, "webhooks", webhooks)
}

func (g *GitRepositoryCache) GetWebhooks(ctx context.Context, name string, into any) (bool, error) {
	return g.GetField(ctx, name, "webhooks", into)
}
