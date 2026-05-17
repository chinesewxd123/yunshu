package md2html

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Labels holds localized UI strings (Chinese default for yunshu docs).
type Labels struct {
	Lang            string
	RecLabel        string
	TOCTitle        string
	PrintTooltip    string
	ThemeTooltip    string
	CloseLabel      string
	SkipLink        string
	ReadTimeSuffix  string
	HighlightLabel  string
	SourcePrefix    string
	FooterNote      string
	BrandLabel      string
}

var (
	reH2       = regexp.MustCompile(`(?is)<h2\s+id="([^"]+)"[^>]*>(.*?)</h2>`)
	reH3       = regexp.MustCompile(`(?is)<h3\s+id="([^"]+)"[^>]*>(.*?)</h3>`)
	reMermaid  = regexp.MustCompile(`(?is)<pre><code class="language-mermaid">(.*?)</code></pre>`)
	reTable    = regexp.MustCompile(`(?is)(<table>.*?</table>)`)
	reBlockquote = regexp.MustCompile(`(?is)<blockquote>\s*<p>(.*?)</p>(?:\s*<p>(.*?)</p>)*\s*</blockquote>`)
)

// Convert renders a Markdown file into a self-contained HTML page using the md2html template.
func Convert(mdPath, outPath, templatePath string) error {
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}
	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	meta := analyzeMarkdown(string(raw), filepath.Base(mdPath))
	bodyHTML, err := renderMarkdown(raw)
	if err != nil {
		return err
	}
	bodyHTML = enhanceBody(bodyHTML, meta)
	toc := buildTOC(bodyHTML)
	out := string(tpl)
	out = strings.ReplaceAll(out, "<!-- TOC_ENTRIES -->", toc)
	out = replaceBetween(out, "<!-- CONTENT_START -->", "<!-- CONTENT_END -->", bodyHTML)
	out = applyPlaceholders(out, meta)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(out), 0o644)
}

func replaceBetween(s, start, end, insert string) string {
	i := strings.Index(s, start)
	j := strings.Index(s, end)
	if i < 0 || j < 0 || j <= i {
		return s
	}
	return s[:i+len(start)] + "\n" + insert + "\n      " + s[j:]
}

func applyPlaceholders(tpl string, m docMeta) string {
	l := m.Labels
	repl := map[string]string{
		"{{LANG}}":            l.Lang,
		"{{REC_LABEL}}":       l.RecLabel,
		"{{TITLE}}":           escapeHTML(m.Title),
		"{{SUBTITLE}}":        escapeHTML(m.Subtitle),
		"{{DOC_TYPE}}":        m.DocType,
		"{{SOURCE_FILE}}":     escapeHTML(m.SourceFile),
		"{{DATE}}":            m.Date,
		"{{READ_TIME}}":       m.ReadTime,
		"{{BRAND_LABEL}}":     l.BrandLabel,
		"{{TOC_TITLE}}":       l.TOCTitle,
		"{{PRINT_TOOLTIP}}":   l.PrintTooltip,
		"{{THEME_TOOLTIP}}":   l.ThemeTooltip,
		"{{CLOSE_LABEL}}":     l.CloseLabel,
		"{{SKIP_LINK_LABEL}}": l.SkipLink,
		"{{FOOTER_NOTE}}":     l.FooterNote,
	}
	for k, v := range repl {
		tpl = strings.ReplaceAll(tpl, k, v)
	}
	return tpl
}

type docMeta struct {
	Labels
	Title      string
	Subtitle   string
	DocType    string
	SourceFile string
	Date       string
	ReadTime   string
}

func analyzeMarkdown(src, basename string) docMeta {
	lines := strings.Split(src, "\n")
	title := strings.TrimSuffix(basename, filepath.Ext(basename))
	subtitle := ""
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				if next != "" && !strings.HasPrefix(next, "#") {
					subtitle = trimMDInline(next)
					if len(subtitle) > 200 {
						subtitle = subtitle[:197] + "…"
					}
				}
			}
			break
		}
	}
	words := len(strings.Fields(src))
	mins := (words + 124) / 125
	if mins < 1 {
		mins = 1
	}
	lang := detectLang(src)
	labels := labelsFor(lang)
	labels.FooterNote = labels.SourcePrefix + " " + basename
	return docMeta{
		Labels:     labels,
		Title:      title,
		Subtitle:   subtitle,
		DocType:    inferDocType(src, basename),
		SourceFile: basename,
		Date:       time.Now().Format("2006-01-02"),
		ReadTime:   fmt.Sprintf("~%d %s", mins, labels.ReadTimeSuffix),
	}
}

func detectLang(src string) string {
	cjk := 0
	latin := 0
	for _, r := range src {
		if unicode.Is(unicode.Han, r) {
			cjk++
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			latin++
		}
	}
	if cjk > latin/4 {
		return "zh"
	}
	return "en"
}

func labelsFor(lang string) Labels {
	if lang == "zh" {
		return Labels{
			Lang:           "zh",
			RecLabel:       "★ 推荐",
			TOCTitle:       "目录",
			PrintTooltip:   "打印 / 保存 PDF",
			ThemeTooltip:   "切换主题",
			CloseLabel:     "关闭",
			SkipLink:       "跳到正文",
			ReadTimeSuffix: "分钟阅读",
			HighlightLabel: "要点",
			SourcePrefix:   "来源:",
			BrandLabel:     "文档",
		}
	}
	return Labels{
		Lang:           "en",
		RecLabel:       "★ Recommended",
		TOCTitle:       "Contents",
		PrintTooltip:   "Print / Save PDF",
		ThemeTooltip:   "Toggle theme",
		CloseLabel:     "Close",
		SkipLink:       "Skip to content",
		ReadTimeSuffix: "min read",
		HighlightLabel: "Key point",
		SourcePrefix:   "Source:",
		BrandLabel:     "Document",
	}
}

func inferDocType(src, basename string) string {
	low := strings.ToLower(src + " " + basename)
	switch {
	case strings.Contains(low, "runbook") || strings.Contains(basename, "runbook"):
		return "RUNBOOK"
	case strings.Contains(low, "postmortem") || strings.Contains(basename, "postmortem"):
		return "POSTMORTEM"
	case strings.Contains(low, "rfc"):
		return "RFC"
	case strings.Contains(low, "spec") || strings.Contains(basename, "spec"):
		return "SPEC"
	case strings.Contains(basename, "design") || strings.Contains(low, "系统设计") || strings.Contains(low, "architecture"):
		return "SYSTEM DESIGN"
	case strings.Contains(low, "migration") || strings.Contains(low, "迁移"):
		return "PLAN"
	case strings.Contains(low, "guide") || strings.Contains(low, "指南"):
		return "NOTES"
	default:
		return "NOTES"
	}
}

func renderMarkdown(raw []byte) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(),
			html.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(raw, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func enhanceBody(body string, meta docMeta) string {
	body = wrapMermaid(body)
	body = wrapTables(body)
	body = convertBlockquotes(body, meta.Labels)
	body = wrapStepTimelines(body)
	return body
}

func wrapMermaid(body string) string {
	return reMermaid.ReplaceAllStringFunc(body, func(m string) string {
		sub := reMermaid.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		code := strings.TrimSpace(sub[1])
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&amp;", "&")
		return fmt.Sprintf(`<figure class="diagram"><pre class="mermaid">%s</pre><figcaption class="diagram-caption">架构 / 流程图</figcaption></figure>`, code)
	})
}

func wrapTables(body string) string {
	return reTable.ReplaceAllString(body, `<div class="table-wrap">$1</div>`)
}

func convertBlockquotes(body string, labels Labels) string {
	_ = labels
	return reBlockquote.ReplaceAllStringFunc(body, func(m string) string {
		inner := reBlockquote.FindStringSubmatch(m)
		if len(inner) < 2 {
			return m
		}
		text := stripTags(inner[1])
		variant, title := calloutKind(text)
		bodyHTML := inner[1]
		if len(inner) > 2 && strings.TrimSpace(inner[2]) != "" {
			bodyHTML += "</p><p>" + inner[2]
		}
		icon := calloutIcon(variant)
		return fmt.Sprintf(
			`<aside class="callout callout-%s"><svg class="callout-icon" viewBox="0 0 24 24" aria-hidden="true"><use href="#%s"/></svg><div class="callout-body"><p class="callout-title">%s</p><p>%s</p></div></aside>`,
			variant, icon, escapeHTML(title), bodyHTML,
		)
	})
}

func calloutKind(text string) (variant, title string) {
	low := strings.ToLower(text)
	switch {
	case strings.Contains(text, "禁止") || strings.Contains(low, "do not"):
		return "danger", "禁止操作"
	case strings.Contains(text, "注意") || strings.Contains(text, "警告") || strings.Contains(low, "warning"):
		return "warn", "注意"
	case strings.Contains(text, "决定") || strings.Contains(low, "decision"):
		return "decision", "决定"
	case strings.Contains(text, "已完成") || strings.Contains(low, "done"):
		return "success", "已完成"
	case strings.Contains(text, "提示") || strings.Contains(low, "tip"):
		return "tip", "提示"
	default:
		return "info", "说明"
	}
}

func calloutIcon(variant string) string {
	switch variant {
	case "danger":
		return "i-danger"
	case "warn":
		return "i-warn"
	case "decision":
		return "i-decision"
	case "success":
		return "i-success"
	case "tip":
		return "i-tip"
	default:
		return "i-info"
	}
}

func wrapStepTimelines(body string) string {
	reSection := regexp.MustCompile(`(?is)(<h2[^>]*>.*?</h2>)\s*(<ol>.*?</ol>)`)
	return reSection.ReplaceAllStringFunc(body, func(m string) string {
		sub := reSection.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		h2 := sub[1]
		if !stepHeading(stripTags(h2)) {
			return m
		}
		return h2 + "\n" + buildTimelineFromOL(sub[2])
	})
}

func stepHeading(h string) bool {
	for _, kw := range []string{"步骤", "流程", "链路", "阶段", "实施", "验证", "操作"} {
		if strings.Contains(h, kw) {
			return true
		}
	}
	return false
}

var reLI = regexp.MustCompile(`(?is)<li>(.*?)</li>`)

func buildTimelineFromOL(ol string) string {
	items := reLI.FindAllStringSubmatch(ol, -1)
	if len(items) == 0 {
		return ol
	}
	var b strings.Builder
	b.WriteString(`<div class="timeline">`)
	for i, it := range items {
		body := strings.TrimSpace(it[1])
		title, desc := splitStepLI(body)
		b.WriteString(fmt.Sprintf(
			`<article class="step"><div class="step-num">%d</div><div class="step-body"><h3>%s</h3><p>%s</p></div></article>`,
			i+1, title, desc,
		))
	}
	b.WriteString(`</div>`)
	return b.String()
}

func splitStepLI(html string) (title, desc string) {
	if idx := strings.Index(html, "</p><p>"); idx > 0 {
		return stripTags(html[:idx]), stripTags(html[idx+7:])
	}
	plain := stripTags(html)
	if len(plain) > 80 {
		return plain[:80] + "…", plain
	}
	return plain, ""
}

func buildTOC(body string) string {
	var b strings.Builder
	for _, m := range reH2.FindAllStringSubmatch(body, -1) {
		id, text := m[1], stripTags(m[2])
		b.WriteString(fmt.Sprintf(`<a href="#%s" class="lvl-2">%s</a>`, id, escapeHTML(text)))
	}
	for _, m := range reH3.FindAllStringSubmatch(body, -1) {
		id, text := m[1], stripTags(m[2])
		b.WriteString(fmt.Sprintf(`<a href="#%s" class="lvl-3">%s</a>`, id, escapeHTML(text)))
	}
	return b.String()
}

func trimMDInline(s string) string {
	s = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

func stripTags(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(s)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
