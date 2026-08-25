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

// TestFailuresStop: every failure is a stopped:<reason> line and exit 2,
// never a stack trace — an unknown subcommand, a records dir with no
// records, `lint` on a missing file, an unknown flag.
func TestFailuresStop(t *testing.T) {
	if code, _, errb := runCapture(t, "bogus"); code != 2 || !strings.Contains(errb, "stopped:unknown-subcommand") {
		t.Errorf("unknown subcommand: exit %d, stderr %q", code, errb)
	}
	if code, _, errb := runCapture(t, "index", "--status", "--records", t.TempDir()); code != 2 || !strings.Contains(errb, "stopped:no-records") {
		t.Errorf("empty records dir: exit %d, stderr %q", code, errb)
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

// corpusDir writes a small synthetic records dir: two Drafts rewriting one
// function without citing each other, a Final that cites one of them, and
// an Implemented record, plus a README index table that is stale for one.
func corpusDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	head := func(num, title, status string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status +
			"\n- **Profile**: standard\n- **Priority**: High\n"
	}
	ev := func(anchor string) string {
		return "\n## Critical Assumptions\n\n- **A1**: claim\n  - **Method**: Source Search\n  - **Evidence**: `" + anchor + "`\n  - **Status**: Verified\n"
	}
	files := map[string]string{
		"0001-alpha.md": head("0001", "Alpha", "Draft") + ev("uniqueid.go::checkUniqueNameIn"),
		"0002-beta.md":  head("0002", "Beta", "Draft") + ev("internal/validate/uniqueid.go::checkUniqueNameIn"),
		"0003-gamma.md": head("0003", "Gamma", "Final") + "- **Predecessors**: 0001-alpha\n" + ev("walk.go::Walk") +
			"\n## Problem Statement\n\nSee 0001-alpha A1 and 0002-beta.\n",
		"0004-delta.md": head("0004", "Delta", "Implemented") + ev("uniqueid.go::checkUniqueNameIn"),
		"README.md": "| ID | Title | Status | Priority |\n| --- | --- | --- | --- |\n" +
			"| [0001](0001-alpha.md) | Alpha | Draft | High |\n" +
			"| [0002](0002-beta.md) | Beta | Final | High |\n" +
			"| [0003](0003-gamma.md) | Gamma | Final | High |\n" +
			"| [0004](0004-delta.md) | Delta | Implemented | High |\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestIndexGraphIsDeterministic: the bare index is one document with
// records, elements, edges and derived backlinks, and two builds are the
// same bytes.
func TestIndexGraphIsDeterministic(t *testing.T) {
	dir := corpusDir(t)
	code, a, errb := runCapture(t, "index", "--json", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	_, b, _ := runCapture(t, "index", "--json", "--records", dir)
	if a != b {
		t.Error("two builds of the graph differ")
	}
	var g struct {
		Schema    string           `json:"schema"`
		Records   []map[string]any `json:"records"`
		Elements  []map[string]any `json:"elements"`
		Edges     []map[string]any `json:"edges"`
		Backlinks map[string]any   `json:"backlinks"`
	}
	if err := json.Unmarshal([]byte(a), &g); err != nil {
		t.Fatal(err)
	}
	if g.Schema != schemaVersion || len(g.Records) != 4 || len(g.Elements) == 0 || len(g.Edges) == 0 || len(g.Backlinks) == 0 {
		t.Errorf("graph is missing a facet: schema %q, %d records, %d elements, %d edges, %d backlink targets",
			g.Schema, len(g.Records), len(g.Elements), len(g.Edges), len(g.Backlinks))
	}
	code, text, _ := runCapture(t, "index", "--records", dir)
	if code != 0 || !strings.Contains(text, "total 4 records") {
		t.Errorf("text form:\n%s", text)
	}
}

// TestIndexInFlight: the worklist is Draft and Final, never Implemented.
func TestIndexInFlight(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "index", "--in-flight", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if strings.Contains(out, "0004") || !strings.Contains(out, "total 3 in flight over 4 records") {
		t.Errorf("worklist wrong:\n%s", out)
	}
	code, out, _ = runCapture(t, "index", "--status", "--records", dir)
	if code != 0 || !strings.Contains(out, "Draft          2  0001 0002") || !strings.Contains(out, "Implemented    1  0004") {
		t.Errorf("status groups wrong:\n%s", out)
	}
}

// TestIndexBacklinksToTarget: `--backlinks=NNNN[:elem]` is the impact
// query — typed edges and mentions, for one target or a whole record.
func TestIndexBacklinksToTarget(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "index", "--backlinks=0001:A1", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0001:A1: 0 typed, 1 mentions") {
		t.Errorf("element target:\n%s", out)
	}
	code, out, _ = runCapture(t, "index", "--backlinks=0001", "--records", dir)
	if code != 0 || !strings.Contains(out, "predecessor") || !strings.Contains(out, "0001: 1 typed, 1 mentions") {
		t.Errorf("record target should gather the record and its elements:\n%s", out)
	}
	if code, _, _ := runCapture(t, "index", "--backlinks=nope", "--records", dir); code != 2 {
		t.Errorf("a non-record target should stop with usage, got exit %d", code)
	}
	code, out, _ = runCapture(t, "index", "--backlinks", "--records", dir)
	if code != 0 || !strings.Contains(out, "0001") {
		t.Errorf("the bare form is still the whole table:\n%s", out)
	}
}

// TestIndexAnchorIntersect: the after-propose scan fires on the uncited
// pair, and only over in-flight records unless --all.
func TestIndexAnchorIntersect(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "index", "--anchor-intersect", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.HasPrefix(out, "0001 0002 UNCITED 1 shared: internal/validate/uniqueid.go::checkUniqueNameIn") {
		t.Errorf("the uncited pair should lead:\n%s", out)
	}
	if strings.Contains(out, "0004") || !strings.Contains(out, "1 with no cross-citation") {
		t.Errorf("implemented records are out of scope:\n%s", out)
	}
	code, out, _ = runCapture(t, "index", "--anchor-intersect", "--all", "--records", dir)
	if code != 0 || !strings.Contains(out, "0004") {
		t.Errorf("--all should include the implemented record:\n%s", out)
	}
}

// TestIndexReadmeDrift: the index table is checked against the records
// and the one stale row is named with its line.
func TestIndexReadmeDrift(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "index", "--readme", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, `0002 status          readme "Final"  record "Draft"`) || !strings.Contains(out, "total 1 drift over 4 rows, 4 records") {
		t.Errorf("drift report:\n%s", out)
	}
	if code, _, _ := runCapture(t, "index", "--readme="+filepath.Join(dir, "missing.md"), "--records", dir); code != 2 {
		t.Errorf("a missing README should stop, got exit %d", code)
	}
}

// TestIndexClusterCandidateTier: two in-flight records that only mention
// each other are reported as candidates, never as asserted members, and
// a one-way mention is nothing.
func TestIndexClusterCandidateTier(t *testing.T) {
	dir := t.TempDir()
	head := func(num, title, status string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status + "\n- **Profile**: standard\n"
	}
	for name, body := range map[string]string{
		"0001-alpha.md": head("0001", "Alpha", "Final") + "\n## Problem Statement\n\nSee 0002-beta and 0003-gamma and 0004-delta.\n",
		"0002-beta.md":  head("0002", "Beta", "Final") + "\n## Problem Statement\n\nSee 0001-alpha.\n",
		"0003-gamma.md": head("0003", "Gamma", "Final") + "\n## Problem Statement\n\nSynthetic.\n",
		"0004-delta.md": head("0004", "Delta", "Implemented") + "\n## Problem Statement\n\nSee 0001-alpha.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errb := runCapture(t, "index", "--cluster-of", "0001", "--json", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got struct {
		Cluster []struct {
			Record, Relation, Status string
			Candidate                bool
		}
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Cluster) != 2 || got.Cluster[1].Record != "0002" || got.Cluster[1].Relation != "mutual-mentions" || !got.Cluster[1].Candidate || got.Cluster[1].Status != "Final" {
		t.Errorf("want the seed and one Final candidate with its status:\n%s", out)
	}
}

// TestSourceAnchorsResolveWithoutRecordsDir: `--repo` is its own
// authority. Source-anchor resolution greps the repo and never consults
// the records map, so a missing or unwalkable `--records` must not take
// the symbol verdicts down with it. The failure this pins: `resolveEdges`
// used to return early in both cases, skipping the resolver entirely, so
// a wrong `--records` silently suppressed resolution `--repo` alone would
// have reached — a skipped check reading as an unchecked one.
func TestSourceAnchorsResolveWithoutRecordsDir(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "widget.go"),
		[]byte("package widget\n\nfunc KnownSymbol() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	rec := filepath.Join(dir, "0001-anchored.md")
	body := "# Recommendation 0001: Anchored\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSee `widget.go::KnownSymbol` for the site.\n"
	if err := os.WriteFile(rec, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	// The symbol edge must read `true` in every one of these, because in
	// every one of them a valid --repo was supplied.
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"repo only, record by path", []string{"inspect", "--json", "--repo", repo, rec}},
		{"repo plus records", []string{"inspect", "--json", "--repo", repo, "--records", dir, "0001"}},
		{"repo plus unwalkable records", []string{"inspect", "--json", "--repo", repo,
			"--records", filepath.Join(dir, "nonexistent"), rec}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errb := runCapture(t, tc.args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, errb)
			}
			var env struct {
				Edges []struct {
					Kind     string `json:"kind"`
					To       string `json:"to"`
					Resolved *bool  `json:"resolved"`
				} `json:"edges"`
			}
			if err := json.Unmarshal([]byte(out), &env); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			var seen int
			for _, e := range env.Edges {
				if !strings.Contains(e.To, "::") {
					continue
				}
				seen++
				if e.Resolved == nil {
					t.Errorf("source anchor %q reads absent, but --repo was given", e.To)
				} else if !*e.Resolved {
					t.Errorf("source anchor %q reads false; the symbol is in the repo", e.To)
				}
			}
			if seen == 0 {
				t.Fatal("no source-anchor edge was projected; the fixture cannot pin the behaviour")
			}
		})
	}

	// The element half still goes absent with no corpus: that is the
	// three-valued contract, not a regression.
	code, out, errb := runCapture(t, "inspect", "--json", "--repo", repo, rec)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "KnownSymbol") {
		t.Errorf("projection lost the anchor:\n%s", out)
	}
}
