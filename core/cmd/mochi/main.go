package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thread_koder/mochi/core/internal/api"
	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/kubernetes"
	"github.com/thread_koder/mochi/core/internal/logger"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"github.com/thread_koder/mochi/core/internal/redis"
	"github.com/thread_koder/mochi/core/internal/workers"
)

var (
	// Version is the build version.
	// Release builds set it with -ldflags "-X main.Version=...".
	// The default is "dev".
	Version = "dev"

	// BuildTime is when the binary was built.
	// Release builds set it with -ldflags "-X main.BuildTime=...".
	// The default is "unknown".
	BuildTime = "unknown"

	// GitCommit is the source revision at build time.
	// Release builds set it with -ldflags "-X main.GitCommit=...".
	// The default is "unknown".
	GitCommit = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger.Init(cfg.Log.Level, cfg.Log.Format)

	log := logger.WithComponent("main")
	log.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Str("git_commit", GitCommit).
		Msg("Mochi - Kubernetes Resource Optimization Platform")

	log.Info().Msg("Initializing components...")
	if err := database.Init(&cfg.Database); err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer database.Close()

	if err := database.Migrate(&cfg.Database); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	if err := kubernetes.Init(&cfg.Kubernetes); err != nil {
		return fmt.Errorf("initialize Kubernetes client: %w", err)
	}

	if err := prometheus.Init(&cfg.Prometheus); err != nil {
		return fmt.Errorf("initialize Prometheus client: %w", err)
	}

	if err := redis.Init(&cfg.Redis); err != nil {
		return fmt.Errorf("initialize Redis client: %w", err)
	}
	defer redis.Close()

	workerPool, err := workers.NewWorkerPool(&cfg.Workers)
	if err != nil {
		return fmt.Errorf("create worker pool: %w", err)
	}

	srv, err := api.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("create API server: %w", err)
	}
	log.Info().Msg("Components initialized")

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	workerPool.Start()
	defer workerPool.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var startErr error
	select {
	case <-quit:
	case startErr = <-errCh:
	}

	// Bound shutdown so SIGTERM/SIGINT can't leave the process stuck
	// if Close or in-flight work never completes.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
		if startErr != nil {
			return fmt.Errorf("API server: %w (shutdown: %v)", startErr, err)
		}
		return fmt.Errorf("API server shutdown: %w", err)
	}

	log.Info().Msg("Server shutdown")
	if startErr != nil {
		return fmt.Errorf("API server: %w", startErr)
	}
	return nil
}
