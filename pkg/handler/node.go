package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// NodeHandler handles node-related HTTP requests
type NodeHandler struct {
	client corev1.NodeInterface
}

// NewNodeHandler creates a new NodeHandler with dependency injection
func NewNodeHandler(client corev1.NodeInterface) *NodeHandler {
	return &NodeHandler{
		client: client,
	}
}

// GetNodesHandle handles GET requests for nodes
// curl http://localhost:8090/api/v1/node\?limit\=1|jq .
func (h *NodeHandler) GetNodesHandle(c *gin.Context) {
	nodeName := c.Query("name")
	labelSelector := c.Query("labelSelector")
	limitStr := c.Query("limit")
	continueToken := c.Query("continue")

	if nodeName != "" {
		node, err := h.client.Get(c.Request.Context(), nodeName, metav1.GetOptions{})
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

	nodeList, err := h.client.List(c.Request.Context(), listOptions)
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, nodeList)
}
