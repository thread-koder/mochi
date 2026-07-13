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
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
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
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer database.Close()

	if err := database.Migrate(&cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	if err := kubernetes.Init(&cfg.Kubernetes); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Kubernetes client")
	}

	if err := prometheus.Init(&cfg.Prometheus); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Prometheus client")
	}

	if err := redis.Init(&cfg.Redis); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis client")
	}
	defer redis.Close()

	workerPool, err := workers.NewWorkerPool(&cfg.Workers)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create worker pool")
	}

	server, err := api.NewServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create API server")
	}
	log.Info().Msg("Components initialized")

	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start API server")
		}
	}()

	workerPool.Start()
	defer workerPool.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Bound shutdown so SIGTERM/SIGINT can't leave the process stuck
	// if Close or in-flight work never completes.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
		os.Exit(1)
	}

	log.Info().Msg("Server shutdown")
}
