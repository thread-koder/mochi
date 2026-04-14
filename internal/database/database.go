// Package database holds PostgreSQL persistence for the Kubernetes resource snapshot synced from
// the cluster, compute recommendations. Runtime access uses the shared Pool created by Init.
// Migrate applies embedded SQL using a separate database/sql connection
// required by golang-migrate (see Migrate).
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

// Pool is the global database connection pool.
var Pool *pgxpool.Pool

// Init builds and configures the database connection pool and verifies connection.
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

	if cfg.SSLMode != "disable" {
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

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("Connection established")

	return nil
}

// Close closes the database connection pool if it was initialized.
func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

// HealthCheck verifies the database reachability with a Ping call.
func HealthCheck(ctx context.Context) error {
	if Pool == nil {
		return fmt.Errorf("database connection not initialized")
	}

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
