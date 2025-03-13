package tests

import (
	"context"
	"fmt"
	"log"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGetNodeInfo(t *testing.T) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatalf("无法加载 kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("无法创建 Kubernetes 客户端: %v", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("无法获取节点列表: %v", err)
	}

	for _, node := range nodes.Items {
		cpuQty := node.Status.Capacity["cpu"]
		cpuCores := cpuQty.Value()
		cpuMilli := cpuQty.MilliValue()

		memQty := node.Status.Capacity["memory"]
		memKi := memQty.ScaledValue(resource.Kilo) // KiB
		memMi := memQty.ScaledValue(resource.Mega) // MiB
		memGi := memQty.ScaledValue(resource.Giga) // GiB

		gpuInfo := "nvidia.com/gpu: 0"
		if gpuQty, hasGPU := node.Status.Capacity["nvidia.com/gpu"]; hasGPU {
			gpuInfo = fmt.Sprintf("nvidia.com/gpu: %d", gpuQty.Value())
		}

		fmt.Printf("Node: %s\n", node.Name)
		fmt.Printf("  CPU: %d cores (%d mCPU)\n", cpuCores, cpuMilli)
		fmt.Printf("  Memory: %d Ki (%d Mi, %d Gi)\n", memKi, memMi, memGi)
		fmt.Printf("  %s\n", gpuInfo)
	}
}
