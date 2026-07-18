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

package deployment

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"
)

func newTestDeploymentCache(t *testing.T) *DeploymentCache {
	t.Helper()
	return NewDeploymentCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestDeploymentCache_FieldRoundTrips(t *testing.T) {
	c := newTestDeploymentCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "d1"}

	if err := c.SetServiceUnits(ctx, nn, 1, []string{"api", "worker"}); err != nil {
		t.Fatalf("SetServiceUnits: %v", err)
	}
	if v, found, err := c.GetServiceUnits(ctx, nn, 1); err != nil || !found || len(v) != 2 {
		t.Errorf("GetServiceUnits = %v, %v, %v", v, found, err)
	}

	if err := c.SetRuntime(ctx, nn, 1, "kubernetes.io/container-runtime"); err != nil {
		t.Fatalf("SetRuntime: %v", err)
	}
	if v, found, err := c.GetRuntime(ctx, nn, 1); err != nil || !found || v != "kubernetes.io/container-runtime" {
		t.Errorf("GetRuntime = %q, %v, %v", v, found, err)
	}

	if err := c.SetStrategy(ctx, nn, 1, "Rolling"); err != nil {
		t.Fatalf("SetStrategy: %v", err)
	}
	if v, found, err := c.GetStrategy(ctx, nn, 1); err != nil || !found || v != "Rolling" {
		t.Errorf("GetStrategy = %q, %v, %v", v, found, err)
	}

	if err := c.SetImageAutomation(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetImageAutomation: %v", err)
	}
	if v, found, err := c.GetImageAutomation(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetImageAutomation = %v, %v, %v", v, found, err)
	}

	if err := c.SetReconciliationStrategy(ctx, nn, 1, "Kustomize"); err != nil {
		t.Fatalf("SetReconciliationStrategy: %v", err)
	}
	if v, found, err := c.GetReconciliationStrategy(ctx, nn, 1); err != nil || !found || v != "Kustomize" {
		t.Errorf("GetReconciliationStrategy = %q, %v, %v", v, found, err)
	}

	if err := c.SetGitOwner(ctx, nn, 1, "blanketops"); err != nil {
		t.Fatalf("SetGitOwner: %v", err)
	}
	if v, found, err := c.GetGitOwner(ctx, nn, 1); err != nil || !found || v != "blanketops" {
		t.Errorf("GetGitOwner = %q, %v, %v", v, found, err)
	}

	repo := deploymentResolution.ResolvedManifestsRepo{URL: "https://github.com/x/y"}
	if err := c.SetManifestsRepo(ctx, nn, 1, repo); err != nil {
		t.Fatalf("SetManifestsRepo: %v", err)
	}
	var gotRepo deploymentResolution.ResolvedManifestsRepo
	if found, err := c.GetManifestsRepo(ctx, nn, 1, &gotRepo); err != nil || !found || gotRepo != repo {
		t.Errorf("GetManifestsRepo = %+v, %v, %v", gotRepo, found, err)
	}
}

func TestDeploymentCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "d1"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestDeploymentCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestDeploymentCache(t)
		r := &deploymentResolution.ResolvedDeployment{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("minimal spec publishes mandatory fields only", func(t *testing.T) {
		c := newTestDeploymentCache(t)
		r := &deploymentResolution.ResolvedDeployment{
			Spec: &deploymentResolution.ResolvedDeploymentSpec{
				Runtime:                deploymentResolution.RuntimeKubernetes,
				Strategy:               deploymentResolution.StrategyRolling,
				ReconciliationStrategy: deploymentResolution.ReconciliationImperative,
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if _, found, _ := c.GetServiceUnits(context.Background(), nn, 1); found {
			t.Error("GetServiceUnits: found = true, want false for an empty slice")
		}
		if v, found, _ := c.GetGitOwner(context.Background(), nn, 1); found || v != "" {
			t.Errorf("GetGitOwner = %q, %v; want unset for empty GitOwner", v, found)
		}
		var repo deploymentResolution.ResolvedManifestsRepo
		if found, _ := c.GetManifestsRepo(context.Background(), nn, 1, &repo); found {
			t.Error("GetManifestsRepo: found = true, want false when nil")
		}
	})

	t.Run("full spec publishes optional fields too", func(t *testing.T) {
		c := newTestDeploymentCache(t)
		r := &deploymentResolution.ResolvedDeployment{
			Spec: &deploymentResolution.ResolvedDeploymentSpec{
				Runtime:                deploymentResolution.RuntimeKubernetes,
				Strategy:               deploymentResolution.StrategyRolling,
				ReconciliationStrategy: deploymentResolution.ReconciliationKustomize,
				ServiceUnits:           []string{"api"},
				GitOwner:               "blanketops",
				ManifestsRepo:          &deploymentResolution.ResolvedManifestsRepo{URL: "https://github.com/x/y"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetServiceUnits(context.Background(), nn, 1); !found || len(v) != 1 {
			t.Errorf("GetServiceUnits = %v, %v", v, found)
		}
		if v, found, _ := c.GetGitOwner(context.Background(), nn, 1); !found || v != "blanketops" {
			t.Errorf("GetGitOwner = %q, %v", v, found)
		}
		var repo deploymentResolution.ResolvedManifestsRepo
		if found, _ := c.GetManifestsRepo(context.Background(), nn, 1, &repo); !found || repo.URL != "https://github.com/x/y" {
			t.Errorf("GetManifestsRepo = %+v, %v", repo, found)
		}
	})
}
