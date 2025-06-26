package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	pb "github.com/modcoco/OpsFlow/pkg/proto"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"google.golang.org/grpc"
)

func main() {
	// 初始化 Kubernetes 客户端
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Panicf("无法加载 kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Panicf("无法创建 Kubernetes 客户端: %v", err)
	}

	// 创建服务实例
	execService := &podExecService{
		clientset: clientset,
		config:    cfg,
	}

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPodExecServiceServer(grpcServer, execService)

	log.Println("gRPC server started on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}

// podExecService 实现 PodExecServiceServer 接口
type podExecService struct {
	pb.UnimplementedPodExecServiceServer
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func (s *podExecService) Exec(stream pb.PodExecService_ExecServer) error {
	// 接收第一个消息，必须是初始化消息
	first, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive first message: %w", err)
	}

	init := first.GetInit()
	if init == nil {
		return stream.Send(&pb.ExecMessage{
			Content: &pb.ExecMessage_Error{Error: "First message must be ExecInit"},
		})
	}

	// 创建管道用于数据流
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	ctx := stream.Context()
	done := make(chan error)

	// 启动执行器
	go func() {
		err := s.startExec(ctx, init, stdinR, stdoutW, stderrW)
		done <- err
	}()

	// 处理输入流
	go func() {
		defer stdinW.Close()
		for {
			in, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("Failed to receive message: %v", err)
				return
			}

			switch x := in.Content.(type) {
			case *pb.ExecMessage_Stdin:
				if _, err := stdinW.Write(x.Stdin); err != nil {
					log.Printf("Failed to write stdin: %v", err)
					return
				}
			case *pb.ExecMessage_Close:
				return
			case *pb.ExecMessage_Resize:
				// TODO: 处理终端大小调整
			}
		}
	}()

	// 处理输出流
	buf := make([]byte, 4096)
	go func() {
		defer stdoutR.Close()
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				if err := stream.Send(&pb.ExecMessage{
					Content: &pb.ExecMessage_Stdout{Stdout: buf[:n]},
				}); err != nil {
					log.Printf("Failed to send stdout: %v", err)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Failed to read stdout: %v", err)
				}
				return
			}
		}
	}()

	// 处理错误流
	go func() {
		defer stderrR.Close()
		for {
			n, err := stderrR.Read(buf)
			if n > 0 {
				if err := stream.Send(&pb.ExecMessage{
					Content: &pb.ExecMessage_Stderr{Stderr: buf[:n]},
				}); err != nil {
					log.Printf("Failed to send stderr: %v", err)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Failed to read stderr: %v", err)
				}
				return
			}
		}
	}()

	return <-done
}

// startExec 启动实际的命令执行
func (s *podExecService) startExec(ctx context.Context, init *pb.ExecInit, stdin io.Reader, stdout, stderr io.Writer) error {
	req := s.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(init.PodName).
		Namespace(init.Namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: init.ContainerName,
			Command:   init.Command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       init.Tty,
		}, scheme.ParameterCodec)

	wsExec, wsErr := remotecommand.NewWebSocketExecutor(s.config, "POST", req.URL().String())
	spdyExec, spdyErr := remotecommand.NewSPDYExecutor(s.config, "POST", req.URL())

	if wsErr != nil && spdyErr != nil {
		return fmt.Errorf("both WebSocket and SPDY executors failed: WebSocket: %v, SPDY: %v", wsErr, spdyErr)
	}

	if wsErr == nil && spdyErr == nil {
		exec, err := remotecommand.NewFallbackExecutor(wsExec, spdyExec, func(err error) bool {
			return err != nil
		})
		if err != nil {
			return fmt.Errorf("failed to create fallback executor: %w", err)
		}

		return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
			Tty:    init.Tty,
		})
	}

	var executor remotecommand.Executor
	if wsErr == nil {
		executor = wsExec
	} else {
		executor = spdyExec
	}

	return executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    init.Tty,
	})
}
