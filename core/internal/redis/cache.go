package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Get(ctx context.Context, key string) ([]byte, error) {
	val, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	return val, nil
}

func Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := Client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

func SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return Set(ctx, key, data, ttl)
}

func GetJSON(ctx context.Context, key string, dest any) error {
	data, err := Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return nil
}

func Delete(ctx context.Context, key string) error {
	if err := Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}
