package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/modcoco/OpsFlow/pkg/node"
	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Task struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type TaskHandler interface {
	Handle(payload any) error
}

type EmailHandler struct{}

func (h *EmailHandler) Handle(payload interface{}) error {
	emailContent, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload for email task: %v", payload)
	}
	fmt.Printf("Sending email: %s\n", emailContent)
	return nil
}

type NotificationHandler struct{}

func (h *NotificationHandler) Handle(payload interface{}) error {
	notificationContent, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload for notification task: %v", payload)
	}
	fmt.Printf("Sending notification: %s\n", notificationContent)
	return nil
}

type ReportHandler struct{}

func (h *ReportHandler) Handle(payload any) error {
	reportContent, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload for report task: %v", payload)
	}
	fmt.Printf("Generating report: %s\n", reportContent)
	return nil
}

type NodeBatchHandler struct {
	clientset *kubernetes.Clientset
	crdClient *dynamic.NamespaceableResourceInterface
}

func NewNodeBatchHandler(clientset *kubernetes.Clientset, crdClient *dynamic.NamespaceableResourceInterface) *NodeBatchHandler {
	return &NodeBatchHandler{
		clientset: clientset,
		crdClient: crdClient,
	}
}

func (h *NodeBatchHandler) Handle(payload any) error {
	nodeNames, err := extractNodeNames(payload)
	if err != nil {
		return fmt.Errorf("failed to extract node names: %v", err)
	}

	fmt.Printf("Processing node batch: %v\n", nodeNames)

	nodes, err := h.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/hostname in (%s)", strings.Join(nodeNames, ",")), // 使用 in 语法批量过滤
	})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	resourceNamesToTrack := map[string]bool{
		"cpu":            true,
		"memory":         true,
		"nvidia.com/gpu": true,
	}

	opts := node.BatchUpdateCreateOptions{
		Clientset:            h.clientset,
		CRDClient:            h.crdClient,
		Nodes:                nodes,
		ResourceNamesToTrack: resourceNamesToTrack,
		Parallelism:          3,
	}

	if err := node.BatchAddNodeResourceInfo(opts); err != nil {
		return fmt.Errorf("failed to batch update node resource info: %v", err)
	}

	return nil
}

func extractNodeNames(payload any) ([]string, error) {
	payloadSlice, ok := payload.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type: expected []any, got %T", payload)
	}

	nodeNames := make([]string, 0, len(payloadSlice))
	for _, item := range payloadSlice {
		nodeName, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("invalid node name in payload: %v", item)
		}
		nodeNames = append(nodeNames, nodeName)
	}

	return nodeNames, nil
}

// 任务处理器
type TaskProcessor struct {
	handlers map[string]TaskHandler
}

func NewTaskProcessor(clientset *kubernetes.Clientset, crdClient *dynamic.NamespaceableResourceInterface) *TaskProcessor {
	return &TaskProcessor{
		handlers: map[string]TaskHandler{
			"email":        &EmailHandler{},
			"notification": &NotificationHandler{},
			"report":       &ReportHandler{},
			"node_batch":   NewNodeBatchHandler(clientset, crdClient), // 传入 Kubernetes 客户端
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

func monitorTaskQueue(ctx context.Context, client *redis.ClusterClient, queueName string, taskChannel chan<- Task) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Task monitoring stopped.")
			return
		default:
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

// 从 Channel 中读取任务
func processTasks(ctx context.Context, workerID int, taskChannel <-chan Task, processor *TaskProcessor) {
	for task := range taskChannel {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d exiting...\n", workerID)
			return
		default:
			fmt.Printf("Worker %d processing task: %v\n", workerID, task.Payload)
			if err := processor.Process(task); err != nil {
				log.Printf("Worker %d failed to process task: %v\n", workerID, err)
			}
		}
	}
}
