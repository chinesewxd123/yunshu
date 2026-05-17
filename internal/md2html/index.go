package md2html

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GenerateIndex writes a simple index.html listing linked pages (same theme via template).
func GenerateIndex(outDir string, templatePath string, entries []IndexEntry) error {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Title < entries[j].Title
	})
	var links strings.Builder
	for _, e := range entries {
		href := filepath.ToSlash(e.RelPath)
		links.WriteString(fmt.Sprintf(
			`<li><a href="%s"><strong>%s</strong><span class="index-desc">%s</span></a></li>`,
			href, escapeHTML(e.Title), escapeHTML(e.Description),
		))
	}
	body := fmt.Sprintf(`
<section class="index-hero">
  <h2>云枢文档（HTML）</h2>
  <p>由 Markdown 自动生成，可离线打开或邮件发送。含侧边目录、Mermaid 图、时间线与提示框。</p>
  <p class="index-meta">生成时间：%s · 共 %d 篇</p>
</section>
<ul class="index-list">%s</ul>
<style>
.index-hero { margin-bottom: 32px; }
.index-meta { color: var(--text-subtle); font-size: 14px; }
.index-list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 10px; }
.index-list a {
  display: flex; flex-direction: column; gap: 4px;
  padding: 14px 18px; border: 1px solid var(--border); border-radius: var(--radius);
  text-decoration: none; color: var(--text); background: var(--surface);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.index-list a:hover { border-color: var(--accent-border); box-shadow: var(--shadow-sm); }
.index-desc { font-size: 13px; color: var(--text-muted); font-weight: 400; }
</style>
`, time.Now().Format("2006-01-02 15:04"), len(entries), links.String())

	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	meta := docMeta{
		Labels:     labelsFor("zh"),
		Title:      "云枢产品文档",
		Subtitle:   "Markdown → 单文件 HTML（md2html 主题）",
		DocType:    "NOTES",
		SourceFile: "index",
		Date:       time.Now().Format("2006-01-02"),
		ReadTime:   "~2 分钟阅读",
	}
	meta.FooterNote = meta.SourcePrefix + " docs/html/"
	out := string(tpl)
	out = strings.ReplaceAll(out, "<!-- TOC_ENTRIES -->", `<a href="#main" class="lvl-2 active">文档列表</a>`)
	out = replaceBetween(out, "<!-- CONTENT_START -->", "<!-- CONTENT_END -->", body)
	out = applyPlaceholders(out, meta)
	outPath := filepath.Join(outDir, "index.html")
	return os.WriteFile(outPath, []byte(out), 0o644)
}

// IndexEntry is one link on the docs index page.
type IndexEntry struct {
	RelPath     string
	Title       string
	Description string
}
