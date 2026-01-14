package api

import (
	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/handlers"
	computeHandlers "github.com/thread_koder/mochi/internal/api/handlers/compute"
	webHandlers "github.com/thread_koder/mochi/internal/api/handlers/web"
	"github.com/thread_koder/mochi/internal/api/middleware"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/redis"
)

// Configures all API routes
func setupRoutes(router *gin.Engine) {
	cacheTTL := redis.GetDefaultTTL(&config.AppConfig.Redis)
	// Health check endpoints
	router.GET("/health", handlers.Health)
	router.GET("/health/database", handlers.DatabaseHealth)
	router.GET("/health/kubernetes", handlers.KubernetesHealth)
	router.GET("/health/prometheus", handlers.PrometheusHealth)
	router.GET("/health/redis", handlers.RedisHealth)

	// Web UI routes
	router.GET("/", webHandlers.Home)
	router.GET("/namespaces/:namespace", webHandlers.Namespace)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Web UI API endpoints (no caching)
		v1.GET("/stats", webHandlers.GetStats)
		v1.GET("/activity", webHandlers.GetActivity)
		v1.GET("/namespaces", webHandlers.GetNamespaces)

		// Compute domain
		compute := v1.Group("/compute")
		{
			// Analysis endpoints (cached)
			analysisGroup := compute.Group("")
			analysisGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				analysisGroup.GET("/analyze/namespaces/:namespace", computeHandlers.AnalyzeNamespace)
				analysisGroup.GET("/analyze/workloads/:workloadType/:workloadName", computeHandlers.AnalyzeWorkload)
			}

			// Recommendation endpoints (cached)
			recommendationsGroup := compute.Group("")
			recommendationsGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				recommendationsGroup.GET("/recommendations", computeHandlers.GetRecommendations)
				recommendationsGroup.GET("/recommendations/:id", computeHandlers.GetRecommendationByID)
				recommendationsGroup.GET("/recommendations/workloads/:workloadType/:workloadName/latest", computeHandlers.GetLatestWorkloadRecommendation)
				recommendationsGroup.GET("/recommendations/generate/:workloadType/:workloadName", computeHandlers.GenerateRecommendations)
			}

			// POST endpoints (no caching)
			compute.POST("/recommendations/apply", computeHandlers.ApplyRecommendation)
		}
	}
}
