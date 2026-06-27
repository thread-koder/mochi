package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func WriteAPIError(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIError{
		Error:   code,
		Message: message,
	})
}

func WriteNotFoundError(c *gin.Context, code, message string) {
	WriteAPIError(c, http.StatusNotFound, code, message)
}

func WriteNoMetricsError(c *gin.Context, code, message string) {
	WriteAPIError(c, http.StatusUnprocessableEntity, code, message)
}

func WriteValidationError(c *gin.Context, code, message string) {
	WriteAPIError(c, http.StatusBadRequest, code, message)
}

func WriteInternalError(c *gin.Context, message string) {
	WriteAPIError(c, http.StatusInternalServerError, "internal_error", message)
}
