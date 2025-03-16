package tests

import (
	"context"
	"log"
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
		Parallelism:          1,
	}

	if err := node.BatchAddNodeResourceInfo(opts); err != nil {
		log.Fatalf("批量更新或创建 NodeResourceInfo 失败: %v", err)
	}

	crd.DeleteNonExistingNodeResourceInfo(crdClient, nodes)
}
