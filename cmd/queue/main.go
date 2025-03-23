package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

// Task 定义任务结构体
type Task struct {
	Type    string `json:"type"`    // 任务类型
	Payload any    `json:"payload"` // 任务数据
}

// TaskHandler 定义任务处理接口
type TaskHandler interface {
	Handle(payload any) error
}

// EmailHandler 处理邮件任务
type EmailHandler struct{}

func (h *EmailHandler) Handle(payload interface{}) error {
	emailContent, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload for email task: %v", payload)
	}
	fmt.Printf("Sending email: %s\n", emailContent)
	return nil
}

// NotificationHandler 处理通知任务
type NotificationHandler struct{}

func (h *NotificationHandler) Handle(payload interface{}) error {
	notificationContent, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload for notification task: %v", payload)
	}
	fmt.Printf("Sending notification: %s\n", notificationContent)
	return nil
}

// ReportHandler 处理报告任务
type ReportHandler struct{}

func (h *ReportHandler) Handle(payload any) error {
	reportContent, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload for report task: %v", payload)
	}
	fmt.Printf("Generating report: %s\n", reportContent)
	return nil
}

// NodeBatchHandler 处理批量节点任务
type NodeBatchHandler struct{}

func (h *NodeBatchHandler) Handle(payload any) error {
	// 将 payload 转换为 []interface{}
	payloadSlice, ok := payload.([]any)
	if !ok {
		return fmt.Errorf("invalid payload for node_batch task: %v", payload)
	}

	// 将 []interface{} 转换为 []string
	nodeNames := make([]string, 0, len(payloadSlice))
	for _, item := range payloadSlice {
		nodeName, ok := item.(string)
		if !ok {
			return fmt.Errorf("invalid node name in payload: %v", item)
		}
		nodeNames = append(nodeNames, nodeName)
	}

	fmt.Printf("Processing node batch: %v\n", nodeNames)
	// 在这里处理批量节点数据
	return nil
}

// TaskProcessor 任务处理器
type TaskProcessor struct {
	handlers map[string]TaskHandler
}

func NewTaskProcessor() *TaskProcessor {
	return &TaskProcessor{
		handlers: map[string]TaskHandler{
			"email":        &EmailHandler{},
			"notification": &NotificationHandler{},
			"report":       &ReportHandler{},
			"node_batch":   &NodeBatchHandler{}, // 注册批量节点任务处理器
		},
	}
}

func (p *TaskProcessor) Process(task Task) error {
	handler, ok := p.handlers[task.Type]
	if !ok {
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
	return handler.Handle(task.Payload)
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
	go monitorTaskQueue(ctx, redisClient, "task_queue", taskChannel)

	// 启动 Worker Pool
	workerCount := 3 // 设置 Worker 数量
	var wg sync.WaitGroup
	processor := NewTaskProcessor()

	for i := range workerCount {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processTasks(ctx, workerID, taskChannel, processor)
		}(i)
	}

	// 监听退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅退出
	close(taskChannel)
	wg.Wait()
	log.Println("All workers have exited.")
}

// monitorTaskQueue 监控 Redis 任务队列并将任务发送到 Channel
func monitorTaskQueue(ctx context.Context, client *redis.ClusterClient, queueName string, taskChannel chan<- Task) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Task monitoring stopped.")
			return
		default:
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
}

// processTasks 从 Channel 中读取任务并处理
func processTasks(ctx context.Context, workerID int, taskChannel <-chan Task, processor *TaskProcessor) {
	for task := range taskChannel {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d exiting...\n", workerID)
			return
		default:
			// 处理任务
			fmt.Printf("Worker %d processing task: %v\n", workerID, task.Payload)
			if err := processor.Process(task); err != nil {
				log.Printf("Worker %d failed to process task: %v\n", workerID, err)
			}

			// 模拟任务处理时间
			time.Sleep(1 * time.Second)
		}
	}
}
