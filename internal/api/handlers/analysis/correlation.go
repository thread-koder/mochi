package analysis

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/analysis"
	"github.com/thread_koder/mochi/internal/api/handlers/common"
	"github.com/thread_koder/mochi/internal/database"
)

// AnalyzeWorkloadCorrelations runs cross-metric correlation analysis for a workload.
func AnalyzeWorkloadCorrelations(c *gin.Context) {
	workloadType := c.Param("workloadType")
	workloadName := c.Param("workloadName")
	namespace := c.Query("namespace")
	timeRangeStr := c.Query("timeRange")
	maxLagStr := c.Query("maxLag")

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

	opts := analysis.DefaultCorrelationOptions()

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
		opts.SetTimeRange(timeRange)
	}

	if maxLagStr != "" {
		maxLag, err := common.ParseTimeRange(maxLagStr)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "failed to parse max lag",
				"details": err.Error(),
			})
			return
		}
		opts.MaxLag = maxLag
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

	result, err := analysis.AnalyzeWorkloadCorrelations(ctx, workloadType, workloadName, namespace, pods, opts)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to analyze workload",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
