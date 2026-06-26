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

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/ntlaletsi70/blanketops-environments/cache"
	"github.com/ntlaletsi70/blanketops-environments/core"
	domainResolution "github.com/ntlaletsi70/blanketops-environments/resolution/domain"
)

// DomainCache provides domain-specific, field-level caching for Domain resources.
type DomainCache struct {
	*bocache.ObjectCache
}

// NewDomainCache constructs a new DomainCache with the provided core.Cache.
func NewDomainCache(c *core.Cache) *DomainCache {
	return &DomainCache{ObjectCache: bocache.NewObjectCache(c, "domain", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers
// -----------------------------------------------------------------------------

func (d *DomainCache) SetHost(ctx context.Context, nn types.NamespacedName, gen int64, host string) error {
	return d.SetField(ctx, nn, gen, "host", host)
}
func (d *DomainCache) GetHost(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var h string
	found, err := d.GetField(ctx, nn, gen, "host", &h)
	return h, found, err
}

// SetRouteRef caches the owning Route name from ResolvedDomainRouteRef.Name.
func (d *DomainCache) SetRouteRef(ctx context.Context, nn types.NamespacedName, gen int64, routeRefName string) error {
	return d.SetField(ctx, nn, gen, "routeRef", routeRefName)
}
func (d *DomainCache) GetRouteRef(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var r string
	found, err := d.GetField(ctx, nn, gen, "routeRef", &r)
	return r, found, err
}

func (d *DomainCache) SetTLSStrategy(ctx context.Context, nn types.NamespacedName, gen int64, strategy string) error {
	return d.SetField(ctx, nn, gen, "tlsStrategy", strategy)
}
func (d *DomainCache) GetTLSStrategy(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var s string
	found, err := d.GetField(ctx, nn, gen, "tlsStrategy", &s)
	return s, found, err
}

// SetMTLSEnforced caches the enforced flag from ResolvedDomainMTLS.Enforced.
func (d *DomainCache) SetMTLSEnforced(ctx context.Context, nn types.NamespacedName, gen int64, enforced bool) error {
	return d.SetField(ctx, nn, gen, "mtlsEnforced", enforced)
}
func (d *DomainCache) GetMTLSEnforced(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var e bool
	found, err := d.GetField(ctx, nn, gen, "mtlsEnforced", &e)
	return e, found, err
}

func (d *DomainCache) SetRenewBefore(ctx context.Context, nn types.NamespacedName, gen int64, renewBefore string) error {
	return d.SetField(ctx, nn, gen, "renewBefore", renewBefore)
}
func (d *DomainCache) GetRenewBefore(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var r string
	found, err := d.GetField(ctx, nn, gen, "renewBefore", &r)
	return r, found, err
}

// PublishResolved writes the resolved contract as a generation-scoped,
// field-level projection. All writes are best-effort: failures cost
// queryability, never correctness. Returns the first error encountered
// for optional logging; callers should not fail reconciliation on it.
func (d *DomainCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, res *domainResolution.ResolvedDomain) error {
	if res == nil || res.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	record(d.SetHost(ctx, nn, gen, res.Spec.Host))
	record(d.SetRouteRef(ctx, nn, gen, res.Spec.RouteRef.Name))
	if res.Spec.TLSStrategy != nil {
		record(d.SetTLSStrategy(ctx, nn, gen, string(*res.Spec.TLSStrategy)))
	}
	record(d.SetMTLSEnforced(ctx, nn, gen, res.Spec.MTLS.Enforced))
	if res.Spec.RenewBefore != "" {
		record(d.SetRenewBefore(ctx, nn, gen, res.Spec.RenewBefore))
	}
	return firstErr
}
