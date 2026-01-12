package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/logger"
)

// Logs HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build base fields
		fields := map[string]any{
			"component":  "server",
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency":    fmt.Sprintf("%dms", latency.Milliseconds()),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		// Add query string if present
		if raw != "" {
			fields["query"] = raw
		}

		// Add errors if present
		if len(c.Errors) > 0 {
			var errorStrs []string
			for _, err := range c.Errors {
				errorStrs = append(errorStrs, err.Error())
			}
			fields["errors"] = errorStrs
		}

		// Log based on status code
		eventLogger := logger.WithFields(fields)
		statusCode := c.Writer.Status()

		if statusCode >= 500 {
			eventLogger.Error().Msg("HTTP request failed")
		} else if statusCode >= 400 {
			eventLogger.Warn().Msg("HTTP request failed")
		} else {
			eventLogger.Debug().Msg("HTTP request successful")
		}
	}
}
