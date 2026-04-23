package cmd

import (
	"context"
	"time"
	"yunshu/internal/agent"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logAgentCmd)
	logAgentCmd.Flags().StringVar(&agentServerURL, "grpc-server", "127.0.0.1:18080", "platform grpc server address")
	logAgentCmd.Flags().StringVar(&agentPlatformURL, "platform-url", "", "platform http address for health report, e.g. http://10.10.10.10:8080")
	logAgentCmd.Flags().UintVar(&agentProjectID, "project-id", 0, "project id")
	logAgentCmd.Flags().UintVar(&agentServerID, "server-id", 0, "server id")
	logAgentCmd.Flags().UintVar(&agentLogSourceID, "log-source-id", 0, "log source id")
	logAgentCmd.Flags().StringVar(&agentToken, "token", "", "long-lived agent ingest token")
	logAgentCmd.Flags().StringVar(&agentRegisterSecret, "register-secret", "", "public agent register secret (agent-first mode when token is empty)")
	logAgentCmd.Flags().StringVar(&agentName, "name", "log-agent", "agent name")
	logAgentCmd.Flags().StringVar(&agentVersion, "version", agent.DefaultVersion, "agent version")
	logAgentCmd.Flags().StringVar(&agentSourceType, "source-type", "file", "log source type: file or journal")
	logAgentCmd.Flags().StringVar(&agentPath, "path", "", "file path or systemd unit")
	logAgentCmd.Flags().IntVar(&agentTailLines, "tail-lines", 200, "tail lines for startup")
	logAgentCmd.Flags().IntVar(&agentBatchSize, "batch-size", 50, "lines per batch")
	logAgentCmd.Flags().DurationVar(&agentFlushInterval, "flush-interval", 250*time.Millisecond, "batch flush interval")
	logAgentCmd.Flags().DurationVar(&agentResendAfter, "resend-after", 3*time.Second, "resend pending batch after duration")
	logAgentCmd.Flags().BoolVar(&agentDebug, "debug", false, "enable debug logs for agent collectors")
	logAgentCmd.Flags().IntVar(&agentListenPort, "listen-port", 0, "本机监听端口（0=不监听；仅上报展示，当前 Agent 为出站 gRPC）")
	logAgentCmd.Flags().BoolVar(&agentEnableRuntimePull, "enable-runtime-pull", true, "enable pulling runtime config from platform")
	logAgentCmd.Flags().BoolVar(&agentEnableFallback, "enable-fallback", false, "enable fallback single-source mode when runtime config unavailable")
	logAgentCmd.Flags().BoolVar(&agentEnableDiscovery, "enable-discovery", true, "enable discovery scan and report")
	logAgentCmd.Flags().BoolVar(&agentEnableHealth, "enable-health-report", true, "enable health report to platform")
}

var (
	agentServerURL         string
	agentPlatformURL       string
	agentProjectID         uint
	agentServerID          uint
	agentLogSourceID       uint
	agentToken             string
	agentRegisterSecret    string
	agentName              string
	agentVersion           string
	agentSourceType        string
	agentPath              string
	agentTailLines         int
	agentBatchSize         int
	agentFlushInterval     time.Duration
	agentResendAfter       time.Duration
	agentDebug             bool
	agentListenPort        int
	agentEnableRuntimePull bool
	agentEnableFallback    bool
	agentEnableDiscovery   bool
	agentEnableHealth      bool
)

var logAgentCmd = &cobra.Command{
	Use:   "log-agent",
	Short: "Run lightweight local log collection agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.Run(context.Background(), agent.Config{
			GrpcServer:        agentServerURL,
			PlatformURL:       agentPlatformURL,
			ProjectID:         agentProjectID,
			ServerID:          agentServerID,
			LogSourceID:       agentLogSourceID,
			Token:             agentToken,
			RegisterSecret:    agentRegisterSecret,
			Name:              agentName,
			Version:           agentVersion,
			SourceType:        agentSourceType,
			Path:              agentPath,
			TailLines:         agentTailLines,
			BatchSize:         agentBatchSize,
			FlushInterval:     agentFlushInterval,
			ResendAfter:       agentResendAfter,
			Debug:             agentDebug,
			ListenPort:        agentListenPort,
			EnableRuntimePull: agentEnableRuntimePull,
			EnableFallback:    agentEnableFallback,
			EnableDiscovery:   agentEnableDiscovery,
			EnableHealth:      agentEnableHealth,
		})
	},
}
