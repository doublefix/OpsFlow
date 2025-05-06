package queue

import (
	"context"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type TaskProcessorConfig struct {
	Clientset   kubernetes.Interface
	CRDClient   dynamic.NamespaceableResourceInterface
	RpcConn     *grpc.ClientConn
	RedisClient redis.Cmdable
	WorkerCount int
	QueueName   string
}

func StartTaskQueueProcessor(ctx context.Context, config TaskProcessorConfig) {
	if err := config.RedisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	taskChannel := make(chan Task)

	go monitorTaskQueue(ctx, config.RedisClient, config.QueueName, taskChannel)

	var wg sync.WaitGroup
	processor := NewTaskProcessor(config.Clientset, &config.CRDClient, config.RpcConn)

	for i := range config.WorkerCount {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processTasks(ctx, workerID, taskChannel, processor)
		}(i)
	}

	<-ctx.Done()
	close(taskChannel)

	wg.Wait()
	log.Println("All workers have exited.")
}
