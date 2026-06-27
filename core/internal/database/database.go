package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// Pool is the global database connection pool.
var Pool *pgxpool.Pool

func Init(cfg *config.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database config is nil")
	}

	log := logger.WithComponent("database")
	log.Info().Msg("Initializing connection pool...")

	dsn := buildDSN(cfg)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database connection string: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MaxConnIdleTime = time.Duration(cfg.ConnMaxIdleTime) * time.Second
	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Second
	poolConfig.MinIdleConns = int32(cfg.MinIdleConns)

	if cfg.TLS.Enabled {
		tlsConfig, err := config.BuildTLSConfig(cfg.TLS)
		if err != nil {
			return fmt.Errorf("build TLS config: %w", err)
		}
		poolConfig.ConnConfig.TLSConfig = tlsConfig
	}

	Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	var versionNum int
	if err := Pool.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&versionNum); err != nil {
		return fmt.Errorf("failed to read PostgreSQL version: %w", err)
	}
	if versionNum < 180000 {
		return fmt.Errorf("PostgreSQL 18 or newer is required (server_version_num=%d)", versionNum)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("Connection established")

	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

func HealthCheck(ctx context.Context) error {
	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

func buildDSN(cfg *config.DatabaseConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)
}
