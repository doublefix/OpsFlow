package main

import (
	"log"
	"net"

	"github.com/modcoco/OpsFlow/pkg/handler"
	pb "github.com/modcoco/OpsFlow/pkg/proto"

	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 1. 初始化 Kubernetes 客户端
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatalf("加载 kubeconfig 失败: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("创建 K8s 客户端失败: %v", err)
	}

	// 2. 启动 gRPC 服务
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}
	grpcServer := grpc.NewServer()

	// 3. 注入客户端并注册服务
	pb.RegisterPodLogServiceServer(grpcServer, handler.NewPodLogHandler(clientset))

	log.Println("gRPC 服务启动，监听端口 :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("服务失败: %v", err)
	}
}
