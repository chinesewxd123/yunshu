package mysqlbackup

import (
	"encoding/json"
	"testing"
)

func TestRemoteCheckResultJSONUsesLowerCamel(t *testing.T) {
	b, err := json.Marshal(RemoteCheckResult{
		OK: true, BackupFile: "/data/a.tar.gz", LogFile: "/log/a.log",
		LogCompleted: true, Message: "ok",
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["ok"] != true {
		t.Fatalf("want ok=true, got %v", m["ok"])
	}
	if m["OK"] != nil {
		t.Fatalf("unexpected PascalCase OK: %v", m["OK"])
	}
	if m["message"] != "ok" {
		t.Fatalf("want message, got %v", m)
	}
}
