package mysqlbackup

import "testing"

func TestParseFindLatestBackupOutput_OK(t *testing.T) {
	stdout := "SKIP|/data/p_1_2_3_20260517.tar.gz|log_incomplete|err\nOK|/data/p_1_2_3_20260518.tar.gz|/log/p_1_2_3_20260518.log|p_1_2_3_20260518\n"
	got := ParseFindLatestBackupOutput(stdout, 3306)
	if !got.OK {
		t.Fatalf("expected OK, got %+v", got)
	}
	if got.BackupFile != "/data/p_1_2_3_20260518.tar.gz" {
		t.Fatalf("backup file: %s", got.BackupFile)
	}
	if got.Basename != "p_1_2_3_20260518" {
		t.Fatalf("basename: %s", got.Basename)
	}
}

func TestParseFindLatestBackupOutput_NotFound(t *testing.T) {
	got := ParseFindLatestBackupOutput("NOT_FOUND\n", 3306)
	if got.OK {
		t.Fatal("expected not OK")
	}
}
