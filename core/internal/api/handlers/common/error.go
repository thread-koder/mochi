package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError is a sanitized error payload returned to API clients.
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteAPIError writes a sanitized error response.
func WriteAPIError(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIError{
		Error:   code,
		Message: message,
	})
}

// WriteNotFoundError writes a sanitized 404 response.
func WriteNotFoundError(c *gin.Context, code, message string) {
	WriteAPIError(c, http.StatusNotFound, code, message)
}

// WriteNoMetricsError writes a sanitized 422 response when telemetry is missing.
func WriteNoMetricsError(c *gin.Context, code, message string) {
	WriteAPIError(c, http.StatusUnprocessableEntity, code, message)
}

// WriteValidationError writes a client-correctable validation error.
func WriteValidationError(c *gin.Context, code, message string) {
	WriteAPIError(c, http.StatusBadRequest, code, message)
}

// WriteInternalError writes a generic 500 response without internal details.
func WriteInternalError(c *gin.Context, message string) {
	WriteAPIError(c, http.StatusInternalServerError, "internal_error", message)
}
