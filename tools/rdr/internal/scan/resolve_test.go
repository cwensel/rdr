package scan

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
)

// corpus writes a records dir from named sources and scans it, so a
// resolution test states the whole world it resolves against.
func corpus(t *testing.T, files map[string]string) []*Document {
	t.Helper()
	dir := t.TempDir()
	var docs []*Document
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sortStrings(names)
	for _, name := range names {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(files[name]), 0o600); err != nil {
			t.Fatal(err)
		}
		d, err := File(p, Options{})
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, d)
	}
	return docs
}

// record builds a minimal conformant record.
func record(num, slug, body string) string {
	return "# Recommendation " + num + ": " + slug + `

## Metadata

- **Date**: 2026-08-01
- **Status**: Final
- **Type**: Architecture
` + body + `

## Problem Statement

Synthetic.

## Critical Assumptions

- **A1 [The first claim holds]**
  - **Status**: Verified
  - **Method**: Derivation
  - **Evidence**: Shown inline.
  - **If wrong**: The design does not hold.
`
}

func findEdge(t *testing.T, d *Document, kind edge.Kind, to string) Edge {
	t.Helper()
	for _, e := range d.Edges {
		if e.Kind == kind && e.To == to {
			return e
		}
	}
	t.Fatalf("no %s edge to %q in %s; edges were %v", kind, to, d.Record, d.Edges)
	return Edge{}
}

// TestUnresolvedIsAFinding is the acceptance criterion: an unresolved
// typed edge is caught mechanically — "a Peer-RDR record citing 0055 A9
// when 0055 has A1–A7".
func TestUnresolvedIsAFinding(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", ""),
		"0002-beta.md": record("0002", "Beta", `- **Predecessors**: 0001-alpha
- **Overrides**: 0001-alpha A1 and 0001-alpha A9`),
	})
	NewResolver(docs, "").ResolveAll(docs)
	beta := docs[1]

	real := findEdge(t, beta, edge.Overrides, "0001:A1")
	if real.Resolved == nil || !*real.Resolved {
		t.Errorf("0001:A1 exists but resolved = %v", show(real.Resolved))
	}
	phantom := findEdge(t, beta, edge.Overrides, "0001:A9")
	if phantom.Resolved == nil || *phantom.Resolved {
		t.Errorf("0001 has no A9 but resolved = %v", show(phantom.Resolved))
	}
	pred := findEdge(t, beta, edge.Predecessor, "0001")
	if pred.Resolved == nil || !*pred.Resolved {
		t.Errorf("0001 exists but the predecessor edge resolved = %v", show(pred.Resolved))
	}
}

// TestMissingRecordIsUnresolved: a reference to a record the dir does not
// hold.
func TestMissingRecordIsUnresolved(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0002-beta.md": record("0002", "Beta", "- **Predecessors**: 0099-never-written"),
	})
	NewResolver(docs, "").ResolveAll(docs)
	e := findEdge(t, docs[0], edge.Predecessor, "0099")
	if e.Resolved == nil || *e.Resolved {
		t.Errorf("0099 does not exist but resolved = %v", show(e.Resolved))
	}
}

// TestStaleSlugIsUnresolved: a number and a filename slug that name two
// different records is a reference the author copied and half-updated.
func TestStaleSlugIsUnresolved(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", ""),
		"0002-beta.md":  record("0002", "Beta", "- **Predecessors**: 0001-the-old-name"),
	})
	NewResolver(docs, "").ResolveAll(docs)
	e := findEdge(t, docs[1], edge.Predecessor, "0001")
	if e.Resolved == nil || *e.Resolved {
		t.Errorf("the slug names a different record but resolved = %v", show(e.Resolved))
	}
}

// TestUncheckedIsNotUnresolved: with nothing to resolve against, an edge
// says so. It must never read as broken (a finding a consumer chases) nor
// as sound (a skipped check reading as a pass).
func TestUncheckedIsNotUnresolved(t *testing.T) {
	d := Bytes(fixture(t, "current-shape.md"), Options{})
	for _, e := range d.Edges {
		if e.Resolved != nil {
			t.Errorf("%s -> %s: resolved = %v with no resolver run", e.Kind, e.To, *e.Resolved)
		}
	}
	// A resolver with no records dir is equally honest.
	NewResolver(nil, "").Resolve(d)
	for _, e := range d.Edges {
		if e.Kind.Class() == edge.TargetElement && e.Resolved != nil {
			t.Errorf("%s -> %s: resolved = %v against an empty corpus", e.Kind, e.To, *e.Resolved)
		}
	}
	// And a source anchor with no repo is unchecked, not broken.
	docs := corpus(t, map[string]string{"0001-alpha.md": record("0001", "Alpha", "")})
	NewResolver(docs, "").Resolve(d)
	for _, e := range d.Edges {
		if e.Kind == edge.SourceAnchor && e.Resolved != nil {
			t.Errorf("anchor %q: resolved = %v with no repo", e.To, *e.Resolved)
		}
	}
}

// TestSymbolResolution is tooling-pass CHECK 5's rule: the SYMBOL is what
// resolves, never the line number, and a symbol found anywhere in the
// repo counts.
func TestSymbolResolution(t *testing.T) {
	repo := t.TempDir()
	src := "package frame\n\nfunc Encode(p []byte) []byte { return p }\n"
	if err := os.WriteFile(filepath.Join(repo, "frame.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Seam Lineage**: `frame::Encode` and `frame::Vanished`"),
	})
	NewResolver(docs, repo).ResolveAll(docs)

	found := findEdge(t, docs[0], edge.SourceAnchor, "frame::Encode")
	if found.Resolved == nil || !*found.Resolved {
		t.Errorf("frame::Encode is defined in the repo but resolved = %v", show(found.Resolved))
	}
	gone := findEdge(t, docs[0], edge.SourceAnchor, "frame::Vanished")
	if gone.Resolved == nil || *gone.Resolved {
		t.Errorf("frame::Vanished is nowhere but resolved = %v", show(gone.Resolved))
	}
}

// TestSymbolIsAWholeWord: `Encode` must not resolve out of `EncodeAll`,
// or every renamed symbol would look alive.
func TestSymbolIsAWholeWord(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "f.go"), []byte("func EncodeAll() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Seam Lineage**: `frame::Encode`"),
	})
	NewResolver(docs, repo).ResolveAll(docs)
	e := findEdge(t, docs[0], edge.SourceAnchor, "frame::Encode")
	if e.Resolved == nil || *e.Resolved {
		t.Errorf("Encode resolved out of EncodeAll (resolved = %v)", show(e.Resolved))
	}
}

// TestReceiverQualifiedSymbolResolvesToItsMember is CHECK 5 applied to a
// `Type.Method` anchor. No language writes the qualifier adjacent to the
// member at the definition, so grepping the dotted string whole reports a
// live method as missing — a false finding, which is strictly worse than
// the absent verdict a skipped check gives. The member is what must
// resolve; a method that is genuinely gone still reports false.
func TestReceiverQualifiedSymbolResolvesToItsMember(t *testing.T) {
	repo := t.TempDir()
	src := "package genealogy\n\nfunc (v *TableVertex) LiveConstraints() []string { return nil }\n"
	if err := os.WriteFile(filepath.Join(repo, "views.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Seam Lineage**: `views.go::TableVertex.LiveConstraints` "+
			"and `views.go::TableVertex.Vanished`"),
	})
	NewResolver(docs, repo).ResolveAll(docs)

	live := findEdge(t, docs[0], edge.SourceAnchor, "views.go::TableVertex.LiveConstraints")
	if live.Resolved == nil || !*live.Resolved {
		t.Errorf("LiveConstraints is declared on TableVertex but resolved = %v", show(live.Resolved))
	}
	gone := findEdge(t, docs[0], edge.SourceAnchor, "views.go::TableVertex.Vanished")
	if gone.Resolved == nil || *gone.Resolved {
		t.Errorf("Vanished is nowhere but resolved = %v", show(gone.Resolved))
	}
}

// TestQualifiedFallbackKeepsTheWholeWordRule: falling back to the member
// must not resolve `Type.Encode` out of `EncodeAll`, or the fallback would
// undo the rule TestSymbolIsAWholeWord exists to hold.
func TestQualifiedFallbackKeepsTheWholeWordRule(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "f.go"), []byte("func EncodeAll() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Seam Lineage**: `frame.go::Codec.Encode`"),
	})
	NewResolver(docs, repo).ResolveAll(docs)
	e := findEdge(t, docs[0], edge.SourceAnchor, "frame.go::Codec.Encode")
	if e.Resolved == nil || *e.Resolved {
		t.Errorf("Codec.Encode resolved out of EncodeAll (resolved = %v)", show(e.Resolved))
	}
}

// TestWordBoundaryHoldsAtEveryPosition pins the boundary rule against the
// index-based matcher: a leading, trailing or embedded identifier byte all
// disqualify a hit, and a later standalone occurrence still resolves.
func TestWordBoundaryHoldsAtEveryPosition(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{"EncodeAll()", false},
		{"reEncode()", false},
		{"x.reEncodeAll()", false},
		{"Encode()", true},
		{"EncodeAll(); Encode()", true},
		{"// Encode\n", true},
		{"Encode", true},
		{"NotEncode", false},
	} {
		if got := containsWord([]byte(tc.body), []byte("Encode")); got != tc.want {
			t.Errorf("containsWord(%q, Encode) = %v, want %v", tc.body, got, tc.want)
		}
	}
}

// TestSectionCitationResolvesExactly: resolution never guesses which
// section a partial citation meant. A reference that lands on no section
// of the target is reported — whether the author under-specified it or
// the citation grammar clipped it — because a dangling reference is
// record data to correct, not parser tolerance to add.
func TestSectionCitationResolvesExactly(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "") + "\n### Identity stack\n\nProse.\n\n### Semantic contract\n\nProse.\n\n### Semantic no-ops\n\nProse.\n",
		"0002-beta.md": record("0002", "Beta", "- **Overrides**: 0001-alpha §Identity stack, "+
			"0001-alpha §Semantic, and 0001-alpha §Nowhere at all"),
	})
	NewResolver(docs, "").ResolveAll(docs)

	ok := findEdge(t, docs[1], edge.Overrides, "0001:§identity-stack")
	if ok.Resolved == nil || !*ok.Resolved {
		t.Errorf("an exact section citation should resolve; resolved = %v", show(ok.Resolved))
	}
	// `§Semantic` names two headings and none exactly. Guessing one would
	// be the parser deciding what the author meant.
	ambiguous := findEdge(t, docs[1], edge.Overrides, "0001:§semantic")
	if ambiguous.Resolved == nil || *ambiguous.Resolved {
		t.Errorf("an under-specified citation must not resolve; resolved = %v", show(ambiguous.Resolved))
	}
	bad := findEdge(t, docs[1], edge.Overrides, "0001:§nowhere-at-all")
	if bad.Resolved == nil || *bad.Resolved {
		t.Errorf("a citation to no section should not resolve; resolved = %v", show(bad.Resolved))
	}
}

// TestUnresolvedEdgeCarriesARange: a dangling reference is fixed by a
// minimal pointer correction in the record, so the finding hands its
// reader the lines to open — a wrapped field's reference may be written
// on any of them.
func TestUnresolvedEdgeCarriesARange(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0002-beta.md": record("0002", "Beta", "- **Predecessors**: 0099-missing and\n  0098-also-missing"),
	})
	NewResolver(docs, "").ResolveAll(docs)
	for _, e := range docs[0].Edges {
		if e.Resolved == nil || *e.Resolved {
			continue
		}
		if e.LineEnd < e.Line {
			t.Errorf("%s -> %s: line_end %d precedes line %d", e.Kind, e.To, e.LineEnd, e.Line)
		}
		if e.Field == "Predecessors" && e.LineEnd == e.Line {
			t.Errorf("%s -> %s: a wrapped field reported a single line, not its range", e.Kind, e.To)
		}
	}
}

// TestReverseEdgesDerive is the acceptance criterion that reverse edges
// are derivable "without re-parsing": Reverse transposes the forward set
// and reads no file.
func TestReverseEdgesDerive(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", ""),
		"0002-beta.md":  record("0002", "Beta", "- **Predecessors**: 0001-alpha"),
		"0003-gamma.md": record("0003", "Gamma", "- **Predecessors**: 0001-alpha\n- **Overrides**: 0001-alpha"),
	})
	back := Reverse(docs)
	in := back["0001"]
	var preds, overs int
	for _, e := range in {
		switch e.Kind {
		case edge.Predecessor:
			preds++
		case edge.Overrides:
			overs++
		}
	}
	if preds != 2 || overs != 1 {
		t.Errorf("inbound to 0001: %d predecessor, %d overrides; want 2 and 1", preds, overs)
	}
	// A record nothing points at has no entry rather than a wrong one.
	if len(back["0003"]) != 0 {
		t.Errorf("0003 has inbound edges it should not: %v", back["0003"])
	}
}

// TestClusterRuleIsAQuery is the acceptance criterion that Stage 7.1's
// membership rule — "mutual Predecessors, Peer-RDR citations, or a shared
// Cross-Cutting Concern" — is expressible over edges[].
func TestClusterRuleIsAQuery(t *testing.T) {
	peer := `- **Predecessors**: 0001-alpha

## Critical Assumptions

- **A1 [The peer settles the question]**
  - **Status**: Pending
  - **Method**: Peer RDR
  - **Evidence**: 0001-alpha A1 owns the answer.
  - **If wrong**: Both records lock incompatible contracts.
`
	docs := corpus(t, map[string]string{
		// Mutual predecessors: each names the other.
		"0001-alpha.md": record("0001", "Alpha", "- **Predecessors**: 0002-beta"),
		"0002-beta.md":  record("0002", "Beta", "- **Predecessors**: 0001-alpha"),
		// One-way predecessor: the ordinary build-order dependency.
		"0003-gamma.md": record("0003", "Gamma", "- **Predecessors**: 0001-alpha"),
		// Peer-RDR citation.
		"0004-delta.md": "# Recommendation 0004: Delta\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Final\n" + peer,
		// Cross-cutting ownership.
		"0005-eps.md": record("0005", "Eps", "") + "\n### Cross-Cutting Concerns\n\n- **Versioning**: 0001-alpha owns the project-wide policy.\n",
		// Unrelated.
		"0006-zeta.md": record("0006", "Zeta", ""),
	})
	got := map[string]string{}
	for _, m := range ClusterOf(docs, "0001") {
		got[m.Record] = m.Relation
	}
	want := map[string]string{
		"0001": "seed",
		"0002": "mutual-predecessor",
		"0004": "peer-evidence",
		"0005": "cross-cutting",
	}
	for rec, rel := range want {
		if got[rec] != rel {
			t.Errorf("member %s: relation %q, want %q", rec, got[rec], rel)
		}
	}
	// A one-way predecessor is NOT a cluster member: every record has
	// several, and they resolve by implementing one first.
	if _, in := got["0003"]; in {
		t.Errorf("0003 is a one-way predecessor and should not be a member (got %q)", got["0003"])
	}
	if _, in := got["0006"]; in {
		t.Errorf("0006 is unrelated and should not be a member")
	}
}

// TestDeclaredClusterIsAuthoritative: an author's own Cluster field needs
// no derivation, in either direction.
func TestDeclaredClusterIsAuthoritative(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Cluster**: 0002-beta"),
		"0002-beta.md":  record("0002", "Beta", ""),
	})
	for _, seed := range []string{"0001", "0002"} {
		var found bool
		for _, m := range ClusterOf(docs, seed) {
			if m.Relation == "declared" {
				found = true
			}
		}
		if !found {
			t.Errorf("cluster of %s: the declared sibling is missing", seed)
		}
	}
}

// TestResolutionIsDeterministic: the same corpus resolves the same way,
// so a projection diff means a record changed.
func TestResolutionIsDeterministic(t *testing.T) {
	files := map[string]string{
		"0001-alpha.md": record("0001", "Alpha", ""),
		"0002-beta.md":  record("0002", "Beta", "- **Predecessors**: 0001-alpha, 0099-missing"),
	}
	first := corpus(t, files)
	NewResolver(first, "").ResolveAll(first)
	second := corpus(t, files)
	NewResolver(second, "").ResolveAll(second)
	for i := range first {
		a, b := first[i].Edges, second[i].Edges
		if len(a) != len(b) {
			t.Fatalf("%s: %d edges then %d", first[i].Record, len(a), len(b))
		}
		for j := range a {
			if a[j].To != b[j].To || a[j].Kind != b[j].Kind || show(a[j].Resolved) != show(b[j].Resolved) {
				t.Errorf("%s edge %d differs between runs", first[i].Record, j)
			}
		}
	}
}

func show(b *bool) string {
	if b == nil {
		return "unchecked"
	}
	if *b {
		return "true"
	}
	return "false"
}

// TestUniqueHeadingPrefixResolves: a section citation that is a whole-word
// PREFIX of exactly one heading of the target names that heading.
//
// This is deliberately NOT the prefix matching that was tried and
// reversed. That rule resolved any prefix, so `§Semantic` against five
// `Semantic *` headings picked one — the parser guessing what the author
// meant. The rule here refuses precisely that case (see
// TestAmbiguousHeadingPrefixStaysUnresolved, the guard on that decision)
// and resolves only where ONE heading can be meant, which is not a guess
// but an identification.
//
// The corpus clips both ways, so the relation is tested in both
// directions: the citation shorter than the heading (the author named it
// by its opening words, or the heading later grew a qualifier), and the
// citation longer (the grammar's word window ran past the heading into
// the sentence about it).
func TestUniqueHeadingPrefixResolves(t *testing.T) {
	target := record("0001", "Alpha", "") +
		"\n### No-op operations and OpID collision\n\nProse.\n" +
		"\n### Safety boundary (normative)\n\nProse.\n" +
		"\n### Failure Modes\n\nProse.\n"
	docs := corpus(t, map[string]string{
		"0001-alpha.md": target,
		"0002-beta.md": record("0002", "Beta", "- **Overrides**: "+
			// citation shorter than the heading
			"0001-alpha §No-op operations, "+
			"0001-alpha §Safety-boundary, "+
			// citation longer: the window ran into the prose
			"0001-alpha §Failure-Modes residual chartered it as a"),
	})
	NewResolver(docs, "").ResolveAll(docs)
	for _, want := range []string{
		"0001:§no-op-operations",
		"0001:§safety-boundary",
		"0001:§failure-modes-residual-chartered-it-as-a",
	} {
		e := findEdge(t, docs[1], edge.Overrides, want)
		if e.Resolved == nil || !*e.Resolved {
			t.Errorf("%s names exactly one heading of 0001 but resolved = %v", want, show(e.Resolved))
		}
	}
}

// TestAmbiguousHeadingPrefixStaysUnresolved is the REGRESSION GUARD on the
// decision that reversed prefix matching: a prefix shared by two or more
// headings identifies none of them, and choosing between them — first
// match, longest match, any tiebreak at all — is the parser guessing.
//
// It holds at every length the shortening walk tries, which is the part a
// tiebreak would quietly undo: `§Semantic contract engine` must not fall
// back through `§Semantic contract` to `§Semantic` and then pick one of
// the three.
func TestAmbiguousHeadingPrefixStaysUnresolved(t *testing.T) {
	target := record("0001", "Alpha", "") +
		"\n### Semantic contract\n\nProse.\n" +
		"\n### Semantic per-op replay\n\nProse.\n" +
		"\n### Semantic no-ops and simplifications\n\nProse.\n"
	docs := corpus(t, map[string]string{
		"0001-alpha.md": target,
		"0002-beta.md": record("0002", "Beta", "- **Overrides**: "+
			"0001-alpha §Semantic, 0001-alpha §Semantic engine rewrites the fold, "+
			"and 0001-alpha §Semantic per-op"),
	})
	NewResolver(docs, "").ResolveAll(docs)
	for _, want := range []string{"0001:§semantic", "0001:§semantic-engine-rewrites-the-fold"} {
		e := findEdge(t, docs[1], edge.Overrides, want)
		if e.Resolved == nil || *e.Resolved {
			t.Errorf("%s is ambiguous across three headings but resolved = %v", want, show(e.Resolved))
		}
	}
	// The guard rejects ambiguity, not specificity: one more word picks
	// out exactly one heading, and that one resolves.
	ok := findEdge(t, docs[1], edge.Overrides, "0001:§semantic-per-op")
	if ok.Resolved == nil || !*ok.Resolved {
		t.Errorf("§Semantic per-op names one heading of the three but resolved = %v", show(ok.Resolved))
	}
}

// TestHeadingPrefixIsWholeWords: a prefix relation counts only where the
// shorter slug ends on a segment boundary of the longer. `§norm` is not a
// citation of `Normative Contracts`; it is four letters that happen to
// start it.
func TestHeadingPrefixIsWholeWords(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "") + "\n### Normative Contracts\n\nProse.\n",
		"0002-beta.md":  record("0002", "Beta", "- **Overrides**: 0001-alpha §Norm"),
	})
	NewResolver(docs, "").ResolveAll(docs)
	e := findEdge(t, docs[1], edge.Overrides, "0001:§norm")
	if e.Resolved == nil || *e.Resolved {
		t.Errorf("a partial-word prefix must not resolve; resolved = %v", show(e.Resolved))
	}
}

// TestBoldLeadIsAddressable: the corpus cites bold paragraph leads with
// `§` and a quoted string — the most precise citation the grammar offers,
// and always verbatim. Indexing only `#`-headings left the most careful
// references in the corpus resolving against nothing.
func TestBoldLeadIsAddressable(t *testing.T) {
	target := record("0001", "Alpha", "") + `
### Approach

**The values.** ` + "`purpose ∈ {a, b, c}`" + ` is the closed set.

**Attribute resolution is as-authored, byte-preserving.**
Text-valued attributes hash by their literal UTF-8 bytes.

The paragraph continues, and a **bold run mid-paragraph** is emphasis.
`
	docs := corpus(t, map[string]string{
		"0001-alpha.md": target,
		"0002-beta.md": record("0002", "Beta", "- **Overrides**: "+
			`0001-alpha §"The values", `+
			`0001-alpha §"Attribute resolution is as-authored, byte-preserving", `+
			`0001-alpha §"bold run mid-paragraph"`),
	})
	NewResolver(docs, "").ResolveAll(docs)

	for _, want := range []string{
		"0001:§the-values",
		"0001:§attribute-resolution-is-as-authored-byte-preserving",
	} {
		e := findEdge(t, docs[1], edge.Overrides, want)
		if e.Resolved == nil || !*e.Resolved {
			t.Errorf("%s is a bold lead of 0001 but resolved = %v", want, show(e.Resolved))
		}
	}
	// Emphasis inside a paragraph is not a lead: it opens nothing and
	// names nothing.
	mid := findEdge(t, docs[1], edge.Overrides, "0001:§bold-run-mid-paragraph")
	if mid.Resolved == nil || *mid.Resolved {
		t.Errorf("mid-paragraph emphasis is not addressable; resolved = %v", show(mid.Resolved))
	}
}

// TestBoldLeadResolvesExactlyOnly: a heading is named by the template and
// repeated across the corpus, so a prefix of one identifies it. A bold
// lead is a sentence written once, and a prefix of a sentence is not a
// citation of it.
func TestBoldLeadResolvesExactlyOnly(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "") +
			"\n### Approach\n\n**The values are closed.** Prose follows.\n",
		"0002-beta.md": record("0002", "Beta", `- **Overrides**: 0001-alpha §"The values"`),
	})
	NewResolver(docs, "").ResolveAll(docs)
	e := findEdge(t, docs[1], edge.Overrides, "0001:§the-values")
	if e.Resolved == nil || *e.Resolved {
		t.Errorf("a prefix of a bold lead must not resolve; resolved = %v", show(e.Resolved))
	}
}

// TestNumberedDecisionsAreAddressable: a cohort of records numbered their
// decisions instead of keying them by the template's classes, under a
// heading the template of their day did not name.
//
// Numbering is a label like any other — the author named the element — so
// the key is as-written, and it is the spelling the citation grammar
// already reads (`§D6` maps to `D-6`). Reading zero elements because the
// heading is not the canonical one made every citation into such a record
// dangle against a section that is plainly there.
func TestNumberedDecisionsAreAddressable(t *testing.T) {
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "") + `
### Load-Bearing Decisions

- **D1** Bodies inline on the op. No sidefiles, no blob hashes.
- **D2** Element-ID-keyed storage throughout.
- **D6** The rule walks the inverse-reference graph.
`,
		"0002-beta.md": record("0002", "Beta", "- **Overrides**: 0001-alpha §D6, 0001-alpha §D9"),
	})
	NewResolver(docs, "").ResolveAll(docs)

	var keys []string
	for _, e := range docs[0].Elements {
		if e.Kind == "D" {
			keys = append(keys, e.ID)
			if e.Derived {
				t.Errorf("%s is numbered by its author but reported derived", e.ID)
			}
		}
	}
	if len(keys) != 3 {
		t.Fatalf("0001 projects %d decisions, want 3: %v", len(keys), keys)
	}
	ok := findEdge(t, docs[1], edge.Overrides, "0001:D-6")
	if ok.Resolved == nil || !*ok.Resolved {
		t.Errorf("0001 defines D6 but resolved = %v", show(ok.Resolved))
	}
	// Resolution stays exact: a number the record does not have is still
	// a dangling reference.
	bad := findEdge(t, docs[1], edge.Overrides, "0001:D-9")
	if bad.Resolved == nil || *bad.Resolved {
		t.Errorf("0001 has no D9 but resolved = %v", show(bad.Resolved))
	}
}

// TestClusterTraversalReadsNoSource is the cost half of `--cluster-of`'s
// gate: the facet walks the edge GRAPH — three edge kinds and a direction
// test — and reads no `resolved` verdict, so deriving a cluster must not
// touch the source tree at all. Counted rather than timed, because the
// defect it guards is a walk that happens, not a walk that is slow.
//
// The output half is main.TestClusterOfIsIndependentOfTheRepo: same corpus
// with and without a --repo, byte-identical. Together they say the facet
// neither needs the walk nor pays for it.
func TestClusterTraversalReadsNoSource(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "w.go"), []byte("package w\n\nfunc Known() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Predecessors**: 0002\n- **Seam Lineage**: `w::Known`"),
		"0002-beta.md":  record("0002", "Beta", "- **Predecessors**: 0001"),
	})

	reads := 0
	r := NewResolver(docs, repo)
	r.onRead = func() { reads++ }

	members := ClusterOf(docs, "0001")
	if len(members) == 0 {
		t.Fatal("no cluster derived; the fixture no longer exercises the traversal")
	}
	if reads != 0 {
		t.Errorf("deriving a cluster read %d source files; the traversal reads no verdict and must read no source", reads)
	}

	// And the resolver still walks when something does ask for a verdict,
	// so this test cannot pass by the walk being broken outright.
	r.ResolveAll(docs)
	if reads == 0 {
		t.Error("resolution read no source at all; the counter is not wired to the walk")
	}
}

// TestSingleDocumentResolutionPrimesTheSymbolCache locks the walk count,
// not the verdicts. ResolveAll's contract is that the repo is walked ONCE
// however many symbols are cited; Resolve alone honours no such thing —
// it greps per symbol, so a record citing N symbols walks the tree N
// times. The single-record `inspect`/`lint` path used to call Resolve and
// paid exactly that: 1.45s against a 2k-file repo where one primed walk
// is 0.56s, for identical verdicts.
//
// The verdicts being identical is why this needs its own test. Nothing in
// the output distinguishes the two paths, so a revert to Resolve is
// invisible to every other test here and shows up only as a slow tool.
func TestSingleDocumentResolutionPrimesTheSymbolCache(t *testing.T) {
	repo := t.TempDir()
	// Distinct symbols, none of them defined: an absent symbol is the
	// case that cannot short-circuit, so each one costs a FULL walk when
	// the cache is not primed.
	src := "package p\n\nfunc Present() {}\n"
	if err := os.WriteFile(filepath.Join(repo, "p.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha",
			"- **Seam Lineage**: `p::Present`, `p::GoneOne`, `p::GoneTwo`, `p::GoneThree`"),
	})
	d := docs[0]

	reads := 0
	r := NewResolver(nil, repo)
	r.onRead = func() { reads++ }
	r.ResolveAll([]*Document{d})

	// One walk over a one-file repo reads one file. Per-symbol grepping
	// would read it once per distinct symbol.
	if reads != 1 {
		t.Errorf("the repo was read %d times for one record; ResolveAll must prime the symbol cache and walk once", reads)
	}

	// The priming must not change what the walk decides.
	if e := findEdge(t, d, edge.SourceAnchor, "p::Present"); e.Resolved == nil || !*e.Resolved {
		t.Errorf("p::Present is defined but resolved = %v", show(e.Resolved))
	}
	for _, gone := range []string{"p::GoneOne", "p::GoneTwo", "p::GoneThree"} {
		if e := findEdge(t, d, edge.SourceAnchor, gone); e.Resolved == nil || *e.Resolved {
			t.Errorf("%s is nowhere but resolved = %v", gone, show(e.Resolved))
		}
	}
}

// TestPrimingCachesMissesWithoutBreakingTheFallback is the pair of
// properties that make caching a miss safe, asserted together because
// either alone is satisfiable by a wrong implementation.
//
// COST: an absent symbol must be decided by the priming walk. Caching hits
// alone left it a cache miss, and a miss sends resolveSymbol back to grep —
// one more walk of the whole tree per absent symbol, which is what
// resolution is mostly looking for.
//
// CORRECTNESS: a receiver-qualified anchor is absent as a dotted string in
// every language that declares it as a member, so seeding the walk's raw
// result would answer the fallback before the member was tried and report a
// live method as missing. That is a false finding, strictly worse than the
// absent verdict a skipped check gives, and it is the defect 852aee4 fixed.
// The qualified symbol must still take its member's verdict.
func TestPrimingCachesMissesWithoutBreakingTheFallback(t *testing.T) {
	repo := t.TempDir()
	// Go declares the member without the qualifier, so `TableVertex.Live`
	// is nowhere in the tree while `Live` is.
	src := "package v\n\nfunc (v *TableVertex) Live() {}\n\nfunc Plain() {}\n"
	if err := os.WriteFile(filepath.Join(repo, "v.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	docs := corpus(t, map[string]string{
		"0001-alpha.md": record("0001", "Alpha", "- **Seam Lineage**: "+
			"`v.go::TableVertex.Live`, `v.go::TableVertex.Gone`, `v.go::Plain`, `v.go::Missing`"),
	})
	d := docs[0]

	reads := 0
	r := NewResolver(nil, repo)
	r.onRead = func() { reads++ }
	r.ResolveAll([]*Document{d})
	primed := reads

	if primed != 1 {
		t.Errorf("priming read the repo %d times; every cited symbol, present or absent, must be decided in one walk", primed)
	}

	for _, tc := range []struct {
		anchor string
		want   bool
	}{
		{"v.go::TableVertex.Live", true}, // via the member — the fallback
		{"v.go::Plain", true},
		{"v.go::TableVertex.Gone", false},
		{"v.go::Missing", false},
	} {
		e := findEdge(t, d, edge.SourceAnchor, tc.anchor)
		if e.Resolved == nil || *e.Resolved != tc.want {
			t.Errorf("%s resolved = %v, want %v", tc.anchor, show(e.Resolved), tc.want)
		}
	}

	// Deciding every cited symbol during priming is the point: resolving
	// again must reach no further file.
	r.Resolve(d)
	if reads != primed {
		t.Errorf("resolution walked again after priming (%d more reads); a decided symbol must never re-grep", reads-primed)
	}
}

// TestBothSeekStrategiesAgree is the gate on the two-strategy walk. Above
// tokenScanFloor the resolver enumerates each file's identifier tokens
// once; below it, it tests each needle against each file. They are chosen
// on COUNT alone, so a corpus pass and a single-record pass take different
// code paths to the same question — and if they ever disagree, a record's
// `resolved` verdict silently depends on how many symbols its neighbours
// cite, which is the false-finding class this whole file exists to refuse.
//
// The bodies are the cases a naive tokeniser gets wrong: a needle embedded
// in a longer identifier, a qualified name split across a call site, a
// dotted needle whose halves are adjacent but not dot-separated, and a
// needle carrying a byte the token stream cannot represent.
func TestBothSeekStrategiesAgree(t *testing.T) {
	repo := t.TempDir()
	files := map[string]string{
		"a.go": "package a\n\nfunc Present() {}\nfunc PresentAll() {}\n" +
			"func (v *TableVertex) LiveConstraints() {}\n",
		"b.go": "package b\n\nvar x = tv.LiveConstraints()\nvar y = OtherVertex.Member\n" +
			"// TableVertex LiveConstraints are not dotted here\n",
		"c.md": "Docs mention force.option.required and Standalone.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	needles := []string{
		"Present",                     // also a prefix of PresentAll
		"PresentAll",                  // the longer identifier
		"Presen",                      // a prefix of both, defined nowhere
		"TableVertex.LiveConstraints", // dotted, adjacent only in a comment
		"OtherVertex.Member",          // dotted, genuinely present
		"TableVertex.Missing",         // dotted, absent
		"LiveConstraints",             // the bare member
		"Standalone",                  // present in a doc
		"force.option.required",       // two dots: neither strategy may tokenise it
		"Absent",                      // present nowhere
	}
	want := map[string][]byte{}
	for _, n := range needles {
		want[n] = []byte(n)
	}

	r := &Resolver{repo: repo, symbols: map[string]bool{}}
	perNeedle := r.seekPerNeedle(want)
	tokens := r.seekTokens(want)

	for _, n := range needles {
		if perNeedle[n] != tokens[n] {
			t.Errorf("%q: per-needle=%v token-scan=%v; the two strategies must decide every needle identically",
				n, perNeedle[n], tokens[n])
		}
	}
	// Pin the verdicts themselves, so a change that breaks BOTH strategies
	// the same way is caught too — agreement alone would not see it.
	for n, want := range map[string]bool{
		"Present": true, "PresentAll": true, "Presen": false,
		"TableVertex.LiveConstraints": false, "OtherVertex.Member": true,
		"TableVertex.Missing": false, "LiveConstraints": true,
		"Standalone": true, "force.option.required": true, "Absent": false,
	} {
		if perNeedle[n] != want {
			t.Errorf("%q resolved %v, want %v", n, perNeedle[n], want)
		}
	}
}

// TestSeekPicksTheStrategyByNeedleCount pins the routing itself.
//
// The floor is invisible in OUTPUT — both strategies return the same
// answers, which is what the test above guarantees — so no verdict
// assertion can catch a floor that stopped being applied, and a mutation
// check proved it: an earlier version of this test, which only checked
// that both paths found their needles, PASSED with the floor set to zero.
//
// What is observable is which strategy RAN. seek is a router with two
// destinations, so the test asks it directly, by giving each destination a
// distinguishable footprint: a needle set whose needles are all absent
// forces both paths to read every file, and one containing a needle that
// only the token stream can decide separates them by result. Combined with
// the read counts, that pins the boundary in both directions — below the
// floor the per-needle path runs, at and above it the token path does.
func TestSeekPicksTheStrategyByNeedleCount(t *testing.T) {
	repo := t.TempDir()
	// `Alpha.Beta` appears ONLY as a dotted pair. Both strategies decide it
	// correctly — that is the agreement test above — so it is not a verdict
	// probe; it is here so the needle set is realistic at both sizes.
	if err := os.WriteFile(filepath.Join(repo, "a.go"),
		[]byte("package a\nfunc Zero() {}\nvar v = Alpha.Beta\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		if err := os.WriteFile(filepath.Join(repo, fmt.Sprintf("z%02d.go", i)),
			[]byte("package a\n// padding\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reads := func(want map[string][]byte) (int, map[string]bool) {
		n := 0
		r := &Resolver{repo: repo, symbols: map[string]bool{}, onRead: func() { n++ }}
		return n, r.seek(want)
	}

	// Below the floor: every needle is present in the first file, so the
	// per-needle strategy's early exit stops the walk after one read.
	nSmall, hitSmall := reads(map[string][]byte{"Zero": []byte("Zero")})
	if !hitSmall["Zero"] {
		t.Fatal("a one-needle seek must still find its needle")
	}
	if nSmall != 1 {
		t.Errorf("below the floor the walk read %d files, want 1: the per-needle strategy exits as soon as its last needle is found", nSmall)
	}

	// At and above the floor: exactly tokenScanFloor needles, all absent
	// but one, so neither strategy can exit early and the walk reads the
	// whole repo. What is asserted is that the ANSWER is still right at
	// this size — the token scan is what produced it.
	big := map[string][]byte{"Zero": []byte("Zero"), "Alpha.Beta": []byte("Alpha.Beta")}
	for i := len(big); i < tokenScanFloor; i++ {
		s := fmt.Sprintf("Sym%04d", i)
		big[s] = []byte(s)
	}
	if len(big) != tokenScanFloor {
		t.Fatalf("built %d needles, want exactly the floor (%d)", len(big), tokenScanFloor)
	}
	nBig, hitBig := reads(big)
	if !hitBig["Zero"] || !hitBig["Alpha.Beta"] {
		t.Errorf("at the floor the token scan found %v; both present needles must resolve", hitBig)
	}
	if len(hitBig) != 2 {
		t.Errorf("the token scan found %d needles, want only the two that exist", len(hitBig))
	}
	if nBig != 13 {
		t.Errorf("at the floor the walk read %d files, want all 13: absent needles cannot short-circuit", nBig)
	}
}

// BenchmarkSeekStrategies is what defends tokenScanFloor's VALUE. The two
// strategies agree on every verdict, so no unit test can tell a floor of
// 128 from one of 512 — a mutation check confirms both pass everything
// above. Only a measurement separates them, and this is it: run it against
// a real source tree to re-derive the crossover if the corpus, the repo or
// the machine changes.
//
//	go test ./internal/scan -bench SeekStrategies -benchtime 1x \
//	    -run '^$' -args -seekrepo /path/to/repo
//
// Measured 2026-08-28 on the reference repo (3,672 files, 59 MB), the
// per-needle strategy costs 70ms at 62 needles and 11.8s at 1,305, while
// the token scan is ~330ms at any size — so they cross near 256.
func BenchmarkSeekStrategies(b *testing.B) {
	repo := seekBenchRepo
	if repo == "" {
		b.Skip("set -seekrepo to a source tree to measure the crossover")
	}
	needles := func(n int) map[string][]byte {
		out := map[string][]byte{}
		for i := 0; i < n; i++ {
			s := fmt.Sprintf("Sym%06d", i)
			out[s] = []byte(s)
		}
		return out
	}
	for _, n := range []int{16, 64, 128, 256, 512, 1024} {
		want := needles(n)
		b.Run(fmt.Sprintf("perNeedle/%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				(&Resolver{repo: repo, symbols: map[string]bool{}}).seekPerNeedle(want)
			}
		})
		b.Run(fmt.Sprintf("tokens/%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				(&Resolver{repo: repo, symbols: map[string]bool{}}).seekTokens(want)
			}
		})
	}
}

var seekBenchRepo string

func init() {
	flag.StringVar(&seekBenchRepo, "seekrepo", "", "source tree for BenchmarkSeekStrategies")
}
