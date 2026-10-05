package dependency

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/span"
)

func IngestSpans(c *gin.Context) {
	var body span.IngestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(err)
		common.WriteValidationError(c, "invalid_request_body", "Request body is invalid.")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := span.Ingest(ctx, body.Spans); err != nil {
		c.Error(err)
		if _, ok := errors.AsType[*span.InvalidError](err); ok {
			common.WriteValidationError(c, "invalid_spans", "Span batch is invalid.")
			return
		}
		common.WriteInternalError(c, "Failed to ingest spans.")
		return
	}

	c.Status(http.StatusNoContent)
}

func GetNamespaceSpans(c *gin.Context) {
	namespace := c.Param("namespace")

	timeRange := 7 * 24 * time.Hour
	if q := c.Query("timeRange"); q != "" {
		parsed, err := common.ParseTimeRange(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		timeRange = parsed
	}

	limit := span.DefaultLimit
	if q := c.Query("limit"); q != "" {
		parsed, err := strconv.Atoi(q)
		if err != nil || parsed <= 0 {
			c.Error(fmt.Errorf("invalid limit %q", q))
			common.WriteValidationError(c, "invalid_limit", "limit must be a positive integer.")
			return
		}
		limit = parsed
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if !common.EnsureNamespaceExists(c, ctx, namespace) {
		return
	}

	since := time.Now().UTC().Add(-timeRange)
	spans, err := span.GetNamespaceSpans(ctx, namespace, since, limit)
	if err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to get namespace spans.")
		return
	}

	c.JSON(http.StatusOK, spans)
}

func GetWorkloadSpans(c *gin.Context) {
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

	timeRange := 7 * 24 * time.Hour
	if q := c.Query("timeRange"); q != "" {
		parsed, err := common.ParseTimeRange(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		timeRange = parsed
	}

	limit := span.DefaultLimit
	if q := c.Query("limit"); q != "" {
		parsed, err := strconv.Atoi(q)
		if err != nil || parsed <= 0 {
			c.Error(fmt.Errorf("invalid limit %q", q))
			common.WriteValidationError(c, "invalid_limit", "limit must be a positive integer.")
			return
		}
		limit = parsed
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if _, ok := common.EnsureWorkloadExists(c, ctx, workloadType, workloadName, namespace); !ok {
		return
	}

	since := time.Now().UTC().Add(-timeRange)
	spans, err := span.GetWorkloadSpans(ctx, workloadType, namespace, workloadName, since, limit)
	if err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to get workload spans.")
		return
	}

	c.JSON(http.StatusOK, spans)
}
