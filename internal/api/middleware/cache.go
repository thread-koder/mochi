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

// responseWriter captures response bytes so successful responses can be cached.
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// CacheMiddleware caches successful JSON GET responses in Redis for ttl.
func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		cacheKey := generateCacheKey(c)

		ctx := c.Request.Context()
		cachedBytes, err := redis.Get(ctx, cacheKey)
		if err == nil && cachedBytes != nil {
			setCacheHeaders(c, ttl, true)
			c.Data(http.StatusOK, "application/json", cachedBytes)
			c.Abort()
			return
		}

		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		if c.Writer.Status() == http.StatusOK {
			responseBytes := writer.body.Bytes()
			if err := redis.Set(ctx, cacheKey, responseBytes, ttl); err != nil {
				log := logger.WithComponent("server")
				log.Warn().Err(err).Str("cache_key", cacheKey).Msg("Failed to cache response")
			} else {
				setCacheHeaders(c, ttl, false)
			}
		}
	}
}

// generateCacheKey hashes path and raw query to scope cache entries per request URL.
func generateCacheKey(c *gin.Context) string {
	key := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		key += "?" + c.Request.URL.RawQuery
	}

	hash := sha256.Sum256([]byte(key))
	return "api:cache:" + hex.EncodeToString(hash[:])
}

// setCacheHeaders marks responses as cache HIT or MISS for clients.
func setCacheHeaders(c *gin.Context, ttl time.Duration, isCacheHit bool) {
	maxAge := int(ttl.Seconds())
	c.Header("Cache-Control", "public, max-age="+fmt.Sprintf("%d", maxAge))
	if isCacheHit {
		c.Header("X-Cache", "HIT")
	} else {
		c.Header("X-Cache", "MISS")
	}
}
