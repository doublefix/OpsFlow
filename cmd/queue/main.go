package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Task 定义任务结构体
type Task struct {
	Type    string `json:"type"`    // 任务类型
	Payload string `json:"payload"` // 任务数据
}

func main() {
	// 创建 Redis 集群客户端
	redisClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"10.187.6.3:31000",
			"10.187.6.4:31001",
			"10.187.6.5:31002",
			"10.187.6.3:31100",
			"10.187.6.4:31101",
			"10.187.6.5:31102",
		},
		Password: "pass12345",
	})

	// 检查 Redis 连接是否成功
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 创建一个 Channel 用于传递任务
	taskChannel := make(chan Task)

	// 启动任务监控 Goroutine
	go monitorTaskQueue(redisClient, "task_queue", taskChannel)

	// 启动 Worker Pool
	workerCount := 3 // 设置 Worker 数量
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processTasks(workerID, taskChannel)
		}(i)
	}

	// 保持主 Goroutine 运行
	wg.Wait()
}

// monitorTaskQueue 监控 Redis 任务队列并将任务发送到 Channel
func monitorTaskQueue(client *redis.ClusterClient, queueName string, taskChannel chan<- Task) {
	ctx := context.Background()

	for {
		// 使用 BLPOP 阻塞式地从队列中获取任务
		result, err := client.BLPop(ctx, 0, queueName).Result()
		if err != nil {
			log.Printf("Error while popping task from queue: %v", err)
			continue
		}

		// 反序列化任务数据
		var task Task
		taskData := result[1]
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			log.Printf("Failed to unmarshal task: %v", err)
			continue
		}

		// 将任务发送到 Channel
		taskChannel <- task
	}
}

// processTasks 从 Channel 中读取任务并处理
func processTasks(workerID int, taskChannel <-chan Task) {
	for task := range taskChannel {
		// 根据任务类型执行不同的逻辑
		switch task.Type {
		case "email":
			fmt.Printf("Worker %d processing email task: %s\n", workerID, task.Payload)
			sendEmail(task.Payload)
		case "notification":
			fmt.Printf("Worker %d processing notification task: %s\n", workerID, task.Payload)
			sendNotification(task.Payload)
		case "report":
			fmt.Printf("Worker %d processing report task: %s\n", workerID, task.Payload)
			generateReport(task.Payload)
		default:
			fmt.Printf("Worker %d received unknown task type: %s\n", workerID, task.Type)
		}

		// 模拟任务处理时间
		time.Sleep(1 * time.Second)
	}
}

// sendEmail 模拟发送邮件任务
func sendEmail(payload string) {
	fmt.Printf("Sending email: %s\n", payload)
}

// sendNotification 模拟发送通知任务
func sendNotification(payload string) {
	fmt.Printf("Sending notification: %s\n", payload)
}

// generateReport 模拟生成报告任务
func generateReport(payload string) {
	fmt.Printf("Generating report: %s\n", payload)
}
