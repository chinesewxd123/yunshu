package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "yunshu/internal/grpc/proto"
)

// runAgentSupervisor 在 ingest 断线或平台配置变更时自动重启采集会话。
func runAgentSupervisor(ctx context.Context, client pb.AgentRuntimeServiceClient, cfg Config, projectID uint, token string, initial []runtimeSource, initialRoots []string) error {
	sources := append([]runtimeSource(nil), initial...)
	runtimeRoots := append([]string(nil), initialRoots...)
	reloadInterval := cfg.RuntimeReloadInterval
	if reloadInterval <= 0 && cfg.EnableRuntimePull {
		reloadInterval = 60 * time.Second
	}
	discoveryInterval := cfg.DiscoveryInterval

	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return nil
		}
		sessionCtx, cancel := context.WithCancel(ctx)
		errCh := make(chan error, 1)
		go func() {
			errCh <- runIngestSession(sessionCtx, client, cfg, projectID, token, sources)
		}()

		var reloadCh <-chan time.Time
		var reloadTicker *time.Ticker
		if reloadInterval > 0 && cfg.EnableRuntimePull {
			reloadTicker = time.NewTicker(reloadInterval)
			reloadCh = reloadTicker.C
		}
		var discoveryCh <-chan time.Time
		var discoveryTicker *time.Ticker
		if discoveryInterval > 0 && cfg.EnableDiscovery {
			discoveryTicker = time.NewTicker(discoveryInterval)
			discoveryCh = discoveryTicker.C
		}

		var sessionErr error
		reloaded := false
	waitSession:
		for {
			select {
			case <-ctx.Done():
				cancel()
				<-errCh
				if reloadTicker != nil {
					reloadTicker.Stop()
				}
				if discoveryTicker != nil {
					discoveryTicker.Stop()
				}
				return nil
			case sessionErr = <-errCh:
				break waitSession
			case <-discoveryCh:
				go runDiscoveryReport(ctx, client, cfg, token, sources, runtimeRoots)
			case <-reloadCh:
				bundle, err := fetchRuntimeConfig(ctx, client, token)
				if err != nil || len(bundle.Sources) == 0 {
					logDebugf(cfg.Debug, "runtime reload skipped err=%v", err)
					continue
				}
				if sourcesSignature(sources) == sourcesSignature(bundle.Sources) {
					continue
				}
				logInfof("runtime-config changed, reloading sources")
				sources = append([]runtimeSource(nil), bundle.Sources...)
				runtimeRoots = append([]string(nil), bundle.Roots...)
				if bundle.ProjectID > 0 {
					projectID = bundle.ProjectID
				}
				reloaded = true
				go runDiscoveryReport(ctx, client, cfg, token, sources, runtimeRoots)
				cancel()
				<-errCh
				break waitSession
			}
		}
		if reloadTicker != nil {
			reloadTicker.Stop()
		}
		if discoveryTicker != nil {
			discoveryTicker.Stop()
		}

		if ctx.Err() != nil {
			return nil
		}
		if reloaded {
			backoff = time.Second
			continue
		}
		if sessionErr != nil {
			logInfof("ingest session ended: %v; reconnect in %v", sessionErr, backoff)
		} else {
			logInfof("ingest session closed; reconnect in %v", backoff)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func sourcesSignature(s []runtimeSource) string {
	if len(s) == 0 {
		return ""
	}
	cp := append([]runtimeSource(nil), s...)
	for i := range cp {
		for j := i + 1; j < len(cp); j++ {
			if cp[j].LogSourceID < cp[i].LogSourceID {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	var b strings.Builder
	for _, it := range cp {
		b.WriteString(fmt.Sprintf("%d:%s:%s;", it.LogSourceID, it.LogType, it.Path))
	}
	return b.String()
}
