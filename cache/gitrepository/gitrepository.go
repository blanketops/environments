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
Package gitrepository provides domain-specific, field-level caching for
GitRepository resources. GitRepositoryCache embeds cache.ObjectCache and
adds typed Set* / Get* helpers for each resolved spec field (provider,
hookUrl, repository, webhooks), plus PublishResolved, which projects a
*gitrepositoryResolution.ResolvedGitRepository into those fields —
hookUrl and webhooks only when resolution actually populated them.
*/
package gitrepository

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/blanketops/environments/cache"
	"github.com/blanketops/environments/core/cache"
	gitrepositoryResolution "github.com/blanketops/environments/resolution/gitrepository/resolve"
)

// GitRepositoryCache provides domain-specific, field-level caching for GitRepository resources.
type GitRepositoryCache struct {
	*bocache.ObjectCache
}

// NewGitRepositoryCache constructs a new GitRepositoryCache with the provided cache.Cache.
func NewGitRepositoryCache(c *cache.Cache) *GitRepositoryCache {
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

func (g *GitRepositoryCache) SetHookURL(ctx context.Context, nn types.NamespacedName, gen int64, name string, url string) error {
	return g.SetField(ctx, nn, gen, "hookUrl", url)
}

func (g *GitRepositoryCache) GetHookURL(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
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

// PublishResolved writes the resolved repository contract as a
// generation-scoped, field-level projection. All writes are best-effort:
// failures cost queryability, never correctness. Returns the first error
// encountered for optional logging; callers should not fail
// reconciliation on it.
func (g *GitRepositoryCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *gitrepositoryResolution.ResolvedGitRepository) error {
	if r == nil || r.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	record(g.SetProvider(ctx, nn, gen, "", r.Spec.Provider))
	record(g.SetRepository(ctx, nn, gen, "", r.Spec.Repository))
	if r.Spec.HookURL != "" {
		record(g.SetHookURL(ctx, nn, gen, "", r.Spec.HookURL))
	}
	if r.Spec.Webhooks != nil {
		record(g.SetWebhooks(ctx, nn, gen, "", r.Spec.Webhooks))
	}
	return firstErr
}
