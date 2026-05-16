package logpath

import "testing"

func TestPathMatchesSource(t *testing.T) {
	if !PathMatchesSource("/var/log/a.log", "/var/log/*.log") {
		t.Fatal("glob should match")
	}
	if PathMatchesSource("/var/log/other/b.log", "/var/log/*.log") {
		t.Fatal("glob should not match nested dir file")
	}
	if !PathMatchesSource("/var/log/messages", "/var/log/messages") {
		t.Fatal("exact should match")
	}
}
