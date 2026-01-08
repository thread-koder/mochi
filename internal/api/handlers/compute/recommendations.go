package compute

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/compute"
	"github.com/thread_koder/mochi/internal/database"
)

// Generates compute resource recommendations for a workload
func GenerateRecommendations(c *gin.Context) {
	workloadType := strings.ToLower(c.Param("workloadType"))
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")
	timeRangeStr := c.Query("timeRange")
	modeStr := strings.ToLower(c.Query("mode"))

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
	analysisOpts := compute.DefaultAnalysisOptions()
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
		analysisOpts.TimeRange = timeRange
	}

	// Parse recommendation config
	recConfig := compute.DefaultRecommendationConfig()
	if modeStr != "" {
		mode := compute.RecommendationMode(modeStr)
		if mode != compute.ModeBurstable && mode != compute.ModeGuaranteed {
			err := fmt.Errorf("mode must be one of: burstable, guaranteed")
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid recommendation mode",
				"details": err.Error(),
			})
			return
		}
		recConfig.Mode = mode
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

	// Generate recommendations
	recommendation, err := compute.GenerateWorkloadRecommendations(
		ctx,
		workloadType,
		workloadName,
		namespace,
		pods,
		recConfig,
		analysisOpts,
	)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate recommendations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

// Lists compute recommendations with optional filters
func ListRecommendations(c *gin.Context) {
	namespace := c.Query("namespace")
	status := c.Query("status")
	mode := c.Query("mode")
	workloadType := c.Query("workloadType")
	workloadName := c.Query("workloadName")

	// Parse pagination parameters
	limit := 100 // Default limit
	offset := 0  // Default offset
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := parseInt(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := parseInt(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare filter pointers
	var namespacePtr, statusPtr, modePtr, workloadTypePtr, workloadNamePtr *string
	if namespace != "" {
		namespacePtr = &namespace
	}
	if status != "" {
		statusPtr = &status
	}
	if mode != "" {
		modePtr = &mode
	}
	if workloadType != "" {
		workloadTypePtr = &workloadType
	}
	if workloadName != "" {
		workloadNamePtr = &workloadName
	}

	// Get recommendations
	recommendations, err := database.GetComputeRecommendations(
		ctx,
		namespacePtr,
		statusPtr,
		modePtr,
		workloadTypePtr,
		workloadNamePtr,
		limit,
		offset,
	)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get recommendations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, recommendations)
}

// Gets a compute recommendation by ID
func GetRecommendationByID(c *gin.Context) {
	idStr := c.Param("id")

	// Parse ID
	id, err := parseInt64(idStr)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid recommendation ID",
			"details": err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get recommendation
	recommendation, err := database.GetComputeRecommendationByID(ctx, id)
	if err != nil {
		c.Error(err)
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "recommendation not found",
				"details": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to get recommendation",
				"details": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

// Gets the latest compute recommendation for a workload
func GetLatestWorkloadRecommendation(c *gin.Context) {
	workloadType := strings.ToLower(c.Param("workloadType"))
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")

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

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get latest recommendation
	recommendation, err := database.GetLatestComputeRecommendation(ctx, workloadType, workloadName, namespace)
	if err != nil {
		c.Error(err)
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "recommendation not found",
				"details": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to get recommendation",
				"details": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

// Helper function to parse integer from string
func parseInt(s string) (int, error) {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer: %w", err)
	}
	return result, nil
}

// Helper function to parse int64 from string
func parseInt64(s string) (int64, error) {
	result, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int64: %w", err)
	}
	return result, nil
}
