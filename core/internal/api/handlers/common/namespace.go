package common

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
)

func EnsureNamespaceExists(c *gin.Context, ctx context.Context, namespace string) bool {
	if _, err := database.GetNamespaceByName(ctx, namespace); err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NotFoundError{}) {
			WriteNotFoundError(c, "namespace_not_found", "Namespace not found.")
		} else {
			WriteInternalError(c, "Failed to get namespace.")
		}
		return false
	}
	return true
}
