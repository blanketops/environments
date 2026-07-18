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

package route

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	routeResolution "github.com/blanketops/environments/resolution/route/resolve"
)

func newTestRouteCache(t *testing.T) *RouteCache {
	t.Helper()
	return NewRouteCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestRouteCache_FieldRoundTrips(t *testing.T) {
	c := newTestRouteCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "r1"}

	if err := c.SetHost(ctx, nn, 1, "app.dev.blanketops.online"); err != nil {
		t.Fatalf("SetHost: %v", err)
	}
	if v, found, err := c.GetHost(ctx, nn, 1); err != nil || !found || v != "app.dev.blanketops.online" {
		t.Errorf("GetHost = %q, %v, %v", v, found, err)
	}

	if err := c.SetEnabled(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if v, found, err := c.GetEnabled(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetEnabled = %v, %v, %v", v, found, err)
	}

	if err := c.SetPath(ctx, nn, 1, "/api"); err != nil {
		t.Fatalf("SetPath: %v", err)
	}
	if v, found, err := c.GetPath(ctx, nn, 1); err != nil || !found || v != "/api" {
		t.Errorf("GetPath = %q, %v, %v", v, found, err)
	}

	if err := c.SetTLSEnabled(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetTLSEnabled: %v", err)
	}
	if v, found, err := c.GetTLSEnabled(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetTLSEnabled = %v, %v, %v", v, found, err)
	}

	if err := c.SetRuntime(ctx, nn, 1, "knative-service"); err != nil {
		t.Fatalf("SetRuntime: %v", err)
	}
	if v, found, err := c.GetRuntime(ctx, nn, 1); err != nil || !found || v != "knative-service" {
		t.Errorf("GetRuntime = %q, %v, %v", v, found, err)
	}

	if err := c.SetServiceUnitRefName(ctx, nn, 1, "api"); err != nil {
		t.Fatalf("SetServiceUnitRefName: %v", err)
	}
	if v, found, err := c.GetServiceUnitRefName(ctx, nn, 1); err != nil || !found || v != "api" {
		t.Errorf("GetServiceUnitRefName = %q, %v, %v", v, found, err)
	}
}

func TestRouteCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "r1"}
	runtime := routeResolution.RuntimeKnativeService

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestRouteCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestRouteCache(t)
		r := &routeResolution.ResolvedRoute{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("minimal spec publishes mandatory fields only", func(t *testing.T) {
		c := newTestRouteCache(t)
		r := &routeResolution.ResolvedRoute{
			Spec: &routeResolution.ResolvedRouteSpec{
				Host:    "app.dev.blanketops.online",
				Enabled: true,
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetHost(context.Background(), nn, 1); !found || v != "app.dev.blanketops.online" {
			t.Errorf("GetHost = %q, %v", v, found)
		}
		if _, found, _ := c.GetRuntime(context.Background(), nn, 1); found {
			t.Error("GetRuntime: found = true, want false when Runtime is nil")
		}
		if _, found, _ := c.GetServiceUnitRefName(context.Background(), nn, 1); found {
			t.Error("GetServiceUnitRefName: found = true, want false when ServiceUnitRef is nil")
		}
	})

	t.Run("full spec publishes optional fields too", func(t *testing.T) {
		c := newTestRouteCache(t)
		r := &routeResolution.ResolvedRoute{
			Spec: &routeResolution.ResolvedRouteSpec{
				Host:           "app.dev.blanketops.online",
				Enabled:        true,
				Path:           "/api",
				TLSEnabled:     true,
				Runtime:        &runtime,
				ServiceUnitRef: &routeResolution.ResolvedServiceUnitRef{Name: "api"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetRuntime(context.Background(), nn, 1); !found || v != "knative-service" {
			t.Errorf("GetRuntime = %q, %v", v, found)
		}
		if v, found, _ := c.GetServiceUnitRefName(context.Background(), nn, 1); !found || v != "api" {
			t.Errorf("GetServiceUnitRefName = %q, %v", v, found)
		}
	})

	t.Run("ServiceUnitRef with empty name is not published", func(t *testing.T) {
		c := newTestRouteCache(t)
		r := &routeResolution.ResolvedRoute{
			Spec: &routeResolution.ResolvedRouteSpec{
				Host:           "app.dev.blanketops.online",
				ServiceUnitRef: &routeResolution.ResolvedServiceUnitRef{Name: ""},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if _, found, _ := c.GetServiceUnitRefName(context.Background(), nn, 1); found {
			t.Error("GetServiceUnitRefName: found = true, want false for an empty-name ref")
		}
	})
}
