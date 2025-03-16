package tests

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/apis/opsflow.io/v1alpha1" // 需要导入 v1alpha1 包
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestCreateNodeResourceInfo 用于创建 NodeResourceInfo CRD
func TestCreateNodeResourceInfo(t *testing.T) {
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

	// 获取 Pod 列表
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("无法获取 Pod 列表: %v", err)
	}

	// 遍历所有节点
	for _, node := range nodes.Items {
		fmt.Printf("Node: %s\n", node.Name)

		// 创建 NodeResourceInfo 对象
		nodeResourceInfo := &v1alpha1.NodeResourceInfo{
			ObjectMeta: metav1.ObjectMeta{
				Name: node.Name,
			},
			Spec: v1alpha1.NodeResourceInfoSpec{
				NodeName:  node.Name,
				Resources: map[string]v1alpha1.ResourceInfo{},
			},
		}

		// 遍历节点的资源信息
		for resourceName, totalResource := range node.Status.Capacity {
			// 如果资源不在追踪列表中，则跳过
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

			// 将资源信息添加到 NodeResourceInfo 对象中
			nodeResourceInfo.Spec.Resources[string(resourceName)] = v1alpha1.ResourceInfo{
				Total:       totalResource.String(),
				Allocatable: allocatableResource.String(),
				Used:        usedResource.String(),
			}
		}

		// 使用动态客户端创建 CRD 实例
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(nodeResourceInfo)
		if err != nil {
			log.Fatalf("无法转换 NodeResourceInfo 对象: %v", err)
		}

		// 将 "Kind" 字段添加到 unstructured 对象
		unstructuredObj["kind"] = "NodeResourceInfo"
		// 将 "apiVersion" 字段添加到 unstructured 对象
		unstructuredObj["apiVersion"] = "opsflow.io/v1alpha1"

		// 创建 NodeResourceInfo CRD
		_, err = crdClient.Create(context.TODO(), &unstructured.Unstructured{Object: unstructuredObj}, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("无法创建 NodeResourceInfo CRD: %v", err)
		}
	}
}
