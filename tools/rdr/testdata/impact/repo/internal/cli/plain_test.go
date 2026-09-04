package cli

// No record-named test. Predicted only when --literal names a token
// this body carries ("dml-sidecar"); with no literal it yields no row.

import "testing"

func TestPlainRoundTrip(t *testing.T) {
	_ = "dml-sidecar"
}
