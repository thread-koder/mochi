package compute

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/compute"
)

func GetNamespaceBottlenecks(c *gin.Context) {
	namespace := c.Param("namespace")

	opts := compute.DefaultAnalysisOptions()

	if q := c.Query("timeRange"); q != "" {
		timeRange, err := common.ParseTimeRange(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		opts.SetTimeRange(timeRange)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	if !common.EnsureNamespaceExists(c, ctx, namespace) {
		return
	}

	result, err := compute.GetNamespaceBottlenecks(ctx, namespace, opts)
	if err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NoMetricsError{}) {
			common.WriteNoMetricsError(c, "no_metrics_available", "No metrics available for the requested namespace and time range.")
		} else {
			common.WriteInternalError(c, "Failed to get namespace bottlenecks.")
		}
		return
	}

	c.JSON(http.StatusOK, result)
}
