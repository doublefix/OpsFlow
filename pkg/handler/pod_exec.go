package handler

import (
	"context"
	"fmt"
	"io"

	pb "github.com/modcoco/OpsFlow/pkg/proto"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type PodExecServer struct {
	pb.UnimplementedPodExecServiceServer
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewPodExecServer() (*PodExecServer, error) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &PodExecServer{
		clientset: clientset,
		config:    cfg,
	}, nil
}

func (s *PodExecServer) Exec(stream pb.PodExecService_ExecServer) error {
	// Get initial config
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive initial config: %w", err)
	}

	config := req.GetConfig()
	if config == nil {
		return fmt.Errorf("first message must contain config")
	}

	// Create the exec options
	option := &v1.PodExecOptions{
		Container: config.Container,
		Command:   config.Command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       config.Tty,
	}

	k8sReq := s.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(config.PodName).
		Namespace(config.Namespace).
		SubResource("exec").
		VersionedParams(option, scheme.ParameterCodec)

	wsExec, _ := remotecommand.NewWebSocketExecutor(s.config, "POST", k8sReq.URL().String())
	spdyExec, _ := remotecommand.NewSPDYExecutor(s.config, "POST", k8sReq.URL())

	exec, err := remotecommand.NewFallbackExecutor(wsExec, spdyExec, func(err error) bool {
		return err != nil
	})
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Setup pipe and context
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	resizeChan := make(chan remotecommand.TerminalSize, 1)

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// Handle input from client
	go func() {
		defer cancel() // cancel on exit
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				fmt.Printf("stream.Recv error: %v\n", err)
				return
			}

			switch {
			case req.GetStdin() != nil:
				_, _ = stdinWriter.Write(req.GetStdin())

			case req.GetResize() != nil:
				if resize := req.GetResize(); resize != nil {
					fmt.Printf("Received resize: %dx%d\n", resize.Width, resize.Height)
					size := remotecommand.TerminalSize{
						Width:  uint16(resize.Width),
						Height: uint16(resize.Height),
					}
					// 更简洁的处理方式
					select {
					case resizeChan <- size: // 如果能立即发送
					default: // 如果通道满
						<-resizeChan       // 丢弃一个旧值
						resizeChan <- size // 放入新值
					}
				}
			}
		}
	}()

	// Handle stdout
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutReader.Read(buf)
			if err != nil {
				return
			}
			if sendErr := stream.Send(&pb.ExecResponse{
				Output: &pb.ExecResponse_Stdout{Stdout: buf[:n]},
			}); sendErr != nil {
				fmt.Printf("stream.Send stdout error: %v\n", sendErr)
				cancel()
				return
			}
		}
	}()

	// Handle stderr
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderrReader.Read(buf)
			if err != nil {
				return
			}
			if sendErr := stream.Send(&pb.ExecResponse{
				Output: &pb.ExecResponse_Stderr{Stderr: buf[:n]},
			}); sendErr != nil {
				fmt.Printf("stream.Send stderr error: %v\n", sendErr)
				cancel()
				return
			}
		}
	}()

	handler := &streamHandler{
		stdin:      stdinReader,
		stdout:     stdoutWriter,
		stderr:     stderrWriter,
		resizeChan: resizeChan,
	}

	// Start streaming exec session
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               option.TTY,
	}); err != nil {
		return fmt.Errorf("stream exec failed: %w", err)
	}

	// Notify client that session is closed
	_ = stream.Send(&pb.ExecResponse{
		Output: &pb.ExecResponse_Closed{Closed: true},
	})

	return nil
}

// Implements io.Reader, io.Writer, remotecommand.TerminalSizeQueue
type streamHandler struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	resizeChan chan remotecommand.TerminalSize
}

func (h *streamHandler) Read(p []byte) (int, error) {
	return h.stdin.Read(p)
}

func (h *streamHandler) Write(p []byte) (int, error) {
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
