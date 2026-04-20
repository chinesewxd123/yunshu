// logagent 独立二进制：运行节点上的日志采集 Agent，通过 gRPC 与平台通信。
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"go-permission-system/internal/agent"
)

func main() {
	cfg := agent.Config{}
	flag.StringVar(&cfg.GrpcServer, "grpc-server", "127.0.0.1:18080", "platform grpc server address")
	flag.StringVar(&cfg.PlatformURL, "platform-url", "", "platform http address for health report, e.g. http://10.10.10.10:8080")
	flag.UintVar(&cfg.ProjectID, "project-id", 0, "project id")
	flag.UintVar(&cfg.ServerID, "server-id", 0, "server id")
	flag.UintVar(&cfg.LogSourceID, "log-source-id", 0, "log source id")
	flag.StringVar(&cfg.Token, "token", "", "long-lived agent ingest token")
	flag.StringVar(&cfg.RegisterSecret, "register-secret", "", "public agent register secret (agent-first mode when token is empty)")
	flag.StringVar(&cfg.Name, "name", "log-agent", "agent name")
	flag.StringVar(&cfg.Version, "version", agent.DefaultVersion, "agent version")
	flag.StringVar(&cfg.SourceType, "source-type", "file", "log source type: file or journal")
	flag.StringVar(&cfg.Path, "path", "", "file path or systemd unit")
	flag.IntVar(&cfg.TailLines, "tail-lines", 200, "tail lines for startup")
	flag.IntVar(&cfg.BatchSize, "batch-size", 50, "lines per batch")
	flag.DurationVar(&cfg.FlushInterval, "flush-interval", 250*time.Millisecond, "batch flush interval")
	flag.DurationVar(&cfg.ResendAfter, "resend-after", 3*time.Second, "resend pending batch after duration")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug logs for agent collectors")
	flag.IntVar(&cfg.ListenPort, "listen-port", 0, "本机监听端口（0=不监听；仅上报给平台展示，当前 Agent 为出站 gRPC 客户端）")
	flag.BoolVar(&cfg.EnableRuntimePull, "enable-runtime-pull", true, "enable pulling runtime config from platform")
	flag.BoolVar(&cfg.EnableFallback, "enable-fallback", false, "enable fallback single-source mode when runtime config unavailable")
	flag.BoolVar(&cfg.EnableDiscovery, "enable-discovery", true, "enable discovery scan and report")
	flag.BoolVar(&cfg.EnableHealth, "enable-health-report", true, "enable health report to platform")
	flag.Parse()

	if err := agent.Run(context.Background(), cfg); err != nil {
		log.Fatal(err)
	}
}
