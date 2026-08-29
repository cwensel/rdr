package main

// The fact table's tests.
//
// Two things are being defended here, and they are different.
//
// The EVALUATOR must answer a probe from an exact path and answer
// nothing at all when the root behind it is unbound. Those two cases are
// one line apart in the code and opposite in consequence: `false` routes
// a consumer to re-run a lens, absent tells it nothing was looked at.
//
// The TABLE must keep covering `skills/rdr-status/SKILL.md`. That file is
// prose, this one is data, and the whole point of the change is that they
// stop being two independent descriptions of the same tree. The coverage
// test reads the skill at test time, in the same spirit as the
// same-commit rule's template tests: a signal added to the skill and not
// to the table fails here rather than in a consumer's run.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/model"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// repoFile names a file in the engine repo, two levels above this package.
func repoFile(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join("..", "..", rel)
}

func loadRealTable(t *testing.T) *FactTable {
	t.Helper()
	tbl, err := LoadFactTable(repoFile(t, filepath.Join("models", factTableName)))
	if err != nil {
		t.Fatalf("the shipped fact table does not load: %v", err)
	}
	return tbl
}

// TestShippedFactTableLoads is the first thing that would break if the
// table gained a key or a kind the binary does not read. It is the
// same-commit rule applied to this file: the data and the reader of the
// data ship together.
func TestShippedFactTableLoads(t *testing.T) {
	tbl := loadRealTable(t)
	if tbl.Version != 1 {
		t.Fatalf("version = %d, want 1", tbl.Version)
	}
	if len(tbl.Facts) == 0 {
		t.Fatal("the table declares no facts")
	}
	for _, f := range tbl.Facts {
		if f.Description == "" {
			t.Errorf("fact %q carries no description; the table is read by people too", f.Name)
		}
	}
}

// newEvidenceTree builds a synthetic consumer: an evidence root and a
// records root, in the invented vocabulary the fixtures use. Nothing here
// is drawn from a real record — testdata/README.md's hard rule, and it
// applies to directory names as much as to prose.
func newEvidenceTree(t *testing.T, slug string, evidencePaths, artifactPaths []string) (evidence, records string) {
	t.Helper()
	base := t.TempDir()
	evidence = filepath.Join(base, "evidence")
	records = filepath.Join(base, "records")

	for _, p := range evidencePaths {
		full := filepath.Join(evidence, slug, "evidence", filepath.FromSlash(p))
		mkParents(t, full, p)
	}
	for _, p := range artifactPaths {
		full := filepath.Join(records, slug, filepath.FromSlash(p))
		mkParents(t, full, p)
	}
	return evidence, records
}

// mkParents makes a path; a trailing-slash-free name with an extension is
// a file, everything else a directory.
func mkParents(t *testing.T, full, rel string) {
	t.Helper()
	if strings.HasSuffix(rel, ".md") {
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("Model: synthetic-model\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
}

func factValue(facts []Fact, name string) (string, bool) {
	for _, f := range facts {
		if f.Name == name {
			return f.Value, true
		}
	}
	return "", false
}

// TestProbeReadsAnExactPath: a lens folder under the per-RDR-first tree
// answers true, and a lens that did not run answers false — both decided,
// because the root was bound and the tool actually looked.
func TestProbeReadsAnExactPath(t *testing.T) {
	slug := "0010-frame-header"
	evidence, records := newEvidenceTree(t, slug,
		[]string{"grounding/findings.md", "3amigo/consolidation.md"},
		[]string{"artifacts/gate.md"})

	tbl := loadRealTable(t)
	env := testEnv(t, tbl, slug, evidence, records)
	facts := tbl.Evaluate(env)

	for name, want := range map[string]string{
		"lens_grounding":            "true",
		"lens_grounding_findings":   "true",
		"lens_3amigo":               "true",
		"lens_3amigo_consolidation": "true",
		"lens_critique":             "false",
		"lens_cove":                 "false",
		"gate_written":              "true",
		"impl_capsule":              "false",
	} {
		got, ok := factValue(facts, name)
		if !ok {
			t.Errorf("%s: absent; want %s (the root was bound, so the probe must decide)", name, want)
			continue
		}
		if got != want {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
}

// TestUnboundRootLeavesTheFactAbsent is the rule the whole file turns on.
//
// With no evidence root bound, "the lens did not run" is not something
// this tool knows. Answering false would send a consumer to re-run a lens
// that may have run months ago; the honest answer is to say nothing, the
// same way `resolved` goes absent rather than false when no --repo was
// given. Downstream the omission leaves a rule UNDECIDED where false
// would have decided it.
func TestUnboundRootLeavesTheFactAbsent(t *testing.T) {
	slug := "0010-frame-header"
	_, records := newEvidenceTree(t, slug, nil, []string{"artifacts/gate.md"})

	tbl := loadRealTable(t)
	// Bind the records root and deliberately leave the evidence root unbound.
	env := testEnv(t, tbl, slug, "", records)
	facts := tbl.Evaluate(env)

	for _, name := range []string{"lens_grounding", "lens_critique", "spikes", "reconcile", "iter_2"} {
		if v, ok := factValue(facts, name); ok {
			t.Errorf("%s = %q with no evidence root bound; want the fact absent — "+
				"false says \"looked, not there\" and nothing looked", name, v)
		}
	}
	// The artifact root IS bound, so its probes still decide. Absent must
	// mean "no root", not "the evaluator gave up".
	if v, ok := factValue(facts, "gate_written"); !ok || v != "true" {
		t.Errorf("gate_written = %q/%v; a bound root still decides", v, ok)
	}
}

// TestRootNamingNoDirectoryIsUnbound: a var pointing at a tree that is
// not there is the unbound case, not the false one.
//
// This is the sharper half of the rule above, and the one a consumer
// actually hits: RDR_EVIDENCE left over from a moved checkout, or typo'd
// in a marker. The var is set, so binding it on non-emptiness alone
// roots every probe under a path that cannot exist and answers false for
// all of them — reporting every lens that ran as un-run, which is the
// failure the navigator warns about everywhere else.
//
// The BASE is what is checked. A record that has produced no evidence
// has no <slug>/evidence directory under a perfectly good root, and that
// is a true false which must survive.
func TestRootNamingNoDirectoryIsUnbound(t *testing.T) {
	slug := "0010-frame-header"
	_, records := newEvidenceTree(t, slug, nil, []string{"artifacts/gate.md"})
	tbl := loadRealTable(t)

	t.Setenv("RDR_EVIDENCE", filepath.Join(t.TempDir(), "moved-away"))
	t.Setenv("RDR_RECORDS", records)
	facts := tbl.Evaluate(NewFactEnv(tbl, nil, slug))

	for _, name := range []string{"lens_grounding", "lens_critique", "spikes", "reconcile"} {
		if v, ok := factValue(facts, name); ok {
			t.Errorf("%s = %q under a root that names no directory; want the fact "+
				"absent — nothing looked, so false would report a lens that ran as un-run", name, v)
		}
	}
	// The records root DOES exist, so its probes still decide: the check
	// is per root, never a global give-up.
	if v, ok := factValue(facts, "gate_written"); !ok || v != "true" {
		t.Errorf("gate_written = %q/%v; a root that exists still decides", v, ok)
	}
}

// TestBoundRootWithNoRecordTreeIsFalse guards the other side of the check
// above: the root exists, this record simply has no evidence under it.
// The tool looked, so false is the honest answer and going absent here
// would hide a lens that genuinely has not run.
func TestBoundRootWithNoRecordTreeIsFalse(t *testing.T) {
	slug := "0010-frame-header"
	evidence, records := newEvidenceTree(t, slug, nil, []string{"artifacts/gate.md"})
	tbl := loadRealTable(t)

	// The consumer keeps this tree; only THIS record is absent from it.
	if err := os.MkdirAll(evidence, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RDR_EVIDENCE", evidence)
	t.Setenv("RDR_RECORDS", records)
	facts := tbl.Evaluate(NewFactEnv(tbl, nil, slug))

	if v, ok := factValue(facts, "lens_grounding"); !ok || v != "false" {
		t.Errorf("lens_grounding = %q/%v with the root present and no lens run; "+
			"want false — the tool looked", v, ok)
	}
}

// TestProbeRefusesAPattern: a glob in a probe path is a load error, not a
// path that happens to match nothing. The table cannot ship a guess.
func TestProbeRefusesAPattern(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, factTableName)
	body := `[facts]
version = 1

[root.evidence]
var = "RDR_EVIDENCE"
suffix = "{slug}/evidence"

[fact.lens_any]
kind = "bool"
source = "probe"
root = "evidence"
path = "critique/*.md"
description = "a glob"
`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFactTable(p)
	if err == nil {
		t.Fatal("a globbed probe path loaded; it must be refused")
	}
	if !strings.Contains(err.Error(), "exact path") {
		t.Errorf("error %q does not say why a pattern is refused", err)
	}
}

// TestUnknownKeyIsRefused: the parser never skips what it does not
// understand. A misspelled key that loads as "no value" turns a declared
// fact into a differently-behaving one, silently.
func TestUnknownKeyIsRefused(t *testing.T) {
	for name, body := range map[string]string{
		"unknown key": `[facts]
version = 1
[fact.x]
kind = "bool"
source = "probe"
root = "evidence"
path = "a"
pathh = "typo"
`,
		"unknown source": `[facts]
version = 1
[fact.x]
kind = "bool"
source = "divination"
path = "a"
`,
		"unknown kind": `[facts]
version = 1
[fact.x]
kind = "colour"
source = "field"
path = "a"
`,
		"undeclared root": `[facts]
version = 1
[fact.x]
kind = "bool"
source = "probe"
root = "nowhere"
path = "a"
`,
		"unknown table": `[facts]
version = 1
[whatever.x]
a = "b"
`,
		"duplicate fact": `[facts]
version = 1
[root.evidence]
var = "RDR_EVIDENCE"
suffix = ""
[fact.x]
kind = "bool"
source = "probe"
root = "evidence"
path = "a"
[fact.x]
kind = "bool"
source = "probe"
root = "evidence"
path = "b"
`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, factTableName)
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadFactTable(p); err == nil {
				t.Fatalf("%s loaded; the parser must refuse rather than skip", name)
			}
		})
	}
}

// TestABareFolderIsNotCompletion: the folder says the lens ran, the file
// inside says it finished, and they are separate facts because §lens-row
// says a bare folder is not completion. A critique folder holding only
// the first model's pass is in-progress, not done.
func TestABareFolderIsNotCompletion(t *testing.T) {
	slug := "0011-frame-header"
	evidence, records := newEvidenceTree(t, slug,
		[]string{"critique/critique.md", "repeatability/run-1.md", "repeatability/diff.md"}, nil)

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))

	for name, want := range map[string]string{
		"lens_critique":           "true",
		"lens_critique_single":    "true",
		"lens_critique_modelb":    "false", // the dual-model pass is owed
		"lens_critique_diff":      "false",
		"lens_repeatability":      "true",
		"lens_repeatability_run1": "true",
		"lens_repeatability_run2": "false", // lite variant: only the full row owes these
		"lens_repeatability_run3": "false",
		"lens_repeatability_diff": "true",
		"reconcile":               "false",
		"reconcile_report":        "false",
	} {
		if got, _ := factValue(facts, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

// TestReconcileReportSpellings: the corpus writes the reconcile report
// three ways and no glob may choose between them, so each spelling is its
// own probe and the routing takes the disjunction. A folder with none of
// them is a real state — the stage started and left no report — and must
// not read as done.
func TestReconcileReportSpellings(t *testing.T) {
	for _, spelling := range []struct {
		file string
		fact string
	}{
		{"reconcile/reconcile.md", "reconcile_report"},
		{"reconcile/report.md", "reconcile_report_alt"},
		{"reconcile/reconcile-report.md", "reconcile_report_alt2"},
	} {
		t.Run(spelling.file, func(t *testing.T) {
			slug := "0012-frame-header"
			evidence, records := newEvidenceTree(t, slug, []string{spelling.file}, nil)
			tbl := loadRealTable(t)
			facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
			if got, _ := factValue(facts, "reconcile"); got != "true" {
				t.Errorf("reconcile = %q, want true", got)
			}
			if got, _ := factValue(facts, spelling.fact); got != "true" {
				t.Errorf("%s = %q, want true", spelling.fact, got)
			}
		})
	}

	t.Run("bare folder reports nothing", func(t *testing.T) {
		slug := "0012-frame-header"
		evidence, records := newEvidenceTree(t, slug, []string{"reconcile"}, nil)
		tbl := loadRealTable(t)
		facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
		if got, _ := factValue(facts, "reconcile"); got != "true" {
			t.Errorf("reconcile = %q, want true (the folder is there)", got)
		}
		for _, f := range []string{"reconcile_report", "reconcile_report_alt", "reconcile_report_alt2"} {
			if got, _ := factValue(facts, f); got != "false" {
				t.Errorf("%s = %q, want false (no report was written)", f, got)
			}
		}
	})
}

// TestLegacyEvidenceShapeIsProbed: the migration moved directory-shaped
// evidence and left file-shaped evidence where it was. Those records are
// all terminal today, so this changes no routing — but a claim about the
// corpus that nothing checks is a claim that quietly stops being true,
// and the failure it guards against is a lens that ran reading as un-run.
func TestLegacyEvidenceShapeIsProbed(t *testing.T) {
	slug := "0013-frame-header"
	base := t.TempDir()
	evidence := filepath.Join(base, "evidence")
	records := filepath.Join(base, "records")
	if err := os.MkdirAll(filepath.Join(evidence, "3amigo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidence, "3amigo", slug+".md"), []byte("legacy\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))

	if got, _ := factValue(facts, "legacy_evidence_shape"); got != "true" {
		t.Errorf("legacy_evidence_shape = %q, want true", got)
	}
	// The per-RDR probe still answers false, honestly: nothing is under
	// this record's own folder. The two facts together are what keeps a
	// legacy folder from reading as "lens un-run" with nothing said.
	if got, _ := factValue(facts, "lens_3amigo"); got != "false" {
		t.Errorf("lens_3amigo = %q, want false", got)
	}
}

// testEnv binds an env with explicit roots, bypassing the seam so a test
// never depends on the machine's marker.
func testEnv(t *testing.T, tbl *FactTable, slug, evidence, records string) *FactEnv {
	t.Helper()
	e := &FactEnv{Slug: slug, Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat}
	for name, r := range tbl.Roots {
		var base string
		switch r.Var {
		case "RDR_EVIDENCE":
			base = evidence
		case "RDR_RECORDS":
			base = records
		}
		if base == "" {
			continue
		}
		e.Roots[name] = filepath.Join(base, strings.ReplaceAll(r.Suffix, "{slug}", slug))
	}
	return e
}

// --- the record half -------------------------------------------------------

// TestFieldsComeOffTheProjection: a fact reads what the projector already
// published rather than re-parsing the string it rendered. The Profile's
// rationale tail is the case that matters — §lens-row says match the
// leading word, and a consumer that has to do that itself means the
// projection was missing a field.
func TestFieldsComeOffTheProjection(t *testing.T) {
	doc := scan.Bytes([]byte(`# Recommendation 0014: Frame Header

## Metadata

- **Date**: 2026-01-01
- **Status**: Draft [revised from Final 2026-01-01; re-verify A1 — a narrowed claim]
- **Profile**: mid — one contract plus the metrics surface
- **Cluster**: 0015-frame-trailer, 0016-frame-index

## Problem Statement

Synthetic.
`), scan.Options{})

	tbl := loadRealTable(t)
	env := &FactEnv{Doc: doc, Slug: "0014-frame-header", Roots: map[string]string{},
		readFile: os.ReadFile, statPath: os.Stat}
	facts := tbl.Evaluate(env)

	for name, want := range map[string]string{
		"status":         "Draft",
		"status_form":    "revised-from",
		"status_reentry": "true",
		"status_parked":  "false",
		"profile":        "mid",
		"profile_raw":    "mid — one contract plus the metrics surface",
	} {
		got, ok := factValue(facts, name)
		if !ok {
			t.Errorf("%s: absent, want %q", name, want)
			continue
		}
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	// A set crosses sorted and duplicate-free, which is the form the
	// consuming seam compares by bytes.
	for _, f := range facts {
		if f.Name != "cluster" {
			continue
		}
		if len(f.Members) != 2 || f.Members[0] != "0015" || f.Members[1] != "0016" {
			t.Errorf("cluster members = %v, want [0015 0016]", f.Members)
		}
	}
}

// TestABareStatusHasAFormOfNone: the projector omits an absent qualifier,
// but "this Status carries no qualifier" is a real answer and the routing
// reads it. Absent here would wrongly mean "nothing looked".
func TestABareStatusHasAFormOfNone(t *testing.T) {
	doc := scan.Bytes([]byte(`# Recommendation 0015: Frame Trailer

## Metadata

- **Date**: 2026-01-01
- **Status**: Final

## Problem Statement

Synthetic.
`), scan.Options{})
	tbl := loadRealTable(t)
	facts := tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0015-frame-trailer",
		Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat})

	if got, ok := factValue(facts, "status_form"); !ok || got != "none" {
		t.Errorf("status_form = %q/%v, want \"none\"", got, ok)
	}
	if got, _ := factValue(facts, "status_reentry"); got != "false" {
		t.Errorf("status_reentry = %q, want false", got)
	}
}

// TestPlaceholderAssumptionsAreNotPending is the tally's sharp edge.
//
// A Draft that still carries the template legend `Verified | Pending |
// Unverified` has an UNFILLED list, not a pending one. Counting the
// legend as Pending would report a seeded record as carrying a real
// un-run assumption; dropping it would hide that the list was never
// written. It gets its own bucket and the rollup refuses to certify.
func TestPlaceholderAssumptionsAreNotPending(t *testing.T) {
	doc := scan.Bytes([]byte(`# Recommendation 0016: Frame Index

## Metadata

- **Date**: 2026-01-01
- **Status**: Draft

## Critical Assumptions

- **A1 [The index fits one page]**
  - **Status**: Verified | Pending | Unverified
`), scan.Options{})

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0016-frame-index",
		Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat})

	if got, _ := factValue(facts, "ca_placeholder"); got != "1" {
		t.Errorf("ca_placeholder = %q, want 1", got)
	}
	if got, _ := factValue(facts, "ca_pending"); got != "0" {
		t.Errorf("ca_pending = %q, want 0 — the legend is not a Pending assumption", got)
	}
	if got, _ := factValue(facts, "ca"); got != "unknown-plan" {
		t.Errorf("ca = %q, want unknown-plan — an unfilled list certifies nothing", got)
	}
}

// TestCARollup covers the four shapes the routing branches on, including
// the corpus-only terminal verdicts the template never listed. Refuted is
// a verdict that CLOSES an assumption; counting it as neither terminal
// nor open would leave a finished record reading as mixed forever.
func TestCARollup(t *testing.T) {
	for _, tc := range []struct {
		name     string
		statuses []string
		want     string
	}{
		{"all pending", []string{"Pending", "Pending"}, "all-pending"},
		{"all verified", []string{"Verified", "Verified"}, "all-terminal"},
		{"observed terminal counts as terminal", []string{"Verified", "**Refuted** — the spike disproved it"}, "all-terminal"},
		{"mixed", []string{"Verified", "Pending"}, "mixed"},
		{"unverified is open", []string{"Unverified", "Unverified"}, "all-pending"},
		{"no assumptions at all", nil, "unknown-plan"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			b.WriteString("# Recommendation 0017: Frame Codec\n\n## Metadata\n\n" +
				"- **Date**: 2026-01-01\n- **Status**: Draft\n\n## Critical Assumptions\n\n")
			for i, s := range tc.statuses {
				b.WriteString("- **A" + string(rune('1'+i)) + " [A claim]**\n  - **Status**: " + s + "\n")
			}
			doc := scan.Bytes([]byte(b.String()), scan.Options{})
			tbl := loadRealTable(t)
			facts := tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0017-frame-codec",
				Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat})
			if got, _ := factValue(facts, "ca"); got != tc.want {
				t.Errorf("ca = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestVerdictLinesAreReadFromTheSection: `Premortem:` and `Ground-sweep:`
// are prose the projector does not carry, so the only way to see them is
// to read Decision Rationale's own bytes — bounded by the section's line
// range, never the whole file. `Joint-check:` is NOT here: it projects as
// a JC element with a parsed verdict, so it is a field.
func TestVerdictLinesAreReadFromTheSection(t *testing.T) {
	body := `# Recommendation 0018: Frame Sync

## Metadata

- **Date**: 2026-01-01
- **Status**: Draft

## Decision Rationale

**Premortem:** hardened — the failure modes are written down.

Some prose.

## Consequences

Ground-sweep: this line is in the wrong section and must not count.
`
	dir := t.TempDir()
	p := filepath.Join(dir, "0018-frame-sync.md")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := scan.File(p, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0018-frame-sync",
		Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat})

	if got, _ := factValue(facts, "premortem_line"); got != "true" {
		t.Errorf("premortem_line = %q, want true (emphasis is presentation)", got)
	}
	if got, _ := factValue(facts, "ground_sweep_line"); got != "false" {
		t.Errorf("ground_sweep_line = %q, want false — a line outside "+
			"Decision Rationale is not that section's verdict", got)
	}
}

// TestCapsuleStateReadsTheHeader: the Stage-9 capsule header is the
// authoritative resume read because it is written down. A state word
// further down the file is narrative, not the header, and must not
// answer for it; an absent capsule answers nothing rather than guessing.
func TestCapsuleStateReadsTheHeader(t *testing.T) {
	slug := "0019-frame-writer"
	base := t.TempDir()
	records := filepath.Join(base, "records")
	if err := os.MkdirAll(filepath.Join(records, slug), 0o755); err != nil {
		t.Fatal(err)
	}
	capsule := "# Status\n\n- phase: 3\n- next: run the codec tests\n- blocker: none\n- state: IN-PROGRESS\n"
	if err := os.WriteFile(filepath.Join(records, slug, "status.md"), []byte(capsule), 0o600); err != nil {
		t.Fatal(err)
	}

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, "", records))
	if got, _ := factValue(facts, "impl_state"); got != "IN-PROGRESS" {
		t.Errorf("impl_state = %q, want IN-PROGRESS", got)
	}
	if got, _ := factValue(facts, "impl_capsule"); got != "true" {
		t.Errorf("impl_capsule = %q, want true", got)
	}

	t.Run("no capsule answers nothing", func(t *testing.T) {
		other := t.TempDir()
		recs := filepath.Join(other, "records")
		if err := os.MkdirAll(filepath.Join(recs, slug), 0o755); err != nil {
			t.Fatal(err)
		}
		facts := tbl.Evaluate(testEnv(t, tbl, slug, "", recs))
		if v, ok := factValue(facts, "impl_state"); ok {
			t.Errorf("impl_state = %q with no capsule; want the fact absent", v)
		}
		if got, _ := factValue(facts, "impl_capsule"); got != "false" {
			t.Errorf("impl_capsule = %q, want false — the root was bound", got)
		}
	})
}

// TestEnumFactsStayInTheirDomain: an evaluator that can emit a value its
// own declaration does not list is a fact the consuming model will refuse
// at the seam, and the refusal would surface a long way from the cause.
func TestEnumFactsStayInTheirDomain(t *testing.T) {
	tbl := loadRealTable(t)
	byName := map[string]FactDecl{}
	for _, d := range tbl.Facts {
		byName[d.Name] = d
	}

	// Every lifecycle status the model knows must be in status's domain,
	// and every qualifier form in status_form's. These come from the
	// vocabulary and the qualifier grammar rather than a second list here,
	// so a value added there fails this rather than escaping into a run.
	statusDomain := map[string]bool{}
	for _, v := range byName["status"].Domain {
		statusDomain[v] = true
	}
	for _, v := range append(append([]string{}, modelStatusCanonical()...), modelStatusObserved()...) {
		if !statusDomain[v] {
			t.Errorf("status %q is in the model's vocabulary and not in the fact's domain", v)
		}
	}

	formDomain := map[string]bool{}
	for _, v := range byName["status_form"].Domain {
		formDomain[v] = true
	}
	for _, f := range allQualifierForms() {
		if !formDomain[f] {
			t.Errorf("qualifier form %q is not in status_form's domain", f)
		}
	}

	// `readme_status` is the SAME lifecycle vocabulary, plus two
	// sentinels, and it was the one copy of it chained to nothing.
	//
	// Its two copies (the fact's domain here, and rdr-write's
	// `[tags.readme_status]`) are pinned to EACH OTHER by
	// TestRoutingTagsMatchFactKindAndDomain, so a one-side edit is
	// caught. What was not caught is editing both the same way: adding a
	// status to TEMPLATE.md and walking the three failures that follow
	// leaves `readme_status` behind, and the suite stays green —
	// measured. `readme_status` is a live routing dimension in
	// rdr-write's `readme_status` x `status` product, so the record that
	// carries the new status against a README row is unroutable
	// (flow-guard-unevaluable, exit 2), found by whoever next runs the
	// navigator rather than at build time.
	//
	// The extras are asserted too, and in both directions: the sentinels
	// are what make this domain wider than the vocabulary, so an
	// unexplained third value is drift rather than a sentinel.
	readmeSentinels := map[string]bool{"unindexed": true, "none": true}
	readmeDomain := map[string]bool{}
	for _, v := range byName["readme_status"].Domain {
		readmeDomain[v] = true
	}
	for _, v := range append(append([]string{}, modelStatusCanonical()...), modelStatusObserved()...) {
		if !readmeDomain[v] {
			t.Errorf("status %q is in the model's vocabulary and not in readme_status's domain; "+
				"a record with a README row carrying it would be unroutable", v)
		}
		delete(readmeDomain, v)
	}
	for v := range readmeDomain {
		if !readmeSentinels[v] {
			t.Errorf("readme_status admits %q, which is neither a lifecycle status nor a declared sentinel", v)
		}
		delete(readmeSentinels, v)
	}
	for v := range readmeSentinels {
		t.Errorf("readme_status no longer admits the sentinel %q", v)
	}
}

// The vocabulary and grammar bridges. They exist so the domain test above
// reads its expectations from the model rather than restating them, which
// is what makes it fail when the model gains a value.

func modelStatusCanonical() []string { return model.StatusVocabulary().Canonical }
func modelStatusObserved() []string  { return model.StatusVocabulary().ObservedAccepted }

// allQualifierForms enumerates model.QualifierForm by walking it from its
// zero value until String() stops naming a form. The enum is a run of
// iota constants, so the walk is total by construction and a form added
// to it is caught here rather than at a consumer's seam.
func allQualifierForms() []string {
	var out []string
	for i := 0; ; i++ {
		s := model.QualifierForm(i).String()
		if s == "unknown" {
			return out
		}
		out = append(out, s)
	}
}

// --- the acceptance criterion ---------------------------------------------

// stageFacts maps each row of rdr-status's signal table to the facts that
// answer it. The map is the claim "this row is expressed as data", and
// the test below checks BOTH directions of it against the two files.
//
// A row maps to the facts a MECHANICAL reading of it needs, not to a
// verdict. Stage 3 is human-judged and Stage 4 has an inline-verdict
// escape hatch; a fact can say the CA tallies are all terminal, it cannot
// say "Refine was judged done". The judgment stays in the skill, which is
// why these are inputs to a row rather than answers for one.
var stageFacts = map[string][]string{
	"1 Seed":    {"status"},
	"2 Propose": {"ca_total", "premortem_line", "ground_sweep_line", "joint_checks", "propose_premortem"},
	"3 Refine":  {"ca", "ca_verified", "spikes"},
	"4 Resolve": {"ca", "ca_pending", "ca_verified", "ca_other_terminal", "spikes"},
	"5+6 Pre-Lock (review+resolve)": {
		"profile", "contracts",
		"lens_grounding", "lens_3amigo", "lens_critique", "lens_repeatability", "lens_cove",
		"lens_grounding_findings", "lens_cove_findings", "lens_3amigo_consolidation",
		"lens_critique_single", "lens_critique_modelb", "lens_critique_diff",
		"lens_repeatability_run1", "lens_repeatability_run2", "lens_repeatability_run3",
		"lens_repeatability_diff", "iter_2", "reconcile",
		// Completion, as opposed to "the lens ran": which models wrote
		// critique's two passes, and which variant run-1 declares. The
		// `critique` and `repeatability` outcome groups route on these.
		"critique_models", "repeatability_variant",
	},
	"6 Reconcile": {"reconcile", "reconcile_report", "reconcile_report_alt", "reconcile_report_alt2", "ca"},
	"7 Finalize":  {"status", "gate_written"},
	// 7.1 Cluster reads three different things, and all are facts.
	// `cluster` is what the record DECLARES, which is what the tandem
	// barrier reads and what the row surfaces; `clustered` is that
	// reduced to the yes/no the stage's own precondition asks ("only
	// when the RDR is in a cluster"), because a set cannot be a routing
	// dimension; `cluster_reconciled` is whether a run actually wrote a
	// directory covering this record, which is what routes a Final to
	// /rdr-cluster-reconcile before /rdr-implement.
	//
	// The pair is load-bearing, not redundant: an UNCLUSTERED Final
	// reads `cluster_reconciled=false` correctly — no run covers it —
	// and reading that flag alone would route every solo Final to a
	// stage with nothing to reconcile.
	//
	// `cluster_key` is the fourth: WHICH set a run reconciled, absent
	// when none did. It is what the row prints beside a `true`, and the
	// argument a re-entering 7.1 run is invoked with — the membership
	// read back from the tree rather than re-derived from a claim.
	"7.1 Cluster": {"cluster", "clustered", "cluster_reconciled", "cluster_key"},
	"8 Implement": {"impl_capsule", "impl_state"},
}

// modelTags reads the `[tags.<name>]` keys a routing model declares.
//
// It is READ FROM THE MODELS rather than restated here. The fact names
// are a cross-repo contract, and the whole point of the backward check
// below is that a declared fact nothing reads is dead weight — so "what
// reads it" has to be measured against the files that do the reading. A
// hand-kept list would have to be edited every time a model starts or
// stops guarding on a fact, and the edit that gets forgotten is exactly
// the one this test exists to catch.
func modelTags(t *testing.T, path string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("routing model %s: %v", path, err)
	}
	out := map[string]bool{}
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "[tags.") || !strings.HasSuffix(ln, "]") {
			continue
		}
		out[strings.TrimSuffix(strings.TrimPrefix(ln, "[tags."), "]")] = true
	}
	if len(out) == 0 {
		t.Fatalf("routing model %s declares no [tags.*] — the reader is wrong, not the model", path)
	}
	return out
}

// routingFacts are declared for the "How it decides next" section rather
// than for a signal-table row: the Status qualifier forms that route, and
// the corpus claim the legacy probe defends.
var routingFacts = map[string]string{
	"status_form":           "the qualifier grammar every routing branch reads",
	"status_reentry":        "Draft [revised from Final …] routes to /rdr-resolve",
	"status_parked":         "Deferred [revisit when …] is parked, not terminal",
	"open_joint_decisions":  "a Final waiting on a joint decision",
	"profile_raw":           "the Profile caveat line, verbatim",
	"ca_unverified":         "an Unverified assumption is open, like a Pending one",
	"ca_placeholder":        "an unfilled legend is not a Pending assumption",
	"ca_off_vocabulary":     "a Status in no vocabulary certifies nothing",
	"legacy_evidence_shape": "pre-migration evidence must never read as lens un-run",
	"critique_model_a":      "the stamp the critique dual-model comparison reads",
	"critique_model_b":      "the stamp the critique dual-model comparison reads",
	"contracts_prose":       "the Determinacy trigger's input: contracts written as prose, which a zero C count cannot see",
}

// TestEveryStageRowIsExpressedAsFacts is the issue's acceptance criterion,
// checked in both directions.
//
// Forward: every row of the skill's signal table maps to facts, and each
// of those facts is really declared in the table. A row added to the
// skill with no facts fails here — which is the whole point of moving the
// signals into data, since two prose descriptions of one tree drift and
// nothing notices.
//
// Backward: every fact in the table is claimed by some row or named as a
// routing fact. A fact nothing reads is a signal nobody asked for, and
// the table is a contract other repos match on by name.
func TestEveryStageRowIsExpressedAsFacts(t *testing.T) {
	skill, err := os.ReadFile(repoFile(t, filepath.Join("skills", "rdr-status", "SKILL.md")))
	if err != nil {
		t.Fatalf("the skill this table mirrors is unreadable: %v", err)
	}
	rows := signalTableRows(string(skill))
	if len(rows) == 0 {
		t.Fatal("no signal-table rows found; the skill's table moved and this test is blind")
	}

	tbl := loadRealTable(t)
	declared := map[string]bool{}
	for _, f := range tbl.Facts {
		declared[f.Name] = true
	}

	// Forward: the skill's rows are covered, and by facts that exist.
	claimed := map[string]bool{}
	for _, row := range rows {
		facts, ok := stageFacts[row]
		if !ok {
			t.Errorf("signal-table row %q has no facts; add them to the table and to stageFacts", row)
			continue
		}
		if len(facts) == 0 {
			t.Errorf("signal-table row %q claims no facts", row)
		}
		for _, name := range facts {
			if !declared[name] {
				t.Errorf("row %q names fact %q, which the table does not declare", row, name)
			}
			claimed[name] = true
		}
	}

	// stageFacts must not describe rows the skill no longer has, or the
	// coverage above is measured against a table that moved on.
	present := map[string]bool{}
	for _, r := range rows {
		present[r] = true
	}
	for row := range stageFacts {
		if !present[row] {
			t.Errorf("stageFacts covers row %q, which is not in the skill's signal table", row)
		}
	}

	// A fact a ROUTING MODEL guards on is read, whichever model it is.
	// Both are checked, so a fact that moved from one table to the other
	// stays covered and one that left both is reported.
	routedByModel := map[string]bool{}
	for _, m := range []string{"rdr-status.toml", "rdr-write.toml"} {
		for k := range modelTags(t, filepath.Join("..", "..", "models", m)) {
			routedByModel[k] = true
		}
	}

	// Backward: no fact is declared that nothing reads.
	for _, f := range tbl.Facts {
		if claimed[f.Name] {
			continue
		}
		if _, ok := routingFacts[f.Name]; ok {
			continue
		}
		if routedByModel[f.Name] {
			continue
		}
		t.Errorf("fact %q is declared and no signal-table row or routing rule reads it; "+
			"the fact names are a contract, so an unread one is dead weight another repo may bind to", f.Name)
	}

	for name := range routingFacts {
		if !declared[name] {
			t.Errorf("routingFacts names %q, which the table does not declare", name)
		}
	}
}

// signalTableRows reads the Stage column out of the skill's signal table.
// It is deliberately keyed on the table's own shape — a markdown row
// whose first cell names a stage — so a table that moves or is rewritten
// makes this test fail loudly rather than silently pass over nothing.
func signalTableRows(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| ") || !strings.Contains(line, " | ") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 2 {
			continue
		}
		stage := strings.TrimSpace(cells[0])
		if stage == "" || stage == "Stage" || strings.HasPrefix(stage, "---") {
			continue
		}
		// The signal table's rows lead with a stage number; the skill's
		// other tables do not.
		if stage[0] < '0' || stage[0] > '9' {
			continue
		}
		out = append(out, stage)
	}
	return out
}

// TestFactTablePathFindsTheShippedTable: the table is found from
// $RDR_HOME, and an explicit path outranks it. The fallback beside the
// binary is what keeps a directly-invoked build working with no marker;
// it is not exercised here because a test binary lives in a temp dir by
// design, which is the case the $RDR_HOME lookup exists to cover.
func TestFactTablePathFindsTheShippedTable(t *testing.T) {
	t.Setenv("RDR_HOME", filepath.Join("..", ".."))
	got, err := factTablePath("")
	if err != nil {
		t.Fatalf("the shipped table was not found from $RDR_HOME: %v", err)
	}
	if filepath.Base(got) != factTableName {
		t.Errorf("found %q, want a path ending in %s", got, factTableName)
	}

	t.Run("explicit wins", func(t *testing.T) {
		got, err := factTablePath("/somewhere/else.toml")
		if err != nil || got != "/somewhere/else.toml" {
			t.Errorf("factTablePath(explicit) = %q/%v; a caller who spells a path means it", got, err)
		}
	})

	t.Run("no table is a stopped reason, not a panic", func(t *testing.T) {
		t.Setenv("RDR_HOME", t.TempDir())
		if _, err := factTablePath(""); err == nil {
			t.Error("a missing table must report stopped:no-fact-table")
		}
	})
}

// TestEmitFactsRendersTheVector pins the envelope 0158 will hand to a
// resolver: schema, record, and the facts in declaration order. A set
// carries members and no scalar value, so a consumer never has to guess
// which field to read.
func TestEmitFactsRendersTheVector(t *testing.T) {
	var out, errOut strings.Builder
	code := emitFacts([]Fact{
		{Name: "status", Kind: "enum", Value: "Draft"},
		{Name: "cluster", Kind: "set", Members: []string{"0015", "0016"}},
	}, "0014", &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got struct {
		Schema string `json:"schema"`
		Record string `json:"record"`
		Facts  []Fact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("the vector is not valid JSON: %v", err)
	}
	if got.Record != "0014" || got.Schema != schemaVersion {
		t.Errorf("schema/record = %q/%q", got.Schema, got.Record)
	}
	if len(got.Facts) != 2 || got.Facts[0].Name != "status" {
		t.Fatalf("facts = %+v; declaration order is the contract", got.Facts)
	}
	if got.Facts[1].Value != "" || len(got.Facts[1].Members) != 2 {
		t.Errorf("a set carries members and no scalar value: %+v", got.Facts[1])
	}
}

// TestClusterMemberSegmentsTheEpochs is the boundary this fact is built
// on. Stage 7.1's directory is keyed by the cluster, and the corpus holds
// TWO keying conventions: a current shape whose key is the members'
// numbers joined, and an earlier topical shape named for the subject.
//
// The current shape is exact — the key IS the membership — so it is read.
// The topical shape is excluded BY SHAPE rather than read and filtered,
// and the reason is the `2026` case below: `final-cluster-2026-06-22` is
// a real directory in the reference corpus whose name contains a
// well-formed four-digit record number that names no member of anything.
// A rule that pulled numbers out of a name would report Stage 7.1 as
// having reconciled a record 2026 that no run ever touched — the exact
// inversion (a stage reading as run when it did not) that the no-globs
// rule exists to prevent, in the direction that is harder to notice.
func TestClusterMemberSegmentsTheEpochs(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"0021-0022",                // current shape: the key is the membership
		"0021-0022-0023",           // the widened re-run; nested overlap, not ambiguity
		"final-cluster-2026-06-22", // topical epoch, and the 2026 trap
		"dml-purpose",              // topical epoch, no numbers at all
		"replay-perf-2026-06-25",   // topical epoch, dated
	} {
		if err := os.MkdirAll(filepath.Join(root, "cluster-reconcile", dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A loose FILE whose name looks like a key must not answer either:
	// only a directory is a run's output.
	if err := os.WriteFile(filepath.Join(root, "cluster-reconcile", "0031-0032"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	decl := FactDecl{Name: "cluster_reconciled", Kind: "bool",
		Source: "cluster-member", Root: "evidence-root", Path: "cluster-reconcile"}

	for _, c := range []struct {
		slug string
		want string
		why  string
	}{
		{"0021-cache-warmup-order", "true", "a member of the current-shape key"},
		{"0022-cache-metrics-surface", "true", "in BOTH dirs — a widened re-run answers true once"},
		{"0023-cache-shard-count", "true", "admitted by the second iteration only"},
		{"0024-cache-persistence", "false", "in no cluster: 7.1 is what runs next"},
		{"2026-a-record-that-never-clustered", "false",
			"THE GUARD: final-cluster-2026-06-22 must not make 2026 a member"},
		{"0006-topical-era-member", "false",
			"the topical epoch is out of scope by shape, not covered by a number rule"},
		{"0031-loose-file-not-a-dir", "false", "a file named like a key is not a run"},
	} {
		e := &FactEnv{Slug: c.slug, Roots: map[string]string{"evidence-root": root},
			readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
		got, ok := e.clusterMember(decl)
		if !ok {
			t.Errorf("%s: the fact went absent though the root is bound", c.slug)
			continue
		}
		if got.Value != c.want {
			t.Errorf("%s: cluster_reconciled = %s, want %s — %s", c.slug, got.Value, c.want, c.why)
		}
	}

	// An unbound root is absent, never false: "no evidence root is
	// configured" and "7.1 has not run" are different answers.
	e := &FactEnv{Slug: "0021-cache-warmup-order", Roots: map[string]string{},
		readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
	if _, ok := e.clusterMember(decl); ok {
		t.Error("an unbound root answered instead of going absent")
	}

	// A bound root with no cluster-reconcile tree at all IS false: the
	// tool looked, and nothing has ever been reconciled here.
	e = &FactEnv{Slug: "0021-cache-warmup-order", Roots: map[string]string{"evidence-root": t.TempDir()},
		readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
	got, ok := e.clusterMember(decl)
	if !ok || got.Value != "false" {
		t.Errorf("a bound root with no tree = %+v/%v, want false", got, ok)
	}
}

// writeEvidence puts exact bytes at a path under the evidence tree, for
// the header-reading facts: what these read is the CONTENT of a file, so
// the shared helper's placeholder body cannot drive them.
func writeEvidence(t *testing.T, evidence, slug, rel, body string) {
	t.Helper()
	full := filepath.Join(evidence, slug, "evidence", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestCritiqueModelsComparesTheStamps covers the four states the corpus
// actually holds, because each routes differently: `differ` is the
// dual-model pass §model-stamp wants, `same` is the sanctioned
// single-model fallback, `unknown` is a pass with no stamp (which the
// rule says never to read as a match), and `absent` is no second pass.
func TestCritiqueModelsComparesTheStamps(t *testing.T) {
	for _, c := range []struct {
		name       string
		passA      string
		passB      string
		wantModels string
	}{
		{"two models", "model: claude-opus-5\n", "model: claude-sonnet-5\n", "differ"},
		{"one model, fresh contexts", "model: claude-opus-5\n", "model: claude-opus-5\n", "same"},
		{"second pass unstamped", "model: claude-opus-5\n", "# no stamp here\n", "unknown"},
		{"first pass unstamped", "# no stamp here\n", "model: claude-opus-5\n", "unknown"},
		// A trailing note is prose for a human, never part of the id:
		// the same model must not read as two.
		{"trailing prose ignored",
			"model: claude-opus-5[1m]   (fresh-context run A; single-model fallback)\n",
			"model: claude-opus-5[1m]   (fresh-context run B)\n", "same"},
	} {
		t.Run(c.name, func(t *testing.T) {
			slug := "0010-frame-header"
			evidence, records := newEvidenceTree(t, slug, nil, nil)
			writeEvidence(t, evidence, slug, "critique/critique.md", c.passA)
			writeEvidence(t, evidence, slug, "critique/critique-modelB.md", c.passB)

			tbl := loadRealTable(t)
			facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
			if got, _ := factValue(facts, "critique_models"); got != c.wantModels {
				t.Errorf("critique_models = %q, want %q", got, c.wantModels)
			}
		})
	}
}

// TestCritiqueModelsAbsentWithNoSecondPass separates "no second pass" from
// "a second pass with no stamp". Both leave the stamp unreadable and they
// are different states: one owes a pass, the other owes nothing.
func TestCritiqueModelsAbsentWithNoSecondPass(t *testing.T) {
	slug := "0010-frame-header"
	evidence, records := newEvidenceTree(t, slug, nil, nil)
	writeEvidence(t, evidence, slug, "critique/critique.md", "model: claude-opus-5\n")

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
	if got, _ := factValue(facts, "critique_models"); got != "absent" {
		t.Errorf("critique_models = %q with no critique-modelB.md, want absent", got)
	}
}

// TestRepeatabilityVariantReadsTheHeader is §repeatability-variant's own
// rule: the intended variant lives on disk, and the file COUNT must never
// stand in for it — that is what silently promotes a large RDR to full x3.
func TestRepeatabilityVariantReadsTheHeader(t *testing.T) {
	for _, c := range []struct {
		name, run1, want string
	}{
		{"lite", "model: m\nvariant: lite (profile: large)\n", "lite"},
		{"full", "model: m\nvariant: full (profile: foundational)\n", "full"},
		// The escalation form is still the full variant; the reason in
		// parentheses is a note, not part of the value.
		{"escalated", "model: m\nvariant: full (escalated: accretion floor)\n", "full"},
	} {
		t.Run(c.name, func(t *testing.T) {
			slug := "0010-frame-header"
			evidence, records := newEvidenceTree(t, slug, nil, nil)
			writeEvidence(t, evidence, slug, "repeatability/run-1.md", c.run1)

			tbl := loadRealTable(t)
			facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
			if got, _ := factValue(facts, "repeatability_variant"); got != c.want {
				t.Errorf("repeatability_variant = %q, want %q", got, c.want)
			}
		})
	}
}

// TestRepeatabilityVariantAbsentWithoutAHeader is the case 27 run-1 files
// in the corpus are in: written before the header existed. The variant is
// genuinely unknown, and the routing must stop rather than infer one from
// how many run files are present.
func TestRepeatabilityVariantAbsentWithoutAHeader(t *testing.T) {
	slug := "0010-frame-header"
	evidence, records := newEvidenceTree(t, slug, nil, nil)
	writeEvidence(t, evidence, slug, "repeatability/run-1.md", "# Repeatability run 1\n\nNo header.\n")
	writeEvidence(t, evidence, slug, "repeatability/run-2.md", "# run 2\n")

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
	if got, ok := factValue(facts, "repeatability_variant"); ok && got != "" {
		t.Errorf("repeatability_variant = %q; an unstamped run-1 must not report a variant, "+
			"and two run files must never be read as `full`", got)
	}
}

// TestHeaderFieldRefusesAnOffDomainValue keeps an evaluator inside the
// domain its own table declares: a stamp naming something else is not a
// new member to export, it is absent.
func TestHeaderFieldRefusesAnOffDomainValue(t *testing.T) {
	slug := "0010-frame-header"
	evidence, records := newEvidenceTree(t, slug, nil, nil)
	writeEvidence(t, evidence, slug, "repeatability/run-1.md", "variant: exhaustive\n")

	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, records))
	if got, ok := factValue(facts, "repeatability_variant"); ok && got != "" {
		t.Errorf("repeatability_variant = %q; `exhaustive` is not in the declared domain", got)
	}
}

// TestContractsProseSeparatesTemplateFromAuthored is the distinction the
// fact exists for, and it is the one a zero `contracts` count cannot make.
//
// A section holding only the template's guidance and a draft placeholder
// is EMPTY however many lines it spans — measured on the corpus, that is
// cli/0069's 52 lines. A section of prose contracts is NOT empty even
// though nothing in it is labelled — cli/0053's 27 lines. Reading the
// count alone conflates them, which is why four skills were told to stop
// and read the section by hand.
func TestContractsProseSeparatesTemplateFromAuthored(t *testing.T) {
	for _, c := range []struct {
		name, body string
		want       string
	}{
		{"draft placeholder only", "_Draft placeholder — /rdr-propose._\n", "false"},
		{"wrapped draft placeholder",
			"_Draft placeholder — /rdr-propose. Single load-bearing contract\nexpected; watch the split signal._\n", "false"},
		{"prose contract, unlabelled",
			"Composite fields are author-ordered in the wire, and the resolver\nmints one name per relation for the life of the corpus.\n", "true"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "0010-frame-header.md")
			doc := "# Recommendation 0010: Frame header\n\n## Normative Contracts\n\n" + c.body
			if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
				t.Fatal(err)
			}
			d, err := scan.File(path, scan.Options{})
			if err != nil {
				t.Fatal(err)
			}
			tbl := loadRealTable(t)
			env := NewFactEnv(tbl, d, "0010-frame-header")
			if got, _ := factValue(tbl.Evaluate(env), "contracts_prose"); got != c.want {
				t.Errorf("contracts_prose = %q, want %q, for:\n%s", got, c.want, c.body)
			}
		})
	}
}

// readmeIndex writes a records-dir index README carrying the given rows.
func readmeIndex(t *testing.T, records string, rows ...string) {
	t.Helper()
	body := "# RDRs\n\n## Index\n\n| RDR | Title | Status | Priority |\n| --- | --- | --- | --- |\n"
	for _, r := range rows {
		body += r + "\n"
	}
	if err := os.WriteFile(filepath.Join(records, "README.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// recordAt writes a minimal record and returns its parsed document.
func recordAt(t *testing.T, records, slug, status string) *scan.Document {
	t.Helper()
	path := filepath.Join(records, slug+".md")
	num := slug[:4]
	body := "# Recommendation " + num + ": Synthetic\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-28\n- **Status**: " + status + "\n- **Profile**: small\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := scan.File(path, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestReadmeRowIsThreeValued is the whole contract of the fact, and each
// of the three answers drives a different write op.
//
// The distinction that matters is `none` vs absent. "The table was read
// and this record has no row" is the exact condition `readme --add`
// exists for, so it is a DECLARED value a routing row can claim. "There
// is no README, or it carries no index table" is nothing-looked, and a
// write op must not act on it at all — inventing a row from an unreadable
// index is how a second index gets written beside the real one.
func TestReadmeRowIsThreeValued(t *testing.T) {
	for _, c := range []struct {
		name  string
		rows  []string
		noIdx bool
		want  string
		ok    bool
	}{
		{name: "row says Final",
			rows: []string{"| [0009](0009-thing.md) | Thing | Final | High |"},
			want: "Final", ok: true},
		{name: "row says Draft",
			rows: []string{"| [0009](0009-thing.md) | Thing | Draft | High |"},
			want: "Draft", ok: true},
		{name: "table read, this record has no row",
			rows: []string{"| [0001](0001-other.md) | Other | Final | High |"},
			want: "none", ok: true},
		{name: "no README at all — nothing looked", noIdx: true, ok: false},
	} {
		t.Run(c.name, func(t *testing.T) {
			records := t.TempDir()
			doc := recordAt(t, records, "0009-thing", "Draft")
			if !c.noIdx {
				readmeIndex(t, records, c.rows...)
			}
			tbl := loadRealTable(t)
			env := testEnv(t, tbl, "0009-thing", "", records)
			env.Doc = doc
			got, ok := factValue(tbl.Evaluate(env), "readme_status")
			if ok != c.ok {
				t.Fatalf("present = %v, want %v (value %q)", ok, c.ok, got)
			}
			if ok && got != c.want {
				t.Errorf("readme_status = %q, want %q", got, c.want)
			}
		})
	}
}

// TestReadmeRowWithoutATableIsAbsentNotNone: a README that is prose only
// says NOTHING about whether this record is indexed. Answering `none`
// there would send `readme --add` at a records dir whose index lives
// somewhere this parser cannot see, and it would append a second table.
func TestReadmeRowWithoutATableIsAbsentNotNone(t *testing.T) {
	records := t.TempDir()
	doc := recordAt(t, records, "0009-thing", "Draft")
	if err := os.WriteFile(filepath.Join(records, "README.md"),
		[]byte("# RDRs\n\nThe index lives elsewhere.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tbl := loadRealTable(t)
	env := testEnv(t, tbl, "0009-thing", "", records)
	env.Doc = doc
	if got, ok := factValue(tbl.Evaluate(env), "readme_status"); ok {
		t.Errorf("readme_status = %q on a README with no index table; want absent", got)
	}
}

// TestReadmeRowMatchesOnTheNumberNotTheTitle: the number is the key the
// table is written on. A title drifts — `index --readme` reports title
// drift as an ordinary finding — so matching on it would make the fact
// disagree with the facet on exactly the records that need fixing.
func TestReadmeRowMatchesOnTheNumberNotTheTitle(t *testing.T) {
	records := t.TempDir()
	doc := recordAt(t, records, "0009-thing", "Draft")
	readmeIndex(t, records, "| [0009](0009-thing.md) | A Stale Title | Final | High |")
	tbl := loadRealTable(t)
	env := testEnv(t, tbl, "0009-thing", "", records)
	env.Doc = doc
	got, ok := factValue(tbl.Evaluate(env), "readme_status")
	if !ok || got != "Final" {
		t.Errorf("readme_status = %q (present %v); a drifted title must not hide the row", got, ok)
	}
}
