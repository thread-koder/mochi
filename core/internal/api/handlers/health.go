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
	"golang.org/x/sync/errgroup"
)

// HealthResponse is the payload for GET /health.
type HealthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks"`
}

// CheckResult is one dependency health entry in the aggregated health response.
type CheckResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Health reports aggregated service health for readiness checks.
func Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dbErr, kubeErr, promErr, redisErr error

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		dbErr = database.HealthCheck(gctx)
		if dbErr != nil {
			c.Error(dbErr)
		}
		return nil
	})

	g.Go(func() error {
		kubeErr = kubernetes.HealthCheck(gctx)
		if kubeErr != nil {
			c.Error(kubeErr)
		}
		return nil
	})

	g.Go(func() error {
		promErr = prometheus.HealthCheck(gctx)
		if promErr != nil {
			c.Error(promErr)
		}
		return nil
	})

	g.Go(func() error {
		redisErr = redis.HealthCheck(gctx)
		if redisErr != nil {
			c.Error(redisErr)
		}
		return nil
	})

	_ = g.Wait()

	response := HealthResponse{
		Status: "healthy",
		Checks: make(map[string]CheckResult, 4),
	}

	for _, item := range []struct {
		key string
		err error
	}{
		{"database", dbErr},
		{"kubernetes", kubeErr},
		{"prometheus", promErr},
		{"redis", redisErr},
	} {
		if item.err != nil {
			response.Status = "unhealthy"
			response.Checks[item.key] = CheckResult{
				Status: "unhealthy",
				Error:  "Dependency is unavailable",
			}
		} else {
			response.Checks[item.key] = CheckResult{Status: "healthy"}
		}
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// DatabaseHealth reports database connectivity health.
func DatabaseHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.HealthCheck(ctx); err != nil {
		c.Error(err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Dependency is unavailable",
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
			"error":  "Dependency is unavailable",
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
			"error":  "Dependency is unavailable",
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
			"error":  "Dependency is unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}
