package validateutil

import (
	"fmt"
	"reflect"
	"strings"
)

// SplitRecipientString supports comma/semicolon/newline separated list.
func SplitRecipientString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ";", ",")
	s = strings.ReplaceAll(s, "\n", ",")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// NormalizeRecipientList supports string、[]string、[]any 及任意 slice（元素递归），避免对 []string 使用 fmt.Sprintf("%v") 变成 "[a@x b@y]" 单串。
func NormalizeRecipientList(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		return SplitRecipientString(t)
	case []string:
		var out []string
		for _, it := range t {
			it = strings.TrimSpace(it)
			if it == "" {
				continue
			}
			out = append(out, SplitRecipientString(it)...)
		}
		return out
	case []any:
		var out []string
		for _, it := range t {
			out = append(out, NormalizeRecipientList(it)...)
		}
		return out
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice {
			var out []string
			for i := 0; i < rv.Len(); i++ {
				out = append(out, NormalizeRecipientList(rv.Index(i).Interface())...)
			}
			return out
		}
		return SplitRecipientString(fmt.Sprintf("%v", v))
	}
}

