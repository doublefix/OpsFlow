package tests

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/crd"
	"github.com/modcoco/OpsFlow/pkg/node"
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
		CRDClient:            crdClient,
		Nodes:                nodes,
		ResourceNamesToTrack: resourceNamesToTrack,
		Parallelism:          3,
	}

	optsDelCRD := crd.DeleteNodeResourceInfoOptions{
		CRDClient:   crdClient,
		BatchNodes:  nodes,
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

// const taskRunningKey = "task_running" // 任务状态标记的 Key

// func isTaskRunning(rdb *redis.Client) (bool, error) {
// 	result, err := rdb.Get(context.TODO(), taskRunningKey).Result()
// 	if err == redis.Nil {
// 		return false, nil // 任务状态标记不存在，表示没有任务正在运行
// 	}
// 	if err != nil {
// 		return false, fmt.Errorf("获取任务状态失败: %w", err)
// 	}
// 	return result == "true", nil
// }

// func setTaskRunning(rdb *redis.Client, running bool) error {
// 	value := "false"
// 	if running {
// 		value = "true"
// 	}
// 	return rdb.Set(context.TODO(), taskRunningKey, value, 0).Err()
// }

// func startScheduler(clientset *kubernetes.Clientset, rdb *redis.Client, instanceID string, totalShards int, pageSize int, interval time.Duration) {
// 	ticker := time.NewTicker(interval)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		// 检查任务状态
// 		running, err := isTaskRunning(rdb)
// 		if err != nil {
// 			log.Printf("检查任务状态失败: %v", err)
// 			continue
// 		}
// 		if running {
// 			log.Println("任务正在运行，跳过本次触发")
// 			continue
// 		}

// 		// 设置任务状态为运行中
// 		if err := setTaskRunning(rdb, true); err != nil {
// 			log.Printf("设置任务状态失败: %v", err)
// 			continue
// 		}

// 		// 执行任务
// 		if err := processNodesInPages(clientset, rdb, instanceID, totalShards, pageSize); err != nil {
// 			log.Printf("任务执行失败: %v", err)
// 		} else {
// 			log.Println("任务执行成功")
// 		}

// 		// 设置任务状态为未运行
// 		if err := setTaskRunning(rdb, false); err != nil {
// 			log.Printf("重置任务状态失败: %v", err)
// 		}
// 	}
// }

// func processNodesInPages(clientset *kubernetes.Clientset, rdb *redis.Client, instanceID string, totalShards int, pageSize int) error {
// 	var continueToken string
// 	lockKey := "node_shard_lock"
// 	shardKey := "node_shards"

// 	// 尝试获取分布式锁
// 	locked, err := acquireLock(rdb, lockKey, instanceID, 10*time.Second)
// 	if err != nil {
// 		return fmt.Errorf("获取分布式锁失败: %w", err)
// 	}

// 	if locked {
// 		defer rdb.Del(context.TODO(), lockKey) // 释放锁
// 	}

// 	for {
// 		// 分页获取节点
// 		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
// 			Limit:    int64(pageSize),
// 			Continue: continueToken,
// 		})
// 		if err != nil {
// 			return fmt.Errorf("分页获取节点失败: %w", err)
// 		}

// 		if locked {
// 			// 当前实例负责分配分片
// 			if err := assignShards(rdb, shardKey, nodes, totalShards); err != nil {
// 				return fmt.Errorf("分配分片失败: %w", err)
// 			}
// 		} else {
// 			// 等待分片分配完成
// 			time.Sleep(1 * time.Second)
// 		}

// 		// 获取当前实例的分片
// 		shard := getNodeShard(instanceID, totalShards)
// 		shardNodes, err := getNodesForShard(rdb, shardKey, shard, nodes)
// 		if err != nil {
// 			return fmt.Errorf("获取分片节点失败: %w", err)
// 		}

// 		// 处理分片任务
// 		if err := processShardNodes(clientset, shardNodes); err != nil {
// 			return fmt.Errorf("处理分片节点失败: %w", err)
// 		}

// 		continueToken = nodes.Continue
// 		if continueToken == "" {
// 			break
// 		}
// 	}

// 	return nil
// }

// func processShardNodes(clientset *kubernetes.Clientset, nodes *corev1.NodeList) error {
// 	opts := node.BatchUpdateCreateOptions{
// 		Clientset:            clientset,
// 		CRDClient:            crdClient,
// 		Nodes:                nodes,
// 		ResourceNamesToTrack: resourceNamesToTrack,
// 		Parallelism:          1,
// 	}

// 	if err := node.BatchAddNodeResourceInfo(opts); err != nil {
// 		return fmt.Errorf("批量更新或创建 NodeResourceInfo 失败: %w", err)
// 	}

// 	// 删除不存在的节点资源
// 	crd.DeleteNonExistingNodeResourceInfo(crdClient, nodes)

// 	return nil
// }

// func acquireLock(rdb *redis.Client, lockKey string, instanceID string, ttl time.Duration) (bool, error) {
// 	result, err := rdb.SetNX(context.TODO(), lockKey, instanceID, ttl).Result()
// 	if err != nil {
// 		return false, fmt.Errorf("获取锁失败: %w", err)
// 	}
// 	return result, nil
// }

// func assignShards(rdb *redis.Client, shardKey string, nodes *corev1.NodeList, totalShards int) error {
// 	// 清空旧的分片信息
// 	if err := rdb.Del(context.TODO(), shardKey).Err(); err != nil {
// 		return fmt.Errorf("清空分片信息失败: %w", err)
// 	}

// 	// 计算每个节点的分片并存储到 Redis
// 	for _, node := range nodes.Items {
// 		shard := getNodeShard(node.Name, totalShards)
// 		if err := rdb.HSet(context.TODO(), shardKey, node.Name, shard).Err(); err != nil {
// 			return fmt.Errorf("存储分片信息失败: %w", err)
// 		}
// 	}

// 	return nil
// }

// // 计算节点分片
// func getNodeShard(nodeName string, totalShards int) int {
// 	h := fnv.New32a()
// 	h.Write([]byte(nodeName))
// 	return int(h.Sum32()) % totalShards
// }

// func getNodesForShard(rdb *redis.Client, shardKey string, shard int, nodes *corev1.NodeList) (*corev1.NodeList, error) {
// 	// 获取所有节点的分片信息
// 	shardMap, err := rdb.HGetAll(context.TODO(), shardKey).Result()
// 	if err != nil {
// 		return nil, fmt.Errorf("获取分片信息失败: %w", err)
// 	}

// 	// 过滤出当前实例需要处理的节点
// 	var filteredNodes []corev1.Node
// 	for _, node := range nodes.Items {
// 		if shardMap[node.Name] == fmt.Sprintf("%d", shard) {
// 			filteredNodes = append(filteredNodes, node)
// 		}
// 	}

// 	return &corev1.NodeList{Items: filteredNodes}, nil
// }
