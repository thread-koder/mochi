package api

import (
	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/handlers"
	analysisHandlers "github.com/thread_koder/mochi/internal/api/handlers/analysis"
	computeHandlers "github.com/thread_koder/mochi/internal/api/handlers/compute"
	diskHandlers "github.com/thread_koder/mochi/internal/api/handlers/disk"
	networkHandlers "github.com/thread_koder/mochi/internal/api/handlers/network"
	webHandlers "github.com/thread_koder/mochi/internal/api/handlers/web"
	"github.com/thread_koder/mochi/internal/api/middleware"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/redis"
)

// setupRoutes registers all public HTTP routes.
func setupRoutes(router *gin.Engine) {
	cacheTTL := redis.GetDefaultTTL(&config.AppConfig.Redis)

	router.GET("/health", handlers.Health)
	router.GET("/health/database", handlers.DatabaseHealth)
	router.GET("/health/kubernetes", handlers.KubernetesHealth)
	router.GET("/health/prometheus", handlers.PrometheusHealth)
	router.GET("/health/redis", handlers.RedisHealth)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/home", webHandlers.GetHome)
		v1.GET("/namespaces", webHandlers.GetNamespaces)
		v1.GET("/namespaces/:namespace", webHandlers.GetNamespace)
		v1.GET("/workloads/:namespace/:type/:name", webHandlers.GetWorkload)

		compute := v1.Group("/compute")
		{
			analysisGroup := compute.Group("")
			analysisGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				analysisGroup.GET("/analyze/namespaces/:namespace", computeHandlers.AnalyzeNamespace)
				analysisGroup.GET("/analyze/workloads/:workloadType/:workloadName", computeHandlers.AnalyzeWorkload)
			}

			recommendationsGroup := compute.Group("")
			recommendationsGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				recommendationsGroup.GET("/recommendations", computeHandlers.GetRecommendations)
				recommendationsGroup.GET("/recommendations/:id", computeHandlers.GetRecommendationByID)
				recommendationsGroup.GET("/recommendations/workloads/:workloadType/:workloadName/latest", computeHandlers.GetLatestWorkloadRecommendation)
			}

			compute.POST("/recommendations/generate/:workloadType/:workloadName", computeHandlers.GenerateRecommendations)
			compute.POST("/recommendations/apply", computeHandlers.ApplyRecommendation)
		}

		network := v1.Group("/network")
		{
			networkAnalysisGroup := network.Group("")
			networkAnalysisGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				networkAnalysisGroup.GET("/analyze/namespaces/:namespace", networkHandlers.AnalyzeNamespace)
				networkAnalysisGroup.GET("/analyze/workloads/:workloadType/:workloadName", networkHandlers.AnalyzeWorkload)
			}
		}

		diskGroup := v1.Group("/disk")
		{
			diskAnalysisGroup := diskGroup.Group("")
			diskAnalysisGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				diskAnalysisGroup.GET("/analyze/namespaces/:namespace", diskHandlers.AnalyzeNamespace)
				diskAnalysisGroup.GET("/analyze/workloads/:workloadType/:workloadName", diskHandlers.AnalyzeWorkload)
			}
		}

		// This group is used for analyses that are not specific
		// to a single domain (cross-domain).
		analysisGroup := v1.Group("/analysis")
		{
			correlationGroup := analysisGroup.Group("")
			correlationGroup.Use(middleware.CacheMiddleware(cacheTTL))
			{
				correlationGroup.GET("/correlations/workloads/:workloadType/:workloadName", analysisHandlers.AnalyzeWorkloadCorrelations)
			}
		}
	}
}
