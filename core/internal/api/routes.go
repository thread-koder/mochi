package api

import (
	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers"
	analysisHandlers "github.com/thread_koder/mochi/core/internal/api/handlers/analysis"
	computeHandlers "github.com/thread_koder/mochi/core/internal/api/handlers/compute"
	dependencyHandlers "github.com/thread_koder/mochi/core/internal/api/handlers/dependency"
	diskHandlers "github.com/thread_koder/mochi/core/internal/api/handlers/disk"
	networkHandlers "github.com/thread_koder/mochi/core/internal/api/handlers/network"
	webHandlers "github.com/thread_koder/mochi/core/internal/api/handlers/web"
	"github.com/thread_koder/mochi/core/internal/api/middleware"
	"github.com/thread_koder/mochi/core/internal/config"
)

func setupRoutes(router *gin.Engine, cfg *config.Config) {
	cacheTTL := cfg.Redis.CacheTTLDuration()

	router.GET("/health", handlers.Health)
	router.GET("/health/database", handlers.DatabaseHealth)
	router.GET("/health/kubernetes", handlers.KubernetesHealth)
	router.GET("/health/prometheus", handlers.PrometheusHealth)
	router.GET("/health/redis", handlers.RedisHealth)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/home", webHandlers.GetHome(cfg.Kubernetes.ClusterName))
		v1.GET("/namespaces", webHandlers.GetNamespaces)
		v1.GET("/namespaces/:namespace", webHandlers.GetNamespace)
		v1.GET("/workloads/:namespace/:type/:name", webHandlers.GetWorkload)

		compute := v1.Group("/compute")
		{
			computeAnalysis := compute.Group("")
			computeAnalysis.Use(middleware.CacheMiddleware(cacheTTL))
			{
				computeAnalysis.GET("/analyze/namespaces/:namespace", computeHandlers.AnalyzeNamespace)
				computeAnalysis.GET("/analyze/workloads/:workloadType/:workloadName", computeHandlers.AnalyzeWorkload)
			}

			computeRecommendations := compute.Group("")
			computeRecommendations.Use(middleware.CacheMiddleware(cacheTTL))
			{
				computeRecommendations.GET("/recommendations", computeHandlers.GetRecommendations)
				computeRecommendations.GET("/recommendations/:id", computeHandlers.GetRecommendationByID)
			}

			compute.POST("/recommendations/generate/:workloadType/:workloadName", computeHandlers.GenerateRecommendations)
			compute.POST("/recommendations/apply", computeHandlers.ApplyRecommendation)
		}

		network := v1.Group("/network")
		{
			networkAnalysis := network.Group("")
			networkAnalysis.Use(middleware.CacheMiddleware(cacheTTL))
			{
				networkAnalysis.GET("/analyze/namespaces/:namespace", networkHandlers.AnalyzeNamespace)
				networkAnalysis.GET("/analyze/workloads/:workloadType/:workloadName", networkHandlers.AnalyzeWorkload)
			}
		}

		disk := v1.Group("/disk")
		{
			diskAnalysis := disk.Group("")
			diskAnalysis.Use(middleware.CacheMiddleware(cacheTTL))
			{
				diskAnalysis.GET("/analyze/namespaces/:namespace", diskHandlers.AnalyzeNamespace)
				diskAnalysis.GET("/analyze/workloads/:workloadType/:workloadName", diskHandlers.AnalyzeWorkload)
			}
		}

		dependency := v1.Group("/dependency")
		{
			dependencyAnalysis := dependency.Group("")
			dependencyAnalysis.Use(middleware.CacheMiddleware(cacheTTL))
			{
				dependencyAnalysis.GET("/analyze/namespaces/:namespace", dependencyHandlers.AnalyzeNamespace)
				dependencyAnalysis.GET("/analyze/workloads/:workloadType/:workloadName", dependencyHandlers.AnalyzeWorkload)
			}
		}

		// This group is used for analyses that are not specific
		// to a single domain (cross-domain).
		analysis := v1.Group("/analysis")
		analysis.Use(middleware.CacheMiddleware(cacheTTL))
		{
			analysis.GET("/correlations/workloads/:workloadType/:workloadName", analysisHandlers.AnalyzeWorkloadCorrelations)
		}
	}
}
