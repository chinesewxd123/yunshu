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
	flag.StringVar(&cfg.ServerURL, "server-url", "http://127.0.0.1:8080", "platform base url")
	flag.UintVar(&cfg.ProjectID, "project-id", 0, "project id")
	flag.UintVar(&cfg.ServerID, "server-id", 0, "server id")
	flag.UintVar(&cfg.LogSourceID, "log-source-id", 0, "log source id")
	flag.StringVar(&cfg.Token, "token", "", "long-lived agent ingest token")
	flag.StringVar(&cfg.RegisterSecret, "register-secret", "", "public agent register secret (agent-first mode when token is empty)")
	flag.StringVar(&cfg.Name, "name", "log-agent", "agent name")
	flag.StringVar(&cfg.Version, "version", "v0.1.0", "agent version")
	flag.StringVar(&cfg.SourceType, "source-type", "file", "log source type: file or journal")
	flag.StringVar(&cfg.Path, "path", "", "file path or systemd unit")
	flag.IntVar(&cfg.TailLines, "tail-lines", 200, "tail lines for startup")
	flag.IntVar(&cfg.BatchSize, "batch-size", 50, "lines per batch")
	flag.DurationVar(&cfg.FlushInterval, "flush-interval", 250*time.Millisecond, "batch flush interval")
	flag.DurationVar(&cfg.ResendAfter, "resend-after", 3*time.Second, "resend pending batch after duration")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug logs for agent collectors")
	flag.Parse()

	if err := agent.Run(context.Background(), cfg); err != nil {
		log.Fatal(err)
	}
}
