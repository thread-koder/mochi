package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs embedded SQL migrations. It uses database/sql via pgx stdlib because
// golang-migrate's postgres driver expects that interface, separate from the pgxpool Pool.
func Migrate(cfg *config.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database config is nil")
	}

	log := logger.WithComponent("database")
	log.Info().Msg("Applying migrations...")

	dsn := buildDSN(cfg)

	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database connection string for migrations: %w", err)
	}

	if cfg.SSLMode != "disable" {
		tlsConfig, err := config.BuildTLSConfig(cfg.TLS)
		if err != nil {
			return fmt.Errorf("build TLS config: %w", err)
		}
		connConfig.TLSConfig = tlsConfig
	}

	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

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
