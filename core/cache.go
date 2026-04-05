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
	"encoding/json"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExternalCache defines a simple distributed cache contract.
// You can plug in Redis, Memcached, etc. It’s safe to no-op.
type ExternalCache interface {
	Set(ctx context.Context, key string, val any, ttl time.Duration) error
	Get(ctx context.Context, key string, into any) (found bool, err error)
	Del(ctx context.Context, key string) error
}

// NoopExternalCache safely ignores all calls (used when Redis is unavailable).
type NoopExternalCache struct{}

func (NoopExternalCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	return nil
}
func (NoopExternalCache) Get(ctx context.Context, key string, into any) (bool, error) {
	return false, nil
}
func (NoopExternalCache) Del(ctx context.Context, key string) error { return nil }

// Cache wraps controller-runtime's shared cache with optional external cache.
type Cache struct {
	Reader   client.Reader
	Indexer  client.FieldIndexer
	External ExternalCache
}

// NewCache wires the shared informer cache + external cache (Redis or Noop).
func NewCache(mgr ctrl.Manager, external ExternalCache) *Cache {
	if external == nil {
		external = NoopExternalCache{}
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

// ListByField lists objects from the cache by a field=value pair.
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

// Helper JSON marshal/unmarshal for external cache values.
func marshalJSON(v any) ([]byte, error)      { return json.Marshal(v) }
func unmarshalJSON(b []byte, into any) error { return json.Unmarshal(b, into) }
