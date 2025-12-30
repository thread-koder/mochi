package api

import (
	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/handlers"
)

// Configures all API routes
func setupRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/health", handlers.HealthHandler)
	router.GET("/health/database", handlers.DatabaseHealthHandler)
	router.GET("/health/kubernetes", handlers.KubernetesHealthHandler)
	router.GET("/health/prometheus", handlers.PrometheusHealthHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Cluster info
		v1.GET("/cluster/info", handlers.ClusterInfoHandler)
	}
}
