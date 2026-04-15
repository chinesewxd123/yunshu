package validateutil

import (
	"fmt"
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

// NormalizeRecipientList supports string/[]any and nested separators.
func NormalizeRecipientList(v any) []string {
	switch t := v.(type) {
	case string:
		return SplitRecipientString(t)
	case []any:
		out := make([]string, 0, len(t))
		for _, it := range t {
			s := strings.TrimSpace(fmt.Sprintf("%v", it))
			if s == "" {
				continue
			}
			if sub := SplitRecipientString(s); len(sub) > 1 {
				out = append(out, sub...)
			} else {
				out = append(out, s)
			}
		}
		return out
	default:
		if v == nil {
			return nil
		}
		return SplitRecipientString(fmt.Sprintf("%v", v))
	}
}

