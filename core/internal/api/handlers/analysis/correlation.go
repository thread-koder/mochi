package analysis

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/analysis"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func AnalyzeWorkloadCorrelations(c *gin.Context) {
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")

	if !common.ValidateWorkloadType(c, workloadType) {
		return
	}

	if namespace == "" {
		err := fmt.Errorf("namespace query parameter is empty or missing")
		c.Error(err)
		common.WriteValidationError(c, "missing_namespace", "Namespace query parameter is required.")
		return
	}

	opts := analysis.DefaultCorrelationOptions()

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

	pods, ok := common.ResolveWorkloadPods(c, ctx, workloadType, workloadName, namespace, time.Now().Add(-opts.TimeRange))
	if !ok {
		return
	}

	result, err := analysis.AnalyzeWorkloadCorrelations(ctx, workloadType, workloadName, namespace, pods.All, opts)
	if err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NoMetricsError{}) {
			common.WriteNoMetricsError(c, "no_metrics_available", "No metrics available for the requested workload and time range.")
		} else {
			common.WriteInternalError(c, "Failed to analyze workload correlations.")
		}
		return
	}

	c.JSON(http.StatusOK, result)
}
