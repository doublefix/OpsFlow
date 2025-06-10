package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/handler"
)

type Config struct {
	ListenAddr string
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	return &Config{
		ListenAddr: getEnv("LISTEN_ADDR", ":8090"),
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
		api.DELETE("/deployments/:namespace/:name", handler.DeleteDeploymentHandle)
		api.POST("/services", handler.CreateServiceHandle)
		api.DELETE("/services/:namespace/:name", handler.DeleteServiceHandle)
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

	r := CreateGinRouter(client)
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	server.ListenAndServe()

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server exited properly")
}
