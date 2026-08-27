package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The `status` fixture corpus: five synthetic records and an evidence
// tree beside them, spanning the shapes a navigator has to tell apart —
// an all-Pending front-half Draft, a foundational Final mid-lens-row
// with a written gate, a scoped re-entry with a Stage-9 capsule, a
// terminal record whose evidence is still in the pre-migration file
// shape, and a parked Deferred.
func statusFixture(t *testing.T) (records, evidence, table string) {
	t.Helper()
	base, err := filepath.Abs(filepath.Join("testdata", "status"))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "records"), filepath.Join(base, "evidence"), factTableForTest(t)
}

// bindStatusFixture points the seam vars at the fixture tree. The vars
// are the only way a probe learns where to look, so a test that forgets
// one gets absent facts rather than wrong ones — which is the contract,
// but makes for a confusing failure.
func bindStatusFixture(t *testing.T) (records, table string) {
	t.Helper()
	recs, ev, tbl := statusFixture(t)
	t.Setenv("RDR_RECORDS", recs)
	t.Setenv("RDR_EVIDENCE", ev)
	t.Setenv("RDR_SOURCE_REPO", "")
	return recs, tbl
}

// TestStatusGolden pins every fixture's whole fact vector.
//
// A golden over the LIVE corpus was the other option and is the wrong
// one: it would fail on every record edit, cannot run where the consumer
// repo is absent, and would publish that repo's content into a public
// one. The fixtures reproduce the SHAPES instead — which is what the
// facts read — so this fails when the evaluator changes and not when a
// record does.
func TestStatusGolden(t *testing.T) {
	_, table := bindStatusFixture(t)
	var got strings.Builder
	for _, n := range []string{"0020", "0021", "0022", "0023", "0024"} {
		code, out, errb := runCapture(t, "status", "--facts", table, n)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", n, code, errb)
		}
		got.WriteString("# " + n + "\n")
		got.WriteString(out)
		got.WriteString("\n")
	}
	goldenPath := filepath.Join("testdata", "status", "facts.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, []byte(got.String()), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("golden rewritten; re-run without UPDATE_GOLDEN")
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != string(want) {
		t.Errorf("the fact vectors moved. Diff them, and if the change is "+
			"intended re-run with UPDATE_GOLDEN=1.\n--- got ---\n%s", got.String())
	}
}

// TestStatusFixturesKeepTheirShapeSignals pins what makes each fixture
// worth having. The golden above would fail if any of these changed, but
// it would fail as a wall of text; this says which distinction was lost.
func TestStatusFixturesKeepTheirShapeSignals(t *testing.T) {
	_, table := bindStatusFixture(t)
	for _, c := range []struct {
		record string
		want   map[string]string
		why    string
	}{
		{"0020", map[string]string{"status": "Draft", "ca": "all-pending", "profile": "mid"},
			"the front-half Draft whose CAs are all Pending: Refine is the open ~"},
		{"0021", map[string]string{"status": "Final", "status_form": "joint-decision",
			"profile": "foundational", "contracts": "1", "lens_cove": "true",
			"lens_3amigo": "true", "lens_critique_single": "true",
			"lens_critique_modelb": "false", "gate_written": "true"},
			"foundational mid-row: critique is single-model, so the dual-model diff is still owed"},
		{"0022", map[string]string{"status": "Draft", "status_form": "revised-from",
			"status_reentry": "true", "impl_capsule": "true", "impl_state": "IN-PROGRESS"},
			"the scoped backward edge, with a capsule header that states its own state"},
		{"0023", map[string]string{"status": "Implemented", "lens_3amigo": "false",
			"legacy_evidence_shape": "true"},
			"the warning case: 3amigo DID run, in the pre-migration file shape — " +
				"without this probe the record reads as never lensed"},
		{"0024", map[string]string{"status": "Deferred", "status_parked": "true"},
			"parked is neither in flight nor terminal"},
	} {
		code, out, errb := runCapture(t, "status", "--json", "--facts", table, c.record)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", c.record, code, errb)
		}
		var env struct {
			Facts []Fact `json:"facts"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, f := range env.Facts {
			got[f.Name] = f.Value
		}
		for name, want := range c.want {
			if got[name] != want {
				t.Errorf("%s: %s = %q, want %q — %s", c.record, name, got[name], want, c.why)
			}
		}
	}
}

// TestStatusWorklistIsTheInFlightSet: no argument is the worklist, and
// the worklist is Draft and Final — never terminal, never parked. It is
// the heir of `index --in-flight` and pins the same rule, now with the
// facts that used to cost a call per record.
func TestStatusWorklistIsTheInFlightSet(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "status", "--facts", table)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"0020-cache-eviction-policy", "0021-cache-warmup-order",
		"0022-cache-metrics-surface", "total 3 in flight over 5 records"} {
		if !strings.Contains(out, want) {
			t.Errorf("worklist lacks %q:\n%s", want, out)
		}
	}
	// Implemented is terminal and Deferred is parked; neither is work.
	for _, gone := range []string{"0023-cache-shard-count", "0024-cache-persistence"} {
		if strings.Contains(out, gone) {
			t.Errorf("worklist includes %q, which is not in flight:\n%s", gone, out)
		}
	}
	// The facts travel WITH the row — that is the whole point of the
	// verb. A worklist that named the records and left the caller to go
	// and derive each one's position is the choreography it replaced.
	if !strings.Contains(out, "ca                           enum    all-pending") {
		t.Errorf("worklist rows carry no facts:\n%s", out)
	}
}

// TestStatusTagsRenderShellSafeArgv is the load-bearing one.
//
// The composition this verb exists for is
// `intrastate flow resolve --model … $(rdr status --tags NNNN)`, and an
// UNQUOTED `$(…)` splits on whitespace and globs the pieces. A value
// carrying a space therefore does not arrive as one argument: it arrives
// as several, the first being `k=<head>` — a well-formed tag with a
// silently truncated value. Verified against `sh` and `bash`: with files
// `k=a` and `k=b` present, the word `k=[ab]` expands to two arguments.
//
// So every rendered word must be one shell word. This asserts the
// property directly rather than trusting the renderer.
func TestStatusTagsRenderShellSafeArgv(t *testing.T) {
	_, table := bindStatusFixture(t)
	for _, n := range []string{"0020", "0021", "0022", "0023", "0024"} {
		code, out, errb := runCapture(t, "status", "--tags", "--facts", table, n)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", n, code, errb)
		}
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines)%2 != 0 {
			t.Fatalf("%s: %d lines; --tag and its k=v pair up", n, len(lines))
		}
		for i, line := range lines {
			if i%2 == 0 {
				if line != "--tag" {
					t.Errorf("%s line %d = %q, want --tag", n, i, line)
				}
				continue
			}
			if strings.ContainsAny(line, " \t") {
				t.Errorf("%s: %q carries whitespace and would split into several "+
					"arguments under an unquoted $(…)", n, line)
			}
			if !strings.Contains(line, "=") {
				t.Errorf("%s: %q is not name=value", n, line)
			}
		}
	}
}

// TestStatusTagsOmitProseAndJSONKeepsIt: `profile_raw` is the rationale
// tail — a sentence on most real records — and it is declared `prose` in
// the table. It is absent from the tag rendering, because as argv it
// would arrive truncated at the first space; it is present in `--json`,
// where it is a JSON string and nothing splits it.
//
// The two halves are asserted together on purpose. Omitting it from the
// tags is only correct BECAUSE the value is still reachable; if `--json`
// ever dropped it too, the fact would be gone and this would still pass
// as two separate tests.
func TestStatusTagsOmitProseAndJSONKeepsIt(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, tags, errb := runCapture(t, "status", "--tags", "--facts", table, "0020")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if strings.Contains(tags, "profile_raw") {
		t.Errorf("a prose fact was rendered as a tag:\n%s", tags)
	}
	if !strings.Contains(tags, "profile=mid") {
		t.Errorf("the routing half of Profile is missing:\n%s", tags)
	}

	code, out, _ := runCapture(t, "status", "--json", "--facts", table, "0020")
	if code != 0 {
		t.Fatal(out)
	}
	var env struct {
		Facts []Fact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	var raw string
	for _, f := range env.Facts {
		if f.Name == "profile_raw" {
			raw = f.Value
		}
	}
	if !strings.Contains(raw, "eviction metrics surface") {
		t.Errorf("--json dropped the prose fact too; then it is not reachable at all: %q", raw)
	}
	// And it is the value the tag rendering could not have carried.
	if !strings.ContainsAny(raw, " ") {
		t.Errorf("the fixture's Profile no longer carries a rationale tail, so this "+
			"test no longer exercises the case it was written for: %q", raw)
	}
}

// TestUnsafeTagWordCatchesWhatTheShellWouldRewrite pins the rule itself,
// including the two cases it must NOT refuse: a canonical set rendering,
// whose brackets this tool writes around content it has already quoted,
// and an empty one.
func TestUnsafeTagWordCatchesWhatTheShellWouldRewrite(t *testing.T) {
	for _, c := range []struct {
		fact Fact
		word string
		want bool
		why  string
	}{
		{Fact{Kind: "enum"}, "status=Draft", false, "an enum value is one word"},
		{Fact{Kind: "int"}, "contracts=3", false, "digits are one word"},
		{Fact{Kind: "scalar"}, "profile_raw=mid — one contract", true, "a space splits the word"},
		{Fact{Kind: "scalar"}, "x=a\tb", true, "a tab splits it too"},
		{Fact{Kind: "scalar"}, "x=[ab]", true, "a bracket expression can be rewritten to a filename"},
		{Fact{Kind: "scalar"}, "x=*", true, "a star can be rewritten to a filename"},
		{Fact{Kind: "set", Members: []string{"0131", "0132"}}, `cluster=["0131","0132"]`, false,
			"the tool writes these brackets around quoted members; no filename looks like this"},
		{Fact{Kind: "set", Members: nil}, "cluster=[]", false, "an empty set is still one word"},
		{Fact{Kind: "set", Members: []string{"a*"}}, `cluster=["a*"]`, true,
			"a metacharacter INSIDE a member came from a record, and is caught"},
	} {
		if got := unsafeTagWord(c.fact, c.word); got != c.want {
			t.Errorf("unsafeTagWord(%q) = %v, want %v — %s", c.word, got, c.want, c.why)
		}
	}
}

// TestStatusTagsRefuseTheWorklist: `--tags` renders ONE resolver's argv.
// Over a worklist it would produce a command line naming no record, so
// it stops rather than emitting something a caller could paste and run.
func TestStatusTagsRefuseTheWorklist(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "status", "--tags", "--facts", table)
	if code != 2 || !strings.Contains(errb, "stopped:usage") {
		t.Errorf("worklist --tags: exit %d, stdout %q, stderr %q", code, out, errb)
	}
}

// TestStatusStopsWithoutAFactTable: the table is the grammar, so its
// absence is a stop and not an empty answer. A `status` that printed
// nothing would read as "this record has no signals".
func TestStatusStopsWithoutAFactTable(t *testing.T) {
	recs, _, _ := statusFixture(t)
	t.Setenv("RDR_RECORDS", recs)
	t.Setenv("RDR_HOME", t.TempDir())
	code, out, errb := runCapture(t, "status", "--facts", filepath.Join(t.TempDir(), "nope.toml"), "0020")
	if code != 2 || !strings.Contains(errb, "stopped:no-fact-table") {
		t.Errorf("missing table: exit %d, stdout %q, stderr %q", code, out, errb)
	}
	if strings.Count(errb, "stopped:") != 1 {
		t.Errorf("one failure should be reported once, got: %q", errb)
	}
}

// TestStatusNamesItselfInTheUsageLog: the worklist scans the corpus and
// one record does not — three orders of magnitude apart — so the log has
// to tell them apart, and the rendering too, since `--tags` is the call
// the navigator actually makes.
func TestStatusNamesItselfInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		args          []string
		worklist, rec string
	}{
		{nil, "worklist", "record"},
		{[]string{"-json"}, "worklist:json", "record:json"},
		{[]string{"-tags"}, "worklist:tags", "record:tags"},
	} {
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("status", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		// The operand is what separates a corpus scan from one record,
		// and no flag carries it.
		if got := usageFacet("status", f, ""); got != c.worklist {
			t.Errorf("status %v with no operand logs as %q, want %q", c.args, got, c.worklist)
		}
		if got := usageFacet("status", f, "0020"); got != c.rec {
			t.Errorf("status %v on a record logs as %q, want %q", c.args, got, c.rec)
		}
	}
}
