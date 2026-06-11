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

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/ntlaletsi70/blanketops-environments/cache"
	"github.com/ntlaletsi70/blanketops-environments/core"
)

// GitRepositoryCache provides domain-specific, field-level caching for GitRepository resources.
type GitRepositoryCache struct {
	*bocache.ObjectCache
}

// NewGitRepositoryCache creates a new GitRepositoryCache instance.
func NewGitRepositoryCache(c *core.Cache) *GitRepositoryCache {
	return &GitRepositoryCache{ObjectCache: bocache.NewObjectCache(c, "gitrepository", 0)}

}

// -----------------------------------------------------------------------------
// Typed Helpers: Provider
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetProvider(ctx context.Context, nn types.NamespacedName, gen int64, name string, provider string) error {
	return g.SetField(ctx, nn, gen, "provider", provider)
}

func (g *GitRepositoryCache) GetProvider(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var p string
	found, err := g.GetField(ctx, nn, gen, "provider", &p)
	return p, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: HookUrl
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetHookUrl(ctx context.Context, nn types.NamespacedName, gen int64, name string, url string) error {
	return g.SetField(ctx, nn, gen, "hookUrl", url)
}

func (g *GitRepositoryCache) GetHookUrl(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var u string
	found, err := g.GetField(ctx, nn, gen, "hookUrl", &u)
	return u, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Repository
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetRepository(ctx context.Context, nn types.NamespacedName, gen int64, name string, repo any) error {
	return g.SetField(ctx, nn, gen, "repository", repo)
}

func (g *GitRepositoryCache) GetRepository(ctx context.Context, nn types.NamespacedName, gen int64, name string, into any) (bool, error) {
	return g.GetField(ctx, nn, gen, "repository", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Webhooks
// -----------------------------------------------------------------------------

func (g *GitRepositoryCache) SetWebhooks(ctx context.Context, nn types.NamespacedName, gen int64, name string, webhooks any) error {
	return g.SetField(ctx, nn, gen, "webhooks", webhooks)
}

func (g *GitRepositoryCache) GetWebhooks(ctx context.Context, nn types.NamespacedName, gen int64, name string, into any) (bool, error) {
	return g.GetField(ctx, nn, gen, "webhooks", into)
}
