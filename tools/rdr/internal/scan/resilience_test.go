package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
)

// The resilience contract, as tests (README §The resilience contract).
//
// Every fixture under testdata/ — the four epochs and the variants, one
// per known parser failure — is held to the zero-silent-drop property,
// and each variant fixture pins a hand tally of what the scanner must
// read out of it.

// allFixtures lists every synthetic fixture, epochs and variants.
func allFixtures(t *testing.T) []string {
	t.Helper()
	var out []string
	root := filepath.Join("..", "..", "testdata")
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") && d.Name() != "README.md" {
			rel, _ := filepath.Rel(root, p)
			out = append(out, rel)
		}
		return err
	})
	if err != nil || len(out) < 9 {
		t.Fatalf("walking testdata: %v (%d fixtures)", err, len(out))
	}
	return out
}

func variant(t *testing.T, name string) *Document {
	t.Helper()
	return Bytes(fixture(t, filepath.Join("variants", name)), Options{})
}

// TestZeroSilentDrop is the property itself, on every fixture:
//
//   - every non-blank line lies inside the outline, and the innermost
//     node containing it is unique (nodes nest; siblings never overlap);
//   - every labelled bullet outside a fence is an element's lead line, a
//     metadata field, a field of the element it sits in, a section field,
//     or inside a warning's range — never nothing;
//   - counts.fields adds up to exactly the fields recorded.
func TestZeroSilentDrop(t *testing.T) {
	for _, f := range allFixtures(t) {
		doc := Bytes(fixture(t, f), Options{})
		if doc.Record == "" {
			t.Errorf("%s: no record number", f)
			continue
		}

		// Outline: covered, and the innermost node is unique.
		innermost := make([]int, doc.Lines+1)
		for _, n := range doc.Outline {
			for i := n.LineStart; i <= n.LineEnd; i++ {
				innermost[i]++
			}
		}
		for i := 1; i <= doc.Lines; i++ {
			if strings.TrimSpace(doc.Line(i)) == "" {
				continue
			}
			if innermost[i] == 0 {
				t.Errorf("%s: line %d is outside every outline node", f, i)
			}
		}
		for i, n := range doc.Outline {
			for _, m := range doc.Outline[i+1:] {
				overlap := m.LineStart <= n.LineEnd && n.LineStart <= m.LineEnd
				nested := (m.LineStart >= n.LineStart && m.LineEnd <= n.LineEnd) || (n.LineStart >= m.LineStart && n.LineEnd <= m.LineEnd)
				if overlap && !nested {
					t.Errorf("%s: %s and %s overlap without nesting", f, n.ID, m.ID)
				}
			}
		}

		// Labelled bullets: each one is accounted for somewhere.
		accounted := map[int]string{}
		warned := make([]bool, doc.Lines+1)
		for _, w := range doc.Warnings {
			for i := w.LineStart; i <= w.LineEnd && i <= doc.Lines; i++ {
				warned[i] = true
			}
		}
		for _, e := range doc.Elements {
			accounted[e.LineStart] = "element " + e.ID
			for _, fl := range e.Fields {
				accounted[fl.LineStart] = "field of " + e.ID
				if fl.Element != e.ID {
					t.Errorf("%s: field %q at %d names element %q, sits in %s", f, fl.Label, fl.LineStart, fl.Element, e.ID)
				}
			}
		}
		for _, fl := range doc.Metadata {
			accounted[fl.LineStart] = "metadata"
		}
		for _, fl := range doc.Fields {
			accounted[fl.LineStart] = "section field"
		}
		total := len(doc.Metadata) + len(doc.Fields)
		for _, e := range doc.Elements {
			total += len(e.Fields)
		}
		for i := 1; i <= doc.Lines; i++ {
			if doc.fenced[i-1] || !labelledBullet.MatchString(doc.Line(i)) {
				continue
			}
			if _, ok := accounted[i]; !ok && !warned[i] {
				t.Errorf("%s: labelled bullet at line %d is dropped: %q", f, i, doc.Line(i))
			}
		}
		counted := 0
		for _, n := range doc.Counts.Fields {
			counted += n
		}
		if counted != total {
			t.Errorf("%s: counts.fields sums to %d, %d fields recorded", f, counted, total)
		}

		// Coverage: unclassified is exactly the warned non-blank lines.
		want := 0
		for i := 1; i <= doc.Lines; i++ {
			if warned[i] && strings.TrimSpace(doc.Line(i)) != "" {
				want++
			}
		}
		if doc.Coverage.Unclassified != want {
			t.Errorf("%s: coverage.unclassified = %d, warned non-blank lines = %d", f, doc.Coverage.Unclassified, want)
		}
	}
}

// TestConformantFixturesHaveFullCoverage: a record that conforms to its
// epoch has no warnings and an unclassified rate of exactly zero. A
// warning on such a record is a projector bug to fix with a fixture,
// never a reason to edit the record.
func TestConformantFixturesHaveFullCoverage(t *testing.T) {
	for _, f := range []string{"epoch-a.md", "epoch-b.md", "epoch-c.md", "epoch-d.md",
		"variants/heading-level.md", "variants/label-variants.md", "variants/status-parenthetical.md",
		"variants/addressable-text.md"} {
		doc := Bytes(fixture(t, f), Options{})
		if len(doc.Warnings) != 0 || doc.Coverage.Unclassified != 0 || doc.Coverage.Rate != 0 {
			t.Errorf("%s: %d warnings, coverage %+v", f, len(doc.Warnings), doc.Coverage)
		}
		if doc.Coverage.Lines == 0 {
			t.Errorf("%s: coverage counted no lines", f)
		}
	}
}

// assumptionField returns one canonical field of an assumption.
func assumptionField(t *testing.T, doc *Document, id, canonical string) Field {
	t.Helper()
	e := element(t, doc, id)
	for _, f := range e.Fields {
		if f.Canonical == canonical {
			return f
		}
	}
	t.Fatalf("%s has no %s field; fields %+v", id, canonical, e.Fields)
	return Field{}
}

func metadataField(t *testing.T, doc *Document, canonical string) Field {
	t.Helper()
	for _, f := range doc.Metadata {
		if f.Canonical == canonical {
			return f
		}
	}
	t.Fatalf("%s has no %s metadata field; have %+v", doc.Record, canonical, doc.Metadata)
	return Field{}
}

// TestHeadingLevelVariant: Critical Assumptions written at `###` is the
// same section, level-variant, and every assumption under it is read.
func TestHeadingLevelVariant(t *testing.T) {
	doc := variant(t, "heading-level.md")
	var ca *Node
	for i := range doc.Outline {
		if doc.Outline[i].Canonical == "Critical Assumptions" {
			ca = &doc.Outline[i]
		}
	}
	if ca == nil || ca.Level != 3 || ca.Match != "level-variant" || ca.ID != "0101:§critical-assumptions" {
		t.Errorf("Critical Assumptions node = %+v", ca)
	}
	if n := doc.Counts.Elements[ident.Assumption]; n != 2 {
		t.Errorf("assumptions = %d, want 2", n)
	}
	st := metadataField(t, doc, "Status").Status
	if st == nil || st.Value != "Final" || st.Form != "joint-decision" || !strings.HasPrefix(st.Qualifier, "joint decision → 0004") {
		t.Errorf("status = %+v", st)
	}
}

// TestLabelVariants: Evidence Record labels extended into clauses match
// their field by prefix; a colon inside the bold is still a label; an
// author's own sub-field is recorded as author, not dropped, not warned.
func TestLabelVariants(t *testing.T) {
	doc := variant(t, "label-variants.md")
	cases := []struct{ id, canonical, label, match string }{
		{"0102:A1", "Status", "Status", "exact"}, // written `**Status:**`
		{"0102:A1", "Evidence", "Evidence — the two accessor paths", "prefix"},
		{"0102:A1", "If wrong", "If wrong (a refusal is owed)", "prefix"},
		{"0102:A2", "Evidence", "Evidence (plan)", "prefix"},
		{"0102:A2", "If wrong", "If wrong", "exact"},
	}
	for _, c := range cases {
		f := assumptionField(t, doc, c.id, c.canonical)
		if f.Label != c.label || f.Match != c.match {
			t.Errorf("%s %s: label %q match %s, want %q %s", c.id, c.canonical, f.Label, f.Match, c.label, c.match)
		}
	}
	a1 := element(t, doc, "0102:A1")
	var note *Field
	for i := range a1.Fields {
		if a1.Fields[i].Label == "Note" {
			note = &a1.Fields[i]
		}
	}
	if note == nil || note.Match != "author" || note.Canonical != "" {
		t.Errorf("author Note field = %+v", note)
	}
	if m := assumptionField(t, doc, "0102:A1", "Method").Method; m == nil || len(m.Members) != 1 || m.Members[0] != "Source Search" || len(m.OffVocabulary) != 0 {
		t.Errorf("glossed method = %+v", m)
	}
	if m := assumptionField(t, doc, "0102:A2", "Method").Method; m == nil || strings.Join(m.Members, "+") != "Docs Only+Spike" {
		t.Errorf("compound method = %+v", m)
	}
	if doc.Counts.Fields["prefix"] != 3 || doc.Counts.Fields["author"] != 1 || len(doc.Warnings) != 0 {
		t.Errorf("counts %v warnings %v", doc.Counts.Fields, doc.Warnings)
	}
}

// TestStatusForms: the lifecycle Status with a parenthetical qualifier
// normalises to {value, qualifier, raw}; the Evidence Record's Status is
// normalised the same way across the corpus's forms, and the template
// legend left in place is a placeholder, never `Verified`.
func TestStatusForms(t *testing.T) {
	doc := variant(t, "status-parenthetical.md")
	st := metadataField(t, doc, "Status").Status
	if st == nil || st.Value != "Implemented" || st.Qualifier != "`main` a1b2c3d" || st.Form != "parenthetical" || st.Tier != "canonical" {
		t.Errorf("lifecycle status = %+v", st)
	}
	cases := []struct {
		id, value, qualifier, tier string
		placeholder                bool
	}{
		{"0103:A1", "Verified", "", "canonical", false}, // `**Verified**`
		{"0103:A2", "Refuted", "one test pinned the old order", "observed-accepted", false},
		{"0103:A3", "Verified", "as narrowed — total over entries with an expiry; entries without one sort last, by insertion.", "canonical", false},
		{"0103:A4", "", "", "off-vocabulary", true},
		{"0103:A5", "Resolved", "settled by the Naming decision below", "observed-accepted", false},
	}
	for _, c := range cases {
		s := assumptionField(t, doc, c.id, "Status").Status
		if s == nil || s.Value != c.value || s.Qualifier != c.qualifier || s.Tier != c.tier || s.Placeholder != c.placeholder {
			t.Errorf("%s status = %+v, want %+v", c.id, s, c)
		}
	}
	if m := assumptionField(t, doc, "0103:A2", "Method").Method; m == nil || strings.Join(m.Members, "+") != "Source Search+Spike" {
		t.Errorf("compound method = %+v", m)
	}
}

// TestWrappedMetadata: wrapped values are joined; a template guidance
// comment ends the value; the em-dash Status form is read; a legacy field
// label maps to its canonical; an author's field the alias table knows is
// recognized-unmapped; a label nobody has written warns — and is still
// recorded.
func TestWrappedMetadata(t *testing.T) {
	doc := variant(t, "wrapped-metadata.md")
	st := metadataField(t, doc, "Status")
	if st.LineStart != 9 || st.LineEnd != 12 {
		t.Errorf("Status spans %d-%d, want 9-12 (the guidance comment ends it)", st.LineStart, st.LineEnd)
	}
	// The label is canonical since TEMPLATE.md gained the Deferred
	// spelling; the FORM is still the legacy undelimited em-dash clause
	// this fixture exists to pin, not the bracketed `[revisit when …]`
	// a new record writes.
	if st.Status == nil || st.Status.Value != "Deferred" || st.Status.Form != "dash" || st.Status.Tier != "canonical" {
		t.Errorf("status = %+v", st.Status)
	}
	pred := metadataField(t, doc, "Predecessors")
	if pred.LineStart != 19 || pred.LineEnd != 21 || strings.Count(pred.Value, "000") != 3 {
		t.Errorf("Predecessors = %+v", pred)
	}
	seam := metadataField(t, doc, "Seam Lineage")
	if seam.LineEnd != 28 || !strings.Contains(seam.Value, "Accretion disposition") || strings.Contains(seam.Value, "Disposition owner") {
		t.Errorf("Seam Lineage = %+v (a nested bullet is not part of the value)", seam)
	}
	if rel := metadataField(t, doc, "Related Issues"); rel.Label != "Related" || rel.Match != "legacy-alias" {
		t.Errorf("Related = %+v", rel)
	}
	var scope, refby *Field
	for i := range doc.Metadata {
		switch doc.Metadata[i].Label {
		case "Release scope":
			scope = &doc.Metadata[i]
		case "Referenced by":
			refby = &doc.Metadata[i]
		}
	}
	if scope == nil || scope.Match != "recognized-unmapped" {
		t.Errorf("Release scope = %+v", scope)
	}
	if refby == nil || refby.Match != "author" {
		t.Errorf("Referenced by = %+v", refby)
	}
	if len(doc.Warnings) != 1 || doc.Warnings[0].Code != "field:unknown-to-template" || doc.Warnings[0].LineStart != 31 {
		t.Errorf("warnings = %+v", doc.Warnings)
	}
	if doc.Coverage.Unclassified != 1 {
		t.Errorf("coverage = %+v", doc.Coverage)
	}
	// The nested labelled bullet under Seam Lineage is a section field.
	found := false
	for _, f := range doc.Fields {
		if f.Label == "Disposition owner" && f.Match == "author" && f.Section == "0104:§metadata" {
			found = true
		}
	}
	if !found {
		t.Errorf("nested metadata bullet not recorded as a section field: %+v", doc.Fields)
	}
}

// TestAuthorStructure: an author's sub-headings inside a template section
// are that section's content, not foreign sections; a foreign top-level
// section warns once, with its own sub-headings inside the range;
// labelled bullets the section's prose names are observed, the author's
// own are recorded as author.
func TestAuthorStructure(t *testing.T) {
	doc := variant(t, "author-structure.md")
	byHeading := map[string]Node{}
	for _, n := range doc.Outline {
		byHeading[n.Heading] = n
	}
	for _, h := range []string{"Why not a vendored copy", "The maintenance argument", "The trust argument", "Why not a wrapper"} {
		if n := byHeading[h]; n.Match != "author-subsection" || !n.Derived {
			t.Errorf("%q = %+v, want author-subsection", h, n)
		}
	}
	if n := byHeading["Appendix A — Reader Catalog"]; n.Match != "unknown" {
		t.Errorf("Appendix = %+v", n)
	}
	for _, h := range []string{"Reader 1", "Reader 2"} {
		if n := byHeading[h]; n.Match != "unknown" {
			t.Errorf("%q = %+v, want unknown (inside a foreign section)", h, n)
		}
	}
	if len(doc.Warnings) != 1 || doc.Warnings[0].Code != "section:unknown-to-template" || !strings.Contains(doc.Warnings[0].Message, "Appendix A") {
		t.Errorf("warnings = %+v, want exactly the Appendix", doc.Warnings)
	}
	if doc.Coverage.Unclassified != 5 {
		t.Errorf("coverage = %+v, want the Appendix's 5 non-blank lines", doc.Coverage)
	}
	want := map[string]string{ // label → match
		"Positive": "exact", "Negative": "exact", "Versioning": "exact", "Determinism": "exact",
		"Forward-compat": "author", "Category": "author",
	}
	got := map[string]string{}
	risks := map[string]string{}
	for _, f := range doc.Fields {
		got[f.Label] = f.Match
		if f.Label == "Risk" {
			risks[f.Section] = f.Match
		}
	}
	for l, m := range want {
		if got[l] != m {
			t.Errorf("field %q = %q, want %q", l, got[l], m)
		}
	}
	if risks["0105:§risks-and-mitigations"] != "exact" || risks["0105:§step-1-add-the-pin"] != "author" {
		t.Errorf("Risk fields = %v: template-drawn in its section, the author's under a step", risks)
	}
	// Failure Modes bullets are F elements, so their labels are element
	// labels, not fields.
	if n := doc.Counts.Elements[ident.Failure]; n != 4 {
		t.Errorf("failure modes = %d, want 4", n)
	}
	st := metadataField(t, doc, "Status").Status
	if st == nil || st.Value != "Rejected" || st.Tier != "observed-accepted" || st.Form != "parenthetical" {
		t.Errorf("status = %+v", st)
	}
}

// TestFieldsSurviveFilenameRecord: when the record number comes from the
// filename, every field's section and element IDs are re-derived with it.
func TestFieldsSurviveFilenameRecord(t *testing.T) {
	raw := fixture(t, filepath.Join("variants", "label-variants.md"))
	raw = []byte(strings.Replace(string(raw), "# Recommendation 0102: ", "# ", 1))
	dir := t.TempDir()
	p := filepath.Join(dir, "0102-cache-entry-expiry-clock.md")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := File(p, Options{Project: "cache"})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Record != "0102" {
		t.Fatalf("record %q", doc.Record)
	}
	for _, f := range doc.Metadata {
		if f.Section != "cache/0102:§metadata" {
			t.Errorf("metadata field %q section %q", f.Label, f.Section)
		}
	}
	e := element(t, doc, "cache/0102:A1")
	for _, f := range e.Fields {
		if f.Element != "cache/0102:A1" || f.Section != "cache/0102:§critical-assumptions" {
			t.Errorf("field %q element %q section %q", f.Label, f.Element, f.Section)
		}
	}
}

// TestAddressableTextVariant pins the two addressable-text paths against
// the fixture: a bold paragraph lead a `§` citation can name, and a
// `### Decisions` heading whose author-numbered bullets are D-elements.
//
// Both are things the corpus writes that an exact-literal reader projects
// as NOTHING — the lead is not a heading, the section is not the
// template's name — so a citation of either resolves against nothing
// while the text it names sits plainly in the file.
func TestAddressableTextVariant(t *testing.T) {
	doc := variant(t, "addressable-text.md")

	// Bold paragraph leads are addressable; emphasis inside a paragraph
	// is not, and a contract's own label is not a second identity for it.
	got := map[string]bool{}
	for _, a := range doc.Anchors {
		got[a.ID] = true
	}
	for _, want := range []string{
		"0102:§the-alignment-values",
		"0102:§pad-bytes-are-zero-filled-and-never-inspected",
	} {
		if !got[want] {
			t.Errorf("bold lead %s is not addressable; anchors are %v", want, doc.Anchors)
		}
	}
	for _, never := range []string{"0102:§bold-run-written", "0102:§c1"} {
		if got[never] {
			t.Errorf("%s was minted as an anchor; it names no paragraph lead", never)
		}
	}
	// An anchor round-trips to the bytes it names, like every other ID.
	for _, a := range doc.Anchors {
		start, end, ok := doc.Select(a.ID)
		if !ok || start != a.Line || end != a.Line {
			t.Errorf("%s selects (%d,%d,%v), want its own line %d", a.ID, start, end, ok, a.Line)
		}
	}

	// `### Decisions` with numbered bullets: three D-elements, keyed as
	// written, and the gap at D3..D5 is the author's numbering kept.
	var keys []string
	for _, e := range doc.Elements {
		if e.Kind == ident.Decision {
			keys = append(keys, e.ID)
			if e.Derived {
				t.Errorf("%s is author-numbered but reported derived", e.ID)
			}
		}
	}
	want := []string{"0102:D-1", "0102:D-2", "0102:D-6"}
	if len(keys) != len(want) {
		t.Fatalf("decisions = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("decision %d = %s, want %s", i, keys[i], want[i])
		}
	}
}
