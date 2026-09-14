package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// kinds renders each kind as "count/derived", where derived is the
// LABELLING BACKLOG. A kind the template does not label (BR, F, MVV)
// reports a 0 backlog however its ids were minted, and carries its minted
// count in the structural column instead — rendered as "count/0+n" so a
// fixture states which column the ids landed in.
func kinds(doc *Document) map[ident.Kind]string {
	out := map[ident.Kind]string{}
	for _, k := range ident.Kinds {
		if k == ident.Section {
			continue
		}
		n := doc.Counts.Elements[k]
		if n == 0 {
			continue
		}
		s := itoa(n) + "/" + itoa(doc.Counts.Derived[k])
		if st := doc.Counts.Structural[k]; st > 0 {
			s += "+" + itoa(st)
		}
		out[k] = s
	}
	return out
}

func itoa(n int) string { return strconv.Itoa(n) }

func element(t *testing.T, doc *Document, id string) Element {
	t.Helper()
	for _, e := range doc.Elements {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("no element %s in %s; have %v", id, doc.Record, ids(doc))
	return Element{}
}

func ids(doc *Document) []string {
	var out []string
	for _, e := range doc.Elements {
		out = append(out, e.ID)
	}
	return out
}

// TestFixtureTallies pins hand-counted element sets per whole-record fixture:
// which kinds appear, how many, and how many are derived.
func TestFixtureTallies(t *testing.T) {
	cases := []struct {
		file   string
		record string
		want   map[ident.Kind]string // kind → "count/derived"
		ids    []string              // a few IDs that must exist
	}{
		{"legacy-shape.md", "0001",
			map[ident.Kind]string{"ALT": "1/0", "BR": "1/0+1", "S": "1/0", "MVV": "1/0", "G": "5/0"},
			[]string{"0001:G-contradiction", "0001:G-proportionality", "0001:ALT1"}},
		{"assumptions-nested.md", "0002",
			map[ident.Kind]string{"A": "2/0", "D": "2/0", "ALT": "1/0", "BR": "1/0+1", "S": "1/0", "MVV": "1/0", "G": "5/0"},
			[]string{"0002:A1", "0002:A2", "0002:D-identity", "0002:D-selection-predicate"}},
		{"gate-inline.md", "0003",
			map[ident.Kind]string{"A": "2/0", "C": "1/1", "BR": "1/0+1", "S": "1/0", "MVV": "1/0"},
			[]string{"0003:C1", "0003:BR1"}},
		{"current-shape.md", "0004",
			map[ident.Kind]string{"A": "3/0", "C": "1/1", "D": "2/0", "ALT": "2/0", "BR": "1/0+1", "S": "1/0", "MVV": "1/0"},
			[]string{"0004:A1", "0004:A3", "0004:C1", "0004:D-wire-byte-format", "0004:D-naming", "0004:ALT2", "0004:MVV", "0004:S1"}},
	}
	for _, c := range cases {
		doc := Bytes(fixture(t, c.file), Options{})
		if doc.Record != c.record {
			t.Errorf("%s: record %s, want %s", c.file, doc.Record, c.record)
		}
		got := kinds(doc)
		for k, want := range c.want {
			if got[k] != want {
				t.Errorf("%s: %s = %q, want %q", c.file, k, got[k], want)
			}
		}
		for k := range got {
			if _, ok := c.want[k]; !ok {
				t.Errorf("%s: unexpected kind %s = %s", c.file, k, got[k])
			}
		}
		for _, id := range c.ids {
			element(t, doc, id)
		}
		if len(doc.Warnings) != 0 {
			t.Errorf("%s: warnings on a conformant fixture: %+v", c.file, doc.Warnings)
		}
	}
}

// TestGateElementsOnlyWhenInlined: a gate.md pointer means the responses
// live outside the record, so there is nothing to address.
func TestGateElementsOnlyWhenInlined(t *testing.T) {
	if n := Bytes(fixture(t, "gate-inline.md"), Options{}).Counts.Elements[ident.Gate]; n != 0 {
		t.Errorf("gate-inline (gate pointer) has %d G elements, want 0", n)
	}
	if n := Bytes(fixture(t, "legacy-shape.md"), Options{}).Counts.Elements[ident.Gate]; n != 5 {
		t.Errorf("legacy-shape (inlined gate) has %d G elements, want 5", n)
	}
}

// TestLineRangesRoundTrip: every element's ID selects the bytes it names,
// and those bytes start where the element's grammar says they start.
func TestLineRangesRoundTrip(t *testing.T) {
	for _, f := range []string{"legacy-shape.md", "assumptions-nested.md", "gate-inline.md", "current-shape.md"} {
		doc := Bytes(fixture(t, f), Options{Project: "cli"})
		for _, e := range doc.Elements {
			start, end, ok := doc.Select(e.ID)
			if !ok || start != e.LineStart || end != e.LineEnd {
				t.Errorf("%s: Select(%s) = %d-%d %v, want %d-%d", f, e.ID, start, end, ok, e.LineStart, e.LineEnd)
			}
			if _, _, ok := doc.Select(strings.TrimPrefix(e.ID, "cli/")); !ok {
				t.Errorf("%s: local form of %s does not select", f, e.ID)
			}
			if _, _, ok := doc.Select("other/" + strings.TrimPrefix(e.ID, "cli/")); ok {
				t.Errorf("%s: %s selected under a foreign project prefix", f, e.ID)
			}
			first := strings.TrimSpace(doc.Line(e.LineStart))
			switch e.Kind {
			case ident.Assumption:
				if !strings.HasPrefix(first, "- **A"+e.Key) {
					t.Errorf("%s: %s starts at %q", f, e.ID, first)
				}
			case ident.Contract:
				if !strings.HasPrefix(first, "```normative") && !strings.Contains(first, "C"+e.Key) {
					t.Errorf("%s: %s starts at %q", f, e.ID, first)
				}
				if last := strings.TrimSpace(doc.Line(e.LineEnd)); last != "```" {
					t.Errorf("%s: %s ends at %q, want the closing fence", f, e.ID, last)
				}
			case ident.Alternative, ident.Gate:
				if e.Kind == ident.Alternative && !strings.HasPrefix(first, "### Alternative "+e.Key) {
					t.Errorf("%s: %s starts at %q", f, e.ID, first)
				}
			case ident.Decision, ident.Rejected, ident.Scenario, ident.Failure, ident.RoundTrip:
				if !strings.HasPrefix(first, "- ") && !strings.HasPrefix(first, e.Key+".") {
					t.Errorf("%s: %s starts at %q", f, e.ID, first)
				}
			}
			if strings.TrimSpace(doc.Line(e.LineEnd)) == "" {
				t.Errorf("%s: %s ends on a blank line", f, e.ID)
			}
		}
		for _, n := range doc.Outline {
			start, end, ok := doc.Select(n.ID)
			if !ok || start != n.LineStart || end != n.LineEnd {
				t.Errorf("%s: Select(%s) = %d-%d %v", f, n.ID, start, end, ok)
			}
		}
	}
}

// TestOutlineNestsAndCovers: children lie inside their parent, siblings do
// not overlap, and every non-blank line is inside the deepest node that
// claims it — no line of a record is outside the outline.
func TestOutlineNestsAndCovers(t *testing.T) {
	for _, f := range []string{"legacy-shape.md", "current-shape.md"} {
		doc := Bytes(fixture(t, f), Options{})
		byID := map[string]Node{}
		for _, n := range doc.Outline {
			byID[n.ID] = n
		}
		for i, n := range doc.Outline {
			if n.Parent != "" {
				p := byID[n.Parent]
				if n.LineStart < p.LineStart || n.LineEnd > p.LineEnd {
					t.Errorf("%s: %s (%d-%d) outside parent %s (%d-%d)", f, n.ID, n.LineStart, n.LineEnd, p.ID, p.LineStart, p.LineEnd)
				}
			}
			for _, m := range doc.Outline[i+1:] {
				if m.Parent == n.Parent && m.LineStart <= n.LineEnd && n.LineStart <= m.LineEnd {
					t.Errorf("%s: siblings %s and %s overlap", f, n.ID, m.ID)
				}
			}
		}
		covered := make([]bool, doc.Lines+1)
		for _, n := range doc.Outline {
			for i := n.LineStart; i <= n.LineEnd; i++ {
				covered[i] = true
			}
		}
		for i := 1; i <= doc.Lines; i++ {
			if !covered[i] && strings.TrimSpace(doc.Line(i)) != "" {
				t.Errorf("%s: line %d is outside every outline node", f, i)
			}
		}
	}
}

// mutate applies a line-level edit to a fixture and returns the bytes.
func mutate(raw []byte, fn func(lines []string) []string) []byte {
	lines := strings.Split(string(raw), "\n")
	return []byte(strings.Join(fn(lines), "\n"))
}

// section returns the 0-based [start, end) of the top-level `## heading`
// block in lines.
func section(t *testing.T, lines []string, heading string) (int, int) {
	t.Helper()
	start := -1
	for i, l := range lines {
		if l == "## "+heading {
			start = i
			continue
		}
		if start >= 0 && strings.HasPrefix(l, "## ") {
			return start, i
		}
	}
	if start < 0 {
		t.Fatalf("no ## %s", heading)
	}
	return start, len(lines)
}

// TestStabilityUnderProseEdits: editing prose elsewhere changes no element
// ID and no element hash; only the edited section's own node hash moves.
func TestStabilityUnderProseEdits(t *testing.T) {
	raw := fixture(t, "current-shape.md")
	before := Bytes(raw, Options{})
	edited := mutate(raw, func(lines []string) []string {
		s, e := section(t, lines, "Problem Statement")
		for i := s + 1; i < e; i++ {
			lines[i] = strings.ReplaceAll(lines[i], "helper", "shared helper routine")
		}
		lines[s+1] = "An extra paragraph inserted into the problem statement."
		return append(lines[:s+2], append([]string{"", "And another."}, lines[s+2:]...)...)
	})
	after := Bytes(edited, Options{})

	if len(after.Elements) != len(before.Elements) {
		t.Fatalf("element count changed: %d → %d", len(before.Elements), len(after.Elements))
	}
	for i := range before.Elements {
		b, a := before.Elements[i], after.Elements[i]
		if a.ID != b.ID || a.Hash != b.Hash || a.Derived != b.Derived {
			t.Errorf("%s changed: id %s hash %s derived %v", b.ID, a.ID, a.Hash, a.Derived)
		}
		if a.LineStart <= b.LineStart {
			t.Errorf("%s did not shift down past the inserted lines", b.ID)
		}
	}
	for i := range before.Outline {
		b, a := before.Outline[i], after.Outline[i]
		if a.ID != b.ID {
			t.Errorf("section %s renamed to %s", b.ID, a.ID)
		}
		moved := a.Hash != b.Hash
		if own := b.ID == "0004:§problem-statement" || b.ID == "0004:§title"; moved != own {
			t.Errorf("section %s hash moved=%v, want %v", b.ID, moved, own)
		}
	}
}

// TestStabilityUnderReorder: swapping two unrelated top-level sections
// changes no element ID and no element hash.
func TestStabilityUnderReorder(t *testing.T) {
	raw := fixture(t, "current-shape.md")
	before := Bytes(raw, Options{})
	swapped := mutate(raw, func(lines []string) []string {
		cs, ce := section(t, lines, "Context")
		rs, re := section(t, lines, "Research Findings")
		if ce != rs {
			t.Fatalf("fixture layout changed: Context %d-%d, Research Findings %d-%d", cs, ce, rs, re)
		}
		out := append([]string{}, lines[:cs]...)
		out = append(out, lines[rs:re]...)
		out = append(out, lines[cs:ce]...)
		return append(out, lines[re:]...)
	})
	after := Bytes(swapped, Options{})

	key := func(doc *Document) map[string]string {
		m := map[string]string{}
		for _, e := range doc.Elements {
			m[e.ID] = e.Hash
		}
		for _, n := range doc.Outline {
			if n.ID != "0004:§title" {
				m[n.ID] = n.Hash
			}
		}
		return m
	}
	b, a := key(before), key(after)
	if len(a) != len(b) {
		t.Fatalf("id set size changed: %d → %d", len(b), len(a))
	}
	for id, h := range b {
		if a[id] != h {
			t.Errorf("%s: hash %s → %s", id, h, a[id])
		}
	}
	if len(after.Warnings) != 0 {
		t.Errorf("reorder produced warnings: %+v", after.Warnings)
	}
}

// TestDerivedIDChangesOnlyWithOwnContent: editing a contract's body keeps
// its derived ID and moves only its hash.
func TestDerivedIDChangesOnlyWithOwnContent(t *testing.T) {
	raw := fixture(t, "current-shape.md")
	before := Bytes(raw, Options{})
	c1 := element(t, before, "0004:C1")
	if !c1.Derived {
		t.Fatalf("fixture C1 should be derived")
	}
	edited := mutate(raw, func(lines []string) []string {
		lines[c1.LineStart] = strings.Replace(lines[c1.LineStart], "0x1EDC6F41", "0x04C11DB7", 1)
		return lines
	})
	after := Bytes(edited, Options{})
	c1b := element(t, after, "0004:C1")
	if c1b.Hash == c1.Hash {
		t.Errorf("contract edit left hash %s alone", c1.Hash)
	}
	for _, e := range before.Elements {
		if e.ID == "0004:C1" {
			continue
		}
		if element(t, after, e.ID).Hash != e.Hash {
			t.Errorf("%s hash moved on an unrelated contract edit", e.ID)
		}
	}
}

// TestLabelledContractIsAsWritten: an author's `**C4**` before a fence is
// honoured, the label line joins the element's range, and a derived
// ordinal that a label already claims yields with a warning.
func TestLabelledContractIsAsWritten(t *testing.T) {
	raw := fixture(t, "current-shape.md")
	fenceLine := element(t, Bytes(raw, Options{}), "0004:C1").LineStart

	labelled := mutate(raw, func(lines []string) []string {
		return append(lines[:fenceLine-1], append([]string{"**C4** — the checksum surface", ""}, lines[fenceLine-1:]...)...)
	})
	doc := Bytes(labelled, Options{})
	c4 := element(t, doc, "0004:C4")
	if c4.Derived || c4.LineStart != fenceLine || c4.Label != "the checksum surface" {
		t.Errorf("C4 = %+v; want as-written, starting on the label line %d", c4, fenceLine)
	}
	if doc.Counts.Derived[ident.Contract] != 0 || len(doc.Warnings) != 0 {
		t.Errorf("derived %d, warnings %+v", doc.Counts.Derived[ident.Contract], doc.Warnings)
	}

	// A prose lead that merely mentions C2 is not a label.
	prose := mutate(raw, func(lines []string) []string {
		return append(lines[:fenceLine-1], append([]string{"**C2 — this is a claim, not a label**", ""}, lines[fenceLine-1:]...)...)
	})
	if e := element(t, Bytes(prose, Options{}), "0004:C1"); !e.Derived {
		t.Errorf("bold prose mentioning C2 was read as a label: %+v", e)
	}

	// Two fences; the second labelled C1: the first cannot be C1 by
	// ordinal, so it takes the first free number above the count.
	mixed := mutate(raw, func(lines []string) []string {
		second := []string{"", "**C1**", "```normative", "func Verify(frame []byte) error", "```"}
		end := element(t, Bytes(raw, Options{}), "0004:C1").LineEnd
		return append(lines[:end], append(second, lines[end:]...)...)
	})
	doc = Bytes(mixed, Options{})
	if e := element(t, doc, "0004:C1"); e.Derived || !strings.Contains(doc.Line(e.LineStart+2), "Verify") {
		t.Errorf("labelled C1 did not win its key: %+v", e)
	}
	if e := element(t, doc, "0004:C3"); !e.Derived {
		t.Errorf("displaced derived contract = %+v, want derived C3", e)
	}
	if len(doc.Warnings) != 1 || doc.Warnings[0].Code != "c:collision" {
		t.Errorf("warnings = %+v, want one c:collision", doc.Warnings)
	}
}

// TestClausesAreMintedUnderTheirContract: a labelled clause inside a
// normative fence is an element of its own, keyed by the label as written,
// with its contract as Parent; the label must open a definition (two
// spaces, ` (`, `:`, or a bold close), not a wrapped line of prose; and a
// label defined twice in the record mints nothing and warns with both
// candidates, so `--select` can refuse it as ambiguous rather than pick.
func TestClausesAreMintedUnderTheirContract(t *testing.T) {
	raw := []byte(`# Recommendation 0009: Layers

## Metadata

- **Status**: Draft

#### Normative Contracts

**C1**

` + "```normative" + `
L-1  input := owned records (cli/0110 O-3), each
     carrying a label; L-2 reads them.
L-2 (chain state): the fold at render point p.

L-3 tolerates nothing here — this line opens with a label but is prose.
**REQ-4b (the emission shape).** No row is dropped.
NC-5(2) is a reference to a sub-item, not a definition.
` + "```" + `

**C2**

` + "```normative" + `
E-1:  ordinal of a record within its file.
` + "```" + `
`)
	doc := Bytes(raw, Options{})
	want := map[string][3]any{ // id → parent, label, line range
		"0009:L-1":    {"0009:C1", "input := owned records (cli/0110 O-3), each", [2]int{12, 13}},
		"0009:L-2":    {"0009:C1", "(chain state): the fold at render point p.", [2]int{14, 16}},
		"0009:REQ-4b": {"0009:C1", "(the emission shape). No row is dropped.", [2]int{17, 18}},
		"0009:E-1":    {"0009:C2", "ordinal of a record within its file.", [2]int{24, 24}},
	}
	for id, w := range want {
		e := element(t, doc, id)
		if e.Kind != ident.Clause || e.Parent != w[0] || e.Label != w[1] || e.Derived {
			t.Errorf("%s = %+v; want clause under %s labelled %q", id, e, w[0], w[1])
		}
		if r := w[2].([2]int); e.LineStart != r[0] || e.LineEnd != r[1] {
			t.Errorf("%s spans %d-%d, want %d-%d", id, e.LineStart, e.LineEnd, r[0], r[1])
		}
	}
	for _, e := range doc.Elements {
		if e.Kind == ident.Clause && (e.Key == "L-3" || e.Key == "NC-5") {
			t.Errorf("prose opening with a label was minted: %+v", e)
		}
	}
	if doc.Counts.Elements[ident.Clause] != 4 || doc.Counts.Derived[ident.Clause] != 0 || len(doc.Warnings) != 0 {
		t.Errorf("counts %+v, warnings %+v", doc.Counts, doc.Warnings)
	}
	// Selecting a clause returns its lines, and the contract still spans them all.
	if s, e, ok := doc.Select("0009:L-2"); !ok || s != 14 || e != 16 {
		t.Errorf("Select(L-2) = %d-%d %v", s, e, ok)
	}
	if c := element(t, doc, "0009:C1"); c.LineStart > 12 || c.LineEnd < 18 {
		t.Errorf("C1 no longer spans its clauses: %+v", c)
	}

	// The same label defined under two contracts: no id, one warning naming both.
	dup := Bytes(bytes.Replace(raw, []byte("E-1:  ordinal"), []byte("L-1:  ordinal"), 1), Options{})
	for _, e := range dup.Elements {
		if e.Key == "L-1" {
			t.Errorf("a duplicated label was minted: %+v", e)
		}
	}
	if len(dup.Warnings) != 1 || dup.Warnings[0].Code != "clause:duplicate" ||
		!strings.Contains(dup.Warnings[0].Message, "0009:C1 12-13") || !strings.Contains(dup.Warnings[0].Message, "0009:C2 24-24") {
		t.Errorf("warnings = %+v; want one clause:duplicate naming both candidates", dup.Warnings)
	}
	if why := dup.Ambiguity("0009:L-1"); !strings.HasPrefix(why, "L-1 is defined") {
		t.Errorf("Ambiguity(L-1) = %q", why)
	}
	if dup.Ambiguity("0009:L-2") != "" || dup.Ambiguity("0009:C1") != "" {
		t.Error("Ambiguity answered for an id that is not a duplicated clause")
	}
	// A clause id can never collide with another kind's: the hyphen form
	// is a clause, the bare form is the list kind, and D-/G- stay slugs.
	for _, id := range []string{"0009:S1", "0009:F1", "0009:D-1"} {
		if _, _, ok := doc.Select(id); ok {
			t.Errorf("%s resolved on a record with no such element", id)
		}
	}
}

// TestTransientContract: the Transient marker inside a block flags it.
func TestTransientContract(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0009: Bridge

## Metadata

- **Status**: Draft

## Critical Assumptions

- **A1 [x]**
  - **Status**: Verified

#### Normative Contracts

`+"```normative"+`
Transient — scheduled deletion by 0010-successor, Phase 1; delete the shim.
func Shim() error
`+"```"+`
`), Options{})
	if e := element(t, doc, "0009:C1"); !e.Transient {
		t.Errorf("contract not flagged transient: %+v", e)
	}
}

// TestContractMarkersBelowTheFence: the corpus writes a marker as a
// blockquote UNDER the fence it qualifies; it is that contract's marker,
// and the Surface marker names the root as this record's own element.
func TestContractMarkersBelowTheFence(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0009: Bridge

#### Normative Contracts

**C1**

`+"```normative"+`
type Class int
`+"```"+`

**C2**

`+"```normative"+`
func Gate(c Class) error
`+"```"+`

> Surface — of C1; refuses a value outside the taxonomy.

**C3**

`+"```normative"+`
func Shim() error
`+"```"+`

> Transient — scheduled deletion by 0010-successor, Phase 1; delete the shim.
`), Options{})
	if e := element(t, doc, "0009:C1"); e.SurfaceOf != "" || e.Transient {
		t.Errorf("root carries a marker: %+v", e)
	}
	if e := element(t, doc, "0009:C2"); e.SurfaceOf != "0009:C1" {
		t.Errorf("surface_of = %q, want 0009:C1", e.SurfaceOf)
	}
	if e := element(t, doc, "0009:C3"); !e.Transient {
		t.Errorf("contract below-fence marker not flagged transient: %+v", e)
	}
	so := edgesOf(doc, edge.SurfaceOf)
	if len(so) != 1 || so[0].From != "0009:C2" || so[0].To != "0009:C1" {
		t.Errorf("surface-of edge: %v", so)
	}
}

// TestAssumptionForms: the four corpus forms and a split label all yield
// as-written keys; a legend bullet in a labelled section is not an
// assumption; a label-free checkbox list is addressed by ordinal.
func TestAssumptionForms(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0010: Forms

## Metadata

- **Status**: Draft

## Critical Assumptions

- **A1 [Bracketed statement that
  wraps]**
  - **Status**: Verified
- **A2 — Dashed statement.**
  - **Status**: Verified
- **A3** Bold closes on the label
  - **Status**: Verified
- **A4 Bare statement in bold**
  - **Status**: Verified
- **A4.b Split label**
  - **Status**: Verified

**Method vocabulary** (pick one):

- **Source Search** — verified against source.
`), Options{})
	want := map[string]string{
		"0010:A1":  "Bracketed statement that wraps",
		"0010:A2":  "Dashed statement.",
		"0010:A3":  "Bold closes on the label",
		"0010:A4":  "Bare statement in bold",
		"0010:A4b": "Split label",
	}
	if n := doc.Counts.Elements[ident.Assumption]; n != len(want) {
		t.Errorf("A count %d, want %d: %v", n, len(want), ids(doc))
	}
	for id, label := range want {
		if e := element(t, doc, id); e.Label != label || e.Derived {
			t.Errorf("%s = %q derived=%v, want %q", id, e.Label, e.Derived, label)
		}
	}

	legacy := Bytes([]byte(`# Recommendation 0011: Legacy

## Metadata

- **Status**: Implemented

## Proposed Solution

### Critical Assumptions

- [x] **The first claim** — verified.
- [ ] **The second claim** — pending.
`), Options{})
	if n := legacy.Counts.Derived[ident.Assumption]; n != 2 {
		t.Errorf("checkbox assumptions derived = %d, want 2: %v", n, ids(legacy))
	}
	if e := element(t, legacy, "0011:A2"); e.Label != "The second claim" {
		t.Errorf("A2 label = %q", e.Label)
	}
}

// TestDuplicateScenarioNumbersDerive: a scenario list that restarts is
// addressed by ordinal with a warning, not by clashing numbers.
func TestDuplicateScenarioNumbersDerive(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0012: Scenarios

## Metadata

- **Status**: Draft

## Validation

### Testing Strategy

1. **Scenario**: first list, one.
   **Expected**: ok
2. **Scenario**: first list, two.
   **Expected**: ok

Negative cases:

1. **Scenario**: second list, one.
   **Expected**: refused
`), Options{})
	if n := doc.Counts.Elements[ident.Scenario]; n != 3 {
		t.Fatalf("S count %d: %v", n, ids(doc))
	}
	if e := element(t, doc, "0012:S2"); e.Derived || e.Label != "first list, two." {
		t.Errorf("S2 = %+v", e)
	}
	if e := element(t, doc, "0012:S3"); !e.Derived || e.Label != "second list, one." {
		t.Errorf("S3 = %+v", e)
	}
	if e := element(t, doc, "0012:S1"); !e.Derived {
		t.Errorf("S1 should be derived once its number is ambiguous: %+v", e)
	}
	if len(doc.Warnings) != 2 || doc.Warnings[0].Code != "s:duplicate" {
		t.Errorf("warnings = %+v", doc.Warnings)
	}
}

// TestStructuralIdsAreNotBacklog pins the split R1c drew: a minted id is
// a labelling backlog only where the template gives the kind somewhere to
// write one. BR and F have no such place, so their ids — 976 of them
// corpus-wide, every one minted — are the elements' permanent identity
// and must never be counted as pending edits.
//
// The property protects the ids themselves. `index --derived` is the
// queue that says how much labelling is left; while BR and F sat in it
// reading 436/436 and 540/540, the obvious way to empty the queue was to
// stop minting their ids, which would unanchor the 236 edges written from
// those elements. Counting them apart is what makes the queue's zero mean
// what it says.
func TestStructuralIdsAreNotBacklog(t *testing.T) {
	doc := Bytes(fixture(t, "current-shape.md"), Options{})
	for _, k := range []ident.Kind{ident.Rejected, ident.Failure, ident.MVV} {
		if model.KindKeys(string(k)) {
			t.Errorf("%s: the template keys none of these kinds", k)
		}
		if n := doc.Counts.Derived[k]; n != 0 {
			t.Errorf("%s: backlog = %d, want 0 — the template labels it nowhere", k, n)
		}
	}
	// The ids are still minted, and still name bytes: BR is structural,
	// not absent.
	if n := doc.Counts.Structural[ident.Rejected]; n == 0 {
		t.Error("BR: structural = 0; the ids stopped being minted")
	}
	e := element(t, doc, "0004:BR1")
	if !e.Derived || e.Backlog || e.LineStart == 0 {
		t.Errorf("0004:BR1 = %+v, want a minted id, not a backlog, naming real lines", e)
	}
	// A labelled kind keeps reporting a backlog: the split narrows the
	// queue, it does not empty it.
	if !model.KindKeys(string(ident.Contract)) || doc.Counts.Derived[ident.Contract] != 1 {
		t.Errorf("C backlog = %d, want 1 — an unlabelled contract is still work",
			doc.Counts.Derived[ident.Contract])
	}
	if n := doc.Counts.Structural[ident.Contract]; n != 0 {
		t.Errorf("C structural = %d, want 0", n)
	}
}

// TestDerivedKeysNeverCollideWithAuthoredOnes pins the rule that makes an
// id name one element's bytes: the derived branch and the authored branch
// draw from ONE namespace. An unlabelled first list mints positional
// S1..S3 while a second list's author-written 1..3 are unique and keep
// theirs — both were minted as S1..S3, so three ids each named two
// different elements and `rdr inspect 0012:S1` was a coin flip.
func TestDerivedKeysNeverCollideWithAuthoredOnes(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0012: Scenarios

## Metadata

- **Status**: Draft

## Validation

### Testing Strategy

- unlabelled one.
- unlabelled two.
- unlabelled three.

Numbered cases:

1. **Scenario**: authored one.
   **Expected**: ok
2. **Scenario**: authored two.
   **Expected**: ok
3. **Scenario**: authored three.
   **Expected**: ok
`), Options{})
	seen := map[string]int{}
	for _, e := range doc.Elements {
		seen[e.ID]++
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("id %s names %d elements", id, n)
		}
	}
	// The author's own numbers are unique here, so they stay as written.
	for _, id := range []string{"0012:S1", "0012:S2", "0012:S3"} {
		if e := element(t, doc, id); e.Derived {
			t.Errorf("%s should be the author's own key: %+v", id, e)
		}
	}
	// The unlabelled items take a suffixed ordinal — and it must be a
	// form the ID grammar accepts, or nothing could cite it.
	for _, id := range []string{"0012:S1a", "0012:S2a", "0012:S3a"} {
		e := element(t, doc, id)
		if !e.Derived {
			t.Errorf("%s should be derived: %+v", id, e)
		}
		if !ident.IsID(id) {
			t.Errorf("%s is not a parseable element id", id)
		}
	}
}

// TestSectionSlugs: canonical sections take the canonical slug at any
// level or case; scaffolds and aliases slug their own heading; a repeat
// gets a numbered slug and a warning; an unknown heading warns.
func TestSectionSlugs(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0013: Slugs

## Metadata

- **Status**: Draft
- **Profile**: foundational

## Proposed Solution

### critical assumptions

- **A1 [x]**

### Premortem

Legacy heading.

### Decision Rationale

The real one.

## Implementation Plan

### Phase 1: Code

#### Step 1: One

#### Step 1: One again

## Something Foreign

## Finalization Gate

See gate.md.
`), Options{})
	want := map[string]string{
		"0013:§critical-assumptions": "case-variant",
		"0013:§premortem":            "recognized-unmapped",
		"0013:§decision-rationale":   "exact",
		"0013:§step-1-one":           "scaffold-instance",
		"0013:§step-1-one-again":     "scaffold-instance",
		"0013:§something-foreign":    "unknown",
		"0013:§finalization-gate":    "exact",
	}
	got := map[string]string{}
	for _, n := range doc.Outline {
		got[n.ID] = n.Match
	}
	for id, match := range want {
		if got[id] != match {
			t.Errorf("%s: match %q, want %q (have %v)", id, got[id], match, got)
		}
	}
	codes := map[string]int{}
	for _, w := range doc.Warnings {
		codes[w.Code]++
	}
	if codes["section:unknown-to-template"] != 1 || len(doc.Warnings) != 1 {
		t.Errorf("warnings = %+v", doc.Warnings)
	}
	if e := element(t, doc, "0013:A1"); e.Section != "0013:§critical-assumptions" {
		t.Errorf("A1 read from %s", e.Section)
	}
}

// TestRecordFromFilename: a record whose title carries no number takes it
// from the filename, and every ID is re-derived.
func TestRecordFromFilename(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "0099-untitled.md")
	if err := os.WriteFile(p, []byte("# An RDR with no number\n\n## Metadata\n\n- **Status**: Draft\n\n## Critical Assumptions\n\n- **A1 [x]**\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := File(p, Options{Project: "cli"})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Record != "0099" {
		t.Fatalf("record %q", doc.Record)
	}
	e := element(t, doc, "cli/0099:A1")
	if e.Section != "cli/0099:§critical-assumptions" {
		t.Errorf("section %s", e.Section)
	}
	for _, n := range doc.Outline {
		if !strings.HasPrefix(n.ID, "cli/0099:") || (n.Parent != "" && !strings.HasPrefix(n.Parent, "cli/0099:")) {
			t.Errorf("node not re-keyed: %+v", n)
		}
	}
}

// TestDeterministic: the same bytes project to the same JSON.
func TestDeterministic(t *testing.T) {
	raw := fixture(t, "current-shape.md")
	a, _ := json.Marshal(Bytes(raw, Options{Project: "cli"}))
	b, _ := json.Marshal(Bytes(raw, Options{Project: "cli"}))
	if string(a) != string(b) {
		t.Fatal("projection is not byte-deterministic")
	}
	if !strings.Contains(string(a), `"schema":"`+SchemaVersion+`"`) {
		t.Error("envelope lacks the schema version")
	}
}

// TestOpenJointDecisionsReadsTheFormNotTheProse: a record that has
// FINISHED answering its joint decisions lists them in its qualifier, so
// scraping `JD-\d+` out of the text reports open questions on a record
// that has none — 12 false positives over 3 records on the corpus that
// prompted this. Only the routing form means one is open.
func TestOpenJointDecisionsReadsTheFormNotTheProse(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "the routing form is the only open one",
			raw:  "Final [joint decision → JDR 0001 §JD-18: conforming-view enforcer]",
			want: []string{"JD-18"},
		},
		{
			name: "several on one routing qualifier",
			raw:  "Final [joint decision → JDR 0001 §JD-9, §JD-19, §JD-20: owned-state assembly]",
			want: []string{"JD-9", "JD-19", "JD-20"},
		},
		{
			name: "answered prose names them and opens nothing",
			raw: "Final [all joint decisions answered 2026-08-24 — JDR 0001 §D8 (§JD-9), " +
				"§D9 (§JD-19), §D11 (§JD-20). No joint decision is open against this RDR.]",
			want: nil,
		},
		{
			name: "a plain status opens nothing",
			raw:  "Implemented (`main` c1926e1)",
			want: nil,
		},
		{
			name: "a JD named in a revised-from qualifier is not open",
			raw:  "Draft [revised from Final; re-verify A2 — §JD-18 moved]",
			want: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := lifecycleStatus(tc.raw).OpenJointDecisions
			if len(got) != len(tc.want) {
				t.Fatalf("open = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("open[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestTableRowsAndJointChecksAreElements: a Testing Strategy written as a
// table is the same list in another shape, and a `Joint-check:` line is
// data — the open/ruled fact was being scraped out of prose by grep,
// truncated by `| head`, and misread.
func TestTableRowsAndJointChecksAreElements(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0009: Rows

## Metadata

- **Date**: 2026-08-01
- **Status**: Draft
- **Profile**: standard

## Validation

### Testing Strategy

| Test | Asserts |
| --- | --- |
| T-6 | an engine-scoped counter is zero |
| **T-34** | report-only agrees with emitting |

## Finalization Gate

Joint-check: fired → 0113 (home: cli/0113 §Normative Contracts) — shared literals
Joint-check: fired → 0106, 0110, cli/0142 (home: OPEN) — the retention seam
Joint-check: clear
`), Options{})
	var s, jc []Element
	for _, e := range doc.Elements {
		switch e.Kind {
		case ident.Scenario:
			s = append(s, e)
		case ident.JointCheck:
			jc = append(jc, e)
		}
	}
	if len(s) != 2 || s[0].ID != "0009:S6" || s[1].ID != "0009:S34" || s[0].Derived {
		t.Fatalf("scenarios = %+v, want S6 and S34 keyed by their leads", s)
	}
	if len(jc) != 3 {
		t.Fatalf("joint checks = %d, want 3", len(jc))
	}
	if jc[0].ID != "0009:JC1" || jc[0].Joint.Verdict != "fired" || jc[0].Joint.Open || jc[0].Joint.Home != "cli/0113 §Normative Contracts" ||
		len(jc[0].Joint.Targets) != 1 || jc[0].Joint.Targets[0] != "0113" {
		t.Errorf("JC1 = %+v", jc[0].Joint)
	}
	if !jc[1].Joint.Open || len(jc[1].Joint.Targets) != 3 || jc[1].Joint.Targets[2] != "cli/0142" {
		t.Errorf("JC2 = %+v, want open with three targets", jc[1].Joint)
	}
	if jc[2].Joint.Verdict != "clear" || jc[2].Joint.Targets != nil || jc[2].Joint.Open {
		t.Errorf("JC3 = %+v, want clear", jc[2].Joint)
	}
	if jc[0].Section != "0009:§finalization-gate" {
		t.Errorf("JC1 section = %q", jc[0].Section)
	}
	if start, end, ok := doc.Select("0009:JC2"); !ok || start != end {
		t.Errorf("select JC2 = %d-%d %v, want one line", start, end, ok)
	}
}

// TestProjectedSectionsComeFromTheSidecar pins the one-source rule for
// "which section answers for which element kind". The map lives in
// rdr-template.toml [elements]; before this, four of those names were
// ALSO written as Go literals in extract(), so the table could be edited
// and the projector would keep reading the old section.
//
// It asserts the BINDING, not today's values: a record is synthesised
// with one uniquely-worded item under each section the sidecar names,
// and each kind must come back carrying its own section's item. That is
// what makes a retarget visible — point a kind at another section and
// the item it returns is the other section's, so the test fails. An
// assertion that merely walked a fixture's elements would pass
// vacuously for any kind the fixture happens not to exercise.
func TestProjectedSectionsComeFromTheSidecar(t *testing.T) {
	sections := model.ElementSections()
	if len(sections) == 0 {
		t.Fatal("the sidecar names no element sections; the projector would read nothing")
	}

	// Three kinds the sidecar names are not projected from a section's
	// LIST, so a labelled bullet is not how they are found: C is read
	// from the normative fences wherever they sit, and ALT and MVV key
	// off the heading itself. They still belong in [elements] — that is
	// what answers "does the template key this kind" — but the binding
	// this test pins is the list one.
	byHeadingOrFence := map[string]bool{
		string(ident.Contract): true, string(ident.Alternative): true, string(ident.MVV): true,
	}

	// One item per section, worded so the item names the section it sits
	// in — that is what turns a silent retarget into a visible one.
	var b strings.Builder
	b.WriteString("# Recommendation 0099: Sidecar binding\n\n## Metadata\n\n- **Status**: Draft\n")
	want := map[string]string{}
	for kind, name := range sections {
		if byHeadingOrFence[kind] {
			continue
		}
		// A scaffold section names itself with a placeholder
		// (`Alternative 1: [Name]`); an instance fills it in.
		heading := strings.Replace(name, "[Name]", "Only", 1)
		marker := "item of " + kind
		want[kind] = marker
		fmt.Fprintf(&b, "\n## %s\n\n- **%s**: body\n", heading, marker)
	}

	doc := Bytes([]byte(b.String()), Options{})
	got := map[string]string{}
	for _, e := range doc.Elements {
		if _, named := want[string(e.Kind)]; !named {
			continue
		}
		if _, seen := got[string(e.Kind)]; !seen {
			got[string(e.Kind)] = e.Label
		}
	}

	for kind, marker := range want {
		switch got[kind] {
		case "":
			t.Errorf("%s: the sidecar names section %q but nothing projected from it",
				kind, sections[kind])
		case marker:
		default:
			t.Errorf("%s: projected %q, want %q — the kind is reading a section "+
				"other than the %q the sidecar names",
				kind, got[kind], marker, sections[kind])
		}
	}
}

// TestBulletGateProjectsTheSameItems pins the second shape a Finalization
// Gate is written in. Two of the oldest records answer it as a labelled
// list rather than as sub-headings; reading only sub-headings projected
// nothing for them, and because the bullets WERE classified, coverage
// reported no unclassified lines — the responses were dropped in silence,
// which is the one thing the projector is not allowed to do.
//
// The keys must come from TEMPLATE.md's own `[Gate key: …]` markers, not
// from the bullet's wording, because that is what peers cite. A label
// naming no declared item still projects, derived: it is a real response
// the record wrote.
func TestBulletGateProjectsTheSameItems(t *testing.T) {
	doc := Bytes([]byte(`# Recommendation 0099: Bullet gate

## Metadata

- **Status**: Final

## Finalization Gate

- **Contradiction Check**: none found.
- **Assumption Verification**: A1 stands.
- **Scope Verification**: the MVV is in scope.
- **API Verification**: the surface was re-read.
- **Cross-Cutting**: no project-wide policy changes.
- **Proportionality**: the design is no larger than the problem.
`), Options{})

	want := []struct {
		id      string
		derived bool
	}{
		{"0099:G-contradiction", false},
		{"0099:G-assumptions", false},
		{"0099:G-scope", false},
		{"0099:G-api-verification", true}, // the author's own item
		{"0099:G-cross-cutting", false},   // an abbreviated declared name
		{"0099:G-proportionality", false},
	}
	if n := doc.Counts.Elements[ident.Gate]; n != len(want) {
		t.Fatalf("bullet gate projected %d G elements, want %d", n, len(want))
	}
	for _, w := range want {
		e := element(t, doc, w.id)
		if e.Derived != w.derived {
			t.Errorf("%s: derived = %v, want %v", w.id, e.Derived, w.derived)
		}
		if e.LineStart == 0 || e.LineEnd < e.LineStart {
			t.Errorf("%s: names lines %d-%d, want a real range", w.id, e.LineStart, e.LineEnd)
		}
	}

	// The two shapes are alternatives, not additives: a gate with
	// sub-headings must not also read its prose as items.
	sub := Bytes(fixture(t, "legacy-shape.md"), Options{})
	if n := sub.Counts.Elements[ident.Gate]; n != 5 {
		t.Errorf("sub-heading gate projected %d G elements, want 5 — the bullet "+
			"fallback fired on a gate that already had sub-sections", n)
	}
}

// TestBulletGateKeysAreNotGuessed pins the narrow match. A label is a
// declared item only when it IS the declared name or its leading words;
// anything looser would mint a gate key the record never answered, and a
// wrong key is worse than a derived one because peers cite these.
func TestBulletGateKeysAreNotGuessed(t *testing.T) {
	for _, tc := range []struct{ label, want string }{
		{"Contradiction Check", "contradiction"},
		{"contradiction check", "contradiction"}, // case is not the author's meaning
		{"Scope", "scope"},                       // a declared name's own lead
		{"Scoped Review", ""},                    // a longer word, not a lead
		{"Verification", ""},                     // a trailing word is not a lead
		{"", ""},
	} {
		if got := gateKeyForLabel(tc.label); got != tc.want {
			t.Errorf("gateKeyForLabel(%q) = %q, want %q", tc.label, got, tc.want)
		}
	}
}

// TestTemplateExampleIsNotAContract is the rule that a fence whose body
// is TEMPLATE.md's OWN example is the template's words, not the author's.
//
// The projector reads every ```normative fence document-wide, which is
// right — a fence is a contract wherever it sits. But a seeded record
// carries the template's example verbatim until the section is authored,
// and counting it mints a C1 nobody wrote. Measured on the reference
// corpus before this rule: 13 records projected it, 11 of them in-flight
// Drafts whose `contracts` count therefore read 1 with a true count of 0.
//
// The exclusion is DERIVED from TEMPLATE.md, never listed here: change
// the template's example and this test's fixture must change with it,
// which is the coupling that keeps the rule from going stale.
func TestTemplateExampleIsNotAContract(t *testing.T) {
	example := ""
	for body := range model.TemplateNormativeBodies() {
		example = body
		break
	}
	if example == "" {
		t.Fatal("TEMPLATE.md declares no normative example; this rule has no subject")
	}

	d := Bytes([]byte("# Recommendation 0010: Frame header\n\n"+
		"## Normative Contracts\n\n"+
		"**C1**\n\n```normative\n"+example+"\n```\n"), Options{})
	if n := d.Counts.Elements["C"]; n != 0 {
		t.Errorf("the template's own example projected %d contract(s); it is the template's words, not the author's", n)
	}

	// The warning is the finding: silence would be the same defect one
	// layer down, and a bullet the projector cannot place is a warning
	// rather than a drop.
	var warned bool
	for _, w := range d.Warnings {
		if w.Code == "contract:template-example" {
			warned = true
		}
	}
	if !warned {
		t.Error("the skipped fence produced no warning; a dropped element must never be silent")
	}
}

// TestAuthoredContractStillProjects is the other half: the rule must not
// swallow a real contract. A fence the author wrote is a contract however
// much it resembles the template's shape.
func TestAuthoredContractStillProjects(t *testing.T) {
	d := Bytes([]byte("# Recommendation 0010: Frame header\n\n"+
		"## Normative Contracts\n\n"+
		"**C1**\n\n```normative\nfunc Resolve(name string) (Relation, error)\n```\n"), Options{})
	if n := d.Counts.Elements["C"]; n != 1 {
		t.Errorf("an authored contract projected %d elements, want 1", n)
	}
}

// TestAlignedClauseListMintsWideLabels: in a column-aligned clause list
// the two-space separator collapses to one once the label's digits reach
// two (`R-9  determinism:` but `R-10 cascade guard.`); the wide label is
// still a definition — accepted because a same-prefix sibling in the
// fence defined at two spaces — so R-10 and R-11 mint and R-9's range
// ends where R-10 begins. A wide label whose prefix has no aligned
// sibling in its fence (Q-12, E-12) stays prose, and a single-digit
// label opening a prose line (X-2 is …) never becomes a definition.
func TestAlignedClauseListMintsWideLabels(t *testing.T) {
	raw := []byte(`# Recommendation 0031: Ledger

## Metadata

- **Status**: Draft

#### Normative Contracts

**C1**

` + "```normative" + `
R-1  admission := the gate reads each row once.
R-2  ordering: rows land in ledger order.
R-3  replay guard.
R-4  idempotence.
R-5  checksum.
R-6  fan-out cap.
R-7  retry budget.
R-8  quiet close.
R-9  determinism: two runs over one ledger agree
     byte for byte; R-2 is what the agreement rests on.
R-10 cascade guard. A refused row refuses its
     dependants in the same pass.
Q-12 stays prose despite its width — no aligned Q sibling.
R-11 audit line. Every refusal writes one line.
X-2 is refuted by the audit line, not minted by the scanner.
` + "```" + `

**C2**

` + "```normative" + `
E-1: the counter never skips.
E-12 references E-1, but this fence's E style is the colon, not columns.
` + "```" + `
`)
	doc := Bytes(raw, Options{})
	want := map[string][2]int{ // id → line range
		"0031:R-9":  {20, 21},
		"0031:R-10": {22, 24}, // Q-12 is prose inside R-10's span
		"0031:R-11": {25, 26}, // X-2 is prose inside R-11's span
		"0031:E-1":  {32, 33}, // E-12 is prose inside E-1's span
	}
	for id, r := range want {
		e := element(t, doc, id)
		if e.Kind != ident.Clause || e.LineStart != r[0] || e.LineEnd != r[1] {
			t.Errorf("%s spans %d-%d, want %d-%d", id, e.LineStart, e.LineEnd, r[0], r[1])
		}
	}
	for _, e := range doc.Elements {
		if e.Kind == ident.Clause && (e.Key == "Q-12" || e.Key == "E-12" || e.Key == "X-2") {
			t.Errorf("prose opening with a label was minted: %+v", e)
		}
	}
	if n := doc.Counts.Elements[ident.Clause]; n != 12 { // R-1..R-11 and E-1
		t.Errorf("clause count = %d, want 12", n)
	}
	if len(doc.Warnings) != 0 {
		t.Errorf("warnings = %+v, want none", doc.Warnings)
	}
	if s, e, ok := doc.Select("0031:R-10"); !ok || s != 22 || e != 24 {
		t.Errorf("Select(R-10) = %d-%d %v", s, e, ok)
	}
}

// TestAlignedClauseListMintsSubLetteredLabels: the collapsed gap is not
// only a second digit. A sub-lettered label (`H-2a`) is one column wider
// than its aligned sibling (`H-1  `) and closes the gap the same way, so
// it is a definition under the same evidence — a same-prefix sibling at
// two spaces in the fence. Without that sibling a line opening `REQ-2a
// requires …` stays the wrapped prose it is (cli/0105's shape), and the
// sub-lettered clause is addressable from a sibling record's citation.
func TestAlignedClauseListMintsSubLetteredLabels(t *testing.T) {
	raw := []byte(`# Recommendation 0032: Carrier

## Metadata

- **Status**: Draft

#### Normative Contracts

**C1**

` + "```normative" + `
H-1  header not the fixed prefix → refused
H-2a disposition=append ∧ zero records → warns and skips
H-2b disposition ∈ {restate, replace} ∧ zero records → refused
     at scan only
H-3  body=none ∧ a fence is present → refused
` + "```" + `

**C2**

` + "```normative" + `
REQ-1: the carrier is one file.
REQ-2a requires the skip carry reason, which REQ-1 leaves
to the producer.
` + "```" + `
`)
	doc := Bytes(raw, Options{})
	want := map[string][2]int{
		"0032:H-1":   {12, 12},
		"0032:H-2a":  {13, 13},
		"0032:H-2b":  {14, 15},
		"0032:H-3":   {16, 16},
		"0032:REQ-1": {22, 24}, // REQ-2a is prose inside REQ-1's span
	}
	for id, r := range want {
		e := element(t, doc, id)
		if e.Kind != ident.Clause || e.LineStart != r[0] || e.LineEnd != r[1] {
			t.Errorf("%s spans %d-%d, want %d-%d", id, e.LineStart, e.LineEnd, r[0], r[1])
		}
	}
	for _, e := range doc.Elements {
		if e.Kind == ident.Clause && e.Key == "REQ-2a" {
			t.Errorf("prose opening with a sub-lettered label was minted: %+v", e)
		}
	}
	if n := doc.Counts.Elements[ident.Clause]; n != 5 {
		t.Errorf("clause count = %d, want 5", n)
	}
	if len(doc.Warnings) != 0 {
		t.Errorf("warnings = %+v, want none", doc.Warnings)
	}
}
