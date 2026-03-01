package core

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisExternalCache implements the ExternalCache interface using Redis.
type RedisExternalCache struct {
	rdb *redis.Client
}

// NewRedisExternalCache returns a configured RedisExternalCache.
// Example: NewRedisExternalCache("redis:6379")
func NewRedisExternalCache(addr string, opts ...func(*redis.Options)) *RedisExternalCache {
	o := &redis.Options{Addr: addr}
	for _, f := range opts {
		f(o)
	}
	return &RedisExternalCache{rdb: redis.NewClient(o)}
}

// Ping verifies Redis connectivity.
func (r *RedisExternalCache) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}

func (r *RedisExternalCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := marshalJSON(val)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, b, ttl).Err()
}

func (r *RedisExternalCache) Get(ctx context.Context, key string, into any) (bool, error) {
	s, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, unmarshalJSON([]byte(s), into)
}

func (r *RedisExternalCache) Del(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}
