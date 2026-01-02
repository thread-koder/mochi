package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/kubernetes"
)

// Returns information about the connected Kubernetes cluster
func ClusterInfoHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := kubernetes.GetClusterInfo(ctx)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get cluster info",
			"details": err.Error(),
		})
		return
	}

	// Get cluster resources
	resources, err := kubernetes.DiscoverCluster(ctx)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to discover cluster resources",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster": gin.H{
			"server_version": info.ServerVersion,
			"cluster_name":   info.ClusterName,
			"context_name":   info.ContextName,
			"api_server_url": info.APIServerURL,
		},
		"resources": gin.H{
			"namespaces":         resources.Namespaces,
			"nodes":              resources.Nodes,
			"pods":               resources.Pods,
			"services":           resources.Services,
			"deployments":        resources.Deployments,
			"statefulsets":       resources.StatefulSets,
			"daemonsets":         resources.DaemonSets,
			"configmaps":         resources.ConfigMaps,
			"secrets":            resources.Secrets,
			"persistent_volumes": resources.PersistentVolumes,
		},
	})
}
