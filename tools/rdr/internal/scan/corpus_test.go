package scan

import (
	"strings"
	"testing"
)

// head is a synthetic epoch-B-or-later record head with the given status.
func head(num, title, status string) string {
	return "# Recommendation " + num + ": " + title +
		"\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: " + status +
		"\n- **Type**: Feature\n- **Profile**: standard\n- **Priority**: High\n"
}

func synth(t *testing.T, num, body string) *Document {
	t.Helper()
	return Bytes([]byte(body), Options{Record: num})
}

// TestSummaryReadsMetadataNotBody: the worklist fields come off the
// classified Metadata, the title loses its lead, and in-flight is exactly
// Draft or Final.
func TestSummaryReadsMetadataNotBody(t *testing.T) {
	cases := map[string]struct{ inFlight, terminal bool }{
		"Draft": {true, false}, "Final": {true, false}, "Draft [revised from Final; re-verify A2]": {true, false},
		"Implemented": {false, true}, "Demoted [→ kata 1234]": {false, true},
		// Parked is the tri-state: a Deferred record is not in flight (no
		// stage is running) and not terminal (it re-enters when its
		// trigger fires). Both spellings are pinned — the bracketed form
		// a new record writes, and the bare one a frozen record left.
		"Deferred": {false, false},
		"Deferred [revisit when upstream exposes a pool knob]": {false, false},
		"Rejected": {false, true},
	}
	for status, want := range cases {
		d := synth(t, "0007", head("0007", "Gamma — a title", status)+"\n## Problem Statement\n\nSynthetic.\n")
		s := Summarize(d)
		if s.ShortTitle != "Gamma — a title" {
			t.Errorf("%q: short title %q", status, s.ShortTitle)
		}
		if s.Status == nil {
			t.Fatalf("%q: no status read", status)
		}
		if s.InFlight != want.inFlight || s.Terminal != want.terminal {
			t.Errorf("%q: in-flight %v terminal %v; want %v %v", status, s.InFlight, s.Terminal, want.inFlight, want.terminal)
		}
		if s.Priority != "High" || s.Type != "Feature" || s.Profile != "standard" {
			t.Errorf("%q: metadata not read: %+v", status, s)
		}
		if s.Hash == "" {
			t.Errorf("%q: no record hash", status)
		}
	}
}

// TestSameAnchor: one symbol, spelled with a bare file, a full path or a
// package, is one anchor; a different symbol or a non-suffix path is not.
func TestSameAnchor(t *testing.T) {
	yes := [][2]string{
		{"uniqueid.go::checkUniqueNameIn", "internal/validate/uniqueid.go::checkUniqueNameIn"},
		{"frame::Encode", "frame::Encode"},
	}
	no := [][2]string{
		{"uniqueid.go::checkUniqueNameIn", "uniqueid.go::checkUniqueName"},
		{"validate/uniqueid.go::Check", "cli/uniqueid.go::Check"},
		{"id.go::Check", "uniqueid.go::Check"}, // suffix, but not at a component boundary
	}
	for _, p := range yes {
		if !sameAnchor(p[0], p[1]) {
			t.Errorf("%q and %q should be one anchor", p[0], p[1])
		}
	}
	for _, p := range no {
		if sameAnchor(p[0], p[1]) {
			t.Errorf("%q and %q should be distinct", p[0], p[1])
		}
	}
}

// evidence writes an assumption whose Evidence cites the given anchors —
// the form Source Search evidence takes.
func evidence(anchors ...string) string {
	return "\n## Critical Assumptions\n\n- **A1**: something\n  - **Method**: Source Search\n  - **Evidence**: `" +
		strings.Join(anchors, "`, `") + "`\n  - **Status**: Verified\n"
}

// TestAnchorIntersectFiresOnUncitedOverlap is the pege validation case in
// miniature: two Drafts rewriting one function, neither naming the other,
// is reported first and marked uncited; a pair that cross-cites is
// reported but not flagged; the template's own `path::Symbol` is ignored;
// an implemented record is out of scope unless asked for.
func TestAnchorIntersectFiresOnUncitedOverlap(t *testing.T) {
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+evidence("uniqueid.go::checkUniqueNameIn", "path::Symbol")),
		synth(t, "0002", head("0002", "Beta", "Draft")+evidence("internal/validate/uniqueid.go::checkUniqueNameIn", "path::Symbol")),
		synth(t, "0003", head("0003", "Gamma", "Final")+evidence("other.go::Walk")+"\nSee 0004-delta for the split.\n"),
		synth(t, "0004", head("0004", "Delta", "Final")+evidence("pkg/other.go::Walk")),
		synth(t, "0005", head("0005", "Epsilon", "Implemented")+evidence("uniqueid.go::checkUniqueNameIn")),
	}
	got := AnchorIntersect(docs, true)
	if len(got) != 2 {
		t.Fatalf("want 2 overlaps over in-flight records, got %d: %+v", len(got), got)
	}
	first := got[0]
	if first.A != "0001" || first.B != "0002" || first.Cited {
		t.Errorf("the uncited pair should lead, uncited: %+v", first)
	}
	if len(first.Anchors) != 1 || first.Anchors[0] != "internal/validate/uniqueid.go::checkUniqueNameIn" {
		t.Errorf("want the one shared symbol in its fullest spelling, no placeholder: %v", first.Anchors)
	}
	if second := got[1]; second.A != "0003" || !second.Cited {
		t.Errorf("the cross-cited pair should be reported and marked cited: %+v", second)
	}
	all := AnchorIntersect(docs, false)
	if len(all) != 4 {
		t.Errorf("--all should add the implemented record's two overlaps: got %d", len(all))
	}
}

// TestReadmeDriftIsACheck: every disagreement class is named and nothing
// is judged — a row is compared on the cells the table carries, an
// escaped pipe stays inside its cell, and a matching table yields nothing.
func TestReadmeDriftIsACheck(t *testing.T) {
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Final")+"\n## Problem Statement\n\nSynthetic.\n"),
		synth(t, "0002", head("0002", "Beta `a|b`", "Draft [revised]")+"\n## Problem Statement\n\nSynthetic.\n"),
		synth(t, "0003", head("0003", "Gamma", "Implemented")+"\n## Problem Statement\n\nSynthetic.\n"),
	}
	readme := strings.Split(`# Records

| ID | Title | Status | Priority |
| --- | --- | --- | --- |
| [0001](0001-alpha.md) | Alpha | Final | High |
| [0002](0002-beta.md) | Beta `+"`a\\|b`"+` | Draft (unblocked) | High |
| [0009](0009-zeta.md) | Zeta | Final | Low |
`, "\n")
	rows := ParseReadmeIndex(readme)
	if len(rows) != 3 || rows[1].Title != "Beta `a|b`" || rows[1].Status != "Draft (unblocked)" {
		t.Fatalf("rows misread: %+v", rows)
	}
	drift := ReadmeDrift(docs, rows)
	want := map[string]string{"0003": "missing-row", "0009": "extra-row"}
	if len(drift) != len(want) {
		t.Fatalf("want %d drift, got %+v", len(want), drift)
	}
	for _, d := range drift {
		if want[d.Record] != d.Kind {
			t.Errorf("unexpected drift %+v", d)
		}
	}

	stale := ParseReadmeIndex([]string{"| [0001](0001-alpha.md) | Alpha | Implemented | Low |"})
	got := ReadmeDrift(docs[:1], stale)
	kinds := map[string]bool{}
	for _, d := range got {
		kinds[d.Kind] = true
	}
	if !kinds["status"] || !kinds["priority"] || kinds["title"] {
		t.Errorf("want status and priority drift only: %+v", got)
	}
}

// TestGraphDerivesBacklinks: the graph's reverse edges are a transposition
// of its forward edges, so every edge appears under its target exactly
// once and the record count is the corpus.
func TestGraphDerivesBacklinks(t *testing.T) {
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Final")+"- **Predecessors**: 0002-beta\n\n## Problem Statement\n\nSee 0002-beta.\n"),
		synth(t, "0002", head("0002", "Beta", "Final")+"\n## Problem Statement\n\nSynthetic.\n"),
	}
	g := BuildGraph(docs, nil)
	if len(g.Records) != 2 || g.Skipped == nil {
		t.Fatalf("records %d skipped %v", len(g.Records), g.Skipped)
	}
	n := 0
	for _, refs := range g.Backlinks {
		n += len(refs)
	}
	if n != len(g.Edges) || n == 0 {
		t.Errorf("backlinks (%d) must transpose edges (%d)", n, len(g.Edges))
	}
	if len(g.Backlinks["0002"]) == 0 {
		t.Errorf("0002 has inbound edges and no backlink entry: %v", g.Backlinks)
	}
}
