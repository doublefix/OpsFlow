package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/modcoco/OpsFlow/pkg/proto" // 替换为你实际 proto 生成路径
	"google.golang.org/grpc"
)

func main() {
	// 1. 建立 gRPC 连接
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure()) // 或使用 TLS
	if err != nil {
		log.Fatalf("无法连接 gRPC 服务: %v", err)
	}
	defer conn.Close()

	client := pb.NewPodLogServiceClient(conn)

	// 2. 构造请求
	req := &pb.LogRequest{
		Namespace:  "kube-system",
		PodName:    "calico-node-rcjsm",
		Container:  "calico-node",
		Follow:     true,  // 是否实时跟随
		Timestamps: false, // 是否带时间戳
		TailLines:  10,    // 最近 10 行日志
		Previous:   false, // 是否获取上一个容器
		// SinceTime: "2024-07-01T00:00:00Z", // 可选
	}

	// 3. 发起日志流请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := client.GetLogs(ctx, req)
	if err != nil {
		log.Fatalf("日志请求失败: %v", err)
	}

	// 4. 读取服务端流返回的日志
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Println("日志读取完毕")
			break
		}
		if err != nil {
			log.Fatalf("读取日志流失败: %v", err)
		}

		switch x := resp.Log.(type) {
		case *pb.LogResponse_Content:
			fmt.Print(string(x.Content)) // 直接输出到 stdout
		case *pb.LogResponse_Error:
			log.Printf("服务端错误: %s\n", x.Error)
		case *pb.LogResponse_Eof:
			log.Println("服务端发送 EOF")
			return
		default:
			log.Println("未知日志响应")
		}
	}
}
