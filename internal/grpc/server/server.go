package server

import (
	"context"
	"net"

	pb "yunshu/internal/grpc/proto"

	"google.golang.org/grpc"
)

type RuntimeServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func Start(addr string, impl *LogPlatformServer, maxRecvBytes, maxSendBytes int) (*RuntimeServer, error) {
	opts := make([]grpc.ServerOption, 0, 2)
	if maxRecvBytes > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(maxRecvBytes))
	}
	if maxSendBytes > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(maxSendBytes))
	}
	s := grpc.NewServer(opts...)
	pb.RegisterProjectServerServiceServer(s, impl)
	pb.RegisterLogSourceServiceServer(s, impl)
	pb.RegisterAgentRuntimeServiceServer(s, impl)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		_ = s.Serve(lis)
	}()
	return &RuntimeServer{grpcServer: s, listener: lis}, nil
}

func (s *RuntimeServer) Stop(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-ctx.Done():
		s.grpcServer.Stop()
	case <-done:
	}
}
