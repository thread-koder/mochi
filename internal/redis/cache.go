package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thread_koder/mochi/internal/config"
)

// Gets a value from cache by key
func Get(ctx context.Context, key string) ([]byte, error) {
	if Client == nil {
		return nil, fmt.Errorf("Redis client not initialized")
	}

	val, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Key not found, not an error
		}
		return nil, fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	return val, nil
}

// Sets a value in cache with the specified TTL
func Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if Client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	if err := Client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

// Sets a JSON value in cache with the specified TTL
func SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return Set(ctx, key, data, ttl)
}

// Gets a JSON value from cache by key
func GetJSON(ctx context.Context, key string, dest any) error {
	data, err := Get(ctx, key)
	if err != nil {
		return err
	}

	if data == nil {
		return redis.Nil // Key not found
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return nil
}

// Deletes a key from cache
func Delete(ctx context.Context, key string) error {
	if Client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	if err := Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}

// Gets the default cache TTL from configuration
func GetDefaultTTL(cfg *config.RedisConfig) time.Duration {
	return time.Duration(cfg.CacheTTL) * time.Second
}
