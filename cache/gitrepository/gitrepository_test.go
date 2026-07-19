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
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	gitrepositoryResolution "github.com/blanketops/environments/resolution/gitrepository/resolve"
)

func newTestGitRepositoryCache(t *testing.T) *GitRepositoryCache {
	t.Helper()
	return NewGitRepositoryCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestGitRepositoryCache_FieldRoundTrips(t *testing.T) {
	c := newTestGitRepositoryCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "r1"}

	if err := c.SetProvider(ctx, nn, 1, "", "github"); err != nil {
		t.Fatalf("SetProvider: %v", err)
	}
	if v, found, err := c.GetProvider(ctx, nn, 1, ""); err != nil || !found || v != "github" {
		t.Errorf("GetProvider = %q, %v, %v", v, found, err)
	}

	if err := c.SetHookURL(ctx, nn, 1, "", "https://hooks.example/x"); err != nil {
		t.Fatalf("SetHookURL: %v", err)
	}
	if v, found, err := c.GetHookURL(ctx, nn, 1, ""); err != nil || !found || v != "https://hooks.example/x" {
		t.Errorf("GetHookURL = %q, %v, %v", v, found, err)
	}

	repo := gitrepositoryResolution.GitRepositoryRef{Owner: "blanketops", Name: "environments"}
	if err := c.SetRepository(ctx, nn, 1, "", repo); err != nil {
		t.Fatalf("SetRepository: %v", err)
	}
	var gotRepo gitrepositoryResolution.GitRepositoryRef
	if found, err := c.GetRepository(ctx, nn, 1, "", &gotRepo); err != nil || !found || gotRepo != repo {
		t.Errorf("GetRepository = %+v, %v, %v", gotRepo, found, err)
	}

	webhooks := []gitrepositoryResolution.GitRepositoryWebhook{{}}
	if err := c.SetWebhooks(ctx, nn, 1, "", webhooks); err != nil {
		t.Fatalf("SetWebhooks: %v", err)
	}
	var gotWebhooks []gitrepositoryResolution.GitRepositoryWebhook
	if found, err := c.GetWebhooks(ctx, nn, 1, "", &gotWebhooks); err != nil || !found || len(gotWebhooks) != 1 {
		t.Errorf("GetWebhooks = %+v, %v, %v", gotWebhooks, found, err)
	}
}

func TestGitRepositoryCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "r1"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestGitRepositoryCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestGitRepositoryCache(t)
		r := &gitrepositoryResolution.ResolvedGitRepository{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("minimal spec publishes mandatory fields only", func(t *testing.T) {
		c := newTestGitRepositoryCache(t)
		r := &gitrepositoryResolution.ResolvedGitRepository{
			Spec: &gitrepositoryResolution.ResolvedGitRepositorySpec{
				Provider:   "github",
				Repository: gitrepositoryResolution.GitRepositoryRef{Owner: "blanketops", Name: "environments"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetProvider(context.Background(), nn, 1, ""); !found || v != "github" {
			t.Errorf("GetProvider = %q, %v", v, found)
		}
		if _, found, _ := c.GetHookURL(context.Background(), nn, 1, ""); found {
			t.Error("GetHookURL: found = true, want false when HookURL is empty")
		}
		var webhooks []gitrepositoryResolution.GitRepositoryWebhook
		if found, _ := c.GetWebhooks(context.Background(), nn, 1, "", &webhooks); found {
			t.Error("GetWebhooks: found = true, want false when Webhooks is nil")
		}
	})

	t.Run("full spec publishes optional fields too", func(t *testing.T) {
		c := newTestGitRepositoryCache(t)
		r := &gitrepositoryResolution.ResolvedGitRepository{
			Spec: &gitrepositoryResolution.ResolvedGitRepositorySpec{
				Provider:   "github",
				HookURL:    "https://hooks.example/x",
				Repository: gitrepositoryResolution.GitRepositoryRef{Owner: "blanketops", Name: "environments"},
				Webhooks:   []gitrepositoryResolution.GitRepositoryWebhook{{}},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetHookURL(context.Background(), nn, 1, ""); !found || v != "https://hooks.example/x" {
			t.Errorf("GetHookURL = %q, %v", v, found)
		}
		var webhooks []gitrepositoryResolution.GitRepositoryWebhook
		if found, _ := c.GetWebhooks(context.Background(), nn, 1, "", &webhooks); !found || len(webhooks) != 1 {
			t.Errorf("GetWebhooks = %+v, %v", webhooks, found)
		}
	})
}
