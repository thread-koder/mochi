package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
)

var (
	// Client is the global Redis client.
	Client *redis.Client
)

// Init configures and verifies the Redis client connection.
func Init(cfg *config.RedisConfig) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	log := logger.WithComponent("redis")
	log.Info().Msg("Initializing client...")

	opts := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username:        cfg.Username,
		Password:        cfg.Password,
		DB:              cfg.Database,
		MaxRetries:      cfg.MaxRetries,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	}

	if cfg.UseTLS {
		tlsConfig, err := config.BuildTLSConfig(cfg.TLS)
		if err != nil {
			return fmt.Errorf("build TLS config: %w", err)
		}
		opts.TLSConfig = tlsConfig
	}

	Client = redis.NewClient(opts)

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

// Close closes the Redis client if it was initialized.
func Close() {
	if Client != nil {
		Client.Close()
	}
}

// HealthCheck verifies Redis reachability with a Ping call.
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}
