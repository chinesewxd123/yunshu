package middleware

import (
	"strings"
	"testing"
)

func TestMaskSensitiveQuery(t *testing.T) {
	if got := maskSensitiveQuery(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	got := maskSensitiveQuery("token=supersecretjwt&foo=bar")
	if strings.Contains(got, "supersecretjwt") {
		t.Fatalf("leaked token value: %q", got)
	}
	// url.Values.Encode 将 * 编码为 %2A
	if !strings.Contains(got, "%2A%2A%2A") && !strings.Contains(got, "***") {
		t.Fatalf("expected masked token param: %q", got)
	}
	got2 := maskSensitiveQuery("ACCESS_TOKEN=secretval&n=1")
	if strings.Contains(got2, "secretval") {
		t.Fatalf("leaked access_token: %q", got2)
	}
}
