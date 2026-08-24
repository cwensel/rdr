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
// stopped:<reason> contract, and one that HAS landed reports its own
// failure rather than the not-implemented one — `lint` on a missing file
// is unreadable, not unbuilt.
func TestUnimplementedStillStops(t *testing.T) {
	for _, args := range [][]string{{"index", "--status"}} {
		code, _, errb := runCapture(t, args...)
		if code != 2 || !strings.Contains(errb, "stopped:not-implemented") {
			t.Errorf("%v: exit %d, stderr %q", args, code, errb)
		}
	}
	if code, _, errb := runCapture(t, "lint", "x.md"); code != 2 || !strings.Contains(errb, "stopped:unreadable") {
		t.Errorf("lint on a missing file: exit %d, stderr %q", code, errb)
	}
	if code, _, errb := runCapture(t, "inspect", "--bogus", "x.md"); code != 2 || !strings.Contains(errb, "bogus") {
		t.Errorf("unknown flag: exit %d, stderr %q", code, errb)
	}
}

// TestIndexCoverage: the drift alarm reports the corpus rate, warnings by
// code, and any unknown heading or author label that recurs across
// records — and nothing that appears in fewer than recurThreshold.
func TestIndexCoverage(t *testing.T) {
	dir := t.TempDir()
	clean, _ := os.ReadFile(fixturePath("epoch-d.md"))
	author, _ := os.ReadFile(filepath.Join("testdata", "variants", "author-structure.md"))
	if err := os.WriteFile(filepath.Join(dir, "0004-checksum.md"), clean, 0o644); err != nil {
		t.Fatal(err)
	}
	// The same author structure in three records is a convention worth
	// listing; the record number is rewritten so each is its own record.
	for _, n := range []string{"0105", "0106", "0107"} {
		raw := strings.Replace(string(author), "Recommendation 0105:", "Recommendation "+n+":", 1)
		if err := os.WriteFile(filepath.Join(dir, n+"-fork-policy.md"), []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "0108-notes.md"), []byte("# Notes\n\nnot a record\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errb := runCapture(t, "index", "--coverage", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{
		"total 4 records",
		"warning section:unknown-to-template  3",
		`recurring heading "Why not a vendored copy"`,
		`recurring heading "Appendix A — Reader Catalog"`, // the alarm case: a foreign root in three records
		`recurring label   "Forward-compat"`,
		"§Cross-Cutting Concerns",
		"skipped " + filepath.Join(dir, "0108-notes.md"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `"Category"`) || strings.Contains(out, `"Reader 1"`) {
		t.Errorf("a heading or label inside a foreign section is that section's warning, not a recurrence:\n%s", out)
	}

	code, out, _ = runCapture(t, "index", "--coverage", "--json", "--records", dir)
	if code != 0 {
		t.Fatal(out)
	}
	var got struct {
		Total struct {
			Lines, Unclassified int
			Rate                float64
		}
		Warnings  map[string]int
		Recurring []struct {
			Kind, Text string
			Records    int
		}
		Skipped []string
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Total.Unclassified != 15 || got.Total.Rate <= 0 || got.Total.Rate > 0.05 {
		t.Errorf("total = %+v, want 3×5 unclassified lines at a small rate", got.Total)
	}
	if got.Warnings["section:unknown-to-template"] != 3 || len(got.Skipped) != 1 {
		t.Errorf("warnings %v skipped %v", got.Warnings, got.Skipped)
	}
	for _, r := range got.Recurring {
		if r.Records != 3 {
			t.Errorf("recurring %s %q in %d records, want 3", r.Kind, r.Text, r.Records)
		}
	}
	if len(got.Recurring) == 0 {
		t.Error("no recurring entries")
	}
}

// TestInspectEdgesFacet: `--select edges` projects the typed relations,
// and the envelope carries them under schema 2.
func TestInspectEdgesFacet(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", "--json", "--select", "edges", fixturePath("epoch-d.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var edges []struct {
		From, To, Kind string
		Resolved       *bool
	}
	if err := json.Unmarshal([]byte(out), &edges); err != nil {
		t.Fatalf("decoding edges: %v", err)
	}
	if len(edges) == 0 {
		t.Fatal("no edges projected")
	}
	kinds := map[string]bool{}
	for _, e := range edges {
		kinds[e.Kind] = true
	}
	for _, want := range []string{"predecessor", "overrides", "cluster", "joint-decision-home", "peer-evidence", "source-anchor"} {
		if !kinds[want] {
			t.Errorf("no %s edge in the projection", want)
		}
	}

	// The envelope carries edges too, at the schema that added them.
	_, full, _ := runCapture(t, "inspect", "--json", fixturePath("epoch-d.md"))
	var env struct {
		Schema string `json:"schema"`
		Edges  []struct {
			Kind string `json:"kind"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(full), &env); err != nil {
		t.Fatal(err)
	}
	if env.Schema != "2" {
		t.Errorf("schema = %q; edges[] is a schema-2 addition", env.Schema)
	}
	if len(env.Edges) != len(edges) {
		t.Errorf("envelope has %d edges, the facet %d", len(env.Edges), len(edges))
	}
}

// TestEdgesAreDeterministic: two projections of the same record produce
// identical edge bytes, so a diff means the record changed.
func TestEdgesAreDeterministic(t *testing.T) {
	_, a, _ := runCapture(t, "inspect", "--json", "--select", "edges", "--project", "proj", fixturePath("epoch-c.md"))
	_, b, _ := runCapture(t, "inspect", "--json", "--select", "edges", "--project", "proj", fixturePath("epoch-c.md"))
	if a != b {
		t.Error("edge projection is not byte-deterministic")
	}
}

// TestIndexUnresolvedFacet: the facet reports typed edges whose target
// was looked for and not found, and never mentions.
func TestIndexUnresolvedFacet(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Profile makes these epoch B or later, which is what a record
	// carrying Predecessors or Cluster actually is: those fields do not
	// exist in epoch A, and the model is right to call them foreign there.
	head := func(num, title string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n"
	}
	write("0001-alpha.md", head("0001", "Alpha")+"\n## Problem Statement\n\nSynthetic.\n")
	write("0002-beta.md", head("0002", "Beta")+
		"- **Predecessors**: 0001-alpha, 0099-missing\n\n## Problem Statement\n\nSee 0001-alpha for context.\n")

	code, out, errb := runCapture(t, "index", "--unresolved", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0099") {
		t.Errorf("the missing record is not reported:\n%s", out)
	}
	if !strings.Contains(out, "total 1 unresolved") {
		t.Errorf("want exactly one finding; got:\n%s", out)
	}
}

// TestIndexClusterFacet: `--cluster-of` answers 7.1's membership question
// from the CLI.
func TestIndexClusterFacet(t *testing.T) {
	dir := t.TempDir()
	head := func(num, title string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n"
	}
	for name, body := range map[string]string{
		"0001-alpha.md": head("0001", "Alpha") + "- **Cluster**: 0002-beta\n\n## Problem Statement\n\nSynthetic.\n",
		"0002-beta.md":  head("0002", "Beta") + "\n## Problem Statement\n\nSynthetic.\n",
		"0003-gamma.md": head("0003", "Gamma") + "\n## Problem Statement\n\nSynthetic.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errb := runCapture(t, "index", "--cluster-of", "0001", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0002") || !strings.Contains(out, "declared") {
		t.Errorf("the declared sibling is missing:\n%s", out)
	}
	if strings.Contains(out, "0003") {
		t.Errorf("an unrelated record is in the cluster:\n%s", out)
	}
	if !strings.Contains(out, "2 members") {
		t.Errorf("want a 2-member cluster:\n%s", out)
	}
}
