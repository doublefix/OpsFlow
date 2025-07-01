package main

import (
	"fmt"
	"net"
	"os"

	"github.com/modcoco/OpsFlow/pkg/handler"
	pb "github.com/modcoco/OpsFlow/pkg/proto"

	"google.golang.org/grpc"
)

func main() {
	server, err := handler.NewPodExecServer()
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
