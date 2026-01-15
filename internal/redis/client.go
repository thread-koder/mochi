package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

var (
	// Global Redis client
	Client *redis.Client
)

// Initializes the Redis client
func Init(cfg *config.RedisConfig) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	log := logger.WithComponent("redis")
	log.Info().Msg("Initializing client...")

	// Create Redis client options
	opts := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.Database,
		MaxRetries:      cfg.MaxRetries,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	}

	// Create Redis client
	Client = redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Int("database", cfg.Database).
		Msg("Connection established")

	return nil
}

// Closes the Redis client connection
func Close() {
	if Client != nil {
		Client.Close()
	}
}

// Performs a health check on the Redis connection
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}
