package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Represents a single activity item
type ActivityItem struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Returns recent activities
func GetActivity(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := parseInt(limitStr); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	activities, err := GetActivityItems(ctx, limit)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get activity",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, activities)
}
