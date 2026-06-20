package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thread_koder/mochi/core/internal/config"
)

// Get returns the raw bytes for the given key.
// It returns (nil, nil) when the key does not exist.
func Get(ctx context.Context, key string) ([]byte, error) {
	if Client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	val, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	return val, nil
}

// Set stores value at key with ttl.
func Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if Client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	if err := Client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

// SetJSON marshals value as JSON and stores it at key with ttl.
func SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return Set(ctx, key, data, ttl)
}

// GetJSON unmarshals cached JSON for key into dest.
// It returns redis.Nil when the key does not exist.
func GetJSON(ctx context.Context, key string, dest any) error {
	data, err := Get(ctx, key)
	if err != nil {
		return err
	}

	if data == nil {
		return redis.Nil
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return nil
}

// Delete removes key from cache.
func Delete(ctx context.Context, key string) error {
	if Client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	if err := Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}

// CacheTTL returns the configured cache TTL in seconds.
func CacheTTL(cfg *config.RedisConfig) time.Duration {
	return time.Duration(cfg.CacheTTL) * time.Second
}
