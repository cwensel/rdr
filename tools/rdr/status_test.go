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
	for _, n := range []string{"0020", "0021", "0022", "0023", "0024", "0025", "0027", "0028", "0029", "0030", "0031", "0032"} {
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
		{"0020", map[string]string{"status": "Draft", "ca": "all-pending", "profile": "mid",
			"seam_lineage_count": "2", "seam_lineage": "2+", "accretion_disposition": "true"},
			"the front-half Draft whose CAs are all Pending: Refine is the open ~; and the " +
				"accretion floor's ESCAPED cell — two prior point-fixes with the disposition " +
				"written as the template's nested bullet, so a mid Profile stands"},
		{"0021", map[string]string{"status": "Final", "status_form": "joint-decision",
			"profile": "foundational", "contracts": "1", "lens_cove": "true",
			"lens_3amigo": "true", "lens_critique_single": "true",
			"lens_critique_modelb": "false", "gate_written": "true",
			"seam_lineage_count": "3", "seam_lineage": "2+", "accretion_disposition": "false"},
			"foundational mid-row: critique is single-model, so the dual-model diff is still owed; " +
				"the floor HOLDS (three prior point-fixes, no disposition) and the field is at it"},
		{"0022", map[string]string{"status": "Draft", "status_form": "revised-from",
			"status_reentry": "true", "impl_capsule": "true", "impl_state": "IN-PROGRESS",
			"cluster_reconciled": "true", "seam_lineage_count": "0", "seam_lineage": "0",
			"req_count": "0-10", "impl_orphans": "0", "impl_open_decisions": "0",
			"impl_mvv_recorded": "true"},
			"the scoped backward edge, with a capsule header that states its own state; " +
				"it is also in TWO cluster dirs (0021-0022 and 0021-0022-0023), which is a " +
				"widened re-run rather than an ambiguity — nested overlap still answers true; " +
				"and the Stage-8 launch-gate ledger (req-list, coverage, deviations) all read clean"},
		{"0030", map[string]string{"status": "Final", "profile": "small", "lines": "0-400",
			"impl_capsule": "true", "impl_state": "COMPLETE", "req_count": "0-10",
			"impl_orphans": "0", "impl_open_decisions": "0", "impl_mvv_recorded": "true"},
			"the launch table's one COMPLETE cell on disk, in the canonical artifacts/ layout " +
				"and the launch prompt's own grammar: a `## REQ-MVV output` heading beside a " +
				"runner heading, and a needs-author-decision line rewritten to RESOLVED in place"},
		{"0023", map[string]string{"status": "Implemented", "lens_3amigo": "false",
			"legacy_evidence_shape": "true", "cluster_reconciled": "true"},
			"the warning case: 3amigo DID run, in the pre-migration file shape — " +
				"without this probe the record reads as never lensed"},
		{"0024", map[string]string{"status": "Deferred", "status_parked": "true",
			"cluster_reconciled": "false"},
			"parked is neither in flight nor terminal; and in no cluster, which is the " +
				"false that routes a Final to /rdr-cluster-reconcile before implement"},
		{"0020", map[string]string{"cluster_reconciled": "false"},
			"the topical-epoch guard: final-cluster-2026-06-22/ exists in the fixture " +
				"tree and carries the four-digit run 2026, but no record matches it — a " +
				"rule that read numbers OUT of a name would mint a claim for record 2026"},
		// The completion fixture: critique ran on TWO base models and was
		// diffed, and run-1 declares its variant. These are the signals a
		// FOLDER cannot carry, and the ones four skills overrode the
		// routing with before the facts existed. Its Normative Contracts
		// are PROSE — unlabelled, so `contracts` is 0 while
		// `contracts_prose` is true, which is the pair that stops a zero
		// count from reading as an empty section.
		{"0026", map[string]string{"status": "Draft", "profile": "foundational",
			"critique_models": "differ", "critique_model_a": "synthetic-model-a",
			"critique_model_b": "synthetic-model-b", "lens_critique_diff": "true",
			"repeatability_variant": "full", "contracts": "0", "contracts_prose": "true"},
			"the dual-model diff and the stamped variant: completion signals the folder " +
				"probes cannot see, and a prose contract section a zero count cannot tell " +
				"from an empty one"},
		{"0025", map[string]string{"status": "Draft", "ca": "unknown-plan",
			"ca_placeholder": "1"},
			"the sparse Draft: no Profile, no Cluster, no Joint-check:, and a CA still " +
				"carrying the template legend. It is the ABSENCE fixture — here in " +
				"--json `profile`, `cluster`, `clustered` and `joint_checks` are all " +
				"ABSENT rather than false, which is the honest answer this table's " +
				"header insists on; TestRoutingSentinelsRenderOnlyAsTags asserts the " +
				"other half, that --tags renders each one's declared sentinel so a " +
				"routing dimension is never merely missing"},
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

	// The absence fixture's other half, asserted as absence rather than
	// as a value: `--json` must OMIT a fact the record does not carry.
	// A sentinel that leaked into this rendering would turn every
	// "nothing looked" into a claim, which is the one thing the fact
	// table's three-valued header forbids.
	code, out, errb := runCapture(t, "status", "--json", "--facts", table, "0025")
	if code != 0 {
		t.Fatalf("0025: exit %d: %s", code, errb)
	}
	var env struct {
		Facts []Fact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, f := range env.Facts {
		present[f.Name] = true
	}
	for _, name := range []string{"profile", "profile_raw", "cluster", "clustered", "joint_checks",
		"seam_lineage_count", "seam_lineage", "accretion_disposition"} {
		if present[name] {
			t.Errorf("0025 carries no such field, but --json reports %q; absence must survive "+
				"the JSON rendering even though --tags substitutes a sentinel there", name)
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
		"0022-cache-metrics-surface", "0025-cache-key-encoding",
		"0026-cache-hash-identity", "0028-cache-flush-hook", "0029-cache-size-report",
		"0030-cache-warm-ratio", "0031-cache-warm-report", "0032-cache-warm-alert",
		"total 10 in flight over 13 records"} {
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

// TestPredecessorsRollup: the launch precheck's fold, read off the
// fixtures. 0031 names 0030 (capsule COMPLETE) and 0021 (no capsule):
// `incomplete`, and the set names 0021. 0032 names a record the dir does
// not hold: `unresolved`, halting first. 0030 declares no Predecessors:
// absent in --json, and `--tags` renders the declared sentinel `none`.
func TestPredecessorsRollup(t *testing.T) {
	_, table := bindStatusFixture(t)
	read := func(rec string) map[string]string {
		t.Helper()
		code, out, errb := runCapture(t, "status", "--facts", table, "--json", "--filter", "predecessors_state,predecessors_incomplete", rec)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", rec, code, errb)
		}
		var got struct {
			Facts []struct {
				Name, Value string
				Members     []string
			}
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		facts := map[string]string{}
		for _, f := range got.Facts {
			if f.Members != nil {
				facts[f.Name] = strings.Join(f.Members, ",")
			} else {
				facts[f.Name] = f.Value
			}
		}
		return facts
	}
	if got := read("0031"); got["predecessors_state"] != "incomplete" || got["predecessors_incomplete"] != "0021" {
		t.Errorf("0031: %v, want incomplete with 0021 named (0030 is COMPLETE, 0021 has no capsule)", got)
	}
	if got := read("0032"); got["predecessors_state"] != "unresolved" || got["predecessors_incomplete"] != "9999" {
		t.Errorf("0032: %v, want unresolved naming 9999", got)
	}
	if got := read("0030"); len(got) != 0 {
		t.Errorf("0030 declares no Predecessors, but --json carries %v; absence must survive the JSON rendering", got)
	}
	code, out, errb := runCapture(t, "status", "--facts", table, "--tags", "--filter", "status,predecessors_state", "0030")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "predecessors_state=none") {
		t.Errorf("--tags on a record with no Predecessors did not render the sentinel:\n%s", out)
	}
}

// TestClusterMembersProposed: the tandem barrier's fold. 0021 declares
// 0020 (an authored Implementation Plan) and 0022 (a draft placeholder):
// `some`. A sibling the dir does not hold reads not-proposed, so a cluster
// with one authored plan and one missing member is `some`, never `all`;
// an unclustered record is absent, rendered `solo`.
func TestClusterMembersProposed(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "status", "--facts", table, "--filter", "cluster_members_proposed", "0021")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "cluster_members_proposed     enum    some") {
		t.Errorf("0021's siblings are one authored plan and one placeholder, want some:\n%s", out)
	}
	code, out, errb = runCapture(t, "status", "--facts", table, "--tags", "--filter", "cluster_members_proposed", "0020")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "cluster_members_proposed=solo") {
		t.Errorf("an unclustered record renders solo:\n%s", out)
	}

	dir := t.TempDir()
	head := func(num, title, cluster string) string {
		return "# Recommendation " + num + ": " + title + "\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Draft\n- **Cluster**: " + cluster +
			"\n\n## Problem Statement\n\nSynthetic.\n"
	}
	for name, body := range map[string]string{
		"0001-a.md": head("0001", "A", "0002-b, 0003-c"),
		"0002-b.md": head("0002", "B", "0001-a") + "\n## Implementation Plan\n\n### Phase 1: Code Implementation\n\nWrite the adapter behind the existing interface first.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errb = runCapture(t, "status", "--facts", table, "--records", dir, "--filter", "cluster_members_proposed", "0001")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "cluster_members_proposed     enum    some") {
		t.Errorf("a missing sibling reads not-proposed, want some:\n%s", out)
	}
	if code, out, _ = runCapture(t, "status", "--facts", table, "--records", dir, "--filter", "cluster_members_proposed", "0002"); code != 0 || !strings.Contains(out, "enum    none") {
		t.Errorf("0002's one sibling has no plan, want none: exit %d\n%s", code, out)
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
	for _, n := range []string{"0020", "0021", "0022", "0023", "0024", "0025"} {
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

// TestShippedTableRendersEveryRecordAsTags is the guarantee the unit rule
// above cannot make on its own: `unsafeTagWord` decides one value, but
// `--tags` refuses the WHOLE call when any fact fails it, so one
// undeclared scalar takes the navigator down on every record that
// carries it.
//
// That is not hypothetical. `critique_model_a` was a bare `scalar`, and a
// model id is an open vocabulary — `claude-opus-5[1m]` is a real stamp,
// and `[1m]` is a glob. Measured 2026-08-28 before the fix: 27 of 157
// records on the reference corpus exited 2 with no output, concentrated
// in the recent in-flight range, which is exactly the set the navigator
// is called on. The golden missed it because every fixture stamp was a
// bracket-free synthetic id; 0021's now carries brackets, like the corpus.
//
// So this asserts over the SHIPPED table rather than a hand-built one: a
// new fact whose value cannot survive argv must be declared prose when it
// is declared, not after a consumer's navigator stops answering.
func TestShippedTableRendersEveryRecordAsTags(t *testing.T) {
	recs, ev, _ := statusFixture(t)
	t.Setenv("RDR_RECORDS", recs)
	t.Setenv("RDR_EVIDENCE", ev)
	t.Setenv("RDR_SOURCE_REPO", "")

	shipped := filepath.Join("..", "..", "models", "rdr-facts.toml")
	if _, err := os.Stat(shipped); err != nil {
		t.Skipf("shipped table not beside the tool: %v", err)
	}
	for _, n := range []string{"0020", "0021", "0022", "0023", "0024", "0025", "0026", "0027", "0028", "0029", "0030"} {
		code, out, errb := runCapture(t, "status", "--tags", "--facts", shipped, n)
		if code != 0 {
			t.Errorf("%s: --tags exit %d (%s) — a fact the shipped table declares "+
				"cannot be rendered as argv; declare it prose", n, code, strings.TrimSpace(errb))
			continue
		}
		// A refusal is the loud failure; a silently empty rendering would
		// be the quiet one, and the navigator cannot tell it from a
		// record with no signals.
		if strings.TrimSpace(out) == "" {
			t.Errorf("%s: --tags exited 0 with no tags at all", n)
		}
	}
}

// TestTagsSurviveABracketedModelID pins the specific value that broke,
// end to end: the id reaches `--json` (so §model-stamp's reader keeps
// it) and is absent from `--tags` (so no caller globs it).
func TestTagsSurviveABracketedModelID(t *testing.T) {
	_, table := bindStatusFixture(t)

	code, tags, errb := runCapture(t, "status", "--tags", "--facts", table, "0021")
	if code != 0 {
		t.Fatalf("a bracketed model id still refuses the call: exit %d: %s", code, errb)
	}
	if strings.Contains(tags, "critique_model_a") {
		t.Errorf("the model id was rendered as a tag, where [1m] is a glob:\n%s", tags)
	}

	code, out, _ := runCapture(t, "status", "--json", "--facts", table, "0021")
	if code != 0 {
		t.Fatal(out)
	}
	var env struct {
		Facts []Fact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	var id string
	for _, f := range env.Facts {
		if f.Name == "critique_model_a" {
			id = f.Value
		}
	}
	if !strings.Contains(id, "[1m]") {
		t.Errorf("--json dropped the model id too; then declaring it prose lost the "+
			"fact rather than relocating it: %q", id)
	}
}

// ---------------------------------------------------------------------
// The SET arity: `rdr status NNNN NNNN …`
//
// Stage 8's predecessor precheck and Stage 7.1's Final-and-unimplemented
// filter both hold a list of records and used to read N `status.md`
// headers by hand. These tests hold the properties that make one call a
// safe replacement for that loop.
// ---------------------------------------------------------------------

// setRows is the parsed `--json` envelope of a set call.
type setRows struct {
	Records []struct {
		Record string `json:"record"`
		Path   string `json:"path"`
		Facts  []Fact `json:"facts"`
	} `json:"records"`
	Skipped []map[string]string `json:"skipped"`
}

func statusSetJSON(t *testing.T, args ...string) setRows {
	t.Helper()
	code, out, errb := runCapture(t, args...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var env setRows
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("unparseable envelope: %v\n%s", err, out)
	}
	return env
}

// TestStatusSetAnswersEveryNamedRecord is the capability itself: several
// records, one call, one row each, in the order asked.
func TestStatusSetAnswersEveryNamedRecord(t *testing.T) {
	_, table := bindStatusFixture(t)
	want := []string{"0020", "0021", "0022"}
	env := statusSetJSON(t, append([]string{"status", "--facts", table, "--json"}, want...)...)
	if len(env.Records) != len(want) {
		t.Fatalf("asked for %d records, got %d", len(want), len(env.Records))
	}
	for i, w := range want {
		if env.Records[i].Record != w {
			t.Errorf("row %d is %q, want %q; the set must answer in the order asked", i, env.Records[i].Record, w)
		}
	}
	if len(env.Skipped) != 0 {
		t.Errorf("every record resolves, so skipped must be empty: %v", env.Skipped)
	}
}

// TestStatusSetAgreesWithStatusOne is what makes the set safe to adopt:
// the same evaluation, so a consumer that switches from N calls to one
// cannot see a different answer.
func TestStatusSetAgreesWithStatusOne(t *testing.T) {
	_, table := bindStatusFixture(t)
	recs := []string{"0020", "0021", "0022", "0023", "0024", "0025"}
	env := statusSetJSON(t, append([]string{"status", "--facts", table, "--json"}, recs...)...)

	for i, rec := range recs {
		code, out, errb := runCapture(t, "status", "--facts", table, "--json", rec)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", rec, code, errb)
		}
		var one struct {
			Facts []Fact `json:"facts"`
		}
		if err := json.Unmarshal([]byte(out), &one); err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(env.Records[i].Facts)
		if err != nil {
			t.Fatal(err)
		}
		wantJSON, err := json.Marshal(one.Facts)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(wantJSON) {
			t.Errorf("%s: the set and the single-record forms disagree.\n set: %s\n one: %s", rec, got, wantJSON)
		}
	}
}

// TestStatusSetReportsAnUnresolvableRecordRatherThanHalting is the
// three-valued discipline carried up to the set, and it is the property
// Stage 8 depends on.
//
// A predecessor with no record — or no capsule — must read as "nothing
// looked", distinguishable from "looked, not COMPLETE": the stage halts
// on the second and reports the first. A refusal for the whole call would
// collapse them, because a set that answers nothing says nothing about
// any member.
func TestStatusSetReportsAnUnresolvableRecordRatherThanHalting(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "status", "--facts", table, "--json", "0020", "9999", "0021")
	if code != 0 {
		t.Fatalf("an unresolvable member must not fail the call: exit %d: %s", code, errb)
	}
	var env setRows
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Records) != 2 {
		t.Errorf("the two resolvable records must still be answered, got %d", len(env.Records))
	}
	if len(env.Skipped) != 1 || env.Skipped[0]["target"] != "9999" {
		t.Fatalf("9999 must be reported as skipped, with its reason: %v", env.Skipped)
	}
	if env.Skipped[0]["why"] == "" {
		t.Error("a skipped row with no reason is an absence nobody can act on")
	}
}

// TestStatusSetAbsentFactStaysAbsent: a record with no capsule renders NO
// impl_state, rather than a false or an INCOMPLETE. `filterFacts` must
// not invent a row for a fact that did not evaluate.
func TestStatusSetAbsentFactStaysAbsent(t *testing.T) {
	_, table := bindStatusFixture(t)
	env := statusSetJSON(t, "status", "--facts", table, "--json", "--filter", "impl_state", "0020", "0021", "0022", "0023", "0024", "0025")
	for _, r := range env.Records {
		for _, f := range r.Facts {
			if f.Name != "impl_state" {
				t.Errorf("%s: --filter impl_state also returned %q", r.Record, f.Name)
			}
			if f.Value == "" {
				t.Errorf("%s: impl_state rendered with an empty value; absent must mean the fact is OMITTED, "+
					"never present-and-blank — a caller cannot tell the second from a real answer", r.Record)
			}
		}
	}
}

// TestStatusSetRefusesTags: `--tags` renders one resolver's argv, and a
// set has no single record to name. The worklist already refuses it with
// these words; the set must not invent a second vocabulary.
func TestStatusSetRefusesTags(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, _, errb := runCapture(t, "status", "--facts", table, "--tags", "0020", "0021")
	if code != 2 {
		t.Fatalf("--tags over a set must refuse, exit %d", code)
	}
	if !strings.Contains(errb, "stopped:usage") || !strings.Contains(errb, "name a record") {
		t.Errorf("the refusal must be the worklist's, verbatim: %q", errb)
	}
}

// TestStatusFilterKeepsOnlyNamedFacts, and refuses a name the table does
// not declare rather than answering nothing — the rule `--filter` already
// follows on inspect and index. An empty answer to a misspelled fact
// reads as "the record does not have it", which is a claim about the
// record instead of about the request.
func TestStatusFilterKeepsOnlyNamedFacts(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "status", "--facts", table, "--json", "--filter", "status,profile", "0020")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var one struct {
		Facts []Fact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &one); err != nil {
		t.Fatal(err)
	}
	if len(one.Facts) != 2 {
		t.Fatalf("--filter status,profile returned %d facts", len(one.Facts))
	}

	code, _, errb = runCapture(t, "status", "--facts", table, "--json", "--filter", "impl_stat", "0020")
	if code != 2 {
		t.Fatalf("a misspelled fact must refuse, exit %d", code)
	}
	if !strings.Contains(errb, "stopped:no-such-fact") || !strings.Contains(errb, "impl_state") {
		t.Errorf("the refusal must name the bad key and list what was available: %q", errb)
	}
}

// TestStatusFilterDropsTheSummary. The reason to filter is a caller's
// context budget, and the summary is most of the payload — Profile alone
// carries the field's whole rationale tail. Filtering the vector while
// still shipping the summary would answer the letter of the request and
// miss its point.
func TestStatusFilterDropsTheSummary(t *testing.T) {
	_, table := bindStatusFixture(t)
	recs := []string{"0020", "0021", "0022", "0023", "0024", "0025"}
	_, full, _ := runCapture(t, append([]string{"status", "--facts", table, "--json"}, recs...)...)
	_, lean, _ := runCapture(t, append([]string{"status", "--facts", table, "--json", "--filter", "status"}, recs...)...)

	if len(lean) >= len(full) {
		t.Errorf("--filter did not shrink the set envelope: %d vs %d bytes", len(lean), len(full))
	}
	for _, key := range []string{"\"title\"", "\"counts\"", "\"profile\"", "\"short_title\""} {
		if strings.Contains(lean, key) {
			t.Errorf("the filtered row still carries %s; the summary is the bulk of the payload", key)
		}
	}
	// Identity survives: a row a caller cannot attribute is not an answer.
	for _, key := range []string{"\"record\"", "\"path\""} {
		if !strings.Contains(lean, key) {
			t.Errorf("the filtered row dropped %s, which is what attributes it to a record", key)
		}
	}
}

// TestStatusSetReadsOnlyTheRecordsNamed is the cost property, and it is
// the whole reason this is an arity rather than a corpus facet.
//
// The worklist scans the records dir; a set must not. Measured on the
// reference corpus the two are 47ms and 2.0s apart, so a set that fell
// through to a scan would be a silent 40x regression on the call Stage 8
// makes every run. The check is structural rather than timed: point
// --records at a dir holding ONLY the named records' files and the answer
// must be identical, which cannot be true of anything that enumerated.
func TestStatusSetReadsOnlyTheRecordsNamed(t *testing.T) {
	recs, table := bindStatusFixture(t)
	want := statusSetJSON(t, "status", "--facts", table, "--json", "--filter", "status", "0020", "0021")

	// A dir with the two named records and nothing else.
	lean := t.TempDir()
	for _, n := range []string{"0020", "0021"} {
		matches, err := filepath.Glob(filepath.Join(recs, n+"-*.md"))
		if err != nil || len(matches) == 0 {
			t.Fatalf("fixture %s: %v", n, err)
		}
		body, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(lean, filepath.Base(matches[0])), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("RDR_RECORDS", lean)
	got := statusSetJSON(t, "status", "--facts", table, "--json", "--filter", "status", "--records", lean, "0020", "0021")

	if len(got.Records) != len(want.Records) {
		t.Fatalf("the set answered %d records against a lean dir, %d against the full one",
			len(got.Records), len(want.Records))
	}
	for i := range got.Records {
		if got.Records[i].Record != want.Records[i].Record {
			t.Errorf("row %d differs: %q vs %q", i, got.Records[i].Record, want.Records[i].Record)
		}
	}
}
