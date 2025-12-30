package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handles errors and returns consistent error responses
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle errors if any
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Determine status code
			statusCode := http.StatusInternalServerError
			switch err.Type {
			case gin.ErrorTypeBind:
				statusCode = http.StatusBadRequest
			case gin.ErrorTypePublic:
				statusCode = http.StatusBadRequest
			}

			// Return error response
			c.JSON(statusCode, gin.H{
				"error":   err.Error(),
				"message": err.Error(),
			})
		}
	}
}
