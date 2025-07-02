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

	"github.com/joho/godotenv"
	"github.com/modcoco/OpsFlow/pkg/app"
	"github.com/modcoco/OpsFlow/pkg/handler"
	pb "github.com/modcoco/OpsFlow/pkg/proto"
	"github.com/modcoco/OpsFlow/pkg/router"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := app.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	container := app.NewContainer(client)
	engine := router.RegisterRoutes(container)
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: engine,
	}

	// 创建 gRPC 服务器
	grpcSrv := grpc.NewServer()
	podExecHandler, err := handler.NewPodExecServer()
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}
	logHandler := handler.NewPodLogHandler(client.Core().(*kubernetes.Clientset))

	pb.RegisterPodExecServiceServer(grpcSrv, podExecHandler)
	pb.RegisterPodLogServiceServer(grpcSrv, logHandler)

	grpcListener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	// 启动 HTTP 和 gRPC 服务器
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
