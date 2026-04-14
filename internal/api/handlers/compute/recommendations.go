package compute

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/api/handlers/common"
	"github.com/thread_koder/mochi/internal/compute"
	"github.com/thread_koder/mochi/internal/database"
)

// GenerateRecommendations creates compute recommendations for a workload.
func GenerateRecommendations(c *gin.Context) {
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")
	timeRangeStr := c.Query("timeRange")
	modeStr := strings.ToLower(c.Query("mode"))

	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true,
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

	if namespace == "" {
		err := fmt.Errorf("namespace query parameter is empty or missing")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "namespace query parameter is required",
			"details": err.Error(),
		})
		return
	}

	analysisOpts := compute.DefaultAnalysisOptions()
	if timeRangeStr != "" {
		timeRange, err := common.ParseTimeRange(timeRangeStr)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var pods []*database.Pod

	if workloadType == "Pod" {
		pod, err := database.GetPodByName(ctx, workloadName, namespace)
		if err != nil {
			c.Error(err)
			if common.IsNotFoundError(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "pod not found",
					"details": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to get pod",
					"details": err.Error(),
				})
			}
			return
		}

		// Reject controller-owned pods so the caller uses the correct workload type.
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

// GetRecommendations lists stored compute recommendations.
func GetRecommendations(c *gin.Context) {
	namespace := c.Query("namespace")
	status := c.Query("status")
	mode := c.Query("mode")
	workloadType := c.Query("workloadType")
	workloadName := c.Query("workloadName")

	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true,
	}
	if workloadType != "" && !validTypes[workloadType] {
		err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid workload type",
			"details": err.Error(),
		})
		return
	}

	limit := 100
	offset := 0
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var namespacePtr, statusPtr, modePtr, workloadTypePtr, workloadNamePtr *string
	if namespace != "" {
		namespacePtr = new(namespace)
	}
	if status != "" {
		statusPtr = new(status)
	}
	if mode != "" {
		modePtr = new(mode)
	}
	if workloadType != "" {
		workloadTypePtr = new(workloadType)
	}
	if workloadName != "" {
		workloadNamePtr = new(workloadName)
	}

	recommendations, total, err := database.GetComputeRecommendations(
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

	c.JSON(http.StatusOK, gin.H{
		"recommendations": recommendations,
		"total":           total,
	})
}

// GetRecommendationByID returns one recommendation by ID.
func GetRecommendationByID(c *gin.Context) {
	idStr := c.Param("id")

	id, err := parseInt64(idStr)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid recommendation ID",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	recommendation, err := database.GetComputeRecommendationByID(ctx, id)
	if err != nil {
		c.Error(err)
		if common.IsNotFoundError(err) {
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

// GetLatestWorkloadRecommendation returns the latest recommendation for a workload.
func GetLatestWorkloadRecommendation(c *gin.Context) {
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")

	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true,
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

	if namespace == "" {
		err := fmt.Errorf("namespace query parameter is empty or missing")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "namespace query parameter is required",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	recommendation, err := database.GetLatestComputeRecommendation(ctx, workloadType, workloadName, namespace)
	if err != nil {
		c.Error(err)
		if common.IsNotFoundError(err) {
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

// ApplyRecommendation applies an existing or inline recommendation to a workload.
func ApplyRecommendation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var recommendation *database.ComputeRecommendation
	var id int64

	idStr := c.Query("id")
	if idStr != "" {
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

		recommendation, err = database.GetComputeRecommendationByID(ctx, id)
		if err != nil {
			c.Error(err)
			if common.IsNotFoundError(err) {
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
			"Pod":         true,
		}
		if !validTypes[bodyRec.WorkloadType] {
			err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid workload type",
				"details": err.Error(),
			})
			return
		}

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

	if err := compute.ApplyRecommendation(ctx, recommendation); err != nil {
		if idStr == "" {
			if delErr := database.DeleteComputeRecommendation(ctx, id); delErr != nil {
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

	if err := database.UpdateComputeRecommendationStatus(ctx, id, "applied"); err != nil {
		c.Error(err)
	}

	if err := database.MarkRecommendationsSuperseded(ctx, recommendation.WorkloadType, recommendation.WorkloadName, recommendation.Namespace, id); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "recommendation applied successfully",
		"id":      id,
	})
}
