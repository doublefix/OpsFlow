package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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

	return &Config{
		GrpcAddr:       getEnv("GRPC_ADDR", "localhost:50051"),
		ListenAddr:     getEnv("LISTEN_ADDR", ":8090"),
		QueueName:      getEnv("QUEUE_NAME", "task_queue"),
		WorkerCount:    workerCount,
		RedisAddrs:     strings.Split(getEnv("REDIS_ADDRS", "127.0.0.1:6379"), ","),
		RedisPwd:       os.Getenv("REDIS_PASSWORD"),
		RedisIsCluster: getEnv("REDIS_CLUSTER", "false") == "true",
	}, nil
}

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

	conn, err := grpc.NewClient(
		cfg.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect to rpc: %v", err)
	}
	defer conn.Close()

	var redisClient redis.Cmdable
	if cfg.RedisIsCluster {
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisAddrs,
			Password: cfg.RedisPwd,
		})
	} else {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddrs[0],
			Password: cfg.RedisPwd,
		})
	}

	queueConfig := queue.TaskProcessorConfig{
		Clientset:   client.Core(),
		CRDClient:   client.DynamicNRI(),
		RpcConn:     conn,
		RedisClient: redisClient,
		WorkerCount: cfg.WorkerCount,
		QueueName:   cfg.QueueName,
	}
	go queue.StartTaskQueueProcessor(ctx, queueConfig)

	tasksConfig := tasks.InitializeTasks(client, redisClient, conn)
	tasks.StartTaskScheduler(redisClient, tasksConfig)

	go func() {
		for {
			namespace, err := client.Core().CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
			if err != nil {
				log.Printf("Get Namespace error: %v", err)
			}
			if err := agent.RunAgent(conn, string(namespace.UID)); err != nil {
				log.Printf("runAgent exited with error: %v, retrying in 5s...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}()

	r := CreateGinRouter(client)
	if err := r.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
