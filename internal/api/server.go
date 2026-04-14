package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/middleware"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

// Server owns the Gin router and HTTP server lifecycle.
type Server struct {
	router *gin.Engine
	server *http.Server
	cfg    *config.APIConfig
}

// NewServer builds an API server from the given API config.
func NewServer(cfg *config.APIConfig) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("api config is nil")
	}

	gin.SetMode(cfg.Mode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())

	setupRoutes(router)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	}

	return &Server{
		router: router,
		server: server,
		cfg:    cfg,
	}, nil
}

// Start begins serving HTTP requests until shutdown is requested.
func (s *Server) Start() error {
	log := logger.WithComponent("server")
	log.Info().
		Str("host", s.cfg.Host).
		Int("port", s.cfg.Port).
		Str("mode", s.cfg.Mode).
		Msg("Server listening...")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithComponent("server")
	log.Info().Msg("Shutting down server...")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
