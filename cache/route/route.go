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

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/blanketops/environments/cache"
	"github.com/blanketops/environments/core/cache"
	routeResolution "github.com/blanketops/environments/resolution/route/resolve"
)

// RouteCache provides domain-specific, field-level caching for Route resources.
type RouteCache struct {
	*bocache.ObjectCache
}

// NewRouteCache constructs a new RouteCache with the provided cache.Cache.
func NewRouteCache(c *cache.Cache) *RouteCache {
	return &RouteCache{ObjectCache: bocache.NewObjectCache(c, "route", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers
// -----------------------------------------------------------------------------

func (r *RouteCache) SetHost(ctx context.Context, nn types.NamespacedName, gen int64, host string) error {
	return r.SetField(ctx, nn, gen, "host", host)
}

func (r *RouteCache) GetHost(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var h string
	found, err := r.GetField(ctx, nn, gen, "host", &h)
	return h, found, err
}

func (r *RouteCache) SetEnabled(ctx context.Context, nn types.NamespacedName, gen int64, enabled bool) error {
	return r.SetField(ctx, nn, gen, "enabled", enabled)
}

func (r *RouteCache) GetEnabled(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var e bool
	found, err := r.GetField(ctx, nn, gen, "enabled", &e)
	return e, found, err
}

func (r *RouteCache) SetPath(ctx context.Context, nn types.NamespacedName, gen int64, path string) error {
	return r.SetField(ctx, nn, gen, "path", path)
}

func (r *RouteCache) GetPath(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var p string
	found, err := r.GetField(ctx, nn, gen, "path", &p)
	return p, found, err
}

func (r *RouteCache) SetTLSEnabled(ctx context.Context, nn types.NamespacedName, gen int64, tlsEnabled bool) error {
	return r.SetField(ctx, nn, gen, "tlsEnabled", tlsEnabled)
}

func (r *RouteCache) GetTLSEnabled(ctx context.Context, nn types.NamespacedName, gen int64) (bool, bool, error) {
	var t bool
	found, err := r.GetField(ctx, nn, gen, "tlsEnabled", &t)
	return t, found, err
}

func (r *RouteCache) SetRuntime(ctx context.Context, nn types.NamespacedName, gen int64, runtime string) error {
	return r.SetField(ctx, nn, gen, "runtime", runtime)
}

func (r *RouteCache) GetRuntime(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var rt string
	found, err := r.GetField(ctx, nn, gen, "runtime", &rt)
	return rt, found, err
}

// SetServiceUnitRefName caches the ServiceUnit name bound to this Route.
// Stored as a string field — the controller derives the ksvc name by
// convention (ksvc name == ServiceUnit name) without a status lookup.
func (r *RouteCache) SetServiceUnitRefName(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return r.SetField(ctx, nn, gen, "serviceUnitRefName", name)
}

// GetServiceUnitRefName retrieves the cached ServiceUnit name for this Route.
func (r *RouteCache) GetServiceUnitRefName(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var n string
	found, err := r.GetField(ctx, nn, gen, "serviceUnitRefName", &n)
	return n, found, err
}

// PublishResolved writes the resolved contract as a generation-scoped,
// field-level projection. All writes are best-effort: failures cost
// queryability, never correctness. Returns the first error encountered
// for optional logging; callers should not fail reconciliation on it.
func (r *RouteCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, res *routeResolution.ResolvedRoute) error {
	if res == nil || res.Spec == nil {
		return nil
	}

	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	record(r.SetHost(ctx, nn, gen, res.Spec.Host))
	record(r.SetEnabled(ctx, nn, gen, res.Spec.Enabled))
	record(r.SetPath(ctx, nn, gen, res.Spec.Path))
	record(r.SetTLSEnabled(ctx, nn, gen, res.Spec.TLSEnabled))

	if res.Spec.Runtime != nil {
		record(r.SetRuntime(ctx, nn, gen, string(*res.Spec.Runtime)))
	}

	// ServiceUnitRef — cache the name only. Controller derives ksvc name by
	// convention (ksvc name == ServiceUnit name); no status lookup needed.
	if res.Spec.ServiceUnitRef != nil && res.Spec.ServiceUnitRef.Name != "" {
		record(r.SetServiceUnitRefName(ctx, nn, gen, res.Spec.ServiceUnitRef.Name))
	}

	return firstErr
}
