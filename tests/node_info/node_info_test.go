package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGetNodeResources(t *testing.T) {
	// 指定需要统计的资源名称
	resourceNamesToTrack := map[string]bool{
		"cpu":            true, // 统计 CPU
		"memory":         true, // 统计内存
		"nvidia.com/gpu": true, // 统计 GPU
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatalf("无法加载 kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("无法创建 Kubernetes 客户端: %v", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("无法获取节点列表: %v", err)
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("无法获取 Pod 列表: %v", err)
	}

	for _, node := range nodes.Items {
		fmt.Printf("Node: %s\n", node.Name)
		fmt.Println("--------------------------------------------------")

		// 获取所有资源类型
		for resourceName, totalResource := range node.Status.Capacity {
			// 如果资源不在指定列表中，则跳过
			if !resourceNamesToTrack[string(resourceName)] {
				continue
			}

			allocatableResource := node.Status.Allocatable[resourceName]

			// 计算已分配资源
			var usedResource resource.Quantity
			for _, pod := range pods.Items {
				if pod.Spec.NodeName == node.Name {
					for _, container := range pod.Spec.Containers {
						if request, ok := container.Resources.Requests[resourceName]; ok {
							usedResource.Add(request)
						}
					}
				}
			}

			fmt.Printf("ResourceName => %s \n", resourceName)
			if resourceName == "cpu" {
				fmt.Printf("    总资源: %d 核 (%d mCPU)\n", totalResource.Value(), totalResource.MilliValue())
				fmt.Printf("    已分配: %d 核 (%d mCPU)\n", usedResource.Value(), usedResource.MilliValue())
				fmt.Printf("    可分配: %d 核 (%d mCPU)\n", allocatableResource.Value(), allocatableResource.MilliValue())
			} else if resourceName == "memory" {
				fmt.Printf("    总资源: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(totalResource, resource.Kilo), utils.ScaledValue(totalResource, resource.Mega), utils.ScaledValue(totalResource, resource.Giga))
				fmt.Printf("    已分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(usedResource, resource.Kilo), utils.ScaledValue(usedResource, resource.Mega), utils.ScaledValue(usedResource, resource.Giga))
				fmt.Printf("    可分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(allocatableResource, resource.Kilo), utils.ScaledValue(allocatableResource, resource.Mega), utils.ScaledValue(allocatableResource, resource.Giga))
			} else {
				fmt.Printf("  [%s]\n", resourceName)
				fmt.Printf("    总资源: %s\n", totalResource.String())
				fmt.Printf("    已分配: %s\n", usedResource.String())
				fmt.Printf("    可分配: %s\n", allocatableResource.String())
			}
		}
	}
}

func TestBuildDeployment(t *testing.T) {
	// curl -X DELETE "http://localhost:8090/api/v1/deployments/default/nginx-deployment"
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.14.2",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(80),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}

	// 2. 创建 Kubernetes JSON 序列化器
	scheme := runtime.NewScheme()
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory, // 元数据工厂
		scheme,                  // ObjectCreater
		scheme,                  // ObjectTyper
		json.SerializerOptions{
			Yaml:   false, // 生成 JSON
			Pretty: true,  // 美化输出
			Strict: true,  // 严格模式
		},
	)

	var buf bytes.Buffer
	if err := serializer.Encode(deployment, &buf); err != nil {
		t.Fatalf("Failed to serialize deployment: %v", err)
	}
	jsonOutput := buf.String()
	t.Logf("Serialized Deployment JSON:\n%s", jsonOutput)

	// 反序列化
	decodedObj, _, err := serializer.Decode(buf.Bytes(), nil, nil)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	decodedDeployment, ok := decodedObj.(*appsv1.Deployment)
	if !ok {
		t.Fatalf("Decoded object is not a Deployment")
	}

	// 验证反序列化后的对象
	if decodedDeployment.Name != "nginx-deployment" {
		t.Errorf("Expected name 'nginx-deployment', got '%s'", decodedDeployment.Name)
	}
	if *decodedDeployment.Spec.Replicas != 3 {
		t.Errorf("Expected 3 replicas, got %d", *decodedDeployment.Spec.Replicas)
	}

	fmt.Println(decodedDeployment.ObjectMeta.Namespace)

	jsonData := buf.Bytes()

	// 创建 HTTP 请求
	apiURL := "http://ubuntu:30968/api/v1/deployments"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	t.Logf("Response status: %s", resp.Status)
	t.Logf("Response body:\n%s", string(body))
}

// 辅助函数：创建 int32 指针
func int32Ptr(i int32) *int32 { return &i }
