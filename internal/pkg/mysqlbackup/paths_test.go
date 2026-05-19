package mysqlbackup

import "testing"

func TestValidateBackupPathIsolation(t *testing.T) {
	t.Parallel()
	if err := ValidateBackupPathIsolation("/export/backup/yunshu", "/export/xtra/data", "/export/xtra/log"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := ValidateBackupPathIsolation("/export/backup", "/export/backup/xtra", ""); err == nil {
		t.Fatal("expected overlap error for nested paths")
	}
	if err := ValidateBackupPathIsolation("/export/xtra", "/export/xtra", ""); err == nil {
		t.Fatal("expected overlap error for same path")
	}
}

