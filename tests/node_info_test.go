package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func TestGetNodeResources(t *testing.T) {
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
		// 获取总资源（Capacity）
		totalCPU := node.Status.Capacity["cpu"]
		totalMemory := node.Status.Capacity["memory"]
		totalGPU := node.Status.Capacity["nvidia.com/gpu"]

		// 获取可分配资源（Allocatable）
		allocatableCPU := node.Status.Allocatable["cpu"]
		allocatableMemory := node.Status.Allocatable["memory"]
		allocatableGPU := node.Status.Allocatable["nvidia.com/gpu"]

		// 计算已分配资源（Allocated）
		var usedCPU, usedMemory, usedGPU resource.Quantity
		for _, pod := range pods.Items {
			if pod.Spec.NodeName == node.Name {
				for _, container := range pod.Spec.Containers {
					usedCPU.Add(*container.Resources.Requests.Cpu())
					usedMemory.Add(*container.Resources.Requests.Memory())
					if gpu, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
						usedGPU.Add(gpu)
					}
				}
			}
		}

		fmt.Printf("Node: %s\n", node.Name)
		status := GetNodeStatus(&node)
		roles := GetNodeRoles(&node)
		kubeletVersion := GetKubeletVersion(&node)
		internalIP := GetInternalIP(&node)
		kernelVersion := GetKernelVersion(&node)
		containerRuntimeVersion := GetContainerRuntimeVersion(&node)
		OSImage := GetOSImage(&node)

		fmt.Println("STATUS:", status)
		fmt.Println("ROLES:", roles)
		fmt.Println("VERSION:", kubeletVersion)
		fmt.Println("InternalIP:", internalIP)
		fmt.Println("KernelVersion:", kernelVersion)
		fmt.Println("ContainerRuntimeVersion:", containerRuntimeVersion)
		fmt.Println("OSImage:", OSImage)

		fmt.Println("--------------------------------------------------")

		fmt.Printf("  [CPU]\n")
		fmt.Printf("    总资源: %d 核 (%d mCPU)\n", totalCPU.Value(), totalCPU.MilliValue())
		fmt.Printf("    已分配: %d 核 (%d mCPU)\n", usedCPU.Value(), usedCPU.MilliValue())
		fmt.Printf("    可分配: %d 核 (%d mCPU)\n", allocatableCPU.Value(), allocatableCPU.MilliValue())

		// Memory 信息
		fmt.Printf("  [Memory]\n")
		fmt.Printf("    总资源: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(totalMemory, resource.Kilo), utils.ScaledValue(totalMemory, resource.Mega), utils.ScaledValue(totalMemory, resource.Giga))
		fmt.Printf("    已分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(usedMemory, resource.Kilo), utils.ScaledValue(usedMemory, resource.Mega), utils.ScaledValue(usedMemory, resource.Giga))
		fmt.Printf("    可分配: %d KiB (%d MiB, %d GiB)\n", utils.ScaledValue(allocatableMemory, resource.Kilo), utils.ScaledValue(allocatableMemory, resource.Mega), utils.ScaledValue(allocatableMemory, resource.Giga))

		// GPU 信息
		fmt.Printf("  [GPU]\n")
		fmt.Printf("    总资源: %d\n", totalGPU.Value())
		fmt.Printf("    已分配: %d\n", usedGPU.Value())
		fmt.Printf("    可分配: %d\n", allocatableGPU.Value())

	}
}

func TestGetNamespaceUID(t *testing.T) {
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

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(namespace.GetUID())
}

func GetNodeStatus(node *v1.Node) string {
	var statuses []string

	for _, condition := range node.Status.Conditions {
		if condition.Status == v1.ConditionTrue {
			statuses = append(statuses, string(condition.Type))
		}
	}

	if node.Spec.Unschedulable {
		statuses = append(statuses, "SchedulingDisabled")
	}

	return strings.Join(statuses, ",")
}

func GetNodeRoles(node *v1.Node) string {
	const roleLabelPrefix = "node-role.kubernetes.io/"
	var roles []string

	for label := range node.Labels {
		if strings.HasPrefix(label, roleLabelPrefix) {
			rolePart := strings.TrimPrefix(label, roleLabelPrefix)
			role := strings.TrimSuffix(rolePart, "=")
			roles = append(roles, role)
		}
	}

	return strings.Join(roles, ",")
}

func GetKubeletVersion(node *v1.Node) string {
	return node.Status.NodeInfo.KubeletVersion
}

func GetOSImage(node *v1.Node) string {
	return node.Status.NodeInfo.OSImage
}

func GetKernelVersion(node *v1.Node) string {
	return node.Status.NodeInfo.KernelVersion
}

func GetContainerRuntimeVersion(node *v1.Node) string {
	return node.Status.NodeInfo.ContainerRuntimeVersion
}

func GetInternalIP(node *v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func TestSSH(t *testing.T) {
	// 加载 kubeconfig
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		t.Fatalf("无法加载 kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("无法创建 Kubernetes 客户端: %v", err)
	}

	option := &v1.PodExecOptions{
		Container: "calico-node",
		Command:   []string{"bash", "-c", "pwd"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
	}

	// 构造 exec 请求
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name("calico-node-rcjsm").
		Namespace("kube-system").
		SubResource("exec").
		VersionedParams(option, scheme.ParameterCodec)

	fmt.Printf("请求 URL: %s\n", req.URL())

	exec, err := remotecommand.NewWebSocketExecutor(cfg, "POST", req.URL().String())
	if err != nil {
		t.Fatalf("创建 executor 失败: %v", err)
	}

	// exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// 执行远程命令并连接本地标准输入输出
	ctx := context.Background()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stdout,
		Tty:    false,
	})
	if err != nil {
		t.Fatalf("执行远程命令失败: %v", err)
	}
}
