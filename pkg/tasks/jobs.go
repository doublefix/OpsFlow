package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/modcoco/OpsFlow/pkg/crd"
	"github.com/modcoco/OpsFlow/pkg/queue"
	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type TaskFunc func(ctx context.Context) error

func task1Func(ctx context.Context) error {
	log.Println("Executing task 1 specific logic...")
	select {
	case <-time.After(70 * time.Second):
		log.Println("Task 1 completed")
	case <-ctx.Done():
		log.Println("Task 1 canceled")
		return ctx.Err()
	}
	return nil
}

func task2Func(ctx context.Context) error {
	log.Println("Executing task 2 specific logic...")
	select {
	case <-time.After(3 * time.Second):
		log.Println("Task 2 completed")
	case <-ctx.Done():
		log.Println("Task 2 canceled")
		return ctx.Err()
	}
	return nil
}

func task3Func(ctx context.Context) error {
	log.Println("Executing task 3 specific logic...")
	select {
	case <-time.After(4 * time.Second):
		log.Println("Task 3 completed")
	case <-ctx.Done():
		log.Println("Task 3 canceled")
		return ctx.Err()
	}
	return nil
}

type QueueConfig struct {
	Clientset   kubernetes.Interface
	RedisClient *redis.ClusterClient
	QueueName   string
	PageSize    int64
}

func UpdateNodeInfo(ctx context.Context, config *QueueConfig) error {
	continueToken := ""

	for {
		nodes, err := config.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			Limit:    config.PageSize,
			Continue: continueToken,
		})
		if err != nil {
			return fmt.Errorf("can't get node list: %v", err)
		}

		nodeNames := make([]string, 0, len(nodes.Items))
		for _, node := range nodes.Items {
			nodeNames = append(nodeNames, node.Name)
		}

		task := queue.Task{
			Type:    "node_batch",
			Payload: nodeNames,
		}

		// 序列化任务
		taskData, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal task: %v", err)
			continue
		}

		err = config.RedisClient.RPush(ctx, config.QueueName, taskData).Err()
		if err != nil {
			log.Printf("Failed to push task to queue: %v", err)
		} else {
			fmt.Printf("Pushed task: %s\n", taskData)
		}

		if nodes.Continue == "" {
			break
		}
		continueToken = nodes.Continue
	}

	return nil
}

func DeleteNonExistingNodeResourceInfoTask(ctx context.Context, opts crd.DeleteNodeResourceInfoOptions) error {
	log.Println("Running DeleteNonExistingNodeResourceInfo task...")

	err := crd.DeleteNonExistingNodeResourceInfo(opts)
	if err != nil {
		log.Printf("DeleteNonExistingNodeResourceInfo failed: %v", err)
		return err
	}

	log.Println("DeleteNonExistingNodeResourceInfo task completed")
	return nil
}
