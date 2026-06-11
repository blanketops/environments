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

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/ntlaletsi70/blanketops-environments/cache"
	"github.com/ntlaletsi70/blanketops-environments/core"
	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// GitHubEventCache provides domain-specific, field-level caching for GitHubEvent resources.
type GitHubEventCache struct {
	*bocache.ObjectCache
}

// NewGitHubEventCache constructs a new GitHubEventCache with the provided core.Cache.
func NewGitHubEventCache(c *core.Cache) *GitHubEventCache {
	return &GitHubEventCache{ObjectCache: bocache.NewObjectCache(c, "githubevent", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: Repository
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetRepository(ctx context.Context, nn types.NamespacedName, gen int64, name string, repo string) error {
	return g.SetField(ctx, nn, gen, "repository", repo)

}

func (g *GitHubEventCache) GetRepository(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var repo string
	found, err := g.GetField(ctx, nn, gen, "repository", &repo)
	return repo, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: EventType
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetEventType(ctx context.Context, nn types.NamespacedName, gen int64, name string, eventType string) error {
	return g.SetField(ctx, nn, gen, "eventType", eventType)
}

func (g *GitHubEventCache) GetEventType(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var et string
	found, err := g.GetField(ctx, nn, gen, "eventType", &et)
	return et, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Ref
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetRef(ctx context.Context, nn types.NamespacedName, gen int64, name string, ref string) error {
	return g.SetField(ctx, nn, gen, "ref", ref)
}

func (g *GitHubEventCache) GetRef(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var r string
	found, err := g.GetField(ctx, nn, gen, "ref", &r)
	return r, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Webhook
// -----------------------------------------------------------------------------

func (g *GitHubEventCache) SetWebhook(ctx context.Context, nn types.NamespacedName, gen int64, name string, webhook any) error {
	return g.SetField(ctx, nn, gen, "webhook", webhook)
}

func (g *GitHubEventCache) GetWebhook(ctx context.Context, nn types.NamespacedName, gen int64, name string, into any) (bool, error) {
	return g.GetField(ctx, nn, gen, "webhook", into)
}

// PublishResolved writes the resolved event contract as a
// generation-scoped, field-level projection. All writes are best-effort:
// failures cost queryability, never correctness. Returns the first error
// encountered for optional logging; callers should not fail
// reconciliation on it.
func (g *GitHubEventCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *githubeventResolution.ResolvedGitHubEvent) error {
	if r == nil || r.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	record(g.SetRepository(ctx, nn, gen, nn.Name, r.Spec.Repository))
	record(g.SetEventType(ctx, nn, gen, nn.Name, r.Spec.EventType))
	record(g.SetRef(ctx, nn, gen, nn.Name, r.Spec.Ref))
	if r.Spec.Webhook != (githubeventResolution.ResolvedWebhook{}) {
		record(g.SetWebhook(ctx, nn, gen, nn.Name, r.Spec.Webhook))
	}
	return firstErr
}
