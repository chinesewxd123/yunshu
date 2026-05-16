package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "yunshu/internal/grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Config struct {
	GrpcServer        string
	PlatformURL       string
	ProjectID         uint
	ServerID          uint
	LogSourceID       uint
	Token             string
	RegisterSecret    string
	Name              string
	Version           string
	SourceType        string
	Path              string
	TailLines         int
	BatchSize         int
	FlushInterval     time.Duration
	ResendAfter       time.Duration
	Debug             bool
	ListenPort        int
	EnableRuntimePull bool
	EnableFallback    bool
	EnableDiscovery   bool
	EnableHealth      bool
	// RuntimeReloadInterval 拉取平台 runtime-config 的间隔；变更后重启采集会话。
	RuntimeReloadInterval time.Duration
	// DiscoveryRoots 额外扫描根目录（逗号分隔 CLI）；与平台 _discovery_root 源合并。
	DiscoveryRoots []string
	// DiscoveryInterval 周期性补扫间隔；0 表示仅启动与配置变更时扫描。
	DiscoveryInterval time.Duration
}

func logInfof(format string, args ...any) {
	fmt.Printf("[agent][info] "+format+"\n", args...)
}

func logDebugf(enabled bool, format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Printf("[agent][debug] "+format+"\n", args...)
}

type runtimeSource struct {
	LogSourceID uint   `json:"log_source_id"`
	LogType     string `json:"log_type"`
	Path        string `json:"path"`
}

func (c *Config) normalize() error {
	if c.ServerID == 0 {
		return errors.New("server-id is required")
	}
	base := strings.TrimSpace(c.GrpcServer)
	if base == "" {
		return errors.New("grpc-server is required")
	}
	c.GrpcServer = base
	if c.TailLines <= 0 {
		c.TailLines = 200
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = 250 * time.Millisecond
	}
	if c.ResendAfter <= 0 {
		c.ResendAfter = 3 * time.Second
	}
	if c.ListenPort < 0 {
		c.ListenPort = 0
	}
	if !c.EnableRuntimePull && !c.EnableFallback && !c.EnableDiscovery && !c.EnableHealth {
		// Backward compatible defaults for legacy callers.
		c.EnableRuntimePull = true
		c.EnableFallback = false
		c.EnableDiscovery = true
		c.EnableHealth = true
	}
	return nil
}

func inferPlatformURL(grpcServer string) string {
	v := strings.TrimSpace(grpcServer)
	if v == "" {
		return ""
	}
	// host:port => http://host:8080
	if strings.Contains(v, "://") {
		return strings.TrimRight(v, "/")
	}
	host := v
	if idx := strings.Index(v, ":"); idx > 0 {
		host = v[:idx]
	}
	if strings.TrimSpace(host) == "" {
		return ""
	}
	return "http://" + host + ":8080"
}

func reportHealth(ctx context.Context, platformURL string, payload map[string]any) error {
	platformURL = strings.TrimRight(strings.TrimSpace(platformURL), "/")
	if platformURL == "" {
		return nil
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, platformURL+"/api/v1/agents/health/report", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("health report status=%d", resp.StatusCode)
	}
	return nil
}

func publicRegister(ctx context.Context, cli pb.AgentRuntimeServiceClient, serverID uint, name, version, secret string) (projectID uint, token string, err error) {
	resp, err := cli.PublicRegister(ctx, &pb.PublicRegisterRequest{
		ServerId:       uint64(serverID),
		Name:           strings.TrimSpace(name),
		Version:        strings.TrimSpace(version),
		RegisterSecret: strings.TrimSpace(secret),
	})
	if err != nil {
		return 0, "", err
	}
	if strings.TrimSpace(resp.GetToken()) == "" {
		return 0, "", fmt.Errorf("public-register empty token")
	}
	return uint(resp.GetProjectId()), resp.GetToken(), nil
}

type ingestMessage struct {
	ProjectID   uint          `json:"project_id"`
	ServerID    uint          `json:"server_id"`
	LogSourceID uint          `json:"log_source_id"`
	Seq         uint64        `json:"seq"`
	Entries     []ingestEntry `json:"entries,omitempty"`
}

type ingestEntry struct {
	Line     string `json:"line"`
	FilePath string `json:"file_path,omitempty"`
}

type discoveryItem struct {
	Kind  string         `json:"kind"`
	Value string         `json:"value"`
	Extra map[string]any `json:"extra,omitempty"`
}

func reportDiscovery(ctx context.Context, cli pb.AgentRuntimeServiceClient, token string, items []discoveryItem) error {
	if len(items) == 0 {
		return nil
	}
	reqItems := make([]*pb.AgentDiscoveryItem, 0, len(items))
	for _, it := range items {
		extra := map[string]string{}
		for k, v := range it.Extra {
			extra[k] = fmt.Sprint(v)
		}
		reqItems = append(reqItems, &pb.AgentDiscoveryItem{
			Kind:  it.Kind,
			Value: it.Value,
			Extra: extra,
		})
	}
	_, err := cli.ReportDiscovery(ctx, &pb.ReportDiscoveryRequest{
		Token: token,
		Items: reqItems,
	})
	if err != nil {
		return err
	}
	return nil
}

type pendingBatch struct {
	msg      ingestMessage
	lastSent time.Time
}

type collectedLine struct {
	logSourceID uint
	line        string
	filePath    string
}

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.normalize(); err != nil {
		return err
	}
	logInfof("starting agent name=%s version=%s server=%d grpc=%s", strings.TrimSpace(cfg.Name), strings.TrimSpace(cfg.Version), cfg.ServerID, cfg.GrpcServer)
	logInfof(
		"features runtime=%v fallback=%v discovery=%v health=%v",
		cfg.EnableRuntimePull, cfg.EnableFallback, cfg.EnableDiscovery, cfg.EnableHealth,
	)
	conn, err := grpc.NewClient(cfg.GrpcServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	agentClient := pb.NewAgentRuntimeServiceClient(conn)
	token := strings.TrimSpace(cfg.Token)
	projectID := cfg.ProjectID
	if token == "" {
		sec := strings.TrimSpace(cfg.RegisterSecret)
		if sec == "" {
			return errors.New("令牌不能为空 (or provide register-secret for agent-first mode)")
		}
		pid, tk, err := publicRegister(ctx, agentClient, cfg.ServerID, cfg.Name, cfg.Version, sec)
		if err != nil {
			return err
		}
		projectID = pid
		token = tk
		logInfof("public register succeeded project=%d server=%d", projectID, cfg.ServerID)
	}
	if strings.TrimSpace(cfg.PlatformURL) == "" {
		cfg.PlatformURL = inferPlatformURL(cfg.GrpcServer)
	}
	if cfg.EnableHealth {
		_ = reportHealth(ctx, cfg.PlatformURL, map[string]any{
			"token":            token,
			"listen_port":      cfg.ListenPort,
			"install_progress": 60,
			"health_status":    "starting",
			"version":          cfg.Version,
		})
	}

	var sources []runtimeSource
	var runtimeRoots []string
	if cfg.EnableRuntimePull {
		bundle, cfgErr := fetchRuntimeConfig(ctx, agentClient, token)
		err = cfgErr
		sources = bundle.Sources
		runtimeRoots = bundle.Roots
		if err == nil && len(sources) > 0 {
			projectID = bundle.ProjectID
			logInfof("runtime-config loaded project=%d sources=%d discovery_roots=%d", projectID, len(sources), len(runtimeRoots))
		}
	}
	if len(sources) == 0 {
		if !cfg.EnableFallback {
			if cfg.EnableRuntimePull {
				if err != nil {
					return fmt.Errorf("runtime-config failed and fallback disabled: %w", err)
				}
				return errors.New("runtime-config empty and fallback disabled")
			}
			return errors.New("runtime-pull disabled and fallback disabled")
		}
		if err != nil {
			logInfof("runtime-config unavailable, fallback mode enabled err=%v", err)
		} else {
			logInfof("runtime-config empty, fallback mode enabled")
		}
		if cfg.LogSourceID == 0 || strings.TrimSpace(cfg.Path) == "" {
			return errors.New("fallback enabled but log-source-id/path is missing")
		}
		if projectID == 0 {
			return errors.New("project-id is required when using fallback single source")
		}
		sources = []runtimeSource{{
			LogSourceID: cfg.LogSourceID,
			LogType:     cfg.SourceType,
			Path:        cfg.Path,
		}}
	}
	if projectID == 0 {
		return errors.New("project-id is empty")
	}
	if cfg.EnableDiscovery {
		go runDiscoveryReport(ctx, agentClient, cfg, token, sources, runtimeRoots)
	}
	err = runAgentSupervisor(ctx, agentClient, cfg, projectID, token, sources, runtimeRoots)
	if cfg.EnableHealth {
		_ = reportHealth(context.Background(), cfg.PlatformURL, map[string]any{
			"token":            token,
			"listen_port":      cfg.ListenPort,
			"install_progress": 100,
			"health_status":    "stopped",
			"last_error":       fmt.Sprint(err),
			"version":          cfg.Version,
		})
	}
	return err
}

func runIngestSession(ctx context.Context, client pb.AgentRuntimeServiceClient, cfg Config, projectID uint, token string, sources []runtimeSource) error {
	logInfof("connecting ingest stream project=%d server=%d", projectID, cfg.ServerID)
	ingestCtx := metadata.AppendToOutgoingContext(ctx, "x-agent-token", token)
	stream, err := client.IngestLogs(ingestCtx)
	if err != nil {
		return err
	}
	logInfof("ingest stream connected")
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	var writeMu sync.Mutex
	writeIngest := func(v ingestMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		entries := make([]*pb.IngestEntry, 0, len(v.Entries))
		for _, e := range v.Entries {
			entries = append(entries, &pb.IngestEntry{Line: e.Line, FilePath: e.FilePath})
		}
		return stream.Send(&pb.IngestLogsRequest{
			ProjectId:   uint64(v.ProjectID),
			ServerId:    uint64(v.ServerID),
			LogSourceId: uint64(v.LogSourceID),
			Seq:         v.Seq,
			Entries:     entries,
		})
	}

	var seq uint64
	pending := map[uint64]*pendingBatch{}
	var pendingMu sync.Mutex

	go func() {
		for {
			ack, err := stream.Recv()
			if err != nil {
				streamCancel()
				return
			}
			pendingMu.Lock()
			delete(pending, ack.GetSeq())
			pendingMu.Unlock()
		}
	}()

	mergedLines := make(chan collectedLine, 4096)
	sourceReceived := make(map[uint]uint64, len(sources))
	sourceSent := make(map[uint]uint64, len(sources))
	sourceLastFile := make(map[uint]string, len(sources))
	sourceLastLine := make(map[uint]string, len(sources))

	// Merged in-process collector for file sources (avoids spawning N tail processes).
	fc := newFileCollector(cfg.Debug)
	var hasMergedFiles bool

	for _, src := range sources {
		src := src
		if strings.ToLower(strings.TrimSpace(src.LogType)) == "file" {
			if err := fc.AddSource(src.LogSourceID, src.Path, cfg.TailLines); err != nil {
				if cfg.Debug {
					fmt.Printf("[agent][file] skip source=%d path=%s err=%v\n", src.LogSourceID, src.Path, err)
				}
				continue
			}
			hasMergedFiles = true
			if cfg.Debug {
				fmt.Printf("[agent][file] add source=%d path=%s\n", src.LogSourceID, src.Path)
			}
			continue
		}
		logInfof("starting source type=%s id=%d path=%s", src.LogType, src.LogSourceID, src.Path)

		ch, err := startLocalSource(ctx, src.LogType, src.Path, cfg.TailLines)
		if err != nil {
			if cfg.Debug {
				fmt.Printf("[agent][source] start failed type=%s path=%s err=%v\n", src.LogType, src.Path, err)
			}
			continue
		}
		go func() {
			for ln := range ch {
				if ln == "" {
					continue
				}
				mergedLines <- collectedLine{logSourceID: src.LogSourceID, line: ln}
			}
		}()
	}
	if hasMergedFiles {
		logInfof("file merged collector enabled")
		ch := fc.Start(ctx, 300*time.Millisecond)
		go func() {
			for it := range ch {
				if it.line == "" || it.logSourceID == 0 {
					continue
				}
				mergedLines <- it
			}
		}()
	}

	flushTicker := time.NewTicker(cfg.FlushInterval)
	defer flushTicker.Stop()
	resendTicker := time.NewTicker(1 * time.Second)
	defer resendTicker.Stop()
	statusTicker := time.NewTicker(1 * time.Minute)
	defer statusTicker.Stop()

	buffers := map[uint][]collectedLine{}
	var receivedLines uint64
	var sentLines uint64
	flushOne := func(logSourceID uint) {
		buf := buffers[logSourceID]
		if len(buf) == 0 {
			return
		}
		entries := make([]ingestEntry, 0, len(buf))
		for _, it := range buf {
			if it.line == "" {
				continue
			}
			entries = append(entries, ingestEntry{
				Line:     it.line,
				FilePath: strings.TrimSpace(it.filePath),
			})
		}
		if len(entries) == 0 {
			buffers[logSourceID] = buffers[logSourceID][:0]
			return
		}
		if len(entries) > 0 {
			sourceSent[logSourceID] += uint64(len(entries))
		}
		s := atomic.AddUint64(&seq, 1)
		msg := ingestMessage{
			ProjectID:   projectID,
			ServerID:    cfg.ServerID,
			LogSourceID: logSourceID,
			Seq:         s,
			Entries:     entries,
		}
		if err := writeIngest(msg); err == nil {
			pendingMu.Lock()
			pending[s] = &pendingBatch{msg: msg, lastSent: time.Now()}
			pendingMu.Unlock()
			atomic.AddUint64(&sentLines, uint64(len(entries)))
			logDebugf(cfg.Debug, "sent batch source=%d seq=%d lines=%d", logSourceID, s, len(entries))
			buffers[logSourceID] = buffers[logSourceID][:0]
		} else {
			logDebugf(cfg.Debug, "send batch failed source=%d seq=%d err=%v (buffer retained)", logSourceID, s, err)
		}
	}
	flushAll := func() {
		for id := range buffers {
			flushOne(id)
		}
	}

	for {
		select {
		case <-streamCtx.Done():
			flushAll()
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("ingest stream disconnected")
		case <-ctx.Done():
			flushAll()
			return nil
		case it := <-mergedLines:
			buffers[it.logSourceID] = append(buffers[it.logSourceID], it)
			atomic.AddUint64(&receivedLines, 1)
			sourceReceived[it.logSourceID]++
			if fp := strings.TrimSpace(it.filePath); fp != "" {
				sourceLastFile[it.logSourceID] = fp
			}
			if ln := strings.TrimSpace(it.line); ln != "" {
				if len(ln) > 120 {
					ln = ln[:120] + "..."
				}
				sourceLastLine[it.logSourceID] = ln
			}
			if len(buffers[it.logSourceID]) >= cfg.BatchSize {
				flushOne(it.logSourceID)
			}
		case <-flushTicker.C:
			flushAll()
		case <-resendTicker.C:
			now := time.Now()
			pendingMu.Lock()
			for _, it := range pending {
				if now.Sub(it.lastSent) >= cfg.ResendAfter {
					if err := writeIngest(it.msg); err != nil {
						logDebugf(cfg.Debug, "resend batch failed seq=%d err=%v", it.msg.Seq, err)
						continue
					}
					it.lastSent = now
				}
			}
			pendingMu.Unlock()
		case <-statusTicker.C:
			pendingMu.Lock()
			pendingCount := len(pending)
			pendingMu.Unlock()
			logInfof("running sources=%d received=%d sent=%d pending=%d", len(sources), atomic.LoadUint64(&receivedLines), atomic.LoadUint64(&sentLines), pendingCount)
			for _, src := range sources {
				lastFile := sourceLastFile[src.LogSourceID]
				lastLine := sourceLastLine[src.LogSourceID]
				logInfof(
					"source id=%d type=%s recv=%d sent=%d buffer=%d path=%s last_file=%s last_line=%q",
					src.LogSourceID,
					src.LogType,
					sourceReceived[src.LogSourceID],
					sourceSent[src.LogSourceID],
					len(buffers[src.LogSourceID]),
					src.Path,
					lastFile,
					lastLine,
				)
			}
			if cfg.EnableHealth {
				_ = reportHealth(ctx, cfg.PlatformURL, map[string]any{
					"token":            token,
					"listen_port":      cfg.ListenPort,
					"install_progress": 100,
					"health_status":    "running",
					"version":          cfg.Version,
				})
			}
		}
	}
}

func startLocalSource(ctx context.Context, sourceType, path string, tailLines int) (<-chan string, error) {
	t := strings.ToLower(strings.TrimSpace(sourceType))
	lines := make(chan string, 2048)
	var command string
	if t == "journal" {
		command = fmt.Sprintf("journalctl -u %q -n %d -f -o cat --no-pager", path, tailLines)
	} else {
		// Normalize common Linux log short names like "messages" -> "/var/log/messages".
		// This keeps the UI config ergonomic while still tailing the real file.
		origPath := strings.TrimSpace(path)
		if runtime.GOOS != "windows" && origPath != "" && !strings.ContainsAny(origPath, `/\`) && !filepath.IsAbs(origPath) {
			path = filepath.Join("/var/log", origPath)
		}

		// If a directory is provided, tail all *.log under it (common for nginx, etc.).
		// This keeps the config ergonomic while still using a single tail process.
		isDir := false
		if st, err := os.Stat(path); err == nil && st.IsDir() {
			isDir = true
		}
		hasGlob := strings.ContainsAny(path, "*?[]")
		if runtime.GOOS == "windows" {
			if isDir {
				p := filepath.Join(path, "*.log")
				command = fmt.Sprintf("powershell -NoProfile -Command \"Get-Content -Path '%s' -Tail %d -Wait\"", p, tailLines)
			} else if hasGlob {
				// Keep wildcard unescaped for PowerShell expansion.
				command = fmt.Sprintf("powershell -NoProfile -Command \"Get-Content -Path %s -Tail %d -Wait\"", path, tailLines)
			} else {
				command = fmt.Sprintf("powershell -NoProfile -Command \"Get-Content -Path '%s' -Tail %d -Wait\"", path, tailLines)
			}
		} else {
			if isDir {
				// Keep glob expansion while safely quoting directory.
				command = fmt.Sprintf("dir=%q; tail -n %d --follow=name --retry --sleep-interval=1 \"$dir\"/*.log", path, tailLines)
			} else if hasGlob {
				// Keep wildcard unquoted so shell can expand (e.g. /var/log/app/*.log).
				command = fmt.Sprintf("tail -n %d --follow=name --retry --sleep-interval=1 %s", tailLines, path)
			} else {
				command = fmt.Sprintf("tail -n %d --follow=name --retry --sleep-interval=1 %q", tailLines, path)
			}
		}
	}
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	} else {
		c = exec.CommandContext(ctx, "sh", "-c", command)
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
		}
	}()
	go func() {
		_ = c.Wait()
	}()
	return lines, nil
}
