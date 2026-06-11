package cache

import (
	"fmt"

	redis "google.golang.org/api/redis/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/ntlaletsi70/blanketops-environments/cache/adapter"
	"github.com/ntlaletsi70/blanketops-environments/core"
)

type Backend string

const (
	BackendNone      Backend = "none"
	BackendRedis     Backend = "redis"
	BackendMemcached Backend = "memcached"
)

type Options struct {
	Backend   Backend
	Redis     adapter.RedisConfig
	Memcached adapter.MemcachedConfig
}

// NewExternal is the single composition point for the external cache.
// Called once from main; everything downstream sees only core.ExternalCache.
func NewExternal(opts Options) (core.ExternalCache, error) {
	log := ctrl.Log.WithName("external-cache")
	switch opts.Backend {
	case BackendRedis:
		log.Info("external cache enabled", "backend", "redis", "addr", opts.Redis.Addr)
		return redis.New(opts.Redis), nil
	case BackendMemcached:
		log.Info("external cache enabled", "backend", "memcached", "servers", opts.Memcached.Servers)
		return memcached.New(opts.Memcached), nil
	case BackendNone, "":
		log.Info("external cache disabled", "backend", "noop")
		return core.NoopExternalCache{}, nil
	default:
		return nil, fmt.Errorf("unknown external cache backend %q (valid: redis|memcached|none)", opts.Backend)
	}
}
