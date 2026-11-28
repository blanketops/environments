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
Package cache is the single composition point for the external cache backend (Redis, Memcached, or no-op).
*/
package cache

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/blanketops/environments/cache/adapter"
	corecache "github.com/blanketops/environments/core/cache"
)

// Backend selects which external cache implementation NewExternal
// constructs.
type Backend string

const (
	BackendNone      Backend = "none"
	BackendRedis     Backend = "redis"
	BackendMemcached Backend = "memcached"
)

// Options configures NewExternal's choice of external cache backend and its
// connection settings.
type Options struct {
	Backend   Backend
	Redis     adapter.RedisConfig
	Memcached adapter.MemcachedConfig
}

// NewExternal is the single composition point for the external cache.
// Called once from main; everything downstream sees only corecache.ExternalCache.
func NewExternal(opts Options) (corecache.ExternalCache, error) {
	log := ctrl.Log.WithName("external-cache")
	switch opts.Backend {
	case BackendRedis:
		log.Info("external cache enabled", "backend", "redis", "addr", opts.Redis.Addr)
		// TODO: wire up a real Redis external cache implementation using adapter when available.
		return corecache.NoopExternalCache{}, nil
	case BackendMemcached:
		log.Info("external cache enabled", "backend", "memcached", "servers", opts.Memcached.Servers)
		// TODO: wire up a real Memcached external cache implementation using adapter when available.
		return corecache.NoopExternalCache{}, nil
	case BackendNone, "":
		log.Info("external cache disabled", "backend", "noop")
		return corecache.NoopExternalCache{}, nil
	default:
		return nil, fmt.Errorf("unknown external cache backend %q (valid: redis|memcached|none)", opts.Backend)
	}
}