package model

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain binds the schema before any test reads a record.
//
// The binary reads TEMPLATE.md at startup; a test binary has no seam
// marker to find it by, so it resolves the engine root relative to this
// package and loads from there. Without this every reader panics, which
// is the intended behaviour of an unbound schema.
func TestMain(m *testing.M) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		panic(err)
	}
	if err := MustBindFrom(root); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
