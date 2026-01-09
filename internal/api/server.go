package api

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
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

	// Load HTML templates
	tmpl, err := loadTemplates()
	if err != nil {
		// Log error but continue, templates might not be available yet
		log := logger.WithComponent("server")
		log.Warn().Err(err).Msg("Failed to load templates, continuing without them")
	} else {
		router.SetHTMLTemplate(tmpl)
	}

	// Serve static files
	router.Static("/static", "./web/static")

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

// Shuts down the server gracefully
func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithComponent("server")
	log.Info().Msg("Shutting down server...")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}

// Loads HTML templates from filesystem
func loadTemplates() (*template.Template, error) {
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Parse base template if it exists separately
	basePath := filepath.Join("web/templates", "base.html")
	if _, err := tmpl.ParseFiles(basePath); err != nil {
		// Base template might already be included, ignore error
	}

	return tmpl, nil
}
