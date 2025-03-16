package tests

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/utils"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGetNodeResources(t *testing.T) {
	// 指定需要统计的资源名称
	resourceNamesToTrack := map[string]bool{
		"cpu":            true, // 统计 CPU
		"memory":         true, // 统计内存
		"nvidia.com/gpu": true, // 统计 GPU
	}

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
		fmt.Printf("Node: %s\n", node.Name)
		fmt.Println("--------------------------------------------------")

		// 获取所有资源类型
		for resourceName, totalResource := range node.Status.Capacity {
			// 如果资源不在指定列表中，则跳过
			if !resourceNamesToTrack[string(resourceName)] {
				continue
			}

			allocatableResource := node.Status.Allocatable[resourceName]

			// 计算已分配资源
			var usedResource resource.Quantity
			for _, pod := range pods.Items {
				if pod.Spec.NodeName == node.Name {
					for _, container := range pod.Spec.Containers {
						if request, ok := container.Resources.Requests[resourceName]; ok {
							usedResource.Add(request)
						}
					}
				}
			}

			fmt.Printf("ResourceName => %s \n", resourceName)
			if resourceName == "cpu" {
				fmt.Printf("    总资源: %d 核 (%d mCPU)\n", totalResource.Value(), totalResource.MilliValue())
				fmt.Printf("    已分配: %d 核 (%d mCPU)\n", usedResource.Value(), usedResource.MilliValue())
				fmt.Printf("    可分配: %d 核 (%d mCPU)\n", allocatableResource.Value(), allocatableResource.MilliValue())
			} else if resourceName == "memory" {
				fmt.Printf("    总资源: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(totalResource, resource.Kilo), utils.ScaledValue(totalResource, resource.Mega), utils.ScaledValue(totalResource, resource.Giga))
				fmt.Printf("    已分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(usedResource, resource.Kilo), utils.ScaledValue(usedResource, resource.Mega), utils.ScaledValue(usedResource, resource.Giga))
				fmt.Printf("    可分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(allocatableResource, resource.Kilo), utils.ScaledValue(allocatableResource, resource.Mega), utils.ScaledValue(allocatableResource, resource.Giga))
			} else {
				fmt.Printf("  [%s]\n", resourceName)
				fmt.Printf("    总资源: %s\n", totalResource.String())
				fmt.Printf("    已分配: %s\n", usedResource.String())
				fmt.Printf("    可分配: %s\n", allocatableResource.String())
			}
		}
	}
}
