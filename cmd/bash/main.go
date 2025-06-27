package main

import (
	"context"
	"fmt"
	"io"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type streamHandler struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	resizeChan chan remotecommand.TerminalSize
}

func (h *streamHandler) Read(p []byte) (int, error) {
	if h.stdin == nil {
		return 0, nil
	}
	return h.stdin.Read(p)
}

func (h *streamHandler) Write(p []byte) (int, error) {
	if h.stdout == nil {
		return 0, nil
	}
	return h.stdout.Write(p)
}

func (h *streamHandler) Stderr() io.Writer {
	return h.stderr
}

func (h *streamHandler) Next() *remotecommand.TerminalSize {
	select {
	case size := <-h.resizeChan:
		return &size
	default:
		return nil
	}
}

func main() {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		fmt.Printf("无法加载kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("无法创建Kubernetes客户端: %v\n", err)
		os.Exit(1)
	}

	option := &v1.PodExecOptions{
		Container: "calico-node",
		Command:   []string{"bash"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name("calico-node-rcjsm").
		Namespace("kube-system").
		SubResource("exec").
		VersionedParams(option, scheme.ParameterCodec)

	fmt.Printf("请求URL: %s\n", req.URL())

	wsExec, wsErr := remotecommand.NewWebSocketExecutor(cfg, "POST", req.URL().String())
	spdyExec, spdyErr := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())

	if wsErr != nil || spdyErr != nil {
		fmt.Printf("初始化执行器失败: websocket: %v, spdy: %v\n", wsErr, spdyErr)
		os.Exit(1)
	}

	exec, err := remotecommand.NewFallbackExecutor(
		wsExec,
		spdyExec,
		func(err error) bool {
			return err != nil
		},
	)
	if err != nil {
		fmt.Printf("创建FallbackExecutor失败: %v\n", err)
		os.Exit(1)
	}

	handler := &streamHandler{
		stdin:      os.Stdin,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		resizeChan: make(chan remotecommand.TerminalSize),
	}

	ctx := context.Background()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               option.TTY,
	})
	if err != nil {
		fmt.Printf("执行远程命令失败: %v\n", err)
		os.Exit(1)
	}
}
