package dependency

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/dependency"
)

func AnalyzeNamespace(c *gin.Context) {
	namespace := c.Param("namespace")

	opts := dependency.DefaultAnalysisOptions()
	if q := c.Query("timeRange"); q != "" {
		timeRange, err := common.ParseTimeRange(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		opts.TimeRange = timeRange
	}
	if q := c.Query("includeExternal"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_include_external", "Invalid includeExternal query parameter. Use true or false.")
			return
		}
		opts.IncludeExternal = v
	}
	if q := c.Query("includeDns"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_include_dns", "Invalid includeDns query parameter. Use true or false.")
			return
		}
		opts.IncludeDNS = v
	}
	if q := c.Query("includeUnknown"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_include_unknown", "Invalid includeUnknown query parameter. Use true or false.")
			return
		}
		opts.IncludeUnknown = v
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	if !common.EnsureNamespaceExists(c, ctx, namespace) {
		return
	}

	analysis, err := dependency.AnalyzeNamespace(ctx, namespace, opts)
	if err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to analyze namespace dependencies.")
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func AnalyzeWorkload(c *gin.Context) {
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

	opts := dependency.DefaultAnalysisOptions()
	if q := c.Query("timeRange"); q != "" {
		timeRange, err := common.ParseTimeRange(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		opts.TimeRange = timeRange
	}
	if q := c.Query("includeExternal"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_include_external", "Invalid includeExternal query parameter. Use true or false.")
			return
		}
		opts.IncludeExternal = v
	}
	if q := c.Query("includeDns"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_include_dns", "Invalid includeDns query parameter. Use true or false.")
			return
		}
		opts.IncludeDNS = v
	}
	if q := c.Query("includeUnknown"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_include_unknown", "Invalid includeUnknown query parameter. Use true or false.")
			return
		}
		opts.IncludeUnknown = v
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	if _, ok := common.EnsureWorkloadExists(c, ctx, workloadType, workloadName, namespace); !ok {
		return
	}

	analysis, err := dependency.AnalyzeWorkload(ctx, workloadType, workloadName, namespace, opts)
	if err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to analyze workload dependencies.")
		return
	}

	c.JSON(http.StatusOK, analysis)
}
