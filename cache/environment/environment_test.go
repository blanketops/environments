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

package environment

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	environmentResolution "github.com/blanketops/environments/resolution/environment/resolve"
)

func newTestEnvironmentCache(t *testing.T) *EnvironmentCache {
	t.Helper()
	return NewEnvironmentCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestEnvironmentCache_FieldRoundTrips(t *testing.T) {
	c := newTestEnvironmentCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "e1"}

	strFields := []struct {
		name string
		set  func(context.Context, types.NamespacedName, int64, string) error
		get  func(context.Context, types.NamespacedName, int64) (string, bool, error)
		val  string
	}{
		{"ApplicationName", c.SetApplicationName, c.GetApplicationName, "myapp"},
		{"EnvironmentType", c.SetEnvironmentType, c.GetEnvironmentType, "dev"},
		{"Version", c.SetVersion, c.GetVersion, "v1"},
		{"SecretStoreProvider", c.SetSecretStoreProvider, c.GetSecretStoreProvider, "vault"},
		{"Build", c.SetBuild, c.GetBuild, "build1"},
		{"GitRepository", c.SetGitRepository, c.GetGitRepository, "repo1"},
		{"GitHubEvent", c.SetGitHubEvent, c.GetGitHubEvent, "event1"},
		{"Deployment", c.SetDeployment, c.GetDeployment, "deploy1"},
		{"Route", c.SetRoute, c.GetRoute, "route1"},
		{"Package", c.SetPackage, c.GetPackage, "pkg1"},
	}

	for _, f := range strFields {
		t.Run(f.name, func(t *testing.T) {
			if err := f.set(ctx, nn, 1, f.val); err != nil {
				t.Fatalf("Set%s: %v", f.name, err)
			}
			got, found, err := f.get(ctx, nn, 1)
			if err != nil || !found || got != f.val {
				t.Errorf("Get%s = %q, %v, %v; want %q", f.name, got, found, err, f.val)
			}
		})
	}

	if err := c.SetServiceUnits(ctx, nn, 1, []string{"api", "worker"}); err != nil {
		t.Fatalf("SetServiceUnits: %v", err)
	}
	if v, found, err := c.GetServiceUnits(ctx, nn, 1); err != nil || !found || len(v) != 2 {
		t.Errorf("GetServiceUnits = %v, %v, %v", v, found, err)
	}
}

func TestEnvironmentCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "e1"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestEnvironmentCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestEnvironmentCache(t)
		r := &environmentResolution.ResolvedEnvironment{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("minimal spec publishes mandatory fields only", func(t *testing.T) {
		c := newTestEnvironmentCache(t)
		r := &environmentResolution.ResolvedEnvironment{
			Spec: &environmentResolution.ResolvedEnvironmentSpec{
				ApplicationName: "myapp",
				EnvironmentType: "dev",
				Version:         "v1",
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetApplicationName(context.Background(), nn, 1); !found || v != "myapp" {
			t.Errorf("GetApplicationName = %q, %v", v, found)
		}
		if _, found, _ := c.GetBuild(context.Background(), nn, 1); found {
			t.Error("GetBuild: found = true, want false when Build is empty")
		}
		if _, found, _ := c.GetServiceUnits(context.Background(), nn, 1); found {
			t.Error("GetServiceUnits: found = true, want false for an empty slice")
		}
	})

	t.Run("full spec publishes optional fields too", func(t *testing.T) {
		c := newTestEnvironmentCache(t)
		r := &environmentResolution.ResolvedEnvironment{
			Spec: &environmentResolution.ResolvedEnvironmentSpec{
				ApplicationName: "myapp",
				EnvironmentType: "dev",
				Version:         "v1",
				Contract:        &environmentResolution.ResolvedEnvironmentContract{SecretStoreProvider: "vault"},
				Build:           "build1",
				GitRepository:   "repo1",
				GitHubEvent:     "event1",
				Deployment:      "deploy1",
				Route:           "route1",
				Package:         "pkg1",
				ServiceUnits:    []string{"api"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetSecretStoreProvider(context.Background(), nn, 1); !found || v != "vault" {
			t.Errorf("GetSecretStoreProvider = %q, %v", v, found)
		}
		if v, found, _ := c.GetBuild(context.Background(), nn, 1); !found || v != "build1" {
			t.Errorf("GetBuild = %q, %v", v, found)
		}
		if v, found, _ := c.GetServiceUnits(context.Background(), nn, 1); !found || len(v) != 1 {
			t.Errorf("GetServiceUnits = %v, %v", v, found)
		}
	})
}
