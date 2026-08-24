package scan

import (
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
)

// The typed-edge contract, as tests. Each of this file's tests answers
// one of the issue's acceptance criteria:
//
//	TestEveryEdgeSyntaxMaps      every syntax maps to a kind
//	TestUnmappedFormWarns        an unmapped form lands in warnings[]
//	TestUnresolvedIsAFinding     an unresolved typed edge is detectable
//	TestReverseEdgesDerive       backlinks come from the forward set
//	TestClusterRuleIsAQuery      7.1's membership rule as a graph query
//	TestUncheckedIsNotUnresolved unchecked never reads as broken or sound

// edgesOf indexes a document's edges by kind.
func edgesOf(d *Document, kind edge.Kind) []Edge {
	var out []Edge
	for _, e := range d.Edges {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func hasEdge(d *Document, kind edge.Kind, from, to string) bool {
	for _, e := range d.Edges {
		if e.Kind == kind && e.From == from && e.To == to {
			return true
		}
	}
	return false
}

// TestEveryEdgeSyntaxMaps is the first acceptance criterion: every
// reference syntax the template and the corpus write maps to a kind. The
// fixtures carry one of each, so the assertions read off real
// projections rather than hand-built strings.
func TestEveryEdgeSyntaxMaps(t *testing.T) {
	d := Bytes(fixture(t, "epoch-d.md"), Options{})

	// Metadata list fields.
	if !hasEdge(d, edge.Predecessor, "0004", "0001") || !hasEdge(d, edge.Predecessor, "0004", "0003") {
		t.Errorf("Predecessors: %v", edgesOf(d, edge.Predecessor))
	}
	if !hasEdge(d, edge.Overrides, "0004", "0003") {
		t.Errorf("Overrides: %v", edgesOf(d, edge.Overrides))
	}
	if !hasEdge(d, edge.Cluster, "0004", "0005") {
		t.Errorf("Cluster: %v", edgesOf(d, edge.Cluster))
	}

	// The joint-decision qualifier names the home and its element.
	if !hasEdge(d, edge.JointDecisionHome, "0004", "0005:A2") {
		t.Errorf("joint-decision home: %v", edgesOf(d, edge.JointDecisionHome))
	}

	// A Peer-RDR assumption's Evidence citation is peer-evidence, and it
	// leaves the ASSUMPTION, not the document: the claim is what rests on
	// the peer.
	peer := edgesOf(d, edge.PeerEvidence)
	if len(peer) != 1 || peer[0].From != "0004:A3" || peer[0].To != "0005:A2" {
		t.Errorf("peer-evidence: %v", peer)
	}

	// Source anchors, from Seam Lineage and from an Evidence line.
	if !hasEdge(d, edge.SourceAnchor, "0004", "frame::Encode") {
		t.Errorf("Seam Lineage anchor missing: %v", edgesOf(d, edge.SourceAnchor))
	}
	if !hasEdge(d, edge.SourceAnchor, "0004:A2", "hash::Sum32") {
		t.Errorf("Evidence anchor missing: %v", edgesOf(d, edge.SourceAnchor))
	}

	// The weak kind exists and is separate.
	if len(edgesOf(d, edge.Mentions)) == 0 {
		t.Error("no mentions edges; the weak kind is not being emitted")
	}

	// Epoch C carries the two remaining metadata-side syntaxes.
	c := Bytes(fixture(t, "epoch-c.md"), Options{})
	if !hasEdge(c, edge.MovedTo, "0003", "0004") {
		t.Errorf("Demoted target: %v", edgesOf(c, edge.MovedTo))
	}
	// The Transient marker is written as a blockquote BELOW the fence it
	// qualifies, outside the contract element's own range, and must still
	// be read and attributed to that contract.
	tr := edgesOf(c, edge.TransientDeletedBy)
	if len(tr) != 1 || tr[0].To != "0004" {
		t.Fatalf("transient-deleted-by: %v", tr)
	}
	if !strings.HasPrefix(tr[0].From, "0003:C") {
		t.Errorf("transient edge leaves %q; it should leave the contract it annotates", tr[0].From)
	}
}

// TestEdgesAreNotDuplicated: a citation read by both a field pass and a
// body pass is one edge. Their offsets are in different coordinate
// systems — a joined value versus a line — so identity, not position, has
// to do the deduplication.
func TestEdgesAreNotDuplicated(t *testing.T) {
	for _, f := range allFixtures(t) {
		d := Bytes(fixture(t, f), Options{})
		seen := map[string]int{}
		for _, e := range d.Edges {
			k := e.From + "|" + e.To + "|" + string(e.Kind) + "|" + itoa(e.Line) + "|" + e.Field
			seen[k]++
			if seen[k] > 1 {
				t.Errorf("%s: %s -> %s (%s) at line %d emitted %d times", f, e.From, e.To, e.Kind, e.Line, seen[k])
			}
		}
	}
}

// TestMetadataIsReadByFieldNotByLine: the body passes never re-read a
// Metadata line. A `- **Date**: 2026-07-30` line offers a bare `2026` to
// anything reading it as prose, and an edge to record `2026` is a false
// relation no query can tell from a true one.
func TestMetadataIsReadByFieldNotByLine(t *testing.T) {
	for _, f := range allFixtures(t) {
		d := Bytes(fixture(t, f), Options{})
		for _, e := range d.Edges {
			if strings.HasSuffix(e.To, "2026") || strings.HasSuffix(e.To, "2025") {
				t.Errorf("%s: edge to %q — a date's year read as a record", f, e.To)
			}
		}
	}
}

// TestPlaceholderFieldsMintNoEdges: a seed value names CANDIDATES the
// author has not confirmed. Reading them as declared predecessors would
// assert a relation the field explicitly withholds.
func TestPlaceholderFieldsMintNoEdges(t *testing.T) {
	src := `# Recommendation 0009: Seeded

## Metadata

- **Date**: 2026-08-01
- **Status**: Draft
- **Predecessors**: _Draft placeholder — confirm in /rdr-propose. Candidates: 0001-alpha and 0002-beta._
- **Overrides**: (none — purely additive)

## Problem Statement

Nothing yet.
`
	d := Bytes([]byte(src), Options{})
	for _, e := range d.Edges {
		if e.Kind == edge.Predecessor || e.Kind == edge.Overrides {
			t.Errorf("placeholder value minted a %s edge to %s", e.Kind, e.To)
		}
	}
	// The candidates are still visible — as what they are.
	if len(edgesOf(d, edge.Mentions)) == 0 {
		t.Error("seed candidates dropped entirely; they should read as mentions")
	}
}

// TestUnmappedFormWarns is the acceptance criterion for AC-1's second
// half: a reference form no grammar reads is a warning, never a silence.
func TestUnmappedFormWarns(t *testing.T) {
	src := `# Recommendation 0009: Unmapped

## Metadata

- **Date**: 2026-08-01
- **Status**: Draft
- **Overrides**: proj/0003 § Approach item 3; the op was renamed → SplitTable

## Problem Statement

Nothing yet.
`
	d := Bytes([]byte(src), Options{Project: "proj"})
	var found bool
	for _, w := range d.Warnings {
		if w.Code == "edge:unmapped-reference" {
			found = true
			if w.LineStart == 0 {
				t.Error("unmapped warning carries no line")
			}
		}
	}
	if !found {
		t.Errorf("the rename arrow produced no warning; warnings were %v", d.Warnings)
	}
}

// TestEdgeWarningsAreNotUnclassifiedLines: the unclassified-line rate is
// the drift alarm for TEMPLATE.md changes. A line whose structure is
// fully classified but whose reference form is untyped is not structural
// drift, and counting it would move the alarm for an unrelated reason.
func TestEdgeWarningsAreNotUnclassifiedLines(t *testing.T) {
	src := `# Recommendation 0009: Unmapped

## Metadata

- **Date**: 2026-08-01
- **Status**: Draft
- **Overrides**: renamed → SplitTable

## Problem Statement

Nothing yet.
`
	d := Bytes([]byte(src), Options{})
	if d.Coverage.Unclassified != 0 {
		t.Errorf("unclassified = %d; an edge warning is not an unclassified line", d.Coverage.Unclassified)
	}
	if len(d.Warnings) == 0 {
		t.Error("the warning itself went missing")
	}
}
