package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "yunshu/internal/grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Doctor 检查 gRPC 连通性与 Token 有效性。
func Doctor(cfg Config) error {
	if err := cfg.normalize(); err != nil {
		return err
	}
	if cfg.ServerID == 0 {
		return fmt.Errorf("server-id is required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" && strings.TrimSpace(cfg.RegisterSecret) == "" {
		return fmt.Errorf("token or register-secret is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(cfg.GrpcServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("grpc dial failed: %w", err)
	}
	defer conn.Close()
	cli := pb.NewAgentRuntimeServiceClient(conn)

	if token == "" {
		pid, tk, err := publicRegister(ctx, cli, cfg.ServerID, cfg.Name, cfg.Version, strings.TrimSpace(cfg.RegisterSecret))
		if err != nil {
			return fmt.Errorf("public-register failed: %w", err)
		}
		// doctor 为 CLI 诊断工具：成功结果输出到 stdout 供运维直接查看。
		fmt.Printf("OK public-register project_id=%d token_len=%d\n", pid, len(tk))
		token = tk
	}

	bundle, err := fetchRuntimeConfig(ctx, cli, token)
	if err != nil {
		return fmt.Errorf("GetRuntimeConfig failed: %w", err)
	}
	fmt.Printf("OK runtime-config project_id=%d sources=%d discovery_roots=%d grpc=%s server_id=%d\n",
		bundle.ProjectID, len(bundle.Sources), len(bundle.Roots), cfg.GrpcServer, cfg.ServerID)
	for _, s := range bundle.Sources {
		fmt.Printf("  - source id=%d type=%s path=%s\n", s.LogSourceID, s.LogType, s.Path)
	}
	return nil
}
