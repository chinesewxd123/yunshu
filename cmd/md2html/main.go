package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yunshu/internal/md2html"
)

func main() {
	var (
		outFlag      string
		outDirFlag   string
		templateFlag string
		bundleFlag   bool
		indexFlag    bool
	)
	flag.StringVar(&outFlag, "out", "", "output .html path (default: same dir as .md with .html ext)")
	flag.StringVar(&outDirFlag, "out-dir", "docs/html", "output directory for --bundle")
	flag.StringVar(&templateFlag, "template", "tools/md2html/template.html", "path to md2html template.html")
	flag.BoolVar(&bundleFlag, "bundle", false, "convert preset core documentation set into --out-dir")
	flag.BoolVar(&indexFlag, "index", true, "with --bundle, also write index.html")
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !filepath.IsAbs(templateFlag) {
		templateFlag = filepath.Join(repoRoot, templateFlag)
	}
	if !filepath.IsAbs(outDirFlag) {
		outDirFlag = filepath.Join(repoRoot, outDirFlag)
	}

	args := flag.Args()
	if bundleFlag {
		if err := runBundle(repoRoot, templateFlag, outDirFlag, indexFlag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("done:", outDirFlag)
		return
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/md2html [--bundle] [file.md ...]")
		fmt.Fprintln(os.Stderr, "  --bundle   build docs/html from preset markdown list")
		os.Exit(2)
	}
	for _, md := range args {
		mdPath := md
		if !filepath.IsAbs(mdPath) {
			mdPath = filepath.Join(repoRoot, mdPath)
		}
		out := outFlag
		if out == "" {
			out = strings.TrimSuffix(mdPath, filepath.Ext(mdPath)) + ".html"
		} else if !filepath.IsAbs(out) {
			out = filepath.Join(repoRoot, out)
		}
		if err := md2html.Convert(mdPath, out, templateFlag); err != nil {
			fmt.Fprintln(os.Stderr, md, ":", err)
			os.Exit(1)
		}
		fmt.Println("wrote", out)
	}
}

func runBundle(repoRoot, templatePath, outDir string, withIndex bool) error {
	pairs := bundleDocs()
	var indexEntries []md2html.IndexEntry
	for _, p := range pairs {
		mdPath := filepath.Join(repoRoot, p.MD)
		outPath := filepath.Join(outDir, p.HTML)
		if err := md2html.Convert(mdPath, outPath, templatePath); err != nil {
			return fmt.Errorf("%s: %w", p.MD, err)
		}
		fmt.Println("wrote", outPath)
		indexEntries = append(indexEntries, md2html.IndexEntry{
			RelPath:     filepath.ToSlash(p.HTML),
			Title:       p.Title,
			Description: p.Desc,
		})
	}
	if withIndex {
		return md2html.GenerateIndex(outDir, templatePath, indexEntries)
	}
	return nil
}

type bundleDoc struct {
	MD    string
	HTML  string
	Title string
	Desc  string
}

func bundleDocs() []bundleDoc {
	return []bundleDoc{
		{"docs/alert-notify-guide.md", "alert-notify-guide.html", "告警通知与恢复", "配置、聚合、通道与 Alertmanager 对接"},
		{"docs/alert-routing-and-delivery-guide.md", "alert-routing-and-delivery-guide.html", "告警路由与投递", "订阅树、处理人、值班、端到端链路"},
		{"docs/alert-subscription-labels-chain.md", "alert-subscription-labels-chain.html", "订阅标签链路", "match_labels 与 Prometheus/平台规则对齐"},
		{"docs/requirements/R-alert-platform-detailed-design.md", "R-alert-platform-detailed-design.html", "告警平台详细设计", "模型、API、Redis、投递流水线"},
		{"docs/handbook/README.md", "handbook-readme.html", "产品手册索引", "handbook 目录与文档导航"},
		{"docs/handbook/requirements/R-03-alert-and-monitor.md", "R-03-alert-and-monitor.html", "需求：告警与监控", "功能结构与注意事项"},
		{"docs/handbook/requirements/menus/menu-alert-monitor-platform.md", "menu-alert-monitor-platform.html", "菜单：告警监控平台", "Tab、API、处理人与值班"},
		{"docs/handbook/requirements/menus/menu-alert-duty.md", "menu-alert-duty.html", "菜单：值班总览", "班次甘特与 CRUD"},
		{"docs/handbook/database/schema-and-relationships.md", "schema-and-relationships.html", "数据库表与关系", "核心 ER 与表分组"},
		{"docs/log-platform-api.md", "log-platform-api.html", "日志平台 API", "gRPC 与 Agent 对接"},
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err2 := os.Stat(filepath.Join(dir, "tools", "md2html", "template.html")); err2 == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find repo root (go.mod + tools/md2html/template.html)")
}
