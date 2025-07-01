package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/modcoco/OpsFlow/pkg/proto"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	clientConn, err := grpc.NewClient("ubuntu:30969", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer clientConn.Close()

	client := pb.NewPodExecServiceClient(clientConn)
	stream, err := client.Exec(context.Background())
	if err != nil {
		fmt.Printf("Failed to create stream: %v\n", err)
		os.Exit(1)
	}

	// Send initial config
	err = stream.Send(&pb.ExecRequest{
		Input: &pb.ExecRequest_Config{
			Config: &pb.ExecConfig{
				Namespace: "kube-system",
				PodName:   "calico-node-rcjsm",
				Container: "calico-node",
				Command:   []string{"bash"},
				Tty:       true,
			},
		},
	})
	if err != nil {
		fmt.Printf("Failed to send config: %v\n", err)
		os.Exit(1)
	}

	// Handle terminal resize
	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		// Send initial size
		if width, height, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
			stream.Send(&pb.ExecRequest{
				Input: &pb.ExecRequest_Resize{
					Resize: &pb.TerminalSize{
						Width:  uint32(width),
						Height: uint32(height),
					},
				},
			})
		}

		// Watch for resize
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGWINCH)
		go func() {
			for range sigChan {
				if width, height, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
					stream.Send(&pb.ExecRequest{
						Input: &pb.ExecRequest_Resize{
							Resize: &pb.TerminalSize{
								Width:  uint32(width),
								Height: uint32(height),
							},
						},
					})
				}
			}
		}()
	}

	// Handle stdin
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			stream.Send(&pb.ExecRequest{
				Input: &pb.ExecRequest_Stdin{Stdin: buf[:n]},
			})
		}
	}()

	// Handle stdout/stderr
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return
		}

		if stdout := resp.GetStdout(); stdout != nil {
			os.Stdout.Write(stdout)
		} else if stderr := resp.GetStderr(); stderr != nil {
			os.Stderr.Write(stderr)
		} else if closed := resp.GetClosed(); closed {
			return
		}
	}
}
