package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// kinds tallies elements per kind, and derived per kind, as "n/derived".
func kinds(doc *Document) map[ident.Kind]string {
	out := map[ident.Kind]string{}
	for _, k := range ident.Kinds {
		if k == ident.Section {
			continue
		}
		if n := doc.Counts.Elements[k]; n > 0 {
			out[k] = strings.Join([]string{itoa(n), itoa(doc.Counts.Derived[k])}, "/")
		}
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

// TestFixtureTallies pins hand-counted element sets per epoch fixture:
// which kinds appear, how many, and how many are derived.
func TestFixtureTallies(t *testing.T) {
	cases := []struct {
		file   string
		record string
		epoch  string
		want   map[ident.Kind]string // kind → "count/derived"
		ids    []string              // a few IDs that must exist
	}{
		{"epoch-a.md", "0001", "A",
			map[ident.Kind]string{"ALT": "1/0", "BR": "1/1", "S": "1/0", "MVV": "1/0", "G": "5/0"},
			[]string{"0001:G-contradiction", "0001:G-proportionality", "0001:ALT1"}},
		{"epoch-b.md", "0002", "B",
			map[ident.Kind]string{"A": "2/0", "D": "2/0", "ALT": "1/0", "BR": "1/1", "S": "1/0", "MVV": "1/0", "G": "5/0"},
			[]string{"0002:A1", "0002:A2", "0002:D-identity", "0002:D-selection-predicate"}},
		{"epoch-c.md", "0003", "C",
			map[ident.Kind]string{"A": "2/0", "C": "1/1", "BR": "1/1", "S": "1/0", "MVV": "1/0"},
			[]string{"0003:C1", "0003:BR1"}},
		{"epoch-d.md", "0004", "D",
			map[ident.Kind]string{"A": "3/0", "C": "1/1", "D": "2/0", "ALT": "2/0", "BR": "1/1", "S": "1/0", "MVV": "1/0"},
			[]string{"0004:A1", "0004:A3", "0004:C1", "0004:D-wire-byte-format", "0004:D-naming", "0004:ALT2", "0004:MVV", "0004:S1"}},
	}
	for _, c := range cases {
		doc := Bytes(fixture(t, c.file), Options{})
		if doc.Record != c.record || doc.Epoch != c.epoch {
			t.Errorf("%s: record %s epoch %s, want %s %s", c.file, doc.Record, doc.Epoch, c.record, c.epoch)
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
	if n := Bytes(fixture(t, "epoch-c.md"), Options{}).Counts.Elements[ident.Gate]; n != 0 {
		t.Errorf("epoch C (gate pointer) has %d G elements, want 0", n)
	}
	if n := Bytes(fixture(t, "epoch-a.md"), Options{}).Counts.Elements[ident.Gate]; n != 5 {
		t.Errorf("epoch A (inlined gate) has %d G elements, want 5", n)
	}
}

// TestLineRangesRoundTrip: every element's ID selects the bytes it names,
// and those bytes start where the element's grammar says they start.
func TestLineRangesRoundTrip(t *testing.T) {
	for _, f := range []string{"epoch-a.md", "epoch-b.md", "epoch-c.md", "epoch-d.md"} {
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
	for _, f := range []string{"epoch-a.md", "epoch-d.md"} {
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
	raw := fixture(t, "epoch-d.md")
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
	raw := fixture(t, "epoch-d.md")
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
	raw := fixture(t, "epoch-d.md")
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
	raw := fixture(t, "epoch-d.md")
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
		"0013:§premortem":            "legacy-alias",
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
	if doc.Epoch != "C" {
		t.Errorf("epoch %s, want C (gate pointer, CA at ###)", doc.Epoch)
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
	raw := fixture(t, "epoch-d.md")
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
