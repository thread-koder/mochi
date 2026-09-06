package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thread_koder/mochi/agent/internal/collection"
	"github.com/thread_koder/mochi/agent/internal/config"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
	"github.com/thread_koder/mochi/agent/internal/server"
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

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log := logger.WithComponent("main")

	log.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Str("git_commit", GitCommit).
		Msg("Mochi Agent")

	registry := metrics.NewRegistry()
	collectionRuntime := collection.Start(cfg, registry)
	defer collectionRuntime.Close()

	srv := server.New(cfg)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

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
			return fmt.Errorf("server: %w (shutdown: %v)", startErr, err)
		}
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info().Msg("Server shutdown")
	if startErr != nil {
		return fmt.Errorf("server: %w", startErr)
	}
	return nil
}
