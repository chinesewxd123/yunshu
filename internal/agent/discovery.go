package agent

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	pb "yunshu/internal/grpc/proto"
)

const (
	discoveryReportBatchSize = 300
	discoveryMaxWalkFiles    = 5000
)

type discoveryScanOptions struct {
	Sources []runtimeSource
	Roots   []string
}

func scanDiscoveryAll(ctx context.Context, opts discoveryScanOptions) []discoveryItem {
	out := make([]discoveryItem, 0, 256)
	seen := map[string]struct{}{}
	add := func(kind, value string, extra map[string]any) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := kind + "\n" + value
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, discoveryItem{Kind: kind, Value: value, Extra: extra})
	}
	addFile := func(path string) {
		extra := fileExtra(path)
		add("file", path, extra)
	}

	expandPatternRoot := func(p string) string {
		p = strings.TrimSpace(p)
		if p == "" {
			return ""
		}
		first := strings.IndexAny(p, "*?[")
		if first < 0 {
			return filepath.Dir(p)
		}
		prefix := p[:first]
		if idx := strings.LastIndexAny(prefix, `/\`); idx >= 0 {
			prefix = prefix[:idx]
		}
		return strings.TrimSpace(prefix)
	}

	addFilesByWalk := func(root string, maxFiles int) {
		if maxFiles <= 0 {
			maxFiles = discoveryMaxWalkFiles
		}
		count := 0
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			addFile(path)
			count++
			if count >= maxFiles {
				return io.EOF
			}
			return nil
		})
	}

	scanFilePath := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if strings.ContainsAny(p, "*?[") {
			matches, _ := filepath.Glob(p)
			for _, f := range matches {
				addFile(f)
			}
			if root := expandPatternRoot(p); root != "" {
				add("dir", root, nil)
				addFilesByWalk(root, discoveryMaxWalkFiles)
			}
			return
		}
		if st, err := os.Stat(p); err == nil {
			if st.IsDir() {
				add("dir", p, nil)
				addFilesByWalk(p, discoveryMaxWalkFiles)
			} else {
				addFile(p)
				add("dir", filepath.Dir(p), nil)
			}
		}
	}

	for _, src := range opts.Sources {
		if strings.ToLower(strings.TrimSpace(src.LogType)) != "file" {
			continue
		}
		scanFilePath(src.Path)
	}
	for _, root := range opts.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if st, err := os.Stat(root); err == nil {
			if st.IsDir() {
				add("dir", root, nil)
				addFilesByWalk(root, discoveryMaxWalkFiles)
			} else {
				addFile(root)
			}
		}
	}

	if runtime.GOOS != "windows" {
		cmd := exec.CommandContext(ctx, "sh", "-c", "systemctl list-units --type=service --state=running --no-pager --no-legend 2>/dev/null | awk '{print $1}'")
		if b, err := cmd.Output(); err == nil {
			for _, ln := range strings.Split(string(b), "\n") {
				ln = strings.TrimSpace(ln)
				if ln != "" && strings.HasSuffix(ln, ".service") {
					add("unit", ln, nil)
				}
			}
		}
	}
	return out
}

func fileExtra(path string) map[string]any {
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return map[string]any{
		"size":       st.Size(),
		"mtime_unix": st.ModTime().Unix(),
	}
}

func reportDiscoveryBatched(ctx context.Context, cli pb.AgentRuntimeServiceClient, token string, items []discoveryItem) error {
	if len(items) == 0 {
		return nil
	}
	accepted := 0
	for i := 0; i < len(items); i += discoveryReportBatchSize {
		end := i + discoveryReportBatchSize
		if end > len(items) {
			end = len(items)
		}
		if err := reportDiscovery(ctx, cli, token, items[i:end]); err != nil {
			return err
		}
		accepted += end - i
	}
	_ = accepted
	return nil
}

func runDiscoveryReport(ctx context.Context, cli pb.AgentRuntimeServiceClient, cfg Config, token string, sources []runtimeSource, roots []string) {
	if !cfg.EnableDiscovery {
		return
	}
	mergedRoots := mergeDiscoveryRoots(cfg.DiscoveryRoots, roots)
	if len(sources) == 0 && len(mergedRoots) == 0 {
		mergedRoots = defaultDiscoveryRoots()
	}
	items := scanDiscoveryAll(ctx, discoveryScanOptions{Sources: sources, Roots: mergedRoots})
	if err := reportDiscoveryBatched(ctx, cli, token, items); err != nil {
		logDebugf(cfg.Debug, "discovery report failed err=%v", err)
	} else {
		logDebugf(cfg.Debug, "discovery report sent items=%d", len(items))
	}
}

func mergeDiscoveryRoots(local, remote []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(local)+len(remote))
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range local {
		add(v)
	}
	for _, v := range remote {
		add(v)
	}
	return out
}

func defaultDiscoveryRoots() []string {
	if runtime.GOOS == "windows" {
		return []string{`C:\inetpub\logs\LogFiles`, `C:\ProgramData`}
	}
	return []string{"/var/log"}
}

// ParseDiscoveryRootsFlag splits comma-separated roots from CLI.
func ParseDiscoveryRootsFlag(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	return mergeDiscoveryRoots(parts, nil)
}
