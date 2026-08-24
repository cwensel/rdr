package scan

import (
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
	d := Bytes(fixture(t, "epoch-d.md"), Options{})
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
