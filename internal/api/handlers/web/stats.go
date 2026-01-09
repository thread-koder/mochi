package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Represents statistics for the home page
type Stats struct {
	Namespaces  int `json:"namespaces"`
	Workloads   int `json:"workloads"`
	Pods        int `json:"pods"`
	HealthScore int `json:"healthScore"`
}

// Returns statistics for the home page
func GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := GetStatsData(ctx)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get stats",
			"details": err.Error(),
		})
		return
	}

	// Calculate health score
	stats.HealthScore = 100 // Default

	c.JSON(http.StatusOK, stats)
}
