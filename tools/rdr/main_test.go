package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

func fixturePath(name string) string { return filepath.Join("testdata", name) }

func runCapture(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	// run() binds --records/--repo into the package-level flagBound map so
	// every OTHER envOrSeam reader sees them too (facts.go's roots, this
	// binary's own $RDR_RECORDS fallback) — a real process only ever calls
	// run() once, but a test binary calls it hundreds of times, and a bind
	// this test made must not answer for the next one, which may resolve
	// the same var through the marker or the environment instead.
	t.Cleanup(func() {
		flagBoundMu.Lock()
		flagBound = map[string]string{}
		flagBoundMu.Unlock()
	})
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// TestSelectRoundTripsToBytes: --select <id> prints exactly the record's
// lines for that element.
func TestSelectRoundTripsToBytes(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", "--select", "0004:A2", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	raw, _ := os.ReadFile(fixturePath("current-shape.md"))
	lines := strings.Split(string(raw), "\n")
	want := strings.Join(lines[43:50], "\n") + "\n" // A2 is lines 44-50
	if out != want {
		t.Errorf("--select output:\n%s\nwant:\n%s", out, want)
	}
	if !strings.HasPrefix(out, "- **A2 [") {
		t.Errorf("selection does not start on the assumption bullet")
	}

	code, _, errb = runCapture(t, "inspect", "--select", "0004:C9", fixturePath("current-shape.md"))
	if code != 2 || !strings.Contains(errb, "stopped:no-such-element") {
		t.Errorf("missing element: exit %d, stderr %q", code, errb)
	}
}

// TestSelectMissWithNoHintListsFacets: a --select token with no near-miss
// hint at all — nothing one edit away, nothing verbatim in the body — is
// otherwise a stop with no clue what else --select takes. It gets a
// "facets:" clause naming the named facets, on the same line; a token
// that DOES get a near-miss hint keeps today's message, unchanged.
func TestSelectMissWithNoHintListsFacets(t *testing.T) {
	code, _, errb := runCapture(t, "inspect", "--select", "ca", fixturePath("current-shape.md"))
	want := "stopped:no-such-element (ca in 0004; facets: outline elements edges warnings metadata fields anchors assumptions)\n"
	if code != 2 || errb != want {
		t.Errorf("no-hint miss should list facets: exit %d, stderr %q, want %q", code, errb, want)
	}

	// A token that resolves to a near-miss hint must not also get the
	// facets clause appended.
	code, _, errb = runCapture(t, "inspect", "--select", "0004:C9", fixturePath("current-shape.md"))
	if code != 2 || strings.Contains(errb, "facets:") {
		t.Errorf("a near-miss hint must not also carry facets: exit %d, stderr %q", code, errb)
	}
}

// TestSelectHelpNamesEveryFacet: the --select help string and the
// no-such-element "facets:" hint both come from selectFacets, so they
// cannot drift; this pins that the help text still names every one.
func TestSelectHelpNamesEveryFacet(t *testing.T) {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	declareFlags("inspect", fs)
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.PrintDefaults()
	help := buf.String()
	for _, facet := range selectFacets {
		if !strings.Contains(help, facet) {
			t.Errorf("--select help is missing facet %q:\n%s", facet, help)
		}
	}
}

// TestSelectReachesAClause: the corpus cites one grain below the contract
// (`cli/0112 §Normative Contracts L-3`) and `--select 0112:L-3` refused
// with no-such-element, so a session guessed six ids and then sed-sliced
// the 530-line C1. A clause label resolves to the clause's own lines, by
// flag and as a pasted citation; a label the record defines twice is
// refused as AMBIGUOUS with both candidates named, which is not the stop
// an absent id gets; and the clause is listed under its contract.
func TestSelectReachesAClause(t *testing.T) {
	// The dir is named `cli` so a pasted `cli/0021:L-4` is local to it.
	dir := filepath.Join(t.TempDir(), "cli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) string {
		p := filepath.Join(dir, "0021-layers.md")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	fence := "```"
	p := write("# Recommendation 0021: Layers\n\n## Metadata\n\n- **Status**: Draft\n\n#### Normative Contracts\n\n**C1**\n\n" +
		fence + "normative\nL-1  input := owned records\nL-3  ONE CAPTURE PER (table, label, anchor)\n     enforced at mint.\nL-4  tableState := {}\n" + fence + "\n")

	code, out, errb := runCapture(t, "inspect", "--select", "0021:L-3", p)
	if code != 0 || out != "L-3  ONE CAPTURE PER (table, label, anchor)\n     enforced at mint.\n" {
		t.Errorf("--select clause: exit %d, out %q, stderr %q", code, out, errb)
	}
	code, out, _ = runCapture(t, "inspect", "--select", "0021:L-1", "--select", "0021:L-4", p)
	if code != 0 || out != "L-1  input := owned records\nL-4  tableState := {}\n" {
		t.Errorf("several clauses: exit %d, out %q", code, out)
	}
	code, out, _ = runCapture(t, "inspect", "--records", dir, "cli/0021:L-4")
	if code != 0 || out != "L-4  tableState := {}\n" {
		t.Errorf("pasted citation: exit %d, out %q", code, out)
	}
	code, out, _ = runCapture(t, "inspect", "--json", "--select", "0021:L-3", p)
	var one map[string]any
	if err := json.Unmarshal([]byte(out), &one); code != 0 || err != nil || one["id"] != "0021:L-3" || one["line_start"] != float64(13) {
		t.Errorf("--json clause select: exit %d, %v, %s", code, err, out)
	}
	code, out, _ = runCapture(t, "inspect", p)
	if i, j := strings.Index(out, "0021:C1 "), strings.Index(out, "0021:L-3 "); code != 0 || i < 0 || j < i {
		t.Errorf("summary lists the clause under its contract: exit %d\n%s", code, out)
	}
	if strings.Contains(out, "clause 0/") {
		t.Errorf("derived line shows a backlog for a kind that is never derived:\n%s", out)
	}
	// The stop carries the record's own near misses — the ids one edit or
	// one kind away, never invented — so the retry needs no second read.
	code, _, errb = runCapture(t, "inspect", "--select", "0021:L-2", p)
	if code != 2 || !strings.Contains(errb, "stopped:no-such-element (0021:L-2 in 0021; near misses: 0021:L-1, 0021:L-3, 0021:L-4)") {
		t.Errorf("absent clause: exit %d, stderr %q", code, errb)
	}

	// The same label under a second contract: neither is picked.
	p = write("# Recommendation 0021: Layers\n\n## Metadata\n\n- **Status**: Draft\n\n#### Normative Contracts\n\n**C1**\n\n" +
		fence + "normative\nL-3  first\n" + fence + "\n\n**C2**\n\n" + fence + "normative\nL-3  second\n" + fence + "\n")
	code, _, errb = runCapture(t, "inspect", "--select", "0021:L-3", p)
	if code != 2 || !strings.Contains(errb, "stopped:ambiguous-element (0021:L-3 in 0021: L-3 is defined in more than one contract (0021:C1 12-12, 0021:C2 18-18)") {
		t.Errorf("duplicated label: exit %d, stderr %q", code, errb)
	}
}

// TestJSONIsDeterministic: two runs produce identical bytes, and the
// envelope carries the schema version and the project prefix.
func TestJSONIsDeterministic(t *testing.T) {
	_, a, _ := runCapture(t, "inspect", "--json", "--project", "cli", fixturePath("assumptions-nested.md"))
	_, b, _ := runCapture(t, "inspect", "--json", "--project", "cli", fixturePath("assumptions-nested.md"))
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
	raw, _ := os.ReadFile(fixturePath("current-shape.md"))
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
	clean, _ := os.ReadFile(fixturePath("current-shape.md"))
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
	code, out, errb := runCapture(t, "inspect", "--json", "--select", "edges", fixturePath("current-shape.md"))
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
	_, full, _ := runCapture(t, "inspect", "--json", fixturePath("current-shape.md"))
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
	_, a, _ := runCapture(t, "inspect", "--json", "--select", "edges", "--project", "proj", fixturePath("gate-inline.md"))
	_, b, _ := runCapture(t, "inspect", "--json", "--select", "edges", "--project", "proj", fixturePath("gate-inline.md"))
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
	// Profile marks these as carrying the later apparatus, which is what a
	// record carrying Predecessors or Cluster actually is: those fields do
	// not exist in the oldest shape, and the model is right to call them
	// foreign there.
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
	// A bare short number is the same vocabulary resolve() already reads:
	// `--cluster-of 1` seeds record 0001 rather than stopping on usage.
	code, short, errb := runCapture(t, "index", "--cluster-of", "1", "--records", dir)
	if code != 0 {
		t.Fatalf("short number refused, exit %d: %s", code, errb)
	}
	if short != out {
		t.Errorf("--cluster-of 1 and --cluster-of 0001 disagree:\nshort: %s\nfull:  %s", short, out)
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

// TestIndexStatusGroups: `--status` groups every record by its status.
// The WORKLIST half of this facet — Draft and Final, never Implemented —
// moved to `rdr status` with no argument, where the facts come with it;
// TestStatusWorklistIsTheInFlightSet is its heir and pins the same rule.
func TestIndexStatusGroups(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "index", "--status", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "Draft          2  0001 0002") || !strings.Contains(out, "Implemented    1  0004") {
		t.Errorf("status groups wrong:\n%s", out)
	}
	// The facet it absorbed is gone, not merely undocumented: a flag that
	// silently parsed and answered the graph would be worse than an error.
	if code, _, _ := runCapture(t, "index", "--in-flight", "--records", dir); code != 2 {
		t.Errorf("index --in-flight should be an unknown flag, got exit %d", code)
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

// TestIndexBacklinksUnpaddedRecord: an unpadded record number in
// --backlinks answers exactly what the padded form does, both bare and
// with an :element suffix — every other verb accepts `1` for `0001`, and
// --backlinks was the one holdout.
func TestIndexBacklinksUnpaddedRecord(t *testing.T) {
	dir := corpusDir(t)

	padCode, padOut, padErr := runCapture(t, "index", "--backlinks=0001", "--records", dir)
	rawCode, rawOut, rawErr := runCapture(t, "index", "--backlinks=1", "--records", dir)
	if padCode != rawCode || padOut != rawOut || padErr != rawErr {
		t.Errorf("unpadded record --backlinks=1 diverged from --backlinks=0001:\ncode %d vs %d\nstdout %q vs %q\nstderr %q vs %q",
			rawCode, padCode, rawOut, padOut, rawErr, padErr)
	}

	padCode, padOut, padErr = runCapture(t, "index", "--backlinks=0001:A1", "--records", dir)
	rawCode, rawOut, rawErr = runCapture(t, "index", "--backlinks=1:A1", "--records", dir)
	if padCode != rawCode || padOut != rawOut || padErr != rawErr {
		t.Errorf("unpadded element --backlinks=1:A1 diverged from --backlinks=0001:A1:\ncode %d vs %d\nstdout %q vs %q\nstderr %q vs %q",
			rawCode, padCode, rawOut, padOut, rawErr, padErr)
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

// TestIndexRowJSON: one record's index row, addressed by number. The
// three answers a writer acts on differently are distinguished — a row,
// no row (exit 0, null), and no table at all (exit 2) — and the row's
// cells agree with what `--readme` reads from the same table.
func TestIndexRowJSON(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "index", "--row-json", "0002", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	// The README says Final for 0002 while the record says Draft — this
	// facet reports the TABLE, so the row's own cell is what comes back.
	// That disagreement is `--readme`'s finding, not this facet's.
	for _, want := range []string{`"record": "0002"`, `"title": "Beta"`, `"status": "Final"`, `"priority": "High"`, `"line": 4`} {
		if !strings.Contains(out, want) {
			t.Errorf("row JSON missing %s:\n%s", want, out)
		}
	}

	// A record with no row is an ANSWER, not a refusal: `readme --add` is
	// the op for it, so it exits 0 carrying a null row.
	code, out, errb = runCapture(t, "index", "--row-json", "0099", "--records", dir)
	if code != 0 {
		t.Fatalf("a record with no row should exit 0, got %d: %s", code, errb)
	}
	if !strings.Contains(out, `"row": null`) {
		t.Errorf("no row should read as null:\n%s", out)
	}

	// A filename works where a bare number does, so the caller may pass
	// whatever it is already holding.
	if code, out, _ := runCapture(t, "index", "--row-json", "0002-beta.md", "--records", dir); code != 0 || !strings.Contains(out, `"status": "Final"`) {
		t.Errorf("a filename should address the same row: exit %d\n%s", code, out)
	}

	// `--readme=PATH` names the table here as it does for the drift check,
	// so a read-back can read the same file the write edited. Given both,
	// --row-json is the facet: it dispatches first.
	if code, out, _ := runCapture(t, "index", "--row-json", "0002", "--readme="+filepath.Join(dir, "README.md"), "--records", dir); code != 0 || !strings.Contains(out, `"status": "Final"`) {
		t.Errorf("--readme=PATH should name the table: exit %d\n%s", code, out)
	}

	// No index table is "nothing looked" — a refusal, never a null row,
	// because a writer must not read it as "no row, go add one".
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, "README.md"), []byte("# Records\n\nprose only.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCapture(t, "index", "--row-json", "0002", "--records", empty); code != 2 {
		t.Errorf("a README with no index table should stop, got exit %d", code)
	}
	if code, _, _ := runCapture(t, "index", "--row-json", "nope", "--records", dir); code != 2 {
		t.Errorf("a non-record argument should stop, got exit %d", code)
	}
}

// TestIndexRowJSONDuplicate: two rows for one record refuse rather than
// resolving arbitrarily — a writer told to edit "the" row would pick one
// and silently leave the other stale.
func TestIndexRowJSONDuplicate(t *testing.T) {
	dir := t.TempDir()
	body := "| ID | Title | Status | Priority |\n| --- | --- | --- | --- |\n" +
		"| [0001](0001-alpha.md) | Alpha | Draft | High |\n" +
		"| [0001](0001-alpha.md) | Alpha | Final | High |\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runCapture(t, "index", "--row-json", "0001", "--records", dir)
	if code != 2 || !strings.Contains(errb, "stopped:duplicate-row") {
		t.Errorf("a duplicated row should stop: exit %d\n%s", code, errb)
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

// TestIndexClusterClosure: `--closure` is ClusterOf to a fixpoint. A
// declares B, B declares C, C rests on D as peer evidence, D and E only
// mention each other, and E mentions G mutually: the closure from A is
// {A,B,C,D} with E a candidate that is never expanded (G stays out).
// `--final-unimplemented` then drops the Implemented member and the
// COMPLETE-capsule member, keeps the one with no capsule, and lists the
// candidate as dropped with why; a solo seed answers itself alone.
func TestIndexClusterClosure(t *testing.T) {
	dir := t.TempDir()
	head := func(num, title, status string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status + "\n- **Profile**: small\n"
	}
	peer := "\n## Critical Assumptions\n\n- **A1 [claim]**\n  - **Status**: Verified\n  - **Method**: Peer RDR\n  - **Evidence**: 0004-delta A1\n"
	for name, body := range map[string]string{
		"0001-alpha.md":   head("0001", "Alpha", "Final") + "- **Cluster**: 0002-beta\n\n## Problem Statement\n\nSynthetic.\n",
		"0002-beta.md":    head("0002", "Beta", "Final") + "- **Cluster**: 0003-gamma\n\n## Problem Statement\n\nSynthetic.\n",
		"0003-gamma.md":   head("0003", "Gamma", "Implemented") + peer + "\n## Problem Statement\n\nSynthetic.\n",
		"0004-delta.md":   head("0004", "Delta", "Final") + "\n## Critical Assumptions\n\n- **A1 [claim]**\n  - **Status**: Verified\n\n## Problem Statement\n\nSee 0005-epsilon.\n",
		"0005-epsilon.md": head("0005", "Epsilon", "Final") + "\n## Problem Statement\n\nSee 0004-delta and 0007-eta.\n",
		"0006-zeta.md":    head("0006", "Zeta", "Final") + "\n## Problem Statement\n\nSynthetic.\n",
		"0007-eta.md":     head("0007", "Eta", "Final") + "\n## Problem Statement\n\nSee 0005-epsilon.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "0002-beta", "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002-beta", "artifacts", "status.md"), []byte("phase: done\nstate: COMPLETE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RDR_RECORDS", dir)
	table := factTableForTest(t)

	type member struct {
		Record, Relation, Via, Status, ImplState string
		Candidate                                bool
	}
	type drop struct{ Record, Status, ImplState, Why string }
	read := func(args ...string) (cluster []member, dropped []drop) {
		t.Helper()
		code, out, errb := runCapture(t, append([]string{"index", "--json", "--records", dir, "--facts", table}, args...)...)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errb)
		}
		var got struct {
			Closure    bool
			Cluster    []member
			OutOfScope []drop `json:"out_of_scope"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		return got.Cluster, got.OutOfScope
	}
	ids := func(ms []member) string {
		var s []string
		for _, m := range ms {
			s = append(s, m.Record)
		}
		return strings.Join(s, " ")
	}

	cluster, _ := read("--cluster-of", "0001", "--closure")
	if ids(cluster) != "0001 0002 0003 0004 0005" {
		t.Fatalf("closure from 0001 = %q, want A B C D and the candidate E", ids(cluster))
	}
	for _, m := range cluster {
		switch m.Record {
		case "0003":
			if m.Via != "0002" || m.Relation != "declared" {
				t.Errorf("C reached %q via %q, want declared via 0002", m.Relation, m.Via)
			}
		case "0004":
			if m.Via != "0003" || m.Relation != "peer-evidence" {
				t.Errorf("D reached %q via %q, want peer-evidence via 0003", m.Relation, m.Via)
			}
		case "0005":
			if !m.Candidate || m.Via != "0004" {
				t.Errorf("E should be a candidate via 0004: %+v", m)
			}
		}
	}
	// One hop from A is still one hop: C is not there without --closure.
	if one, _ := read("--cluster-of", "0001"); ids(one) != "0001 0002" {
		t.Errorf("one hop = %q, want the seed and B", ids(one))
	}
	// Two seeds are a closure without the flag being said.
	if two, _ := read("--cluster-of", "0001,0006"); ids(two) != "0001 0006 0002 0003 0004 0005" {
		t.Errorf("two seeds = %q", ids(two))
	}

	cluster, dropped := read("--cluster-of", "0001", "--closure", "--final-unimplemented")
	if ids(cluster) != "0001 0004" {
		t.Errorf("scoped = %q, want the seed and D (no capsule stays in scope)", ids(cluster))
	}
	why := map[string]string{}
	for _, d := range dropped {
		why[d.Record] = d.Why
	}
	if why["0002"] != "impl-complete" || why["0003"] != "not-final" || why["0005"] != "candidate" || len(dropped) != 3 {
		t.Errorf("out_of_scope = %+v", dropped)
	}

	if solo, dropped := read("--cluster-of", "0006", "--closure", "--final-unimplemented"); ids(solo) != "0006" || len(dropped) != 0 {
		t.Errorf("solo seed = %q dropped %+v", ids(solo), dropped)
	}
}

// TestIndexTopo: build order over predecessor edges. A chain, a Priority
// tie and a number tie order as Kahn with the declared tiebreak; a
// predecessor cycle lands in `cycles` and the rest still orders; a
// predecessor outside the set is `external`; a name the corpus lacks is
// `skipped`; an empty set is `order: []`.
func TestIndexTopo(t *testing.T) {
	dir := t.TempDir()
	rec := func(num, title, priority, preds string) string {
		body := "# Recommendation " + num + ": " + title + "\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Draft\n- **Priority**: " + priority + "\n"
		if preds != "" {
			body += "- **Predecessors**: " + preds + "\n"
		}
		return body + "\n## Problem Statement\n\nSynthetic.\n"
	}
	for name, body := range map[string]string{
		"0001-a.md": rec("0001", "A", "High", ""),
		"0002-b.md": rec("0002", "B", "Low", "0001-a"),
		"0003-c.md": rec("0003", "C", "High", "0001-a"),
		"0004-d.md": rec("0004", "D", "Medium", ""),
		"0005-e.md": rec("0005", "E", "High", "0006-f"),
		"0006-f.md": rec("0006", "F", "High", "0005-e"),
		"0007-g.md": rec("0007", "G", "", "0009-x"),
		"0008-h.md": rec("0008", "H", "High", ""),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	type got struct {
		Order []struct {
			Record, Priority string
			After            []string
		}
		Cycles   [][]string
		External []struct{ Record, Predecessor string }
		Skipped  []string
	}
	read := func(args ...string) got {
		t.Helper()
		code, out, errb := runCapture(t, append([]string{"index", "--json", "--records", dir}, args...)...)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errb)
		}
		var g got
		if err := json.Unmarshal([]byte(out), &g); err != nil {
			t.Fatal(err)
		}
		return g
	}
	order := func(g got) string {
		var s []string
		for _, r := range g.Order {
			s = append(s, r.Record)
		}
		return strings.Join(s, " ")
	}

	g := read("--topo")
	if order(g) != "0001 0003 0008 0004 0002 0007" {
		t.Errorf("order = %q: want A first, then C (High, depends on A) before H (High, later number), then Medium, Low, unset", order(g))
	}
	if len(g.Cycles) != 1 || strings.Join(g.Cycles[0], " ") != "0005 0006" {
		t.Errorf("cycles = %v, want E and F", g.Cycles)
	}
	if len(g.External) != 1 || g.External[0].Record != "0007" || g.External[0].Predecessor != "0009" {
		t.Errorf("external = %+v", g.External)
	}
	if len(g.Order) > 4 && strings.Join(g.Order[4].After, ",") != "0001" {
		t.Errorf("B's after = %v", g.Order[4].After)
	}
	g = read("--topo=0002,0001,0009")
	if order(g) != "0001 0002" || len(g.Skipped) != 1 || g.Skipped[0] != "0009" {
		t.Errorf("named set: order %q skipped %v", order(g), g.Skipped)
	}
	if g = read("--topo=0009"); len(g.Order) != 0 || g.Order == nil {
		t.Errorf("empty set: order %v", g.Order)
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

// TestUsageLogIsOffByDefault: the projector's contract is that it never
// writes. Logging is opt-in via $RDR_USAGE_LOG, and with the var unset a
// full run must leave nothing behind.
func TestUsageLogIsOffByDefault(t *testing.T) {
	t.Setenv(usageEnvVar, "")
	dir := t.TempDir()
	code, _, errb := runCapture(t, "inspect", "--json", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("logging is off, yet %d files were written", len(entries))
	}
}

// TestUsageLogRecordsInvocations: one JSONL line per invocation, carrying
// what was asked and what it cost — the measurement the consumer pass had
// to estimate from byte counts. Appending, never truncating: a second run
// must not lose the first.
func TestUsageLogRecordsInvocations(t *testing.T) {
	log := filepath.Join(t.TempDir(), "nested", "usage.jsonl")
	t.Setenv(usageEnvVar, log)

	code, out, errb := runCapture(t, "inspect", "--json", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	emitted := len(out)
	if code, _, _ = runCapture(t, "inspect", "--select", "outline", fixturePath("current-shape.md")); code != 0 {
		t.Fatalf("second invocation exit %d", code)
	}

	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 appended lines, got %d:\n%s", len(lines), raw)
	}

	// Decode to a map: the wire shape is the contract, and the record
	// type carries no JSON tags precisely so `payload` is the only place
	// a field's spelling is decided.
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 1 is not JSON: %v", err)
	}
	if first["cmd"] != "inspect" || first["facet"] != "json" {
		t.Errorf("cmd/facet = %v/%v, want inspect/json", first["cmd"], first["facet"])
	}
	if got := int(first["bytes_out"].(float64)); got != emitted {
		t.Errorf("bytes_out = %d, want the %d actually emitted", got, emitted)
	}
	if got := int(first["exit"].(float64)); got != 0 {
		t.Errorf("exit = %d, want 0", got)
	}
	ts, _ := first["ts"].(string)
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("ts %q is not RFC3339: %v", ts, err)
	}
	if _, ok := first["elapsed_ms"]; !ok {
		t.Error("no elapsed_ms")
	}
	// Sorted keys are what make the log diffable across runs.
	if !sort.StringsAreSorted(jsonKeys(t, lines[0])) {
		t.Errorf("keys are not sorted: %s", lines[0])
	}

	var second map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("line 2 is not JSON: %v", err)
	}
	if second["facet"] != "select:outline" {
		t.Errorf("facet = %v, want select:outline", second["facet"])
	}
}

// TestUsageLogSurvivesAFailedRun: a stopped invocation is exactly the one
// a cost review must see — it was paid for and returned nothing. The log
// records the non-zero exit rather than dropping the line.
func TestUsageLogSurvivesAFailedRun(t *testing.T) {
	log := filepath.Join(t.TempDir(), "usage.jsonl")
	t.Setenv(usageEnvVar, log)

	code, _, _ := runCapture(t, "inspect", "--select", "0004:C9", fixturePath("current-shape.md"))
	if code != 2 {
		t.Fatalf("want exit 2 from an absent element, got %d", code)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("a failed run wrote no line: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &rec); err != nil {
		t.Fatal(err)
	}
	if got := int(rec["exit"].(float64)); got != 2 {
		t.Errorf("exit = %d, want 2", got)
	}
}

// TestUsageLogFailureNeverBreaksAProjection: measurement is subordinate.
// An unwritable log path must not change the answer or the exit code.
func TestUsageLogFailureNeverBreaksAProjection(t *testing.T) {
	t.Setenv(usageEnvVar, filepath.Join(fixturePath("current-shape.md"), "cannot", "log.jsonl"))
	code, out, errb := runCapture(t, "inspect", "--json", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("a broken log path changed the exit code: %d (%s)", code, errb)
	}
	if !strings.Contains(out, "\"outline\"") {
		t.Error("a broken log path changed the projection")
	}
}

// jsonKeys returns one JSON object's keys in the order they appear on the
// wire, so a test can assert they are sorted.
func jsonKeys(t *testing.T, line string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(line))
	if _, err := dec.Token(); err != nil { // opening brace
		t.Fatal(err)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		k, ok := tok.(string)
		if !ok {
			t.Fatalf("non-string key %v", tok)
		}
		keys = append(keys, k)
		var discard any
		if err := dec.Decode(&discard); err != nil {
			t.Fatal(err)
		}
	}
	return keys
}

// TestBareRecordNumberResolves: the flow's vocabulary is a record number,
// and a caller who types `3` means 0003. Before this, only the shell
// helpers zero-padded, so a direct call fell through to the path branch
// and failed with `open 3: no such file` — an error naming a file nobody
// asked for, which cost a turn to diagnose and a turn to retry.
func TestBareRecordNumberResolves(t *testing.T) {
	dir := t.TempDir()
	body := "# Recommendation 0003: Short\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(dir, "0003-short.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// 0070 exists too: the octal trap that once resolved `0106` to `0070`
	// must not reappear when the padding moves into Go.
	if err := os.WriteFile(filepath.Join(dir, "0070-seventy.md"),
		[]byte(strings.Replace(body, "0003: Short", "0070: Seventy", 1)), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ arg, want string }{
		{"3", "0003"}, {"03", "0003"}, {"003", "0003"}, {"0003", "0003"},
		{"70", "0070"}, {"070", "0070"}, {"0070", "0070"},
	} {
		code, out, errb := runCapture(t, "inspect", "--json", "--records", dir, tc.arg)
		if code != 0 {
			t.Errorf("%q: exit %d: %s", tc.arg, code, errb)
			continue
		}
		var env struct {
			Record string `json:"record"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("%q: %v", tc.arg, err)
		}
		if env.Record != tc.want {
			t.Errorf("%q resolved to %s, want %s", tc.arg, env.Record, tc.want)
		}
	}

	// A short number naming no record still says so, and says it about
	// the record — not about a file called "9".
	code, _, errb := runCapture(t, "inspect", "--json", "--records", dir, "9")
	if code != 2 || !strings.Contains(errb, "no-such-record") {
		t.Errorf("missing record: exit %d, stderr %q", code, errb)
	}
}

// TestRelativeRecordsDirResolvesAgainstTheMarker: a stage prompt runs from
// whatever directory the harness was in. A relative --records correct in
// one cwd named nothing in another, and the tool blamed the record
// (`no-such-record`) for a directory it never read — a wasted turn, then a
// `cd` to work around it. $RDR_RECORDS knows where records live.
func TestRelativeRecordsDirResolvesAgainstTheMarker(t *testing.T) {
	root := t.TempDir()
	recs := filepath.Join(root, "rdr", "cli")
	if err := os.MkdirAll(recs, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Recommendation 0100: Rel\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(recs, "0100-rel.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	// cwd is somewhere else entirely; the relative path resolves from the
	// marker, not from here.
	t.Chdir(t.TempDir())
	t.Setenv("RDR_RECORDS", recs)

	for _, arg := range []string{"rdr/cli", "cli", ""} {
		args := []string{"inspect", "--json", "0100"}
		if arg != "" {
			args = []string{"inspect", "--json", "--records", arg, "0100"}
		}
		code, out, errb := runCapture(t, args...)
		if code != 0 {
			t.Errorf("--records %q: exit %d: %s", arg, code, errb)
			continue
		}
		if !strings.Contains(out, `"record": "0100"`) {
			t.Errorf("--records %q did not reach the record", arg)
		}
	}

	// With no marker to fall back on, an unusable relative path is still
	// an honest failure — the rescue never invents a corpus.
	t.Setenv("RDR_RECORDS", "")
	code, _, errb := runCapture(t, "inspect", "--json", "--records", "nowhere/at/all", "0100")
	if code != 2 || !strings.Contains(errb, "no-such-record") {
		t.Errorf("unrescuable path: exit %d, stderr %q", code, errb)
	}
}

// TestAbsoluteRecordsDirIsObeyedVerbatim: the rescue must never override
// an explicit absolute path, even when $RDR_RECORDS names somewhere else.
// A caller who spells out a directory means that directory.
func TestAbsoluteRecordsDirIsObeyedVerbatim(t *testing.T) {
	want := t.TempDir()
	other := t.TempDir()
	body := "# Recommendation 0100: Here\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(other, "0100-elsewhere.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RDR_RECORDS", other)

	// `want` is empty, so an absolute --records pointing at it must fail
	// rather than quietly reading `other`.
	code, _, errb := runCapture(t, "inspect", "--json", "--records", want, "0100")
	if code != 2 || !strings.Contains(errb, "no-such-record") {
		t.Errorf("absolute --records was overridden by the env: exit %d, stderr %q", code, errb)
	}
}

// TestFilterProjectsOnlyTheNamedKeys: the envelope is lopsided — on a
// large record `elements` and `edges` are ~80% of it — so a caller who
// needs status and counts used to pay the whole thing, or spend two
// invocations to avoid it. In an agent loop an invocation is a TURN, and
// a turn re-sends the conversation, so the second call costs far more
// than the bytes it saves.
func TestFilterProjectsOnlyTheNamedKeys(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", "--json",
		"--filter", "metadata,counts", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Asked for, plus the identity keys that come unasked.
	for _, k := range []string{"metadata", "counts", "schema", "record", "path"} {
		if _, ok := got[k]; !ok {
			t.Errorf("filtered envelope is missing %q", k)
		}
	}
	// The expensive facets must be gone — that is the entire point.
	for _, k := range []string{"elements", "edges", "outline", "warnings", "fields"} {
		if _, ok := got[k]; ok {
			t.Errorf("filtered envelope still carries %q", k)
		}
	}

	full := func() int {
		_, o, _ := runCapture(t, "inspect", "--json", fixturePath("current-shape.md"))
		return len(o)
	}()
	if len(out) >= full {
		t.Errorf("filter saved nothing: %d filtered vs %d full", len(out), full)
	}
}

// TestFilterRejectsAnUnknownFacet: a typo must stop, never return an
// empty projection. A consumer handed `{}` for `--filter elments` would
// conclude the record has no elements — a skipped check reading as a
// passed one, which is the failure this flow exists to prevent.
func TestFilterRejectsAnUnknownFacet(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", "--json",
		"--filter", "elments", fixturePath("current-shape.md"))
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(errb, "no-such-facet") {
		t.Errorf("stderr %q does not name the failure", errb)
	}
	// The error lists what could have been asked for instead, so the fix
	// does not cost another turn to discover.
	if !strings.Contains(errb, "elements") {
		t.Errorf("stderr %q does not list the valid keys", errb)
	}
	// The stop reaches stdout too: sessions habitually 2>/dev/null a read
	// they expect to succeed, and a refusal only stderr carries reads as
	// an empty success.
	if !strings.Contains(out, "stopped:no-such-facet") {
		t.Errorf("stdout %q does not carry the stop", out)
	}
	if strings.Contains(out, "\"schema\"") {
		t.Errorf("a rejected filter still emitted an envelope: %q", out)
	}
}

// TestSelectReachesEveryDocumentedFacet: `metadata`, `fields` and
// `anchors` are documented facets that fell through to the element-id
// branch and failed, so a caller following the docs got
// `stopped:no-such-element` and spent a turn finding out why.
func TestSelectReachesEveryDocumentedFacet(t *testing.T) {
	for _, facet := range []string{"outline", "elements", "edges", "warnings", "metadata", "fields"} {
		code, out, errb := runCapture(t, "inspect", "--json", "--select", facet, fixturePath("current-shape.md"))
		if code != 0 {
			t.Errorf("--select %s: exit %d: %s", facet, code, errb)
			continue
		}
		if strings.TrimSpace(out) == "" {
			t.Errorf("--select %s produced nothing", facet)
		}
	}
}

// TestSlugResolvesLikeANumber: the flow binds `$RDR_SLUG` beside
// `$RDR_PATH` and passes slugs around, but a bare slug was read as a
// cwd-relative path and failed with `no such file` — the same
// misdirection a bare number gave, and the same wasted turn.
func TestSlugResolvesLikeANumber(t *testing.T) {
	dir := t.TempDir()
	body := "# Recommendation 0142: Slug\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(dir, "0142-named-thing.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir()) // a cwd with nothing in it

	for _, arg := range []string{"142", "0142", "0142-named-thing"} {
		code, out, errb := runCapture(t, "inspect", "--json", "--filter", "path", "--records", dir, arg)
		if code != 0 {
			t.Errorf("%q: exit %d: %s", arg, code, errb)
			continue
		}
		if !strings.Contains(out, "0142-named-thing.md") {
			t.Errorf("%q did not reach the record: %s", arg, out)
		}
	}

	// A slug naming no record is still an honest failure.
	code, _, _ := runCapture(t, "inspect", "--json", "--records", dir, "0142-not-this-one")
	if code != 2 {
		t.Errorf("a slug naming nothing exited %d, want 2", code)
	}
}

// TestRecordsDirFailureSaysWhereItLooked: `docs/rdr holds no NNNN-*.md`
// names no directory a reader can go and check — it is true of a hundred
// places, and it does not say whether the cwd, the marker, or a join of
// the two was read. A user hitting it spent a turn guessing, then a turn
// re-running with absolute paths.
func TestRecordsDirFailureSaysWhereItLooked(t *testing.T) {
	marker := t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("RDR_RECORDS", marker)

	_, _, errb := runCapture(t, "index", "--status", "--records", "nope/here")
	if !strings.Contains(errb, "no-records") {
		t.Fatalf("want a no-records stop, got %q", errb)
	}
	// The directory it actually read is named absolutely.
	if !strings.Contains(errb, filepath.Join("nope", "here")) || !strings.HasPrefix(strings.TrimPrefix(errb, "stopped:no-records ("), "/") {
		t.Errorf("failure does not name an absolute directory: %q", errb)
	}
	// And every candidate is listed, so the fix does not cost a turn.
	if !strings.Contains(errb, "looked in:") {
		t.Errorf("failure does not say where it looked: %q", errb)
	}
	if !strings.Contains(errb, "$RDR_RECORDS") {
		t.Errorf("failure does not mention the marker it consulted: %q", errb)
	}
}

// TestRelativeRecordsNeverSilentlyBecomesTheMarker: the rescue resolves a
// relative path, it does not substitute for one. `--records nope/here`
// under a valid marker must FAIL — answering out of $RDR_RECORDS would
// return a real, plausible corpus for a path that names nothing, which is
// worse than an error because the answer looks right.
// factTableForTest names the shipped fact table absolutely, for a test
// that chdirs away from the repo before invoking `status`. The table's
// own $RDR_HOME lookup is covered by TestFactTablePathFindsTheShippedTable;
// here it would only be a second thing able to fail.
func factTableForTest(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "models", factTableName))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestRelativeRecordsNeverSilentlyBecomesTheMarker(t *testing.T) {
	table := factTableForTest(t)
	marker := t.TempDir()
	body := "# Recommendation 0003: M\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(marker, "0003-m.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	t.Setenv("RDR_RECORDS", marker)

	code, out, _ := runCapture(t, "index", "--status", "--records", "nope/here")
	if code == 0 {
		t.Errorf("a --records naming nothing returned the marker's corpus:\n%s", out)
	}

	// With no --records at all, the marker IS the answer — that is the
	// documented default, not a guess.
	code, out, errb := runCapture(t, "status", "--facts", table)
	if code != 0 {
		t.Fatalf("bare status exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0003-m") {
		t.Errorf("the marker default did not resolve:\n%s", out)
	}
}

// TestRecordsDirResolvesFromInsideItself: the shape that produced the
// doubled `…/docs/rdr/docs/rdr` in the wild — $RDR_RECORDS ends with the
// same relative path the caller passed, and the cwd is already inside it.
func TestRecordsDirResolvesFromInsideItself(t *testing.T) {
	table := factTableForTest(t)
	root := t.TempDir()
	recs := filepath.Join(root, "docs", "rdr")
	if err := os.MkdirAll(recs, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Recommendation 0003: Nested\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(recs, "0003-nested.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RDR_RECORDS", recs)
	t.Chdir(recs) // already inside the records dir

	code, out, errb := runCapture(t, "status", "--records", "docs/rdr", "--facts", table)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0003-nested") {
		t.Errorf("did not resolve from inside the records dir:\n%s", out)
	}
}

// TestInspectResolvesOnlyWhenEdgesShow: resolution is the only thing
// inspect does that reads beyond the record — a corpus scan and a repo
// walk — and only edges[] can show its verdict. A facet that cannot show
// `resolved` must not pay for it; a facet that can must still get it.
func TestInspectResolvesOnlyWhenEdgesShow(t *testing.T) {
	dir := t.TempDir()
	rec := filepath.Join(dir, "0001-anchored.md")
	body := "# Recommendation 0001: Anchored\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSee `widget.go::KnownSymbol` for the site.\n"
	if err := os.WriteFile(rec, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// An unwalkable --records makes resolveEdges announce itself on
	// stderr, so the note is the witness for whether resolution ran.
	bad := filepath.Join(dir, "nonexistent")
	for _, tc := range []struct {
		name string
		args []string
		runs bool
	}{
		{"filter counts", []string{"--json", "--filter", "counts"}, false},
		{"filter metadata,counts", []string{"--json", "--filter", "metadata,counts"}, false},
		{"select outline", []string{"--select", "outline"}, false},
		{"select element", []string{"--select", "0001:P"}, false},
		{"text summary", nil, false},
		{"filter edges", []string{"--json", "--filter", "metadata,edges"}, true},
		{"select edges", []string{"--select", "edges"}, true},
		{"whole envelope", []string{"--json"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"inspect", "--records", bad}, tc.args...)
			_, _, errb := runCapture(t, append(args, rec)...)
			if got := strings.Contains(errb, "note:unresolved-edges"); got != tc.runs {
				t.Errorf("resolution ran = %v, want %v (stderr: %q)", got, tc.runs, errb)
			}
		})
	}
}

// TestUsageLogFacetCarriesTheFilter: `--json` and `--json --filter counts`
// differ by two orders of magnitude in bytes; a log that spells both
// "json" cannot say what --filter saved.
func TestUsageLogFacetCarriesTheFilter(t *testing.T) {
	log := filepath.Join(t.TempDir(), "usage.jsonl")
	t.Setenv(usageEnvVar, log)
	if code, _, errb := runCapture(t, "inspect", "--json", "--filter", "metadata, counts", fixturePath("current-shape.md")); code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	var line map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &line); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if line["facet"] != "json:metadata,counts" {
		t.Errorf("facet = %v, want json:metadata,counts", line["facet"])
	}
}

// TestReceiptVouchesOnlyForALintAfterTheLastWrite: the receipt is the
// usage log's own lint line, and only one at or after the record's mtime
// counts — a gate closed on stale lint is the case this exists to catch.
func TestReceiptVouchesOnlyForALintAfterTheLastWrite(t *testing.T) {
	dir := t.TempDir()
	rec := filepath.Join(dir, "0007-receipt.md")
	body := "# Recommendation 0007: Receipt\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Draft\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(rec, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "usage.jsonl")

	t.Setenv(usageEnvVar, "off") // explicit off; usageMarkerFallback keeps the marker out under go test anyway
	if code, _, errb := runCapture(t, "receipt", "--records", dir, "7"); code != 2 || !strings.Contains(errb, "no-usage-log") {
		t.Fatalf("no log: exit %d %q, want 2 stopped:no-usage-log", code, errb)
	}

	t.Setenv(usageEnvVar, log)
	if code, _, errb := runCapture(t, "receipt", "--records", dir, "7"); code != 1 || !strings.Contains(errb, "no-lint-receipt") || !strings.Contains(errb, "last lint never") {
		t.Fatalf("never linted: exit %d %q, want 1 stopped:no-lint-receipt … last lint never", code, errb)
	}

	// A lint by number, then by path, then of the whole dir: each spelling covers the record.
	for _, target := range [][]string{{"7"}, {rec}, {}} {
		if code, _, errb := runCapture(t, append([]string{"lint", "--records", dir}, target...)...); code > 1 {
			t.Fatalf("lint %v: exit %d %s", target, code, errb)
		}
		code, out, errb := runCapture(t, "receipt", "--records", dir, "0007")
		if code != 0 {
			t.Fatalf("after lint %v: exit %d %s", target, code, errb)
		}
		if !strings.Contains(out, `"cmd":"lint"`) {
			t.Errorf("receipt line is not the lint line: %s", out)
		}
	}

	// The record is written again, two seconds after that lint: stale.
	var last struct {
		TS string `json:"ts"`
	}
	raw, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], `"cmd":"lint"`) {
			if err := json.Unmarshal([]byte(lines[i]), &last); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	linted, _ := time.Parse(time.RFC3339, last.TS)
	if err := os.WriteFile(rec, []byte(body+"\nEdited.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(rec, linted.Add(2*time.Second), linted.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if code, _, errb := runCapture(t, "receipt", "--records", dir, "0007"); code != 1 || !strings.Contains(errb, "last lint "+last.TS) {
		t.Fatalf("stale lint: exit %d %q, want 1 naming the last lint", code, errb)
	}
	// --since is the caller's own instant, and outranks the mtime.
	if code, _, _ := runCapture(t, "receipt", "--records", dir, "--since", linted.Add(-time.Second).Format(time.RFC3339), "0007"); code != 0 {
		t.Errorf("--since before the lint: exit %d, want 0", code)
	}
}

// TestIndexOpenJointFacet: both places a record states an open joint
// decision surface in one corpus-wide call, in-flight only by default.
func TestIndexOpenJointFacet(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	head := func(num, title, status string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status + "\n- **Profile**: standard\n\n## Problem Statement\n\nSynthetic.\n"
	}
	write("0001-line.md", head("0001", "Line", "Draft")+"\n## Finalization Gate\n\nJoint-check: fired → 0002, 0003 (home: OPEN) — shared seam\nJoint-check: fired → 0004 (home: cli/0004 §Normative Contracts) — ruled\n")
	write("0002-status.md", head("0002", "Status", "Final [joint decision → JDR 0001 §JD-18: conforming-view enforcer]"))
	write("0003-done.md", head("0003", "Done", "Implemented")+"\nJoint-check: fired → 0001 (home: OPEN) — stale\n")

	code, out, errb := runCapture(t, "index", "--open-joint", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"0001 JC1", "(home: OPEN)", "0002 status", "total 2 open joint decisions over 2 records"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "JC2") || strings.Contains(out, "0003 JC") {
		t.Errorf("a ruled check or a terminal record leaked in:\n%s", out)
	}
	code, out, _ = runCapture(t, "index", "--open-joint", "--all", "--json", "--records", dir)
	if code != 0 || !strings.Contains(out, `"record": "0003"`) || !strings.Contains(out, `"signal": "joint-check"`) {
		t.Errorf("--all --json should include the terminal record's open check:\n%s", out)
	}
}

// TestIndexCyclesFacet: the four shapes the flow cannot progress through,
// and nothing from the symmetric relations.
func TestIndexCyclesFacet(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	head := func(num, title, status, meta string) string {
		return "# Recommendation " + num + ": " + title +
			"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status + "\n- **Profile**: standard\n" + meta + "\n## Problem Statement\n\nSynthetic.\n" +
			// Every Joint-check home below names §Normative Contracts, and a
			// home is an edge the lock resolves — so the section exists.
			"\n## Normative Contracts\n\nSynthetic.\n"
	}
	// 0001 <-> 0002 override each other; 0003 <-> 0005 defer to each other's
	// Draft home; 0004 (Final) has an OPEN check and a home on a Draft, and
	// 0003's home on Final 0004 is a ruling, not a deferral; 0006/0007 cite
	// each other as peer evidence only, which is symmetric and must not appear.
	write("0001-a.md", head("0001", "A", "Draft", "- **Overrides**: cli/0002\n"))
	write("0002-b.md", head("0002", "B", "Draft", "- **Overrides**: cli/0001\n"))
	write("0003-c.md", head("0003", "C", "Draft", "")+"\nJoint-check: fired → 0004 (home: cli/0004 §Normative Contracts)\nJoint-check: fired → 0005 (home: cli/0005 §Normative Contracts)\n")
	write("0004-d.md", head("0004", "D", "Final", "")+"\nJoint-check: fired → 0005 (home: cli/0005 §Normative Contracts)\nJoint-check: fired → 0003 (home: OPEN)\n")
	write("0005-e.md", head("0005", "E", "Draft", "")+"\nJoint-check: fired → 0003 (home: cli/0003 §Normative Contracts)\n")
	write("0006-f.md", head("0006", "F", "Final", "")+"\n## Critical Assumptions\n\n- **A1** peer\n  - **Method**: Peer-RDR\n  - **Evidence**: cli/0007:A1\n")
	write("0007-g.md", head("0007", "G", "Final", "")+"\n## Critical Assumptions\n\n- **A1** peer\n  - **Method**: Peer-RDR\n  - **Evidence**: cli/0006:A1\n")

	code, out, errb := runCapture(t, "index", "--cycles", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{
		"ownership-cycle     0001 0002  0001 overrides 0002; 0002 overrides 0001",
		"home-cycle          0003 0005  Joint-check homes defer to each other: 0003 → 0005; 0005 → 0003",
		"open-at-lock        0004 JC2",
		"home-ahead-of-lock  0004 JC1",
		"total 4 findings over 7 records",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "0006") || strings.Contains(out, "0007") {
		t.Errorf("a symmetric relation was reported as a cycle:\n%s", out)
	}

	// lint sees the half of the ownership cycle its record is on, and it
	// blocks at lock.
	code, out, _ = runCapture(t, "lint", "--locking", "--records", dir, "0001")
	if code != 1 || !strings.Contains(out, "ownership:mutual") {
		t.Errorf("lint --locking 0001: exit %d, want 1 with ownership:mutual:\n%s", code, out)
	}
	if code, out, _ = runCapture(t, "lint", "--locking", "--records", dir, "0003"); code != 0 || strings.Contains(out, "ownership:mutual") {
		t.Errorf("lint --locking 0003: exit %d, want 0 and no ownership finding:\n%s", code, out)
	}
}

// TestSummaryListsSections: the text summary is the read plan for a large
// record, so it carries the outline's line ranges as well as the elements.
func TestSummaryListsSections(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "§ ") || !strings.Contains(out, ":§metadata ") {
		t.Errorf("no section rows in the summary:\n%s", out)
	}
	if !strings.Contains(out, "read: rdr inspect --select <id>") {
		t.Errorf("the summary should end with the read instruction:\n%s", out)
	}
	sec := strings.Index(out, "§ ")
	el := strings.Index(out, ":A1 ")
	if el >= 0 && sec > el {
		t.Errorf("sections should precede elements")
	}
}

// TestEveryIndexFacetNamesItselfInTheUsageLog pins usageFacet to dispatch.
// The two lists are written apart — dispatch routes the facet, usagelog
// names it — and a facet added to one and not the other does not fail: it
// silently logs as `graph`, the default. That is how --cycles and
// --open-joint came to be invisible in the log while both had live skill
// call sites, and an audit that prunes on "no calls recorded" would have
// deleted a facet the flow uses.
func TestEveryIndexFacetNamesItselfInTheUsageLog(t *testing.T) {
	// Every flag dispatch checks, with the facet name it must log as.
	cases := []struct{ flag, want string }{
		{"-derived", "derived"},
		{"-coverage", "coverage"},
		{"-backlinks", "backlinks"},
		{"-unresolved", "unresolved"},
		{"-cluster-of=1", "cluster-of"},
		{"-topo", "topo"},
		{"-status", "status"},
		{"-cycles", "cycles"},
		{"-open-joint", "open-joint"},
		{"-anchor-intersect", "anchor-intersect"},
		{"-literal-intersect", "literal-intersect"},
		{"-readme", "readme"},
		{"-row-json=0055", "row-json"},
	}
	for _, c := range cases {
		fs := flag.NewFlagSet("index", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("index", fs)
		if err := fs.Parse([]string{c.flag}); err != nil {
			t.Errorf("%s: parse: %v", c.flag, err)
			continue
		}
		if got := usageFacet("index", f, ""); got != c.want {
			t.Errorf("%s logs as %q, want %q", c.flag, got, c.want)
		}
	}
	// --record is a modifier, but one whose uptake the log must show: it
	// is the call that replaced an inline filter over every pair.
	{
		fs := flag.NewFlagSet("index", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("index", fs)
		if err := fs.Parse([]string{"-anchor-intersect", "-record=1"}); err != nil {
			t.Fatal(err)
		}
		if got := usageFacet("index", f, ""); got != "anchor-intersect:record" {
			t.Errorf("scoped anchor-intersect logs as %q, want anchor-intersect:record", got)
		}
	}
	// The bare graph is the only call that may fall through to "graph".
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := declareFlags("index", fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if got := usageFacet("index", f, ""); got != "graph" {
		t.Errorf("bare index logs as %q, want graph", got)
	}

	// The list above is hand-written, which is the same failure it exists
	// to catch one level up: a facet added to dispatch and to usagelog but
	// not to this list is still untested, and nothing says so. So the flag
	// set is asked what facets it declares, and every one of them must
	// appear above. Adding --literal-intersect proved this necessary — it
	// was wired into both dispatch and the log, and the test passed
	// without covering it.
	declared := map[string]bool{}
	fs.VisitAll(func(fl *flag.Flag) {
		switch fl.Name {
		case "json", "records", "repo", "project", "template", "all", "filter", "record",
			"closure", "final-unimplemented", "facts":
			return // not facets: shared flags and modifiers (the last three qualify --cluster-of)
		}
		declared[fl.Name] = true
	})
	covered := map[string]bool{}
	for _, c := range cases {
		covered[strings.TrimPrefix(strings.SplitN(c.flag, "=", 2)[0], "-")] = true
	}
	for name := range declared {
		if !covered[name] {
			t.Errorf("index declares --%s but no case above pins the facet it logs as; add one", name)
		}
	}
}

// TestEveryInspectQuestionNamesItselfInTheUsageLog pins inspect's
// question flags to their facet names, for the same reason index's are
// pinned: a question that logs as `text` cannot have its uptake audited.
func TestEveryInspectQuestionNamesItselfInTheUsageLog(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{nil, "text"},
		{[]string{"-json"}, "json"},
		{[]string{"-json", "-filter=metadata,counts"}, "json:metadata,counts"},
		{[]string{"-select=0004:A1"}, "select:element"},
		{[]string{"-grep=x"}, "grep"},
		{[]string{"-touched-since=HEAD"}, "touched-since"},
		{[]string{"-touched-since=HEAD", "-json"}, "touched-since"},
	}
	for _, c := range cases {
		fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("inspect", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if got := usageFacet("inspect", f, "0004"); got != c.want {
			t.Errorf("%v logs as %q, want %q", c.args, got, c.want)
		}
	}
}

// TestIndexResolvesOnlyForFacetsThatShowIt is inspect's rule at corpus
// scale: resolution is the only thing index does that reads beyond the
// records dir, and only a facet that can SHOW a `resolved` verdict may pay
// for it. `--unresolved` queries the verdict and `--backlinks` carries it
// on every row; `--cluster-of` walks the edge graph — three edge kinds and
// a direction test — and reads none.
//
// The witness is source files read, not output, because the resolver
// changes nothing a cluster prints: resolving before the dispatch and
// resolving only inside the two arms that need it are byte-identical, and
// differ only in walking a source tree for every symbol the corpus cites
// to answer a question about record relations.
func TestIndexResolvesOnlyForFacetsThatShowIt(t *testing.T) {
	dir := t.TempDir()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "widget.go"), []byte("package w\n\nfunc KnownSymbol() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const head = "## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n"
	write("0001-alpha.md", "# Recommendation 0001: Alpha\n\n"+head+
		"- **Predecessors**: 0002\n\n## Problem Statement\n\n"+
		"Anchored at `widget.go::KnownSymbol` and at `widget.go::GoneSymbol`.\n")
	write("0002-beta.md", "# Recommendation 0002: Beta\n\n"+head+
		"- **Predecessors**: 0001\n\n## Problem Statement\n\nSynthetic.\n")

	for _, tc := range []struct {
		name  string
		args  []string
		walks bool
	}{
		{"unresolved", []string{"--unresolved"}, true},
		{"backlinks", []string{"--backlinks"}, true},
		{"cluster-of", []string{"--cluster-of", "0001"}, false},
		// The graph obeys inspect's rule too: only a projection that can
		// SHOW a `resolved` verdict pays for the walk that decides one.
		// The whole graph carries edges, so it resolves; a --filter that
		// keeps no edge key cannot show a verdict and must not.
		{"whole graph", nil, true},
		{"filter edges", []string{"--filter", "edges"}, true},
		{"filter backlinks", []string{"--filter", "backlinks"}, true},
		{"filter records", []string{"--filter", "records"}, false},
		{"filter elements", []string{"--filter", "elements"}, false},
		{"filter records,elements", []string{"--filter", "records,elements"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := scan.SourceReads.Load()
			args := append([]string{"index", "--json", "--records", dir, "--repo", repo}, tc.args...)
			code, out, errb := runCapture(t, args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, errb)
			}
			if out == "" {
				t.Fatal("no output; the fixture no longer exercises the facet")
			}
			if got := scan.SourceReads.Load() > before; got != tc.walks {
				t.Errorf("walked the source tree = %v, want %v", got, tc.walks)
			}
		})
	}
}

// TestClusterOfIsIndependentOfTheRepo pins the reason --cluster-of may skip
// resolution: its answer cannot depend on the source tree. Same corpus, with
// and without a --repo, must be byte-identical — if it ever is not, the facet
// grew a dependency on a verdict and skipping resolution for it is wrong.
//
// This is the CORRECTNESS half. The cost half — that the walk does not
// happen — is asserted where the walk can be counted, in
// scan.TestClusterTraversalReadsNoSource.
func TestClusterOfIsIndependentOfTheRepo(t *testing.T) {
	dir := t.TempDir()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "widget.go"), []byte("package w\n\nfunc KnownSymbol() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := "# Recommendation 0001: Alpha\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n" +
		"- **Predecessors**: 0002\n\n## Problem Statement\n\n" +
		"Anchored at `widget.go::KnownSymbol`.\n"
	if err := os.WriteFile(filepath.Join(dir, "0001-alpha.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	peer := "# Recommendation 0002: Beta\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n" +
		"- **Predecessors**: 0001\n\n## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(dir, "0002-beta.md"), []byte(peer), 0o600); err != nil {
		t.Fatal(err)
	}
	_, withRepo, _ := runCapture(t, "index", "--json", "--records", dir, "--repo", repo, "--cluster-of", "0001")
	_, without, _ := runCapture(t, "index", "--json", "--records", dir, "--repo", "", "--cluster-of", "0001")
	if withRepo != without {
		t.Errorf("--cluster-of depends on the repo:\nwith:    %s\nwithout: %s", withRepo, without)
	}
}

// TestResolutionReadsOnlyTheEdgeTargets: resolving one record's edges
// reads the records its edges NAME, not the directory. The verdicts are
// identical either way — a whole-dir scan resolves the same targets, it
// just parses 140 other records first — so nothing in the OUTPUT can
// catch a revert. The corpus resolveEdges builds is the witness, and it
// is load-bearing beyond speed: it is what lint reads as opts.Corpus.
func TestResolutionReadsOnlyTheEdgeTargets(t *testing.T) {
	dir := t.TempDir()
	const head = "## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n"
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// 0001 names 0002 and nothing else. 0003..0006 are the bystanders a
	// whole-dir scan would parse.
	write("0001-alpha.md", "# Recommendation 0001: Alpha\n\n"+head+
		"- **Predecessors**: 0002\n\n## Problem Statement\n\nSynthetic.\n")
	for _, n := range []string{"0002", "0003", "0004", "0005", "0006"} {
		write(n+"-peer.md", "# Recommendation "+n+": Peer\n\n"+head+
			"\n## Problem Statement\n\nSynthetic.\n")
	}

	doc, err := scan.File(filepath.Join(dir, "0001-alpha.md"), scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Through resolveEdges, not the helper underneath it: the corpus it
	// RETURNS is what lint reads as opts.Corpus, and calling the helper
	// directly would still pass against a resolveEdges that scanned the
	// whole dir.
	empty := ""
	f := &flags{records: &dir, project: &empty, repo: &empty}
	got := resolveEdges(doc, f, io.Discard)
	if len(got) != 1 || got[0].Record != "0002" {
		var names []string
		for _, d := range got {
			names = append(names, d.Record)
		}
		t.Errorf("resolution loaded %v; want only the named target [0002]", names)
	}
}

// TestDanglingEdgesResolveFalseNotAbsent is the guard the narrowing
// needs and the whole-dir scan got for free.
//
// `resolved` is three-valued, and the distinction that matters most is
// false (looked, not there — a finding) versus absent (nothing looked —
// no finding). The old code inferred "nothing looked" from an EMPTY
// records map, which was safe only because it loaded the whole dir: a
// non-empty dir always yielded a non-empty map. Loading only the named
// targets breaks that inference — a record whose every target is missing
// yields an empty map from a dir that was read in full.
//
// Left uncorrected this turns every dangling reference on the corpus
// into an unchecked one: a skipped check reading as a pass, which is the
// exact failure mode this flow exists to prevent.
func TestDanglingEdgesResolveFalseNotAbsent(t *testing.T) {
	dir := t.TempDir()
	const head = "## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n"
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// 0001's only typed target is 0099, which does not exist. Another
	// record shares the dir, so the DIR is plainly readable.
	write("0001-alpha.md", "# Recommendation 0001: Alpha\n\n"+head+
		"- **Predecessors**: 0099\n\n## Problem Statement\n\nSynthetic.\n")
	write("0002-beta.md", "# Recommendation 0002: Beta\n\n"+head+
		"\n## Problem Statement\n\nSynthetic.\n")

	code, out, errb := runCapture(t, "inspect", "--json", "--filter", "edges",
		"--records", dir, "--repo", "", "0001")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got struct {
		Edges []struct {
			Kind     string `json:"kind"`
			To       string `json:"to"`
			Resolved *bool  `json:"resolved"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	var seen bool
	for _, e := range got.Edges {
		if e.Kind != "predecessor" {
			continue
		}
		seen = true
		if e.Resolved == nil {
			t.Errorf("predecessor %s reads unchecked; the dir was read and 0099 is not in it, so it is false", e.To)
		} else if *e.Resolved {
			t.Errorf("predecessor %s resolved true, but no such record exists", e.To)
		}
	}
	if !seen {
		t.Fatal("no predecessor edge in the projection; the fixture no longer exercises the case")
	}
}

// TestNarrowedResolutionStillPrimesTheSymbolCache: the narrowing must
// not cost the primed walk. ResolveAll's contract is one walk of the
// source tree however many symbols are cited, and it is invisible in the
// output — a per-symbol grep produces identical verdicts and only reads
// more. The count is the only assertion that can catch a regression, so
// it is the one made here, at the command seam where it would land.
func TestNarrowedResolutionStillPrimesTheSymbolCache(t *testing.T) {
	dir, repo := t.TempDir(), t.TempDir()
	// One file, one defined symbol. The three absent ones are the case
	// that cannot short-circuit: unprimed, each costs a full walk.
	if err := os.WriteFile(filepath.Join(repo, "w.go"),
		[]byte("package w\n\nfunc Present() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	const head = "## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n"
	body := "# Recommendation 0001: Alpha\n\n" + head +
		"- **Predecessors**: 0002\n\n## Problem Statement\n\n" +
		"Anchored at `w.go::Present`, `w.go::GoneOne`, `w.go::GoneTwo`, `w.go::GoneThree`.\n"
	if err := os.WriteFile(filepath.Join(dir, "0001-alpha.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002-beta.md"),
		[]byte("# Recommendation 0002: Beta\n\n"+head+"\n## Problem Statement\n\nSynthetic.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	before := scan.SourceReads.Load()
	code, _, errb := runCapture(t, "inspect", "--json", "--filter", "edges",
		"--records", dir, "--repo", repo, "0001")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	// A one-file repo is read once by a primed pass, and once per
	// distinct absent symbol without priming.
	if got := scan.SourceReads.Load() - before; got != 1 {
		t.Errorf("the repo was read %d times for one record; the narrowed path must still prime the symbol cache and walk once", got)
	}
}

// TestIndexFilterProjectsTheNamedKeys is `--filter` at corpus scale. The
// graph is the tool's largest emission — 5.6 MB on the reference corpus —
// and the joint-decision check reads one of its keys, so a caller that
// wants `elements` must not be handed `edges` and `backlinks` too.
//
// The rules are inspect's, deliberately: same flag, same identity carry,
// same stop on a key that does not exist. A consumer that asked for
// `elments` must be told, never handed `{}` and left to conclude the
// corpus has no elements — a filter that answers an empty set for a typo
// is a skipped check reading as a passed one.
func TestIndexFilterProjectsTheNamedKeys(t *testing.T) {
	dir := t.TempDir()
	const head = "## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Draft\n- **Profile**: standard\n"
	if err := os.WriteFile(filepath.Join(dir, "0001-alpha.md"),
		[]byte("# Recommendation 0001: Alpha\n\n"+head+"\n## Problem Statement\n\nSynthetic.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	code, out, errb := runCapture(t, "index", "--json", "--filter", "records", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unparsable: %v", err)
	}
	if _, ok := got["records"]; !ok {
		t.Error("the named key must be present")
	}
	if _, ok := got["schema"]; !ok {
		t.Error("schema is an identity key and is carried unasked")
	}
	for _, k := range []string{"elements", "edges", "backlinks"} {
		if _, ok := got[k]; ok {
			t.Errorf("%q was not asked for and must not be emitted", k)
		}
	}

	// Two keys, since the whole point is that a caller wanting two need
	// not spend two invocations — and in an agent loop an invocation is a
	// turn.
	code, out, errb = runCapture(t, "index", "--json", "--filter", "records,elements", "--records", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unparsable: %v", err)
	}
	if _, ok := got["records"]; !ok {
		t.Error("both named keys must be present")
	}
	if _, ok := got["elements"]; !ok {
		t.Error("both named keys must be present")
	}
	if _, ok := got["edges"]; ok {
		t.Error("an unnamed key must stay out")
	}

	// A key the graph does not have is a stop, not an empty answer.
	code, _, errb = runCapture(t, "index", "--json", "--filter", "elments", "--records", dir)
	if code != 2 {
		t.Errorf("an unknown filter key must exit 2, got %d", code)
	}
	if !strings.Contains(errb, "stopped:no-such-facet") {
		t.Errorf("the stop must name the fault and what could have been asked for: %s", errb)
	}
	if !strings.Contains(errb, "elements") {
		t.Errorf("the stop should list the keys that do exist: %s", errb)
	}
}

// TestCitationSpellingResolves: the corpus cites its own records as
// `<dir>/NNNN` — `cli/0112` under a records dir named cli — and lint's
// fix hints print that spelling back. Read as a path it named `./cli/0112`
// and exited stopped:unreadable, a retry turn per citation. The prefix is
// the records dir's basename, read off the bound dir; a foreign prefix
// still names another dir and is refused as before.
func TestCitationSpellingResolves(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "cli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Recommendation 0112: Layering\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Critical Assumptions\n\n- **A1**: the claim\n  - **Method**: Source Search\n" +
		"  - **Evidence**: `x.go::F`\n  - **Status**: Verified\n"
	if err := os.WriteFile(filepath.Join(dir, "0112-layering.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	record := func(args ...string) string {
		t.Helper()
		code, out, errb := runCapture(t, append([]string{"inspect", "--json", "--records", dir}, args...)...)
		if code != 0 {
			t.Fatalf("%v: exit %d: %s", args, code, errb)
		}
		var env struct {
			Record string `json:"record"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatal(err)
		}
		if env.ID != "" {
			return env.ID
		}
		return env.Record
	}
	if got := record("cli/0112"); got != "0112" {
		t.Errorf("cli/0112 resolved to %q", got)
	}
	if got := record("--select", "cli/0112:A1", "cli/0112"); got != "0112:A1" {
		t.Errorf("--select cli/0112:A1 selected %q", got)
	}
	// The citation alone reaches the element: it is what a finding prints.
	if got := record("cli/0112:A1"); got != "0112:A1" {
		t.Errorf("cli/0112:A1 as the positional selected %q", got)
	}
	if got := record("0112:A1"); got != "0112:A1" {
		t.Errorf("0112:A1 as the positional selected %q", got)
	}
	// Another dir's prefix is not this dir's record.
	code, _, errb := runCapture(t, "inspect", "--records", dir, "other/0112")
	if code != 2 || !strings.Contains(errb, "stopped:unreadable") {
		t.Errorf("foreign prefix: exit %d, stderr %q", code, errb)
	}
}

// TestRepeatedSelectAccumulates: `--select A1 --select A2` used to answer
// A2 alone, silently — the flag package keeps the last value — so
// sessions fell back to one call per element. Every select is answered
// now, in order: text is each element's bytes in sequence, exactly as
// each prints alone; JSON is an array of the single forms.
func TestRepeatedSelectAccumulates(t *testing.T) {
	fx := fixturePath("current-shape.md")
	_, a1, _ := runCapture(t, "inspect", "--select", "0004:A1", fx)
	_, a2, _ := runCapture(t, "inspect", "--select", "0004:A2", fx)
	code, both, errb := runCapture(t, "inspect", "--select", "0004:A1", "--select", "0004:A2", fx)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if both != a1+a2 {
		t.Errorf("two selects are not the two single answers in order:\n%s", both)
	}

	code, out, errb := runCapture(t, "inspect", "--json", "--select", "0004:A2", "--select", "0004:A1", fx)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var items []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("not an array: %v\n%s", err, out)
	}
	if len(items) != 2 || items[0].ID != "0004:A2" || items[1].ID != "0004:A1" {
		t.Errorf("order not kept: %+v", items)
	}

	// A named facet beside an element keeps its name.
	code, out, errb = runCapture(t, "inspect", "--select", "outline", "--select", "0004:A1", fx)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var mixed []map[string]any
	if err := json.Unmarshal([]byte(out), &mixed); err != nil {
		t.Fatalf("not an array: %v", err)
	}
	if len(mixed) != 2 || mixed[0]["select"] != "outline" || mixed[0]["outline"] == nil || mixed[1]["id"] != "0004:A1" {
		t.Errorf("mixed selects: %v", out)
	}

	// One select of the pair missing is still exit 2, but the resolved
	// select answers first and the stop names only the missing id.
	code, out, errb = runCapture(t, "inspect", "--select", "0004:A1", "--select", "0004:C9", fx)
	if code != 2 || !strings.Contains(errb, "no-such-element (0004:C9 in 0004") {
		t.Errorf("missing element among several: exit %d, stderr %q", code, errb)
	}
	if !strings.Contains(out, "- **A1 [") {
		t.Errorf("the resolved select was suppressed: %q", out)
	}
}

// TestInspectSetArity: `inspect 0097 0108 0110` was refused with the
// answer on stderr — two wasted turns for 7.1's critique agent. A named
// set is answered record by record in the order given; a member that
// does not resolve is a `skipped` row, as `status` writes them, never a
// refusal of the whole call.
func TestInspectSetArity(t *testing.T) {
	dir := corpusDir(t)
	code, out, errb := runCapture(t, "inspect", "--records", dir, "0001", "0002", "0009")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"0001  Recommendation 0001: Alpha", "0002  Recommendation 0002: Beta", "0009 skipped  stopped:no-such-record"} {
		if !strings.Contains(out, want) {
			t.Errorf("text set lacks %q:\n%s", want, out)
		}
	}

	code, out, errb = runCapture(t, "inspect", "--json", "--records", dir, "--select", "0001:A1", "--select", "0002:A1", "0001", "0002", "0009")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var env struct {
		Records []struct {
			Record string `json:"record"`
			Path   string `json:"path"`
			Value  any    `json:"value"`
		} `json:"records"`
		Skipped []map[string]string `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	// A select naming another record's element is that record's miss:
	// 0001 has no 0002:A1, so 0001 is skipped and 0002 skips on 0001:A1.
	if len(env.Records) != 0 || len(env.Skipped) != 3 {
		t.Errorf("cross-record selects: %d rows, %d skipped\n%s", len(env.Records), len(env.Skipped), out)
	}

	// The set with a citation per member is the form that reads several
	// records' elements in one call.
	code, out, errb = runCapture(t, "inspect", "--json", "--records", dir, "0001:A1", "0002:A1")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(env.Records) != 2 || env.Records[0].Record != "0001" || env.Records[1].Record != "0002" || len(env.Skipped) != 0 {
		t.Errorf("citation set: %s", out)
	}
	if v, _ := env.Records[1].Value.(map[string]any); v["id"] != "0002:A1" {
		t.Errorf("row value is not the element: %v", env.Records[1].Value)
	}

	// --filter and --select outline keep their shape per row.
	code, out, errb = runCapture(t, "inspect", "--json", "--records", dir, "--filter", "metadata", "0001", "0002")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if v, _ := env.Records[0].Value.(map[string]any); v["metadata"] == nil || v["elements"] != nil {
		t.Errorf("filtered row: %v", env.Records[0].Value)
	}
}

// TestUsageLogNamesTheInspectArity: a set call and an accumulated select
// are the shapes this change exists to measure; a log that averaged them
// into `text` and `select:element` would read them as never used.
func TestUsageLogNamesTheInspectArity(t *testing.T) {
	dir := corpusDir(t)
	log := filepath.Join(t.TempDir(), "usage.jsonl")
	t.Setenv(usageEnvVar, log)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"0001", "0002"}, "records:text"},
		{[]string{"--json", "0001", "0002"}, "records:json"},
		{[]string{"--select", "0001:A1", "--select", "outline", "0001"}, "select:element,outline"},
		{[]string{"--select", "0001:A1", "0001", "0002"}, "records:select:element"},
	} {
		if err := os.WriteFile(log, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		runCapture(t, append([]string{"inspect", "--records", dir}, tc.args...)...)
		raw, _ := os.ReadFile(log)
		var line map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &line); err != nil {
			t.Fatalf("%v: not JSON: %v", tc.args, err)
		}
		if line["facet"] != tc.want {
			t.Errorf("%v: facet = %v, want %s", tc.args, line["facet"], tc.want)
		}
	}
}

// TestIndexRecordScope: the pair facets emitted every pair in the corpus,
// and a propose session filtered them with inline python twice to keep
// one record's rows. `--record` keeps the pairs touching one record, in
// any spelling resolve() accepts; a record that does not resolve is a
// stop, because an empty scope for a misspelt number would read as
// "nothing intersects".
func TestIndexRecordScope(t *testing.T) {
	dir := corpusDir(t)
	for _, spelling := range []string{"1", "0001", "0001-alpha", filepath.Join(dir, "0001-alpha.md"), filepath.Base(dir) + "/0001"} {
		code, out, errb := runCapture(t, "index", "--anchor-intersect", "--json", "--records", dir, "--record", spelling)
		if code != 0 {
			t.Fatalf("%q: exit %d: %s", spelling, code, errb)
		}
		var env struct {
			Record   string `json:"record"`
			Overlaps []struct {
				Records [2]string `json:"records"`
			} `json:"overlaps"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("%q: %v", spelling, err)
		}
		if env.Record != "0001" || len(env.Overlaps) != 1 || env.Overlaps[0].Records != [2]string{"0001", "0002"} {
			t.Errorf("%q: %s", spelling, out)
		}
	}
	// A record in no pair is an empty set with the scope stated, not a stop.
	code, out, _ := runCapture(t, "index", "--anchor-intersect", "--records", dir, "--record", "3")
	if code != 0 || !strings.Contains(out, "total 0 overlapping pairs over in-flight records touching 0003") {
		t.Errorf("unpaired record: exit %d\n%s", code, out)
	}
	code, out, _ = runCapture(t, "index", "--open-joint", "--json", "--records", dir, "--record", "2")
	if code != 0 || !strings.Contains(out, `"record": "0002"`) {
		t.Errorf("open-joint scope: exit %d\n%s", code, out)
	}
	code, _, errb := runCapture(t, "index", "--literal-intersect", "--records", dir, "--record", "9")
	if code != 2 || !strings.Contains(errb, "no-such-record") {
		t.Errorf("unresolvable scope: exit %d, stderr %q", code, errb)
	}
}

// TestLensHelpDoesNotEnumerate: the --lens help once listed five lenses
// while the fact table's probes read more (propose-premortem, reconcile,
// tooling-pass …), and nothing in the flag validates the name — the
// tree's `under` is `{lens}`, so any word binds. A list the flag does not
// enforce drifts on its own, so the help names the SOURCE and no lens;
// this pins that no probe folder the table declares under the lens
// tree's root is spelled in the help, and no `a|b|c` list is either.
func TestLensHelpDoesNotEnumerate(t *testing.T) {
	fs := flag.NewFlagSet("paths", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	declareFlags("paths", fs)
	lens := fs.Lookup("lens")
	if lens == nil {
		t.Fatal("paths declares no --lens")
	}
	if strings.Contains(lens.Usage, "|") {
		t.Errorf("--lens help enumerates: %q", lens.Usage)
	}
	tbl, err := LoadFactTable(factTableForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	tree, ok := tbl.Iter.Trees["lens"]
	if !ok {
		t.Fatal("the table declares no lens tree")
	}
	folders := 0
	for _, d := range tbl.Facts {
		if d.Source != "probe" || d.Root != tree.Root {
			continue
		}
		for _, p := range append([]string{d.Path}, d.Paths...) {
			name, _, _ := strings.Cut(p, "/")
			if name == "" {
				continue
			}
			folders++
			if strings.Contains(lens.Usage, name) {
				t.Errorf("--lens help spells the lens folder %q; it must name the table, not a lens", name)
			}
		}
	}
	if folders == 0 {
		t.Fatal("no probe under the lens tree's root; the test pins nothing")
	}
}

// TestSummaryClipsLongLabels: a table-row scenario or a paragraph-long
// contract carries its whole text as its label; uncapped, those rows put
// the summary of a large record past the ~30KB one call returns. The text
// row keeps the head of the label with a marker; JSON keeps it whole.
func TestSummaryClipsLongLabels(t *testing.T) {
	raw, _ := os.ReadFile(fixturePath("current-shape.md"))
	long := strings.Repeat("é", 150) + " tail"
	body := strings.Replace(string(raw), "- **A2 [The checksum helper is the only implementation in the tree]**",
		"- **A2 ["+long+"]**", 1)
	path := filepath.Join(t.TempDir(), "0004-current-shape.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCapture(t, "inspect", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var row string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " 0004:A2 ") {
			row = l
		}
	}
	if row == "" {
		t.Fatalf("no A2 row:\n%s", out)
	}
	label := row[strings.Index(row, "é"):]
	if n := utf8.RuneCountInString(label); n != summaryLabelRunes || !strings.HasSuffix(label, "…") || strings.Contains(row, "tail") {
		t.Errorf("A2 label should be %d runes ending in …, got %d: %q", summaryLabelRunes, n, label)
	}
	code, out, errb = runCapture(t, "inspect", "--json", "--select", "elements", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, long) {
		t.Errorf("JSON elements should carry the whole label")
	}
}

// TestLintHeaderCarriesTierCounts: the record's verdict line says how
// many findings block, how many are resolution tier, how many are a
// surviving placeholder and how many are advisory — the four numbers a
// gate used to re-run lint under three greps to learn. They are checked
// against the JSON findings, so the header cannot drift from the list.
func TestLintHeaderCarriesTierCounts(t *testing.T) {
	path := fixturePath("legacy-shape.md")
	code, jsonOut, errb := runCapture(t, "lint", "--json", path)
	if code > 1 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var rep struct {
		Findings []struct {
			Tier     string `json:"tier"`
			Code     string `json:"code"`
			Blocking bool   `json:"blocking"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) == 0 {
		t.Fatal("the fixture lints clean; the counts need findings to count")
	}
	var blocking, resolution, placeholder int
	for _, fd := range rep.Findings {
		if fd.Blocking {
			blocking++
		}
		if fd.Tier == "resolution" {
			resolution++
		}
		if fd.Code == "placeholder:survived" {
			placeholder++
		}
	}
	want := fmt.Sprintf("blocking=%d resolution=%d placeholder=%d advisory=%d",
		blocking, resolution, placeholder, len(rep.Findings)-blocking)

	_, text, _ := runCapture(t, "lint", path)
	header := strings.SplitN(text, "\n", 2)[0]
	if !strings.HasSuffix(header, want) {
		t.Errorf("header %q does not end with %q", header, want)
	}
	if strings.Count(header, "  ") < 4 {
		t.Errorf("header %q lost the record/status/state/verdict columns", header)
	}
}

// TestAssumptionsFilterIsTheGateProjection: `--filter assumptions` is
// the per-assumption view the exit gates read — id, Status, Method
// members and off-vocabulary, the Evidence span and its anchors — and
// nothing else, so it is a fraction of the elements facet it replaces.
// `resolved` on an anchor is three-valued: absent unless edges were
// resolved in the same call.
func TestAssumptionsFilterIsTheGateProjection(t *testing.T) {
	path := fixturePath("current-shape.md")
	code, out, errb := runCapture(t, "inspect", "--json", "--filter", "assumptions", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got struct {
		Record      string `json:"record"`
		Assumptions []struct {
			ID     string `json:"id"`
			Status *struct {
				Value string `json:"value"`
			} `json:"status"`
			Method *struct {
				Members       []string `json:"members"`
				OffVocabulary []string `json:"off_vocabulary"`
			} `json:"method"`
			Evidence *struct {
				LineStart int `json:"line_start"`
				Anchors   []struct {
					To       string `json:"to"`
					Resolved *bool  `json:"resolved"`
				} `json:"anchors"`
			} `json:"evidence"`
		} `json:"assumptions"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Record != "0004" || len(got.Assumptions) != 3 {
		t.Fatalf("record %q with %d assumptions; want 0004 with 3", got.Record, len(got.Assumptions))
	}
	a2, a3 := got.Assumptions[1], got.Assumptions[2]
	if a2.ID != "0004:A2" || a2.Status == nil || a2.Status.Value != "Verified" {
		t.Errorf("A2 = %+v; want id 0004:A2 at Verified", a2)
	}
	if a2.Evidence == nil || a2.Evidence.LineStart == 0 || len(a2.Evidence.Anchors) == 0 || a2.Evidence.Anchors[0].To != "hash::Sum32" {
		t.Errorf("A2 evidence = %+v; want its span and the hash::Sum32 anchor", a2.Evidence)
	}
	if a2.Evidence != nil && len(a2.Evidence.Anchors) > 0 && a2.Evidence.Anchors[0].Resolved != nil {
		t.Error("an anchor carries a verdict though nothing looked; resolved must be absent")
	}
	if a3.Method == nil || strings.Join(a3.Method.Members, "+") != "Peer RDR+Source Search" || len(a3.Method.OffVocabulary) != 0 {
		t.Errorf("A3 method = %+v; want the compound split into its two sanctioned members", a3.Method)
	}

	_, elements, _ := runCapture(t, "inspect", "--json", "--filter", "elements", path)
	if len(out)*4 > len(elements) {
		t.Errorf("assumptions is %d bytes against %d for elements; the saving is the point", len(out), len(elements))
	}

	// With edges in the same call the anchors carry the edge's verdict —
	// here false, against an empty source root.
	repo := t.TempDir()
	code, out, errb = runCapture(t, "inspect", "--json", "--repo", repo, "--filter", "assumptions,edges", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, `"to": "hash::Sum32",`) || !strings.Contains(out, `"resolved": false`) {
		t.Errorf("with edges resolved the anchor should carry resolved=false:\n%s", out)
	}
	if code, _, _ = runCapture(t, "inspect", "--json", "--select", "assumptions", path); code != 0 {
		t.Errorf("--select assumptions exit %d", code)
	}
}

// TestNoSuchElementHintNamesTheContainingElement: a select for a token
// the author wrote but the projector never mints (a failure register
// label cited as `0021:F-1`) used to stop bare, and the session's next
// turn was a grep to learn the text was right there. When the token
// stands verbatim in the body, the stop names the minted element whose
// lines hold it, with the range — the id that actually reaches it.
func TestNoSuchElementHintNamesTheContainingElement(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "0021-frames.md")
	body := "# Recommendation 0021: Frames\n\n## Metadata\n\n- **Status**: Draft\n\n" +
		"## Critical Assumptions\n\n" +
		"- **A1 [load-bearing]**: The frame parser rejects malformed input.\n" +
		"  - **Evidence**: The failure register F-1 names the overflow case.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runCapture(t, "inspect", "--select", "0021:F-1", p)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	want := "stopped:no-such-element (0021:F-1 in 0021; F-1 appears inside 0021:A1 (9-10))"
	if !strings.Contains(errb, want) {
		t.Errorf("stderr %q\nwant it to carry %q", errb, want)
	}
	// F-12 is not F-1: the token matches whole, never as a prefix.
	if err := os.WriteFile(p, []byte(strings.Replace(body, "F-1 ", "F-12 ", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errb = runCapture(t, "inspect", "--select", "0021:F-1", p)
	if code != 2 || strings.Contains(errb, "appears inside") {
		t.Errorf("a longer token still hinted: exit %d, stderr %q", code, errb)
	}
}

// TestMultiSelectAnswersResolvedThenStops: one bad id among several used
// to zero the whole call, so the good selects' bytes were paid for and
// thrown away. The resolved selects emit first, the stop names ONLY the
// missing ids, the exit stays 2 — and the stop line lands on stdout as
// well as stderr, because sessions habitually 2>/dev/null a read they
// expect to succeed and a stated absence must survive that.
func TestMultiSelectAnswersResolvedThenStops(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "0021-layers.md")
	fence := "```"
	body := "# Recommendation 0021: Layers\n\n## Metadata\n\n- **Status**: Draft\n\n" +
		"#### Normative Contracts\n\n**C1**\n\n" +
		fence + "normative\nL-1  input := owned records\nL-3  one capture per key\n" + fence + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCapture(t, "inspect", "--select", "0021:L-1", "--select", "0021:L-2", p)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.HasPrefix(out, "L-1  input := owned records\n") {
		t.Errorf("the resolved select did not answer first: %q", out)
	}
	stop := "stopped:no-such-element (0021:L-2 in 0021; near misses: 0021:L-1, 0021:L-3)"
	if !strings.Contains(errb, stop) {
		t.Errorf("stderr %q\nwant %q", errb, stop)
	}
	if !strings.Contains(out, stop) {
		t.Errorf("stdout %q does not carry the stop", out)
	}
	if strings.Contains(errb, "0021:L-1 in") {
		t.Errorf("the stop names a select that resolved: %q", errb)
	}
	// All missing: no partial output, one stop line, each id hinted.
	code, out, _ = runCapture(t, "inspect", "--select", "0021:L-2", "--select", "0021:L-9", p)
	if code != 2 || !strings.HasPrefix(out, "stopped:no-such-element (0021:L-2, 0021:L-9 in 0021; near misses for L-2:") {
		t.Errorf("all missing: exit %d, stdout %q", code, out)
	}
}

// TestStopsLandOnStdoutToo: every stopped: diagnostic reaches stdout as
// well as stderr, for every subcommand but env — env's stdout is eval'd,
// so a mirrored stop would be executed rather than read.
func TestStopsLandOnStdoutToo(t *testing.T) {
	if code, out, _ := runCapture(t, "lint", "x.md"); code != 2 || !strings.Contains(out, "stopped:unreadable") {
		t.Errorf("lint: exit %d, stdout %q does not carry the stop", code, out)
	}
	if code, out, _ := runCapture(t, "status", "--tags", "--records", t.TempDir()); code != 2 || !strings.Contains(out, "stopped:usage") {
		t.Errorf("status: exit %d, stdout %q does not carry the stop", code, out)
	}
}

// TestTrailingFlagsRunAsIfTypedFirst: a flag typed after the target
// (`rdr lint 0112 --json`) used to stop with a misleading "flags go
// before the target" refusal, because the flag package stops parsing at
// the first positional and never sees it. hoistFlags reorders argv ahead
// of Parse, so the trailing form must behave exactly like the leading
// form: same exit code, same stdout, same stderr.
func TestTrailingFlagsRunAsIfTypedFirst(t *testing.T) {
	for _, c := range []struct {
		leading, trailing []string
	}{
		{[]string{"lint", "--json", "0112"}, []string{"lint", "0112", "--json"}},
		{[]string{"inspect", "--records", "docs", "0112"}, []string{"inspect", "0112", "--records", "docs"}},
		{[]string{"status", "--tags", "0112"}, []string{"status", "0112", "--tags"}},
		{[]string{"paths", "--next-iter", "0112"}, []string{"paths", "0112", "--next-iter"}},
	} {
		wantCode, wantOut, wantErr := runCapture(t, c.leading...)
		code, out, errb := runCapture(t, c.trailing...)
		if code != wantCode || out != wantOut || errb != wantErr {
			t.Errorf("%v vs %v:\n leading: exit %d, stdout %q, stderr %q\ntrailing: exit %d, stdout %q, stderr %q",
				c.leading, c.trailing, wantCode, wantOut, wantErr, code, out, errb)
		}
	}
}

// TestUndefinedTrailingFlagStillRefuses: an undefined flag typed after the
// target hoists alongside the defined ones, but fs.Lookup finds nothing
// for it, so it carries no operand and Parse refuses it with the flag
// package's own message — the same as if it had been typed first.
func TestUndefinedTrailingFlagStillRefuses(t *testing.T) {
	code, _, errb := runCapture(t, "inspect", fixturePath("current-shape.md"), "--ids")
	if code != 2 || !strings.Contains(errb, "flag provided but not defined: -ids") {
		t.Errorf("exit %d, stderr %q", code, errb)
	}
}

// TestTrailingValueFlagWithDashOperand: a value-taking flag's operand can
// itself start with `-` (a seed label, a select id); hoistFlags must keep
// it paired with its flag rather than treating it as a second flag token.
func TestTrailingValueFlagWithDashOperand(t *testing.T) {
	wantCode, wantOut, wantErr := runCapture(t, "inspect", "--grep", "-seed-label", fixturePath("current-shape.md"))
	code, out, errb := runCapture(t, "inspect", fixturePath("current-shape.md"), "--grep", "-seed-label")
	if code != wantCode || out != wantOut || errb != wantErr {
		t.Errorf("trailing --grep -seed-label diverged:\n want: exit %d, stdout %q, stderr %q\n got:  exit %d, stdout %q, stderr %q",
			wantCode, wantOut, wantErr, code, out, errb)
	}
}

// TestDoubleDashKeepsFollowingTokenPositional: a literal `--` stops the
// hoist scan; it and everything after it stay in place as positionals, so
// a target that happens to start with `-` is not mistaken for a flag.
func TestDoubleDashKeepsFollowingTokenPositional(t *testing.T) {
	wantCode, wantOut, wantErr := runCapture(t, "inspect", fixturePath("current-shape.md"))
	code, out, errb := runCapture(t, "inspect", "--", fixturePath("current-shape.md"))
	if code != wantCode || out != wantOut || errb != wantErr {
		t.Errorf("`--` form diverged:\n want: exit %d, stdout %q, stderr %q\n got:  exit %d, stdout %q, stderr %q",
			wantCode, wantOut, wantErr, code, out, errb)
	}
}

// TestSummaryRowsCarryByteSizes: every range row of the text summary says
// what the range weighs, because line counts do not predict bytes — a
// 250-line dense element was read blind at 65KB on a "≈25KB" guess. The
// element's size is the bytes --select returns for it, the JSON elements
// facet carries the same number, and on a record built to be enormous the
// summary itself stays within the one-call cap the label clip protects.
func TestSummaryRowsCarryByteSizes(t *testing.T) {
	code, out, errb := runCapture(t, "inspect", fixturePath("current-shape.md"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	raw, _ := os.ReadFile(fixturePath("current-shape.md"))
	lines := strings.Split(string(raw), "\n")
	want := len(strings.Join(lines[43:50], "\n")) + 1 // A2 is lines 44-50
	var a2, sec string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " 0004:A2 ") {
			a2 = l
		}
		if strings.Contains(l, ":§metadata ") {
			sec = l
		}
	}
	if a2 == "" || !strings.Contains(a2, " "+humanBytes(want)+" ") {
		t.Errorf("A2 row should carry %s:\n%q", humanBytes(want), a2)
	}
	if sec == "" || !strings.Contains(sec, "B ") && !strings.Contains(sec, "K ") {
		t.Errorf("section row should carry a size:\n%q", sec)
	}
	code, jsonOut, _ := runCapture(t, "inspect", "--json", "--select", "elements", fixturePath("current-shape.md"))
	if code != 0 || !strings.Contains(jsonOut, fmt.Sprintf("\"bytes\": %d", want)) {
		t.Errorf("JSON elements should carry bytes %d: exit %d", want, code)
	}

	// A synthetic monster: one assumption whose body is a page of dense
	// lines, plus a long roster of long-labelled peers. The heavy row says
	// so in K, and the whole summary stays under the ~30KB call budget.
	dense := strings.Repeat("  - widget batching holds under replay pressure and never reorders the ledger\n", 900)
	var peers strings.Builder
	for i := 2; i <= 120; i++ {
		fmt.Fprintf(&peers, "- **A%d [%s]**\n  - Evidence: synthetic\n", i, strings.Repeat("x", 160))
	}
	body := "# Recommendation 0031: Weighing\n\n## Metadata\n\n- **Status**: Draft\n\n" +
		"## Critical Assumptions\n\n- **A1 [The heavy one]**\n" + dense + peers.String()
	path := filepath.Join(t.TempDir(), "0031-weighing.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb = runCapture(t, "inspect", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var heavy string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " 0031:A1 ") {
			heavy = l
		}
	}
	if !strings.Contains(heavy, "K ") {
		t.Errorf("the dense element's row should carry a K size:\n%q", heavy)
	}
	if len(out) > 30*1024 {
		t.Errorf("summary is %d bytes; the size column must not break the ~30KB cap", len(out))
	}
}

// TestGrepNamesTheContainingElement: `inspect --grep` answers which
// minted elements hold a literal — the author-label question that used to
// be a select-per-element hunt. A label buried mid-prose inside a clause
// reports the clause, one row with id, range, line numbers and the first
// matching line; a bare miss is a stated absence at exit 0; and crossing
// --grep with --select has no one answer, so it is refused as usage.
func TestGrepNamesTheContainingElement(t *testing.T) {
	fence := "```"
	body := "# Recommendation 0022: Bands\n\n## Metadata\n\n- **Status**: Draft\n\n" +
		"#### Normative Contracts\n\n**C1**\n\n" +
		fence + "normative\nL-1  input := owned rows\nL-2  the BandHold rule keeps every row\n" +
		"     in its band until release\n" + fence + "\n\nProse mentions BandHold once more.\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "0022-bands.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCapture(t, "inspect", "--grep", "BandHold", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	rows := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(rows) != 2 || !strings.HasPrefix(rows[0], "0022:L-2 ") {
		t.Fatalf("want the clause first, then the prose section:\n%s", out)
	}
	if !strings.Contains(rows[0], "lines 13") || !strings.Contains(rows[0], "L-2  the BandHold rule keeps every row") {
		t.Errorf("clause row should carry its line number and first match:\n%q", rows[0])
	}
	if !strings.Contains(rows[1], ":§") {
		t.Errorf("a hit outside every element names its section:\n%q", rows[1])
	}
	code, out, _ = runCapture(t, "inspect", "--json", "--grep", "BandHold", path)
	var rep struct {
		Grep    string `json:"grep"`
		Matches []struct {
			ID    string `json:"id"`
			Lines []int  `json:"lines"`
			First string `json:"first"`
		} `json:"matches"`
	}
	if err := json.Unmarshal([]byte(out), &rep); code != 0 || err != nil ||
		rep.Grep != "BandHold" || len(rep.Matches) != 2 || rep.Matches[0].ID != "0022:L-2" {
		t.Errorf("--json grep: exit %d, %v, %s", code, err, out)
	}

	// A miss answers, it does not error: absence stated, exit 0.
	code, out, _ = runCapture(t, "inspect", "--grep", "bandhold", path)
	if code != 0 || !strings.Contains(out, "no-match") {
		t.Errorf("case-sensitive miss: exit %d, out %q", code, out)
	}

	code, _, errb = runCapture(t, "inspect", "--grep", "BandHold", "--select", "0022:C1", path)
	if code != 2 || !strings.Contains(errb, "stopped:usage") {
		t.Errorf("--grep with --select: exit %d, stderr %q", code, errb)
	}
}

// TestIndexAnchorIntersectOverTheStatusCorpus: the fence fact's first arm,
// by the CLI, over the same fixtures — 0025 and 0026 share the key
// pre-image anchor and neither cites the other, so the pair is UNCITED
// under `--record` for either side.
func TestIndexAnchorIntersectOverTheStatusCorpus(t *testing.T) {
	recs, _ := bindStatusFixture(t)
	for _, rec := range []string{"0025", "0026"} {
		code, out, errb := runCapture(t, "index", "--anchor-intersect", "--record", rec, "--json", "--records", recs)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", rec, code, errb)
		}
		var env struct {
			Overlaps []scan.Overlap `json:"overlaps"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatal(err)
		}
		uncited := 0
		for _, o := range env.Overlaps {
			if o.Cited {
				continue
			}
			uncited++
			if o.Records != [2]string{"0025", "0026"} || !containsString(o.Anchors, "internal/cache/key.go::Preimage") {
				t.Errorf("%s: uncited pair %+v, want 0025/0026 over key.go::Preimage", rec, o)
			}
		}
		if uncited != 1 {
			t.Errorf("%s: %d uncited pairs, want 1:\n%s", rec, uncited, out)
		}
	}
}
