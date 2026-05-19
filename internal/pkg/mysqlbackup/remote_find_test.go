package mysqlbackup

import "testing"

func TestParseFindLatestRemoteBackupOutput_OK(t *testing.T) {
	stdout := "SKIP|/data/full_20260517.tar.gz|log_incomplete|error\nOK|/data/full_20260518.tar.gz|/log/full_backup_data_2026-05-18.log|2026-05-18\n"
	got := ParseFindLatestRemoteBackupOutput(stdout, 3306)
	if !got.OK {
		t.Fatalf("expected OK, got %+v", got)
	}
	if got.BackupFile != "/data/full_20260518.tar.gz" {
		t.Fatalf("backup file: %s", got.BackupFile)
	}
	if got.BackupDay != "2026-05-18" {
		t.Fatalf("day: %s", got.BackupDay)
	}
}

func TestParseFindLatestRemoteBackupOutput_NotFound(t *testing.T) {
	got := ParseFindLatestRemoteBackupOutput("NOT_FOUND\n", 3306)
	if got.OK {
		t.Fatal("expected not OK")
	}
}
