package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Runs database migrations using a temporary sql.DB connection
func Migrate(cfg *config.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database config is nil")
	}

	log := logger.WithComponent("database")
	log.Info().Msg("Applying migrations...")

	dsn := buildDSN(cfg)

	// Parse connection string and create connection config
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database connection string for migrations: %w", err)
	}

	// Apply TLS when SSL is enabled
	if cfg.SSLMode != "disable" {
		tlsConfig, err := config.BuildTLSConfig(cfg.TLS)
		if err != nil {
			return fmt.Errorf("build TLS config: %w", err)
		}
		connConfig.TLSConfig = tlsConfig
	}

	// Create postgres connection for migrations
	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create source driver from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Info().Msg("Migrations already up to date")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Info().Msg("Migrations applied")
	return nil
}
