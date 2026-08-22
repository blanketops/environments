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

/*
Package cache provides the generic, reusable half of the external cache
layer: ObjectCache, a namespace- and generation-scoped field cache for a
single CR kind, plus NewExternal, the single composition point that picks
and constructs the concrete backend (Redis, Memcached, or a no-op) from
Options.

ObjectCache is deliberately generic — it knows nothing about any specific
CR's spec shape. The per-kind subpackages (cache/build, cache/deployment,
cache/domain, and siblings) embed it and add typed Set* / Get* helpers
plus a PublishResolved method that projects a resolved contract into
individual fields.

Every read is a fast-path optimization, never a correctness dependency: a
miss or backend error always falls through to the informer cache (see
core/cache), so this package's failure modes cost queryability, not
correctness.
*/
package cache

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	corecache "github.com/blanketops/environments/core/cache"
)

// ObjectCache provides namespace-aware, generation-scoped field caching
// for a single CR kind. Domain caches embed this and add typed helpers.
//
// Keys: <kind>:<namespace>/<name>:gen<generation>:<field>
// Generation scoping means a spec change naturally orphans stale entries
// (they expire via TTL) — no invalidation hook required, and it works
// identically on backends without key enumeration (Memcached).
type ObjectCache struct {
	cache *corecache.Cache
	kind  string
	ttl   time.Duration
}

// NewObjectCache constructs an ObjectCache scoped to the given CR kind,
// with fields expiring after ttl (0 disables expiry).
func NewObjectCache(c *corecache.Cache, kind string, ttl time.Duration) *ObjectCache {
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}
	return &ObjectCache{cache: c, kind: kind, ttl: ttl}
}

func (o *ObjectCache) key(nn types.NamespacedName, gen int64, field string) string {
	return fmt.Sprintf("%s:%s/%s:gen%d:%s", o.kind, nn.Namespace, nn.Name, gen, field)
}

// prefix covers every field of every generation for one object.
func (o *ObjectCache) prefix(nn types.NamespacedName) string {
	return fmt.Sprintf("%s:%s/%s:", o.kind, nn.Namespace, nn.Name)
}

// SetField stores an individual value scoped to the object's generation.
func (o *ObjectCache) SetField(ctx context.Context, nn types.NamespacedName, gen int64, field string, val any) error {
	return o.cache.External.Set(ctx, o.key(nn, gen, field), val, o.ttl)
}

// GetField retrieves a value for the given generation. (false, nil) is a
// miss — fall through to the informer cache.
func (o *ObjectCache) GetField(ctx context.Context, nn types.NamespacedName, gen int64, field string, into any) (bool, error) {
	return o.cache.External.Get(ctx, o.key(nn, gen, field), into)
}

// Invalidate drops all cached fields for an object (all generations).
// Call on delete events. Backends without enumeration no-op; generation
// scoping already protects correctness there.
func (o *ObjectCache) Invalidate(ctx context.Context, nn types.NamespacedName) error {
	return o.cache.External.DelPrefix(ctx, o.prefix(nn))
}
