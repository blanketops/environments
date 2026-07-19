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
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	buildResolution "github.com/blanketops/environments/resolution/build/resolve"
)

func newTestBuildCache(t *testing.T) *BuildCache {
	t.Helper()
	return NewBuildCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestBuildCache_FieldRoundTrips(t *testing.T) {
	c := newTestBuildCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "b1"}

	if err := c.SetImage(ctx, nn, 1, "docker.io/x:v1"); err != nil {
		t.Fatalf("SetImage: %v", err)
	}
	if v, found, err := c.GetImage(ctx, nn, 1); err != nil || !found || v != "docker.io/x:v1" {
		t.Errorf("GetImage = %q, %v, %v", v, found, err)
	}

	strategy := buildResolution.ResolvedStrategy{Name: "s1", StrategyKind: "ClusterBuildStrategy"}
	if err := c.SetStrategy(ctx, nn, 1, strategy); err != nil {
		t.Fatalf("SetStrategy: %v", err)
	}
	var gotStrategy buildResolution.ResolvedStrategy
	if found, err := c.GetStrategy(ctx, nn, 1, &gotStrategy); err != nil || !found || gotStrategy != strategy {
		t.Errorf("GetStrategy = %+v, %v, %v", gotStrategy, found, err)
	}

	source := buildResolution.ResolvedSource{URL: "https://github.com/x/y"}
	if err := c.SetSource(ctx, nn, 1, source); err != nil {
		t.Fatalf("SetSource: %v", err)
	}
	var gotSource buildResolution.ResolvedSource
	if found, err := c.GetSource(ctx, nn, 1, &gotSource); err != nil || !found || gotSource != source {
		t.Errorf("GetSource = %+v, %v, %v", gotSource, found, err)
	}

	sa := buildResolution.ResolvedServiceAccount{Name: "sa1"}
	if err := c.SetServiceAccount(ctx, nn, 1, sa); err != nil {
		t.Fatalf("SetServiceAccount: %v", err)
	}
	var gotSA buildResolution.ResolvedServiceAccount
	if found, err := c.GetServiceAccount(ctx, nn, 1, &gotSA); err != nil || !found || gotSA != sa {
		t.Errorf("GetServiceAccount = %+v, %v, %v", gotSA, found, err)
	}

	policy := buildResolution.ResolvedBuildPolicy{Retry: &buildResolution.ResolvedRetryPolicy{OnFailure: true, MaxAttempts: 3}}
	if err := c.SetPolicy(ctx, nn, 1, policy); err != nil {
		t.Fatalf("SetPolicy: %v", err)
	}
	var gotPolicy buildResolution.ResolvedBuildPolicy
	if found, err := c.GetPolicy(ctx, nn, 1, &gotPolicy); err != nil || !found || gotPolicy.Retry.MaxAttempts != 3 {
		t.Errorf("GetPolicy = %+v, %v, %v", gotPolicy, found, err)
	}
}

func TestBuildCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "b1"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestBuildCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestBuildCache(t)
		r := &buildResolution.ResolvedBuild{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("minimal spec publishes mandatory fields only", func(t *testing.T) {
		c := newTestBuildCache(t)
		r := &buildResolution.ResolvedBuild{
			Spec: &buildResolution.ResolvedBuildSpec{
				Image:  "docker.io/x:v1",
				Source: buildResolution.ResolvedSource{URL: "https://github.com/x/y"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetImage(context.Background(), nn, 1); !found || v != "docker.io/x:v1" {
			t.Errorf("GetImage = %q, %v", v, found)
		}
		var sa buildResolution.ResolvedServiceAccount
		if found, _ := c.GetServiceAccount(context.Background(), nn, 1, &sa); found {
			t.Error("GetServiceAccount: found = true, want false when ServiceAccount is nil")
		}
		var policy buildResolution.ResolvedBuildPolicy
		if found, _ := c.GetPolicy(context.Background(), nn, 1, &policy); found {
			t.Error("GetPolicy: found = true, want false when Policy is nil")
		}
	})

	t.Run("full spec publishes optional fields too", func(t *testing.T) {
		c := newTestBuildCache(t)
		r := &buildResolution.ResolvedBuild{
			Spec: &buildResolution.ResolvedBuildSpec{
				Image:          "docker.io/x:v1",
				Source:         buildResolution.ResolvedSource{URL: "https://github.com/x/y"},
				ServiceAccount: &buildResolution.ResolvedServiceAccount{Name: "sa1"},
				Policy:         &buildResolution.ResolvedBuildPolicy{Retry: &buildResolution.ResolvedRetryPolicy{OnFailure: true, MaxAttempts: 3}},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		var sa buildResolution.ResolvedServiceAccount
		if found, _ := c.GetServiceAccount(context.Background(), nn, 1, &sa); !found || sa.Name != "sa1" {
			t.Errorf("GetServiceAccount = %+v, %v", sa, found)
		}
		var policy buildResolution.ResolvedBuildPolicy
		if found, _ := c.GetPolicy(context.Background(), nn, 1, &policy); !found || policy.Retry.MaxAttempts != 3 {
			t.Errorf("GetPolicy = %+v, %v", policy, found)
		}
	})
}
