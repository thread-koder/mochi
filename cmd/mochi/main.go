package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thread_koder/mochi/internal/api"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/kubernetes"
	"github.com/thread_koder/mochi/internal/logger"
	"github.com/thread_koder/mochi/internal/prometheus"
	"github.com/thread_koder/mochi/internal/redis"
	"github.com/thread_koder/mochi/internal/workers"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with config
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	log := logger.WithComponent("main")
	log.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Str("git_commit", GitCommit).
		Msg("Mochi - Kubernetes Resource Optimization Platform")

	log.Info().Msg("Initializing components...")
	// Initialize database
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer database.Close()

	// Run database migrations
	if err := database.Migrate(&cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Initialize Kubernetes client
	if err := kubernetes.Init(&cfg.Kubernetes); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Kubernetes client")
	}

	// Initialize Prometheus client
	if err := prometheus.Init(&cfg.Prometheus); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Prometheus client")
	}

	// Initialize Redis client
	if err := redis.Init(&cfg.Redis); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis client")
	}
	defer redis.Close()

	// Create and start worker pool
	workerPool := workers.NewWorkerPool(&cfg.Workers)
	workerPool.Start()
	defer workerPool.Stop()

	// Create and start API server
	server := api.NewServer(&cfg.API)
	log.Info().Msg("Components initialized")

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start API server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
		os.Exit(1)
	}

	log.Info().Msg("Server shutdown")
	os.Exit(0)
}
