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

	goredis "github.com/redis/go-redis/v9"

	"github.com/BlanketOps/blanketops-environments/core"
)

// RedisConfig holds connection settings for the Redis backend.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// RedisCache implements core.ExternalCache backed by Redis.
type RedisCache struct {
	client *goredis.Client
}

var _ core.ExternalCache = (*RedisCache)(nil)

func NewRedis(cfg RedisConfig) *RedisCache {
	return &RedisCache{client: goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})}
}

func (c *RedisCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, ttl).Err()
}

func (c *RedisCache) Get(ctx context.Context, key string, into any) (bool, error) {
	b, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return false, nil // miss, not an error
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(b, into)
}

func (c *RedisCache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) DelPrefix(ctx context.Context, prefix string) error {
	iter := c.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Ping verifies connectivity; useful at startup or in health checks.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
