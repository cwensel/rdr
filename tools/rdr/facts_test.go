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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
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

	for _, name := range []string{"lens_grounding", "lens_critique", "spikes", "reconcile", "iter_depth"} {
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
	for _, r := range tbl.Roots {
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
		e.bindRoot(r, base)
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

// TestReentryTargetIsAbsentUntilWritten: the `@<stage>` slot is a fact
// only when the qualifier names it. A historical re-entry (no slot) and
// a bare Draft both leave it ABSENT in plain output — absent ≠ false,
// nothing was said — and `--tags` renders the declared `none` sentinel
// so the routing table's fallback row can claim the cell.
func TestReentryTargetIsAbsentUntilWritten(t *testing.T) {
	tbl := loadRealTable(t)
	eval := func(status string) []Fact {
		doc := scan.Bytes([]byte("# Recommendation 0017: Frame Pad\n\n## Metadata\n\n- **Date**: 2026-01-01\n- **Status**: "+
			status+"\n\n## Problem Statement\n\nSynthetic.\n"), scan.Options{})
		return tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0017-frame-pad",
			Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat})
	}
	for status, want := range map[string]string{
		"Draft [revised from Final 2026-01-01; re-verify A15, A17 @refine — a contract edit]": "refine",
		"Draft [revised from Final 2026-01-01; re-verify none @propose — approach reopened]":  "propose",
		"Draft [revised from Final 2026-01-01; @resolve — a narrowed claim]":                  "resolve",
	} {
		if got, ok := factValue(eval(status), "reentry_target"); !ok || got != want {
			t.Errorf("%s: reentry_target = %q/%v, want %q", status, got, ok, want)
		}
	}
	for _, status := range []string{
		"Draft [revised from Final 2026-01-01; re-verify A1 — a narrowed claim]",
		"Draft",
		"Final",
	} {
		facts := eval(status)
		if got, ok := factValue(facts, "reentry_target"); ok {
			t.Errorf("%s: reentry_target = %q, want absent", status, got)
		}
		if got, ok := factValue(withAbsentSentinels(tbl, facts, nil), "reentry_target"); !ok || got != "none" {
			t.Errorf("%s: --tags reentry_target = %q/%v, want the none sentinel", status, got, ok)
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
	// WHICH assumptions are open, and which a re-entry reopened: the
	// gate questions the tallies alone could not answer.
	"4 Resolve": {"ca", "ca_pending", "ca_pending_ids", "reverify", "ca_verified", "ca_other_terminal", "spikes"},
	"5+6 Pre-Lock (review+resolve)": {
		"profile", "contracts",
		"lens_grounding", "lens_3amigo", "lens_critique", "lens_repeatability", "lens_cove",
		"lens_grounding_findings", "lens_cove_findings", "lens_3amigo_consolidation",
		"lens_critique_single", "lens_critique_modelb", "lens_critique_diff",
		"lens_repeatability_run1", "lens_repeatability_run2", "lens_repeatability_run3",
		"lens_repeatability_diff", "iter_depth", "lens_findings_open", "reconcile",
		// Completion, as opposed to "the lens ran": which models wrote
		// critique's two passes, and which variant run-1 declares. The
		// `critique` and `repeatability` outcome groups route on these.
		"critique_models", "repeatability_variant",
		// The Stage-5 Determinacy judgement as the `Determinacy:` line
		// records it; the `determinacy` group routes the lite add-on on it.
		"determinacy",
		// The accretion floor's inputs. The floor itself is the `floor`
		// outcome's row over the bucket and the disposition; the count is
		// published for the reader and routes nothing directly.
		"seam_lineage_count", "seam_lineage", "accretion_disposition",
	},
	"6 Reconcile": {"reconcile", "reconcile_report", "reconcile_report_alt", "reconcile_report_alt2", "ca"},
	// The anchor and peer-evidence tallies are §mechanical-gate's
	// numbers; three-valued, so unlooked travels with total.
	// On demand: evaluated only when --filter names them.
	"7 Finalize": {"status", "gate_written", "gate_stale", "rulings_open",
		"anchors_total", "anchors_unresolved", "anchors_unlooked", "peer_evidence_unresolved"},
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
	"8 Implement": {"impl_capsule", "impl_state", "lines", "req_count", "impl_orphans",
		"impl_open_decisions", "impl_deviation_types_unknown", "impl_mvv_recorded", "impl_verification_recorded",
		"impl_findings_open"},
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
// modelGuardKeys is the set of tags some row actually guards on — the
// dimensions, as opposed to the keys merely declared. Mirrors modelTags'
// line reading rather than parsing TOML, for the same reason.
func modelGuardKeys(t *testing.T, path string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("routing model %s: %v", path, err)
	}
	out := map[string]bool{}
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "[rule.guard.") || !strings.HasSuffix(ln, "]") {
			continue
		}
		// `[rule.guard.all.<key>]`, `[rule.guard.unless.<key>]`, `[rule.guard.any.<key>]`
		body := strings.TrimSuffix(strings.TrimPrefix(ln, "[rule.guard."), "]")
		if _, key, ok := strings.Cut(body, "."); ok {
			out[key] = true
		}
	}
	return out
}

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
	"status_reentry":        "Draft [revised from Final …] is a scoped re-entry",
	"reentry_target":        "the @<stage> a re-entry names; none routes to the resolve fallback",
	"status_parked":         "Deferred [revisit when …] is parked, not terminal",
	"open_joint_decisions":  "a Final waiting on a joint decision",
	"profile_raw":           "the Profile caveat line, verbatim",
	"ca_unverified":         "an Unverified assumption is open, like a Pending one",
	"ca_placeholder":        "an unfilled legend is not a Pending assumption",
	"ca_off_vocabulary":     "a Status in no vocabulary certifies nothing",
	"ca_off_vocabulary_ids": "which assumptions carry that Status, so the gate names them",
	"legacy_evidence_shape": "pre-migration evidence must never read as lens un-run",
	"critique_model_a":      "the stamp the critique dual-model comparison reads",
	"critique_model_b":      "the stamp the critique dual-model comparison reads",
	"contracts_prose":       "the Determinacy trigger's input: contracts written as prose, which a zero C count cannot see",
	"contracts_transient":   "how many labelled contracts the Transient marker excludes from the Profile axis",
	"contracts_durable":     "the Profile contract axis: labelled minus Transient, bucketed, subtracted by the projector",
	"spikes_unrun":          "Stage 6's third open-set source: spikes the record names with no run on disk, as a set rather than a walk",

	// the rollups: the set behind a routed word, and the two set questions
	// no row guards because they are about another record
	"predecessors_incomplete":     "the predecessors the launch precheck's stopped:predecessor-incomplete names",
	"predecessors_retired":        "the predecessors the launch precheck's stopped:predecessor-retired names — closed without implementing, so no capsule exists and none ever will",
	"cluster_members_proposed":    "the Stage-2 tandem barrier: every sibling's plan authored",
	"related_final_unimplemented": "Finalize's next step: a related Final still unimplemented routes to 7.1 first",
	"cluster_unordered":           "the cluster siblings behind related_final_unordered: implement them first, or their Predecessors must name this record",
	"cluster_members_in_flight":   "the after-lock group's other half: a cluster sibling still Draft, caught before a 7.1 pass runs on a partial set",
	"prerequisites_owed":          "the records behind prerequisites_unimplemented: the launch precheck's stopped:prerequisite-unimplemented names them",
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
	for _, m := range routingModelNames {
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

// TestContractFactsSubtractTransient: the Profile contract axis excludes
// Transient-marked contracts (TEMPLATE.md), and the subtraction is the
// projector's — the durable bucket arrives computed, so the `profile`
// rows compare and never count.
func TestContractFactsSubtractTransient(t *testing.T) {
	fence := func(label, body string) string {
		return "**" + label + "**\n\n```normative\n" + body + "\n```\n\n"
	}
	transient := "Transient — scheduled deletion by 0032-frame-bridge, phase 2; bridge only"
	for _, c := range []struct {
		name, body, transientN, durable string
	}{
		{"prose only", "Composite fields are author-ordered in the wire.\n", "0", "0"},
		{"one durable", fence("C1", "func Encode(f Frame) []byte"), "0", "1"},
		{"one durable, one transient", fence("C1", "func Encode(f Frame) []byte") + fence("C2", "func EncodeLegacy(f Frame) []byte\n"+transient), "1", "1"},
		{"two durable, one transient", fence("C1", "a") + fence("C2", "b") + fence("C3", "c\n"+transient), "1", "2+"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "0011-frame-codec.md")
			doc := "# Recommendation 0011: Frame codec\n\n## Normative Contracts\n\n" + c.body
			if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
				t.Fatal(err)
			}
			d, err := scan.File(path, scan.Options{})
			if err != nil {
				t.Fatal(err)
			}
			tbl := loadRealTable(t)
			facts := tbl.Evaluate(NewFactEnv(tbl, d, "0011-frame-codec"))
			if got, _ := factValue(facts, "contracts_transient"); got != c.transientN {
				t.Errorf("contracts_transient = %q, want %q", got, c.transientN)
			}
			if got, _ := factValue(facts, "contracts_durable"); got != c.durable {
				t.Errorf("contracts_durable = %q, want %q", got, c.durable)
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

// TestReadmeRowReadsAnUnlinkedRow: the link is the convention, the number
// is the key. Read as no row, the record answers `none` and routes to
// `readme --add`, which appends a second row beside the one already
// there — the exact shape the `none`/absent split exists to prevent.
func TestReadmeRowReadsAnUnlinkedRow(t *testing.T) {
	records := t.TempDir()
	doc := recordAt(t, records, "0009-thing", "Draft")
	readmeIndex(t, records, "| 0009 | Thing | Final | High |")
	tbl := loadRealTable(t)
	env := testEnv(t, tbl, "0009-thing", "", records)
	env.Doc = doc
	got, ok := factValue(tbl.Evaluate(env), "readme_status")
	if !ok || got != "Final" {
		t.Errorf("readme_status = %q (present %v); an unlinked row is a row", got, ok)
	}
}

// TestStaleLensDatesEvidenceAgainstTheDemote covers the four readings
// the freshness facts make: a content `Date:` outranks the mtime, the
// mtime dates a file with no stamp, a re-run under iter-N makes the lens
// current however old the loose pass is, and a record with no demote
// date can never read stale at all.
func TestStaleLensDatesEvidenceAgainstTheDemote(t *testing.T) {
	tbl := loadRealTable(t)
	slug := "0029-synthetic-reentry"
	reentered := "Draft [revised from Final 2026-08-20; re-verify none — synthetic rework]"
	old := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)

	run := func(t *testing.T, status string, lay func(evidence, records string)) []Fact {
		t.Helper()
		evidence, records := newEvidenceTree(t, slug, nil, nil)
		if err := os.MkdirAll(records, 0o755); err != nil {
			t.Fatal(err)
		}
		lay(evidence, records)
		doc := recordAt(t, records, slug, status)
		e := testEnv(t, tbl, slug, evidence, records)
		e.Doc, e.readDir = doc, os.ReadDir
		return tbl.Evaluate(e)
	}
	touch := func(t *testing.T, evidence, rel string, when time.Time) {
		t.Helper()
		full := filepath.Join(evidence, slug, "evidence", filepath.FromSlash(rel))
		if err := os.Chtimes(full, when, when); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a Date header predating the demote is stale whatever the mtime says", func(t *testing.T) {
		facts := run(t, reentered, func(ev, _ string) {
			writeEvidence(t, ev, slug, "grounding/findings.md", "Model: m\nDate: 2026-08-10. Verdict: ok\n")
		})
		if v, _ := factValue(facts, "lens_stale"); v != "grounding" {
			t.Fatalf("lens_stale = %q, want grounding", v)
		}
	})
	t.Run("an unstamped file is dated by its mtime", func(t *testing.T) {
		facts := run(t, reentered, func(ev, _ string) {
			writeEvidence(t, ev, slug, "3amigo/consolidation.md", "Model: m\n")
			touch(t, ev, "3amigo/consolidation.md", old)
		})
		if v, _ := factValue(facts, "lens_stale"); v != "3amigo" {
			t.Fatalf("lens_stale = %q, want 3amigo", v)
		}
		facts = run(t, reentered, func(ev, _ string) {
			writeEvidence(t, ev, slug, "3amigo/consolidation.md", "Model: m\n")
		})
		if v, _ := factValue(facts, "lens_stale"); v != "none" {
			t.Fatalf("a file written now: lens_stale = %q, want none", v)
		}
	})
	t.Run("a re-run under iter-N makes the lens current", func(t *testing.T) {
		facts := run(t, reentered, func(ev, _ string) {
			writeEvidence(t, ev, slug, "cove/findings.md", "Model: m\n")
			touch(t, ev, "cove/findings.md", old)
			writeEvidence(t, ev, slug, "cove/iter-2/findings.md", "Model: m\nDate: 2026-08-20\n")
		})
		if v, _ := factValue(facts, "lens_stale"); v != "none" {
			t.Fatalf("lens_stale = %q, want none (same-day iter-2 is fresh)", v)
		}
	})
	t.Run("the first stale lens in row order is named", func(t *testing.T) {
		facts := run(t, reentered, func(ev, _ string) {
			writeEvidence(t, ev, slug, "cove/findings.md", "Date: 2026-08-21\n")
			writeEvidence(t, ev, slug, "3amigo/consolidation.md", "Date: 2026-08-01\n")
			writeEvidence(t, ev, slug, "critique/critique.md", "Date: 2026-08-01\n")
		})
		if v, _ := factValue(facts, "lens_stale"); v != "3amigo" {
			t.Fatalf("lens_stale = %q, want 3amigo", v)
		}
	})
	t.Run("a Final and a first-pass Draft never read stale", func(t *testing.T) {
		for _, status := range []string{"Final", "Draft", "Draft [revised from Final 2026-08-20 but not the grammar]"} {
			facts := run(t, status, func(ev, rec string) {
				writeEvidence(t, ev, slug, "grounding/findings.md", "Date: 2026-01-01\n")
				gate := filepath.Join(rec, slug, "artifacts", "gate.md")
				if err := os.MkdirAll(filepath.Dir(gate), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(gate, []byte("- **Date**: 2026-01-01\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			})
			if v, _ := factValue(facts, "lens_stale"); v != "none" {
				t.Errorf("%s: lens_stale = %q, want none", status, v)
			}
			if v, _ := factValue(facts, "gate_stale"); v != "false" {
				t.Errorf("%s: gate_stale = %q, want false", status, v)
			}
		}
	})
	t.Run("gate.md is dated by its own header", func(t *testing.T) {
		facts := run(t, reentered, func(_, rec string) {
			gate := filepath.Join(rec, slug, "artifacts", "gate.md")
			if err := os.MkdirAll(filepath.Dir(gate), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(gate, []byte("# Gate\n\n- **Date**: 2026-08-19\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		})
		if v, _ := factValue(facts, "gate_stale"); v != "true" {
			t.Fatalf("gate_stale = %q, want true", v)
		}
		facts = run(t, reentered, func(_, _ string) {})
		if v, _ := factValue(facts, "gate_stale"); v != "false" {
			t.Fatalf("no gate.md: gate_stale = %q, want false (unwritten is not old)", v)
		}
	})
	t.Run("an unbound evidence root leaves both absent", func(t *testing.T) {
		_, records := newEvidenceTree(t, slug, nil, nil)
		if err := os.MkdirAll(records, 0o755); err != nil {
			t.Fatal(err)
		}
		doc := recordAt(t, records, slug, reentered)
		e := &FactEnv{Doc: doc, Slug: slug, Roots: map[string]string{},
			readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
		facts := tbl.Evaluate(e)
		for _, name := range []string{"lens_stale", "gate_stale"} {
			if _, ok := factValue(facts, name); ok {
				t.Errorf("%s answered with no root bound", name)
			}
		}
	})
}

// --- gate facts -------------------------------------------------------------

func factMembers(facts []Fact, name string) ([]string, bool) {
	for _, f := range facts {
		if f.Name == name {
			return f.Members, true
		}
	}
	return nil, false
}

// TestGateFactsNameTheAssumptions: the id sets carry exactly the members
// the tallies count, and the re-verify set is ABSENT off a re-entry
// rather than empty — an empty set would read "nothing to re-verify" on
// a record that was never revised.
func TestGateFactsNameTheAssumptions(t *testing.T) {
	recs, _, _ := statusFixture(t)
	tbl := loadRealTable(t)
	facts := func(rec string) []Fact {
		t.Helper()
		doc, err := scan.File(filepath.Join(recs, rec), scan.Options{})
		if err != nil {
			t.Fatal(err)
		}
		return tbl.Evaluate(NewFactEnv(tbl, doc, strings.TrimSuffix(rec, ".md")))
	}

	all := facts("0020-cache-eviction-policy.md")
	if got, _ := factMembers(all, "ca_pending_ids"); strings.Join(got, ",") != "0020:A1,0020:A2" {
		t.Errorf("ca_pending_ids = %v, want both Pending assumptions", got)
	}
	if n, _ := factValue(all, "ca_pending"); n != "2" {
		t.Errorf("ca_pending = %q; the ids and the count must agree", n)
	}
	if _, ok := factMembers(all, "reverify"); ok {
		t.Error("reverify answered on a bare Draft; no re-entry means no set, not an empty one")
	}
	if got, _ := factMembers(all, "ca_off_vocabulary_ids"); len(got) != 0 {
		t.Errorf("ca_off_vocabulary_ids = %v on a clean list", got)
	}

	re := facts("0022-cache-metrics-surface.md")
	if got, ok := factMembers(re, "reverify"); !ok || strings.Join(got, ",") != "0022:A1,0022:A2" {
		t.Errorf("reverify = %v (%v); want the qualifier's re-verify list as ids", got, ok)
	}
	if got, _ := factMembers(re, "ca_pending_ids"); len(got) != 0 {
		t.Errorf("ca_pending_ids = %v on a record whose assumptions are Refuted/Verified", got)
	}
}

// TestEdgeTalliesAreThreeValued: with nothing looked for every anchor is
// unlooked and the peer-evidence count is absent; once the caller's hook
// decides the edges, the same facts report what was found. The hook is
// asked once, and only when a tally is evaluated.
func TestEdgeTalliesAreThreeValued(t *testing.T) {
	tbl := loadRealTable(t)
	doc, err := scan.File(fixturePath("current-shape.md"), scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	anchors, peers := 0, 0
	for _, e := range doc.Edges {
		switch e.Kind {
		case edge.SourceAnchor:
			anchors++
		case edge.PeerEvidence:
			peers++
		}
	}
	if anchors == 0 || peers == 0 {
		t.Fatalf("fixture carries %d anchors and %d peer citations; the test needs both", anchors, peers)
	}

	// Unfiltered, the tallies are not evaluated at all: they are declared
	// on demand, because deciding edges is the one costly fact and the
	// navigator renders --tags several times a stage.
	if quiet := tbl.Evaluate(NewFactEnv(tbl, doc, "current-shape")); len(quiet) > 0 {
		for _, name := range []string{"anchors_total", "anchors_unresolved", "anchors_unlooked", "peer_evidence_unresolved"} {
			if _, ok := factValue(quiet, name); ok {
				t.Errorf("%s was evaluated by an unfiltered call; it is on demand", name)
			}
		}
	}
	tallies := map[string]bool{"anchors_total": true, "anchors_unresolved": true, "anchors_unlooked": true, "peer_evidence_unresolved": true}
	quietEnv := NewFactEnv(tbl, doc, "current-shape")
	quietEnv.Want = tallies
	unlooked := tbl.Evaluate(quietEnv)
	if v, _ := factValue(unlooked, "anchors_total"); v != strconv.Itoa(anchors) {
		t.Errorf("anchors_total = %q, want %d", v, anchors)
	}
	if v, _ := factValue(unlooked, "anchors_unlooked"); v != strconv.Itoa(anchors) {
		t.Errorf("anchors_unlooked = %q with no hook, want every anchor", v)
	}
	if v, _ := factValue(unlooked, "anchors_unresolved"); v != "0" {
		t.Errorf("anchors_unresolved = %q with nothing looked", v)
	}
	if _, ok := factValue(unlooked, "peer_evidence_unresolved"); ok {
		t.Error("peer_evidence_unresolved answered while the peer edges are undecided")
	}

	calls := 0
	env := NewFactEnv(tbl, doc, "current-shape")
	env.ResolveEdges = func() {
		calls++
		no := false
		for i := range doc.Edges {
			doc.Edges[i].Resolved = &no
		}
	}
	env.Want = map[string]bool{"status": true}
	if tbl.Evaluate(env); calls != 0 {
		t.Errorf("a call filtered to the cheap facts resolved edges %d times", calls)
	}
	env.Want = tallies
	looked := tbl.Evaluate(env)
	if calls != 1 {
		t.Errorf("the hook ran %d times for four tallies, want once", calls)
	}
	if v, _ := factValue(looked, "anchors_unresolved"); v != strconv.Itoa(anchors) {
		t.Errorf("anchors_unresolved = %q after every anchor resolved false, want %d", v, anchors)
	}
	if v, _ := factValue(looked, "anchors_unlooked"); v != "0" {
		t.Errorf("anchors_unlooked = %q after resolution", v)
	}
	if v, ok := factValue(looked, "peer_evidence_unresolved"); !ok || v != strconv.Itoa(peers) {
		t.Errorf("peer_evidence_unresolved = %q (%v), want %d", v, ok, peers)
	}
}

// TestAlwaysOnSetFactsNeedNoDeclaration: the skills pass the whole `rdr
// status --tags NNNN` vector to intrastate in one substitution, and a set
// arrives as a JSON array literal.
//
// This test used to assert the OPPOSITE — that every always-on set fact
// must be declared `[tags.<name>] kind = "set"` in every routing model
// that receives the vector — because an undeclared key was validated
// against the zero declaration, whose kind is not `set`, so one such fact
// refused EVERY call over the corpus (exit 2, "not set-valued"). That was
// intrastate#3exy, and it cost a corpus-wide refusal once.
//
// intrastate RDR 0020 admits an undeclared key as a pure carrier, so the
// declarations came out. The invariant is inverted and still worth
// pinning: a set fact a model does not guard on must NOT need declaring,
// because the moment it does, adding a fact to rdr-facts.toml silently
// breaks every routing call until someone edits each model. Asserting the
// absence keeps the regression visible — a re-introduced declaration is
// dead weight, and a re-introduced REFUSAL is 3exy returning.
//
// A guarded set is a different thing and stays declared: it is a
// dimension, and the coverage proof needs it.
func TestAlwaysOnSetFactsNeedNoDeclaration(t *testing.T) {
	tbl := loadRealTable(t)
	for _, model := range routingModelNames {
		if callerTags[model] != nil {
			continue
		}
		path := repoFile(t, filepath.Join("models", model))
		tags := modelTags(t, path)
		guards := modelGuardKeys(t, path)
		for _, f := range tbl.Facts {
			if f.Kind != "set" || f.OnDemand || f.Prose {
				continue
			}
			if tags[f.Name] && !guards[f.Name] {
				t.Errorf("%s: set fact %q is declared but no row guards it; since intrastate RDR 0020 an undeclared key crosses as a pure carrier, so the declaration is dead weight — delete it", model, f.Name)
			}
		}
	}
}

// --- the accretion floor's facts -------------------------------------------

// seamRecord writes a record whose Seam Lineage field is `field` (or none
// when empty) and returns its facts.
func seamRecord(t *testing.T, field string) []Fact {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "0031-frame-seam.md")
	doc := "# Recommendation 0031: Frame seam\n\n## Metadata\n\n- **Date**: 2026-08-30\n- **Status**: Draft\n- **Profile**: mid\n"
	if field != "" {
		doc += "- **Seam Lineage**: " + field + "\n"
	}
	doc += "\n## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := scan.File(path, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	tbl := loadRealTable(t)
	return tbl.Evaluate(NewFactEnv(tbl, d, "0031-frame-seam"))
}

// TestSeamLineageFactsBucketTheCount: the count is published raw, the
// bucket is what the `floor` rows read (0, 1, 2+), and a field whose
// count the grammar cannot read is `unread` — a value, not an absence,
// because the field IS written and the floor cannot be resolved off it.
func TestSeamLineageFactsBucketTheCount(t *testing.T) {
	for _, c := range []struct {
		name, field, count, bucket, disp string
	}{
		{"none declared", "`area:x` — no prior accretion.", "0", "0", "false"},
		{"one", "`a::B` — 1st point-fix; trail: k1.", "1", "1", "false"},
		{"two", "`a::B` — 2nd point-fix; trail: k1 + k2.", "2", "2+", "false"},
		{"many, escaped", "`a::B` — 5 prior point-fixes; trail: k1. Accretion disposition: five siting fixes; cite: 0004.", "5", "2+", "true"},
		{"unread", "`a::B` — Nth point-fix (N≥3); trail: k1.", "", "unread", "false"},
	} {
		t.Run(c.name, func(t *testing.T) {
			facts := seamRecord(t, c.field)
			if got, ok := factValue(facts, "seam_lineage_count"); got != c.count || ok != (c.count != "") {
				t.Errorf("seam_lineage_count = %q (present %v), want %q", got, ok, c.count)
			}
			if got, _ := factValue(facts, "seam_lineage"); got != c.bucket {
				t.Errorf("seam_lineage = %q, want %q", got, c.bucket)
			}
			if got, _ := factValue(facts, "accretion_disposition"); got != c.disp {
				t.Errorf("accretion_disposition = %q, want %q", got, c.disp)
			}
		})
	}
}

// TestSeamLineageFactsAreAbsentWithoutTheField: no field, and a seed's
// placeholder, evaluate to NOTHING — `--json` omits all three and `--tags`
// renders the declared sentinels (`none`, `false`), which is the half
// TestRoutingSentinelsRenderOnlyAsTags holds for every sentinel fact.
func TestSeamLineageFactsAreAbsentWithoutTheField(t *testing.T) {
	for _, c := range []struct{ name, field string }{
		{"no field", ""},
		{"placeholder", "[seed placeholder — populate at propose]"},
	} {
		t.Run(c.name, func(t *testing.T) {
			facts := seamRecord(t, c.field)
			for _, name := range []string{"seam_lineage_count", "seam_lineage", "accretion_disposition"} {
				if v, ok := factValue(facts, name); ok {
					t.Errorf("%s = %q; with no readable field the fact must be absent, not defaulted", name, v)
				}
			}
		})
	}
}

// --- the Determinacy line ---------------------------------------------------

// contractsRecord writes a mid record whose Normative Contracts section
// holds `body` (no section at all when empty) and returns its facts.
func contractsRecord(t *testing.T, body string) []Fact {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "0032-frame-codec.md")
	doc := "# Recommendation 0032: Frame codec\n\n## Metadata\n\n- **Date**: 2026-09-01\n- **Status**: Draft\n- **Profile**: mid\n\n## Problem Statement\n\nSynthetic.\n"
	if body != "" {
		doc += "\n## Normative Contracts\n\n" + body + "\n"
	}
	doc += "\n## Decision Rationale\n\nSynthetic.\n"
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := scan.File(path, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	tbl := loadRealTable(t)
	return tbl.Evaluate(NewFactEnv(tbl, d, "0032-frame-codec"))
}

// TestDeterminacyReadsTheWrittenLine: the Stage-5 judgement is a reading
// no fact makes, so the fact reads the LINE the reading leaves behind —
// its leading word, with `n/a` folded to `na` and emphasis dropped. No
// line, a word off the domain, and a line inside template guidance all
// read ABSENT (rendered `unjudged` under --tags), never `na`: silence is
// not a disposition, and the table stops on it by name.
func TestDeterminacyReadsTheWrittenLine(t *testing.T) {
	fence := "```normative\nfunc Encode(f Frame) []byte\n```\n\n"
	for _, c := range []struct{ name, body, want string }{
		{"fired", fence + "Determinacy: fired — C1 (hashing), C2 (step order)", "fired"},
		{"n/a folds to na", fence + "Determinacy: n/a — a flag surface; nothing algorithmic is locked.", "na"},
		{"case and a trailing stop", fence + "Determinacy: N/A.", "na"},
		{"bold label", fence + "**Determinacy:** fired — C1.", "fired"},
		{"bold label, colon outside", fence + "**Determinacy**: fired — C1.", "fired"},
		{"no line", fence, ""},
		{"no section", "", ""},
		{"off-domain word", fence + "Determinacy: maybe — decide later.", ""},
		{"template guidance is not a line", "[Required — never omit.\nDeterminacy: fired — <contracts> or\nDeterminacy: n/a — <reason>.]", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			facts := contractsRecord(t, c.body)
			got, ok := factValue(facts, "determinacy")
			if got != c.want || ok != (c.want != "") {
				t.Errorf("determinacy = %q (present %v), want %q", got, ok, c.want)
			}
			if c.want != "" {
				return
			}
			tbl := loadRealTable(t)
			if v, _ := factValue(withAbsentSentinels(tbl, facts, nil), "determinacy"); v != "unjudged" {
				t.Errorf("--tags renders determinacy=%q for an unwritten line, want the `unjudged` sentinel", v)
			}
		})
	}
}

// implArtifactEnv writes the named files under a synthetic slug directory
// and binds a FactEnv at it, the same shape TestCapsuleStateReadsTheHeader
// uses for the Stage-9 capsule: a temp records root, no evidence root, no
// seam var reached.
func implArtifactEnv(t *testing.T, tbl *FactTable, files map[string]string) *FactEnv {
	t.Helper()
	slug := "0099-synthetic-artifact-record"
	base := t.TempDir()
	records := filepath.Join(base, "records")
	dir := filepath.Join(records, slug, "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return testEnv(t, tbl, slug, "", records)
}

// TestImplArtifactFactsReadTheLedger is the Stage-8 launch gate's four
// facts, read from the artifact ledger a launch prompt writes beside the
// capsule — req-list.md, coverage.md, deviations.md — never from the
// record itself.
func TestImplArtifactFactsReadTheLedger(t *testing.T) {
	tbl := loadRealTable(t)
	names := []string{"req_count", "impl_orphans", "impl_open_decisions", "impl_mvv_recorded",
		"impl_verification_recorded", "impl_findings_open"}

	t.Run("happy path", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"req-list.md": "- **[REQ-1]** \"first.\"\n" +
				"- **[REQ-2]** \"second.\"\n" +
				"- **[REQ-3]** \"third.\"\n" +
				"- [REQ-MVV] \"acceptance.\"\n",
			"coverage.md": "| Requirement | Test |\n" +
				"| --- | --- |\n" +
				"| `REQ-1` | `TestOne` |\n" +
				"| `REQ-2` | `TestTwo` |\n" +
				"| `REQ-3` | `TestThree` |\n" +
				"\n## REQ-MVV output (recorded)\n\nA caller saw it work.\n",
			"deviations.md": "- **Status: mechanical translation** — a rename, nothing more.\n",
		})
		facts := tbl.Evaluate(e)
		want := map[string]string{
			"req_count": "0-10", "impl_orphans": "0",
			"impl_open_decisions": "0", "impl_mvv_recorded": "true",
		}
		for name, w := range want {
			if got, ok := factValue(facts, name); !ok || got != w {
				t.Errorf("%s = %q (ok %v), want %q", name, got, ok, w)
			}
		}
	})

	t.Run("artifact dir absent", func(t *testing.T) {
		slug := "0099-synthetic-artifact-record"
		base := t.TempDir()
		records := filepath.Join(base, "records")
		if err := os.MkdirAll(records, 0o755); err != nil {
			t.Fatal(err)
		}
		e := testEnv(t, tbl, slug, "", records)
		facts := tbl.Evaluate(e)
		for _, name := range names {
			if v, ok := factValue(facts, name); ok {
				t.Errorf("%s = %q; with no artifact dir the fact must be absent", name, v)
			}
		}
	})

	t.Run("req-list absent, coverage present", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"coverage.md": "| Requirement | Test |\n| --- | --- |\n" +
				"| `REQ-9` | `TestNine` |\n\n## REQ-MVV output (recorded)\n\nSeen.\n",
		})
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "req_count"); ok {
			t.Errorf("req_count = %q; no req-list.md means absent", v)
		}
		if v, ok := factValue(facts, "impl_orphans"); ok {
			t.Errorf("impl_orphans = %q; orphans needs both files", v)
		}
		if got, _ := factValue(facts, "impl_mvv_recorded"); got != "true" {
			t.Errorf("impl_mvv_recorded = %q, want true — it reads coverage.md alone", got)
		}
	})

	t.Run("orphan in req-list only", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"req-list.md": "- **[REQ-3]** \"uncovered.\"\n",
			"coverage.md": "| Requirement | Test |\n| --- | --- |\n" +
				"| `REQ-3` | |\n",
		})
		facts := tbl.Evaluate(e)
		if got, _ := factValue(facts, "impl_orphans"); got != "1+" {
			t.Errorf("impl_orphans = %q, want 1+ — a listed REQ with an empty test cell is an orphan", got)
		}
	})

	t.Run("orphan in coverage only", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"req-list.md": "- **[REQ-1]** \"covered.\"\n",
			"coverage.md": "| Requirement | Test |\n| --- | --- |\n" +
				"| `REQ-1` | `TestOne` |\n" +
				"| `REQ-9` | `TestNine` |\n",
		})
		facts := tbl.Evaluate(e)
		if got, _ := factValue(facts, "impl_orphans"); got != "1+" {
			t.Errorf("impl_orphans = %q, want 1+ — REQ-9 is in coverage.md but not req-list.md", got)
		}
	})

	t.Run("hyphenated ids read the same in both files", func(t *testing.T) {
		// `REQ-OVR-1` used to match nothing in req-list.md (anchored on `]`)
		// and truncate to `REQ-OVR` in coverage.md (prefix-only): a phantom
		// orphan on every hyphenated id.
		e := implArtifactEnv(t, tbl, map[string]string{
			"req-list.md": "- **[REQ-OVR-1]** \"one.\"\n- **[REQ-OVR-2]** \"two.\"\n- **[REQ-7.a-iv]** \"three.\"\n",
			"coverage.md": "| Requirement | Test |\n| --- | --- |\n" +
				"| `REQ-OVR-1` | `TestOne` |\n" +
				"| `REQ-OVR-2` | `TestTwo` |\n" +
				"| `REQ-7.a-iv` | `TestThree` |\n",
		})
		facts := tbl.Evaluate(e)
		if got, _ := factValue(facts, "impl_orphans"); got != "0" {
			t.Errorf("impl_orphans = %q, want 0 — hyphenated ids must agree across both files", got)
		}
		e = implArtifactEnv(t, tbl, map[string]string{
			"req-list.md": "- **[REQ-OVR-1]** \"one.\"\n- **[REQ-OVR-2]** \"two.\"\n",
			"coverage.md": "| Requirement | Test |\n| --- | --- |\n" +
				"| `REQ-OVR-1` | `TestOne` |\n",
		})
		facts = tbl.Evaluate(e)
		if got, _ := factValue(facts, "impl_orphans"); got != "1+" {
			t.Errorf("impl_orphans = %q, want 1+ — REQ-OVR-2 is listed but uncovered", got)
		}
	})

	t.Run("eleven-plus, MVV and duplicates excluded", func(t *testing.T) {
		var b strings.Builder
		for i := 1; i <= 12; i++ {
			b.WriteString("- **[REQ-" + strconv.Itoa(i) + "]** \"n.\"\n")
		}
		b.WriteString("- [REQ-MVV] \"acceptance.\"\n")
		b.WriteString("- **[REQ-1]** \"duplicate of REQ-1.\"\n")
		e := implArtifactEnv(t, tbl, map[string]string{"req-list.md": b.String()})
		if got, _ := factValue(tbl.Evaluate(e), "req_count"); got != "11+" {
			t.Errorf("req_count = %q, want 11+ — 12 distinct ids, MVV and the dup do not add to the count", got)
		}

		var b2 strings.Builder
		for i := 1; i <= 10; i++ {
			b2.WriteString("- **[REQ-" + strconv.Itoa(i) + "]** \"n.\"\n")
		}
		b2.WriteString("- [REQ-MVV] \"acceptance.\"\n")
		b2.WriteString("- **[REQ-1]** \"duplicate of REQ-1.\"\n")
		e2 := implArtifactEnv(t, tbl, map[string]string{"req-list.md": b2.String()})
		if got, _ := factValue(tbl.Evaluate(e2), "req_count"); got != "0-10" {
			t.Errorf("req_count = %q, want 0-10 — 10 distinct ids, MVV and the dup do not add to the count", got)
		}
	})

	t.Run("open decisions", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "- **Status: needs author decision (recorded, run continued)**\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_open_decisions"); got != "1+" {
			t.Errorf("impl_open_decisions = %q, want 1+ — the leading phrase is still open", got)
		}

		e2 := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "- **Status: needs author decision → RESOLVED (author picked option B).**\n" +
				"- **Status: accepted.**\n",
		})
		if got, _ := factValue(tbl.Evaluate(e2), "impl_open_decisions"); got != "0" {
			t.Errorf("impl_open_decisions = %q, want 0 — a rewritten line closes, and the arrow must not cut it early", got)
		}

		e3 := implArtifactEnv(t, tbl, map[string]string{})
		if v, ok := factValue(tbl.Evaluate(e3), "impl_open_decisions"); ok {
			t.Errorf("impl_open_decisions = %q; with no deviations.md the fact must be absent", v)
		}
	})

	// impact_families reads only the `families:` header line `rdr impact`
	// writes; the sections below it are the Phase 2 leg's, not the fact's.
	// A file without that line was not the projection's output and is
	// unread — the shard route must stop on it, not read it as `0`.
	t.Run("impact families", func(t *testing.T) {
		for _, c := range []struct{ name, body, want string }{
			{"none predicted", "# Impact — 0099\n\nfamilies: 0\nrows: 0\nrecords: none\n", "0"},
			{"predicted", "# Impact — 0099\n\nfamilies: 3\nrows: 7\nrecords: 0021\n\n## TestX (7 tests, 2 files)\n", "1+"},
		} {
			e := implArtifactEnv(t, tbl, map[string]string{"impact.md": c.body})
			if got, _ := factValue(tbl.Evaluate(e), "impact_families"); got != c.want {
				t.Errorf("%s: impact_families = %q, want %q", c.name, got, c.want)
			}
		}
		for _, c := range []struct{ name, body string }{
			{"no header line", "# Impact — 0099\n\n## TestX (1 tests, 1 files)\n"},
			{"unparseable count", "families: many\n"},
		} {
			e := implArtifactEnv(t, tbl, map[string]string{"impact.md": c.body})
			if v, ok := factValue(tbl.Evaluate(e), "impact_families"); ok {
				t.Errorf("%s: impact_families = %q; a file with no readable families: line must be absent", c.name, v)
			}
		}
		e := implArtifactEnv(t, tbl, map[string]string{})
		if v, ok := factValue(tbl.Evaluate(e), "impact_families"); ok {
			t.Errorf("impact_families = %q; with no impact.md the fact must be absent", v)
		}
	})

	t.Run("open decisions, legacy spellings", func(t *testing.T) {
		cases := []struct {
			name string
			line string
			want string
		}{
			{"bold label, colon inside, list item",
				"- **Status:** needs author decision", "1+"},
			{"bold label, colon inside",
				"**Status:** needs author decision", "1+"},
			{"trailing bold, no cut mark",
				"**Status: needs author decision**", "1+"},
			{"trailing bold with parenthesis",
				"**Status: needs author decision (low stakes)**", "1+"},
			{"qualifier words before the cut",
				"Status: needs author decision on resumption ordering (recorded, NOT", "1+"},
			{"rewritten, bold label",
				"**Status:** needs author decision → RESOLVED (option B).", "0"},
			{"rewritten after a parenthesis",
				"Status: needs author decision (low stakes) → RESOLVED (kept).", "0"},
			{"rewritten, plain, wrapped-line indent",
				"  Status: needs author decision → RESOLVED (an empty tier reads 1.0; there is", "0"},
			{"bold non-open status",
				"**Status:** mechanical translation", "0"},
			{"history line, not a Status line",
				"- **Original status:** needs author decision", "0"},
			{"label mentioned mid-prose",
				"recorded here with `Status: needs author decision`, and work continued.", "0"},
			{"label as a longer word (boundary)",
				"Status: needs author decisions", "0"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				e := implArtifactEnv(t, tbl, map[string]string{"deviations.md": c.line + "\n"})
				if got, _ := factValue(tbl.Evaluate(e), "impl_open_decisions"); got != c.want {
					t.Errorf("impl_open_decisions = %q, want %q for line %q", got, c.want, c.line)
				}
			})
		}
	})

	t.Run("open decisions, emphasis-insensitive key spellings", func(t *testing.T) {
		cases := []struct {
			name string
			open string // the open-line spelling
			done string // the same entry, rewritten to RESOLVED
		}{
			{"plain", "Status: needs author decision", "Status: needs author decision → RESOLVED (kept)."},
			{"bold key only", "**Status**: needs author decision", "**Status**: needs author decision → RESOLVED (kept)."},
			{"bold key and colon", "**Status:** needs author decision", "**Status:** needs author decision → RESOLVED (kept)."},
			{"backtick key", "`Status`: needs author decision", "`Status`: needs author decision → RESOLVED (kept)."},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				open := implArtifactEnv(t, tbl, map[string]string{"deviations.md": c.open + "\n"})
				if got, _ := factValue(tbl.Evaluate(open), "impl_open_decisions"); got != "1+" {
					t.Errorf("open: impl_open_decisions = %q, want 1+ for line %q", got, c.open)
				}
				done := implArtifactEnv(t, tbl, map[string]string{"deviations.md": c.done + "\n"})
				if got, _ := factValue(tbl.Evaluate(done), "impl_open_decisions"); got != "0" {
					t.Errorf("resolved: impl_open_decisions = %q, want 0 for line %q", got, c.done)
				}
			})
		}
	})

	t.Run("open decisions, mid-line Status key", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "- **Type**: SPEC-DEFECT. **Status**: needs author decision.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_open_decisions"); got != "midline" {
			t.Errorf("impl_open_decisions = %q, want midline — the Status key sits after the Type field, uncountable", got)
		}
	})

	t.Run("open decisions, whole line bold wraps both fields", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "**Type: X. Status: needs author decision**\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_open_decisions"); got != "midline" {
			t.Errorf("impl_open_decisions = %q, want midline — Status sits after Type even though the whole line is bold", got)
		}
	})

	t.Run("open decisions, midline beats a plain open line", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "Status: needs author decision\n" +
				"**Type**: SPEC-DEFECT. **Status**: needs author decision.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_open_decisions"); got != "midline" {
			t.Errorf("impl_open_decisions = %q, want midline — a mid-line entry makes the whole count untrustworthy", got)
		}
	})

	t.Run("mvv recorded", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{"coverage.md": "## REQ-MVV runner\n\nHow to run it.\n"})
		if got, _ := factValue(tbl.Evaluate(e), "impl_mvv_recorded"); got != "false" {
			t.Errorf("impl_mvv_recorded = %q, want false — a runner heading is not a recording", got)
		}

		e2 := implArtifactEnv(t, tbl, map[string]string{
			"coverage.md": "**REQ-MVV output (recorded after Phase 2):**\n\nIt worked.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e2), "impl_mvv_recorded"); got != "true" {
			t.Errorf("impl_mvv_recorded = %q, want true", got)
		}
	})

	// verification.md is the Phase 3 record. A finding (either id form) or
	// a clean Verdict is a run; headings alone are the template nobody
	// filled in, and no file at all is absent — the sentinel `false` is
	// what the gate stops on.
	t.Run("verification recorded, a heading finding", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"verification.md": "# Verification\n\n### FAIL-1 — the cold path divides by zero\n\nSeen.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_verification_recorded"); got != "true" {
			t.Errorf("impl_verification_recorded = %q, want true", got)
		}
	})

	t.Run("verification recorded, a bullet finding", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"verification.md": "# Verification\n\n- **ADV-1** — a warm tier reads NaN\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_verification_recorded"); got != "true" {
			t.Errorf("impl_verification_recorded = %q, want true", got)
		}
	})

	t.Run("verification recorded, a clean verdict alone", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"verification.md": "# Verification\n\n## Verdict — clean\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_verification_recorded"); got != "true" {
			t.Errorf("impl_verification_recorded = %q, want true — a clean run is a run", got)
		}
	})

	t.Run("verification stub is not a run", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"verification.md": "# Verification\n\n## Phase 3a — CoVe\n\n## Phase 3b — Adversarial\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_verification_recorded"); got != "false" {
			t.Errorf("impl_verification_recorded = %q, want false — headings with neither finding nor verdict are a stub", got)
		}
	})

	t.Run("verification absent", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{"coverage.md": "## REQ-MVV output\n\nSeen.\n"})
		if v, ok := factValue(tbl.Evaluate(e), "impl_verification_recorded"); ok {
			t.Errorf("impl_verification_recorded = %q; no verification.md means absent", v)
		}
	})

	// triage.md is the Phase 4 ledger: an IN-SCOPE row (by label prefix,
	// qualifiers and all) counts as open work until its Outcome cell
	// settles to fixed:<sha> or held:<reason>; anything else — a bare
	// `open`, an empty cell, or a shipped-as-kata `kata ab12` — is still
	// open, because those are not the ledger's spelling of "done".
	t.Run("findings open, one open in-scope row", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "| # | Source | Verdict | Outcome |\n" +
				"| --- | --- | --- | --- |\n" +
				"| 1 | cove | IN-SCOPE | open |\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "1+" {
			t.Errorf("impl_findings_open = %q, want 1+", got)
		}
	})

	t.Run("findings open, shipped as kata is still open", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "| # | Source | Verdict | Outcome |\n" +
				"| --- | --- | --- | --- |\n" +
				"| 1 | cove | IN-SCOPE | kata ab12 |\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "1+" {
			t.Errorf("impl_findings_open = %q, want 1+ — a shipped-as-kata in-scope defect is still open", got)
		}
	})

	t.Run("findings closed, fixed and held", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "| # | Source | Verdict | Outcome |\n" +
				"| --- | --- | --- | --- |\n" +
				"| 1 | cove | IN-SCOPE | fixed:cb26fb39 |\n" +
				"| 2 | grounding | IN-SCOPE | held:contract (C2-4) |\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "0" {
			t.Errorf("impl_findings_open = %q, want 0", got)
		}
	})

	t.Run("findings open, a qualified verdict still counts", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "| # | Source | Verdict | Outcome |\n" +
				"| --- | --- | --- | --- |\n" +
				"| 1 | cove | IN-SCOPE (C2-4: mixed spelling) | open |\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "1+" {
			t.Errorf("impl_findings_open = %q, want 1+ — a qualified verdict still starts with the label", got)
		}
	})

	t.Run("findings closed, a legacy verdict table", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "| # | Source | Verdict | Outcome |\n" +
				"| --- | --- | --- | --- |\n" +
				"| 1 | cove | FIX-NOW | fixed:0a1b2c3 |\n" +
				"| 2 | grounding | KATA-BUG | held:tracked elsewhere |\n" +
				"| 3 | 3amigo | RDR-SEED | fixed:1234567 |\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "0" {
			t.Errorf("impl_findings_open = %q, want 0 — none of these verdicts is IN-SCOPE", got)
		}
	})

	t.Run("findings closed, prose with no table", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "# Triage\n\nNothing was found worth a table.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "0" {
			t.Errorf("impl_findings_open = %q, want 0 — a ledger with no verdict table has nothing open", got)
		}
	})

	t.Run("findings open, columns in a different order", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{
			"triage.md": "| # | Outcome | Source | Verdict |\n" +
				"| --- | --- | --- | --- |\n" +
				"| 1 | open | cove | IN-SCOPE |\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_findings_open"); got != "1+" {
			t.Errorf("impl_findings_open = %q, want 1+ — the header binds the columns, not their position", got)
		}
	})

	t.Run("findings absent, no triage.md", func(t *testing.T) {
		e := implArtifactEnv(t, tbl, map[string]string{"coverage.md": "## REQ-MVV output\n\nSeen.\n"})
		if v, ok := factValue(tbl.Evaluate(e), "impl_findings_open"); ok {
			t.Errorf("impl_findings_open = %q; no triage.md means absent", v)
		}
	})

	t.Run("--tags renders the sentinels", func(t *testing.T) {
		slug := "0099-synthetic-artifact-record"
		base := t.TempDir()
		records := filepath.Join(base, "records")
		if err := os.MkdirAll(records, 0o755); err != nil {
			t.Fatal(err)
		}
		e := testEnv(t, tbl, slug, "", records)
		facts := tbl.Evaluate(e)
		want := map[string]bool{"req_count": true, "impl_orphans": true,
			"impl_open_decisions": true, "impl_mvv_recorded": true,
			"impl_verification_recorded": true, "impl_findings_open": true}
		tagged := withAbsentSentinels(tbl, facts, want)
		got := map[string]string{}
		for _, f := range tagged {
			got[f.Name] = f.Value
		}
		wantVals := map[string]string{
			"req_count": "none", "impl_orphans": "none",
			"impl_open_decisions": "none", "impl_mvv_recorded": "false",
			"impl_verification_recorded": "false", "impl_findings_open": "none",
		}
		for name, w := range wantVals {
			if got[name] != w {
				t.Errorf("--tags %s = %q, want sentinel %q", name, got[name], w)
			}
		}
	})
}

// deviationTaxonomy is the taxonomy string rdr-facts.toml declares for
// impl_deviation_types_unknown — kept as one constant here so a taxonomy
// edit only breaks this file in one place.
const deviationTaxonomy = "SPEC-DEFECT SPEC-UNDER DEPENDENCY-LIMIT TEST-FIXTURE IMPL-DECISION IMPL-GAP"

// TestUnknownDeviationTypes covers unknownDeviationTypes directly (the
// token extraction) and through the impl_deviation_types_unknown fact
// itself (the "0" / "1+" / "none" rollup), mirroring how open-decisions is
// tested in TestImplArtifactFactsReadTheLedger.
func TestUnknownDeviationTypes(t *testing.T) {
	t.Run("known types, list and bold-label forms", func(t *testing.T) {
		raw := []byte("- **Type**: SPEC-UNDER — the RDR under-specified the retry budget.\n" +
			"**Type**: IMPL-GAP.\n")
		if got := unknownDeviationTypes(raw, deviationTaxonomy); len(got) != 0 {
			t.Errorf("unknownDeviationTypes = %v, want none — both types are in the taxonomy", got)
		}
	})

	t.Run("slash-joined value, both pieces known", func(t *testing.T) {
		raw := []byte("**Type**: SPEC-DEFECT (candidate) / SPEC-UNDER.\n")
		if got := unknownDeviationTypes(raw, deviationTaxonomy); len(got) != 0 {
			t.Errorf("unknownDeviationTypes = %v, want none — SPEC-DEFECT and SPEC-UNDER are both known", got)
		}
	})

	t.Run("unknown types, first-seen order, deduped", func(t *testing.T) {
		raw := []byte("**Type**: PRECHECK-SCAFFOLDING.\n" +
			"- **Type**: SHIP-ORDER — shipped ahead of its dependency.\n" +
			"**Type**: PRECHECK-SCAFFOLDING.\n")
		got := unknownDeviationTypes(raw, deviationTaxonomy)
		want := []string{"PRECHECK-SCAFFOLDING", "SHIP-ORDER"}
		if len(got) != len(want) {
			t.Fatalf("unknownDeviationTypes = %v, want %v", got, want)
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("unknownDeviationTypes[%d] = %q, want %q (first-seen order, each once)", i, got[i], w)
			}
		}
	})

	t.Run("history and status lines are not Type lines", func(t *testing.T) {
		raw := []byte("Original type: FOO-BAR\n" +
			"- **Status**: needs author decision\n")
		if got := unknownDeviationTypes(raw, deviationTaxonomy); len(got) != 0 {
			t.Errorf("unknownDeviationTypes = %v, want none — neither line opens with a Type key", got)
		}
	})

	t.Run("unclassified placeholder has no taxonomy-shaped token", func(t *testing.T) {
		raw := []byte("**Type**: (to be classified)\n")
		if got := unknownDeviationTypes(raw, deviationTaxonomy); len(got) != 0 {
			t.Errorf("unknownDeviationTypes = %v, want none — a placeholder is skipped, not an unknown", got)
		}
	})

	t.Run("a parenthetical quoting REQ ids is dropped before the / split", func(t *testing.T) {
		raw := []byte("**Type**: SPEC-DEFECT (implementation gap against REQ-43/REQ-44, fixed at implement)\n")
		if got := unknownDeviationTypes(raw, deviationTaxonomy); len(got) != 0 {
			t.Errorf("unknownDeviationTypes = %v, want none — REQ-43/REQ-44 sit inside the parenthetical, not a type", got)
		}
	})

	t.Run("a real unknown type survives a REQ-quoting parenthetical", func(t *testing.T) {
		raw := []byte("**Type**: FOO-BAR (see REQ-1/REQ-2)\n")
		got := unknownDeviationTypes(raw, deviationTaxonomy)
		want := []string{"FOO-BAR"}
		if len(got) != 1 || got[0] != want[0] {
			t.Errorf("unknownDeviationTypes = %v, want %v — FOO-BAR is unknown, the REQ ids are not types", got, want)
		}
	})

	t.Run("through the fact: 0, 1+, and none", func(t *testing.T) {
		tbl := loadRealTable(t)
		e := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "- **Type**: SPEC-UNDER — prose.\n**Type**: IMPL-GAP.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e), "impl_deviation_types_unknown"); got != "0" {
			t.Errorf("impl_deviation_types_unknown = %q, want 0 — only known types present", got)
		}

		e2 := implArtifactEnv(t, tbl, map[string]string{
			"deviations.md": "**Type**: PRECHECK-SCAFFOLDING.\n",
		})
		if got, _ := factValue(tbl.Evaluate(e2), "impl_deviation_types_unknown"); got != "1+" {
			t.Errorf("impl_deviation_types_unknown = %q, want 1+ — PRECHECK-SCAFFOLDING is not in the taxonomy", got)
		}

		e3 := implArtifactEnv(t, tbl, map[string]string{})
		if v, ok := factValue(tbl.Evaluate(e3), "impl_deviation_types_unknown"); ok {
			t.Errorf("impl_deviation_types_unknown = %q; with no deviations.md the fact must be absent", v)
		}
	})
}

// TestArtifactRootNeverDoublesTheFolder: the artifact root already ends
// in `artifacts/`, so every fact under it is spelled from inside that
// folder. A path still reading `artifacts/gate.md` would probe
// `<slug>/artifacts/artifacts/gate.md`, miss, and then the legacy leg
// would find the real file under the flat folder — true for the wrong
// reason, and false the day the legacy leg drops.
func TestArtifactRootNeverDoublesTheFolder(t *testing.T) {
	slug := "0010-frame-header"
	_, records := newEvidenceTree(t, slug, nil, []string{"artifacts/gate.md"})
	tbl := loadRealTable(t)
	env := testEnv(t, tbl, slug, "", records)

	want := filepath.Join(records, slug, "artifacts")
	if got := env.Roots["artifacts"]; got != want {
		t.Fatalf("artifacts root = %q, want %q", got, want)
	}
	if got := env.Legacy["artifacts"]; got != filepath.Join(records, slug) {
		t.Errorf("legacy leg = %q, want the flat record folder", got)
	}
	for _, d := range tbl.Facts {
		if d.Root != "artifacts" && !(d.Source == "capsule-state") {
			continue
		}
		for _, p := range append([]string{d.Path}, d.Paths...) {
			if p == "" {
				continue
			}
			if strings.HasPrefix(p, "artifacts/") {
				t.Errorf("%s: path %q is spelled from outside the artifact root", d.Name, p)
			}
			full, ok := env.under("artifacts", p)
			if !ok || strings.Contains(filepath.ToSlash(full), "artifacts/artifacts") {
				t.Errorf("%s: %q resolves to %q", d.Name, p, full)
			}
		}
	}
	if v, ok := factValue(tbl.Evaluate(env), "gate_written"); !ok || v != "true" {
		t.Errorf("gate_written = %q/%v; the gate sits under the canonical root", v, ok)
	}
}

// TestUnboundRecordsRootLeavesTheArtifactFactsAbsent is the artifact
// half of TestUnboundRootLeavesTheFactAbsent: with no records root bound
// nothing under the artifact root was looked at, and a legacy leg is a
// second place to look, never a reason to answer.
func TestUnboundRecordsRootLeavesTheArtifactFactsAbsent(t *testing.T) {
	slug := "0010-frame-header"
	evidence, _ := newEvidenceTree(t, slug, []string{"grounding/findings.md"}, nil)
	tbl := loadRealTable(t)
	facts := tbl.Evaluate(testEnv(t, tbl, slug, evidence, ""))

	for _, name := range []string{"gate_written", "gate_stale", "impl_capsule", "impl_state",
		"req_count", "impl_orphans", "impl_open_decisions", "impl_mvv_recorded",
		"impl_verification_recorded", "impl_findings_open"} {
		if v, ok := factValue(facts, name); ok {
			t.Errorf("%s = %q with no records root bound; want the fact absent", name, v)
		}
	}
	if v, ok := factValue(facts, "lens_grounding"); !ok || v != "true" {
		t.Errorf("lens_grounding = %q/%v; the evidence root is bound and still decides", v, ok)
	}
}

// TestLinesFactBucketsTheRecord is the size gate's line cap as a fact:
// the record's own length — the number `inspect` prints as `lines` — read
// as `0-400` or `401+`, so a routing row compares a member and nobody
// compares a count in prose. The boundary is pinned on both sides,
// because a cap that reads 400 as over is the off-by-one no golden
// fixture would ever show.
func TestLinesFactBucketsTheRecord(t *testing.T) {
	tbl := loadRealTable(t)
	head := "# Recommendation 0031: Frame Length\n\n## Metadata\n\n- **Status**: Draft\n- **Profile**: small\n\n## Problem Statement\n\n"
	for _, c := range []struct {
		lines int
		want  string
	}{
		{lines: 52, want: "0-400"},
		{lines: 400, want: "0-400"},
		{lines: 401, want: "401+"},
		{lines: 997, want: "401+"},
	} {
		body := head + strings.Repeat("filler\n", c.lines-strings.Count(head, "\n")-1) + "end"
		doc := scan.Bytes([]byte(body), scan.Options{})
		if doc.Lines != c.lines {
			t.Fatalf("fixture of %d lines scanned as %d; the case is not testing the boundary it names", c.lines, doc.Lines)
		}
		facts := tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0031-frame-length", Roots: map[string]string{},
			readFile: os.ReadFile, statPath: os.Stat})
		if got, ok := factValue(facts, "lines"); !ok || got != c.want {
			t.Errorf("%d lines: lines = %q (ok %v), want %q", c.lines, got, ok, c.want)
		}
	}

	// A table declaring the bucket without one of its members is refused
	// at load, exactly as an impl-artifact domain is.
	dir := t.TempDir()
	bad := filepath.Join(dir, "facts.toml")
	if err := os.WriteFile(bad, []byte("[fact.lines]\nkind = \"enum\"\nsource = \"record-lines\"\ndomain = [\"0-400\"]\ndescription = \"half a bucket\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFactTable(bad); err == nil || !strings.Contains(err.Error(), "401+") {
		t.Errorf("a record-lines domain missing 401+ loaded (err %v); the member check is what keeps the row claimable", err)
	}
}

// statusDocs scans the status fixture corpus, as `records` does.
func statusDocs(t *testing.T, dir string) []*scan.Document {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9][0-9]-*.md"))
	var docs []*scan.Document
	for _, p := range paths {
		doc, err := scan.File(p, scan.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if doc.IsRecord() {
			docs = append(docs, doc)
		}
	}
	if len(docs) == 0 {
		t.Fatalf("no records under %s", dir)
	}
	return docs
}

// TestJointCheckHomeFoldsWorstFirst pins the lock's reading of the
// Joint-check lines over the status fixtures: clear, homed (every home
// segment resolved true), open, unhomed (a §-anchor the corpus lacks),
// and absent when no line is written. Resolution is the caller's hook:
// with none, a homed line reads unhomed — false or absent is not a pass
// — and a clear line never asks for it. The fence fact is on demand,
// answers 1+ for the uncited pair and absent with no corpus.
func TestJointCheckHomeFoldsWorstFirst(t *testing.T) {
	tbl := loadRealTable(t)
	recs, _, _ := statusFixture(t)
	docs := statusDocs(t, recs)
	byRec := map[string]*scan.Document{}
	for _, d := range docs {
		byRec[d.Record] = d
	}
	homeOnly := map[string]bool{"joint_check_home": true}
	for rec, want := range map[string]string{"0020": "clear", "0026": "homed", "0033": "open", "0022": "unhomed"} {
		doc := byRec[rec]
		env := NewFactEnv(tbl, doc, rec)
		env.ResolveEdges = func() { scan.NewResolverOver(docs, "", true).ResolveAll([]*scan.Document{doc}) }
		env.Want = homeOnly
		if got, ok := factValue(tbl.Evaluate(env), "joint_check_home"); !ok || got != want {
			t.Errorf("%s: joint_check_home = %q (%v), want %s", rec, got, ok, want)
		}
	}
	env := NewFactEnv(tbl, byRec["0025"], "0025")
	env.Want = homeOnly
	if got, ok := factValue(tbl.Evaluate(env), "joint_check_home"); ok {
		t.Errorf("0025 writes no Joint-check: line but joint_check_home = %q; the fact must be absent", got)
	}

	// Nothing looked: the homed record cannot read homed. A fresh scan,
	// because the loop above resolved the shared corpus doc in place.
	fresh, err := scan.File(byRec["0026"].Path, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	env = NewFactEnv(tbl, fresh, "0026")
	env.Want = homeOnly
	if got, _ := factValue(tbl.Evaluate(env), "joint_check_home"); got != "unhomed" {
		t.Errorf("0026 with no ResolveEdges hook = %q, want unhomed", got)
	}
	// A clear line never pays for resolution.
	calls := 0
	env = NewFactEnv(tbl, byRec["0020"], "0020")
	env.ResolveEdges = func() { calls++ }
	env.Want = homeOnly
	if got, _ := factValue(tbl.Evaluate(env), "joint_check_home"); got != "clear" || calls != 0 {
		t.Errorf("0020 = %q with %d resolutions, want clear and 0", got, calls)
	}

	// overlap_uncited: on demand, 1+ for the uncited pair, absent unbound.
	corpus := func() []*scan.Document { return docs }
	for _, rec := range []string{"0025", "0026"} {
		env = NewFactEnv(tbl, byRec[rec], rec)
		env.Corpus = corpus
		if _, ok := factValue(tbl.Evaluate(env), "overlap_uncited"); ok {
			t.Errorf("%s: overlap_uncited evaluated unfiltered; it is on demand", rec)
		}
		env.Want = map[string]bool{"overlap_uncited": true}
		if got, _ := factValue(tbl.Evaluate(env), "overlap_uncited"); got != "1+" {
			t.Errorf("%s: overlap_uncited = %q, want 1+ (shares key.go::Preimage with its peer, neither cites the other)", rec, got)
		}
	}
	env = NewFactEnv(tbl, byRec["0033"], "0033")
	env.Corpus, env.Want = corpus, map[string]bool{"overlap_uncited": true}
	if got, _ := factValue(tbl.Evaluate(env), "overlap_uncited"); got != "0" {
		t.Errorf("0033: overlap_uncited = %q, want 0 (its fire is cited by the JC target edge)", got)
	}
	env = NewFactEnv(tbl, byRec["0025"], "0025")
	env.Want = map[string]bool{"overlap_uncited": true}
	if got, ok := factValue(tbl.Evaluate(env), "overlap_uncited"); ok {
		t.Errorf("overlap_uncited = %q with no corpus bound; nothing looked must be absent", got)
	}
	// …and `--tags` carries that absence as the declared sentinel, so
	// the fence group receives its dimension rather than refusing.
	t.Setenv("RDR_RECORDS", t.TempDir())
	t.Setenv("RDR_EVIDENCE", "")
	code, out, errb := runCapture(t, "status", "--facts", factTableForTest(t), "--tags", "--filter", "overlap_uncited", byRec["0025"].Path)
	if code != 0 || strings.TrimSpace(out) != "--tag\noverlap_uncited=unchecked" {
		t.Errorf("unbound corpus: exit %d, tags %q (%s), want overlap_uncited=unchecked", code, out, errb)
	}
}

// --- ledger-tally and iter-max ----------------------------------------------

// TestLedgerTallyReadsTheCurrentIteration covers the reader's shape: an
// open vs an absorbed/fixed line, the newest iteration beating the loose
// file, the int and enum renderings, and the two absence cases — a bound
// root with no ledger anywhere is 0 (looked, none), an unbound root is
// absent.
func TestLedgerTallyReadsTheCurrentIteration(t *testing.T) {
	tbl := loadRealTable(t)
	slug := "0040-synthetic-ledger"

	t.Run("rulings.md: an absorbed line is closed, an unmarked one is open", func(t *testing.T) {
		evidence, records := newEvidenceTree(t, slug, nil, nil)
		writeEvidence(t, evidence, slug, "rulings.md",
			"- **R1**: use the shared adapter — absorbed @refine 2026-08-20\n"+
				"- **R2**: keep the legacy path — still open\n")
		e := testEnv(t, tbl, slug, evidence, records)
		e.table, e.readDir = tbl, os.ReadDir
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "rulings_open"); !ok || v != "1+" {
			t.Fatalf("rulings_open = %q (%v), want 1+ (R2 carries no `absorbed @`)", v, ok)
		}
	})

	t.Run("dispositions.md under the newest iter-N beats the loose file", func(t *testing.T) {
		evidence, records := newEvidenceTree(t, slug, nil, nil)
		writeEvidence(t, evidence, slug, "grounding/dispositions.md",
			"| `F1` | fixed | ok |\n| `F2` | needs-tiebreaker | ok |\n")
		writeEvidence(t, evidence, slug, "grounding/iter-3/dispositions.md",
			"| `F3` | needs-tiebreaker | ok |\n")
		e := testEnv(t, tbl, slug, evidence, records)
		e.table, e.readDir = tbl, os.ReadDir
		facts := tbl.Evaluate(e)
		// Only the iter-3 file is read: one entry, one open — the loose
		// file's two rows (one open) must not also be counted.
		if v, ok := factValue(facts, "lens_findings_open"); !ok || v != "1" {
			t.Fatalf("lens_findings_open = %q (%v), want 1 (iter-3 is current, the loose file is superseded)", v, ok)
		}
	})

	t.Run("a bound root with no ledger anywhere answers 0, not absent", func(t *testing.T) {
		evidence, records := newEvidenceTree(t, slug, nil, nil)
		if err := os.MkdirAll(filepath.Join(evidence, slug, "evidence"), 0o755); err != nil {
			t.Fatal(err)
		}
		e := testEnv(t, tbl, slug, evidence, records)
		e.table, e.readDir = tbl, os.ReadDir
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "rulings_open"); !ok || v != "0" {
			t.Errorf("rulings_open = %q (%v), want 0 (looked, no rulings.md)", v, ok)
		}
		if v, ok := factValue(facts, "lens_findings_open"); !ok || v != "0" {
			t.Errorf("lens_findings_open = %q (%v), want 0 (looked, no dispositions.md in any lens dir)", v, ok)
		}
	})

	t.Run("an unbound root is absent", func(t *testing.T) {
		e := &FactEnv{Slug: slug, Roots: map[string]string{}, table: tbl,
			readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "rulings_open"); ok {
			t.Errorf("rulings_open = %q with no evidence root bound, want absent", v)
		}
		if v, ok := factValue(facts, "lens_findings_open"); ok {
			t.Errorf("lens_findings_open = %q with no evidence root bound, want absent", v)
		}
	})
}

// TestIterMaxFindsTheDeepestSegment: loose files only is depth 1 (the
// declared first pass), the highest iter-N segment across the swept dirs
// wins over a shallower one elsewhere, and an unbound root is absent.
func TestIterMaxFindsTheDeepestSegment(t *testing.T) {
	tbl := loadRealTable(t)
	slug := "0041-synthetic-iter-max"

	t.Run("loose files only is depth 1", func(t *testing.T) {
		evidence, records := newEvidenceTree(t, slug, []string{"grounding/findings.md"}, nil)
		e := testEnv(t, tbl, slug, evidence, records)
		e.table, e.readDir = tbl, os.ReadDir
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "iter_depth"); !ok || v != "1" {
			t.Fatalf("iter_depth = %q (%v), want 1", v, ok)
		}
	})

	t.Run("the deepest segment across the swept dirs wins", func(t *testing.T) {
		evidence, records := newEvidenceTree(t, slug, nil, nil)
		writeEvidence(t, evidence, slug, "grounding/iter-2/findings.md", "Model: m\n")
		writeEvidence(t, evidence, slug, "cove/iter-5/findings.md", "Model: m\n")
		writeEvidence(t, evidence, slug, "critique/iter-3/critique.md", "Model: m\n")
		e := testEnv(t, tbl, slug, evidence, records)
		e.table, e.readDir = tbl, os.ReadDir
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "iter_depth"); !ok || v != "5" {
			t.Fatalf("iter_depth = %q (%v), want 5 (cove/iter-5 is the deepest of the three)", v, ok)
		}
	})

	t.Run("an unbound root is absent", func(t *testing.T) {
		e := &FactEnv{Slug: slug, Roots: map[string]string{}, table: tbl,
			readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
		facts := tbl.Evaluate(e)
		if v, ok := factValue(facts, "iter_depth"); ok {
			t.Errorf("iter_depth = %q with no evidence root bound, want absent", v)
		}
	})
}

// TestRelatedRollupSelectsDraftMembers: `select = "draft"` counts an
// asserted cluster member (ClusterOf, mutual `Cluster:`) that is still
// Status Draft; a mutual-mentions candidate is excluded, same as the
// final-unimplemented select.
func TestRelatedRollupSelectsDraftMembers(t *testing.T) {
	_, table := bindStatusFixture(t)
	dir := t.TempDir()
	head := func(num, title, status, cluster string) string {
		return "# Recommendation " + num + ": " + title + "\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status +
			"\n- **Cluster**: " + cluster + "\n\n## Problem Statement\n\nSynthetic.\n"
	}
	for name, body := range map[string]string{
		"0001-a.md": head("0001", "A", "Final", "0002-b, 0003-c"),
		"0002-b.md": head("0002", "B", "Draft", "0001-a"),
		"0003-c.md": head("0003", "C", "Final", "0001-a"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A mutual-mentions candidate: 0004 names 0001 in prose, 0001 does not
	// declare it back, so ClusterOf marks it Candidate and the select must
	// exclude it even though it is Draft.
	if err := os.WriteFile(filepath.Join(dir, "0004-d.md"),
		[]byte("# Recommendation 0004: D\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Draft\n\n"+
			"## Problem Statement\n\nSee 0001 for background.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCapture(t, "status", "--facts", table, "--records", dir, "--filter", "cluster_members_in_flight", "0001")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "cluster_members_in_flight    enum    1+") {
		t.Errorf("0001's cluster has one Draft member (0002) and one Final (0003), want 1+:\n%s", out)
	}
	// ClusterOf is one hop from the seed: 0003 declares Cluster only to
	// 0001, not to 0002, so from 0003's own seat the Draft sibling is
	// simply not a member and the select correctly reads 0.
	code, out, errb = runCapture(t, "status", "--facts", table, "--records", dir, "--filter", "cluster_members_in_flight", "0003")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "cluster_members_in_flight    enum    0") {
		t.Errorf("0003 declares Cluster only to 0001 (Final) — no direct Draft member, want 0:\n%s", out)
	}
}

// TestRelatedRollupAnyCountsTheWalk: `select = "any"` is the ground
// ladder's cluster rung and counts what `index --cluster-of` reports —
// so a record with NO Cluster field reads 1+ when a peer declares it,
// where `clustered` reads false, and a mutual-mentions candidate counts
// too. The gate once guarded on `clustered` and skipped the rung on a
// record the walk gave four members.
func TestRelatedRollupAnyCountsTheWalk(t *testing.T) {
	_, table := bindStatusFixture(t)
	dir := t.TempDir()
	head := func(num, title, status, extra, body string) string {
		return "# Recommendation " + num + ": " + title + "\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status +
			extra + "\n\n## Problem Statement\n\n" + body + "\n"
	}
	for name, body := range map[string]string{
		"0001-a.md": head("0001", "A", "Final", "", "Synthetic."),
		"0002-b.md": head("0002", "B", "Draft", "\n- **Cluster**: 0001-a", "Synthetic."),
		"0003-c.md": head("0003", "C", "Draft", "", "See RDR 0004 for background."),
		"0004-d.md": head("0004", "D", "Draft", "", "See RDR 0003 for background."),
		"0005-e.md": head("0005", "E", "Final", "", "Alone."),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for rec, want := range map[string]string{
		"0001": "peers=1+", // declared only by 0002; clustered is false here
		"0003": "peers=1+", // mutual-mentions candidate with 0004
		"0005": "peers=0",
	} {
		code, out, errb := runCapture(t, "status", "--tags", "--facts", table, "--records", dir, "--filter", "peers,clustered", rec)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", rec, code, errb)
		}
		if !strings.Contains(out, want+"\n") || !strings.Contains(out, "clustered=false") {
			t.Errorf("%s: want %q and clustered=false:\n%s", rec, want, out)
		}
	}
}

// TestRoutedBackProjectsReentryFacts is the seam the whole fix turns on:
// a Draft sent backward must project facts the navigator's `reentry`
// group already routes on, with no new rows. `status_form` carries the
// new grammar's name, `reentry_target` carries the `@<stage>` the
// demotion form writes into the same slot, and `status_reentry` stays
// FALSE — that bool is declared `form == revised-from`, a narrower
// question than "is this a re-entry", and the routing reads the form.
func TestRoutedBackProjectsReentryFacts(t *testing.T) {
	tbl := loadRealTable(t)
	eval := func(status string) []Fact {
		doc := scan.Bytes([]byte("# Recommendation 0018: Frame Guard\n\n## Metadata\n\n- **Date**: 2026-01-01\n- **Status**: "+
			status+"\n\n## Problem Statement\n\nSynthetic.\n"), scan.Options{})
		return tbl.Evaluate(&FactEnv{Doc: doc, Slug: "0018-frame-guard",
			Roots: map[string]string{}, readFile: os.ReadFile, statPath: os.Stat})
	}

	for status, want := range map[string]string{
		"Draft [routed back from resolve 2026-09-11; re-verify A5,A6 @propose — the approach is refuted]": "propose",
		"Draft [routed back from finalize 2026-09-11; re-verify none @prelock — determinacy never ran]":   "prelock",
		"Draft [routed back from cluster-reconcile 2026-09-11; @reconcile — a spike is open]":             "reconcile",
	} {
		facts := eval(status)
		if got, ok := factValue(facts, "status"); !ok || got != "Draft" {
			t.Errorf("%s: status = %q/%v, want Draft", status, got, ok)
		}
		if got, ok := factValue(facts, "status_form"); !ok || got != "routed-back" {
			t.Errorf("%s: status_form = %q/%v, want \"routed-back\"", status, got, ok)
		}
		if got, ok := factValue(facts, "reentry_target"); !ok || got != want {
			t.Errorf("%s: reentry_target = %q/%v, want %q", status, got, ok, want)
		}
		if got, _ := factValue(facts, "status_reentry"); got != "false" {
			t.Errorf("%s: status_reentry = %q, want false — the bool names the demotion form alone", status, got)
		}
	}

	// A route-back that names no target leaves the slot ABSENT, and
	// `--tags` renders the same `none` sentinel the untargeted demotion
	// renders: the fallback row claims one cell, not two.
	const untargeted = "Draft [routed back from resolve 2026-09-11; re-verify A5 — no target named]"
	facts := eval(untargeted)
	if got, ok := factValue(facts, "reentry_target"); ok {
		t.Errorf("untargeted route-back: reentry_target = %q, want absent", got)
	}
	if got, ok := factValue(withAbsentSentinels(tbl, facts, nil), "reentry_target"); !ok || got != "none" {
		t.Errorf("untargeted route-back: --tags reentry_target = %q/%v, want the none sentinel", got, ok)
	}

	// A malformed origin degrades to a free-text note, exactly as a
	// malformed demotion does — and the record routes as a plain Draft
	// rather than to a stage nobody can name.
	bad := eval("Draft [routed back from propose 2026-09-11; re-verify A5 @refine — propose sends nothing back]")
	if got, ok := factValue(bad, "status_form"); !ok || got != "bracketed" {
		t.Errorf("malformed route-back: status_form = %q/%v, want \"bracketed\"", got, ok)
	}
	if got, ok := factValue(bad, "reentry_target"); ok {
		t.Errorf("malformed route-back: reentry_target = %q, want absent", got)
	}
}
