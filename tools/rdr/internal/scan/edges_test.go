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
		{"naming the edge type leads", "cli/0092:A6 — narrows the rung; also overrides cli/0092:A4 and cli/0081 REQ-37", []string{"cli/0092:A4", "cli/0092:A6"}, []string{"cli/0081"}},
		{"light punctuation leads", "cli/0103 REQ-13; **cli/0092**'s default rung; — (cli/0120) REQ-6; and cli/0089 A5", []string{"cli/0089:A5", "cli/0092", "cli/0103", "cli/0120"}, nil},
		{"semicolon inside the first clause", "cli/0103 REQ-38 (A5 coalesce; scenario (i) → ONE) — its normalization survives; cli/0104 REQ-35 → multiset", []string{"cli/0103", "cli/0104"}, nil},
		// A clause also ends at a sentence boundary — `.` followed by
		// whitespace — so a supersession the author wrote as its own
		// sentence opens on its record; a period inside a code token
		// bounds nothing, and a sentence led by prose stays a mention.
		{"sentence-form supersession", "cli/0030 — the op was renamed (fix r1; contract narrowed here). cli/0032:A8 — its rejection is reassigned to this RDR", []string{"cli/0030", "cli/0032:A8"}, nil},
		{"bold sentence lead", "cli/0104 REQ-35 → multiset. **cli/0133's** closed guard set is WIDENED by one member", []string{"cli/0104", "cli/0133"}, nil},
		{"also-overrides sentence", "cli/0092:A6 — narrows the rung. Also overrides cli/0092:A4 — reverses the serialization choice", []string{"cli/0092:A4", "cli/0092:A6"}, nil},
		{"prose-led sentence stays a mention", "cli/0103 REQ-13's file grain. The guard identifier is the contract, and cli/0133 is Implemented", []string{"cli/0103"}, []string{"cli/0133"}},
		{"code period is not a boundary", "cli/0029's totality premise, which replay.go::doDecomposeTable enforces today and cli/0034's recognition premise restates", []string{"cli/0029"}, []string{"cli/0034"}},
		// A record cannot override itself: a self-reference opening a
		// sentence is the record speaking about its own choice.
		{"self-reference is a mention", "cli/0092's rung is narrowed. RDR 0113 adopts frozen labels for the rest", []string{"cli/0092"}, []string{"cli/0113"}},
		// Read as written: a sentence OPENING on a record it then
		// disclaims still mints the override — the grammar reads clause
		// structure, not the prose after the reference (a negation
		// guard measurably demotes true overrides whose explanation
		// contains an unquoted `not`). The edge carries its clause as
		// evidence, so a reader who follows it sees the disclaimer.
		{"negated sentence lead reads as written", "cli/0135 REQ-3, by named supersession. cli/0129 REQ-71 needs no supersession — its A3 pre-authorizes the flag", []string{"cli/0129", "cli/0135"}, nil},
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

// TestJointCheckLineMintsHomeAndTargetEdges: the propose gate checks a
// fire's home "as an edge, not a string", so the `(home: …)` of a
// `Joint-check:` line mints a joint-decision-home edge FROM THE JC
// ELEMENT, one per `|` segment naming a reference, with the segment as
// its evidence. The fired target is a mention (what makes an intersect's
// `cited` honest for a fire), OPEN mints nothing, the mentions pass does
// not re-mint the home reference weakly, and a bare `NNNN §Section` homes
// on the section rather than the document.
func TestJointCheckLineMintsHomeAndTargetEdges(t *testing.T) {
	head := "# Recommendation 0026: X\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Draft\n\n## Decision Rationale\n\n"
	d := Bytes([]byte(head+"Joint-check: fired → 0113 (home: cli/0113 §Normative Contracts | OPEN) — shared.\n"), Options{})

	homes := edgesOf(d, edge.JointDecisionHome)
	if len(homes) != 1 {
		t.Fatalf("want exactly one home edge, got %v", homes)
	}
	h := homes[0]
	if h.From != "0026:JC1" || h.To != "cli/0113:§normative-contracts" || h.Evidence != "cli/0113 §Normative Contracts" || h.Field != "Joint-check" {
		t.Errorf("home edge = %+v", h)
	}
	// The element records which segments named a reference: one of two.
	if jc := d.Elements[len(d.Elements)-1]; jc.Joint == nil || len(jc.Joint.Homes) != 1 || jc.Joint.Homes[0] != "cli/0113 §Normative Contracts" {
		t.Errorf("JC homes = %+v, want the one referencing segment", jc.Joint)
	}
	// Two segments homing on one section are one edge and two homes.
	two := Bytes([]byte(head+"Joint-check: fired → 0113 (home: cli/0113 §Normative Contracts S-3 | cli/0113 §Normative Contracts E-1) — split.\n"), Options{})
	if e := edgesOf(two, edge.JointDecisionHome); len(e) != 1 {
		t.Errorf("same-section segments minted %d edges, want 1 (identity dedupes)", len(e))
	}
	if jc := two.Elements[len(two.Elements)-1]; jc.Joint == nil || len(jc.Joint.Homes) != 2 {
		t.Errorf("same-section segments recorded %+v, want two homes", jc.Joint)
	}
	var mentions []Edge
	for _, e := range d.Edges {
		if e.Kind == edge.Mentions {
			mentions = append(mentions, e)
		}
	}
	if len(mentions) != 1 || mentions[0].From != "0026:JC1" || mentions[0].To != "0113" || mentions[0].Field != "Joint-check" {
		t.Errorf("want one mention of the fired target from the JC element, got %v", mentions)
	}
	for _, e := range d.Edges {
		if strings.Contains(strings.ToUpper(e.To), "OPEN") {
			t.Errorf("OPEN minted an edge: %+v", e)
		}
	}

	// A clear line mints nothing; a bare-number home carries its section.
	d = Bytes([]byte(head+"Joint-check: clear (3 peers; no shared anchor).\nJoint-check: fired → 0033 (home: 0033 §Normative Contracts) — pre-image.\n"), Options{})
	homes = edgesOf(d, edge.JointDecisionHome)
	if len(homes) != 1 || homes[0].From != "0026:JC2" || homes[0].To != "0033:§normative-contracts" {
		t.Errorf("bare home: %v", homes)
	}
	if !hasEdge(d, edge.Mentions, "0026:JC2", "0033") {
		t.Errorf("fired target not mentioned: %v", edgesOf(d, edge.Mentions))
	}
	for _, e := range d.Edges {
		if e.From == "0026:JC1" {
			t.Errorf("a clear line minted %+v", e)
		}
	}
}

// TestJDRQualifierDoesNotResolveAgainstARecord is the check DOCUMENT-TIERS.md
// §3 asked for before anything was built on the qualifier: the Status
// grammar parses `Final [joint decision → JDR 0001 §JD-18: …]`, but
// qualifierEdges hands the home text to edge.FindRefs — the RECORD grammar
// — so `JDR 0001` was read as record 0001 and `JDR cli/0001 §DX-13` as
// element `cli/0001`.
//
// The harm is a false positive, not a miss. Where a record 0001 exists the
// home resolved TRUE against a document that is not the registry, and the
// propose gate requires that edge to resolve before a record advances; so
// a lock could be cleared by the wrong document. Where it does not, the
// author is told record 0001 is unresolved rather than that the registry
// was never found.
//
// A `JDR` reference is therefore claimed by the JDR grammar first, and
// what remains for the record grammar is a reference with no `JDR` marker.
func TestJDRQualifierDoesNotResolveAgainstARecord(t *testing.T) {
	head := "# Recommendation 0042: X\n\n## Metadata\n\n- **Date**: 2026-09-16\n"
	for _, tc := range []struct{ name, status, to string }{
		{"bare number", "Final [joint decision → JDR 0001 §JD-18: the enforcer]", "jdr:0001:§jd-18"},
		{"project-qualified", "Final [joint decision → JDR cli/0001 §DX-13: the shape]", "jdr:cli/0001:§dx-13"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := Bytes([]byte(head+"- **Status**: "+tc.status+"\n\n## Problem Statement\n\nX.\n"), Options{})
			homes := edgesOf(d, edge.JointDecisionHome)
			if len(homes) != 1 {
				t.Fatalf("want one home edge, got %v", homes)
			}
			if got := homes[0].To; got != tc.to {
				t.Errorf("home target = %q, want %q — a JDR reference must never mint a record target", got, tc.to)
			}
		})
	}
}

// TestRFDReferenceDoesNotResolveAgainstARecord is TestJDRQualifierDoes
// NotResolveAgainstARecord's other half, and the same defect one tier
// over: `RFD 0004 §Decision 1` is four digits followed by an anchor — a
// record reference by shape — so every position that hands text to
// edge.FindRefs read it as record 0004.
//
// The harm is the JDR one exactly. On the reference corpus a Status
// qualifier homing on `RFD 0004 §3c` minted a joint-decision-home edge
// that resolved TRUE against cli/0004, a different document, and the
// propose gate reads that edge to clear a lock. `rfd/0004/README.md` is
// worse still: the path form parses as project `rfd`, record `0004`,
// which is a record target in a project that does not exist.
//
// An RFD reference is therefore claimed by the RFD grammar first, in
// every position FindRefs is called from, and what reaches the record
// grammar is a reference with no RFD marker.
func TestRFDReferenceDoesNotResolveAgainstARecord(t *testing.T) {
	head := "# Recommendation 0042: X\n\n## Metadata\n\n- **Date**: 2026-09-16\n"

	// The kata's example: a joint-decision home on an RFD.
	t.Run("status qualifier home", func(t *testing.T) {
		for _, tc := range []struct{ name, status, to string }{
			{"legacy decision", "Final [joint decision → RFD 0007 Decision 1: the band]", "rfd/0007:§decision-1"},
			{"section", "Final [joint decision → RFD 0004 §3c: how they compose]", "rfd/0004:§3c"},
			{"decision row", "Final [joint decision → RFD 0004 DX-13: the shape]", "rfd/0004:§dx-13"},
			{"principle", "Final [joint decision → RFD 0004 P-2: the rule]", "rfd/0004:§p-2"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				d := Bytes([]byte(head+"- **Status**: "+tc.status+"\n\n## Problem Statement\n\nX.\n"), Options{})
				homes := edgesOf(d, edge.JointDecisionHome)
				if len(homes) != 1 {
					t.Fatalf("want one home edge, got %v", homes)
				}
				if got := homes[0].To; got != tc.to {
					t.Errorf("home target = %q, want %q — an RFD reference must never mint a record target", got, tc.to)
				}
			})
		}
	})

	// The Joint-check line's home half, which reads its segments through
	// homeRefs rather than the qualifier grammar.
	t.Run("joint-check home", func(t *testing.T) {
		body := "# Recommendation 0026: X\n\n## Metadata\n\n- **Date**: 2026-08-01\n- **Status**: Draft\n\n" +
			"## Decision Rationale\n\nJoint-check: fired → 0113 (home: RFD 0004 §3c) — shared.\n"
		d := Bytes([]byte(body), Options{})
		homes := edgesOf(d, edge.JointDecisionHome)
		if len(homes) != 1 || homes[0].To != "rfd/0004:§3c" {
			t.Errorf("joint-check home = %v, want one edge to rfd/0004:§3c", homes)
		}
	})

	// The three record-list Metadata fields, whose bare-number sweep is
	// what read `RFD 0004` as `0004` in the first place.
	t.Run("record-list metadata", func(t *testing.T) {
		d := Bytes([]byte(head+"- **Status**: Draft\n- **Predecessors**: RFD 0004 §3c fixes the band.\n\n## Problem Statement\n\nX.\n"), Options{})
		for _, e := range d.Edges {
			if e.Field == "Predecessors" && !strings.HasPrefix(e.To, "rfd/") {
				t.Errorf("Predecessors minted %+v, want an rfd target only", e)
			}
		}
	})

	// A Peer-RDR assumption's Evidence, read with bare=false: the path
	// form is what fires here, since `rfd/0004` carries its own marker.
	t.Run("peer evidence", func(t *testing.T) {
		body := head + "- **Status**: Draft\n\n## Critical Assumptions\n\n" +
			"### A1: the band holds\n\n- **Method**: Peer RDR\n- **Evidence**: rfd/0004/README.md §3c states it.\n"
		d := Bytes([]byte(body), Options{})
		for _, e := range d.Edges {
			if e.Kind == edge.PeerEvidence || (e.Kind == edge.Mentions && strings.Contains(e.To, "0004")) {
				if !strings.HasPrefix(e.To, "rfd/") {
					t.Errorf("peer evidence minted %+v, want an rfd target", e)
				}
			}
		}
	})

	// Free prose: the mentions pass claims the RFD span before FindRefs
	// sees it, so the path form is not a record mention either.
	t.Run("prose mention", func(t *testing.T) {
		d := Bytes([]byte(head+"- **Status**: Draft\n\n## Problem Statement\n\nThe band is set by rfd/0004/README.md §3c.\n"), Options{})
		for _, e := range d.Edges {
			if e.Kind == edge.Mentions && strings.Contains(e.To, "0004") {
				t.Errorf("prose mention minted %+v, want the rfd edge only", e)
			}
		}
	})
}
