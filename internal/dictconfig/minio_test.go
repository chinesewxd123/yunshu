package dictconfig

import "testing"

func TestNormalizeMinioEndpoint(t *testing.T) {
	if got := NormalizeMinioEndpoint("175.178.156.23:9001"); got != "175.178.156.23:9000" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeMinioEndpoint("http://minio:9000"); got != "minio:9000" {
		t.Fatalf("got %q", got)
	}
}
