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
	callTimeout  time.Duration
}

func Dial(addr string, dialTimeout time.Duration, maxRecvBytes, maxSendBytes int, callTimeout time.Duration) (*RuntimeClient, error) {
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	if callTimeout <= 0 {
		callTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(unaryTimeoutInterceptor(callTimeout)),
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
		callTimeout:  callTimeout,
	}, nil
}

func unaryTimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if _, ok := ctx.Deadline(); !ok && timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (c *RuntimeClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
