package svc

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GenerateRayClusterService(namespace, clusterName string) *corev1.Service {
	serviceName := fmt.Sprintf("%s-vllm-svc", clusterName)
	identifier := fmt.Sprintf("%s-head", clusterName)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/created-by": "kuberay-operator",
				"app.kubernetes.io/name":       "kuberay",
				"ray.io/cluster":               clusterName,
				"ray.io/identifier":            identifier,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "target-port",
					Protocol:   corev1.ProtocolTCP,
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/created-by": "kuberay-operator",
				"app.kubernetes.io/name":       "kuberay",
				"ray.io/cluster":               clusterName,
				"ray.io/identifier":            identifier,
				"ray.io/node-type":             "head",
			},
			PublishNotReadyAddresses: true,
		},
	}
}
