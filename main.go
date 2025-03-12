package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/handler"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func CreateGinRouter(client core.Client) *gin.Engine {
	r := gin.Default()
	r.Use(core.AppContextMiddleware(client))

	r.GET("/api/v1/pod", handler.GetPodInfo)
	r.POST("/api/v1/raycluster", handler.GetCreateRayClusterInfo)
	r.POST("/api/v1/rayjob", handler.CreateRayJobHandle)
	r.GET("/api/v1/rayjob/:namespace/:name", handler.RayJobInfoHandle)
	r.DELETE("/api/v1/rayjob/:namespace/:name", handler.RemoveRayJobHandle)

	return r
}

func main() {
	client, err := core.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	r := CreateGinRouter(client)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
