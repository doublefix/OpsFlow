package tests

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/utils"
	v1 "k8s.io/api/core/v1"
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
		status := GetNodeStatusString(&node)
		fmt.Println("STATUS:", status)

		fmt.Println("--------------------------------------------------")

		fmt.Printf("  [CPU]\n")
		fmt.Printf("    总资源: %d 核 (%d mCPU)\n", totalCPU.Value(), totalCPU.MilliValue())
		fmt.Printf("    已分配: %d 核 (%d mCPU)\n", usedCPU.Value(), usedCPU.MilliValue())
		fmt.Printf("    可分配: %d 核 (%d mCPU)\n", allocatableCPU.Value(), allocatableCPU.MilliValue())

		// Memory 信息
		fmt.Printf("  [Memory]\n")
		fmt.Printf("    总资源: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(totalMemory, resource.Kilo), utils.ScaledValue(totalMemory, resource.Mega), utils.ScaledValue(totalMemory, resource.Giga))
		fmt.Printf("    已分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(usedMemory, resource.Kilo), utils.ScaledValue(usedMemory, resource.Mega), utils.ScaledValue(usedMemory, resource.Giga))
		fmt.Printf("    可分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(allocatableMemory, resource.Kilo), utils.ScaledValue(allocatableMemory, resource.Mega), utils.ScaledValue(allocatableMemory, resource.Giga))

		// GPU 信息
		fmt.Printf("  [GPU]\n")
		fmt.Printf("    总资源: %d\n", totalGPU.Value())
		fmt.Printf("    已分配: %d\n", usedGPU.Value())
		fmt.Printf("    可分配: %d\n", allocatableGPU.Value())

	}
}

func TestGetNamespaceUID(t *testing.T) {
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

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(namespace.GetUID())
}

func GetNodeStatusString(node *v1.Node) string {
	var statuses []string

	for _, condition := range node.Status.Conditions {
		if condition.Status == v1.ConditionTrue {
			statuses = append(statuses, string(condition.Type))
		}
	}

	if node.Spec.Unschedulable {
		statuses = append(statuses, "SchedulingDisabled")
	}

	return strings.Join(statuses, ",")
}
