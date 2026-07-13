package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/middleware"
	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// Server owns the Gin router and HTTP server lifecycle.
type Server struct {
	router *gin.Engine
	server *http.Server
	cfg    *config.Config
}

func NewServer(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	gin.SetMode(cfg.API.Mode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())

	setupRoutes(router, cfg)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:      router,
		ReadTimeout:  cfg.API.ReadTimeoutDuration(),
		WriteTimeout: cfg.API.WriteTimeoutDuration(),
	}

	return &Server{
		router: router,
		server: server,
		cfg:    cfg,
	}, nil
}

func (s *Server) Start() error {
	log := logger.WithComponent("server")
	log.Info().
		Str("host", s.cfg.API.Host).
		Int("port", s.cfg.API.Port).
		Str("mode", s.cfg.API.Mode).
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
