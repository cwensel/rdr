package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixturePath(name string) string { return filepath.Join("testdata", name) }

func runCapture(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// TestSelectRoundTripsToBytes: --select <id> prints exactly the record's
// lines for that element.
func TestSelectRoundTripsToBytes(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", "--select", "0004:A2", fixturePath("epoch-d.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	raw, _ := os.ReadFile(fixturePath("epoch-d.md"))
	lines := strings.Split(string(raw), "\n")
	want := strings.Join(lines[43:50], "\n") + "\n" // A2 is lines 44-50
	if out != want {
		t.Errorf("--select output:\n%s\nwant:\n%s", out, want)
	}
	if !strings.HasPrefix(out, "- **A2 [") {
		t.Errorf("selection does not start on the assumption bullet")
	}

	code, _, errb = runCapture(t, "inspect", "--select", "0004:C9", fixturePath("epoch-d.md"))
	if code != 2 || !strings.Contains(errb, "stopped:no-such-element") {
		t.Errorf("missing element: exit %d, stderr %q", code, errb)
	}
}

// TestJSONIsDeterministic: two runs produce identical bytes, and the
// envelope carries the schema version and the project prefix.
func TestJSONIsDeterministic(t *testing.T) {
	_, a, _ := runCapture(t, "inspect", "--json", "--project", "cli", fixturePath("epoch-b.md"))
	_, b, _ := runCapture(t, "inspect", "--json", "--project", "cli", fixturePath("epoch-b.md"))
	if a != b {
		t.Fatal("inspect --json is not byte-deterministic")
	}
	var env struct {
		Schema   string `json:"schema"`
		Record   string `json:"record"`
		Project  string `json:"project"`
		Elements []struct {
			ID string `json:"id"`
		} `json:"elements"`
	}
	if err := json.Unmarshal([]byte(a), &env); err != nil {
		t.Fatal(err)
	}
	if env.Schema != schemaVersion || env.Record != "0002" || env.Project != "cli" {
		t.Errorf("envelope = %+v", env)
	}
	if len(env.Elements) == 0 || !strings.HasPrefix(env.Elements[0].ID, "cli/0002:") {
		t.Errorf("elements not project-qualified: %+v", env.Elements)
	}
}

// TestResolveSkipsPostmortem: NNNN resolves to the record, not to the
// `-postmortem.md` sibling the flow writes beside it, and index --derived
// counts records only.
func TestResolveSkipsPostmortem(t *testing.T) {
	dir := t.TempDir()
	raw, _ := os.ReadFile(fixturePath("epoch-d.md"))
	if err := os.WriteFile(filepath.Join(dir, "0004-checksum.md"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0004-checksum-postmortem.md"), []byte("# Post-Mortem: RDR-0004\n\n## Escaped-Defect Ledger\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCapture(t, "inspect", "--records", dir, "0004")
	if code != 0 || !strings.HasPrefix(out, "0004  Recommendation 0004") {
		t.Errorf("exit %d, out %q, err %q", code, out, errb)
	}
	code, out, errb = runCapture(t, "index", "--derived", "--records", dir)
	if code != 0 || !strings.Contains(out, "total 1 records") || strings.Contains(out, "skipped") {
		t.Errorf("index: exit %d, out %q, err %q", code, out, errb)
	}
	if !strings.Contains(out, "C 1/1") {
		t.Errorf("index does not report the derived contract backlog: %q", out)
	}
}

// TestUnimplementedStillStops: the facets that have not landed keep the
// stopped:<reason> contract.
func TestUnimplementedStillStops(t *testing.T) {
	for _, args := range [][]string{{"lint", "x.md"}, {"index", "--status"}} {
		code, _, errb := runCapture(t, args...)
		if code != 2 || !strings.Contains(errb, "stopped:not-implemented") {
			t.Errorf("%v: exit %d, stderr %q", args, code, errb)
		}
	}
	if code, _, errb := runCapture(t, "inspect", "--bogus", "x.md"); code != 2 || !strings.Contains(errb, "bogus") {
		t.Errorf("unknown flag: exit %d, stderr %q", code, errb)
	}
}
