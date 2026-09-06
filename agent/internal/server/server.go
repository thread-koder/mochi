package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thread_koder/mochi/agent/internal/config"
	"github.com/thread_koder/mochi/agent/internal/logger"
)

// Server exposes Prometheus scrape and health endpoints.
type Server struct {
	server *http.Server
	cfg    config.Config
}

func New(cfg config.Config) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return &Server{
		cfg: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.MetricsHost, cfg.MetricsPort),
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	log := logger.WithComponent("server")
	log.Info().
		Str("host", s.cfg.MetricsHost).
		Int("port", s.cfg.MetricsPort).
		Msg("Server listening...")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithComponent("server")
	log.Info().Msg("Shutting down server...")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}
	return nil
}
