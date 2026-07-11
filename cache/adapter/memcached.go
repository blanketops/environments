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

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"

	"github.com/BlanketOps/environments/core"
)

// MemcachedConfig holds connection settings for the Memcached backend.
type MemcachedConfig struct {
	Servers []string // host:port
}

// MemcachedCache implements core.ExternalCache backed by Memcached.
type MemcachedCache struct {
	client *memcache.Client
}

var _ core.ExternalCache = (*MemcachedCache)(nil)

func NewMemcached(cfg MemcachedConfig) *MemcachedCache {
	return &MemcachedCache{client: memcache.New(cfg.Servers...)}
}

func (c *MemcachedCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.client.Set(&memcache.Item{
		Key:        key,
		Value:      b,
		Expiration: int32(ttl.Seconds()),
	})
}

func (c *MemcachedCache) Get(ctx context.Context, key string, into any) (bool, error) {
	item, err := c.client.Get(key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return false, nil // miss, not an error
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(item.Value, into)
}

func (c *MemcachedCache) Del(ctx context.Context, key string) error {
	err := c.client.Delete(key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil // already gone
	}
	return err
}

// DelPrefix: Memcached cannot enumerate keys, so prefix invalidation is
// a no-op here. Generation-scoped keys (see cache.ObjectCache) are the
// correctness guarantee on this backend — stale entries expire via TTL.
func (c *MemcachedCache) DelPrefix(ctx context.Context, prefix string) error {
	return nil
}
