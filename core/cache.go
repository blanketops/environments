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

package core

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExternalCache defines a simple distributed cache contract.
// Implementations (Redis, Memcached) live under cache/ — core never
// imports backend libraries. Serialization is the implementation's
// responsibility.
//
// Semantics:
//   - Get returning (false, nil) is a MISS, not an error. Callers MUST
//     treat the external cache as advisory and fall through to Reader.
//   - Errors from any method are infrastructure failures only and should
//     degrade, never abort, reconciliation.
type ExternalCache interface {
	Set(ctx context.Context, key string, val any, ttl time.Duration) error
	Get(ctx context.Context, key string, into any) (found bool, err error)
	Del(ctx context.Context, key string) error
	// DelPrefix removes all keys sharing the given prefix. Backends that
	// cannot enumerate keys (e.g. Memcached) may implement this as a
	// no-op; callers should prefer generation-scoped keys over relying
	// on prefix invalidation.
	DelPrefix(ctx context.Context, prefix string) error
}

// NoopExternalCache safely ignores all calls (used when no external
// backend is configured). Every Get is a miss.
type NoopExternalCache struct{}

func (NoopExternalCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (NoopExternalCache) Get(context.Context, string, any) (bool, error)        { return false, nil }
func (NoopExternalCache) Del(context.Context, string) error                     { return nil }
func (NoopExternalCache) DelPrefix(context.Context, string) error               { return nil }

// Cache wraps controller-runtime's shared informer cache (source of
// truth) with an optional external cache (opportunistic layer).
type Cache struct {
	Reader   client.Reader
	Indexer  client.FieldIndexer
	External ExternalCache
}

// NewCache wires the shared informer cache + external cache. A nil
// external falls back to Noop — logged so a misconfigured backend is
// never an invisible failure.
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

func (c *Cache) IndexField(ctx context.Context, obj client.Object, field string, extract func(client.Object) []string) error {
	return c.Indexer.IndexField(ctx, obj, field, extract)
}

// ListByField lists objects from the informer cache by a field=value pair.
//
// Example:
//
//	var builds buildv1.BuildList
//	err := core.ListByField(ctx, cache.Reader, &builds, ".spec.repo", "github.com/foo/bar")
func ListByField[T client.ObjectList](ctx context.Context, rdr client.Reader, list T, field, value string) error {
	return rdr.List(ctx, list, client.MatchingFields{field: value})
}

// WaitForSync ensures the informer cache is ready before reconciliation begins.
func WaitForSync(ctx context.Context, c cache.Cache) bool {
	return c.WaitForCacheSync(ctx)
}
