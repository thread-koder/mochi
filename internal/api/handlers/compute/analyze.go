package compute

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/compute"
	"github.com/thread_koder/mochi/internal/database"
)

// Analyzes a namespace
func AnalyzeNamespace(c *gin.Context) {
	namespace := c.Param("namespace")
	timeRangeStr := c.Query("timeRange")

	// Parse analysis options
	opts := compute.DefaultAnalysisOptions()

	if timeRangeStr != "" {
		timeRange, err := parseTimeRange(timeRangeStr)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "failed to parse time range",
				"details": err.Error(),
			})
			return
		}
		opts.TimeRange = timeRange
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Perform analysis
	analysis, err := compute.AnalyzeNamespace(ctx, namespace, opts)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to analyze namespace",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// Analyzes a workload (deployment, statefulset, daemonset, or standalone pod)
func AnalyzeWorkload(c *gin.Context) {
	workloadType := strings.ToLower(c.Param("workloadType"))
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")
	timeRangeStr := c.Query("timeRange")

	// Validate workload type
	validTypes := map[string]bool{
		"deployment":  true,
		"statefulset": true,
		"daemonset":   true,
		"pod":         true, // For standalone pods
	}
	if !validTypes[workloadType] {
		err := fmt.Errorf("workload type must be one of: deployment, statefulset, daemonset, pod")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid workload type",
			"details": err.Error(),
		})
		return
	}

	// Validate namespace
	if namespace == "" {
		err := fmt.Errorf("namespace query parameter is empty or missing")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "namespace query parameter is required",
			"details": err.Error(),
		})
		return
	}

	// Parse analysis options
	opts := compute.DefaultAnalysisOptions()
	if timeRangeStr != "" {
		timeRange, err := parseTimeRange(timeRangeStr)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "failed to parse time range",
				"details": err.Error(),
			})
			return
		}
		opts.TimeRange = timeRange
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var pods []*database.Pod

	// Handle standalone pods differently
	if workloadType == "pod" {
		// Get the standalone pod by name
		pod, err := database.GetPodByName(ctx, workloadName, namespace)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "pod not found",
				"details": err.Error(),
			})
			return
		}

		// Validate it's actually a standalone pod (no owner)
		if pod.OwnerKind != nil && *pod.OwnerKind != "" {
			err := fmt.Errorf("pod %s belongs to %s/%s, use workload endpoint with type %s instead",
				workloadName, *pod.OwnerKind, *pod.OwnerName, strings.ToLower(*pod.OwnerKind))
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "pod belongs to a workload",
				"details": err.Error(),
			})
			return
		}

		pods = []*database.Pod{pod}
	} else {
		// Get pods for this workload (deployment/statefulset/daemonset)
		podsList, err := database.GetPodsByWorkload(ctx, workloadType, workloadName, namespace)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to get pods for workload",
				"details": err.Error(),
			})
			return
		}
		pods = podsList
	}

	if len(pods) == 0 {
		err := fmt.Errorf("no pods found for workload %s/%s", workloadName, namespace)
		c.Error(err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "no pods found",
			"details": err.Error(),
		})
		return
	}

	// Perform analysis
	analysis, err := compute.AnalyzeWorkload(ctx, workloadType, workloadName, namespace, pods, opts)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to analyze workload",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
}
