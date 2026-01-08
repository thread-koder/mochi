package api

import (
	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/handlers"
	computeHandlers "github.com/thread_koder/mochi/internal/api/handlers/compute"
)

// Configures all API routes
func setupRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/health", handlers.HealthHandler)
	router.GET("/health/database", handlers.DatabaseHealthHandler)
	router.GET("/health/kubernetes", handlers.KubernetesHealthHandler)
	router.GET("/health/prometheus", handlers.PrometheusHealthHandler)
	router.GET("/health/redis", handlers.RedisHealthHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Cluster info
		v1.GET("/cluster/info", handlers.ClusterInfoHandler)

		// Compute domain
		compute := v1.Group("/compute")
		{
			// Analysis endpoints
			compute.GET("/analyze/namespaces/:namespace", computeHandlers.AnalyzeNamespaceHandler)
			compute.GET("/analyze/workloads/:workloadType/:workloadName", computeHandlers.AnalyzeWorkloadHandler)
		}
	}
}
