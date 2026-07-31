package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/compute"
	"github.com/thread_koder/mochi/core/internal/database"
)

func GenerateRecommendations(c *gin.Context) {
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

	analysisOpts := compute.DefaultAnalysisOptions()
	if q := c.Query("timeRange"); q != "" {
		timeRange, err := common.ParseTimeRange(q)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_time_range", "Invalid timeRange query parameter. Use values like 24h, 7d, or 1h30m.")
			return
		}
		analysisOpts.SetTimeRange(timeRange)
	}

	recConfig := compute.DefaultRecommendationConfig()
	if q := strings.ToLower(c.Query("mode")); q != "" {
		mode := compute.RecommendationMode(q)
		if mode != compute.ModeBurstable && mode != compute.ModeGuaranteed && mode != compute.ModeCostOptimized {
			err := fmt.Errorf("recommendation mode must be one of: burstable, guaranteed, cost_optimized")
			c.Error(err)
			common.WriteValidationError(c, "invalid_recommendation_mode", "Recommendation mode must be one of burstable, guaranteed, or cost_optimized.")
			return
		}
		recConfig.Mode = mode
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	pods, ok := common.ResolveWorkloadPods(c, ctx, workloadType, workloadName, namespace)
	if !ok {
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
		if errors.Is(err, &apperrors.NoMetricsError{}) {
			common.WriteNoMetricsError(c, "no_metrics_available", "No metrics available for the requested workload and time range.")
		} else {
			common.WriteInternalError(c, "Failed to generate recommendations.")
		}
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

func GetRecommendations(c *gin.Context) {
	namespace := c.Query("namespace")
	status := c.Query("status")
	mode := c.Query("mode")
	workloadType := c.Query("workloadType")
	workloadName := c.Query("workloadName")

	if workloadType != "" && !common.ValidateWorkloadType(c, workloadType) {
		return
	}

	limit := 100
	offset := 0
	if q := c.Query("limit"); q != "" {
		if parsed, err := parseInt(q); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if q := c.Query("offset"); q != "" {
		if parsed, err := parseInt(q); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
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
		common.WriteInternalError(c, "Failed to get recommendations.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommendations": recommendations,
		"total":           total,
	})
}

func GetRecommendationByID(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.Error(err)
		common.WriteValidationError(c, "invalid_recommendation_id", "Recommendation ID must be a valid UUID.")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	recommendation, err := database.GetComputeRecommendationByID(ctx, id)
	if err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NotFoundError{}) {
			common.WriteNotFoundError(c, "recommendation_not_found", "Recommendation not found.")
		} else {
			common.WriteInternalError(c, "Failed to get recommendation.")
		}
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

func GetLatestRecommendation(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	recommendation, err := database.GetLatestComputeRecommendation(
		ctx,
		workloadType,
		workloadName,
		namespace,
	)
	if err != nil {
		c.Error(err)
		if errors.Is(err, &apperrors.NotFoundError{}) {
			common.WriteNotFoundError(c, "recommendation_not_found", "Recommendation not found.")
		} else {
			common.WriteInternalError(c, "Failed to get recommendation.")
		}
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

// ApplyRecommendation applies an existing (by ID) or inline (in request body) recommendation to a workload.
func ApplyRecommendation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	var recommendation *database.ComputeRecommendation
	var id uuid.UUID
	var err error

	idQuery := c.Query("id")
	if idQuery != "" {
		id, err = parseUUID(idQuery)
		if err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_recommendation_id", "Recommendation ID must be a valid UUID.")
			return
		}

		recommendation, err = database.GetComputeRecommendationByID(ctx, id)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "recommendation_not_found", "Recommendation not found.")
			} else {
				common.WriteInternalError(c, "Failed to get recommendation.")
			}
			return
		}

		if recommendation.Status == "applied" {
			err = fmt.Errorf("recommendation %s is already applied", id)
			c.Error(err)
			common.WriteValidationError(c, "recommendation_already_applied", "Recommendation already applied.")
			return
		}
	} else {
		var bodyRec compute.Recommendation
		if err = c.ShouldBindJSON(&bodyRec); err != nil {
			c.Error(err)
			common.WriteValidationError(c, "invalid_request_body", "Request body is invalid.")
			return
		}

		if bodyRec.WorkloadType == "" || bodyRec.WorkloadName == "" || bodyRec.Namespace == "" ||
			bodyRec.AnalysisTimeRange == "" || bodyRec.RecommendationMode == "" {
			err = fmt.Errorf("workload_type, workload_name, namespace, analysis_time_range, and recommendation_mode are required")
			c.Error(err)
			common.WriteValidationError(c, "missing_required_fields", "workload_type, workload_name, namespace, analysis_time_range, and recommendation_mode are required.")
			return
		}

		if !common.ValidateWorkloadType(c, bodyRec.WorkloadType) {
			return
		}

		recommendation, err = compute.NewComputeRecommendationRecord(bodyRec)
		if err != nil {
			c.Error(err)
			common.WriteInternalError(c, "Failed to create recommendation record.")
			return
		}
		id = recommendation.ID
	}

	if err := compute.ApplyRecommendation(ctx, recommendation); err != nil {
		c.Error(err)
		writeApplyRecommendationError(c, err)
		return
	}

	if idQuery == "" {
		recommendation.Status = "applied"
		if err := database.InsertComputeRecommendation(ctx, recommendation); err != nil {
			c.Error(err)
			common.WriteInternalError(c, "Failed to save recommendation record.")
			return
		}
	} else if err := database.UpdateComputeRecommendationStatus(ctx, id, "applied"); err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to update recommendation status to applied.")
		return
	}

	if err := database.SupersedeComputeRecommendations(
		ctx,
		recommendation.WorkloadType,
		recommendation.WorkloadName,
		recommendation.Namespace,
		id,
	); err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to supersede recommendations.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "recommendation applied successfully",
		"id":      id,
	})
}

func writeApplyRecommendationError(c *gin.Context, err error) {
	if errors.Is(err, compute.ErrNoContainerRecommendations) {
		common.WriteValidationError(c, "missing_container_recommendations", "No container recommendations to apply.")
		return
	}

	if applyErr, ok := errors.AsType[*compute.ApplyNotSupportedError](err); ok {
		if applyErr.WorkloadType == "Pod" {
			common.WriteValidationError(c, "pod_apply_not_supported", "Pods are immutable and cannot be auto-updated. Update resources manually, or apply via the owning Deployment, StatefulSet, or DaemonSet.")
			return
		}
		common.WriteValidationError(c, "workload_type_not_supported", "Workload type is not supported.")
		return
	}

	if notFound, ok := errors.AsType[*apperrors.NotFoundError](err); ok {
		switch notFound.Resource {
		case "deployment":
			common.WriteNotFoundError(c, "deployment_not_found", "Deployment not found.")
		case "statefulset":
			common.WriteNotFoundError(c, "statefulset_not_found", "StatefulSet not found.")
		case "daemonset":
			common.WriteNotFoundError(c, "daemonset_not_found", "DaemonSet not found.")
		default:
			common.WriteNotFoundError(c, "workload_not_found", "Workload not found.")
		}
		return
	}

	common.WriteInternalError(c, "Failed to apply recommendation.")
}
