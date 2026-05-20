package mysqlbackup

import (
	"strings"
	"testing"
)

func TestFormatMysqldumpFlags(t *testing.T) {
	flags, err := FormatMysqldumpFlags(DefaultMysqldumpOptionIDs(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--single-transaction", "--quick", "--routines", "--triggers"} {
		if !strings.Contains(flags, want) {
			t.Fatalf("missing %s in %s", want, flags)
		}
	}
}

func TestMysqldumpOptionCatalogSize(t *testing.T) {
	t.Parallel()
	if len(MysqldumpOptionCatalog) < 40 {
		t.Fatalf("expected expanded catalog, got %d options", len(MysqldumpOptionCatalog))
	}
	_, err := FormatMysqldumpFlags([]string{"add_drop_table", "default_charset_utf8mb4", "column_statistics_off"}, "")
	if err != nil {
		t.Fatal(err)
	}
}
