package model

import "testing"

func TestK8sForwardedEvent_ShouldForward(t *testing.T) {
	tests := []struct {
		typ  string
		want bool
	}{
		{"Normal", false},
		{"normal", false},
		{"Warning", true},
		{"warning", true},
		{"", true},
		{"Error", true},
	}
	for _, tt := range tests {
		ev := &K8sForwardedEvent{Type: tt.typ}
		if got := ev.ShouldForward(); got != tt.want {
			t.Fatalf("Type=%q: got %v want %v", tt.typ, got, tt.want)
		}
	}
}
