package resourceinfo

import (
	"context"
	"fmt"

	"github.com/modcoco/OpsFlow/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/modcoco/OpsFlow/pkg/apis/opsflow.io/v1alpha1"
)

type NodeResourceQuery struct {
	Clientset            *kubernetes.Clientset
	Node                 *v1.Node
	ResourceNamesToTrack map[string]bool
}

// 更新 NodeResourceInfo
func LoadNodeResourceInfoFromNode(query NodeResourceQuery, nodeResourceInfo *v1alpha1.NodeResourceInfo) error {
	nodeResourceInfo.ObjectMeta = metav1.ObjectMeta{
		Name: query.Node.Name,
	}
	nodeResourceInfo.Spec = v1alpha1.NodeResourceInfoSpec{
		NodeName:  query.Node.Name,
		Resources: make(map[string]v1alpha1.ResourceInfo),
	}

	for resourceName, totalResource := range query.Node.Status.Capacity {
		if !query.ResourceNamesToTrack[string(resourceName)] {
			continue
		}

		allocatableResource := query.Node.Status.Allocatable[resourceName]

		var usedResource resource.Quantity
		pods, err := query.Clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", query.Node.Name),
		})
		if err != nil {
			return fmt.Errorf("无法获取 Pod 列表: %w", err)
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				if request, ok := container.Resources.Requests[resourceName]; ok {
					usedResource.Add(request)
				}
			}
		}

		resName := string(resourceName)
		var resourceInfo v1alpha1.ResourceInfo
		if resName == "cpu" {
			resourceInfo = v1alpha1.ResourceInfo{
				Total:       fmt.Sprintf("%dm", totalResource.MilliValue()),
				Allocatable: fmt.Sprintf("%dm", allocatableResource.MilliValue()),
				Used:        fmt.Sprintf("%dm", usedResource.MilliValue()),
			}
		} else if resName == "memory" {
			resourceInfo = v1alpha1.ResourceInfo{
				Total:       fmt.Sprintf("%dMi", utils.ScaledValue(totalResource, resource.Mega)),
				Allocatable: fmt.Sprintf("%dMi", utils.ScaledValue(allocatableResource, resource.Mega)),
				Used:        fmt.Sprintf("%dMi", utils.ScaledValue(usedResource, resource.Mega)),
			}
		} else {
			resourceInfo = v1alpha1.ResourceInfo{
				Total:       totalResource.String(),
				Allocatable: allocatableResource.String(),
				Used:        usedResource.String(),
			}
		}

		nodeResourceInfo.Spec.Resources[resName] = resourceInfo
	}

	return nil
}
