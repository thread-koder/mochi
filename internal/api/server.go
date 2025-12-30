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

// Holds the HTTP server and router
type Server struct {
	router *gin.Engine
	server *http.Server
	cfg    *config.APIConfig
}

// Creates a new API server instance
func NewServer(cfg *config.APIConfig) *Server {
	// Set Gin mode
	gin.SetMode(cfg.Mode)

	// Create router
	router := gin.New()

	// Apply middleware
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.ErrorHandlingMiddleware())

	// Setup routes
	setupRoutes(router)

	// Create HTTP server
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
	}
}

// Starts the HTTP server
func (s *Server) Start() error {
	log := logger.WithComponent("api")
	log.Info().
		Str("host", s.cfg.Host).
		Int("port", s.cfg.Port).
		Str("mode", s.cfg.Mode).
		Msg("Starting server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shuts down the server gracefully
func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithComponent("api")
	log.Info().Msg("Shutting down server")

	return s.server.Shutdown(ctx)
}
