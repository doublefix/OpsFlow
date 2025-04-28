package resourceinfo

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	pb "github.com/modcoco/OpsFlow/pkg/apis/proto"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	"github.com/modcoco/OpsFlow/pkg/apis/opsflow.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxRetries = 3               // 最大重试次数
	retryDelay = 1 * time.Second // 重试延迟
)

// 更新或创建 NodeResourceInfo CRD
func UpdateCreateNodeResourceInfo(crdClient dynamic.NamespaceableResourceInterface, grpcClient *grpc.ClientConn, nodeResourceInfo *v1alpha1.NodeResourceInfo, clusterId string) error {
	var retryCount int

	for {
		// 获取当前 CRD 资源
		existingResourceInfo, err := crdClient.Get(context.TODO(), nodeResourceInfo.Spec.NodeName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// CRD 不存在，则创建
				return CreateNodeResourceInfo(crdClient, grpcClient, nodeResourceInfo, clusterId)
			}
			return fmt.Errorf("获取 NodeResourceInfo 失败: %w", err)
		}

		// 检查是否需要更新
		needsUpdate, err := isNodeResourceInfoUpdated(existingResourceInfo, nodeResourceInfo)
		if err != nil {
			return fmt.Errorf("检查 NodeResourceInfo 资源变更失败: %w", err)
		}

		if !needsUpdate {
			log.Printf("NodeResourceInfo %s 没有变动，无需更新", nodeResourceInfo.Spec.NodeName)
			return nil
		}

		// 设置 resourceVersion 以支持乐观锁
		resourceVersion, found, err := unstructured.NestedString(existingResourceInfo.Object, "metadata", "resourceVersion")
		if err != nil || !found {
			return fmt.Errorf("无法获取现有 CRD 的 resourceVersion")
		}
		nodeResourceInfo.ObjectMeta.ResourceVersion = resourceVersion

		// 尝试更新
		err = updateNodeResourceInfo(crdClient, existingResourceInfo, nodeResourceInfo)
		if err == nil {
			// TODO
			c := pb.NewNodeManagerClient(grpcClient)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resources := make([]*pb.NodeResource, 0, len(nodeResourceInfo.Spec.Resources))
			for resourceName, resourceInfo := range nodeResourceInfo.Spec.Resources {
				var capacity, allocatable, unit string

				switch resourceName {
				case "cpu":
					capacity = strings.TrimSuffix(resourceInfo.Total, "m")
					allocatable = strings.TrimSuffix(resourceInfo.Allocatable, "m")
					unit = "m"
				case "memory":
					capacity = strings.TrimSuffix(resourceInfo.Total, "Mi")
					allocatable = strings.TrimSuffix(resourceInfo.Allocatable, "Mi")
					unit = "Mi"
				default:
					capacity = resourceInfo.Total
					allocatable = resourceInfo.Allocatable
					unit = ""
				}
				resources = append(resources, &pb.NodeResource{
					ResourceName: resourceName,
					Capacity:     capacity,
					Allocatable:  allocatable,
					Unit:         unit,
					IsRemoved:    false,
				})
			}
			addResp, err := c.UpdateNode(ctx, &pb.UpdateNodeRequest{
				NodeName:   nodeResourceInfo.Name,
				ClusterId:  clusterId,
				NodeStatus: nodeResourceInfo.Spec.Status,
				Resources:  resources,
			})

			if err != nil {
				log.Printf("Failed to call AddNode: %v", err)
				return fmt.Errorf("调用 rpc AddNode 失败: %w", err)
			}
			genericResp := addResp
			log.Printf("Received response: %v", genericResp)

			var addNodeResp pb.AddNodeResponse
			if err := genericResp.GetData().UnmarshalTo(&addNodeResp); err != nil {
				log.Printf("Failed to unmarshal AddNodeResponse from data: %v", err)
			}
			log.Printf("Add node: %s", addNodeResp.GetNodeName())

			log.Printf("NodeResourceInfo %s 已创建", nodeResourceInfo.Name)
			return nil // 更新成功
		}

		// 处理冲突
		if errors.IsConflict(err) {
			retryCount++
			if retryCount >= maxRetries {
				return fmt.Errorf("更新 NodeResourceInfo %s 失败，已达到最大重试次数: %w", nodeResourceInfo.Spec.NodeName, err)
			}

			log.Printf("NodeResourceInfo %s 更新冲突，正在重试 (重试次数: %d/%d)", nodeResourceInfo.Spec.NodeName, retryCount, maxRetries)
			time.Sleep(retryDelay) // 延迟后重试
			continue
		}

		// 其他错误
		return fmt.Errorf("无法更新 NodeResourceInfo CRD: %w", err)
	}
}

// createNodeResourceInfo 创建新的 NodeResourceInfo CRD
func CreateNodeResourceInfo(crdClient dynamic.NamespaceableResourceInterface, grpcClient *grpc.ClientConn, nodeResourceInfo *v1alpha1.NodeResourceInfo, clusterId string) error {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(nodeResourceInfo)
	if err != nil {
		return fmt.Errorf("无法转换 NodeResourceInfo 对象: %w", err)
	}

	unstructuredObj["kind"] = "NodeResourceInfo"
	unstructuredObj["apiVersion"] = "opsflow.io/v1alpha1"

	// 不存在则创建
	_, err = crdClient.Get(context.TODO(), nodeResourceInfo.Name, metav1.GetOptions{})
	if err == nil {
		log.Printf("NodeResourceInfo %s 已存在，跳过创建", nodeResourceInfo.Name)
	} else if errors.IsNotFound(err) {
		_, err = crdClient.Create(
			context.TODO(),
			&unstructured.Unstructured{Object: unstructuredObj},
			metav1.CreateOptions{},
		)
		if err != nil {
			return fmt.Errorf("无法创建 NodeResourceInfo CRD: %w", err)
		}
		log.Printf("成功创建 NodeResourceInfo %s", nodeResourceInfo.Name)
	} else {
		return fmt.Errorf("无法查询 NodeResourceInfo %s: %w", nodeResourceInfo.Name, err)
	}

	// TODO：Add Node
	c := pb.NewNodeManagerClient(grpcClient)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	log.Printf("NodeResourceInfo %s 创建中", nodeResourceInfo.Name)
	resources := make([]*pb.NodeResource, 0, len(nodeResourceInfo.Spec.Resources))
	for resourceName, resourceInfo := range nodeResourceInfo.Spec.Resources {
		var capacity, allocatable, unit string

		switch resourceName {
		case "cpu":
			capacity = strings.TrimSuffix(resourceInfo.Total, "m")
			allocatable = strings.TrimSuffix(resourceInfo.Allocatable, "m")
			unit = "m"
		case "memory":
			capacity = strings.TrimSuffix(resourceInfo.Total, "Mi")
			allocatable = strings.TrimSuffix(resourceInfo.Allocatable, "Mi")
			unit = "Mi"
		default:
			capacity = resourceInfo.Total
			allocatable = resourceInfo.Allocatable
			unit = ""
		}
		resources = append(resources, &pb.NodeResource{
			ResourceName: resourceName,
			Capacity:     capacity,
			Allocatable:  allocatable,
			Unit:         unit,
			IsRemoved:    false,
		})
	}

	addResp, err := c.AddNode(ctx, &pb.AddNodeRequest{
		NodeName:   nodeResourceInfo.Name,
		ClusterId:  clusterId,
		NodeStatus: nodeResourceInfo.Spec.Status,
		Resources:  resources,
	})

	if err != nil {
		log.Printf("Failed to call AddNode: %v", err)
		return fmt.Errorf("调用 rpc AddNode 失败: %w", err)
	}
	genericResp := addResp
	log.Printf("Received response: %v", genericResp)

	var addNodeResp pb.AddNodeResponse
	if err := genericResp.GetData().UnmarshalTo(&addNodeResp); err != nil {
		log.Printf("Failed to unmarshal AddNodeResponse from data: %v", err)
	}
	log.Printf("Add node: %s", addNodeResp.GetNodeName())

	log.Printf("NodeResourceInfo %s 已创建", nodeResourceInfo.Name)
	return nil
}

// 更新已有的 NodeResourceInfo CRD
func updateNodeResourceInfo(crdClient dynamic.NamespaceableResourceInterface, existing *unstructured.Unstructured, nodeResourceInfo *v1alpha1.NodeResourceInfo) error {
	existing.Object["spec"] = nodeResourceInfo.Spec

	_, err := crdClient.Update(context.TODO(), existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("无法更新 NodeResourceInfo CRD: %w", err)
	}

	log.Printf("NodeResourceInfo %s 已更新", nodeResourceInfo.Name)
	return nil
}

// 检查 CRD 是否需要更新
func isNodeResourceInfoUpdated(existing *unstructured.Unstructured, newNodeResourceInfo *v1alpha1.NodeResourceInfo) (bool, error) {
	// 将 Unstructured 转换为 NodeResourceInfo
	var existingNodeResourceInfo v1alpha1.NodeResourceInfo
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(existing.UnstructuredContent(), &existingNodeResourceInfo)
	if err != nil {
		return true, fmt.Errorf("转换 Unstructured 到 NodeResourceInfo 失败: %v", err)
	}

	// 检查 Status 是否变化
	if existingNodeResourceInfo.Spec.Status != newNodeResourceInfo.Spec.Status {
		log.Printf("Status 发生变化: 旧值 = %v, 新值 = %v", existingNodeResourceInfo.Spec.Status, newNodeResourceInfo.Spec.Status)
		return true, nil
	}

	// 检查 Resources 是否变化
	if len(existingNodeResourceInfo.Spec.Resources) != len(newNodeResourceInfo.Spec.Resources) {
		log.Printf("Resources 数量发生变化: 旧数量 = %d, 新数量 = %d", len(existingNodeResourceInfo.Spec.Resources), len(newNodeResourceInfo.Spec.Resources))
		return true, nil
	}

	for resourceName, newResource := range newNodeResourceInfo.Spec.Resources {
		existingResource, exists := existingNodeResourceInfo.Spec.Resources[resourceName]
		if !exists {
			log.Printf("资源 %s 在现有 CRD 中不存在，新增该资源", resourceName)
			return true, nil
		}

		if existingResource.Total != newResource.Total ||
			existingResource.Allocatable != newResource.Allocatable ||
			existingResource.Used != newResource.Used {
			log.Printf("资源 %s 发生变化: 旧值 = %+v, 新值 = %+v", resourceName, existingResource, newResource)
			return true, nil
		}
	}

	return false, nil
}
