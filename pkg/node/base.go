package node

import (
	"fmt"
	"log"
	"sync"

	"github.com/modcoco/OpsFlow/pkg/apis/opsflow.io/v1alpha1"
	"github.com/modcoco/OpsFlow/pkg/node/resourceinfo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func CheckNodeExistsFromBatchList(nodeName string, batchNodesList *corev1.NodeList) bool {
	for _, node := range batchNodesList.Items {
		if node.Name == nodeName {
			return true
		}
	}
	return false
}

type BatchUpdateCreateOptions struct {
	Clientset            *kubernetes.Clientset
	CRDClient            dynamic.NamespaceableResourceInterface
	Nodes                *corev1.NodeList
	ResourceNamesToTrack map[string]bool
	Parallelism          int // 最大并行度，0 或 负值时表示无限制
}

// 批量添加 NodeResourceInfo
func BatchAddNodeResourceInfo(opts BatchUpdateCreateOptions) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(opts.Nodes.Items))

	// 限制并行度
	semaphore := make(chan struct{}, opts.Parallelism)
	if opts.Parallelism <= 0 {
		semaphore = nil // 不限并发数
	}

	for _, node := range opts.Nodes.Items {
		wg.Add(1)
		go func(n corev1.Node) {
			defer wg.Done()

			if semaphore != nil {
				semaphore <- struct{}{}        // 占用一个并发槽
				defer func() { <-semaphore }() // 释放并发槽
			}

			log.Printf("处理节点: %s", n.Name)

			nodeResourceInfo := &v1alpha1.NodeResourceInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name: n.Name,
				},
				Spec: v1alpha1.NodeResourceInfoSpec{
					NodeName:  n.Name,
					Resources: map[string]v1alpha1.ResourceInfo{},
				},
			}

			nodeQuery := resourceinfo.NodeResourceQuery{
				Clientset:            opts.Clientset,
				Node:                 &n,
				ResourceNamesToTrack: opts.ResourceNamesToTrack,
			}

			resourceinfo.LoadNodeResourceInfoFromNode(nodeQuery, nodeResourceInfo)
			err := resourceinfo.UpdateCreateNodeResourceInfo(opts.CRDClient, nodeResourceInfo)
			if err != nil {
				errCh <- fmt.Errorf("节点 %s 处理失败: %w", n.Name, err)
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	var finalErr error
	for err := range errCh {
		if finalErr == nil {
			finalErr = err
		} else {
			finalErr = fmt.Errorf("%v; %v", finalErr, err)
		}
	}
	return finalErr
}
