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
	d := Bytes(fixture(t, "current-shape.md"), Options{})

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
	c := Bytes(fixture(t, "gate-inline.md"), Options{})
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

// TestOverridesClauseLeaderIsTheTarget pins the Overrides grammar: within
// each `;`-clause the first reference is what the record overrides, and
// every further reference is a peer the explanation names. The last case
// is the corpus shape that read as a mutual override — 0113's field
// overrides cli/0092 and names cli/0112 as the band it stays inside.
func TestOverridesClauseLeaderIsTheTarget(t *testing.T) {
	// Expected targets are listed in sorted order: same-line edges sort
	// by target, and a local number inherits the document's prefix.
	cases := []struct {
		name, value string
		overrides   []string
		mentions    []string
	}{
		{"single ref", "cli/0003 Approach item 3; the op was renamed", []string{"cli/0003"}, nil},
		{"multi-ref clause", "cli/0092's default rung is overridden outside cli/0112's fold band", []string{"cli/0092"}, []string{"cli/0112"}},
		{"multiple clauses", "cli/0103 REQ-13's file grain; cli/0120 REQ-CARRIER-6's zero-record read; 0089 A5", []string{"cli/0089", "cli/0103", "cli/0120"}, nil},
		{"element cite leads", "cli/0092:A6 — narrows the rung cli/0094 shipped; cli/0092:A4 (tracker#ngzs) and cli/0081 REQ-37", []string{"cli/0092:A4", "cli/0092:A6"}, []string{"cli/0081", "cli/0094"}},
		{"0113 shape", "overrides cli/0092; cli/0112 stays authoritative for the fold band cli/0092 sits inside", []string{"cli/0092", "cli/0112"}, []string{"cli/0092"}},
		// A leader is the first token of its clause. Prose after a `;`
		// makes the reference a mention — the prose semicolon that read
		// a cross-pointer as an override — while emphasis, a bracket, a
		// dash or the list joiner `and` do not.
		{"prose semicolon", "cli/0103 REQ-38's read arm; this field is the record. Also owed to a peer, not an override: cli/0112 L-3's prose names a spelling I-1 removes", []string{"cli/0103"}, []string{"cli/0112"}},
		{"verb after semicolon", "cli/0092:A6 — narrows the rung; also overrides cli/0092:A4 and cli/0081 REQ-37", []string{"cli/0092:A6"}, []string{"cli/0081", "cli/0092:A4"}},
		{"light punctuation leads", "cli/0103 REQ-13; **cli/0092**'s default rung; — (cli/0120) REQ-6; and cli/0089 A5", []string{"cli/0089:A5", "cli/0092", "cli/0103", "cli/0120"}, nil},
		{"semicolon inside the first clause", "cli/0103 REQ-38 (A5 coalesce; scenario (i) → ONE) — its normalization survives; cli/0104 REQ-35 → multiset", []string{"cli/0103", "cli/0104"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := "# Recommendation 0113: Clause leaders\n\n## Metadata\n\n- **Date**: 2026-08-29\n- **Status**: Draft\n- **Overrides**: " + c.value + "\n\n## Problem Statement\n\nNothing yet.\n"
			d := Bytes([]byte(src), Options{Project: "cli"})
			got := map[edge.Kind][]string{}
			for _, e := range d.Edges {
				if e.Field != "Overrides" {
					continue
				}
				got[e.Kind] = append(got[e.Kind], e.To)
			}
			if !equalStrings(got[edge.Overrides], c.overrides) {
				t.Errorf("overrides: got %v, want %v", got[edge.Overrides], c.overrides)
			}
			if !equalStrings(got[edge.Mentions], c.mentions) {
				t.Errorf("mentions: got %v, want %v", got[edge.Mentions], c.mentions)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestQuotedPeerCitationIsAMention pins the quotation exemption: a record
// token inside a closed double-quote pair of a Peer-RDR Evidence is the
// peer speaking, so it mints a quoted mention, not peer-evidence. Straight
// and curly pairs both count; an unclosed quote claims nothing, so the
// citation after a stray mark keeps its peer-evidence kind.
func TestQuotedPeerCitationIsAMention(t *testing.T) {
	src := "# Recommendation 0113: Quoted spans\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-29\n- **Status**: Draft\n\n" +
		"## Critical Assumptions\n\n- **A1 [Load-bearing]**: the peer holds.\n" +
		"  - **Status**: Verified\n  - **Method**: Peer RDR\n" +
		"  - **Evidence**: cli/0112:A3 states the bound — \"the fold admits cli/0092 once\" —\n" +
		"    and “cli/0081 owns the rows” is the peer's phrase; a stray \" leaves cli/0055 a citation.\n"
	d := Bytes([]byte(src), Options{Project: "cli"})
	want := map[string]struct {
		kind   edge.Kind
		quoted bool
	}{
		"cli/0112:A3": {edge.PeerEvidence, false},
		"cli/0092":    {edge.Mentions, true},
		"cli/0081":    {edge.Mentions, true},
		"cli/0055":    {edge.PeerEvidence, false},
	}
	seen := map[string]bool{}
	for _, e := range d.Edges {
		w, ok := want[e.To]
		if !ok || e.Field != "Evidence" {
			continue
		}
		seen[e.To] = true
		if e.Kind != w.kind || e.Quoted != w.quoted {
			t.Errorf("%s: kind=%s quoted=%v, want kind=%s quoted=%v", e.To, e.Kind, e.Quoted, w.kind, w.quoted)
		}
	}
	for to := range want {
		if !seen[to] {
			t.Errorf("no Evidence edge minted to %s", to)
		}
	}
}
