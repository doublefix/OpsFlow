package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func CreateGinRouter(client internal.Client) *gin.Engine {
	r := gin.Default()
	r.Use(internal.AppContextMiddleware(client))

	r.GET("/test", GetPodInfo)

	return r
}

func main() {
	client, err := internal.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	r := CreateGinRouter(client)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func GetPodInfo(c *gin.Context) {
	appCtx := internal.GetAppContext(c)
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
