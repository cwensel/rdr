package dep

// Under vendor/, which the walk skips: pins 0103 and carries ".dml.sql",
// and must contribute nothing.

import "testing"

func TestDep0103_REQ1_Vendored(t *testing.T) {
	_ = ".dml.sql"
}
