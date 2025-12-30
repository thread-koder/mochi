package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/kubernetes"
	"github.com/thread_koder/mochi/internal/prometheus"
)

// Returns the overall health status
func HealthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := gin.H{
		"status": "healthy",
		"checks": gin.H{},
	}

	// Check database
	if err := database.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(gin.H)["database"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(gin.H)["database"] = gin.H{
			"status": "healthy",
		}
	}

	// Check Kubernetes
	if err := kubernetes.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(gin.H)["kubernetes"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(gin.H)["kubernetes"] = gin.H{
			"status": "healthy",
		}
	}

	// Check Prometheus
	if err := prometheus.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(gin.H)["prometheus"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(gin.H)["prometheus"] = gin.H{
			"status": "healthy",
		}
	}

	statusCode := http.StatusOK
	if health["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// Returns the database health status
func DatabaseHealthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// Returns the Kubernetes health status
func KubernetesHealthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := kubernetes.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// Returns the Prometheus health status
func PrometheusHealthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := prometheus.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}
