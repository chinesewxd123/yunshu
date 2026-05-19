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
