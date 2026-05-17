package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/kubernetes"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"github.com/thread_koder/mochi/core/internal/redis"
)

// Health reports aggregated service health for readiness checks.
func Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := gin.H{
		"status": "healthy",
		"checks": gin.H{},
	}

	if err := database.HealthCheck(ctx); err != nil {
		c.Error(err)
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

	if err := kubernetes.HealthCheck(ctx); err != nil {
		c.Error(err)
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

	if err := prometheus.HealthCheck(ctx); err != nil {
		c.Error(err)
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

	if err := redis.HealthCheck(ctx); err != nil {
		c.Error(err)
		health["status"] = "unhealthy"
		health["checks"].(gin.H)["redis"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(gin.H)["redis"] = gin.H{
			"status": "healthy",
		}
	}

	statusCode := http.StatusOK
	if health["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// DatabaseHealth reports database connectivity health.
func DatabaseHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.HealthCheck(ctx); err != nil {
		c.Error(err)
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

// KubernetesHealth reports Kubernetes client health.
func KubernetesHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := kubernetes.HealthCheck(ctx); err != nil {
		c.Error(err)
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

// PrometheusHealth reports Prometheus API health.
func PrometheusHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := prometheus.HealthCheck(ctx); err != nil {
		c.Error(err)
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

// RedisHealth reports Redis connectivity health.
func RedisHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redis.HealthCheck(ctx); err != nil {
		c.Error(err)
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
