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

// Generates compute resource recommendations for a workload
func GenerateRecommendations(c *gin.Context) {
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")
	timeRangeStr := c.Query("timeRange")
	modeStr := strings.ToLower(c.Query("mode"))

	// Validate workload type
	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true, // For standalone pods
	}
	if !validTypes[workloadType] {
		err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
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
		analysisOpts.SetTimeRange(timeRange)
	}

	// Parse recommendation config
	recConfig := compute.DefaultRecommendationConfig()
	if modeStr != "" {
		mode := compute.RecommendationMode(modeStr)
		if mode != compute.ModeBurstable && mode != compute.ModeGuaranteed && mode != compute.ModeCostOptimized {
			err := fmt.Errorf("mode must be one of: burstable, guaranteed, cost_optimized")
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid recommendation mode",
				"details": err.Error(),
			})
			return
		}
		recConfig.Mode = mode
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var pods []*database.Pod

	// Handle standalone pods
	if workloadType == "Pod" {
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

		// Validate it's a standalone pod (no owner) or a system pod (Node-owned)
		if pod.OwnerKind != nil && *pod.OwnerKind != "" && *pod.OwnerKind != "Node" {
			err := fmt.Errorf("pod %s belongs to %s/%s, use workload endpoint with type %s instead",
				workloadName, *pod.OwnerKind, *pod.OwnerName, *pod.OwnerKind)
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "pod belongs to a workload",
				"details": err.Error(),
			})
			return
		}

		pods = []*database.Pod{pod}
	} else {
		// Get pods for this workload (Deployment, StatefulSet, DaemonSet)
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

// Gets compute recommendations with optional filters
func GetRecommendations(c *gin.Context) {
	namespace := c.Query("namespace")
	status := c.Query("status")
	mode := c.Query("mode")
	workloadType := c.Query("workloadType")
	workloadName := c.Query("workloadName")

	// Validate workload type
	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true, // For standalone pods
	}
	if !validTypes[workloadType] {
		err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid workload type",
			"details": err.Error(),
		})
		return
	}

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

	// Create context
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

	// Create context
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
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")

	// Validate workload type
	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true, // For standalone pods
	}
	if !validTypes[workloadType] {
		err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
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

	// Create context
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

// Applies a compute recommendation to the target workload
func ApplyRecommendation(c *gin.Context) {
	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var recommendation *database.ComputeRecommendation
	var id int64

	// Check if ID is provided in query param
	idStr := c.Query("id")
	if idStr != "" {
		// Apply by ID (stored recommendation)
		var err error
		id, err = parseInt64(idStr)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid recommendation ID",
				"details": err.Error(),
			})
			return
		}

		// Get recommendation from database
		recommendation, err = database.GetComputeRecommendationByID(ctx, id)
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

		// Check if recommendation is already applied
		if recommendation.Status == "applied" {
			err := fmt.Errorf("recommendation %d was already applied", id)
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "recommendation already applied",
				"details": err.Error(),
			})
			return
		}
	} else {
		// Apply from body (immediate apply)
		var bodyRec compute.Recommendation
		if err := c.ShouldBindJSON(&bodyRec); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid request body",
				"details": err.Error(),
			})
			return
		}

		validTypes := map[string]bool{
			"Deployment":  true,
			"StatefulSet": true,
			"DaemonSet":   true,
			"Pod":         true, // For standalone pods
		}
		// Validate workload type
		if !validTypes[bodyRec.WorkloadType] {
			err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid workload type",
				"details": err.Error(),
			})
			return
		}

		// Validate required fields
		if bodyRec.WorkloadName == "" || bodyRec.Namespace == "" || bodyRec.AnalysisTimeRange == "" || bodyRec.RecommendationMode == "" {
			err := fmt.Errorf("workload_name, namespace, analysis_time_range, and recommendation_mode are required")
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "missing required fields",
				"details": err.Error(),
			})
			return
		}

		if len(bodyRec.Recommendations) == 0 {
			err := fmt.Errorf("recommendations field is required")
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "missing recommendations",
				"details": err.Error(),
			})
			return
		}

		// Convert to database model
		dbRec, err := compute.ComputeRecommendationToDB(bodyRec)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to convert recommendation",
				"details": err.Error(),
			})
			return
		}

		recommendation = dbRec
		if err := database.InsertComputeRecommendation(ctx, recommendation); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to save recommendation",
				"details": err.Error(),
			})
			return
		}
		id = recommendation.ID
	}

	// Apply the recommendation
	if err := compute.ApplyRecommendation(ctx, recommendation); err != nil {
		if idStr == "" {
			if delErr := database.DeleteComputeRecommendation(ctx, id); delErr != nil {
				// Log error but don't fail
				c.Error(delErr)
			}
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to apply recommendation",
			"details": err.Error(),
		})
		return
	}

	// Update status to "applied"
	if err := database.UpdateComputeRecommendationStatus(ctx, id, "applied"); err != nil {
		// Log error but don't fail
		c.Error(err)
	}

	// Mark other pending recommendations as superseded
	if err := database.MarkRecommendationsSuperseded(ctx, recommendation.WorkloadType, recommendation.WorkloadName, recommendation.Namespace, id); err != nil {
		// Log error but don't fail
		c.Error(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "recommendation applied successfully",
		"id":      id,
	})
}
