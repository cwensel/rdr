package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// TestMain binds the schema before any test reads a record.
//
// The binary reads TEMPLATE.md at startup, from $RDR_HOME or beside
// itself; a test binary is neither, so it resolves the engine root
// relative to this package. Without this every reader panics, which is
// the intended behaviour of an unbound schema.
func TestMain(m *testing.M) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		panic(err)
	}
	if err := model.MustBindFrom(root); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
