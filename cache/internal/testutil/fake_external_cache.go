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
Package testutil provides a shared in-memory core/cache.ExternalCache fake
for cache/* package tests. Importable only from packages rooted under
cache/, per Go's internal-package visibility rule.
*/
package testutil

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// FakeExternalCache is an in-memory, JSON-serializing implementation of
// core/cache.ExternalCache. Values are marshaled on Set and unmarshaled on
// Get, matching how the real Redis/Memcached backends behave — a naive
// map[string]any passthrough would hide (de)serialization bugs.
type FakeExternalCache struct {
	mu   sync.Mutex
	data map[string][]byte
	ttl  map[string]time.Duration
}

func NewFakeExternalCache() *FakeExternalCache {
	return &FakeExternalCache{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
}

func (f *FakeExternalCache) Set(_ context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = b
	f.ttl[key] = ttl
	return nil
}

func (f *FakeExternalCache) Get(_ context.Context, key string, into any) (bool, error) {
	f.mu.Lock()
	b, ok := f.data[key]
	f.mu.Unlock()
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(b, into); err != nil {
		return false, err
	}
	return true, nil
}

func (f *FakeExternalCache) Del(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
	delete(f.ttl, key)
	return nil
}

func (f *FakeExternalCache) DelPrefix(_ context.Context, prefix string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k := range f.data {
		if strings.HasPrefix(k, prefix) {
			delete(f.data, k)
			delete(f.ttl, k)
		}
	}
	return nil
}

// Keys returns a snapshot of every key currently stored, for assertions on
// exact key formatting.
func (f *FakeExternalCache) Keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := make([]string, 0, len(f.data))
	for k := range f.data {
		keys = append(keys, k)
	}
	return keys
}

// TTL returns the TTL passed to the most recent Set for key, and whether
// the key exists at all.
func (f *FakeExternalCache) TTL(key string) (time.Duration, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ttl, ok := f.ttl[key]
	return ttl, ok
}
