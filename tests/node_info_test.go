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

func TestGetNodeResources(t *testing.T) {
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

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("无法获取 Pod 列表: %v", err)
	}

	for _, node := range nodes.Items {
		// 获取总资源（Capacity）
		totalCPU := node.Status.Capacity["cpu"]
		totalMemory := node.Status.Capacity["memory"]
		totalGPU := node.Status.Capacity["nvidia.com/gpu"]

		// 获取可分配资源（Allocatable）
		allocatableCPU := node.Status.Allocatable["cpu"]
		allocatableMemory := node.Status.Allocatable["memory"]
		allocatableGPU := node.Status.Allocatable["nvidia.com/gpu"]

		// 计算已分配资源（Allocated）
		var usedCPU, usedMemory, usedGPU resource.Quantity
		for _, pod := range pods.Items {
			if pod.Spec.NodeName == node.Name {
				for _, container := range pod.Spec.Containers {
					usedCPU.Add(*container.Resources.Requests.Cpu())
					usedMemory.Add(*container.Resources.Requests.Memory())
					if gpu, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
						usedGPU.Add(gpu)
					}
				}
			}
		}

		fmt.Printf("Node: %s\n", node.Name)
		fmt.Println("--------------------------------------------------")

		fmt.Printf("  [CPU]\n")
		fmt.Printf("    总资源: %d 核 (%d mCPU)\n", totalCPU.Value(), totalCPU.MilliValue())
		fmt.Printf("    已分配: %d 核 (%d mCPU)\n", usedCPU.Value(), usedCPU.MilliValue())
		fmt.Printf("    可分配: %d 核 (%d mCPU)\n", allocatableCPU.Value(), allocatableCPU.MilliValue())

		// Memory 信息
		fmt.Printf("  [Memory]\n")
		fmt.Printf("    总资源: %d KiB (%d MiB, %d GiB)\n", totalMemory.ScaledValue(resource.Kilo), totalMemory.ScaledValue(resource.Mega), totalMemory.ScaledValue(resource.Giga))
		fmt.Printf("    已分配: %d KiB (%d MiB, %d GiB)\n", usedMemory.ScaledValue(resource.Kilo), usedMemory.ScaledValue(resource.Mega), usedMemory.ScaledValue(resource.Giga))
		fmt.Printf("    可分配: %d KiB (%d MiB, %d GiB)\n", allocatableMemory.ScaledValue(resource.Kilo), allocatableMemory.ScaledValue(resource.Mega), allocatableMemory.ScaledValue(resource.Giga))

		// GPU 信息
		fmt.Printf("  [GPU]\n")
		fmt.Printf("    总资源: %d\n", totalGPU.Value())
		fmt.Printf("    已分配: %d\n", usedGPU.Value())
		fmt.Printf("    可分配: %d\n", allocatableGPU.Value())

	}
}
