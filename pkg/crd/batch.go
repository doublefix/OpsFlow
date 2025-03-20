package crd

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/modcoco/OpsFlow/pkg/node"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
)

type DeleteNodeResourceInfoOptions struct {
	CRDClient   dynamic.ResourceInterface // CRD 客户端
	BatchNodes  *corev1.NodeList          // 现有的 Node 列表
	Parallelism int                       // 并发数
}

// DeleteNonExistingNodeResourceInfo 删除不存在的 NodeResourceInfo CRD 实例
func DeleteNonExistingNodeResourceInfo(opts DeleteNodeResourceInfoOptions) error {
	var continueToken string
	var wg sync.WaitGroup
	errCh := make(chan error, 100) // 设置一个合理的缓冲区，防止阻塞

	// 控制最大并发数
	semaphore := make(chan struct{}, opts.Parallelism)
	if opts.Parallelism <= 0 {
		semaphore = nil // 不限并发数
	}

	for {
		crdList, newContinueToken, err := GetCRDList(opts.CRDClient, continueToken)
		if err != nil {
			return fmt.Errorf("无法查询 CRD 实例: %w", err)
		}

		for _, crd := range crdList.Items {
			nodeName := crd.GetName()

			if !node.CheckNodeExistsFromBatchList(nodeName, opts.BatchNodes) {
				wg.Add(1)
				go func(n string) {
					defer wg.Done()

					if semaphore != nil {
						semaphore <- struct{}{}        // 占用并发槽
						defer func() { <-semaphore }() // 释放并发槽
					}

					if err := DeleteCRD(opts.CRDClient, n); err != nil {
						log.Printf("无法删除 NodeResourceInfo CRD %s: %v", n, err)
						errCh <- fmt.Errorf("删除失败: %s, 错误: %w", n, err)
					} else {
						log.Printf("已删除 NodeResourceInfo CRD %s", n)
					}
				}(nodeName)
			}
		}

		if newContinueToken == "" {
			break
		}
		continueToken = newContinueToken
	}

	wg.Wait()
	close(errCh)

	// 使用 errors.Join 合并错误
	return errors.Join(<-errCh)
}
