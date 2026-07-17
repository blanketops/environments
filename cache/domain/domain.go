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

	bocache "github.com/blanketops/environments/cache"
	"github.com/blanketops/environments/core/cache"
	domainResolution "github.com/blanketops/environments/resolution/domain"
)

// DomainCache provides domain-specific, field-level caching for Domain resources.
type DomainCache struct {
	*bocache.ObjectCache
}

// NewDomainCache constructs a new DomainCache with the provided cache.Cache.
func NewDomainCache(c *cache.Cache) *DomainCache {
	return &DomainCache{ObjectCache: bocache.NewObjectCache(c, "domain", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers — Spec fields
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

// -----------------------------------------------------------------------------
// Typed Helpers — Status fields
//
// Written by the controller after materialising resources — not during
// resolution. The Route mediator reads certificateRef to gate DomainMapping
// dispatch on TLS secret existence without an API server lookup.
// -----------------------------------------------------------------------------

// SetDomainReady caches the scalar domainReady bool from DomainStatus.
// True once both DomainClaim and DomainMapping are reconciled and active.
func (d *DomainCache) SetDomainReady(ctx context.Context, nn types.NamespacedName, gen int64, ready bool) error {
	return d.SetField(ctx, nn, gen, "domainReady", ready)
}

func (d *DomainCache) GetDomainReady(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var r bool
	found, err := d.GetField(ctx, nn, gen, "domainReady", &r)
	return r, found, err
}

// SetCertificateRefName caches the Certificate resource name emitted by the
// controller. Set only for custom TLS strategy — empty for platform strategy.
func (d *DomainCache) SetCertificateRefName(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return d.SetField(ctx, nn, gen, "certificateRefName", name)
}

func (d *DomainCache) GetCertificateRefName(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var n string
	found, err := d.GetField(ctx, nn, gen, "certificateRefName", &n)
	return n, found, err
}

// SetCertificateRefNamespace caches the Certificate resource namespace.
// For custom strategy this is always the Domain's own namespace.
func (d *DomainCache) SetCertificateRefNamespace(ctx context.Context, nn types.NamespacedName, gen int64, namespace string) error {
	return d.SetField(ctx, nn, gen, "certificateRefNamespace", namespace)
}

func (d *DomainCache) GetCertificateRefNamespace(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var ns string
	found, err := d.GetField(ctx, nn, gen, "certificateRefNamespace", &ns)
	return ns, found, err
}

// SetDomainMappingRefName caches the Knative DomainMapping resource name
// emitted by the controller. Set once materialised, cleared on deletion.
func (d *DomainCache) SetDomainMappingRefName(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return d.SetField(ctx, nn, gen, "domainMappingRefName", name)
}

func (d *DomainCache) GetDomainMappingRefName(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var n string
	found, err := d.GetField(ctx, nn, gen, "domainMappingRefName", &n)
	return n, found, err
}

// SetDomainMappingRefNamespace caches the Knative DomainMapping namespace.
// Always the Domain's own namespace — cached for mediator fast-path reads.
func (d *DomainCache) SetDomainMappingRefNamespace(ctx context.Context, nn types.NamespacedName, gen int64, namespace string) error {
	return d.SetField(ctx, nn, gen, "domainMappingRefNamespace", namespace)
}

func (d *DomainCache) GetDomainMappingRefNamespace(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var ns string
	found, err := d.GetField(ctx, nn, gen, "domainMappingRefNamespace", &ns)
	return ns, found, err
}

// -----------------------------------------------------------------------------
// Publish helpers
// -----------------------------------------------------------------------------

// PublishResolved writes the resolved spec contract as a generation-scoped,
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

// PublishStatus writes controller-observed status fields after resource
// materialisation. Separate from PublishResolved — these are never derived
// from spec; they reflect what the controller actually created.
// The Route mediator reads certificateRef from here to gate DomainMapping
// dispatch without an additional API server lookup.
// All writes are best-effort — callers must not fail reconciliation on errors.
func (d *DomainCache) PublishStatus(
	ctx context.Context,
	nn types.NamespacedName,
	gen int64,
	domainReady bool,
	certRefName, certRefNamespace string,
	mappingRefName, mappingRefNamespace string,
) error {
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	record(d.SetDomainReady(ctx, nn, gen, domainReady))

	// certificateRef — custom strategy only; empty strings are valid no-ops
	// for platform strategy (no cert emitted).
	if certRefName != "" {
		record(d.SetCertificateRefName(ctx, nn, gen, certRefName))
		record(d.SetCertificateRefNamespace(ctx, nn, gen, certRefNamespace))
	}

	// domainMappingRef — set once DomainMapping is materialised.
	if mappingRefName != "" {
		record(d.SetDomainMappingRefName(ctx, nn, gen, mappingRefName))
		record(d.SetDomainMappingRefNamespace(ctx, nn, gen, mappingRefNamespace))
	}

	return firstErr
}
