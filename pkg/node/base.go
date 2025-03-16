package node

import (
	corev1 "k8s.io/api/core/v1"
)

func CheckNodeExistsFromBatchList(nodeName string, batchNodesList *corev1.NodeList) bool {
	for _, node := range batchNodesList.Items {
		if node.Name == nodeName {
			return true
		}
	}
	return false
}
