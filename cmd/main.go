package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/handler"
	pb "github.com/modcoco/OpsFlow/pkg/proto"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
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
	engine := gin.Default()
	engine.Use(core.AppContextMiddleware(client))

	api := engine.Group("/api/v1")
	{
		// api.GET("/pod", handler.GetPodInfo)

		api.GET("/node", handler.GetNodesHandle)
		api.POST("/pod", handler.CreatePodHandle)
		api.GET("/pod", handler.GetPodsHandle)
		api.POST("/deployments", handler.CreateDeploymentHandle)
		api.DELETE("/deployments/:namespace/:name", handler.DeleteDeploymentHandle)
		api.DELETE("/pod/:namespace/:name", handler.DeletePodHandle)
		api.POST("/services", handler.CreateServiceHandle)
		api.DELETE("/services/:namespace/:name", handler.DeleteServiceHandle)

	}

	return engine
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

	// 创建 HTTP 服务器
	r := CreateGinRouter(client)
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	// 创建 gRPC 服务器
	grpcServer, err := handler.NewPodExecServer()
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}

	grpcListener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterPodExecServiceServer(grpcSrv, grpcServer)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("HTTP server listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		log.Printf("gRPC server listening on %s", ":50051")
		if err := grpcSrv.Serve(grpcListener); err != nil {
			return fmt.Errorf("gRPC server error: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-sigChan:
			log.Printf("Received signal: %v", sig)
			cancel()
			return nil
		}
	})

	g.Go(func() error {
		<-ctx.Done()
		log.Println("Shutting down servers...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		stopped := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(stopped)
		}()

		select {
		case <-shutdownCtx.Done():
			grpcSrv.Stop()
		case <-stopped:
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("Server error: %v", err)
	}

	log.Println("All servers stopped gracefully")
}
