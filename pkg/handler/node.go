package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// curl http://localhost:8090/api/v1/node\?limit\=1|jq .
func GetNodesHandle(c *gin.Context) {
	nodeName := c.Query("name")
	labelSelector := c.Query("labelSelector")
	limitStr := c.Query("limit")
	continueToken := c.Query("continue")

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Nodes()

	if nodeName != "" {
		node, err := client.Get(appCtx.Ctx(), nodeName, metav1.GetOptions{})
		if err != nil {
			handleK8sError(c, err)
			return
		}
		c.JSON(http.StatusOK, node)
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
			return
		}
		listOptions.Limit = limit
	}

	if continueToken != "" {
		listOptions.Continue = continueToken
	}

	nodeList, err := client.List(appCtx.Ctx(), listOptions)
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, nodeList)
}
