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
}

// The vocabulary and grammar bridges. They exist so the domain test above
// reads its expectations from the model rather than restating them, which
// is what makes it fail when the model gains a value.

func modelStatusCanonical() []string { return model.StatusVocabulary.Canonical }
func modelStatusObserved() []string  { return model.StatusVocabulary.ObservedAccepted }

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
	},
	"7 Reconcile": {"reconcile", "reconcile_report", "reconcile_report_alt", "reconcile_report_alt2", "ca"},
	"8 Finalize":  {"status", "gate_written"},
	// 8.1 Cluster is the row with no probe, and the table says why: the
	// output is keyed by cluster, not by slug, and the key is not
	// derivable from this record. `cluster` carries what the record
	// DECLARES, which is what the tandem barrier reads; the directory
	// itself stays a search the skill performs.
	"8.1 Cluster": {"cluster"},
	"9 Implement": {"impl_capsule", "impl_state"},
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

	// Backward: no fact is declared that nothing reads.
	for _, f := range tbl.Facts {
		if claimed[f.Name] {
			continue
		}
		if _, ok := routingFacts[f.Name]; ok {
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
