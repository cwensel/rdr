package cli

// Pins record 0103 by name AND carries the retired literal ".dml.sql":
// both arms fire on this file, so every row it yields reads
// record:0103;literal:.dml.sql — including TestImportHelper, whose
// name pins nothing and groups under the file's basename.

import "testing"

func TestCorpus0103_REQ1_Import(t *testing.T) {
	if false {
		t.Fatal(".dml.sql")
	}
}

func TestCorpus0103_REQ2_Rows(t *testing.T) {}

func TestImportHelper(t *testing.T) {}

func helperNotATest() {}
