package crd

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/modcoco/OpsFlow/pkg/node"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type DeleteNodeResourceInfoOptions struct {
	CRDClient   dynamic.ResourceInterface // CRD 客户端
	KubeClient  kubernetes.Interface      // Kubernetes 客户端
	Parallelism int                       // 并发数
}

// 批量删除不存在的 NodeResourceInfo CRD 实例
func DeleteNonExistingNodeResourceInfo(opts DeleteNodeResourceInfoOptions) error {
	var continueToken string
	var wg sync.WaitGroup
	errCh := make(chan error, 100) // 缓冲通道，防止阻塞

	semaphore := make(chan struct{}, opts.Parallelism)
	if opts.Parallelism <= 0 {
		semaphore = nil // 不限并发数
	}

	for {
		// 1. 获取 CRD 列表
		crdList, newContinueToken, err := GetCRDList(opts.CRDClient, continueToken)
		if err != nil {
			return fmt.Errorf("无法查询 CRD 实例: %w", err)
		}

		// 2. 收集 CRD 里的所有 nodeName
		crdNodeNames := make([]string, 0, len(crdList.Items))
		for _, crd := range crdList.Items {
			crdNodeNames = append(crdNodeNames, crd.GetName())
		}

		if len(crdNodeNames) == 0 {
			if newContinueToken == "" {
				break
			}
			continueToken = newContinueToken
			continue
		}

		// 3. 直接按 name 过滤批量查询 Node 是否存在
		nonExistingNodes, err := node.BatchCheckNodesNotExist(opts.KubeClient, crdNodeNames)
		if err != nil {
			return fmt.Errorf("查询 Node 失败: %w", err)
		}

		// 4. 并发删除不存在的 CRD 实例
		deleteCRDsConcurrently(opts.CRDClient, nonExistingNodes, semaphore, &wg, errCh)

		if newContinueToken == "" {
			break
		}
		continueToken = newContinueToken
	}

	wg.Wait()
	close(errCh)

	var allErrors []error
	for err := range errCh {
		allErrors = append(allErrors, err)
	}
	return errors.Join(allErrors...)
}

// 并发删除 CRD 实例
func deleteCRDsConcurrently(
	crdClient dynamic.ResourceInterface,
	nodeNames []string,
	semaphore chan struct{},
	wg *sync.WaitGroup,
	errCh chan<- error,
) {
	for _, nodeName := range nodeNames {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()

			if semaphore != nil {
				semaphore <- struct{}{}        // 占用并发槽
				defer func() { <-semaphore }() // 释放并发槽
			}

			if err := DeleteCRD(crdClient, n); err != nil {
				log.Printf("无法删除 NodeResourceInfo CRD %s: %v", n, err)
				errCh <- fmt.Errorf("删除失败: %s, 错误: %w", n, err)
			} else {
				log.Printf("已删除 NodeResourceInfo CRD %s", n)
			}
		}(nodeName)
	}
}
