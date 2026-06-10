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

	"github.com/bradfitz/gomemcache/memcache" // Standard Go client
)

// MemcachedExternalCache implements the ExternalCache interface using Memcached.
type MemcachedExternalCache struct {
	client *memcache.Client
}

func NewMemcachedExternalCache(servers ...string) *MemcachedExternalCache {
	return &MemcachedExternalCache{client: memcache.New(servers...)}
}

func (m *MemcachedExternalCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := marshalJSON(val)
	if err != nil {
		return err
	}
	// Memcached TTL is in seconds
	return m.client.Set(&memcache.Item{Key: key, Value: b, Expiration: int32(ttl.Seconds())})
}

func (m *MemcachedExternalCache) Get(ctx context.Context, key string, into any) (bool, error) {
	item, err := m.client.Get(key)
	if err == memcache.ErrCacheMiss {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, unmarshalJSON(item.Value, into)
}

func (m *MemcachedExternalCache) Del(ctx context.Context, key string) error {
	return m.client.Delete(key)
}
