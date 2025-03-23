package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/modcoco/OpsFlow/pkg/crd"
	"github.com/modcoco/OpsFlow/pkg/node"
	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestCreateOrUpdateNodeResourceInfo 用于创建或更新 NodeResourceInfo CRD
func TestCreateOrUpdateNodeResourceInfo(t *testing.T) {
	// 需要追踪的资源类型
	resourceNamesToTrack := map[string]bool{
		"cpu":            true, // 统计 CPU
		"memory":         true, // 统计内存
		"nvidia.com/gpu": true, // 统计 GPU
	}

	// 加载 kubeconfig 配置
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatalf("无法加载 kubeconfig: %v", err)
	}

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("无法创建 Kubernetes 客户端: %v", err)
	}

	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("无法创建动态客户端: %v", err)
	}

	// 获取 CRD 客户端（用于管理 CRD）
	crdClient := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "opsflow.io",
		Version:  "v1alpha1",
		Resource: "noderesourceinfos",
	})

	// 获取节点列表
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("无法获取节点列表: %v", err)
	}

	opts := node.BatchUpdateCreateOptions{
		Clientset:            clientset,
		CRDClient:            &crdClient,
		Nodes:                nodes,
		ResourceNamesToTrack: resourceNamesToTrack,
		Parallelism:          3,
	}

	optsDelCRD := crd.DeleteNodeResourceInfoOptions{
		CRDClient:   crdClient,
		KubeClient:  clientset,
		Parallelism: 3,
	}

	// 多线程运行
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// 启动批量更新或创建 NodeResourceInfo
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := node.BatchAddNodeResourceInfo(opts); err != nil {
			errCh <- fmt.Errorf("批量更新或创建 NodeResourceInfo 失败: %w", err)
		}
	}()

	// 启动删除不存在的 NodeResourceInfo
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := crd.DeleteNonExistingNodeResourceInfo(optsDelCRD); err != nil {
			errCh <- fmt.Errorf("删除 NodeResourceInfo 失败: %w", err)
		}
	}()

	// 等待所有 goroutine 完成
	wg.Wait()
	close(errCh)

	// 处理错误
	var finalErr error
	for err := range errCh {
		if finalErr == nil {
			finalErr = err
		} else {
			finalErr = fmt.Errorf("%v; %v", finalErr, err)
		}
	}

	if finalErr != nil {
		log.Printf("批量操作过程中发生错误: %v", finalErr)
	}
}

type Task struct {
	Type    string `json:"type"`    // 任务类型
	Payload any    `json:"payload"` // 任务数据
}

func TestAddToQueue(t *testing.T) {
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

	// 动态生成任务并推送到队列
	queueName := "task_queue"
	for i := 1; i <= 10; i++ {
		task := Task{
			Type:    "email",
			Payload: fmt.Sprintf("Email content %d", i),
		}

		// 序列化任务
		taskData, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal task: %v", err)
			continue
		}

		// 推送任务到队列
		err = redisClient.RPush(ctx, queueName, taskData).Err()
		if err != nil {
			log.Printf("Failed to push task to queue: %v", err)
		} else {
			fmt.Printf("Pushed task: %s\n", taskData)
		}

		// 模拟任务生成间隔
		time.Sleep(500 * time.Millisecond)
	}
}

func TestAddNodeToQueue(t *testing.T) {
	// 加载 kubeconfig 配置
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatalf("无法加载 kubeconfig: %v", err)
	}

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("无法创建 Kubernetes 客户端: %v", err)
	}

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

	// 分页获取节点数据并推送到公共队列
	queueName := "task_queue" // 公共任务队列
	pageSize := int64(50)     // 每页最大50条
	continueToken := ""

	for {
		// 获取节点列表
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			Limit:    pageSize,
			Continue: continueToken,
		})
		if err != nil {
			log.Fatalf("无法获取节点列表: %v", err)
		}

		// 将一组节点数据打包成一个任务
		nodeNames := make([]string, 0, len(nodes.Items))
		for _, node := range nodes.Items {
			nodeNames = append(nodeNames, node.Name)
		}

		// 创建任务
		task := Task{
			Type:    "node_batch", // 任务类型为 "node_batch"
			Payload: nodeNames,    // 任务数据为节点名称列表
		}

		// 序列化任务
		taskData, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal task: %v", err)
			continue
		}

		// 推送任务到公共队列
		err = redisClient.RPush(ctx, queueName, taskData).Err()
		if err != nil {
			log.Printf("Failed to push task to queue: %v", err)
		} else {
			fmt.Printf("Pushed task: %s\n", taskData)
		}

		// 如果没有更多的数据，退出循环
		if nodes.Continue == "" {
			break
		}

		// 更新 continueToken 以获取下一页数据
		continueToken = nodes.Continue
	}
}
