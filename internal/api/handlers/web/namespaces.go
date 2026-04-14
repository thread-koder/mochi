package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/database"
)

// Namespace represents one namespace row in the web list response.
type Namespace struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
}

// GetNamespaces returns all namespaces for the web UI.
func GetNamespaces(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespaces, err := database.GetNamespaces(ctx)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get namespaces",
			"details": err.Error(),
		})
		return
	}

	items := make([]Namespace, 0, len(namespaces))
	for _, ns := range namespaces {
		items = append(items, Namespace{
			Name:  ns.Name,
			Phase: ns.Phase,
		})
	}

	c.JSON(http.StatusOK, items)
}
