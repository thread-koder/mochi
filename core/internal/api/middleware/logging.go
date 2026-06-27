package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/logger"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		fields := map[string]any{
			"component":  "server",
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency":    fmt.Sprintf("%dms", latency.Milliseconds()),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		if raw != "" {
			fields["query"] = raw
		}

		if len(c.Errors) > 0 {
			var errorStrs []string
			for _, err := range c.Errors {
				errorStrs = append(errorStrs, err.Error())
			}
			fields["errors"] = errorStrs
		}

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
