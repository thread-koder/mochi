package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/network"
)

func AnalyzeNamespace(c *gin.Context) {
	namespace := c.Param("namespace")
	timeRangeStr := c.Query("timeRange")

	opts := network.DefaultAnalysisOptions()
	opts.IncludeTimeSeries = true

	if timeRangeStr != "" {
		timeRange, err := common.ParseTimeRange(timeRangeStr)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		opts.SetTimeRange(timeRange)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	if _, err := database.GetNamespaceByName(ctx, namespace); err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NotFoundError{}) {
			common.WriteNotFoundError(c, "namespace_not_found", "Namespace not found.")
		} else {
			common.WriteInternalError(c, "Failed to get namespace.")
		}
		return
	}

	analysis, err := network.AnalyzeNamespace(ctx, namespace, opts)
	if err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NoMetricsError{}) {
			common.WriteNoMetricsError(c, "no_metrics_available", "No metrics available for the requested namespace and time range.")
		} else {
			common.WriteInternalError(c, "Failed to analyze namespace.")
		}
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func AnalyzeWorkload(c *gin.Context) {
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")
	timeRangeStr := c.Query("timeRange")

	if !common.ValidateWorkloadType(c, workloadType) {
		return
	}

	if namespace == "" {
		err := fmt.Errorf("namespace query parameter is empty or missing")
		c.Error(err)
		common.WriteValidationError(c, "missing_namespace", "Namespace query parameter is required.")
		return
	}

	opts := network.DefaultAnalysisOptions()
	opts.IncludeTimeSeries = true
	if timeRangeStr != "" {
		timeRange, err := common.ParseTimeRange(timeRangeStr)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		opts.SetTimeRange(timeRange)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	if _, err := database.GetNamespaceByName(ctx, namespace); err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NotFoundError{}) {
			common.WriteNotFoundError(c, "namespace_not_found", "Namespace not found.")
		} else {
			common.WriteInternalError(c, "Failed to get namespace.")
		}
		return
	}

	pods, ok := common.ResolveWorkloadPods(c, ctx, workloadType, workloadName, namespace)
	if !ok {
		return
	}

	analysis, err := network.AnalyzeWorkload(ctx, workloadType, workloadName, namespace, pods, opts, true)
	if err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NoMetricsError{}) {
			common.WriteNoMetricsError(c, "no_metrics_available", "No metrics available for the requested workload and time range.")
		} else {
			common.WriteInternalError(c, "Failed to analyze workload.")
		}
		return
	}

	c.JSON(http.StatusOK, analysis)
}
