package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

var (
	// Global database connection pool
	Pool *pgxpool.Pool
)

// Initializes the database connection pool using pgx
func Init(cfg *config.DatabaseConfig) error {
	log := logger.WithComponent("database")
	log.Info().Msg("Initializing connection pool...")

	dsn := buildDSN(cfg)

	// Parse connection string and create pool config
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database connection string: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MaxConnIdleTime = time.Duration(cfg.ConnMaxIdleTime) * time.Second
	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Second
	poolConfig.MinConns = int32(cfg.MaxIdleConns)

	// Create connection pool
	Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}

	// Test connection
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

// Closes the database connection pool
func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

// Performs a health check on the database connection
func HealthCheck(ctx context.Context) error {
	if Pool == nil {
		return fmt.Errorf("database connection not initialized")
	}

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// Builds the PostgreSQL connection string (DSN) for pgx
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
