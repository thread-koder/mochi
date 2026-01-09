package api

import (
	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/handlers"
	computeHandlers "github.com/thread_koder/mochi/internal/api/handlers/compute"
	webHandlers "github.com/thread_koder/mochi/internal/api/handlers/web"
)

// Configures all API routes
func setupRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/health", handlers.Health)
	router.GET("/health/database", handlers.DatabaseHealth)
	router.GET("/health/kubernetes", handlers.KubernetesHealth)
	router.GET("/health/prometheus", handlers.PrometheusHealth)
	router.GET("/health/redis", handlers.RedisHealth)

	// Web UI routes
	router.GET("/", webHandlers.Home)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Cluster info
		v1.GET("/cluster/info", handlers.ClusterInfo)

		// Web UI API endpoints
		v1.GET("/stats", webHandlers.GetStats)
		v1.GET("/activity", webHandlers.GetActivity)
		v1.GET("/namespaces", webHandlers.GetNamespaces)

		// Compute domain
		compute := v1.Group("/compute")
		{
			// Analysis endpoints
			compute.GET("/analyze/namespaces/:namespace", computeHandlers.AnalyzeNamespace)
			compute.GET("/analyze/workloads/:workloadType/:workloadName", computeHandlers.AnalyzeWorkload)

			// Recommendation endpoints
			compute.POST("/recommendations/generate/:workloadType/:workloadName", computeHandlers.GenerateRecommendations)
			compute.GET("/recommendations", computeHandlers.GetRecommendations)
			compute.GET("/recommendations/:id", computeHandlers.GetRecommendationByID)
			compute.GET("/recommendations/workloads/:workloadType/:workloadName/latest", computeHandlers.GetLatestWorkloadRecommendation)
			compute.POST("/recommendations/apply", computeHandlers.ApplyRecommendation)
		}
	}
}
