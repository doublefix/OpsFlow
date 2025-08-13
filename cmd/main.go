package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/modcoco/OpsFlow/pkg/agent"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/handler"
	"github.com/modcoco/OpsFlow/pkg/queue"
	"github.com/modcoco/OpsFlow/pkg/tasks"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	redisAddrs := strings.Split(getEnv("REDIS_ADDRS", "10.187.6.5:6379"), ",")
	if len(redisAddrs) == 0 {
		return nil, fmt.Errorf("no Redis addresses provided")
	}

	return &Config{
		GrpcAddr:       getEnv("GRPC_ADDR", "idp.baihai.co:8980"),
		ListenAddr:     getEnv("LISTEN_ADDR", ":8090"),
		QueueName:      getEnv("QUEUE_NAME", "task_queue"),
		WorkerCount:    workerCount,
		RedisAddrs:     redisAddrs,
		RedisPwd:       "Rma4399",
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
		api.GET("/rayjob/:namespace/:name", handler.RayJobInfoHandle)
		api.DELETE("/rayjob/:namespace/:name", handler.RemoveRayJobHandle)
	}

	return r
}

func createRedisClient(cfg *Config) (redis.Cmdable, error) {
	if cfg.RedisIsCluster {
		client := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisAddrs,
			Password: cfg.RedisPwd,
		})
		if err := client.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
		}
		return client, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddrs[0],
		Password: cfg.RedisPwd,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return client, nil
}

func runAgent(ctx context.Context, conn *grpc.ClientConn, client core.Client) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			namespace, err := client.Core().CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
			if err != nil {
				log.Printf("Get Namespace error: %v", err)
				continue
			}

			if err := agent.RunAgent(conn, string(namespace.UID)); err != nil {
				log.Printf("runAgent exited with error: %v", err)
			}
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize gRPC connection
	fmt.Println(cfg.GrpcAddr)
	conn, err := grpc.NewClient(
		cfg.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	// conn, err := grpc.NewClient(
	// 	"idp.baihai.co:443", // 必须带端口
	// 	grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
	// 		NextProtos: []string{"h2"},  // 强制HTTP/2
	// 		ServerName: "idp.baihai.co", // SNI
	// 	})),
	// )
	if err != nil {
		log.Fatalf("did not connect to rpc: %v", err)
	}
	defer conn.Close()

	// client1 := pb.NewAgentServiceClient(conn)
	// _, err = client1.AgentStream(ctx)
	// if err != nil {
	// 	fmt.Println(err)
	// 	panic("err")
	// }

	client, err := core.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	redisClient, err := createRedisClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	// Start task queue processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		queueConfig := queue.TaskProcessorConfig{
			Clientset:   client.Core(),
			CRDClient:   client.DynamicNRI(),
			RpcConn:     conn,
			RedisClient: redisClient,
			WorkerCount: cfg.WorkerCount,
			QueueName:   cfg.QueueName,
		}
		queue.StartTaskQueueProcessor(ctx, queueConfig)
	}()

	// Start task scheduler
	wg.Add(1)
	go func() {
		defer wg.Done()
		tasksConfig := tasks.InitializeTasks(client, redisClient, conn)
		tasks.StartTaskScheduler(redisClient, tasksConfig)
	}()

	// Start agent
	wg.Add(1)
	go func() {
		defer wg.Done()
		runAgent(ctx, conn, client)
	}()

	// Start HTTP server
	r := CreateGinRouter(client)
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Handle shutdown gracefully
	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	wg.Wait()
	log.Println("Server exited properly")
}
