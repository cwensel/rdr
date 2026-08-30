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

// fixtures are the whole-record shapes plus the variants: between them
// they carry every element grammar the corpus writes.
func fixtures(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, g := range []string{
		// The whole-record fixtures live directly in testdata/. They are
		// named for the SHAPE each one exercises, not for a generation, so
		// there is no shared prefix to glob — README.md is the only
		// non-record .md in the directory.
		filepath.Join("..", "..", "testdata", "*.md"),
		filepath.Join("..", "..", "testdata", "variants", "*.md"),
		filepath.Join("..", "..", "testdata", "lint", "*.md"),
	} {
		paths, err := filepath.Glob(g)
		if err != nil || len(paths) == 0 {
			t.Fatalf("no fixtures for %s: %v", g, err)
		}
		for _, p := range paths {
			if filepath.Base(p) == "README.md" {
				continue // the directory's own doc, not a record
			}
			out = append(out, p)
		}
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
		r := Run(pre[p], Options{Corpus: docsOf(pre)})
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
			ps := patchesOf(Run(d, Options{}))
			if pass == 2 && len(ps) > 0 {
				t.Errorf("%s: %d patches still proposed after applying them once", p, len(ps))
				break
			}
			lines = apply(lines, ps)
		}
	}
}

// TestConformanceAdvisesAndNeverBlocks: the invariant that let the
// current-template reading become the only one.
//
// A terminal record is told what migrating it would cost — it gets
// conformance findings, because its STRUCTURE may be brought to the
// template even though its CONTENT may not — and it still verdicts PASS.
// Conformance advises; it does not fail a record for needing a migration.
//
// This is the property the `--strict` flag used to protect by being
// off. If it ever broke, every gate in the flow would start blocking on
// records nobody is permitted to rewrite, so it is pinned directly:
// findings present, verdict unmoved, and nothing marked blocking.
func TestConformanceAdvisesAndNeverBlocks(t *testing.T) {
	for _, name := range []string{"legacy-shape.md", "assumptions-nested.md", "gate-inline.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})

		// Both readings: mid-flow and at a lock gate. Locking governs
		// resolution delivery, never conformance, so neither may block
		// on a conformance finding.
		for _, opts := range []Options{{}, {Locking: true}} {
			r := Run(d, opts)
			if !r.Terminal {
				t.Fatalf("%s is not terminal; the fixture no longer tests the rule", name)
			}
			n := 0
			for _, f := range r.Findings {
				if f.Tier != TierConformance {
					continue
				}
				n++
				if f.Blocking {
					t.Errorf("%s (locking=%v): conformance finding %s is blocking",
						name, opts.Locking, f.Code)
				}
			}
			if n == 0 {
				t.Errorf("%s (locking=%v): terminal record got no conformance finding",
					name, opts.Locking)
			}
			if r.Verdict != "PASS" {
				t.Errorf("%s (locking=%v): verdict %s; conformance must not fail a record",
					name, opts.Locking, r.Verdict)
			}
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
		"gate:cross-cutting-missing": true,
	}
	for _, p := range fixtures(t) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})
		for _, f := range Run(d, Options{}).Findings {
			if strings.HasPrefix(f.Code, "parse:") || judgment[f.Code] {
				if f.Patch != nil {
					t.Errorf("%s: judgment finding %s carries a patch", p, f.Code)
				}
			}
		}
	}
}

// TestLegacyNameNeverPatches: a rename moves the section's id and can
// re-parent the content below it, so `section:legacy-name` reports and
// withholds. This is the whole reason the code exists separately from
// `heading:level`, which does patch.
//
// It replaces TestRenameDeclinesOnCollision, which pinned the two-aliases-
// of-one-section case. The table is no longer many-to-one — API
// Verification was retired once no corpus record wrote it — and no record
// in either corpus ever carried two aliases of one canonical section, so
// that test asserted a hazard nothing exhibited.
func TestLegacyNameNeverPatches(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "legacy-shape.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := scan.Bytes(raw, scan.Options{})
	n := 0
	for _, f := range Run(d, Options{}).Findings {
		if f.Code != "section:legacy-name" {
			continue
		}
		n++
		if f.Patch != nil {
			t.Errorf("%s carries a patch; a rename is a hand pass with lint as the checker", f.Message)
		}
		if f.Fix == "" {
			t.Errorf("%s carries no fix; a withheld patch must say what to do instead", f.Message)
		}
	}
	if n == 0 {
		t.Fatal("legacy-shape should carry a legacy heading name")
	}
}

// TestLabelPatchWritesTheDerivedID: the safety property of the label
// rule, checked directly rather than through the graph.
func TestLabelPatchWritesTheDerivedID(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "legacy-shape.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := scan.Bytes(raw, scan.Options{})
	n := 0
	for _, f := range Run(d, Options{}).Findings {
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
	for _, f := range Run(d, Options{}).Findings {
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
	for _, f := range Run(d, Options{}).Findings {
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
// on the legacy-shape fixture, promoting `#### Dependency Source Verification` to `##
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
		for _, f := range Run(d, Options{}).Findings {
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
	f := find(t, Run(d, Options{}), "heading:level")
	if f.Patch != nil {
		t.Errorf("promotion that captures a sibling was patched: %q", f.Patch.Text)
	}

	d2 := scan.Bytes([]byte(safe), scan.Options{})
	f2 := find(t, Run(d2, Options{}), "heading:level")
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
	out := apply(strings.Split(safe, "\n"), patchesOf(Run(d2, Options{})))
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
	ps := patchesOf(Run(d, Options{}))
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
	for _, f := range Run(d, Options{}).Findings {
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
	for _, f := range Run(d, Options{}).Findings {
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

// TestGateInlineFiresOnInlinedGate: the rule exists to find the legacy
// and B records whose gate responses are still written into the record,
// and to stay quiet on the gate-pointer records where lock has already
// moved them to artifacts/gate.md and left the one-line pointer.
//
// It is a regression test for a rule that reported NOTHING, on any
// record, for as long as it existed: the guard compared the gate node's
// ID against the element's Section, and a gate element's Section is its
// own sub-heading, never the gate's. Both directions are asserted here,
// because a guard that is always false and a guard that is always true
// are equally wrong and only the pair of cases separates them.
func TestGateInlineFiresOnInlinedGate(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"legacy-shape.md", true},       // five inline gate subsections
		{"assumptions-nested.md", true}, // five inline gate subsections
		{"gate-inline.md", false},       // `See gate.md ...` pointer
		{"current-shape.md", false},     // `See gate.md ...` pointer
	} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", tc.name))
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})
		r := Run(d, Options{})
		if got := has(r, "gate:inline"); got != tc.want {
			t.Errorf("%s: gate:inline fired = %v, want %v (codes: %v)",
				tc.name, got, tc.want, codes(r))
		}
		if !tc.want {
			continue
		}
		// The finding must name the gate section and span its lines, so
		// the reader is pointed at the range to move.
		f := find(t, r, "gate:inline")
		if !strings.HasSuffix(f.Element, ":§finalization-gate") {
			t.Errorf("%s: gate:inline element = %q, want the gate section", tc.name, f.Element)
		}
		if f.LineEnd <= f.LineStart {
			t.Errorf("%s: gate:inline spans %d-%d, want the gate's line range",
				tc.name, f.LineStart, f.LineEnd)
		}
	}
}

// TestGateInlineIgnoresARetainedCrossCuttingConcerns: a locked record keeps
// `### Cross-Cutting Concerns` — it is the one gate item peers cite, so it
// stays projected as `NNNN:G-cross-cutting` — and moves the other four to
// gate.md. That record is correctly locked, and a rule that read any gate
// element as "inlined" would report it forever with nothing to repair.
func TestGateInlineIgnoresARetainedCrossCuttingConcerns(t *testing.T) {
	const rec = `# Recommendation 0031: Split gate

## Metadata

- **Status**: Final

## Finalization Gate

Responses: 0031-split-gate/artifacts/gate.md (Gate PASS 2026-08-26)

### Cross-Cutting Concerns

- **Character encoding**: schema names fold ASCII-only, per PostgreSQL's
  unquoted-identifier rule.
`
	d := scan.Bytes([]byte(rec), scan.Options{})
	r := Run(d, Options{})
	if has(r, "gate:inline") {
		t.Errorf("gate:inline fired on a correctly split gate (codes: %v)", codes(r))
	}
	// The retained item must still be a projected, citable element.
	var found bool
	for _, e := range d.Elements {
		if e.ID == "0031:G-cross-cutting" {
			found = true
		}
	}
	if !found {
		t.Error("0031:G-cross-cutting is not projected; peers cite it by that id")
	}
}

// TestGateCrossCuttingMissingOnBarePointer: a record locked before the
// template retained Cross-Cutting Concerns holds the pointer and nothing
// under it. The section is owed on re-lock, and the finding is what says
// so — pointing at the pointer line, naming the item, and naming the
// gate.md the pointer names as where the prior text lives. It is
// conformance, so it never blocks, and it is quiet the moment the item is
// back, as a heading or as a bullet.
func TestGateCrossCuttingMissingOnBarePointer(t *testing.T) {
	const bare = `# Recommendation 0033: Bare pointer

## Metadata

- **Status**: Final

## Finalization Gate

Responses: ` + "`0033-bare-pointer/artifacts/gate.md`" + ` (Gate PASS 2026-08-11)

## Consequences

- none
`
	d := scan.Bytes([]byte(bare), scan.Options{})
	r := Run(d, Options{Locking: true})
	f := find(t, r, "gate:cross-cutting-missing")
	if f.Tier != TierConformance || f.Blocking {
		t.Errorf("tier %q blocking %v, want conformance and non-blocking", f.Tier, f.Blocking)
	}
	if !strings.HasSuffix(f.Element, ":§finalization-gate") {
		t.Errorf("element = %q, want the gate section", f.Element)
	}
	if f.LineStart != 9 || f.LineEnd != 9 {
		t.Errorf("spans %d-%d, want the pointer line 9", f.LineStart, f.LineEnd)
	}
	if !strings.Contains(f.Message, "Cross-Cutting Concerns") {
		t.Errorf("message does not name the retained item: %q", f.Message)
	}
	if !strings.Contains(f.Fix, "0033-bare-pointer/artifacts/gate.md §Cross-Cutting Concerns") {
		t.Errorf("fix does not say where the prior text lives: %q", f.Fix)
	}
	if f.Patch != nil {
		t.Errorf("a re-answer was patched: %+v", f.Patch)
	}
	if r.Verdict != "PASS" {
		t.Errorf("verdict %q on a locking record whose only finding is conformance", r.Verdict)
	}

	for _, tc := range []struct{ name, rec string }{
		{"heading", `# Recommendation 0034: Kept

## Metadata

- **Status**: Final

## Finalization Gate

Responses: 0034-kept/artifacts/gate.md (Gate PASS 2026-08-26)

### Cross-Cutting Concerns

- **Character encoding**: ASCII-only.
`},
		{"bullet", `# Recommendation 0035: Kept as bullet

## Metadata

- **Status**: Final

## Finalization Gate

Responses: 0035-kept/artifacts/gate.md (Gate PASS 2026-08-26)

- **Cross-Cutting**: ASCII-only.
`},
		{"inlined", `# Recommendation 0036: Inlined

## Metadata

- **Status**: Draft

## Finalization Gate

### Contradiction Check

- none

### Scope Verification

- none
`},
	} {
		d := scan.Bytes([]byte(tc.rec), scan.Options{})
		if r := Run(d, Options{}); has(r, "gate:cross-cutting-missing") {
			t.Errorf("%s: gate:cross-cutting-missing fired (codes: %v)", tc.name, codes(r))
		}
	}
}

// TestPlaceholderSurvivedFiresOnTemplateText: the defect this rule exists
// for is a record that lints PERFECTLY — every section present, every
// heading canonical, coverage zero — while carrying TEMPLATE.md's own
// guidance where the author's words belong. A verbatim template copy was
// the proof: it passed with no findings at all.
//
// Both message variants are asserted, because the pair is the rule's
// judgement. A block sitting above real content and a block standing in
// for content never written need different repairs, and a rule that
// reported one message for both would be telling an author to delete a
// paragraph when the section is empty underneath.
func TestPlaceholderSurvivedFiresOnTemplateText(t *testing.T) {
	const guidanceOnly = `# Recommendation 0091: Authored, guidance left behind

## Metadata

- **Status**: Implemented

## Normative Contracts

[Required — never omit. Load-bearing — implementers must
match exactly.]

- **C1** The ` + "`--into`" + ` flag desugars into the existing record.
`
	const skeleton = `# Recommendation 0115: Never authored

## Metadata

- **Status**: Draft

## Critical Assumptions

[Required — never omit. Load-bearing assumptions — if
wrong, the approach fails.]

- **A1 [Statement]**
  - **Status**: Verified | Pending | Unverified
  - **Method**: ` + "`one of the eight below`" + `
`
	for _, tc := range []struct {
		name    string
		rec     string
		wantMsg string
	}{
		{"guidance above authored content", guidanceOnly, "not the author's"},
		{"guidance over an unfilled skeleton", skeleton, "unfilled skeleton"},
	} {
		d := scan.Bytes([]byte(tc.rec), scan.Options{})
		r := Run(d, Options{})
		f := find(t, r, "placeholder:survived")
		if !strings.Contains(f.Message, tc.wantMsg) {
			t.Errorf("%s: message = %q, want it to contain %q", tc.name, f.Message, tc.wantMsg)
		}
		// Migration advice, never a block - on any record, at any gate.
		if f.Blocking {
			t.Errorf("%s: placeholder:survived blocks; conformance never does", tc.name)
		}
		// The range must be the marker's own lines, not the section's, so
		// the reader is pointed at the text to delete.
		if f.LineEnd < f.LineStart {
			t.Errorf("%s: spans %d-%d", tc.name, f.LineStart, f.LineEnd)
		}
		if got := d.Line(f.LineStart); !strings.HasPrefix(got, "[Required") && !strings.HasPrefix(got, "[Conditional") {
			t.Errorf("%s: range opens at %q, want the marker line", tc.name, got)
		}
		if !strings.Contains(d.Line(f.LineEnd), "]") {
			t.Errorf("%s: range ends at %q, want the marker's closing bracket", tc.name, d.Line(f.LineEnd))
		}
	}
}

// TestPlaceholderSurvivedIgnoresSchemaMarkers: `[Gate key: …]` and
// `[Retained at lock — …]` are the template's SCHEMA, read as data by the
// model - a record carries none of them, and a rule matching every
// bracketed lead would report the template's own grammar as a defect on
// the corpus's own fixtures. A false finding is worse than an absent one.
func TestPlaceholderSurvivedIgnoresSchemaMarkers(t *testing.T) {
	const rec = `# Recommendation 0031: Schema markers are not guidance

## Metadata

- **Status**: Final

## Finalization Gate

[Gate key: contradiction]
[Retained at lock — this sub-section stays in the RDR]
`
	d := scan.Bytes([]byte(rec), scan.Options{})
	if r := Run(d, Options{}); has(r, "placeholder:survived") {
		t.Errorf("placeholder:survived fired on schema markers (codes: %v)", codes(r))
	}
}

// TestPlaceholderSurvivedIsSilentOnAConformantRecord: the rule must be
// quiet on a record whose sections are the author's own words. This is
// the guard that keeps 123 of the corpus's 144 records clean.
func TestPlaceholderSurvivedIsSilentOnAConformantRecord(t *testing.T) {
	for _, name := range []string{"current-shape.md", "gate-inline.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		d := scan.Bytes(raw, scan.Options{})
		if r := Run(d, Options{}); has(r, "placeholder:survived") {
			t.Errorf("%s: placeholder:survived fired on a conformant record", name)
		}
	}
}

// TestGateInlineSeesABulletGate: the two oldest records answer the gate as
// a labelled LIST, not as sub-headings. The projector read nothing there,
// so `gate:inline` never fired on them and the externalization pass left
// them behind — the rule reported a clean corpus while the two records it
// could not see stayed unmigrated.
//
// Their elements carry the GATE's own id as their section (there is no
// sub-heading to carry it), which is why the rule counts the gate node as
// being "under" itself.
func TestGateInlineSeesABulletGate(t *testing.T) {
	const rec = `# Recommendation 0005: Bullet gate

## Metadata

- **Status**: Implemented

## Finalization Gate

- **Contradiction Check**: none found.
- **Assumption Verification**: A1 stands.
- **Scope Verification**: the MVV is in scope.
- **Proportionality**: proportionate.
`
	d := scan.Bytes([]byte(rec), scan.Options{})
	r := Run(d, Options{})
	if !has(r, "gate:inline") {
		t.Fatalf("gate:inline did not fire on a bullet-written gate (codes: %v)", codes(r))
	}
	f := find(t, r, "gate:inline")
	if !strings.HasSuffix(f.Element, ":§finalization-gate") {
		t.Errorf("gate:inline element = %q, want the gate section", f.Element)
	}
}

// TestBulletGateUnderAPointerIsNotInline: a locked record may keep its
// retained item as a BULLET under the gate.md pointer. The pointer is
// checked first and settles it — reading those bullets as an inlined gate
// would report a correctly split record as unmigrated forever.
func TestBulletGateUnderAPointerIsNotInline(t *testing.T) {
	const rec = `# Recommendation 0032: Pointer plus bullets

## Metadata

- **Status**: Final

## Finalization Gate

Responses: 0032-pointer/artifacts/gate.md (Gate PASS 2026-08-26)

- **Cross-Cutting**: schema names fold ASCII-only.
`
	d := scan.Bytes([]byte(rec), scan.Options{})
	r := Run(d, Options{})
	if has(r, "gate:inline") {
		t.Errorf("gate:inline fired on a pointer gate whose retained item is a bullet (codes: %v)", codes(r))
	}
	// The retained item still has to be citable.
	var found bool
	for _, e := range d.Elements {
		if e.ID == "0032:G-cross-cutting" {
			found = true
		}
	}
	if !found {
		t.Error("0032:G-cross-cutting was not projected; a retained bullet must stay citable")
	}
}

// TestQuotedCitationIsNotRestyled: citation:form migrates the author's own
// spelling to the colon id, and a citation inside a closed quotation is
// not the author's — it is the peer's text verbatim, which a lint pass
// must never ask a record to rewrite. The unquoted cite on the same line
// keeps its finding, so the skip is the quotation's, not the field's.
func TestQuotedCitationIsNotRestyled(t *testing.T) {
	target := "# Recommendation 0091: The Target\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-08-01\n\n" +
		"## Critical Assumptions\n\n- **A1 First.**\n- **A2 Second.**\n"
	src := "# Recommendation 0090: Quoted Citation\n\n## Metadata\n\n" +
		"- **Status**: Implemented\n- **Date**: 2026-08-01\n\n" +
		"## Critical Assumptions\n\n- **A1 [Load-bearing]**: the peer holds.\n" +
		"  - **Status**: Verified\n  - **Method**: Peer RDR\n" +
		"  - **Evidence**: cli/0091 A2 stands, and the peer says \"cli/0091 A1 admits it once\".\n"
	d := scan.Bytes([]byte(src), scan.Options{Project: "cli"})
	td := scan.Bytes([]byte(target), scan.Options{Project: "cli"})
	docs := []*scan.Document{d, td}
	scan.NewResolver(docs, "").ResolveAll(docs)

	var msgs []string
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "citation:form" {
			msgs = append(msgs, f.Message)
		}
	}
	if len(msgs) != 1 || !strings.Contains(msgs[0], "cli/0091:A2") {
		t.Errorf("want one citation:form finding on the unquoted cite alone, got: %v", msgs)
	}
}
