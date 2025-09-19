package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	mlv1 "gitlab.openpaper.co/chessbod/cloudmind/api/ml/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ModelPiplineHandler struct {
	client ctrlclient.Client
}

func NewModelPiplineHandler(client ctrlclient.Client) *ModelPiplineHandler {
	return &ModelPiplineHandler{client: client}
}

func (h *ModelPiplineHandler) CreateModelPipline(c *gin.Context) {
	var mdp mlv1.ModelDataPipeline
	if err := c.ShouldBindJSON(&mdp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mdp.APIVersion = "ml.cloudmind.io/v1"
	mdp.Kind = "ModelDataPipeline"
	if mdp.Namespace == "" {
		mdp.Namespace = "default"
	}

	if err := h.client.Create(c.Request.Context(), &mdp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, mdp)
}
