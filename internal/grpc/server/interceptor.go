package server

import (
	"context"

	logx "yunshu/internal/pkg/logger"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// unaryLogInterceptor 为每个 RPC 注入 request_id，并在失败时写统一 gRPC 日志。
func unaryLogInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	rid := logx.RequestIDFromContext(ctx)
	if rid == "" {
		rid = uuid.NewString()
		ctx = logx.WithRequestID(ctx, rid)
	}
	resp, err := handler(ctx, req)
	if err != nil {
		logGRPCError(ctx, info.FullMethod, err)
	}
	return resp, err
}
