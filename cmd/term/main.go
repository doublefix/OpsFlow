package main

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func main() {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		fmt.Println("无法加载 kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Println("无法创建 Kubernetes 客户端: %v", err)
	}

	option := &v1.PodExecOptions{
		Container: "calico-node",
		Command:   []string{"bash"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name("calico-node-rcjsm").
		Namespace("kube-system").
		SubResource("exec").
		VersionedParams(option, scheme.ParameterCodec)

	fmt.Printf("请求 URL: %s\n", req.URL())

	wsExec, wsErr := remotecommand.NewWebSocketExecutor(cfg, "POST", req.URL().String())
	spdyExec, spdyErr := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())

	if wsErr != nil || spdyErr != nil {
		fmt.Println("初始化执行器失败: websocket: %v, spdy: %v", wsErr, spdyErr)
	}

	exec, err := remotecommand.NewFallbackExecutor(
		wsExec,
		spdyExec,
		func(err error) bool {
			return err != nil
		},
	)
	if err != nil {
		fmt.Println("创建 FallbackExecutor 失败: %v", err)
	}

	ctx := context.Background()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		fmt.Println("执行远程命令失败: %v", err)
	}
}
