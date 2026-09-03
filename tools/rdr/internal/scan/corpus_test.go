package scan

import (
	"strings"
	"testing"
)

// head is a synthetic record head with the given status, in the shape that
// carries labelled contracts.
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

// TestReadmeRowNeedsNoLink: an unlinked `| NNNN |` row is a row. The link
// is the convention; the number is the key. Read as no row at all, the
// record reports `none` and `readme --add` appends a duplicate beside the
// row already there — and `index --readme` calls it `missing-row`.
func TestReadmeRowNeedsNoLink(t *testing.T) {
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Final")+"\n## Problem Statement\n\nSynthetic.\n"),
		synth(t, "0002", head("0002", "Beta", "Draft")+"\n## Problem Statement\n\nSynthetic.\n"),
	}
	rows := ParseReadmeIndex(strings.Split(`# Records

| ID | Title | Status | Priority |
| --- | --- | --- | --- |
| 0001 | Alpha | Final | High |
| [0002](0002-beta.md) | Beta | Draft | High |
`, "\n"))
	if len(rows) != 2 || rows[0].Record != "0001" || rows[0].Status != "Final" || rows[0].Title != "Alpha" {
		t.Fatalf("the unlinked row is a row: %+v", rows)
	}
	if drift := ReadmeDrift(docs, rows); len(drift) != 0 {
		t.Errorf("a table that agrees drifts on nothing: %+v", drift)
	}
	// The header and the rule carry no number and are not rows.
	if got := ParseReadmeIndex([]string{"| ID | Title |", "| --- | --- |", "| 12 | short |"}); len(got) != 0 {
		t.Errorf("only a four-digit first cell is a row: %+v", got)
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

// contracts renders a Normative Contracts section holding one contract
// per body, which is what mints the C elements LiteralIntersect reads.
func contracts(bodies ...string) string {
	out := "\n## Proposed Solution\n\n### Technical Design\n\n#### Normative Contracts\n"
	for i, b := range bodies {
		out += "\n**C" + string(rune('1'+i)) + "**\n```normative\n" + b + "\n```\n"
	}
	return out
}

// TestLiteralIntersectFiresOnSharedContractLiterals is the contract arm of
// the joint-decision check in miniature, and it pins the three rules that
// make it a signal rather than a word count: a shared literal inside two
// contracts is reported and ranked uncited-first; TEMPLATE.md's own
// literals never link a pair; and a literal most of the scope carries is
// vocabulary, not coupling.
func TestLiteralIntersectFiresOnSharedContractLiterals(t *testing.T) {
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+
			contracts("refuse with `dml-data-conflict` when the seed is `occupied`")),
		synth(t, "0002", head("0002", "Beta", "Draft")+
			contracts("emit `dml-data-conflict` and stop")),
		synth(t, "0003", head("0003", "Gamma", "Draft")+
			contracts("the `snapshot` is written")+"\nSee 0004-delta.\n"),
		synth(t, "0004", head("0004", "Delta", "Draft")+
			contracts("the `snapshot` is read")),
		synth(t, "0005", head("0005", "Epsilon", "Implemented")+
			contracts("refuse with `dml-data-conflict`")),
	}
	got := LiteralIntersect(docs, true)
	if len(got) == 0 {
		t.Fatal("two in-flight contracts naming one error code must be reported")
	}
	first := got[0]
	if first.A != "0001" || first.B != "0002" || first.Cited {
		t.Errorf("the uncited pair sharing an error code should lead: %+v", first)
	}
	if len(first.Anchors) != 1 || first.Anchors[0] != "dml-data-conflict" {
		t.Errorf("want the one shared literal, got %v", first.Anchors)
	}
	for _, o := range got {
		if o.A == "0003" && o.B == "0004" && !o.Cited {
			t.Errorf("0003 names 0004 in prose, so their pair is cited: %+v", o)
		}
	}
	// Scope: an implemented record joins only when asked for.
	for _, o := range got {
		if o.A == "0005" || o.B == "0005" {
			t.Errorf("an implemented record is out of the in-flight scope: %+v", o)
		}
	}
	if all := LiteralIntersect(docs, false); len(all) <= len(got) {
		t.Errorf("--all must widen past in-flight: %d vs %d", len(all), len(got))
	}
}

// TestLiteralIntersectIgnoresTemplateLiterals pins the one subtraction
// the facet makes. TEMPLATE.md's own literals must never link a pair:
// every record that kept its guidance carries them, so counting them
// would report the schema as a coupling and fire on the whole corpus at
// once.
//
// There is deliberately no vocabulary filter to test — see minLiteral's
// comment for what was tried, measured and removed.
func TestLiteralIntersectIgnoresTemplateLiterals(t *testing.T) {
	// `path::Symbol` is TEMPLATE.md's own example anchor. Two records
	// sharing only that share the schema, not a decision.
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+contracts("anchored at `path::Symbol`")),
		synth(t, "0002", head("0002", "Beta", "Draft")+contracts("anchored at `path::Symbol`")),
	}
	if got := LiteralIntersect(docs, true); len(got) != 0 {
		t.Errorf("a template literal must not link a pair: %+v", got)
	}

	// A literal shorter than minLiteral is a word, not a decision.
	short := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+contracts("takes `-q`")),
		synth(t, "0002", head("0002", "Beta", "Draft")+contracts("takes `-q`")),
	}
	if got := LiteralIntersect(short, true); len(got) != 0 {
		t.Errorf("a sub-minLiteral token must not link a pair: %+v", got)
	}
}

// TestLiteralIntersectReadsOnlyContracts pins the element scope. The arm
// is about CONTRACTS — the surfaces a record owns and an implementer must
// match — and a literal in a Critical Assumption or an Alternative is not
// one. Left unpinned this is silent: the facet would still fire on real
// couplings while also firing on every record that quotes the same symbol
// in its evidence, and a reader cannot tell the two apart from the
// output.
func TestLiteralIntersectReadsOnlyContracts(t *testing.T) {
	// The shared literal lives in an assumption's Evidence, not a
	// contract, so there is nothing for this facet to intersect on.
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+evidence("shared.go::sharedSymbol")),
		synth(t, "0002", head("0002", "Beta", "Draft")+evidence("shared.go::sharedSymbol")),
	}
	if got := LiteralIntersect(docs, true); len(got) != 0 {
		t.Errorf("a literal outside a contract must not link a pair — that is --anchor-intersect's question: %+v", got)
	}
	// The same two records DO fire once the literal is in a contract,
	// which proves the fixture above is silent for the right reason.
	withContract := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+contracts("refuse with `shared-code`")),
		synth(t, "0002", head("0002", "Beta", "Draft")+contracts("emit `shared-code`")),
	}
	if got := LiteralIntersect(withContract, true); len(got) != 1 {
		t.Fatalf("the same literal inside a contract must fire, got %d", len(got))
	}
}

// TestLiteralIntersectWideningOnlyAdds pins scope monotonicity. `--all`
// must be a superset of the in-flight report: a fire that disappears when
// the reader looks at MORE records is a filter reaching backwards, and an
// earlier draft of this facet did exactly that — it derived a noise
// ceiling from the scoped set, so widening moved the denominator and
// suppressed a pair the narrow scope had reported.
func TestLiteralIntersectWideningOnlyAdds(t *testing.T) {
	docs := []*Document{
		synth(t, "0001", head("0001", "Alpha", "Draft")+contracts("refuse with `dml-data-conflict`")),
		synth(t, "0002", head("0002", "Beta", "Draft")+contracts("emit `dml-data-conflict`")),
		synth(t, "0003", head("0003", "Gamma", "Implemented")+contracts("logs `dml-data-conflict`")),
		synth(t, "0004", head("0004", "Delta", "Implemented")+contracts("ignores `dml-data-conflict`")),
	}
	open := LiteralIntersect(docs, true)
	all := LiteralIntersect(docs, false)
	seen := map[string]bool{}
	for _, o := range all {
		seen[o.A+"/"+o.B] = true
	}
	for _, o := range open {
		if !seen[o.A+"/"+o.B] {
			t.Errorf("pair %s/%s fires in-flight but vanishes under --all", o.A, o.B)
		}
	}
	if len(all) <= len(open) {
		t.Errorf("--all must widen past in-flight: %d vs %d", len(all), len(open))
	}
}
