package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"

	pb "github.com/modcoco/OpsFlow/pkg/proto"

	"google.golang.org/grpc"
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
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	return &PodExecServer{
		clientset: clientset,
		config:    cfg,
	}, nil
}

func (s *PodExecServer) Exec(stream pb.PodExecService_ExecServer) error {
	// First message must be the config
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive initial config: %v", err)
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

	// Create the request
	k8sReq := s.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(config.PodName).
		Namespace(config.Namespace).
		SubResource("exec").
		VersionedParams(option, scheme.ParameterCodec)

	// Create executors
	wsExec, _ := remotecommand.NewWebSocketExecutor(s.config, "POST", k8sReq.URL().String())
	spdyExec, _ := remotecommand.NewSPDYExecutor(s.config, "POST", k8sReq.URL())

	exec, err := remotecommand.NewFallbackExecutor(
		wsExec,
		spdyExec,
		func(err error) bool { return err != nil },
	)
	if err != nil {
		return fmt.Errorf("failed to create executor: %v", err)
	}

	// Create pipes for communication
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	resizeChan := make(chan remotecommand.TerminalSize)

	// Stream handling
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// Handle incoming messages from gRPC client
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				cancel()
				return
			}

			if stdinData := req.GetStdin(); stdinData != nil {
				stdinWriter.Write(stdinData)
			} else if resize := req.GetResize(); resize != nil {
				resizeChan <- remotecommand.TerminalSize{
					Width:  uint16(resize.Width),
					Height: uint16(resize.Height),
				}
			}
		}
	}()

	// Handle outgoing messages to gRPC client
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutReader.Read(buf)
			if err != nil {
				return
			}
			stream.Send(&pb.ExecResponse{
				Output: &pb.ExecResponse_Stdout{Stdout: buf[:n]},
			})
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderrReader.Read(buf)
			if err != nil {
				return
			}
			stream.Send(&pb.ExecResponse{
				Output: &pb.ExecResponse_Stderr{Stderr: buf[:n]},
			})
		}
	}()

	// Create the stream handler
	handler := &streamHandler{
		stdin:      stdinReader,
		stdout:     stdoutWriter,
		stderr:     stderrWriter,
		resizeChan: resizeChan,
	}

	// Start the exec session
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               option.TTY,
	})

	// Send closed message
	stream.Send(&pb.ExecResponse{
		Output: &pb.ExecResponse_Closed{Closed: true},
	})

	return err
}

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

func main() {
	server, err := NewPodExecServer()
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPodExecServiceServer(grpcServer, server)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
		os.Exit(1)
	}
}
