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
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	githubeventResolution "github.com/blanketops/environments/resolution/githubevent/resolve"
)

func newTestGitHubEventCache(t *testing.T) *GitHubEventCache {
	t.Helper()
	return NewGitHubEventCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestGitHubEventCache_FieldRoundTrips(t *testing.T) {
	c := newTestGitHubEventCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "ev1"}

	if err := c.SetRepository(ctx, nn, 1, nn.Name, "blanketops/environments"); err != nil {
		t.Fatalf("SetRepository: %v", err)
	}
	if v, found, err := c.GetRepository(ctx, nn, 1, nn.Name); err != nil || !found || v != "blanketops/environments" {
		t.Errorf("GetRepository = %q, %v, %v", v, found, err)
	}

	if err := c.SetEventType(ctx, nn, 1, nn.Name, "push"); err != nil {
		t.Fatalf("SetEventType: %v", err)
	}
	if v, found, err := c.GetEventType(ctx, nn, 1, nn.Name); err != nil || !found || v != "push" {
		t.Errorf("GetEventType = %q, %v, %v", v, found, err)
	}

	if err := c.SetRef(ctx, nn, 1, nn.Name, "refs/heads/main"); err != nil {
		t.Fatalf("SetRef: %v", err)
	}
	if v, found, err := c.GetRef(ctx, nn, 1, nn.Name); err != nil || !found || v != "refs/heads/main" {
		t.Errorf("GetRef = %q, %v, %v", v, found, err)
	}

	webhook := githubeventResolution.ResolvedWebhook{SecretRef: githubeventResolution.ResolvedSecretRef{Name: "s1", Key: "secret"}}
	if err := c.SetWebhook(ctx, nn, 1, nn.Name, webhook); err != nil {
		t.Fatalf("SetWebhook: %v", err)
	}
	var gotWebhook githubeventResolution.ResolvedWebhook
	if found, err := c.GetWebhook(ctx, nn, 1, nn.Name, &gotWebhook); err != nil || !found || gotWebhook != webhook {
		t.Errorf("GetWebhook = %+v, %v, %v", gotWebhook, found, err)
	}
}

func TestGitHubEventCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "ev1"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestGitHubEventCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestGitHubEventCache(t)
		r := &githubeventResolution.ResolvedGitHubEvent{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("zero-value webhook is not published", func(t *testing.T) {
		c := newTestGitHubEventCache(t)
		r := &githubeventResolution.ResolvedGitHubEvent{
			Spec: &githubeventResolution.ResolvedGitHubEventSpec{
				Repository: "blanketops/environments",
				EventType:  "push",
				Ref:        "refs/heads/main",
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetRepository(context.Background(), nn, 1, nn.Name); !found || v != "blanketops/environments" {
			t.Errorf("GetRepository = %q, %v", v, found)
		}
		var webhook githubeventResolution.ResolvedWebhook
		if found, _ := c.GetWebhook(context.Background(), nn, 1, nn.Name, &webhook); found {
			t.Error("GetWebhook: found = true, want false for a zero-value webhook")
		}
	})

	t.Run("non-zero webhook is published", func(t *testing.T) {
		c := newTestGitHubEventCache(t)
		r := &githubeventResolution.ResolvedGitHubEvent{
			Spec: &githubeventResolution.ResolvedGitHubEventSpec{
				Repository: "blanketops/environments",
				EventType:  "push",
				Ref:        "refs/heads/main",
				Webhook:    githubeventResolution.ResolvedWebhook{SecretRef: githubeventResolution.ResolvedSecretRef{Name: "s1"}},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		var webhook githubeventResolution.ResolvedWebhook
		if found, _ := c.GetWebhook(context.Background(), nn, 1, nn.Name, &webhook); !found || webhook.SecretRef.Name != "s1" {
			t.Errorf("GetWebhook = %+v, %v", webhook, found)
		}
	})
}
