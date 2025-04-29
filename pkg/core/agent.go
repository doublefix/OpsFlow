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
	"google.golang.org/protobuf/types/known/structpb"
)

func RunAgent(conn *grpc.ClientConn, agentID string) error {
	client := pb.NewAgentServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect stream: %w", err)
	}

	// Send initial heartbeat (required)
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

	// Start heartbeat goroutine
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

	// Listen for server messages
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
	case *pb.AgentMessage_FunctionRequest:
		req := body.FunctionRequest
		log.Printf("received function request: id=%s function=%s", req.RequestId, req.FunctionName)
		go executeFunction(stream, req)

	case *pb.AgentMessage_CancelTask:
		cancelReq := body.CancelTask
		log.Printf("received cancel for request: id=%s", cancelReq.RequestId)
		// TODO: Implement cancellation logic for running functions

	default:
		log.Println("received unknown message")
	}
}

// executeFunction handles function execution and sends back the result
func executeFunction(stream pb.AgentService_ConnectClient, req *pb.FunctionRequest) {
	log.Printf("executing function %s (request_id: %s)", req.FunctionName, req.RequestId)

	// Simulate execution time
	sleepTime := time.Duration(rand.Intn(5)+1) * time.Second
	time.Sleep(sleepTime)

	// Prepare result
	result := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"execution_time": structpb.NewNumberValue(sleepTime.Seconds()),
			"function_name":  structpb.NewStringValue(req.FunctionName),
		},
	}

	// Send function result
	err := stream.Send(&pb.AgentMessage{
		Body: &pb.AgentMessage_FunctionResult{
			FunctionResult: &pb.FunctionResult{
				RequestId:    req.RequestId,
				Success:      true,
				Result:       result,
				ErrorMessage: "",
			},
		},
	})
	if err != nil {
		log.Printf("failed to send function result: %v", err)
	}
	log.Printf("finished function %s (request_id: %s)", req.FunctionName, req.RequestId)
}
