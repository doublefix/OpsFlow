package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPodInfo(c *gin.Context) {
	appCtx := core.GetAppContext(c)
	pods, err := appCtx.Client().Core().CoreV1().Pods("default").List(
		appCtx.Ctx(),
		metav1.ListOptions{},
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "Kubernetes client is working",
		"pods":    pods.Items,
	})
}
