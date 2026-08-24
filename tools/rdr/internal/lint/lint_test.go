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

// TestGatePointerSubsectionsAreNotMissing: from epoch C on, lock replaces
// the Finalization Gate body with a pointer to gate.md. Advising a record
// to restore the five gate subsections would be advising it to undo the
// current process.
func TestGatePointerSubsectionsAreNotMissing(t *testing.T) {
	r := report(t, "0010", Options{})
	for _, f := range r.Findings {
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
