package main

// `rdr status --checklist` — the stage checklist, rendered by the tool.
//
// `skills/rdr-status/SKILL.md` printed this table by hand: a glyph per
// stage row, derived from the fact vector by six prose rules a model
// re-applied on every run. Those rules were the last place the
// navigator's output was mapped rather than read, and a hand-rendered
// checklist is where a demoted Final reads as a fresh Draft. The rules
// are now this file, and the goldens in testdata/status pin them.
//
// THREE VALUES SURVIVE HERE. `--tags` collapses absence into a declared
// sentinel because argv cannot spell it; the checklist is for a reader,
// and a reader must be able to tell "looked, not there" (`–`) from
// "nothing looked" (`?`). So `withAbsentSentinels` is never applied to
// this rendering: a probe whose root is unbound prints `?` and names
// the root, and a field the record does not carry prints `unset`.
//
// The row names and their order are the skill's "What the facts mean"
// table, verbatim — a test binds the two, so a row cannot be renamed or
// reordered in one place without the other going red.

import (
	"fmt"
	"io"
	"strings"
)

// checklistRow is one line of the checklist: the stage as the skill
// names it, the glyph, and the note that says why.
type checklistRow struct {
	Stage string `json:"stage"`
	Glyph string `json:"glyph"`
	Note  string `json:"note"`
}

// Glyphs, as the skill spells them. `–` is an en dash, not a hyphen.
const (
	glyphDone   = "✓"
	glyphNot    = "–"
	glyphJudged = "~"
	glyphAbsent = "?"
)

// checklistStages is the row order, bound to the skill's table by test.
var checklistStages = []string{
	"1 Seed",
	"2 Propose",
	"3 Refine",
	"4 Resolve",
	"5+6 Pre-Lock (review+resolve)",
	"6 Reconcile",
	"7 Finalize",
	"7.1 Cluster",
	"8 Implement",
}

// checklistLenses is the lens row in table order, each with the probe
// that says it FINISHED rather than merely ran, and what is owed when it
// has not. Critique and repeatability finish on a diff; the skill's
// judgement column says a lone critique.md on a foundational record is
// in progress, and the same is true of a lone run-1.
var checklistLenses = []struct {
	name, ran, done, owed string
}{
	{"cove", "lens_cove", "lens_cove_findings", "findings owed"},
	{"grounding", "lens_grounding", "lens_grounding_findings", "findings owed"},
	{"3amigo", "lens_3amigo", "lens_3amigo_consolidation", "consolidation owed"},
	{"critique", "lens_critique", "lens_critique_single", "critique.md owed"},
	{"repeatability", "lens_repeatability", "lens_repeatability_run1", "run-1 owed"},
}

// factView is the fact vector with the table beside it, so a row can ask
// whether an absent fact is absent because nothing looked.
type factView struct {
	byName map[string]Fact
	decl   map[string]FactDecl
	roots  map[string]FactRoot
	bound  map[string]string
}

func newFactView(tbl *FactTable, facts []Fact, bound map[string]string) *factView {
	v := &factView{
		byName: make(map[string]Fact, len(facts)),
		decl:   make(map[string]FactDecl, len(tbl.Facts)),
		roots:  tbl.Roots,
		bound:  bound,
	}
	for _, f := range facts {
		v.byName[f.Name] = f
	}
	for _, d := range tbl.Facts {
		v.decl[d.Name] = d
	}
	return v
}

// value is the fact's rendered value and whether it is present.
func (v *factView) value(name string) (string, bool) {
	f, ok := v.byName[name]
	if !ok {
		return "", false
	}
	return factValueText(f), true
}

// is reports a bool fact as true; absent is false here, so a caller that
// needs the third value asks `unlooked` first.
func (v *factView) is(name string) bool {
	s, ok := v.value(name)
	return ok && s == "true"
}

// unlooked names the unbound root behind an absent probe, or "" when the
// fact is present or hangs under no root. This is the `?` test: a probe
// under a bound root always answers, so absence there is impossible and
// absence under an unbound one means nothing looked.
func (v *factView) unlooked(name string) string {
	if _, ok := v.byName[name]; ok {
		return ""
	}
	d, ok := v.decl[name]
	if !ok || d.Root == "" {
		return ""
	}
	if _, bound := v.bound[d.Root]; bound {
		return ""
	}
	return d.Root
}

// absentNote is the `?` note: the root, and the var that would bind it.
func (v *factView) absentNote(root string) string {
	if r, ok := v.roots[root]; ok && r.Var != "" {
		return fmt.Sprintf("%s unbound (%s)", root, r.Var)
	}
	return root + " unbound"
}

// kv renders `name=value`, or `name unset` for a field the record does
// not carry — never a sentinel, which would be a claim the record makes.
func (v *factView) kv(name string) string {
	if s, ok := v.value(name); ok {
		return name + "=" + s
	}
	return name + " unset"
}

// renderChecklist evaluates the rows over one record's facts. `bound` is
// the FactEnv's resolved roots: a root missing from it is unbound, and
// every probe under it is `?`.
func renderChecklist(tbl *FactTable, facts []Fact, bound map[string]string) []checklistRow {
	v := newFactView(tbl, facts, bound)
	rows := make([]checklistRow, 0, len(checklistStages))
	for _, stage := range checklistStages {
		glyph, note := checklistCell(stage, v)
		rows = append(rows, checklistRow{Stage: stage, Glyph: glyph, Note: note})
	}
	return rows
}

// checklistCell is the glyph rule per row — the six prose rules of the
// skill's former Output step 2, as code.
func checklistCell(stage string, v *factView) (string, string) {
	switch stage {
	case "1 Seed":
		s, ok := v.value("status")
		if !ok {
			return glyphAbsent, "status unset"
		}
		return glyphDone, s

	case "2 Propose":
		var missing []string
		for _, c := range []struct{ fact, label string }{
			{"premortem_line", "Premortem:"}, {"ground_sweep_line", "Ground-sweep:"},
		} {
			if !v.is(c.fact) {
				missing = append(missing, "no "+c.label)
			}
		}
		if len(missing) == 0 {
			return glyphDone, "Premortem: Ground-sweep: · " + v.kv("joint_checks")
		}
		return glyphNot, strings.Join(missing, ", ") + " · " + v.kv("joint_checks")

	case "3 Refine":
		ca, ok := v.value("ca")
		switch {
		case !ok:
			return glyphAbsent, "ca unset"
		case ca == "all-pending":
			return glyphJudged, "open — CAs all Pending"
		case ca == "unknown-plan":
			return glyphAbsent, "unreadable — CA list unfilled or off-vocabulary"
		}
		return glyphJudged, fmt.Sprintf("judged done — %s Verified, %s terminal",
			v.count("ca_verified"), v.count("ca_other_terminal"))

	case "4 Resolve":
		if root := v.unlooked("spikes"); root != "" {
			return glyphAbsent, "spikes unlooked — " + v.absentNote(root)
		}
		if v.is("spikes") {
			return glyphDone, "spikes"
		}
		ca, ok := v.value("ca")
		switch {
		case !ok:
			return glyphAbsent, "ca unset"
		case ca == "all-terminal":
			return glyphJudged, "judged done — verdicts terminal, no spikes"
		case ca == "unknown-plan":
			return glyphAbsent, "unreadable — CA list unfilled or off-vocabulary"
		}
		return glyphNot, fmt.Sprintf("open — %s of %s Pending", v.count("ca_pending"), v.count("ca_total"))

	case "5+6 Pre-Lock (review+resolve)":
		return prelockCell(v)

	case "6 Reconcile":
		if root := v.unlooked("reconcile"); root != "" {
			return glyphAbsent, "reconcile unlooked — " + v.absentNote(root)
		}
		if !v.is("reconcile") {
			return glyphNot, "no reconcile/"
		}
		for _, c := range []struct{ fact, file string }{
			{"reconcile_report", "reconcile.md"}, {"reconcile_report_alt", "report.md"},
			{"reconcile_report_alt2", "reconcile-report.md"},
		} {
			if v.is(c.fact) {
				return glyphDone, "reconcile/" + c.file
			}
		}
		return glyphJudged, "folder, no report"

	case "7 Finalize":
		if root := v.unlooked("gate_written"); root != "" {
			return glyphAbsent, "gate.md unlooked — " + v.absentNote(root)
		}
		if !v.is("gate_written") {
			return glyphNot, "gate.md"
		}
		if v.is("gate_stale") {
			return glyphJudged, "gate.md stale — predates the re-entry, owed again"
		}
		if v.value1("rulings_open") == "1+" {
			return glyphJudged, "rulings open — absorb evidence/rulings.md before the lock"
		}
		return glyphDone, "gate.md"

	case "7.1 Cluster":
		if root := v.unlooked("cluster_reconciled"); root != "" {
			return glyphAbsent, "cluster-reconcile unlooked — " + v.absentNote(root)
		}
		tail := v.kv("clustered") + " · " + v.kv("cluster_reconciled")
		if key, ok := v.value("cluster_key"); ok {
			tail += " · key=" + key
		} else if set, ok := v.value("cluster"); ok {
			tail += " · declared=" + set
		}
		switch {
		case v.is("cluster_reconciled"):
			return glyphDone, tail
		case v.is("clustered"):
			return glyphNot, tail
		}
		return glyphJudged, "not clustered · " + tail

	case "8 Implement":
		if root := v.unlooked("impl_capsule"); root != "" {
			return glyphAbsent, "status.md unlooked — " + v.absentNote(root)
		}
		if !v.is("impl_capsule") {
			return glyphNot, "no capsule"
		}
		note := strings.Join([]string{v.kv("impl_state"), v.kv("req_count"), v.kv("impl_orphans"),
			v.kv("impl_open_decisions"), v.kv("impl_mvv_recorded")}, " · ")
		state, _ := v.value("impl_state")
		switch state {
		case "COMPLETE":
			return glyphDone, note
		case "":
			return glyphAbsent, note
		}
		return glyphJudged, note
	}
	return glyphAbsent, "no rule for this row"
}

// prelockCell is the lens row: the profile, then every lens that RAN, each
// marked finished or owing its completion artifact. Only lenses on disk
// are printed — an off-profile lens is absent from the row, never `–`,
// and the lens OWED is `emit.next` from the model's `lens` outcome, not
// this row's job.
func prelockCell(v *factView) (string, string) {
	if root := v.unlooked("lens_grounding"); root != "" {
		return glyphAbsent, v.kv("profile") + " · lenses unlooked — " + v.absentNote(root)
	}
	parts := []string{v.kv("profile")}
	ran, owing := 0, 0
	for _, l := range checklistLenses {
		if !v.is(l.ran) {
			continue
		}
		ran++
		switch {
		case v.value1("lens_stale") == l.name:
			owing++
			parts = append(parts, glyphJudged+" "+l.name+" (stale — predates the re-entry, owed again)")
		case l.name == "critique" && v.is("lens_critique_single") && !v.is("lens_critique_diff") &&
			v.value1("profile") == "foundational":
			owing++
			parts = append(parts, glyphJudged+" critique (diff owed)")
		case l.name == "repeatability" && v.is("lens_repeatability_run1") && !v.is("lens_repeatability_diff"):
			owing++
			parts = append(parts, glyphJudged+" repeatability (diff owed)")
		case v.is(l.done):
			parts = append(parts, glyphDone+" "+l.name)
		default:
			owing++
			parts = append(parts, glyphJudged+" "+l.name+" ("+l.owed+")")
		}
	}
	// The Determinacy add-on at mid/large: a `fired` line with no
	// repeatability run is a lens still owed; no line is a judgement still
	// owed (the `repeatability` outcome stops there); `na` adds nothing.
	// Foundational carries the full lens on its row, so the line is moot.
	if p := v.value1("profile"); p == "mid" || p == "large" {
		switch d, ok := v.value("determinacy"); {
		case ok && d == "fired" && !v.is("lens_repeatability"):
			owing++
			parts = append(parts, glyphJudged+" repeatability (Determinacy fired, run-1 owed)")
		case !ok:
			owing++
			parts = append(parts, "Determinacy: unjudged")
		}
	}
	if n := v.value1("iter_depth"); n != "" && n != "0" && n != "1" {
		parts = append(parts, "iter-"+n)
	}
	if n := v.value1("lens_findings_open"); n != "" && n != "0" {
		parts = append(parts, "open findings: "+n)
	}
	// The pre-migration file shape holds real lens output the folder
	// probes cannot see; naming it keeps `–` from reading as un-run.
	if v.is("legacy_evidence_shape") {
		parts = append(parts, "legacy evidence shape")
	}
	note := strings.Join(parts, " · ")
	switch {
	case v.is("reconcile") && owing == 0:
		return glyphDone, note
	case ran == 0:
		return glyphNot, note
	}
	return glyphJudged, note
}

// value1 is value without the presence flag, for a comparison.
func (v *factView) value1(name string) string {
	s, _ := v.value(name)
	return s
}

// count renders an int fact, `?` when the record does not carry it.
func (v *factView) count(name string) string {
	if s, ok := v.value(name); ok {
		return s
	}
	return "?"
}

// emitChecklist is the text form: stage, glyph, note, aligned so the
// glyph column reads down.
func emitChecklist(rows []checklistRow, stdout io.Writer) int {
	for _, r := range rows {
		if r.Note == "" {
			fmt.Fprintf(stdout, "%-30s %s\n", r.Stage, r.Glyph)
			continue
		}
		fmt.Fprintf(stdout, "%-30s %s %s\n", r.Stage, r.Glyph, r.Note)
	}
	return 0
}
