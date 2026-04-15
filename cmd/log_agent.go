package cmd

import (
	"context"
	"go-permission-system/internal/agent"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logAgentCmd)
	logAgentCmd.Flags().StringVar(&agentServerURL, "server-url", "http://127.0.0.1:8080", "platform base url")
	logAgentCmd.Flags().UintVar(&agentProjectID, "project-id", 0, "project id")
	logAgentCmd.Flags().UintVar(&agentServerID, "server-id", 0, "server id")
	logAgentCmd.Flags().UintVar(&agentLogSourceID, "log-source-id", 0, "log source id")
	logAgentCmd.Flags().StringVar(&agentToken, "token", "", "long-lived agent ingest token")
	logAgentCmd.Flags().StringVar(&agentRegisterSecret, "register-secret", "", "public agent register secret (agent-first mode when token is empty)")
	logAgentCmd.Flags().StringVar(&agentName, "name", "log-agent", "agent name")
	logAgentCmd.Flags().StringVar(&agentVersion, "version", "v0.1.0", "agent version")
	logAgentCmd.Flags().StringVar(&agentSourceType, "source-type", "file", "log source type: file or journal")
	logAgentCmd.Flags().StringVar(&agentPath, "path", "", "file path or systemd unit")
	logAgentCmd.Flags().IntVar(&agentTailLines, "tail-lines", 200, "tail lines for startup")
	logAgentCmd.Flags().IntVar(&agentBatchSize, "batch-size", 50, "lines per batch")
	logAgentCmd.Flags().DurationVar(&agentFlushInterval, "flush-interval", 250*time.Millisecond, "batch flush interval")
	logAgentCmd.Flags().DurationVar(&agentResendAfter, "resend-after", 3*time.Second, "resend pending batch after duration")
	logAgentCmd.Flags().BoolVar(&agentDebug, "debug", false, "enable debug logs for agent collectors")
}

var (
	agentServerURL      string
	agentProjectID      uint
	agentServerID       uint
	agentLogSourceID    uint
	agentToken          string
	agentRegisterSecret string
	agentName           string
	agentVersion        string
	agentSourceType     string
	agentPath           string
	agentTailLines      int
	agentBatchSize      int
	agentFlushInterval  time.Duration
	agentResendAfter    time.Duration
	agentDebug          bool
)

var logAgentCmd = &cobra.Command{
	Use:   "log-agent",
	Short: "Run lightweight local log collection agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.Run(context.Background(), agent.Config{
			ServerURL:      agentServerURL,
			ProjectID:      agentProjectID,
			ServerID:       agentServerID,
			LogSourceID:    agentLogSourceID,
			Token:          agentToken,
			RegisterSecret: agentRegisterSecret,
			Name:           agentName,
			Version:        agentVersion,
			SourceType:     agentSourceType,
			Path:           agentPath,
			TailLines:      agentTailLines,
			BatchSize:      agentBatchSize,
			FlushInterval:  agentFlushInterval,
			ResendAfter:    agentResendAfter,
			Debug:          agentDebug,
		})
	},
}
