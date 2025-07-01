package handler

import (
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/modcoco/OpsFlow/pkg/proto"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodLogHandler struct {
	pb.UnimplementedPodLogServiceServer
	client *kubernetes.Clientset
}

func NewPodLogHandler(client *kubernetes.Clientset) *PodLogHandler {
	return &PodLogHandler{client: client}
}

func (h *PodLogHandler) GetLogs(req *pb.LogRequest, stream pb.PodLogService_GetLogsServer) error {
	logOpts := &v1.PodLogOptions{
		Container:  req.Container,
		Follow:     req.Follow,
		Timestamps: req.Timestamps,
		Previous:   req.Previous,
	}

	if req.TailLines > 0 {
		logOpts.TailLines = &req.TailLines
	}
	if req.SinceTime != "" {
		t, err := time.Parse(time.RFC3339, req.SinceTime)
		if err == nil {
			logOpts.SinceTime = &metav1.Time{Time: t}
		}
	}

	// 使用 stream.Context() 替代 context.Background() 以感知客户端断开
	ctx := stream.Context()
	logReq := h.client.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, logOpts)

	readCloser, err := logReq.Stream(ctx)
	if err != nil {
		log.Printf("获取日志流失败: %v", err)
		return stream.Send(&pb.LogResponse{
			Log: &pb.LogResponse_Error{
				Error: fmt.Sprintf("日志流获取失败: %v", err),
			},
		})
	}
	defer readCloser.Close()

	// 启动 goroutine 监听客户端断开
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			log.Printf("客户端断开：%s/%s", req.Namespace, req.PodName)
			readCloser.Close()
		case <-done:
		}
	}()
	defer close(done)

	buf := make([]byte, 4096)
	for {
		n, err := readCloser.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&pb.LogResponse{
				Log: &pb.LogResponse_Content{Content: buf[:n]},
			}); sendErr != nil {
				return sendErr
			}
		}
		if err == io.EOF {
			return stream.Send(&pb.LogResponse{
				Log: &pb.LogResponse_Eof{Eof: true},
			})
		}
		if err != nil {
			return stream.Send(&pb.LogResponse{
				Log: &pb.LogResponse_Error{
					Error: fmt.Sprintf("读取日志出错: %v", err),
				},
			})
		}
	}
}
