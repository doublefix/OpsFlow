package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"

	pb "github.com/modcoco/OpsFlow/pkg/apis/proto"
	"google.golang.org/grpc"
)

const (
	serverAddr = "localhost:50051"
	agentID    = "agent-001"
)

func RunAgent(conn *grpc.ClientConn) error {
	client := pb.NewAgentServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect stream: %w", err)
	}

	// 发送第一次 heartbeat（必须的）
	if err := stream.Send(&pb.AgentMessage{
		Body: &pb.AgentMessage_Heartbeat{
			Heartbeat: &pb.Heartbeat{
				AgentId:   agentID,
				Timestamp: time.Now().Unix(),
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send initial heartbeat: %w", err)
	}

	// 启动心跳 goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := stream.Send(&pb.AgentMessage{
					Body: &pb.AgentMessage_Heartbeat{
						Heartbeat: &pb.Heartbeat{
							AgentId:   agentID,
							Timestamp: time.Now().Unix(),
						},
					},
				})
				if err != nil {
					log.Printf("failed to send heartbeat: %v", err)
					cancel()
					return
				}
				log.Println("sent heartbeat")
			}
		}
	}()

	// 监听服务器下发的任务
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Println("server closed stream")
			return nil
		}
		if err != nil {
			log.Printf("stream recv error: %v", err)
			return err
		}
		handleMessage(stream, in)
	}
}

func handleMessage(stream pb.AgentService_ConnectClient, msg *pb.AgentMessage) {
	switch body := msg.Body.(type) {
	case *pb.AgentMessage_TaskRequest:
		task := body.TaskRequest
		log.Printf("received task: id=%s command=%s", task.TaskId, task.Command)
		go executeTask(stream, task)

	case *pb.AgentMessage_CancelTask:
		cancelTask := body.CancelTask
		log.Printf("received cancel for task: id=%s", cancelTask.TaskId)
		// TODO: 这里可以记录取消的任务ID，实际执行的时候检查是否要终止

	default:
		log.Println("received unknown message")
	}
}

// 简单模拟执行任务
func executeTask(stream pb.AgentService_ConnectClient, task *pb.TaskRequest) {
	log.Printf("executing task %s: %s", task.TaskId, task.Command)

	// 模拟执行时间
	sleepTime := time.Duration(rand.Intn(5)+1) * time.Second
	time.Sleep(sleepTime)

	// 发送任务结果
	err := stream.Send(&pb.AgentMessage{
		Body: &pb.AgentMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId: task.TaskId,
				Status: "success",
				Output: fmt.Sprintf("executed command: %s", task.Command),
				Error:  "",
			},
		},
	})
	if err != nil {
		log.Printf("failed to send task result: %v", err)
	}
	log.Printf("finished task %s", task.TaskId)
}
