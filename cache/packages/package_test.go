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

package packages

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	packagesResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

func newTestPackageCache(t *testing.T) *PackageCache {
	t.Helper()
	return NewPackageCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestPackageCache_FieldRoundTrips(t *testing.T) {
	c := newTestPackageCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "p1"}

	if err := c.SetVersion(ctx, nn, 1, "v1.2.3"); err != nil {
		t.Fatalf("SetVersion: %v", err)
	}
	if v, found, err := c.GetVersion(ctx, nn, 1); err != nil || !found || v != "v1.2.3" {
		t.Errorf("GetVersion = %q, %v, %v", v, found, err)
	}

	repo := packagesResolution.ResolvedPackageRepository{URL: "oci://x", CredentialsSecret: "s1"}
	if err := c.SetPackageRepository(ctx, nn, 1, repo); err != nil {
		t.Fatalf("SetPackageRepository: %v", err)
	}
	var gotRepo packagesResolution.ResolvedPackageRepository
	if found, err := c.GetPackageRepository(ctx, nn, 1, &gotRepo); err != nil || !found || gotRepo != repo {
		t.Errorf("GetPackageRepository = %+v, %v, %v", gotRepo, found, err)
	}

	state := packagesResolution.ResolvedStateRepository{URL: "https://git/x"}
	if err := c.SetStateRepo(ctx, nn, 1, state); err != nil {
		t.Fatalf("SetStateRepo: %v", err)
	}
	var gotState packagesResolution.ResolvedStateRepository
	if found, err := c.GetStateRepo(ctx, nn, 1, &gotState); err != nil || !found || gotState.URL != "https://git/x" {
		t.Errorf("GetStateRepo = %+v, %v, %v", gotState, found, err)
	}

	if err := c.SetKappDiff(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetKappDiff: %v", err)
	}
	if v, found, err := c.GetKappDiff(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetKappDiff = %v, %v, %v", v, found, err)
	}

	if err := c.SetEnabled(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if v, found, err := c.GetEnabled(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetEnabled = %v, %v, %v", v, found, err)
	}

	if err := c.SetChecksum(ctx, nn, 1, "sha256:abc"); err != nil {
		t.Fatalf("SetChecksum: %v", err)
	}
	if v, found, err := c.GetChecksum(ctx, nn, 1); err != nil || !found || v != "sha256:abc" {
		t.Errorf("GetChecksum = %q, %v, %v", v, found, err)
	}
}

func TestPackageCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "p1"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestPackageCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestPackageCache(t)
		r := &packagesResolution.ResolvedPackage{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("zero-value PackageRepository is not published", func(t *testing.T) {
		c := newTestPackageCache(t)
		r := &packagesResolution.ResolvedPackage{
			Spec: &packagesResolution.ResolvedPackageSpec{
				Version: "v1",
				Enabled: true,
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetVersion(context.Background(), nn, 1); !found || v != "v1" {
			t.Errorf("GetVersion = %q, %v", v, found)
		}
		var repo packagesResolution.ResolvedPackageRepository
		if found, _ := c.GetPackageRepository(context.Background(), nn, 1, &repo); found {
			t.Error("GetPackageRepository: found = true, want false for a zero-value repository")
		}
	})

	t.Run("non-zero PackageRepository is published", func(t *testing.T) {
		c := newTestPackageCache(t)
		r := &packagesResolution.ResolvedPackage{
			Spec: &packagesResolution.ResolvedPackageSpec{
				Version:           "v1",
				Enabled:           true,
				PackageRepository: packagesResolution.ResolvedPackageRepository{URL: "oci://x"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		var repo packagesResolution.ResolvedPackageRepository
		if found, _ := c.GetPackageRepository(context.Background(), nn, 1, &repo); !found || repo.URL != "oci://x" {
			t.Errorf("GetPackageRepository = %+v, %v", repo, found)
		}
	})

	t.Run("DiffEnabled and non-nil StateRepository are published", func(t *testing.T) {
		c := newTestPackageCache(t)
		r := &packagesResolution.ResolvedPackage{
			Spec: &packagesResolution.ResolvedPackageSpec{
				Version:         "v1",
				Enabled:         true,
				DiffEnabled:     true,
				StateRepository: &packagesResolution.ResolvedStateRepository{URL: "https://git.example/state"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetKappDiff(context.Background(), nn, 1); !found || !v {
			t.Errorf("GetKappDiff = %v, %v", v, found)
		}
		var stateRepo packagesResolution.ResolvedStateRepository
		if found, _ := c.GetStateRepo(context.Background(), nn, 1, &stateRepo); !found || stateRepo.URL != "https://git.example/state" {
			t.Errorf("GetStateRepo = %+v, %v", stateRepo, found)
		}
	})

	t.Run("nil StateRepository is not published", func(t *testing.T) {
		c := newTestPackageCache(t)
		r := &packagesResolution.ResolvedPackage{
			Spec: &packagesResolution.ResolvedPackageSpec{
				Version: "v1",
				Enabled: true,
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		var stateRepo packagesResolution.ResolvedStateRepository
		if found, _ := c.GetStateRepo(context.Background(), nn, 1, &stateRepo); found {
			t.Error("GetStateRepo: found = true, want false for a nil StateRepository")
		}
	})
}
