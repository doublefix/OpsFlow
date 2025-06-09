package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/handler"
)

type Config struct {
	GrpcAddr       string
	ListenAddr     string
	QueueName      string
	WorkerCount    int
	RedisAddrs     []string
	RedisPwd       string
	RedisIsCluster bool
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	workerCount, err := strconv.Atoi(getEnv("WORKER_COUNT", "1"))
	if err != nil {
		return nil, fmt.Errorf("invalid WORKER_COUNT: %v", err)
	}

	redisAddrs := strings.Split(getEnv("REDIS_ADDRS", "127.0.0.1:6379"), ",")
	if len(redisAddrs) == 0 {
		return nil, fmt.Errorf("no Redis addresses provided")
	}

	return &Config{
		GrpcAddr:       getEnv("GRPC_ADDR", "localhost:50051"),
		ListenAddr:     getEnv("LISTEN_ADDR", ":8090"),
		QueueName:      getEnv("QUEUE_NAME", "task_queue"),
		WorkerCount:    workerCount,
		RedisAddrs:     redisAddrs,
		RedisPwd:       os.Getenv("REDIS_PASSWORD"),
		RedisIsCluster: getEnv("REDIS_CLUSTER", "false") == "true",
	}, nil
}

func CreateGinRouter(client core.Client) *gin.Engine {
	r := gin.Default()
	r.Use(core.AppContextMiddleware(client))

	api := r.Group("/api/v1")
	{
		api.GET("/pod", handler.GetPodInfo)
		api.POST("/raycluster", handler.GetCreateRayClusterInfo)
		api.POST("/rayjob", handler.CreateRayJobHandle)
		api.POST("/deployments", handler.CreateDeploymentHandle)
		api.GET("/rayjob/:namespace/:name", handler.RayJobInfoHandle)
		api.DELETE("/rayjob/:namespace/:name", handler.RemoveRayJobHandle)
	}

	return r
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := core.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Start HTTP server
	r := CreateGinRouter(client)
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	// Handle shutdown gracefully
	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server exited properly")
}
