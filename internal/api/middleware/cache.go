package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/logger"
	"github.com/thread_koder/mochi/internal/redis"
)

// Wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Creates a caching middleware with the specified TTL
func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Generate cache key from request
		cacheKey := generateCacheKey(c)

		// Try to get from cache first
		ctx := c.Request.Context()
		cachedBytes, err := redis.Get(ctx, cacheKey)
		if err == nil && cachedBytes != nil {
			// If cache hit, set HTTP cache headers and return cached response
			setCacheHeaders(c, ttl, true)
			c.Data(http.StatusOK, "application/json", cachedBytes)
			c.Abort()
			return
		}

		// If cache miss, wrap response writer to capture body
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// Execute handler
		c.Next()

		// Only cache successful responses (200 OK)
		if c.Writer.Status() == http.StatusOK {
			// Store response bytes to preserve exact format
			responseBytes := writer.body.Bytes()
			if err := redis.Set(ctx, cacheKey, responseBytes, ttl); err != nil {
				log := logger.WithComponent("server")
				log.Warn().Err(err).Str("cache_key", cacheKey).Msg("Failed to cache response")
			} else {
				// If successful cache, set HTTP cache headers
				setCacheHeaders(c, ttl, false)
			}
		}
	}
}

// Generates a cache key from the request path and query parameters
func generateCacheKey(c *gin.Context) string {
	// Include path and all query parameters
	key := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		key += "?" + c.Request.URL.RawQuery
	}

	hash := sha256.Sum256([]byte(key))
	return "api:cache:" + hex.EncodeToString(hash[:])
}

// Sets HTTP cache headers for the response
func setCacheHeaders(c *gin.Context, ttl time.Duration, isCacheHit bool) {
	maxAge := int(ttl.Seconds())
	c.Header("Cache-Control", "public, max-age="+fmt.Sprintf("%d", maxAge))
	if isCacheHit {
		c.Header("X-Cache", "HIT")
	} else {
		c.Header("X-Cache", "MISS")
	}
}
