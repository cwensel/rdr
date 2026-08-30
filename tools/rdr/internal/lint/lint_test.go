package lint

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// corpus scans the lint fixture dir and resolves every edge in it, which
// is the state Run contracts for: an unresolved edge and an unchecked one
// are different findings, and only a resolved corpus tells them apart.
func corpus(t *testing.T) map[string]*scan.Document {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "lint")
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no lint fixtures in %s: %v", dir, err)
	}
	var docs []*scan.Document
	for _, p := range paths {
		d, err := scan.File(p, scan.Options{})
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, d)
	}
	scan.NewResolver(docs, "").ResolveAll(docs)
	out := map[string]*scan.Document{}
	for _, d := range docs {
		out[d.Record] = d
	}
	return out
}

func report(t *testing.T, record string, opts Options) Report {
	t.Helper()
	d, ok := corpus(t)[record]
	if !ok {
		t.Fatalf("no fixture record %s", record)
	}
	return Run(d, opts)
}

func codes(r Report) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Code)
	}
	return out
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func has(r Report, code string) bool {
	for _, f := range r.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func find(t *testing.T, r Report, code string) Finding {
	t.Helper()
	for _, f := range r.Findings {
		if f.Code == code {
			return f
		}
	}
	t.Fatalf("no %s finding in %v", code, codes(r))
	return Finding{}
}

// TestLabelledRecordIsClean is the acceptance case the rule exists to
// produce: a record that labels its contracts and cites peers by element
// ID has nothing to say about it, at a lock gate or anywhere else.
func TestLabelledRecordIsClean(t *testing.T) {
	r := report(t, "0010", Options{Locking: true})
	if r.Verdict != "PASS" || len(r.Findings) != 0 {
		t.Errorf("labelled record: verdict %s, findings %v", r.Verdict, codes(r))
	}
}

// TestUnlabelledPostRuleRecordBlocks: contracts with no label on a record
// written after the rule landed are a resolution finding, and it blocks
// at a lock gate.
func TestUnlabelledPostRuleRecordBlocks(t *testing.T) {
	r := report(t, "0011", Options{Locking: true})
	f := find(t, r, "label:contracts-required")
	if f.Tier != TierResolution || !f.Blocking {
		t.Errorf("tier %s blocking %v, want resolution+blocking", f.Tier, f.Blocking)
	}
	if r.Verdict != "BLOCK" {
		t.Errorf("verdict %s, want BLOCK", r.Verdict)
	}
	if !strings.Contains(f.Fix, "C1..Cn") {
		t.Errorf("fix does not name the labels to write: %q", f.Fix)
	}
}

// TestAdviceNeverBlocks: the same unlabelled record away from a lock gate
// still reports, and still passes. Advice that stopped a stage would be a
// block wearing another name.
func TestAdviceNeverBlocks(t *testing.T) {
	r := report(t, "0011", Options{})
	if r.Verdict != "PASS" {
		t.Errorf("verdict %s off a lock gate, want PASS", r.Verdict)
	}
	if !has(r, "label:contracts-required") {
		t.Errorf("finding suppressed off the gate: %v", codes(r))
	}
	for _, f := range r.Findings {
		if f.Blocking {
			t.Errorf("%s blocks off a lock gate", f.Code)
		}
	}
}

// TestGrandfatheredRecordGetsAdviceNotABlock: the SAME unlabelled record,
// read as though the rule landed after it was written, drops from the
// blocking tier to the advisory one. Nothing about the file changes —
// only which side of the boundary it sits on.
func TestGrandfatheredRecordGetsAdviceNotABlock(t *testing.T) {
	r := report(t, "0011", Options{Locking: true, Now: "2027-01-01"})
	f := find(t, r, "label:contracts")
	if f.Tier != TierConformance || f.Blocking {
		t.Errorf("tier %s blocking %v, want conformance+advisory", f.Tier, f.Blocking)
	}
	if has(r, "label:contracts-required") {
		t.Error("a grandfathered record still owes labels as a blocking rule")
	}
}

// TestTerminalRecordGetsFixPointerNotABlock is the doctrine this tier
// encodes: a frozen record is never amended, so a dangling reference in
// it is reported as a pointer correction with a line range — and never
// blocks, because the frozen record is not the one locking.
func TestTerminalRecordGetsFixPointerNotABlock(t *testing.T) {
	r := report(t, "0012", Options{Locking: true})
	if !r.Terminal {
		t.Fatalf("fixture 0012 is %s, expected a terminal status", r.Status)
	}
	f := find(t, r, "edge:unresolved-terminal")
	if f.Blocking || r.Verdict != "PASS" {
		t.Errorf("terminal record blocks: finding %v, verdict %s", f.Blocking, r.Verdict)
	}
	if f.LineStart == 0 || f.LineEnd < f.LineStart {
		t.Errorf("fix pointer carries no usable range: %d-%d", f.LineStart, f.LineEnd)
	}
	if !strings.Contains(f.Fix, "pointer only") {
		t.Errorf("fix does not bound the amendment: %q", f.Fix)
	}
}

// TestTerminalRecordGetsNoConformanceAdvice: a record that will never be
// amended must never be told how the template has moved on. That advice
// is unactionable by doctrine, and on 119 of 143 corpus records it would
// be the bulk of the output.
func TestTerminalRecordGetsNoConformanceAdvice(t *testing.T) {
	r := report(t, "0012", Options{})
	for _, f := range r.Findings {
		if f.Tier == TierConformance {
			t.Errorf("terminal record got conformance advice: %s", f.Code)
		}
	}
}

// TestDanglingElementResolvesAsAnEdge, not as a missing-element finding:
// `0010:C9` names an element that does not exist, which is a broken
// reference. Reporting it as "cites no element" would be wrong twice —
// it cites one, and the one it cites is not there.
func TestDanglingElementResolvesAsAnEdge(t *testing.T) {
	r := report(t, "0012", Options{})
	found := false
	for _, fd := range r.Findings {
		if fd.Code == "edge:unresolved-terminal" && strings.Contains(fd.Message, "0010:C9") {
			found = true
		}
	}
	if !found {
		t.Errorf("the dangling element citation is not reported as an edge: %v", codes(r))
	}
	if has(r, "peer-evidence:no-element") {
		t.Error("an element citation was read as a bare record reference")
	}
}

// TestParseWarningsAlwaysSurface: tier 1 is republished on every record,
// terminal included, because on a terminal record a parse warning means
// the scanner is wrong rather than the file.
func TestParseWarningsAlwaysSurface(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "variants", "author-structure.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := scan.Bytes(raw, scan.Options{})
	if len(d.Warnings) == 0 {
		t.Skip("fixture carries no parse warnings")
	}
	r := Run(d, Options{})
	n := 0
	for _, f := range r.Findings {
		if f.Tier == TierParse {
			n++
		}
	}
	if n != len(d.Warnings) {
		t.Errorf("%d parse findings for %d scanner warnings", n, len(d.Warnings))
	}
}

// TestGatePointerSubsectionsAreNotMissing: once the gate moved to a
// pointer, lock replaces
// the Finalization Gate body with a pointer to gate.md. Advising a record
// to restore the five gate subsections would be advising it to undo the
// current process. The one the template retains is a different rule's
// (`gate:cross-cutting-missing`), never a missing-section finding.
func TestGatePointerSubsectionsAreNotMissing(t *testing.T) {
	r := report(t, "0010", Options{})
	for _, f := range r.Findings {
		if f.Code != "template:missing-section" {
			continue
		}
		for _, s := range []string{"Contradiction Check", "Assumption Verification",
			"Scope Verification", "Cross-Cutting Concerns", "Proportionality"} {
			if strings.Contains(f.Message, s) {
				t.Errorf("gate subsection reported missing under a gate pointer: %s", f.Message)
			}
		}
	}
}

// TestFindingsAreOrdered: findings sort by line so a reader walks the
// record top to bottom, and the order is stable across runs.
func TestFindingsAreOrdered(t *testing.T) {
	r := report(t, "0012", Options{})
	for i := 1; i < len(r.Findings); i++ {
		if r.Findings[i-1].LineStart > r.Findings[i].LineStart {
			t.Errorf("findings out of line order at %d: %v", i, codes(r))
		}
	}
}

// TestOwnershipTransferIsByIDAndBacklink is the linking rule's whole
// point, at the moment it earns its keep: when responsibility MOVES, the
// element keeps its ID, the old home is not edited, and the backlink
// query — not a prose sweep — lists what needs review.
//
// 0013 overrides `0010:C1` by ID and cites it as Peer-RDR Evidence; 0014
// is demoted to 0013. The assertions are that both typed edges resolve,
// that they land on the ELEMENT rather than the document, and that the
// overridden contract's backlinks name every citing element.
func TestOwnershipTransferIsByIDAndBacklink(t *testing.T) {
	docs := corpus(t)
	list := make([]*scan.Document, 0, len(docs))
	for _, d := range docs {
		list = append(list, d)
	}

	kindTo := func(record string, kind edge.Kind) []string {
		var out []string
		for _, e := range docs[record].Edges {
			if e.Kind == kind {
				if e.Resolved == nil || !*e.Resolved {
					t.Errorf("%s %s edge to %s did not resolve", record, kind, e.To)
				}
				out = append(out, e.To)
			}
		}
		return out
	}

	// The override reaches the CONTRACT, not just the record: an
	// Overrides field that resolved only to `0010` would say
	// responsibility moved without saying for what. The field names more
	// than one target here — it also says `0010:C2 stands` — so the
	// assertion is that the element edge is among them, not that it is
	// alone.
	if got := kindTo("0013", edge.Overrides); !contains(got, "0010:C1") {
		t.Errorf("overrides edges = %v, want one naming 0010:C1", got)
	}
	// A demotion leaves a moved-to edge to where the work went.
	if got := kindTo("0014", edge.MovedTo); !contains(got, "0013") {
		t.Errorf("moved-to edges = %v, want one naming 0013", got)
	}

	// The worklist: everything pointing at the moved contract, by element.
	back := scan.Reverse(list)
	var citing []string
	for _, e := range back["0010:C1"] {
		if e.Kind.Typed() {
			citing = append(citing, string(e.Kind)+" from "+e.From)
		}
	}
	sort.Strings(citing)
	want := []string{"overrides from 0013", "peer-evidence from 0013:A1"}
	if strings.Join(citing, "; ") != strings.Join(want, "; ") {
		t.Errorf("backlinks for 0010:C1 = %v, want %v", citing, want)
	}

	// The contract the override did NOT touch keeps a clean sheet — the
	// transfer is scoped to the element, never the whole record.
	for _, e := range back["0010:C2"] {
		if e.Kind == edge.Overrides {
			t.Errorf("0010:C2 was overridden; the transfer leaked past its element")
		}
	}

	// The old home is never rewritten: 0010 states no relation to either
	// record that moved its contract.
	for _, e := range docs["0010"].Edges {
		if e.Kind.Typed() && strings.HasPrefix(e.To, "0013") {
			t.Errorf("0010 was edited to point at its successor (%s); ownership transfer rewrites the citing side, not the old home", e.To)
		}
	}
}

// TestEvidenceBudgetIsAdvisory is the guarantee the check is worth
// nothing without: the Stage-7 prompt is explicit that an over-budget
// Evidence field alone never makes the verdict BLOCK, and `lint
// --locking` must still block only on the resolution tier.
func TestEvidenceBudgetIsAdvisory(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Recommendation 0010: Frame header\n\n## Critical Assumptions\n\n")
	b.WriteString("- **A1 [Load-bearing]**: The frame header is stable.\n")
	b.WriteString("  - **Evidence**: the anchor, and then a great deal of verification\n")
	for i := 0; i < 60; i++ {
		b.WriteString("    prose that the grounding sweep reads and must not lose.\n")
	}
	b.WriteString("  - **Status**: Verified\n")

	d := scan.Bytes([]byte(b.String()), scan.Options{})
	r := Run(d, Options{Locking: true})

	var found *Finding
	for i := range r.Findings {
		if r.Findings[i].Code == "evidence:over-budget" {
			found = &r.Findings[i]
		}
	}
	if found == nil {
		t.Fatal("a 60-line Evidence field produced no finding")
	}
	if found.Blocking {
		t.Error("evidence:over-budget is BLOCKING; the prompt is explicit that field length alone never blocks a lock")
	}
	if found.Patch != nil {
		t.Error("evidence:over-budget carries a patch; truncation is never proposed — the mass is usually real verification content")
	}
	if r.Verdict == "BLOCK" {
		t.Errorf("verdict is BLOCK on an advisory-only record: %s", r.Verdict)
	}
}

// TestEvidenceLabelIsMatchedByPrefix is the caveat that broke an earlier
// measurement pass. The corpus writes 872 plain `Evidence` labels and 36
// variants — `Evidence — the two source-checkable channel reductions`,
// `Evidence (MEASURED)`, `Evidence plan` — and a check keyed on the exact
// label undercounts in silence.
func TestEvidenceLabelIsMatchedByPrefix(t *testing.T) {
	for _, label := range []string{
		"Evidence",
		"Evidence plan",
		"Evidence (MEASURED)",
		"Evidence — the two source-checkable channel reductions",
		"Evidence (the peer half, 2026-08-22 Reconcile)",
	} {
		if !isEvidenceLabel(label) {
			t.Errorf("%q is an Evidence field and was not matched; the corpus writes it", label)
		}
	}
	// A label that merely begins with the letters is a different field.
	for _, label := range []string{"Evidential", "Evidences"} {
		if isEvidenceLabel(label) {
			t.Errorf("%q was matched as an Evidence field; the prefix must end at a word boundary", label)
		}
	}
}

// TestProseVocabularyIsScopedToFences keeps the exactness check usable.
// `stages/05-prelock.md` reads a contract from "the fenced ```normative
// block, not the surrounding prose", and unscoped the same sweep hits
// 13,412 times corpus-wide — a report nobody can act on.
func TestProseVocabularyIsScopedToFences(t *testing.T) {
	d := scan.Bytes([]byte("# Recommendation 0010: Frame header\n\n"+
		"## Normative Contracts\n\n"+
		"Every reader canonical in the surrounding prose is ordinary English.\n\n"+
		"**C1**\n\n```normative\nThe encoding is byte-identical across readers.\n```\n"), scan.Options{})
	r := Run(d, Options{})

	var inFence, outside int
	for _, f := range r.Findings {
		if f.Code != "prose:exactness" {
			continue
		}
		if strings.Contains(f.Message, "byte-identical") {
			inFence++
		} else {
			outside++
		}
	}
	if inFence != 1 {
		t.Errorf("the term of art inside the fence produced %d findings, want 1", inFence)
	}
	if outside != 0 {
		t.Errorf("%d findings came from prose outside the fence; the scope is the fence body", outside)
	}
}

// TestScaffoldRowIsReportedWithoutAPatch: a table row still carrying the
// template's `[Capability]` cells is the surviving-template defect in the
// one place the column-zero marker rule cannot reach.
func TestScaffoldRowIsReportedWithoutAPatch(t *testing.T) {
	d := scan.Bytes([]byte("# Recommendation 0010: Frame header\n\n"+
		"## Dependencies and Integration Points\n\n"+
		"| Capability | Source | Status | Impact |\n"+
		"| --- | --- | --- | --- |\n"+
		"| [Capability] | Existing / This RDR / Predecessor / Future | Available / Introduced / Deferred | [Impact] |\n"), scan.Options{})
	r := Run(d, Options{Locking: true})

	var found *Finding
	for i := range r.Findings {
		if r.Findings[i].Code == "scaffold:row" {
			found = &r.Findings[i]
		}
	}
	if found == nil {
		t.Fatal("the template's own scaffold row produced no finding")
	}
	if found.Patch != nil {
		t.Error("scaffold:row carries a patch; filling a row and deleting it are different repairs, and choosing is judgement")
	}
	if found.Blocking {
		t.Error("scaffold:row is blocking; it is conformance advice like every other surviving-template finding")
	}
}

// TestReentryNearMissFiresOnAMalformedRevisedFrom: a bracketed qualifier
// that begins `revised from` and fails the grammar is a near-miss, not a
// free-text note. Routing keys on the FORM, so the near-miss silently
// turns off every re-entry rule — a live demote pass shipped one extra
// word before the semicolon and the flow skipped the scoped re-verify
// pass on seven records while lint said nothing.
func TestReentryNearMissFiresOnAMalformedRevisedFrom(t *testing.T) {
	rec := func(status string) *scan.Document {
		return scan.Bytes([]byte("# Recommendation 0010: Frame header\n\n"+
			"## Metadata\n\n"+
			"- **Status**: "+status+"\n"+
			"- **Date**: 2026-01-01\n"), scan.Options{})
	}

	// The firing case: the date is missing, which no tolerance covers.
	d := rec("Draft [revised from Final; re-verify A2 — the date went missing]")
	r := Run(d, Options{})
	f := find(t, r, "status:reentry-near-miss")
	if f.Tier != TierResolution {
		t.Errorf("tier = %s, want resolution", f.Tier)
	}
	if f.Blocking {
		t.Error("near-miss blocks off a lock gate; mid-flow it is a to-do")
	}
	if f.LineStart != 5 {
		t.Errorf("finding points at line %d, want the Status line 5", f.LineStart)
	}
	if !strings.Contains(f.Fix, "revised from Final YYYY-MM-DD;") {
		t.Errorf("fix does not quote the canonical spelling: %q", f.Fix)
	}

	// At a lock gate the same near-miss blocks, like the tier's peers.
	if f := find(t, Run(d, Options{Locking: true}), "status:reentry-near-miss"); !f.Blocking {
		t.Error("near-miss does not block at a lock gate")
	}

	// The non-firing cases: the canonical spelling, the tolerated
	// stage-token spelling a live pass wrote, and a bracketed note that
	// never claimed to be a re-entry.
	for name, status := range map[string]string{
		"canonical":   "Draft [revised from Final 2026-03-04; re-verify A2,A4 — the frame width was never pinned]",
		"stage token": "Draft [revised from Final 2026-08-29 cluster-reconcile; re-verify none — wording/cross-reference fixes only]",
		"plain note":  "Draft [unblocked — the predecessor reached Implemented]",
	} {
		if r := Run(rec(status), Options{}); has(r, "status:reentry-near-miss") {
			t.Errorf("%s: near-miss fired on %q (codes: %v)", name, status, codes(r))
		}
	}
}

// TestPeerElementHintNamesRealElements: the fix for a bare peer citation
// used to print `NNNN:A3, NNNN:C4` — placeholders that looked real and
// were pasted into sub-agent prompts as if they were. The ids it offers
// now are the peer's own, in the citation's spelling; the command it
// prints resolves as written; and a peer not in hand gets the form and
// no id.
func TestPeerElementHintNamesRealElements(t *testing.T) {
	dir := t.TempDir()
	body := "# Recommendation 0011: Header\n\n## Metadata\n\n- **Date**: 2026-08-01\n" +
		"- **Status**: Draft\n- **Profile**: standard\n\n## Critical Assumptions\n\n" +
		"- **A1**: ten bytes is enough\n  - **Status**: Verified\n  - **Method**: Peer RDR\n" +
		"  - **Evidence**: `cli/0010` fixes the field order.\n  - **If wrong**: overrun.\n"
	if err := os.WriteFile(filepath.Join(dir, "0011-header.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := scan.File(filepath.Join(dir, "0011-header.md"), scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	fix := func(opts Options) string {
		t.Helper()
		for _, f := range Run(d, opts).Findings {
			if f.Code == "peer-evidence:no-element" {
				return f.Fix
			}
		}
		t.Fatalf("the bare peer citation raised no finding: %v", codes(Run(d, opts)))
		return ""
	}
	withPeer := fix(Options{Corpus: []*scan.Document{corpus(t)["0010"]}})
	for _, want := range []string{"cli/0010:C1 \"", "cli/0010:C2 \"", "cli/0010:A1 \"The reader tolerates no reordering.\"",
		"not the labels the author wrote", "`rdr inspect cli/0010`"} {
		if !strings.Contains(withPeer, want) {
			t.Errorf("hint with the peer in hand lacks %q: %s", want, withPeer)
		}
	}
	if strings.Contains(withPeer, "among others") {
		t.Errorf("hint still elides the peer's elements: %s", withPeer)
	}
	alone := fix(Options{})
	if strings.Contains(alone, ":A1") || strings.Contains(alone, ":A3") || strings.Contains(alone, ":C4") {
		t.Errorf("hint without the peer invents an id: %s", alone)
	}
	if !strings.Contains(alone, "cli/0010:A<n>") || !strings.Contains(alone, "`rdr inspect cli/0010`") {
		t.Errorf("hint without the peer lost the form or the command: %s", alone)
	}
}

// record builds a minimal document for the peer rules: a Metadata block
// carrying the Overrides value given, and one Peer-RDR assumption whose
// Evidence is the text given.
func record(t *testing.T, number, overrides, evidence string) *scan.Document {
	t.Helper()
	var b strings.Builder
	b.WriteString("# Recommendation " + number + ": Peer shapes\n\n## Metadata\n\n")
	b.WriteString("- **Date**: 2026-08-29\n- **Status**: Draft\n")
	if overrides != "" {
		b.WriteString("- **Overrides**: " + overrides + "\n")
	}
	b.WriteString("\n## Critical Assumptions\n\n- **A1 [Load-bearing]**: the peer holds.\n")
	b.WriteString("  - **Status**: Verified\n  - **Method**: Peer RDR\n")
	b.WriteString("  - **Evidence**: " + evidence + "\n")
	return scan.Bytes([]byte(b.String()), scan.Options{})
}

func pair(t *testing.T, a, b *scan.Document) []*scan.Document {
	t.Helper()
	docs := []*scan.Document{a, b}
	scan.NewResolver(docs, "").ResolveAll(docs)
	return docs
}

// TestPeerEvidenceIsDischargedPerPeer is the 0113 refine session's shape:
// the Evidence cites `cli/0112:A11` and `cli/0112:A7`, then says
// `cli/0112 is Draft`. The bare mention is prose once an element cite
// stands beside it; six lint iterations pronoun-ified accurate text to
// clear a finding that should not have been lit. A peer with only bare
// mentions still fires, and a second peer is judged on its own cites.
func TestPeerEvidenceIsDischargedPerPeer(t *testing.T) {
	cited := record(t, "0113", "",
		"cli/0112:A11 states the scan posture and cli/0112:A7 the fold; cli/0112 is Draft, so 7.1 reconciles.")
	if r := Run(cited, Options{}); has(r, "peer-evidence:no-element") {
		t.Errorf("a bare mention beside two element cites to the same peer fired: %v", codes(r))
	}

	bare := record(t, "0113", "", "cli/0112 is Draft and its fold is the authority here.")
	if r := Run(bare, Options{}); !has(r, "peer-evidence:no-element") {
		t.Errorf("a peer cited only as a record did not fire: %v", codes(r))
	}

	mixed := record(t, "0113", "", "cli/0112:A11 holds; cli/0092 is where the purpose rungs live.")
	r := Run(mixed, Options{})
	var fired []string
	for _, f := range r.Findings {
		if f.Code == "peer-evidence:no-element" {
			fired = append(fired, f.Message)
		}
	}
	if len(fired) != 1 || !strings.Contains(fired[0], "cli/0092") {
		t.Errorf("element cite to one peer must not discharge a second peer's bare mention: %v", fired)
	}
}

// TestPeerEvidenceIsDischargedByAColonClauseCite: the template's clause
// form `cli/0112:L-3` is an element cite — it resolves against the peer's
// minted clauses and discharges the per-peer rule exactly as `:A11`
// does. A colon cite of a label the peer never minted is an element cite
// that names the wrong thing — as `:A99` is — so it is reported as the
// edge that does not resolve rather than as a missing element. The
// spaced spelling `cli/0112 L-3` is a document mention by design and
// still fires.
func TestPeerEvidenceIsDischargedByAColonClauseCite(t *testing.T) {
	peer := scan.Bytes([]byte("# Recommendation 0112: Fold\n\n## Metadata\n\n- **Status**: Draft\n\n"+
		"## Normative Contracts\n\n**C1**\n\n```normative\n"+
		"L-3  the fold admits a record once.\n"+
		"REQ-12a (the emission shape): no row is dropped.\n```\n"), scan.Options{})

	for _, tc := range []struct {
		name, evidence string
		fires          bool
		unresolved     string
	}{
		{"minted clause", "cli/0112:L-3 admits it once.", false, ""},
		{"minted REQ clause", "cli/0112:REQ-12a keeps every row.", false, ""},
		{"unminted clause", "cli/0112:L-99 would admit it.", false, "cli/0112:L-99"},
		{"spaced spelling", "cli/0112 L-3 admits it once.", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := record(t, "0113", "", tc.evidence)
			docs := pair(t, d, peer)
			r := Run(d, Options{Corpus: docs})
			if has(r, "peer-evidence:no-element") != tc.fires {
				t.Errorf("peer-evidence:no-element fired=%v, want %v: %v", !tc.fires, tc.fires, codes(r))
			}
			var dangling []string
			for _, f := range r.Findings {
				if f.Code == "edge:unresolved" {
					dangling = append(dangling, f.Message)
				}
			}
			switch {
			case tc.unresolved == "" && len(dangling) != 0:
				t.Errorf("a resolvable cite was reported dangling: %v", dangling)
			case tc.unresolved != "" && (len(dangling) != 1 || !strings.Contains(dangling[0], tc.unresolved)):
				t.Errorf("an unminted clause id was not reported dangling: %v", dangling)
			}
		})
	}
}

// TestOwnershipMutualIsCorroboratedFromBothFields is the other half of the
// same session. 0113's Overrides field overrides cli/0092 and names
// cli/0112 inside that clause as context; 0112's field names 0113 the
// same way. Both tokens mint overrides edges, so the old rule read a
// cycle out of two context mentions. The pair is asserted only when each
// record LEADS a clause of the other's field.
func TestOwnershipMutualIsCorroboratedFromBothFields(t *testing.T) {
	a := record(t, "0113",
		"cli/0103 REQ-13's file grain; **cli/0092**'s `classifyPurpose` default rung is overridden only where the inference lands outside cli/0112's fold admission band.",
		"cli/0112:A11")
	b := record(t, "0112",
		"cli/0103 REQ-40 narrows-only selection; also narrows Draft cli/0106 I-4(a), which is what cli/0113 E-1's ordinal orders.",
		"cli/0113:A1")
	docs := pair(t, a, b)
	for _, d := range docs {
		if r := Run(d, Options{Corpus: docs}); has(r, "ownership:mutual") {
			t.Errorf("%s: two context mentions were read as a cycle: %v", d.Record, find(t, r, "ownership:mutual").Message)
		}
	}

	// The real cycle still fires from either side.
	x := record(t, "0113", "cli/0112's fold, replaced by the grammar's ordinal.", "cli/0112:A1")
	y := record(t, "0112", "cli/0113 E-1's ordinal, replaced by the fold.", "cli/0113:A1")
	docs = pair(t, x, y)
	for _, d := range docs {
		if r := Run(d, Options{Corpus: docs}); !has(r, "ownership:mutual") {
			t.Errorf("%s: a corroborated mutual override did not fire: %v", d.Record, codes(r))
		}
	}

	// One side leading, the other mentioning, is not a cycle either —
	// and a peer not in hand asserts nothing.
	if r := Run(x, Options{Corpus: pair(t, x, b)}); has(r, "ownership:mutual") {
		t.Error("a lead on one side and a context mention on the other was asserted mutual")
	}
	if r := Run(x, Options{Corpus: []*scan.Document{x}}); has(r, "ownership:mutual") {
		t.Error("mutual asserted with the peer out of hand")
	}
}

// TestPeerElementHintListsEveryIdBounded: a refine pass read a hint that
// named only the first A and first C "among others", then guessed the
// peer's authored contract label as an element id. The hint now lists
// every contract and assumption with its handle clipped to a few words,
// contracts first, caps the list with a "+N more" tail, and says the
// authored handles are not ids.
func TestPeerElementHintListsEveryIdBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Recommendation 0142: Peer\n\n## Metadata\n\n- **Status**: Draft\n\n## Critical Assumptions\n\n")
	for i := 1; i <= 14; i++ {
		b.WriteString("- **A" + itoa(i) + " assumption number " + itoa(i) + " holds in every case we checked so far.**\n")
		b.WriteString("  - **Status**: Verified\n  - **Method**: Code inspection\n  - **Evidence**: seen.\n  - **If wrong**: no.\n")
	}
	b.WriteString("\n## Normative Contracts\n\n**C1**\n\n```normative\nF-1 (the retirement floor): a retired flag names its successor.\n```\n")
	peer := scan.Bytes([]byte(b.String()), scan.Options{})
	d := record(t, "0143", "", "cli/0142 settles the floor.")
	var hint string
	for _, f := range Run(d, Options{Corpus: pair(t, d, peer)}).Findings {
		if f.Code == "peer-evidence:no-element" {
			hint = f.Fix
		}
	}
	if hint == "" {
		t.Fatal("the bare peer citation raised no finding")
	}
	for _, want := range []string{
		"holds cli/0142:C1 \"F-1 (the retirement floor): a\u2026\", cli/0142:A1 \"assumption number 1 holds in\u2026\"",
		"cli/0142:A11 \"", ", +3 more;", "not the labels the author wrote (F-1, P-a)", "`rdr inspect cli/0142`",
	} {
		if !strings.Contains(hint, want) {
			t.Errorf("hint lacks %q: %s", want, hint)
		}
	}
	for _, bad := range []string{"cli/0142:A12", "cli/0142:F-1", "among others"} {
		if strings.Contains(hint, bad) {
			t.Errorf("hint carries %q: %s", bad, hint)
		}
	}
	if n := strings.Count(hint, "cli/0142:"); n != peerHintIDs {
		t.Errorf("hint lists %d ids, want %d: %s", n, peerHintIDs, hint)
	}
	if got := peerHintLabel("  "); got != "" {
		t.Errorf("an empty handle rendered as %q", got)
	}
	if got := peerHintLabel("averyveryverylongsinglewordthatoverrunsthefortyrunebound"); got != "\"averyveryverylongsinglewordthatoverrunst\u2026\"" {
		t.Errorf("a long single word was not clipped by rune: %q", got)
	}
}

// TestOwnershipMutualProseSemicolonAndFixQuotesTheClause is the refine
// session that deleted a cross-pointer: the field's second sentence
// names a peer after a prose `;` ("…; this field is the record. Also
// owed to a peer, not an override: cli/NNNN …"), the peer overrides
// this record back, and the fix text named neither the clause nor the
// `;`. A reference that does not OPEN its clause is a mention, so the
// pair is not mutual; and when a pair is, the fix quotes the clause the
// reference opens and says how to rewrite it.
func TestOwnershipMutualProseSemicolonAndFixQuotesTheClause(t *testing.T) {
	a := record(t, "0106",
		"cli/0103 REQ-38's read arm; cli/0104 REQ-35's set equality → multiset. Both predecessors are Final; this field is the record. Also owed to a peer, not an override: cli/0112 L-3's prose names a spelling I-1 removes.",
		"cli/0112:L3")
	b := record(t, "0112",
		"cli/0103 REQ-40 narrows-only selection; cli/0106 I-4(a) — narrowed to the cohort space.",
		"cli/0106:A1")
	docs := pair(t, a, b)
	for _, d := range docs {
		if r := Run(d, Options{Corpus: docs}); has(r, "ownership:mutual") {
			t.Errorf("%s: a mention after a prose semicolon was read as an override: %v", d.Record, find(t, r, "ownership:mutual").Message)
		}
	}

	// The same field with the peer LEADING its clause is the cycle, and
	// the fix says which token was read and what to do with it.
	x := record(t, "0106",
		"cli/0103 REQ-38's read arm; cli/0112 L-3's prose names a spelling I-1 removes, not an override.",
		"cli/0112:L3")
	f := find(t, Run(x, Options{Corpus: pair(t, x, b)}), "ownership:mutual")
	for _, want := range []string{
		"cli/0112 at L",
		"opens the `;`-clause \"cli/0112 L-3's prose names a spelling I-1 removes, not an override.\"",
		"if it is a mention, rewrite",
		"if it is the override, remove the other record's line",
	} {
		if !strings.Contains(f.Fix, want) {
			t.Errorf("fix %q lacks %q", f.Fix, want)
		}
	}
}
