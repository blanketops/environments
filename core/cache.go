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
Package core defines the foundational contracts and primitives shared across
all BlanketOps Environments domains.

This file owns the cache layer. There are two distinct caching concerns in
the platform:

 1. The informer cache — the controller-runtime shared cache backed by
    Kubernetes API watches. This is the SOURCE OF TRUTH for all CR reads.
    Correctness-bearing reads (cross-CR lookups, prerequisite checks) always
    go here via client.Reader.

 2. The external cache — an optional distributed backend (Redis, Memcached)
    used as a WRITE-THROUGH PROJECTION LAYER for resolved contracts. This is
    advisory only. A miss or error always falls through to full computation.
    Correctness never depends on a hit.

The two layers are composed in Cache and never cross-wired: the informer
cache provides correctness, the external cache provides observability and
opportunistic reuse within a generation.
*/
package core

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExternalCache defines the distributed cache contract for resolved contract
// projections. Implementations (Redis, Memcached) live under cache/adapter/ —
// core never imports backend libraries directly. Serialization is the
// implementation's responsibility.
//
// Semantics:
//   - Get returning (false, nil) is a MISS, not an error. Callers MUST treat
//     the external cache as advisory and fall through to full computation on
//     any miss.
//   - Errors from any method are infrastructure failures only. They MUST
//     degrade gracefully — log and continue — never abort reconciliation.
//   - Keys should be generation-scoped (kind:namespace/name:genN:field) to
//     guarantee the happy path always misses on spec change. See ObjectCache
//     in cache/ for the canonical key schema.
type ExternalCache interface {
	Set(ctx context.Context, key string, val any, ttl time.Duration) error

	Get(ctx context.Context, key string, into any) (found bool, err error)

	Del(ctx context.Context, key string) error
	// DelPrefix removes all keys sharing the given prefix. Backends without
	// key enumeration (e.g. Memcached) may implement this as a no-op.
	// Callers should prefer generation-scoped keys over relying on prefix
	// invalidation for correctness guarantees.
	DelPrefix(ctx context.Context, prefix string) error
}

// NoopExternalCache is the zero-value external cache. It satisfies the
// ExternalCache interface and safely ignores all calls. Every Get is a miss.
// Used when no external backend is configured so the rest of the platform
// needs no nil checks.
type NoopExternalCache struct{}

func (NoopExternalCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (NoopExternalCache) Get(context.Context, string, any) (bool, error)        { return false, nil }
func (NoopExternalCache) Del(context.Context, string) error                     { return nil }
func (NoopExternalCache) DelPrefix(context.Context, string) error               { return nil }

// Cache composes the controller-runtime shared informer cache (source of truth)
// with an optional external cache (opportunistic projection layer).
//
// The two layers serve different purposes and must not be confused:
//   - Reader: informer-backed, always consistent with the API server via watches.
//     Use for all correctness-bearing reads.
//   - External: distributed backend, advisory only. Use for resolved contract
//     projections where a stale or missing value is safe to recompute.
type Cache struct {
	// Reader is the informer-backed source of truth for all CR reads.
	// Injected from the controller-runtime manager cache.
	Reader client.Reader

	// Indexer allows domains to register field indexes on the informer cache
	// for efficient filtered list queries.
	Indexer client.FieldIndexer

	// External is the optional distributed cache for resolved contract
	// projections. Never nil — falls back to NoopExternalCache if unconfigured.
	External ExternalCache
}

// NewCache constructs a Cache from the controller-runtime manager and an
// optional external backend. A nil external is silently replaced with
// NoopExternalCache and logged so a misconfigured backend is never an
// invisible failure.
func NewCache(mgr ctrl.Manager, external ExternalCache) *Cache {
	log := ctrl.Log.WithName("core.cache")
	if external == nil {
		external = NoopExternalCache{}
	}
	if _, isNoop := external.(NoopExternalCache); isNoop {
		log.Info("external cache: noop (informer cache only)")
	} else {
		log.Info("external cache: enabled")
	}
	return &Cache{
		Reader:   mgr.GetCache(),
		Indexer:  mgr.GetFieldIndexer(),
		External: external,
	}
}

// IndexField registers a field extractor on the informer cache, enabling
// efficient MatchingFields queries via ListByField. Must be called during
// manager setup before the cache starts.
func (c *Cache) IndexField(ctx context.Context, obj client.Object, field string, extract func(client.Object) []string) error {
	return c.Indexer.IndexField(ctx, obj, field, extract)
}

// ListByField lists objects from the informer cache matching a single
// field=value pair. The field must have been registered via IndexField
// before the cache started.
//
// Example:
//
//	var builds buildv1.BuildList
//	err := core.ListByField(ctx, cache.Reader, &builds, ".spec.repo", "github.com/foo/bar")
func ListByField[T client.ObjectList](ctx context.Context, rdr client.Reader, list T, field, value string) error {
	return rdr.List(ctx, list, client.MatchingFields{field: value})
}

// WaitForSync blocks until the informer cache has completed its initial sync
// against the API server. Called during manager startup to ensure reconcilers
// do not operate against a partially populated cache.
func WaitForSync(ctx context.Context, c cache.Cache) bool {
	return c.WaitForCacheSync(ctx)
}
