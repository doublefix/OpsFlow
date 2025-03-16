package resourceinfo

import (
	"context"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	"github.com/modcoco/OpsFlow/pkg/apis/opsflow.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 更新或创建 NodeResourceInfo CRD
func UpdateCreateNodeResourceInfo(crdClient dynamic.NamespaceableResourceInterface, nodeResourceInfo *v1alpha1.NodeResourceInfo) error {
	// 获取当前 CRD 资源
	existingResourceInfo, err := crdClient.Get(context.TODO(), nodeResourceInfo.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		// CRD 不存在，则创建
		return createNodeResourceInfo(crdClient, nodeResourceInfo)
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

	return updateNodeResourceInfo(crdClient, existingResourceInfo, nodeResourceInfo)
}

// createNodeResourceInfo 创建新的 NodeResourceInfo CRD
func createNodeResourceInfo(crdClient dynamic.NamespaceableResourceInterface, nodeResourceInfo *v1alpha1.NodeResourceInfo) error {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(nodeResourceInfo)
	if err != nil {
		return fmt.Errorf("无法转换 NodeResourceInfo 对象: %w", err)
	}

	unstructuredObj["kind"] = "NodeResourceInfo"
	unstructuredObj["apiVersion"] = "opsflow.io/v1alpha1"

	_, err = crdClient.Create(context.TODO(), &unstructured.Unstructured{Object: unstructuredObj}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("无法创建 NodeResourceInfo CRD: %w", err)
	}

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
func isNodeResourceInfoUpdated(existing *unstructured.Unstructured, nodeResourceInfo *v1alpha1.NodeResourceInfo) (bool, error) {
	existingSpec, found, err := unstructured.NestedMap(existing.Object, "spec")
	if err != nil || !found {
		return true, fmt.Errorf("无法解析现有 CRD 的 spec 字段")
	}

	existingResources, found, err := unstructured.NestedMap(existingSpec, "resources")
	if err != nil || !found {
		return true, nil
	}

	for resourceName, resourceInfo := range nodeResourceInfo.Spec.Resources {
		existingResource, exists := existingResources[resourceName]

		if !exists {
			log.Printf("资源 %s 在现有 CRD 中不存在，新增该资源", resourceName)
			return true, nil
		}

		existingResourceMap := existingResource.(map[string]any)
		newResourceMap := map[string]any{
			"total":       resourceInfo.Total,
			"allocatable": resourceInfo.Allocatable,
			"used":        resourceInfo.Used,
		}

		for key, value := range newResourceMap {
			if existingValue, exists := existingResourceMap[key]; !exists || existingValue != value {
				log.Printf("资源 %s 的 %s 字段发生变化: 旧值 = %v, 新值 = %v", resourceName, key, existingValue, value)
				return true, nil
			}
		}
	}

	return false, nil
}
