package alertnotify

import (
	"bytes"
	"html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var mdEngine = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
)

// MarkdownToHTML 将告警 Markdown 转为 HTML（用于邮件 multipart），失败时返回转义后的纯文本包裹。
func MarkdownToHTML(md string) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return "<!DOCTYPE html><html><head><meta charset=\"UTF-8\"/></head><body></body></html>"
	}
	var buf bytes.Buffer
	if err := mdEngine.Convert([]byte(md), &buf); err != nil {
		return "<!DOCTYPE html><html><head><meta charset=\"UTF-8\"/></head><body><pre>" +
			html.EscapeString(md) + "</pre></body></html>"
	}
	return "<!DOCTYPE html><html><head><meta charset=\"UTF-8\"/><style>body{font-family:system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;line-height:1.55;padding:12px 16px;color:#1a1a1a;} code,pre{font-family:ui-monospace,monospace;font-size:0.92em;} table{border-collapse:collapse;} th,td{border:1px solid #ddd;padding:6px 8px;}</style></head><body>" +
		buf.String() + "</body></html>"
}
