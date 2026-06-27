package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/database"
)

type Namespace struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
}

func GetNamespaces(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespaces, err := database.GetNamespaces(ctx)
	if err != nil {
		c.Error(err)
		common.WriteInternalError(c, "Failed to get namespaces.")
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
