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

	// redisClient := redis.NewClusterClient(&redis.ClusterOptions{
	// 	Addrs: []string{
	// 		"10.187.6.3:31000",
	// 		"10.187.6.4:31001",
	// 		"10.187.6.5:31002",
	// 		"10.187.6.3:31100",
	// 		"10.187.6.4:31101",
	// 		"10.187.6.5:31102",
	// 	},
	// 	Password: "pass12345",
	// })
	// config := queue.TaskProcessorConfig{
	// 	Clientset:   client.Core(),
	// 	CRDClient:   crdClient,
	// 	RedisClient: redisClient,
	// 	WorkerCount: 3,
	// 	QueueName:   "task_queue",
	// }
	// go queue.StartTaskQueueProcessor(config)

	r := CreateGinRouter(client)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
