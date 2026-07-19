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

package domain

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	domainResolution "github.com/blanketops/environments/resolution/domain/resolve"
)

func newTestDomainCache(t *testing.T) *DomainCache {
	t.Helper()
	return NewDomainCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestDomainCache_FieldRoundTrips(t *testing.T) {
	c := newTestDomainCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "d1"}

	if err := c.SetHost(ctx, nn, 1, "app.dev.blanketops.online"); err != nil {
		t.Fatalf("SetHost: %v", err)
	}
	if v, found, err := c.GetHost(ctx, nn, 1); err != nil || !found || v != "app.dev.blanketops.online" {
		t.Errorf("GetHost = %q, %v, %v", v, found, err)
	}

	if err := c.SetRouteRef(ctx, nn, 1, "route1"); err != nil {
		t.Fatalf("SetRouteRef: %v", err)
	}
	if v, found, err := c.GetRouteRef(ctx, nn, 1); err != nil || !found || v != "route1" {
		t.Errorf("GetRouteRef = %q, %v, %v", v, found, err)
	}

	if err := c.SetTLSStrategy(ctx, nn, 1, "platform"); err != nil {
		t.Fatalf("SetTLSStrategy: %v", err)
	}
	if v, found, err := c.GetTLSStrategy(ctx, nn, 1); err != nil || !found || v != "platform" {
		t.Errorf("GetTLSStrategy = %q, %v, %v", v, found, err)
	}

	if err := c.SetMTLSEnforced(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetMTLSEnforced: %v", err)
	}
	if v, found, err := c.GetMTLSEnforced(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetMTLSEnforced = %v, %v, %v", v, found, err)
	}

	if err := c.SetRenewBefore(ctx, nn, 1, "720h"); err != nil {
		t.Fatalf("SetRenewBefore: %v", err)
	}
	if v, found, err := c.GetRenewBefore(ctx, nn, 1); err != nil || !found || v != "720h" {
		t.Errorf("GetRenewBefore = %q, %v, %v", v, found, err)
	}

	if err := c.SetDomainReady(ctx, nn, 1, true); err != nil {
		t.Fatalf("SetDomainReady: %v", err)
	}
	if v, found, err := c.GetDomainReady(ctx, nn, 1); err != nil || !found || !v {
		t.Errorf("GetDomainReady = %v, %v, %v", v, found, err)
	}

	if err := c.SetCertificateRefName(ctx, nn, 1, "cert1"); err != nil {
		t.Fatalf("SetCertificateRefName: %v", err)
	}
	if v, found, err := c.GetCertificateRefName(ctx, nn, 1); err != nil || !found || v != "cert1" {
		t.Errorf("GetCertificateRefName = %q, %v, %v", v, found, err)
	}

	if err := c.SetCertificateRefNamespace(ctx, nn, 1, "default"); err != nil {
		t.Fatalf("SetCertificateRefNamespace: %v", err)
	}
	if v, found, err := c.GetCertificateRefNamespace(ctx, nn, 1); err != nil || !found || v != "default" {
		t.Errorf("GetCertificateRefNamespace = %q, %v, %v", v, found, err)
	}

	if err := c.SetDomainMappingRefName(ctx, nn, 1, "mapping1"); err != nil {
		t.Fatalf("SetDomainMappingRefName: %v", err)
	}
	if v, found, err := c.GetDomainMappingRefName(ctx, nn, 1); err != nil || !found || v != "mapping1" {
		t.Errorf("GetDomainMappingRefName = %q, %v, %v", v, found, err)
	}

	if err := c.SetDomainMappingRefNamespace(ctx, nn, 1, "default"); err != nil {
		t.Fatalf("SetDomainMappingRefNamespace: %v", err)
	}
	if v, found, err := c.GetDomainMappingRefNamespace(ctx, nn, 1); err != nil || !found || v != "default" {
		t.Errorf("GetDomainMappingRefNamespace = %q, %v, %v", v, found, err)
	}
}

func TestDomainCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "d1"}
	platform := domainResolution.TLSStrategyPlatform

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestDomainCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestDomainCache(t)
		r := &domainResolution.ResolvedDomain{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("minimal spec publishes mandatory fields only", func(t *testing.T) {
		c := newTestDomainCache(t)
		r := &domainResolution.ResolvedDomain{
			Spec: &domainResolution.ResolvedDomainSpec{
				Host:     "app.dev.blanketops.online",
				RouteRef: domainResolution.ResolvedDomainRouteRef{Name: "route1"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetHost(context.Background(), nn, 1); !found || v != "app.dev.blanketops.online" {
			t.Errorf("GetHost = %q, %v", v, found)
		}
		if _, found, _ := c.GetTLSStrategy(context.Background(), nn, 1); found {
			t.Error("GetTLSStrategy: found = true, want false when TLSStrategy is nil")
		}
		if _, found, _ := c.GetRenewBefore(context.Background(), nn, 1); found {
			t.Error("GetRenewBefore: found = true, want false when RenewBefore is empty")
		}
	})

	t.Run("full spec publishes optional fields too", func(t *testing.T) {
		c := newTestDomainCache(t)
		r := &domainResolution.ResolvedDomain{
			Spec: &domainResolution.ResolvedDomainSpec{
				Host:        "app.dev.blanketops.online",
				RouteRef:    domainResolution.ResolvedDomainRouteRef{Name: "route1"},
				TLSStrategy: &platform,
				MTLS:        domainResolution.ResolvedDomainMTLS{Enforced: true},
				RenewBefore: "720h",
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if v, found, _ := c.GetTLSStrategy(context.Background(), nn, 1); !found || v != "platform" {
			t.Errorf("GetTLSStrategy = %q, %v", v, found)
		}
		if v, found, _ := c.GetMTLSEnforced(context.Background(), nn, 1); !found || !v {
			t.Errorf("GetMTLSEnforced = %v, %v", v, found)
		}
		if v, found, _ := c.GetRenewBefore(context.Background(), nn, 1); !found || v != "720h" {
			t.Errorf("GetRenewBefore = %q, %v", v, found)
		}
	})
}

func TestDomainCache_PublishStatus(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "d1"}

	t.Run("only domainReady when no refs materialised yet", func(t *testing.T) {
		c := newTestDomainCache(t)
		if err := c.PublishStatus(context.Background(), nn, 1, false, "", "", "", ""); err != nil {
			t.Fatalf("PublishStatus: %v", err)
		}
		if v, found, _ := c.GetDomainReady(context.Background(), nn, 1); !found || v {
			t.Errorf("GetDomainReady = %v, %v; want false, true", v, found)
		}
		if _, found, _ := c.GetCertificateRefName(context.Background(), nn, 1); found {
			t.Error("GetCertificateRefName: found = true, want false when certRefName is empty")
		}
		if _, found, _ := c.GetDomainMappingRefName(context.Background(), nn, 1); found {
			t.Error("GetDomainMappingRefName: found = true, want false when mappingRefName is empty")
		}
	})

	t.Run("publishes cert and mapping refs once materialised", func(t *testing.T) {
		c := newTestDomainCache(t)
		if err := c.PublishStatus(context.Background(), nn, 1, true, "cert1", "default", "mapping1", "default"); err != nil {
			t.Fatalf("PublishStatus: %v", err)
		}
		if v, found, _ := c.GetDomainReady(context.Background(), nn, 1); !found || !v {
			t.Errorf("GetDomainReady = %v, %v; want true, true", v, found)
		}
		if v, found, _ := c.GetCertificateRefName(context.Background(), nn, 1); !found || v != "cert1" {
			t.Errorf("GetCertificateRefName = %q, %v", v, found)
		}
		if v, found, _ := c.GetCertificateRefNamespace(context.Background(), nn, 1); !found || v != "default" {
			t.Errorf("GetCertificateRefNamespace = %q, %v", v, found)
		}
		if v, found, _ := c.GetDomainMappingRefName(context.Background(), nn, 1); !found || v != "mapping1" {
			t.Errorf("GetDomainMappingRefName = %q, %v", v, found)
		}
		if v, found, _ := c.GetDomainMappingRefNamespace(context.Background(), nn, 1); !found || v != "default" {
			t.Errorf("GetDomainMappingRefNamespace = %q, %v", v, found)
		}
	})
}
