package common

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
)

func AddLabelToConfigMap(configMap *corev1.ConfigMap, labels map[string]string) {
	if configMap.Labels == nil {
		configMap.Labels = make(map[string]string)
	}
	maps.Copy(configMap.Labels, labels)
}
