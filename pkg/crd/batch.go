package crd

import (
	"log"

	"github.com/modcoco/OpsFlow/pkg/node"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
)

// DeleteNonExistingNodeResourceInfo 删除不存在的 NodeResourceInfo CRD 实例
func DeleteNonExistingNodeResourceInfo(crdClient dynamic.ResourceInterface, batchNodesList *corev1.NodeList) {
	var continueToken string
	for {
		crdList, newContinueToken, err := GetCRDList(crdClient, continueToken)
		if err != nil {
			log.Fatalf("无法查询 CRD 实例: %v", err)
		}

		for _, crd := range crdList.Items {
			nodeName := crd.GetName()

			if !node.CheckNodeExistsFromBatchList(nodeName, batchNodesList) {
				err := DeleteCRD(crdClient, nodeName)
				if err != nil {
					log.Printf("无法删除 NodeResourceInfo CRD %s: %v", nodeName, err)
				} else {
					log.Printf("已删除 NodeResourceInfo CRD %s", nodeName)
				}
			}
		}

		if newContinueToken == "" {
			break
		}
		continueToken = newContinueToken
	}
}
