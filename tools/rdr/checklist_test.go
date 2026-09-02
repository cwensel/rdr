package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The checklist used to be six prose rules in `skills/rdr-status/SKILL.md`
// that a model applied to the fact vector by hand. These tests are those
// rules, recorded as goldens before the prose was deleted, so the
// rendering the skill now prints verbatim is the one the rules produced.

// checklistFixtures is every status fixture the golden pins. 0026 is
// left to the arm tests (its lens facts are pinned by shape elsewhere);
// 0033 is the fixture added for the `✓ spikes` and report-present arms.
var checklistFixtures = []string{"0020", "0021", "0022", "0023", "0024", "0025", "0027", "0028", "0029", "0030", "0033"}

func TestChecklistGolden(t *testing.T) {
	_, table := bindStatusFixture(t)
	var got strings.Builder
	for _, n := range checklistFixtures {
		code, out, errb := runCapture(t, "status", "--checklist", "--facts", table, n)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", n, code, errb)
		}
		got.WriteString("# " + n + "\n")
		got.WriteString(out)
		got.WriteString("\n")
	}
	goldenPath := filepath.Join("testdata", "status", "checklist.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, []byte(got.String()), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("golden rewritten; re-run without UPDATE_GOLDEN")
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("no golden: %v (run with UPDATE_GOLDEN=1 to write it)", err)
	}
	if got.String() != string(want) {
		t.Errorf("checklist rendering drifted from the golden. Diff the golden against the "+
			"intended re-run with UPDATE_GOLDEN=1.\n--- got ---\n%s", got.String())
	}
}

// TestChecklistRowsAreTheSkillTable: the row names and their order are
// the skill's "What the facts mean" table, verbatim — the rule the prose
// used to state ("never invent, split, or rename a row").
func TestChecklistRowsAreTheSkillTable(t *testing.T) {
	skill, err := os.ReadFile(repoFile(t, filepath.Join("skills", "rdr-status", "SKILL.md")))
	if err != nil {
		t.Fatal(err)
	}
	rows := signalTableRows(string(skill))
	if strings.Join(rows, "|") != strings.Join(checklistStages, "|") {
		t.Errorf("checklist rows %q\nskill table %q", checklistStages, rows)
	}
}

// checklistOf decodes one record's checklist from `--json --checklist`.
func checklistOf(t *testing.T, table, rec string) map[string]checklistRow {
	t.Helper()
	code, out, errb := runCapture(t, "status", "--json", "--checklist", "--facts", table, rec)
	if code != 0 {
		t.Fatalf("%s: exit %d: %s", rec, code, errb)
	}
	var env struct {
		Facts     []Fact         `json:"facts"`
		Checklist []checklistRow `json:"checklist"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("%s: %v\n%s", rec, err, out)
	}
	if len(env.Facts) == 0 {
		t.Fatalf("%s: --json --checklist dropped the facts; the checklist is BESIDE the vector, not instead of it", rec)
	}
	byStage := map[string]checklistRow{}
	for _, r := range env.Checklist {
		byStage[r.Stage] = r
	}
	return byStage
}

// TestChecklistArms pins the six former prose rules on the fixture that
// exercises each.
func TestChecklistArms(t *testing.T) {
	_, table := bindStatusFixture(t)
	for _, c := range []struct {
		rec, stage, glyph, note, why string
	}{
		{"0020", "3 Refine", glyphJudged, "open — CAs all Pending",
			"rule 3: all-Pending is an OPEN ~, and it owns Next"},
		{"0020", "4 Resolve", glyphNot, "open — 2 of 2 Pending",
			"rule 2: false is –"},
		{"0021", "5+6 Pre-Lock (review+resolve)", glyphJudged, "~ critique (diff owed)",
			"a lone critique.md on a foundational record is in progress, not done"},
		{"0021", "5+6 Pre-Lock (review+resolve)", glyphJudged, "✓ cove · ✓ 3amigo",
			"rule 5: only the lenses that RAN are printed; grounding is absent, not –"},
		{"0021", "7 Finalize", glyphDone, "gate.md", "rule 1: true is ✓"},
		{"0021", "7.1 Cluster", glyphDone, "key=0021-0022-0023", "the reconciled membership is the dir's own name"},
		{"0022", "6 Reconcile", glyphDone, "reconcile/reconcile.md", "the report-present arm"},
		{"0022", "8 Implement", glyphJudged, "impl_state=IN-PROGRESS", "the capsule's own state word"},
		{"0023", "1 Seed", glyphDone, "Implemented", "the status value, verbatim"},
		{"0023", "5+6 Pre-Lock (review+resolve)", glyphNot, "legacy evidence shape",
			"the pre-migration shape is named so – cannot read as un-run"},
		{"0024", "1 Seed", glyphDone, "Deferred", "parked renders like any status; the qualifier is not a fact"},
		{"0025", "3 Refine", glyphAbsent, "unreadable", "unknown-plan certifies nothing — ? not –"},
		{"0025", "5+6 Pre-Lock (review+resolve)", glyphNot, "profile unset",
			"a field the record does not carry is unset, never a sentinel"},
		{"0026", "5+6 Pre-Lock (review+resolve)", glyphJudged, "~ repeatability (diff owed)",
			"run-1 without the diff is in progress"},
		{"0030", "8 Implement", glyphDone, "impl_state=COMPLETE", "COMPLETE is the ✓ cell"},
		{"0033", "3 Refine", glyphJudged, "judged done — 2 Verified, 0 terminal",
			"rule 3: judged done names Stage 4's product, not just the verdict"},
		{"0033", "4 Resolve", glyphDone, "spikes", "spikes/ is Stage 4's durable artifact"},
		{"0033", "5+6 Pre-Lock (review+resolve)", glyphDone, "✓ grounding · ✓ 3amigo",
			"a mid row with reconcile/ behind it is complete"},
		{"0033", "6 Reconcile", glyphDone, "reconcile/reconcile.md", "the report-present arm"},
	} {
		row, ok := checklistOf(t, table, c.rec)[c.stage]
		if !ok {
			t.Errorf("%s: no row %q", c.rec, c.stage)
			continue
		}
		if row.Glyph != c.glyph || !strings.Contains(row.Note, c.note) {
			t.Errorf("%s %s: got %s %q, want %s …%q… — %s", c.rec, c.stage, row.Glyph, row.Note, c.glyph, c.note, c.why)
		}
	}
}

// TestChecklistAbsentRootIsQuestionNeverDash is rule 4, and the reason
// `withAbsentSentinels` is never applied to this rendering: with the
// evidence root unbound every probe under it is `?` naming the var, and
// none of them is `–`.
func TestChecklistAbsentRootIsQuestionNeverDash(t *testing.T) {
	recs, _, table := statusFixture(t)
	t.Setenv("RDR_RECORDS", recs)
	t.Setenv("RDR_EVIDENCE", "")
	t.Setenv("RDR_SOURCE_REPO", "")
	// An empty variable falls through to the seam marker, and the engine
	// checkout carries one; run from a directory no marker reaches so
	// "unbound" is what the binary sees, not what this shell inherited.
	t.Chdir(t.TempDir())
	rows := checklistOf(t, table, "0021")
	for _, stage := range []string{"4 Resolve", "5+6 Pre-Lock (review+resolve)", "6 Reconcile", "7.1 Cluster"} {
		r := rows[stage]
		if r.Glyph != glyphAbsent || !strings.Contains(r.Note, "RDR_EVIDENCE") {
			t.Errorf("%s with no evidence root: %s %q, want ? naming RDR_EVIDENCE", stage, r.Glyph, r.Note)
		}
	}
	// The artifacts root is still bound, so the gate still answers.
	if r := rows["7 Finalize"]; r.Glyph != glyphDone {
		t.Errorf("7 Finalize: %s %q — the artifacts root is bound and must still answer", r.Glyph, r.Note)
	}
}

// TestChecklistRefusesSetWorklistAndFilter: the checklist is one
// record's, whole. A set or the worklist is refused as `--tags` is, and
// `--filter` is refused because an unasked fact would render as absent.
func TestChecklistRefusesSetWorklistAndFilter(t *testing.T) {
	_, table := bindStatusFixture(t)
	for _, args := range [][]string{
		{"status", "--checklist", "--facts", table},
		{"status", "--checklist", "--facts", table, "0020", "0021"},
		{"status", "--checklist", "--filter", "status", "--facts", table, "0020"},
		{"status", "--checklist", "--tags", "--facts", table, "0020"},
		{"status", "--checklist", "--argv", "--facts", table, "0020"},
	} {
		code, _, errb := runCapture(t, args...)
		if code != 2 || !strings.Contains(errb, "stopped:usage") {
			t.Errorf("%v: exit %d %q, want a stopped:usage refusal", args, code, errb)
		}
	}
}

// TestStatusArgvIsOneLinePerRecordWithDeferred pins the line shape
// bin/rdr-next reads: four tab-separated fields, qualifier last, and the
// parked record present — the worklist's text form still excludes it.
func TestStatusArgvIsOneLinePerRecordWithDeferred(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "status", "--argv", "--facts", table)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		f := strings.SplitN(line, "\t", 4)
		if len(f) != 4 {
			t.Fatalf("not four tab-separated fields: %q", line)
		}
		seen[f[0]] = true
		if !strings.HasPrefix(f[2], "--tag status="+f[1]+" ") {
			t.Errorf("%s: argv does not open on its own status: %q", f[0], f[2])
		}
		// Every word is one shell word: `tagWords` refuses the rest, and
		// a set's own brackets are the tool's (unsafeTagWord exempts them).
		for _, w := range strings.Split(f[2], " ") {
			if strings.ContainsAny(w, "\t\n\r\v\f") || w == "" {
				t.Errorf("%s: argv word %q would split", f[0], w)
			}
		}
		if f[1] == "Deferred" && f[3] != "revisit when the restart budget is measured" {
			t.Errorf("%s: the parked row's qualifier is %q; the revisit condition must ride the line", f[0], f[3])
		}
	}
	if !seen["0024-cache-persistence"] {
		t.Errorf("--argv omits the Deferred record; a park nobody re-reads is an abandon:\n%s", out)
	}
	if seen["0023-cache-shard-count"] {
		t.Errorf("--argv carries an Implemented record; terminal is not work:\n%s", out)
	}
	// One record is one line, same shape.
	code, out, errb = runCapture(t, "status", "--argv", "--facts", table, "0020")
	if code != 0 || strings.Count(out, "\n") != 1 || !strings.HasPrefix(out, "0020-cache-eviction-policy\tDraft\t--tag ") {
		t.Errorf("one record --argv: exit %d %q %s", code, out, errb)
	}
	for _, args := range [][]string{
		{"status", "--argv", "--json", "--facts", table},
		{"status", "--argv", "--tags", "--facts", table, "0020"},
	} {
		if code, _, errb := runCapture(t, args...); code != 2 || !strings.Contains(errb, "stopped:usage") {
			t.Errorf("%v: exit %d %q, want stopped:usage", args, code, errb)
		}
	}
}

func TestChecklistAndArgvNameThemselvesInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		args          []string
		worklist, rec string
	}{
		{[]string{"-checklist"}, "worklist:checklist", "record:checklist"},
		{[]string{"-argv"}, "worklist:argv", "record:argv"},
		{[]string{"-checklist", "-json"}, "worklist:checklist", "record:checklist"},
	} {
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("status", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatal(err)
		}
		if got := usageFacet("status", f, ""); got != c.worklist {
			t.Errorf("status %v with no operand logs as %q, want %q", c.args, got, c.worklist)
		}
		if got := usageFacet("status", f, "0020"); got != c.rec {
			t.Errorf("status %v on a record logs as %q, want %q", c.args, got, c.rec)
		}
	}
}

// TestRdrNextRendersTheWorklist runs bin/rdr-next end to end over the
// fixture: `rdr status --argv` from a freshly built binary beside the
// script, `intrastate flow resolve` per row, chaining where the model
// says to. Skips without `intrastate` (as TestLaunchModelResolvesTheFixture
// does) or without a Go toolchain to build the binary the script binds.
func TestRdrNextRendersTheWorklist(t *testing.T) {
	is := intrastateBinary(t)
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain to build the projector the script binds")
	}
	bindStatusFixture(t)
	home, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if out, err := exec.Command(goBin, "build", "-o", filepath.Join(dir, "rdr"), ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	script, err := os.ReadFile(filepath.Join(home, "bin", "rdr-next"))
	if err != nil {
		t.Fatalf("bin/rdr-next is not in the tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rdr-next"), script, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RDR_HOME", home)
	t.Setenv("RDR_INTRASTATE", is)
	out, err := exec.Command("sh", filepath.Join(dir, "rdr-next")).CombinedOutput()
	if err != nil {
		t.Fatalf("rdr-next: %v\n%s", err, out)
	}
	lines := map[string]string{}
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		lines[strings.SplitN(l, " · ", 2)[0]] = l
	}
	for rec, want := range map[string]string{
		"0020-cache-eviction-policy": "· Draft · next: /rdr-refine 0020 ·",
		"0026-cache-hash-identity":   "· Draft · next: /rdr-prelock cove 0026 ·", // locate chains to lens
		"0022-cache-metrics-surface": "· Draft · next: /rdr-resolve 0022 ·",      // locate chains to reentry
		"0021-cache-warmup-order":    "· Final · next: stopped:check-the-joint-decision-home ·",
		"0024-cache-persistence":     "· Deferred · parked: revisit when the restart budget is measured",
		"0029-cache-size-report":     "· Final · next: /rdr-implement 0029 ·",
	} {
		if !strings.Contains(lines[rec], want) {
			t.Errorf("%s: %q lacks %q\n%s", rec, lines[rec], want, out)
		}
	}
	if _, ok := lines["0023-cache-shard-count"]; ok {
		t.Errorf("rdr-next lists an Implemented record:\n%s", out)
	}
}

// TestChecklistPrelockNamesTheDeterminacyAddOn: at mid/large the
// `Determinacy:` line is part of the row's answer — a fired line with no
// repeatability run is a lens still owed, no line is a judgement still
// owed, and `na` adds nothing. Foundational carries the full lens on its
// row, so the line never shows there.
func TestChecklistPrelockNamesTheDeterminacyAddOn(t *testing.T) {
	recs, evidence, table := statusFixture(t)
	t.Setenv("RDR_RECORDS", recs)
	t.Setenv("RDR_EVIDENCE", evidence)
	for _, c := range []struct {
		rec, want string
		present   bool
	}{
		{"0020", "Determinacy: unjudged", true}, // mid, no line
		{"0033", "Determinacy", false},          // mid, n/a line: silent, row stays done
		{"0026", "Determinacy", false},          // foundational, fired: moot
	} {
		row := checklistOf(t, table, c.rec)["5+6 Pre-Lock (review+resolve)"]
		if strings.Contains(row.Note, c.want) != c.present {
			t.Errorf("%s: 5+6 note %q; want %q present=%v", c.rec, row.Note, c.want, c.present)
		}
	}
	if row := checklistOf(t, table, "0033")["5+6 Pre-Lock (review+resolve)"]; row.Glyph != glyphDone {
		t.Errorf("0033: an n/a line must leave the complete row %s; got %s %q", glyphDone, row.Glyph, row.Note)
	}

	// The fired arm, over a synthetic vector: no fixture is mid + fired
	// with repeatability un-run, and the row's reading must not wait on one.
	tbl := loadRealTable(t)
	facts := []Fact{
		{Name: "profile", Kind: "enum", Value: "mid"},
		{Name: "lens_grounding", Kind: "bool", Value: "true"},
		{Name: "lens_grounding_findings", Kind: "bool", Value: "true"},
		{Name: "lens_3amigo", Kind: "bool", Value: "true"},
		{Name: "lens_3amigo_consolidation", Kind: "bool", Value: "true"},
		{Name: "lens_repeatability", Kind: "bool", Value: "false"},
		{Name: "reconcile", Kind: "bool", Value: "true"},
		{Name: "determinacy", Kind: "enum", Value: "fired"},
	}
	glyph, note := prelockCell(newFactView(tbl, facts, map[string]string{"evidence": "bound"}))
	if glyph != glyphJudged || !strings.Contains(note, "repeatability (Determinacy fired, run-1 owed)") {
		t.Errorf("mid + fired + no run: got %s %q; want %s naming the owed run", glyph, note, glyphJudged)
	}
}
