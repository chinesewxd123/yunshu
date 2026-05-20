package md2html

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertSample(t *testing.T) {
	root := findRoot(t)
	md := filepath.Join(root, "docs", "handbook", "README.md")
	tpl := filepath.Join(root, "tools", "md2html", "template.html")
	out := filepath.Join(t.TempDir(), "readme.html")
	if err := Convert(md, out, tpl); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"目录", "产品手册", "toc-nav", "<main"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in output", want)
		}
	}
}

func findRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
