package scan

import (
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// This file is the typed-edge pass: it reads every relation an RDR
// states — to other records, to their elements, to code, to artifacts and
// to trackers — out of the syntaxes the corpus writes them in, and emits
// each as one typed edge.
//
// The pass runs after fields(), because most edges are read out of a
// classified field's value rather than off a raw line: the Metadata
// block's Predecessors / Overrides / Cluster, an assumption's Method and
// Evidence, a Status qualifier already split into {value, qualifier,
// form}. Reading the classified value is what makes the extraction
// tolerant of wrapped values and label variants for free.
//
// It is a READER, in the package's sense: it never judges whether an
// edge should exist, only records the ones written. Resolution — does the
// target exist? — is a separate pass over a corpus (edges_resolve.go),
// because one file cannot answer it.

// Edge is one typed relation from this record, or from one of its
// elements, to a target.
type Edge struct {
	// From is the element ID the edge leaves, or the document ID when the
	// relation is the record's own (a Metadata field, a status qualifier).
	From string `json:"from"`
	// To is the target: an element or document ID for a record relation,
	// a `path::Symbol` for code, a path for an artifact, a tracker or RFD
	// id otherwise. Kind.Class() says which.
	To string `json:"to"`
	// Kind is the typed relation.
	Kind edge.Kind `json:"kind"`
	// Resolved is whether the target was found: true, false, or absent
	// when nothing checked it. A single-file inspect with no records dir
	// leaves it absent rather than guessing — an unchecked edge must
	// never read as a broken one, nor as a sound one.
	Resolved *bool `json:"resolved,omitempty"`
	// Line is where the reference is written; LineEnd is the last line
	// of the field or clause it was read from, so an unresolved edge
	// hands its reader a RANGE to open rather than a point. A dangling
	// reference is fixed by a minimal pointer correction in the record,
	// and the range is what says where.
	Line    int `json:"line"`
	LineEnd int `json:"line_end"`
	// Evidence is the text the edge was read from, trimmed: the citation
	// itself for a reference, the whole clause for a qualifier.
	Evidence string `json:"evidence"`
	// Field is the label of the field the edge came from (`Predecessors`,
	// `Evidence`, `Status`), or "" when it was read from prose.
	Field string `json:"field,omitempty"`
	// Quoted marks a reference read inside a closed double-quote pair: the
	// text is a verbatim quotation, so the reference is someone else's
	// words rather than this record's own citation. A quoted reference is
	// recorded weakly (see assumptionEdges) and no rule may ask the author
	// to rewrite it — a quotation is not the author's to restyle.
	Quoted bool `json:"quoted,omitempty"`
	// Slug is the filename slug the author wrote alongside the number, so
	// the resolver can report a reference whose number and slug name two
	// different records.
	Slug string `json:"slug,omitempty"`
}

// edges is the extraction pass. Order matters only in that the typed
// passes run before mentions, which claims what they left.
func (d *Document) edges() {
	d.Edges = []Edge{}
	claimed := map[int][][2]int{}
	d.fieldLines = d.fieldOwnedLines()
	d.metadataEdges(claimed)
	d.qualifierEdges(claimed)
	d.jointCheckEdges(claimed)
	d.assumptionEdges(claimed)
	d.contractEdges(claimed)
	d.crossCuttingEdges(claimed)
	d.anchorEdges(claimed)
	d.mentionEdges(claimed)
	d.unmappedWarnings(claimed)
	sort.SliceStable(d.Edges, func(i, j int) bool {
		if d.Edges[i].Line != d.Edges[j].Line {
			return d.Edges[i].Line < d.Edges[j].Line
		}
		return d.Edges[i].To < d.Edges[j].To
	})
}

// fieldOwnedLines is the set of lines a field pass reads by value: the
// whole Metadata block, and every field of every element. The body
// passes skip them.
//
// A field is read by NAME, with its wrapped value joined, which is what
// makes the extraction tolerant of wrapping and label variants. Reading
// those same lines again as raw prose would duplicate every edge the
// field pass found — its offsets are into the joined value and cannot be
// compared against a line's — and would mint false ones besides: a
// `- **Date**: 2026-07-30` line offers a bare `2026` to any pass that
// reads it as prose. So the field pass owns its lines outright.
func (d *Document) fieldOwnedLines() map[int]bool {
	out := map[int]bool{}
	mark := func(start, end int) {
		for i := start; i <= end; i++ {
			out[i] = true
		}
	}
	if n := d.metadataNode(); n != nil {
		mark(n.LineStart, n.LineEnd)
	}
	for _, e := range d.Elements {
		for _, f := range e.Fields {
			mark(f.LineStart, f.LineEnd)
		}
	}
	for _, f := range d.Fields {
		mark(f.LineStart, f.LineEnd)
	}
	return out
}

// docID is the record's own ID: the `from` of every relation the record
// states in its own name rather than through an element.
func (d *Document) docID() string {
	if d.Project != "" {
		return d.Project + "/" + d.Record
	}
	return d.Record
}

// addEdge records one edge and marks the span it read, so the mentions
// pass does not re-read the same citation weakly and the unmapped check
// knows what was covered.
//
// The same (from, to, kind, line) is one edge however many passes reach
// it. Claim spans cannot carry that on their own: a field pass reads a
// joined, wrapped VALUE and its offsets are into that value, while the
// body passes read a LINE — the two coordinate systems do not compare,
// and a citation on a field's first line is legitimately seen by both.
// Identity is the thing that actually holds, so identity is what dedupes.
// Two genuine occurrences of one anchor on one line stay two, because a
// field naming a symbol twice said it twice.
func (d *Document) addEdge(e Edge, claimed map[int][][2]int, span [2]int) {
	if e.To == "" {
		return
	}
	e.Evidence = strings.TrimSpace(e.Evidence)
	if e.LineEnd < e.Line {
		e.LineEnd = e.Line
	}
	for _, x := range d.Edges {
		if x.From == e.From && x.To == e.To && x.Kind == e.Kind && x.Line == e.Line && x.Field == e.Field {
			return
		}
	}
	d.Edges = append(d.Edges, e)
	if span[1] > span[0] {
		claimed[e.Line] = append(claimed[e.Line], span)
	}
}

// target renders a reference as this document's project would write it.
// A reference with no project prefix is local — the corpus omits the
// prefix inside one records dir — so it inherits this document's.
func (d *Document) target(r edge.Ref) string {
	if r.Project == "" {
		r.Project = d.Project
	}
	return r.ID()
}

// --- metadata -----------------------------------------------------------

// metadataFieldKinds maps each edge-bearing Metadata field onto its kind.
// These three fields are record LISTS: their whole value is references,
// so a bare four-digit number in them is a record, not a year.
var metadataFieldKinds = map[string]edge.Kind{
	"Predecessors": edge.Predecessor,
	"Overrides":    edge.Overrides,
	"Cluster":      edge.Cluster,
}

// metadataEdges reads Predecessors, Overrides and Cluster, plus the
// tracker and RFD references of Related Issues and the source anchors of
// Seam Lineage.
//
// Every one of these fields carries prose as well as references — an
// Overrides value explains what it narrows, a Predecessors value says
// which entries are load-bearing and sometimes names a record it is
// explicitly NOT a predecessor of. The pass reads every reference the
// value contains; deciding that a record named inside a negation is not
// really a predecessor is a judgment, and judgment is not this package's.
// The edge's Evidence carries the clause so a consumer can see the
// context the projector refused to interpret.
//
// Overrides is the one field with a SYNTAX for context: it is a run of
// clauses — `;`-separated or full sentences — each opening on the record
// it narrows and then explaining how, and the explanation names peers
// freely — `cli/0092's default rung is overridden … outside cli/0112's
// fold band` overrides 0092 and merely names 0112. So a clause's first
// reference mints the overrides edge and the rest mint mentions; that is
// reading the field's grammar, not judging its prose. Predecessors stays a list of targets — its
// clauses hold `cli/0004 + cli/0030` — and Cluster is a plain list.
func (d *Document) metadataEdges(claimed map[int][][2]int) {
	for _, f := range d.Metadata {
		if placeholderValue(f.Value) {
			// A seed names CANDIDATES, not declared relations, so it mints
			// no typed edge. It is still read: the candidates are recorded
			// as mentions, because a Metadata line is owned by this pass
			// and nothing else will see them — dropping them would be the
			// silent loss the whole projector is built to avoid.
			if metadataFieldKinds[f.Canonical] != "" || f.Canonical == "Related Issues" {
				for _, r := range edge.FindRefs(f.Value, false) {
					d.addEdge(Edge{From: d.docID(), To: d.target(r), Kind: edge.Mentions,
						Line: f.LineStart, LineEnd: f.LineEnd, Evidence: r.Raw, Field: f.Canonical, Slug: r.Slug},
						claimed, [2]int{r.Start, r.End})
				}
				d.issueEdges(d.docID(), f.Value, f.LineStart, f.LineEnd, f.Canonical, claimed)
			}
			continue
		}
		switch {
		case metadataFieldKinds[f.Canonical] != "":
			kind := metadataFieldKinds[f.Canonical]
			refs := edge.FindRefs(f.Value, true)
			for i, r := range refs {
				k := kind
				if kind == edge.Overrides && (r.Record == d.Record || !leadsClause(f.Value, refs, i)) {
					// A record cannot override itself — a self-reference in
					// the field's prose (`RDR NNNN adopts …`) is the record
					// speaking, not a supersession target.
					k = edge.Mentions
				}
				d.addEdge(Edge{From: d.docID(), To: d.target(r), Kind: k,
					Line: f.LineStart, LineEnd: f.LineEnd, Evidence: r.Raw, Field: f.Canonical, Slug: r.Slug},
					claimed, [2]int{r.Start, r.End})
			}
		case f.Canonical == "Related Issues":
			d.issueEdges(d.docID(), f.Value, f.LineStart, f.LineEnd, f.Canonical, claimed)
			for _, r := range edge.FindRefs(f.Value, false) {
				d.addEdge(Edge{From: d.docID(), To: d.target(r), Kind: edge.Mentions,
					Line: f.LineStart, LineEnd: f.LineEnd, Evidence: r.Raw, Field: f.Canonical, Slug: r.Slug},
					claimed, [2]int{r.Start, r.End})
			}
		case f.Canonical == "Seam Lineage":
			d.anchorsIn(d.docID(), f.Value, f.LineStart, f.LineEnd, f.Canonical, claimed)
			d.issueEdges(d.docID(), f.Value, f.LineStart, f.LineEnd, f.Canonical, claimed)
		}
	}
}

// leadsClause reports whether refs[i] opens its clause of value.
// The field's first reference always leads: the field is the statement.
// A later reference leads only when it is the first token of its
// clause — nothing but whitespace and light punctuation (emphasis,
// a bracket, a quote, a dash) between the boundary and the reference.
// A clause ends at `;` or at a sentence end: a supersession the author
// wrote as its own sentence (`…; contract narrowed here). cli/NNNN:A8 —
// its rejection is reassigned…`) opens on its record exactly as a
// `;`-clause does, and a full stop is the stronger separator. A `;`
// is prose too, and a peer named mid-sentence after one is a mention:
// `…; this field is the record. Also owed to a peer: cli/NNNN …` names
// a peer, it does not override it. FindRefs returns refs in offset
// order, so a boundary before the previous reference is not this
// clause's.
func leadsClause(value string, refs []edge.Ref, i int) bool {
	if i == 0 {
		return true
	}
	start := refs[i].Start
	b := clauseBoundary(value[:start])
	if b < refs[i-1].End {
		return false
	}
	return clauseLead(value[b+1 : start])
}

// clauseBoundary returns the index of the clause boundary nearest the
// reference: the last `;`, or a later sentence end — `.`, `!` or `?`
// followed by whitespace. A period inside a path or code token is
// followed by a letter, not whitespace, and bounds nothing.
func clauseBoundary(s string) int {
	b := strings.LastIndex(s, ";")
	for i := len(s) - 1; i > b; i-- {
		c := s[i]
		if c != '.' && c != '!' && c != '?' {
			continue
		}
		if i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\t' || s[i+1] == '\n') {
			return i
		}
	}
	return b
}

// ClauseLeadPunct is what may sit between a `;` and the reference that
// leads the clause without making the reference a mid-sentence mention.
const ClauseLeadPunct = " \t*_`([\"'\u201c\u2018\u2014\u2013-:,"

// clauseLead reports whether s carries no prose. The list joiner `and`
// is not prose: `A; B; and C` is one list of three. Neither is the
// author naming the edge type — `Also overrides cli/NNNN — reverses …`
// declares the override in so many words, while any other prose before
// the verb (`does not override`, `no longer overrides`) keeps the
// reference a mention.
func clauseLead(s string) bool {
	s = strings.TrimLeft(s, ClauseLeadPunct)
	for _, w := range []string{"and", "Also", "also", "Overrides", "overrides"} {
		if rest, ok := strings.CutPrefix(s, w); ok {
			s = strings.TrimLeft(rest, ClauseLeadPunct)
		}
	}
	return s == ""
}

// placeholderValue reports whether a metadata value is the template seed
// left unfilled — `_Draft placeholder — …_`, `[seed — …`, `(none — …)`.
// Reading references out of a seed would mint edges to the candidates an
// author has not yet confirmed, which is the opposite of what the field
// says. The candidates are still in the file, and the mentions pass
// reads them as what they are.
func placeholderValue(v string) bool {
	t := strings.ToLower(strings.TrimLeft(strings.TrimSpace(v), "_*`([ "))
	for _, p := range []string{"draft placeholder", "seed", "pending the decide", "none", "to be confirmed", "tbd", "n/a"} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

// issueEdges reads the tracker and RFD references out of a value.
func (d *Document) issueEdges(from, value string, line, lineEnd int, field string, claimed map[int][][2]int) {
	for _, m := range edge.IssueRe.FindAllStringSubmatchIndex(value, -1) {
		id := firstGroup(value, m)
		if id == "" {
			continue
		}
		// Issue targets are namespaced. `_issues/0022` and RDR 0022 are
		// different objects, and an un-namespaced key would merge their
		// backlinks into one list a consumer cannot separate.
		d.addEdge(Edge{From: from, To: "issue/" + id, Kind: edge.Issue, Line: line, LineEnd: lineEnd,
			Evidence: value[m[0]:m[1]], Field: field}, claimed, [2]int{m[0], m[1]})
	}
	for _, m := range edge.RFDRe.FindAllStringSubmatchIndex(value, -1) {
		id := firstGroup(value, m)
		if id == "" {
			continue
		}
		d.addEdge(Edge{From: from, To: "rfd/" + id, Kind: edge.RFD, Line: line, LineEnd: lineEnd,
			Evidence: value[m[0]:m[1]], Field: field}, claimed, [2]int{m[0], m[1]})
	}
}

func firstGroup(s string, m []int) string {
	for i := 2; i+1 < len(m); i += 2 {
		if m[i] >= 0 && m[i+1] > m[i] {
			return strings.TrimSpace(s[m[i]:m[i+1]])
		}
	}
	return ""
}

// --- status qualifiers --------------------------------------------------

// qualifierEdges reads the three status qualifiers that name a target:
// the joint-decision home the record waits on, the issue a demoted
// record was refiled as, and the self-edges to the assumptions a
// Final→Draft flip reopened.
func (d *Document) qualifierEdges(claimed map[int][][2]int) {
	for _, f := range d.Metadata {
		if f.Canonical != "Status" || f.Status == nil || f.Status.Qualifier == "" {
			continue
		}
		q := f.Status.Qualifier
		switch f.Status.Form {
		case model.QualifierJointDecision.String():
			m := model.JointDecisionGrammar.FindStringSubmatch(q)
			if m == nil {
				continue
			}
			for _, r := range edge.FindRefs(m[1], true) {
				d.addEdge(Edge{From: d.docID(), To: d.target(r), Kind: edge.JointDecisionHome,
					Line: f.LineStart, LineEnd: f.LineEnd, Evidence: q, Field: "Status", Slug: r.Slug}, claimed, [2]int{})
			}
		case model.QualifierDemotedTarget.String():
			m := model.DemotedTargetGrammar.FindStringSubmatch(q)
			if m == nil {
				continue
			}
			d.demotedEdge(m[1], f.LineStart, q, claimed)
		case model.QualifierRevisedFrom.String():
			m := model.RevisedFromGrammar.FindStringSubmatch(q)
			if m == nil || m[2] == "" {
				continue
			}
			// `re-verify A2,A4` names this record's own assumptions: the
			// edge is a self-edge, which is exactly what makes it useful
			// — it is the list a scoped re-verify stage must work through.
			for _, a := range strings.Split(m[2], ",") {
				key := strings.TrimPrefix(strings.TrimSpace(a), "A")
				if key == "" {
					continue
				}
				d.addEdge(Edge{From: d.docID(), To: d.id(ident.Assumption, key),
					Kind: edge.Reverify, Line: f.LineStart, LineEnd: f.LineEnd, Evidence: q, Field: "Status"}, claimed, [2]int{})
			}
		}
	}
}

// demotedEdge reads a `Demoted [→ <target>]` destination, which the
// corpus writes as an issue, a record, or a path.
func (d *Document) demotedEdge(target string, line int, evidence string, claimed map[int][][2]int) {
	target = strings.TrimSpace(target)
	if m := edge.IssueRe.FindStringSubmatchIndex(target); m != nil {
		if id := firstGroup(target, m); id != "" {
			d.addEdge(Edge{From: d.docID(), To: "issue/" + id, Kind: edge.MovedTo, Line: line,
				Evidence: evidence, Field: "Status"}, claimed, [2]int{})
			return
		}
	}
	if refs := edge.FindRefs(target, true); len(refs) > 0 {
		d.addEdge(Edge{From: d.docID(), To: d.target(refs[0]), Kind: edge.MovedTo, Line: line,
			Evidence: evidence, Field: "Status", Slug: refs[0].Slug}, claimed, [2]int{})
		return
	}
	// A destination in no recognised form is still the destination: the
	// edge exists and the target is what the author wrote.
	d.addEdge(Edge{From: d.docID(), To: target, Kind: edge.MovedTo, Line: line,
		Evidence: evidence, Field: "Status"}, claimed, [2]int{})
}

// --- joint checks -------------------------------------------------------

// jointCheckEdges reads the home and the fired targets of every
// `Joint-check:` line that is not `clear`.
//
// The propose gate checks a fire's home "as an edge, not a string": a
// `joint-decision-home` edge that resolved true, where false or absent is
// not a pass. Until this pass the line had no such edge — the kind was
// minted only from the Status qualifier, and the line's references fell
// through to mentions — so the gate's own reading was always absent. The
// home is `|`-segmented (`cli/0113 §R-2 | OPEN`); each segment naming a
// reference mints one home edge whose Evidence is the segment, so a
// reader can count the segments that homed against the segments written.
// OPEN mints nothing: an open home is a value on the element
// (JointCheck.Open), not a target. The fired targets (`fired → 0113`)
// mint mentions, which is what makes an intersect's `cited` honest for a
// fire the author wrote.
func (d *Document) jointCheckEdges(claimed map[int][][2]int) {
	for i := range d.Elements {
		el := &d.Elements[i]
		if el.Kind != ident.JointCheck || el.Joint == nil || el.Joint.Verdict == "clear" {
			continue
		}
		line := el.LineStart
		raw := d.Line(line)
		m := jointCheckLine.FindStringSubmatchIndex(raw)
		if m == nil {
			continue
		}
		head, headEnd := m[4], m[5]
		if h := jointHome.FindStringSubmatchIndex(raw); h != nil {
			headEnd = h[0]
			off := h[2]
			for _, seg := range strings.Split(raw[h[2]:h[3]], "|") {
				start := off
				off += len(seg) + 1
				t := strings.TrimSpace(seg)
				if t == "" || strings.HasPrefix(strings.ToUpper(t), "OPEN") {
					continue
				}
				refs := homeRefs(seg)
				if len(refs) > 0 {
					el.Joint.Homes = append(el.Joint.Homes, t)
				}
				for _, r := range refs {
					d.addEdge(Edge{From: el.ID, To: d.target(r), Kind: edge.JointDecisionHome,
						Line: line, LineEnd: el.LineEnd, Evidence: t, Field: "Joint-check", Slug: r.Slug},
						claimed, [2]int{start + r.Start, start + r.End})
				}
			}
		}
		if headEnd < head {
			continue
		}
		targets := map[string]bool{}
		for _, t := range el.Joint.Targets {
			targets[t] = true
		}
		for _, r := range edge.FindRefs(raw[head:headEnd], true) {
			if !targets[r.Raw] || r.Record == d.Record {
				continue // as mentionEdges: the record naming itself is not a relation
			}
			d.addEdge(Edge{From: el.ID, To: d.target(r), Kind: edge.Mentions,
				Line: line, LineEnd: el.LineEnd, Evidence: r.Raw, Field: "Joint-check", Slug: r.Slug},
				claimed, [2]int{head + r.Start, head + r.End})
		}
	}
}

// homeRefs reads the references of one home segment. The segment IS a
// reference by the line's grammar, so a bare number there is a record
// WITH its element half — `0033 §Normative Contracts` homes on the
// section, not the document. FindRefs' bare-number sweep was built for
// record lists and reads the number alone, so a bare segment is re-read
// behind the marker the reference grammar wants, and the offsets are
// shifted back onto the segment.
func homeRefs(seg string) []edge.Ref {
	if refs := edge.FindRefs(seg, false); len(refs) > 0 {
		return refs
	}
	const marker = "RDR "
	t := strings.TrimLeft(seg, " \t")
	lead := len(seg) - len(t)
	var out []edge.Ref
	for _, r := range edge.FindRefs(marker+t, false) {
		r.Start, r.End = max(r.Start-len(marker), 0)+lead, r.End-len(marker)+lead
		r.Raw = seg[r.Start:r.End]
		out = append(out, r)
	}
	return out
}

// --- assumptions --------------------------------------------------------

// assumptionEdges reads each assumption's Evidence. The kind depends on
// the assumption's Method: a `Peer RDR` assumption's record citations are
// peer-evidence, the claim's own support; the same citation under
// `Source Search` is a mention. Source anchors and artifact paths are
// read under every method, because an Evidence line naming a symbol is a
// code edge whatever the method label says.
//
// A QUOTED TOKEN IS THE PEER SPEAKING, NOT THIS RECORD CITING. Evidence
// under `Peer RDR` quotes the peer's text verbatim, and the peer's own
// prose is full of record references — including the host record's own
// id. Promoting those to peer-evidence asserts a support relation the
// author never wrote, and the no-element rule then demands an element
// cite inside a quotation the author may not alter. So a reference whose
// whole span sits inside a closed double-quote pair stays a mention. The
// exemption never drops an edge — the relation is still recorded, weakly
// — and an unclosed quote exempts nothing, so a stray mark cannot
// silence the field behind it.
func (d *Document) assumptionEdges(claimed map[int][][2]int) {
	for i := range d.Elements {
		el := &d.Elements[i]
		if el.Kind != ident.Assumption {
			continue
		}
		peer := false
		for _, f := range el.Fields {
			if f.Canonical == "Method" && f.Method != nil {
				for _, m := range f.Method.Members {
					if strings.EqualFold(m, "Peer RDR") {
						peer = true
					}
				}
			}
		}
		for _, f := range el.Fields {
			if f.Canonical != "Evidence" && f.Canonical != "Method" && f.Canonical != "If wrong" {
				continue
			}
			kind := edge.Mentions
			var quoted [][2]int
			if peer && f.Canonical != "If wrong" {
				kind = edge.PeerEvidence
				quoted = quotedSpans(f.Value)
			}
			for _, r := range edge.FindRefs(f.Value, false) {
				k, q := kind, false
				if k == edge.PeerEvidence && inQuotes(quoted, r.Start, r.End) {
					k, q = edge.Mentions, true
				}
				d.addEdge(Edge{From: el.ID, To: d.target(r), Kind: k, Line: f.LineStart, LineEnd: f.LineEnd,
					Evidence: r.Raw, Field: f.Canonical, Slug: r.Slug, Quoted: q}, claimed, [2]int{r.Start, r.End})
			}
			d.anchorsIn(el.ID, f.Value, f.LineStart, f.LineEnd, f.Canonical, claimed)
			d.issueEdges(el.ID, f.Value, f.LineStart, f.LineEnd, f.Canonical, claimed)
		}
	}
}

// quotedSpans is the byte ranges of s enclosed in double quotes: straight
// pairs, matched left to right, and the curly `“ ”` pair, which carries
// its own direction. Only a CLOSED pair is a span — an opening quote the
// field never closes claims nothing. Single quotes are not read: in prose
// they are apostrophes far more often than quotation.
func quotedSpans(s string) [][2]int {
	var out [][2]int
	straight, curly := -1, -1
	for i, r := range s {
		switch r {
		case '"':
			if straight < 0 {
				straight = i
			} else {
				out = append(out, [2]int{straight, i + 1})
				straight = -1
			}
		case '“':
			if curly < 0 {
				curly = i
			}
		case '”':
			if curly >= 0 {
				out = append(out, [2]int{curly, i + len("”")})
				curly = -1
			}
		}
	}
	return out
}

// inQuotes reports whether the span [start,end) lies wholly inside one of
// the quoted spans.
func inQuotes(spans [][2]int, start, end int) bool {
	for _, sp := range spans {
		if start >= sp[0] && end <= sp[1] {
			return true
		}
	}
	return false
}

// --- contracts ----------------------------------------------------------

// contractEdges reads the Transient marker: the sibling record scheduled
// to delete a bridge-surface contract. A transient contract counts toward
// neither the Profile contract axis nor the split signal, so the record
// that retires it is a relation a consumer must be able to follow.
//
// The marker is read WHEREVER IT IS WRITTEN, not only inside a fence.
// TEMPLATE.md puts it in Normative Contracts and the corpus writes it as
// a blockquote under the fence it qualifies, which is outside the
// contract element's own range. The edge leaves the contract it annotates
// when one encloses it and the record otherwise; either way the relation
// is recorded, because a marker read as prose is a scheduled deletion
// nobody can query for.
func (d *Document) contractEdges(claimed map[int][][2]int) {
	for i := 1; i <= len(d.lines); i++ {
		m := model.TransientMarker.FindStringSubmatch(d.Line(i))
		if m == nil {
			continue
		}
		from := d.docID()
		if el := d.elementAt(i); el != nil && el.Kind == ident.Contract {
			from = el.ID
		} else if el := d.nearestContractAbove(i); el != "" {
			from = el
		}
		loc := model.TransientMarker.FindStringIndex(d.Line(i))
		for _, r := range edge.FindRefs(m[1], true) {
			d.addEdge(Edge{From: from, To: d.target(r), Kind: edge.TransientDeletedBy,
				Line: i, Evidence: strings.TrimSpace(m[0]), Slug: r.Slug},
				claimed, [2]int{loc[0], loc[1]})
		}
	}
}

// nearestContractAbove is the contract a marker written below a fence
// annotates: the last one to start at or before the marker's line, and
// only while no other contract has since opened.
func (d *Document) nearestContractAbove(line int) string {
	id := ""
	for _, el := range d.Elements {
		if el.Kind == ident.Contract && el.LineStart <= line {
			id = el.ID
		}
	}
	return id
}

// --- cross-cutting ------------------------------------------------------

// crossCuttingEdges reads the peer record named in Cross-Cutting
// Concerns. The template asks the author to state, for each concern,
// either how this RDR addresses it or "which peer RDR owns the
// project-wide policy this RDR conforms to" — the second form is an
// ownership edge, and it is one of the three relations the 7.1 cluster
// gate derives membership from.
func (d *Document) crossCuttingEdges(claimed map[int][][2]int) {
	for _, n := range d.canonicalNodes("Cross-Cutting Concerns") {
		for i := n.LineStart + 1; i <= n.LineEnd; i++ {
			if d.fenced[i-1] {
				continue
			}
			line := d.Line(i)
			for _, r := range edge.FindRefs(line, false) {
				d.addEdge(Edge{From: d.docID(), To: d.target(r), Kind: edge.CrossCuttingOwner,
					Line: i, Evidence: strings.TrimSpace(line), Slug: r.Slug},
					claimed, [2]int{r.Start, r.End})
			}
		}
	}
}

// --- anchors and artifacts ----------------------------------------------

// anchorEdges reads every `path::Symbol` and artifact path in the record
// body, outside the fields already read. A code anchor is the same edge
// wherever it is written — the Evidence line that owns it, the Approach
// paragraph that explains it — and the flow's grounding sweep reads them
// all.
func (d *Document) anchorEdges(claimed map[int][][2]int) {
	for i := 1; i <= len(d.lines); i++ {
		if d.fenced[i-1] || d.fieldLines[i] {
			continue
		}
		d.anchorsIn(d.sectionOwner(i), d.Line(i), i, i, "", claimed)
	}
}

// anchorsIn reads the source anchors and artifact paths of one string.
func (d *Document) anchorsIn(from, s string, line, lineEnd int, field string, claimed map[int][][2]int) {
	// The `::` test is cheap and necessary; the regexp's per-position
	// backtracking is not, and most lines carry no anchor.
	var anchors [][]int
	if strings.Contains(s, "::") {
		anchors = edge.SourceAnchorRe.FindAllStringSubmatchIndex(s, -1)
	}
	for _, m := range anchors {
		anchor := s[m[2]:m[3]] + "::" + s[m[4]:m[5]]
		if d.claimedAt(claimed, line, m[0]) || !edge.IsSourceAnchor(s[m[2]:m[3]]) {
			continue
		}
		d.addEdge(Edge{From: from, To: anchor, Kind: edge.SourceAnchor, Line: line, LineEnd: lineEnd,
			Evidence: strings.TrimSpace(s[m[0]:m[1]]), Field: field}, claimed, [2]int{m[0], m[1]})
	}
	for _, m := range edge.ArtifactRe.FindAllStringSubmatchIndex(s, -1) {
		if d.claimedAt(claimed, line, m[0]) {
			continue
		}
		path := "{" + s[m[2]:m[3]] + "}"
		if m[4] >= 0 && m[5] > m[4] {
			path = strings.TrimRight(path, "}") + "}/" + strings.TrimLeft(s[m[4]:m[5]], "/")
		}
		d.addEdge(Edge{From: from, To: path, Kind: edge.Artifact, Line: line, LineEnd: lineEnd,
			Evidence: strings.TrimSpace(s[m[0]:m[1]]), Field: field}, claimed, [2]int{m[0], m[1]})
	}
}

// sectionOwner is the element ID a body line belongs to, falling back to
// the document. A source anchor inside an assumption is that
// assumption's; one in a paragraph is the record's.
func (d *Document) sectionOwner(line int) string {
	if el := d.elementAt(line); el != nil {
		return el.ID
	}
	return d.docID()
}

func (d *Document) claimedAt(claimed map[int][][2]int, line, off int) bool {
	for _, c := range claimed[line] {
		if off >= c[0] && off < c[1] {
			return true
		}
	}
	return false
}

// --- mentions -----------------------------------------------------------

// mentionEdges is the weak kind: every record reference in the body that
// no typed pass claimed. 9,637 bare `cli/NNNN` mentions carry no stated
// relation, and folding them into the typed kinds would make every typed
// query wrong; dropping them would lose the only trace of most
// cross-references. So they are their own kind, and a consumer chooses.
//
// Bare four-digit numbers are NOT read here. In free prose four digits is
// a year, a byte count or a line number at least as often as a record,
// and a mentions edge to record `2026` from a date is a false relation
// that a query cannot tell from a true one.
func (d *Document) mentionEdges(claimed map[int][][2]int) {
	for i := 1; i <= len(d.lines); i++ {
		if d.fenced[i-1] || d.fieldLines[i] {
			continue
		}
		line := d.Line(i)
		for _, r := range edge.FindRefs(line, false) {
			if d.claimedAt(claimed, i, r.Start) {
				continue
			}
			if r.Record == d.Record && r.Kind == "" {
				continue // the record naming itself is not a relation
			}
			d.addEdge(Edge{From: d.sectionOwner(i), To: d.target(r), Kind: edge.Mentions,
				Line: i, Evidence: r.Raw, Slug: r.Slug}, claimed, [2]int{r.Start, r.End})
		}
	}
}

// --- unmapped forms -----------------------------------------------------

// unmappedWarnings is the acceptance criterion that keeps this file
// honest: a reference form no grammar above claims lands in the warnings
// channel with its line, never in silence. A syntax the corpus grows
// after this pass was written shows up as a warning rather than as an
// edge that is quietly absent.
//
// It runs only over the EDGE-BEARING METADATA FIELDS. Over the whole body
// it would fire on every arrow in every prose sentence and every `§` in
// every heading reference, and a warning channel that cries wolf is a
// warning channel nobody reads.
func (d *Document) unmappedWarnings(claimed map[int][][2]int) {
	for _, f := range d.Metadata {
		if metadataFieldKinds[f.Canonical] == "" && f.Canonical != "Related Issues" {
			continue
		}
		if placeholderValue(f.Value) {
			continue
		}
		for _, u := range edge.Unmapped(f.Value, claimed[f.LineStart]) {
			d.warn("edge:unmapped-reference", f.LineStart, f.LineEnd,
				"%s carries a reference form no edge grammar reads: %q", f.Canonical, u)
		}
	}
}
