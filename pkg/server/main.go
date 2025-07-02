package server

import (
	"fmt"

	"github.com/modcoco/OpsFlow/pkg/handler"
	pb "github.com/modcoco/OpsFlow/pkg/proto"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

func SetupGRPCServer(kubeClient *kubernetes.Clientset) (*grpc.Server, error) {
	grpcSrv := grpc.NewServer()

	podExecHandler, err := handler.NewPodExecServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create PodExec handler: %w", err)
	}

	logHandler := handler.NewPodLogHandler(kubeClient)

	pb.RegisterPodExecServiceServer(grpcSrv, podExecHandler)
	pb.RegisterPodLogServiceServer(grpcSrv, logHandler)

	return grpcSrv, nil
}
