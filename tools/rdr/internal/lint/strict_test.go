package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// Apply is the reference applier, and it lives in the test rather than in
// the package on purpose: this tool writes no record, and the real
// applier is a throwaway script. What the package owes is a patch set
// that CAN be applied correctly, and this is the definition of correctly
// that the acceptance test holds it to.
//
// Bottom-up by LineStart, so a patch's line numbers are still valid when
// it is applied: every earlier patch in the file is still unapplied and
// nothing below it has moved.
func apply(lines []string, patches []*Patch) []string {
	// A patch may be shared by several findings — every citation on one
	// line names the same repair — so the set is deduplicated by identity
	// before anything is applied. An applier that skipped this would
	// rewrite the line once per finding.
	seen := map[*Patch]bool{}
	var sorted []*Patch
	for _, p := range patches {
		if seen[p] {
			continue
		}
		seen[p] = true
		sorted = append(sorted, p)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].LineStart != sorted[j].LineStart {
			return sorted[i].LineStart > sorted[j].LineStart
		}
		return sorted[i].LineEnd > sorted[j].LineEnd
	})
	out := append([]string(nil), lines...)
	for _, p := range sorted {
		text := strings.Split(p.Text, "\n")
		switch p.Op {
		case OpReplace:
			out = append(out[:p.LineStart-1], append(text, out[p.LineEnd:]...)...)
		case OpPrepend:
			out = append(out[:p.LineStart-1], append(text, out[p.LineStart-1:]...)...)
		case OpInsert:
			out = append(out[:p.LineEnd], append(text, out[p.LineEnd:]...)...)
		}
	}
	return out
}

func patchesOf(r Report) []*Patch {
	var out []*Patch
	for _, f := range r.Findings {
		if f.Patch != nil {
			out = append(out, f.Patch)
		}
	}
	return out
}

// fixtures are the epoch fixtures plus the shape variants: between them
// they carry every element grammar the corpus writes.
func fixtures(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, g := range []string{
		filepath.Join("..", "..", "testdata", "epoch-*.md"),
		filepath.Join("..", "..", "testdata", "variants", "*.md"),
		filepath.Join("..", "..", "testdata", "lint", "*.md"),
	} {
		paths, err := filepath.Glob(g)
		if err != nil || len(paths) == 0 {
			t.Fatalf("no fixtures for %s: %v", g, err)
		}
		out = append(out, paths...)
	}
	return out
}

// graph is the projection facts a migration must not move: which elements
// exist, which sections exist, and what every edge resolves to.
//
// Line numbers and hashes are deliberately absent. A migration MOVES
// lines — that is what it is — and re-levelling a heading changes the
// bytes under it, so the hash of a section is expected to move. What must
// not move is identity: an id that resolved to an element before the
// patch resolves to the same element after it.
type graph struct {
	elements []string
	sections []string
	edges    []string
}

func graphOf(d *scan.Document) graph {
	var g graph
	for _, e := range d.Elements {
		g.elements = append(g.elements, e.ID)
	}
	for _, n := range d.Outline {
		g.sections = append(g.sections, n.ID)
	}
	for _, e := range d.Edges {
		resolved := "?"
		if e.Resolved != nil {
			resolved = "false"
			if *e.Resolved {
				resolved = "true"
			}
		}
		g.edges = append(g.edges, string(e.Kind)+" "+e.From+" -> "+e.To+" ["+resolved+"]")
	}
	sort.Strings(g.elements)
	sort.Strings(g.sections)
	sort.Strings(g.edges)
	return g
}

// scanAll scans a set of records and resolves them against each other,
// which is the state a graph comparison needs: an unresolved edge and an
// unchecked one are different facts.
func scanAll(t *testing.T, sources map[string][]byte) map[string]*scan.Document {
	t.Helper()
	var docs []*scan.Document
	var paths []string
	for p := range sources {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		d := scan.Bytes(sources[p], scan.Options{})
		d.Path = p
		docs = append(docs, d)
	}
	scan.NewResolver(docs, "").ResolveAll(docs)
	out := map[string]*scan.Document{}
	for _, d := range docs {
		out[d.Path] = d
	}
	return out
}

// TestStrictPatchesPreserveTheGraph is the acceptance case for the whole
// change, and the one subtle part of it.
//
// A migration is only safe if the ids survive it. Apply every patch
// strict proposes to every fixture, re-project the results, and require
// that the element ids, the section ids and every edge's resolution are
// exactly what they were. A label written by `label:missing` is the id
// the projector already derived, so writing it must be a no-op on the
// graph; a re-levelled heading keeps its canonical slug; a citation in
// the colon form resolves to the same target as the spaced form it
// replaced. If any of those were false, the patch set would be silently
// rewriting the corpus's references while claiming to preserve them.
func TestStrictPatchesPreserveTheGraph(t *testing.T) {
	paths := fixtures(t)
	before := map[string][]byte{}
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		before[p] = raw
	}

	pre := scanAll(t, before)
	after := map[string][]byte{}
	patched := 0
	for _, p := range paths {
		r := Run(pre[p], Options{Strict: true, Corpus: docsOf(pre)})
		ps := patchesOf(r)
		patched += len(ps)
		lines := strings.Split(string(before[p]), "\n")
		after[p] = []byte(strings.Join(apply(lines, ps), "\n"))
	}
	if patched == 0 {
		t.Fatal("no patches proposed over the fixtures; the test proves nothing")
	}
	t.Logf("applied %d patches over %d fixtures", patched, len(paths))

	post := scanAll(t, after)
	for _, p := range paths {
		a, b := graphOf(pre[p]), graphOf(post[p])
		if !reflect.DeepEqual(a.elements, b.elements) {
			t.Errorf("%s: element ids moved\n before %v\n  after %v", p, a.elements, b.elements)
		}
		if !reflect.DeepEqual(a.sections, b.sections) {
			t.Errorf("%s: section ids moved\n before %v\n  after %v", p, a.sections, b.sections)
		}
		if !reflect.DeepEqual(a.edges, b.edges) {
			t.Errorf("%s: edges moved\n before %v\n  after %v", p, a.edges, b.edges)
		}
	}
}

func docsOf(m map[string]*scan.Document) []*scan.Document {
	var out []*scan.Document
	for _, d := range m {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// TestStrictPatchesAreIdempotent: a second strict pass over an already-
// patched record proposes nothing more. A rule that re-proposed its own
// output would loop an applier forever, and it would mean the rule does
// not recognise the form it asks for.
func TestStrictPatchesAreIdempotent(t *testing.T) {
	for _, p := range fixtures(t) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(raw), "\n")
		for pass := 1; pass <= 2; pass++ {
			d := scan.Bytes([]byte(strings.Join(lines, "\n")), scan.Options{})
			ps := patchesOf(Run(d, Options{Strict: true}))
			if pass == 2 && len(ps) > 0 {
				t.Errorf("%s: %d patches still proposed after applying them once", p, len(ps))
				break
			}
			lines = apply(lines, ps)
		}
	}
}

// TestNonStrictIsUnchanged: every finding a non-strict run produces is
// exactly what it produced before this change, and none of them carries a
// patch. The patch field is strict's alone — a stage linting a live
// record mid-flow gets the same advice it always got.
func TestNonStrictIsUnchanged(t *testing.T) {
	for _, p := range fixtures(t) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})
		for _, f := range Run(d, Options{}).Findings {
			if f.Patch != nil {
				t.Errorf("%s: %s carries a patch outside --strict", p, f.Code)
			}
		}
	}
}

// TestStrictSpeaksOnTerminalRecords: the tier lift is the point of the
// flag. A terminal record gets no conformance finding ordinarily —
// advice no one may act on is noise — and gets them under strict,
// because its structure may be migrated even though its content may not.
//
// The verdict is unchanged either way. Strict prices a migration; it
// does not fail a record for needing one.
func TestStrictSpeaksOnTerminalRecords(t *testing.T) {
	for _, name := range []string{"epoch-a.md", "epoch-b.md", "epoch-c.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})

		plain := Run(d, Options{})
		if !plain.Terminal {
			t.Fatalf("%s is not terminal; the fixture no longer tests the tier lift", name)
		}
		for _, f := range plain.Findings {
			if f.Tier == TierConformance {
				t.Errorf("%s: terminal record got conformance finding %s without --strict", name, f.Code)
			}
		}

		strict := Run(d, Options{Strict: true})
		n := 0
		for _, f := range strict.Findings {
			if f.Tier == TierConformance {
				n++
			}
		}
		if n == 0 {
			t.Errorf("%s: terminal record got no conformance finding under --strict", name)
		}
		if strict.Verdict != plain.Verdict {
			t.Errorf("%s: strict changed the verdict %s -> %s; it must change no verdict",
				name, plain.Verdict, strict.Verdict)
		}
	}
}

// TestJudgmentFindingsCarryNoPatch: the never-guess rule, as a test. A
// finding whose repair is a decision must arrive with advice and nothing
// executable, however mechanical the surrounding rule is.
func TestJudgmentFindingsCarryNoPatch(t *testing.T) {
	judgment := map[string]bool{
		"edge:unresolved": true, "edge:unresolved-terminal": true,
		"ownership:mutual": true, "peer-evidence:no-element": true,
		"label:contracts": true, "label:contracts-required": true,
		"template:missing-section": true, "gate:inline": true,
	}
	for _, p := range fixtures(t) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})
		for _, f := range Run(d, Options{Strict: true}).Findings {
			if strings.HasPrefix(f.Code, "parse:") || judgment[f.Code] {
				if f.Patch != nil {
					t.Errorf("%s: judgment finding %s carries a patch", p, f.Code)
				}
			}
		}
	}
}

// TestRenameDeclinesOnCollision: the alias table is many-to-one, and a
// record carrying two predecessors of one canonical section must not be
// told to give both the same name. The second is reported and left alone.
func TestRenameDeclinesOnCollision(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "epoch-a.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := scan.Bytes(raw, scan.Options{})
	seen := map[string]int{}
	patched := map[string]int{}
	for _, f := range Run(d, Options{Strict: true}).Findings {
		if f.Code != "section:legacy-name" {
			continue
		}
		seen[f.Message]++
		if f.Patch != nil {
			patched[f.Patch.Text]++
		}
	}
	if len(seen) < 2 {
		t.Fatalf("epoch-a should carry two aliases of one section, got %v", seen)
	}
	for text, n := range patched {
		if n > 1 {
			t.Errorf("%d patches all write %q; the record would carry duplicate sections", n, text)
		}
	}
}

// TestLabelPatchWritesTheDerivedID: the safety property of the label
// rule, checked directly rather than through the graph.
func TestLabelPatchWritesTheDerivedID(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "epoch-a.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := scan.Bytes(raw, scan.Options{})
	n := 0
	for _, f := range Run(d, Options{Strict: true}).Findings {
		if f.Code != "label:missing" || f.Patch == nil {
			continue
		}
		key := f.Element[strings.LastIndex(f.Element, ":")+1:]
		if !strings.Contains(f.Patch.Text, key) {
			t.Errorf("patch for %s does not write %q: %q", f.Element, key, f.Patch.Text)
		}
		n++
	}
	if n == 0 {
		t.Skip("no label patches in this fixture")
	}
}

// TestSpliceLabelDeclinesOnUnknownShapes: the splice edits one line and
// understands two shapes. Anything else must decline rather than write
// the label somewhere it does not belong.
func TestSpliceLabelDeclinesOnUnknownShapes(t *testing.T) {
	cases := []struct {
		line, label, want string
		ok                bool
	}{
		{"- **Statement.** rest", "A1", "- **A1 Statement.** rest", true},
		{"- [x] **Statement.** rest", "A2", "- [x] **A2 Statement.** rest", true},
		{"  1. **Statement**", "S3", "  1. **S3 Statement**", true},
		{"- plain text item", "S4", "- **S4** plain text item", true},
		{"- text with **bold** inside", "S5", "", false},
		{"- **A1 already labelled**", "A1", "", false},
		{"not a list item at all", "A9", "", false},
		{"", "A9", "", false},
	}
	for _, c := range cases {
		got, ok := spliceLabel(c.line, c.label)
		if ok != c.ok || got != c.want {
			t.Errorf("spliceLabel(%q, %q) = (%q, %v), want (%q, %v)", c.line, c.label, got, ok, c.want, c.ok)
		}
	}
}

// TestCitationPatchSkipsFencedAndRepeatedSpans: a citation inside a fence
// is quoted content, and a citation appearing twice in its range has no
// single place to rewrite. Both are reported without a patch.
func TestCitationPatchSkipsFencedAndRepeatedSpans(t *testing.T) {
	src := `# Recommendation 0090: Fenced Citations

## Metadata

- **Status**: Implemented
- **Date**: 2026-08-01
- **Predecessors**: cli/0091 A1

## Problem Statement

The peer is ` + "`cli/0091 A1`" + ` and again cli/0091 A1 on one line.

` + "```" + `
A quoted record writes cli/0092 C3 inside a fence.
` + "```" + `
`
	d := scan.Bytes([]byte(src), scan.Options{})
	for _, f := range Run(d, Options{Strict: true}).Findings {
		if f.Code != "citation:form" || f.Patch == nil {
			continue
		}
		if d.Fenced(f.Patch.LineStart) {
			t.Errorf("patched a citation inside a fence at line %d", f.Patch.LineStart)
		}
		if strings.Count(d.Line(f.Patch.LineStart), "cli/0091 A1") > 1 {
			t.Errorf("patched a line carrying the citation twice: %q", d.Line(f.Patch.LineStart))
		}
	}
}

// TestColonForm keeps the author's record prefix and changes only the
// separator: adding or dropping a project prefix is a second change the
// rule has no mandate to make.
func TestColonForm(t *testing.T) {
	cases := []struct{ evidence, to, want string }{
		{"cli/0055 A5", "cli/0055:A5", "cli/0055:A5"},
		{"RDR 0055 C4", "0055:C4", "RDR 0055:C4"},
		{"cli/0055 §Technical Design", "cli/0055:§technical-design", "cli/0055:§technical-design"},
		{"0055", "0055", ""},
		{"", "0055:A1", ""},
	}
	for _, c := range cases {
		if got := colonForm(c.evidence, c.to); got != c.want {
			t.Errorf("colonForm(%q, %q) = %q, want %q", c.evidence, c.to, got, c.want)
		}
	}
}

// TestOneCitationPatchPerLine is the regression for the defect that made
// the citation rule unsafe: thirty-three corpus lines carry more than one
// citation, and a patch computed per citation against the original line
// makes the second overwrite the first. Every finding on a line must
// share one patch, and that patch must carry BOTH rewrites.
func TestOneCitationPatchPerLine(t *testing.T) {
	// Only a RESOLVED citation is ever rewritten, so the target record
	// has to exist: the rule migrates a spelling, it never repairs a
	// reference.
	target := "# Recommendation 0091: The Target\n\n" +
		"## Metadata\n\n" +
		"- **Status**: Implemented\n" +
		"- **Date**: 2026-08-01\n\n" +
		"## Critical Assumptions\n\n" +
		"- **A1 First.**\n" +
		"- **A2 Second.**\n\n" +
		"#### Normative Contracts\n\n" +
		"**C3**\n\n" +
		"```normative\nThe third contract.\n```\n"
	src := "# Recommendation 0090: Two Citations On One Line\n\n" +
		"## Metadata\n\n" +
		"- **Status**: Implemented\n" +
		"- **Date**: 2026-08-01\n\n" +
		"## Problem Statement\n\n" +
		"Ordering inherits cli/0091 A2 and drop-safety inherits cli/0091 C3.\n"

	d := scan.Bytes([]byte(src), scan.Options{Project: "cli"})
	td := scan.Bytes([]byte(target), scan.Options{Project: "cli"})
	docs := []*scan.Document{d, td}
	scan.NewResolver(docs, "").ResolveAll(docs)

	var patches []*Patch
	seen := map[*Patch]bool{}
	n := 0
	for _, f := range Run(d, Options{Strict: true}).Findings {
		if f.Code != "citation:form" {
			continue
		}
		n++
		if f.Patch != nil && !seen[f.Patch] {
			seen[f.Patch] = true
			patches = append(patches, f.Patch)
		}
	}
	if n < 2 {
		t.Fatalf("expected two citations on the line, got %d findings", n)
	}
	if len(patches) != 1 {
		t.Fatalf("two citations on one line produced %d distinct patches, want 1", len(patches))
	}
	got := patches[0].Text
	for _, want := range []string{"cli/0091:A2", "cli/0091:C3"} {
		if !strings.Contains(got, want) {
			t.Errorf("the shared patch drops %s: %q", want, got)
		}
	}
	if strings.Contains(got, "cli/0091 A2") || strings.Contains(got, "cli/0091 C3") {
		t.Errorf("the shared patch left a citation unmigrated: %q", got)
	}
}

// TestRenameCarriesNoPatch: a legacy heading is reported and never
// patched. The rename moves the section's id and, because the canonical
// name carries a canonical LEVEL, can re-parent the content below it —
// on the epoch A fixture, promoting `#### API Verification` to `##
// Critical Assumptions` swallows the Normative Contracts that follow and
// reads their bullets as assumptions. Renames are a hand pass with lint
// as the checker.
func TestRenameCarriesNoPatch(t *testing.T) {
	found := 0
	for _, p := range fixtures(t) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})
		for _, f := range Run(d, Options{Strict: true}).Findings {
			if f.Code != "section:legacy-name" {
				continue
			}
			found++
			if f.Patch != nil {
				t.Errorf("%s: section:legacy-name carries a patch: %q", p, f.Patch.Text)
			}
		}
	}
	if found == 0 {
		t.Fatal("no legacy headings in the fixtures; the test proves nothing")
	}
}

// TestReLevellingMovesNoID is the property that makes the largest rule in
// the pass safe: a canonical section takes the canonical slug whatever
// level it is written at, so re-levelling breaks no citation.
func TestReLevellingMovesNoID(t *testing.T) {
	base := "# Recommendation 0001: T\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-01-01\n\n"
	var ids []string
	for _, h := range []string{"###", "##", "####"} {
		d := scan.Bytes([]byte(base+h+" Critical Assumptions\n\n- **A1 Something.**\n"), scan.Options{})
		for _, n := range d.Outline {
			if n.Canonical == "Critical Assumptions" {
				ids = append(ids, n.ID)
			}
		}
	}
	if len(ids) != 3 {
		t.Fatalf("expected the section at three levels, got %v", ids)
	}
	for _, id := range ids {
		if id != ids[0] {
			t.Errorf("re-levelling moved the section id: %v", ids)
		}
	}
}

// TestPromotionThatCapturesASiblingIsNotPatched: re-levelling moves no
// id, but it does change what a section CONTAINS. A record writing
// `### Critical Assumptions` followed by a sibling `### Resolve
// Deviations` must not be told to promote the first to `##`, because the
// second would become its child and that section's bullets would be read
// as assumptions the record never wrote.
func TestPromotionThatCapturesASiblingIsNotPatched(t *testing.T) {
	base := "# Recommendation 0001: T\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-01-01\n\n"
	captures := base +
		"### Critical Assumptions\n\n- **A1 Real.**\n\n" +
		"### Resolve Deviations (author ruling)\n\n- **O-1's note.** Not an assumption.\n"
	safe := base +
		"### Critical Assumptions\n\n- **A1 Real.**\n\n" +
		"## Proposed Solution\n\nBody.\n"

	d := scan.Bytes([]byte(captures), scan.Options{})
	f := find(t, Run(d, Options{Strict: true}), "heading:level")
	if f.Patch != nil {
		t.Errorf("promotion that captures a sibling was patched: %q", f.Patch.Text)
	}

	d2 := scan.Bytes([]byte(safe), scan.Options{})
	f2 := find(t, Run(d2, Options{Strict: true}), "heading:level")
	if f2.Patch == nil {
		t.Error("a promotion that captures nothing should still be patched")
	}

	// And the property that makes re-levelling worth doing at all: the
	// section's id is the same at either level.
	var before, after string
	for _, n := range d2.Outline {
		if n.Canonical == "Critical Assumptions" {
			before = n.ID
		}
	}
	out := apply(strings.Split(safe, "\n"), patchesOf(Run(d2, Options{Strict: true})))
	for _, n := range scan.Bytes([]byte(strings.Join(out, "\n")), scan.Options{}).Outline {
		if n.Canonical == "Critical Assumptions" {
			after = n.ID
		}
	}
	if before == "" || before != after {
		t.Errorf("re-level moved the section id: %q -> %q", before, after)
	}
}

// TestPositionallyReadElementsAreNotLabelled: where a section labels none
// of its items the projector reads them BY POSITION, which is a tolerance
// and not the record's claim. Record 0030 carries `#### Scoping notes
// (not Critical Assumptions)` — a heading that says in words its bullets
// are not assumptions — and labelling them would both contradict it and
// flip the section's labelled flag, at which point the scanner drops
// every unlabelled item and the record's real assumptions disappear.
func TestPositionallyReadElementsAreNotLabelled(t *testing.T) {
	src := "# Recommendation 0001: T\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-01-01\n\n" +
		"## Critical Assumptions\n\n" +
		"- **A1 A real, labelled assumption.**\n" +
		"- **A2 Another real one.**\n\n" +
		"#### Scoping notes (not Critical Assumptions)\n\n" +
		"- **No per-element payload flags.** A note, not an assumption.\n" +
		"- **Triggers are out of scope.** Also not an assumption.\n"
	d := scan.Bytes([]byte(src), scan.Options{})

	beforeIDs := map[string]bool{}
	for _, e := range d.Elements {
		beforeIDs[e.ID] = true
	}
	ps := patchesOf(Run(d, Options{Strict: true}))
	out := apply(strings.Split(src, "\n"), ps)
	nd := scan.Bytes([]byte(strings.Join(out, "\n")), scan.Options{})

	for id := range beforeIDs {
		found := false
		for _, e := range nd.Elements {
			if e.ID == id {
				found = true
			}
		}
		if !found {
			t.Errorf("applying patches destroyed element %s", id)
		}
	}
}

// TestIndentedFenceGetsNoLabel: a ```normative block nested inside an
// assumption's Evidence cannot take a label at column zero — the label
// line would end the enclosing list item. Record 0119's A16 loses two
// thirds of its lines and two dozen edges that way.
func TestIndentedFenceGetsNoLabel(t *testing.T) {
	src := "# Recommendation 0001: T\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-01-01\n\n" +
		"## Critical Assumptions\n\n" +
		"- **A1 An assumption whose evidence quotes a contract.**\n" +
		"  - **Evidence**: the block below.\n\n" +
		"    ```normative\n    Nested contract text.\n    ```\n"
	d := scan.Bytes([]byte(src), scan.Options{})
	for _, f := range Run(d, Options{Strict: true}).Findings {
		if f.Code == "label:missing" && strings.Contains(f.Element, ":C") && f.Patch != nil {
			t.Errorf("indented fence got a label patch: %q at %d", f.Patch.Text, f.Patch.LineStart)
		}
	}
}

// TestSectionCitationsAreNotRewritten: a `§Name` citation is a bounded
// fragment of the target's heading, while the id it resolves to is the
// slug of the whole text. Rewriting it to the full slug replaces what the
// author wrote with something longer, and across the corpus every edge
// broken by an earlier version of this rule was a section citation and
// not one was an element citation.
func TestSectionCitationsAreNotRewritten(t *testing.T) {
	target := "# Recommendation 0091: The Target\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-08-01\n\n" +
		"### Technical Design and the rest of a long heading\n\nBody.\n\n" +
		"## Critical Assumptions\n\n- **A1 First.**\n"
	src := "# Recommendation 0090: Cites A Section\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-08-01\n\n" +
		"## Problem Statement\n\n" +
		"See cli/0091 §Technical Design and the rest, plus cli/0091 A1.\n"
	d := scan.Bytes([]byte(src), scan.Options{Project: "cli"})
	td := scan.Bytes([]byte(target), scan.Options{Project: "cli"})
	docs := []*scan.Document{d, td}
	scan.NewResolver(docs, "").ResolveAll(docs)

	elements := 0
	for _, f := range Run(d, Options{Strict: true}).Findings {
		if f.Code != "citation:form" || f.Patch == nil {
			continue
		}
		if strings.Contains(f.Message, ":§") {
			t.Errorf("a section citation was patched: %q", f.Patch.Text)
		} else {
			elements++
		}
	}
	if elements == 0 {
		t.Error("the element citation on the same line was not patched")
	}
}
