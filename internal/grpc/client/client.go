package client

import (
	"context"
	"time"

	pb "yunshu/internal/grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RuntimeClient struct {
	conn         *grpc.ClientConn
	ProjectSrv   pb.ProjectServerServiceClient
	LogSourceSrv pb.LogSourceServiceClient
	AgentSrv     pb.AgentRuntimeServiceClient
}

func Dial(addr string, timeout time.Duration, maxRecvBytes, maxSendBytes int) (*RuntimeClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if maxRecvBytes > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvBytes)))
	}
	if maxSendBytes > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxSendBytes)))
	}
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, err
	}
	return &RuntimeClient{
		conn:         conn,
		ProjectSrv:   pb.NewProjectServerServiceClient(conn),
		LogSourceSrv: pb.NewLogSourceServiceClient(conn),
		AgentSrv:     pb.NewAgentRuntimeServiceClient(conn),
	}, nil
}

func (c *RuntimeClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
