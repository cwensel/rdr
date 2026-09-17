package scan

import (
	"regexp"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// This file is the corpus level: the questions the flow asks of every
// record at once, answered over the projections instead of by opening
// every file. Each is a pure function of the scanned documents — nothing
// here reads a file or keeps state — so the answers are byte-deterministic
// for a given records dir, which is what lets a skill trust them without
// re-reading.

// Summary is one record as the corpus index carries it: the Metadata a
// worklist or an index table needs, and nothing that requires the body.
type Summary struct {
	Record string `json:"record"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	// ShortTitle is the title without its `Recommendation NNNN:` lead —
	// the form an index table writes.
	ShortTitle string `json:"short_title"`
	Lines      int    `json:"lines"`
	// Hash is the title node's content hash, which governs every line, so
	// it is the record's own: same bytes, same hash.
	Hash   string  `json:"hash"`
	Status *Status `json:"status,omitempty"`
	// Terminal is true after a status the record is never amended from;
	// InFlight is true for the two working statuses, Draft and Final.
	// Deferred and Rejected are neither: parked is not in flight.
	Terminal bool   `json:"terminal"`
	InFlight bool   `json:"in_flight"`
	Type     string `json:"type,omitempty"`
	Profile  string `json:"profile,omitempty"`
	Priority string `json:"priority,omitempty"`
	Counts   Counts `json:"counts"`
	Edges    int    `json:"edges"`
	Warnings int    `json:"warnings"`
}

// titleLead is the `Recommendation NNNN:` / `RDR NNNN:` lead a title
// carries and an index table drops.
var titleLead = regexp.MustCompile(`^(?:Recommendation|RDR|R)\s*[-#]?\s*\d{4}\s*[:—–-]\s*`)

// Summarize reads a record's Summary off its projection.
func Summarize(d *Document) Summary {
	s := Summary{
		Record: d.Record, Path: d.Path, Title: d.Title, Lines: d.Lines,
		ShortTitle: strings.TrimSpace(titleLead.ReplaceAllString(d.Title, "")),
		Counts:     d.Counts, Edges: len(d.Edges), Warnings: len(d.Warnings),
	}
	if len(d.Outline) > 0 {
		s.Hash = d.Outline[0].Hash
	}
	if f := d.MetadataField("Status"); f != nil && f.Status != nil {
		s.Status = f.Status
		s.Terminal = IsTerminal(f.Status.Value)
		s.InFlight = IsInFlight(f.Status.Value)
	}
	for _, m := range []struct {
		label string
		into  *string
	}{{"Type", &s.Type}, {"Profile", &s.Profile}, {"Priority", &s.Priority}} {
		if f := d.MetadataField(m.label); f != nil {
			*m.into = strings.TrimSpace(f.Value)
		}
	}
	return s
}

// MetadataField returns the record's Metadata field with the canonical
// label, or nil. Fields are matched as classified, so a label variant the
// model absorbed (`Status (as of lock)`) is found under its canonical name.
func (d *Document) MetadataField(canonical string) *Field {
	for i := range d.Metadata {
		if d.Metadata[i].Canonical == canonical {
			return &d.Metadata[i]
		}
	}
	return nil
}

// IsTerminal reports whether a status is one a record is never amended
// after (model.TerminalStatuses).
func IsTerminal(status string) bool {
	for _, t := range model.TerminalStatuses() {
		if strings.EqualFold(t, status) {
			return true
		}
	}
	return false
}

// IsInFlight reports whether a status names working state: Draft, or a
// Final that has not been implemented. This is rdr-status's no-arg
// worklist rule.
func IsInFlight(status string) bool {
	return strings.EqualFold(status, "Draft") || strings.EqualFold(status, "Final")
}

// --- the graph document ---------------------------------------------------

// GraphElement is one element with the record it belongs to, flattened
// out of its document for a corpus-wide query.
type GraphElement struct {
	Record string     `json:"record"`
	ID     string     `json:"id"`
	Kind   ident.Kind `json:"kind"`
	Label  string     `json:"label,omitempty"`
	Hash   string     `json:"hash"`
	Line   int        `json:"line_start"`
	End    int        `json:"line_end"`
	// Joint carries a JC element's parsed line, so the graph can answer
	// "which records hold an open joint check" without a per-record walk.
	Joint *JointCheck `json:"joint,omitempty"`
}

// GraphEdge is one edge with the record it leaves.
type GraphEdge struct {
	Record   string    `json:"record"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Kind     edge.Kind `json:"kind"`
	Resolved *bool     `json:"resolved,omitempty"`
	// Alias names the registry that answered this edge when the RFD
	// itself no longer holds the anchor — the same fact `inspect` emits.
	// Resolved says the citation is sound; Alias says why, and a reader
	// auditing a migration needs the why from the graph, not only from a
	// per-record call.
	Alias   string `json:"alias,omitempty"`
	Line    int    `json:"line"`
	LineEnd int    `json:"line_end"`
	Field   string `json:"field,omitempty"`
}

// BackRef is one inbound edge as the backlink table lists it.
type BackRef struct {
	From string    `json:"from"`
	Kind edge.Kind `json:"kind"`
}

// Graph is the one document the corpus index emits: every record, every
// element, every edge, and the reverse edges derived from them. A skill
// that would otherwise open the records dir reads this instead.
type Graph struct {
	Schema    string               `json:"schema"`
	Records   []Summary            `json:"records"`
	Elements  []GraphElement       `json:"elements"`
	Edges     []GraphEdge          `json:"edges"`
	Backlinks map[string][]BackRef `json:"backlinks"`
	Skipped   []Skip               `json:"skipped"`
}

// Skip is one file a corpus walk did not read, and why.
//
// The reason TRAVELS with the row rather than being supplied by the
// print site, because the walk now has more than one reason to skip: a
// file that is not a record, and a file it could not read at all. A
// print site that names the reason itself gets the second case wrong —
// it reports an unreadable file as "not an RDR", which is a confident
// wrong answer about a file nobody looked inside.
type Skip struct {
	// Target is the path that was not read.
	Target string `json:"target"`
	// Why is the reason, in the caller's words.
	Why string `json:"why"`
}

// BuildGraph assembles the graph over already-resolved documents.
func BuildGraph(docs []*Document, skipped []Skip) Graph {
	g := Graph{Schema: SchemaVersion, Records: []Summary{}, Elements: []GraphElement{},
		Edges: []GraphEdge{}, Backlinks: map[string][]BackRef{}, Skipped: skipped}
	if g.Skipped == nil {
		g.Skipped = []Skip{}
	}
	for _, d := range docs {
		g.Records = append(g.Records, Summarize(d))
		for _, e := range d.Elements {
			g.Elements = append(g.Elements, GraphElement{d.Record, e.ID, e.Kind, e.Label, e.Hash, e.LineStart, e.LineEnd, e.Joint})
		}
		for _, e := range d.Edges {
			g.Edges = append(g.Edges, GraphEdge{Record: d.Record, From: e.From, To: e.To,
				Kind: e.Kind, Resolved: e.Resolved, Alias: e.Alias,
				Line: e.Line, LineEnd: e.LineEnd, Field: e.Field})
			g.Backlinks[e.To] = append(g.Backlinks[e.To], BackRef{e.From, e.Kind})
		}
	}
	return g
}

// --- anchor intersection --------------------------------------------------

// Overlap is two records that cite the same code anchors. Cited is true
// when either record names the other by any edge at all — a citation of
// any kind means the authors know the records touch. An overlap with
// Cited false is the signal: two records rewriting one symbol without
// either acknowledging the other.
type Overlap struct {
	A, B    string    `json:"-"`
	Records [2]string `json:"records"`
	Anchors []string  `json:"anchors"`
	Cited   bool      `json:"cited"`
}

// anchorPlaceholder is TEMPLATE.md's own example anchor, copied into
// records verbatim when the legend was kept; it names no code.
const anchorPlaceholder = "path::Symbol"

// sameAnchor reports whether two anchors name one symbol. The symbol
// halves must be equal; the path halves match when one is a
// component-aligned suffix of the other, because authors write both
// `uniqueid.go::checkUniqueNameIn` and
// `internal/validate/uniqueid.go::checkUniqueNameIn` for the same
// function, and a bare package (`frame::Encode`) is the shortest form.
func sameAnchor(a, b string) bool {
	pa, sa, _ := strings.Cut(a, "::")
	pb, sb, _ := strings.Cut(b, "::")
	if sa != sb {
		return false
	}
	return samePath(pa, pb)
}

// samePath is the path half of the anchor rule, alone: equal, or one a
// component-aligned suffix of the other. The `/` in the suffix test is
// what makes it component-aligned — without it `validate.go` would match
// `revalidate.go`.
func samePath(pa, pb string) bool {
	if pa == pb {
		return true
	}
	long, short := pa, pb
	if len(short) > len(long) {
		long, short = short, long
	}
	return strings.HasSuffix(long, "/"+short)
}

// MatchesLocus reports whether a source anchor falls on a seam locus.
//
// A locus is either a full `path::Symbol` or a path prefix, and this is
// the ONE place that rule lives. It shares `samePath` with `sameAnchor`
// so seam membership and the overlap graph cannot drift apart: two
// records that the overlap graph says share an anchor must be on the
// same seam when a registry declares it, and a second copy of the path
// rule is how that stops being true.
//
// `area:*` is NOT a locus. A registry spells its area as the loci it
// stands for (jdr/README.md §Membership is derived), because the
// projector has no table mapping an area name to code and inventing one
// would be a guess.
func MatchesLocus(anchor, locus string) bool {
	if anchor == "" || locus == "" || strings.HasPrefix(locus, "area:") {
		return false
	}
	if lp, ls, ok := strings.Cut(locus, "::"); ok {
		// A full anchor locus: both halves, by the anchor rule.
		ap, as, _ := strings.Cut(anchor, "::")
		if ls != as {
			// The receiver-qualified fallback the resolver uses:
			// `Server.Encode` names `Encode` on a receiver, and a locus
			// naming the member matches the anchor naming the method.
			if i := strings.LastIndex(as, "."); i < 0 || as[i+1:] != ls {
				return false
			}
		}
		return samePath(ap, lp)
	}
	// A path-prefix locus: component-aligned against the anchor's path.
	//
	// It anchors at a COMPONENT BOUNDARY, not at the head of the string,
	// for the same reason `sameAnchor` accepts a short spelling: one
	// repo's `internal/cli/corpus.go` is another record's
	// `a/b/internal/cli/corpus.go`, and a locus that matched only at the
	// head would drop the second out of the seam silently. A seam "may
	// widen, never narrow" (jdr/README.md §Identity), and a matcher that
	// quietly fails to match is a narrowing nobody declared.
	//
	// The boundary is what keeps it honest: `internal/cli` never matches
	// `internal/clip`, and a bare `cli` never matches `vendor/x/cli`
	// unless the locus itself says `vendor/x/cli`.
	ap, _, _ := strings.Cut(anchor, "::")
	loc := strings.Trim(locus, "/")
	if ap == loc {
		return true
	}
	if strings.HasPrefix(ap, loc+"/") {
		return true
	}
	// Matching mid-path needs the locus to be unambiguous on its own, and
	// a single component is not: `cli` would claim `vendor/x/cli` and
	// every other `cli` dir in the tree. The projector's standing rule is
	// that an ambiguous reference stays unresolved rather than being
	// guessed at, so a one-component locus matches only at the head, and
	// a registry that means a nested dir spells enough of it to say so.
	if !strings.Contains(loc, "/") {
		return false
	}
	return strings.Contains(ap, "/"+loc+"/")
}

// dedupeAnchors drops every spelling of an anchor but the fullest, so a
// symbol cited as both `import.go::importValidate` and
// `internal/cli/import.go::importValidate` is listed once.
func dedupeAnchors(anchors []string) []string {
	var out []string
	for i, a := range anchors {
		shadowed := false
		for j, b := range anchors {
			if i != j && len(b) > len(a) && sameAnchor(a, b) {
				shadowed = true
				break
			}
		}
		if !shadowed {
			out = append(out, a)
		}
	}
	return out
}

// AnchorIntersect finds every pair of records sharing a source anchor.
// With openOnly, both records must be in flight: two implemented records
// sharing a symbol is history, and the question this answers — "who else
// is proposing to change this?" — is about the records still being
// written. It is pege's tier-1 scan: run after every propose, the pair
// that shares anchors with no cross-citation is the joint decision that
// otherwise surfaces five gate iterations later.
func AnchorIntersect(docs []*Document, openOnly bool) []Overlap {
	type rec struct {
		doc     *Document
		anchors []string
		cites   map[string]bool
	}
	var recs []rec
	for _, d := range docs {
		if openOnly {
			s := d.MetadataField("Status")
			if s == nil || s.Status == nil || !IsInFlight(s.Status.Value) {
				continue
			}
		}
		r := rec{doc: d, cites: map[string]bool{}}
		seen := map[string]bool{}
		for _, e := range d.Edges {
			if e.Kind == edge.SourceAnchor && e.To != anchorPlaceholder && !seen[e.To] {
				seen[e.To] = true
				r.anchors = append(r.anchors, e.To)
			}
			if t := recordOf(e.To); t != "" {
				r.cites[t] = true
			}
		}
		recs = append(recs, r)
	}
	var out []Overlap
	for i := range recs {
		for j := i + 1; j < len(recs); j++ {
			a, b := recs[i], recs[j]
			shared := map[string]bool{}
			for _, x := range a.anchors {
				for _, y := range b.anchors {
					if sameAnchor(x, y) {
						// Report the fuller spelling, so the reader gets
						// the path when either author wrote one.
						if len(y) > len(x) {
							x = y
						}
						shared[x] = true
					}
				}
			}
			if len(shared) == 0 {
				continue
			}
			anchors := make([]string, 0, len(shared))
			for s := range shared {
				anchors = append(anchors, s)
			}
			sort.Strings(anchors)
			anchors = dedupeAnchors(anchors)
			out = append(out, Overlap{A: a.doc.Record, B: b.doc.Record,
				Records: [2]string{a.doc.Record, b.doc.Record}, Anchors: anchors,
				Cited: a.cites[b.doc.Record] || b.cites[a.doc.Record]})
		}
	}
	sortOverlaps(out)
	return out
}

// sortOverlaps ranks a pair report: uncited first, then by how much is
// shared, then by pair — so the top of the report is the pair most likely
// to be a joint decision. Both intersection facets rank the same way,
// because a reader comparing an anchor fire with a literal fire is
// comparing two answers to one question.
func sortOverlaps(out []Overlap) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Cited != out[j].Cited {
			return !out[i].Cited
		}
		if len(out[i].Anchors) != len(out[j].Anchors) {
			return len(out[i].Anchors) > len(out[j].Anchors)
		}
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
}

// --- README index drift ---------------------------------------------------

// ReadmeRow is one row of a records dir's README index table.
type ReadmeRow struct {
	Record   string `json:"record"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Line     int    `json:"line"`
}

// readmeRow matches `| [NNNN](file) | Title | Status | Priority |` and
// the UNLINKED `| NNNN | Title | Status | Priority |` variant. The link
// is the convention, not the row: a hand-written or delinked row still
// says this record is indexed, and reading only the linked ones reports
// the record as having no row at all — which sends `readme --add` to
// append a second row beside the one already there, and `index --readme`
// to call it `missing-row`. The number is the key either way.
var readmeRow = regexp.MustCompile(`^\|\s*(?:\[(\d{4})\]\([^)]*\)|(\d{4}))\s*\|(.*)$`)

// ParseReadmeIndex reads the index table rows out of README lines. Only
// rows whose first cell is a record number are rows; the header, the
// rule and the prose are not.
func ParseReadmeIndex(lines []string) []ReadmeRow {
	var rows []ReadmeRow
	for i, l := range lines {
		m := readmeRow.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		num := m[1]
		if num == "" {
			num = m[2]
		}
		cells := splitCells(m[3])
		row := ReadmeRow{Record: num, Line: i + 1}
		cell := func(n int) string {
			if n < len(cells) {
				return strings.TrimSpace(cells[n])
			}
			return ""
		}
		row.Title, row.Status, row.Priority = cell(0), cell(1), cell(2)
		rows = append(rows, row)
	}
	return rows
}

// splitCells splits a table row on its unescaped pipes; `\|` is a
// literal pipe inside a cell.
func splitCells(s string) []string {
	var cells []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			b.WriteByte('|')
			i++
		case s[i] == '|':
			cells = append(cells, b.String())
			b.Reset()
		default:
			b.WriteByte(s[i])
		}
	}
	return append(cells, b.String())
}

// Drift is one disagreement between the README index and the records.
type Drift struct {
	Record string `json:"record"`
	// Kind is `missing-row` (a record the table lacks), `extra-row` (a row
	// with no record), `duplicate-row`, or the cell that disagrees:
	// `status`, `title`, `priority`.
	Kind   string `json:"kind"`
	Readme string `json:"readme,omitempty"`
	Actual string `json:"actual,omitempty"`
	Line   int    `json:"line,omitempty"`
}

// ReadmeDrift compares the index table with the records. It is a check,
// never a generator: the finding says what disagrees and the author
// decides which side is wrong. Status and Priority compare on their
// leading word, because the table abbreviates a qualified value
// (`Draft (unblocked — …)`) and the record spells it out; Title compares
// whitespace-collapsed against the record's short title.
func ReadmeDrift(docs []*Document, rows []ReadmeRow) []Drift {
	var out []Drift
	byRow := map[string]ReadmeRow{}
	for _, r := range rows {
		if _, dup := byRow[r.Record]; dup {
			out = append(out, Drift{Record: r.Record, Kind: "duplicate-row", Line: r.Line})
			continue
		}
		byRow[r.Record] = r
	}
	seen := map[string]bool{}
	for _, d := range docs {
		s := Summarize(d)
		seen[d.Record] = true
		r, ok := byRow[d.Record]
		if !ok {
			out = append(out, Drift{Record: d.Record, Kind: "missing-row", Actual: s.ShortTitle})
			continue
		}
		if s.Status != nil && !strings.EqualFold(leadWord(r.Status), s.Status.Value) {
			out = append(out, Drift{d.Record, "status", r.Status, s.Status.Value, r.Line})
		}
		if collapse(r.Title) != collapse(s.ShortTitle) {
			out = append(out, Drift{d.Record, "title", r.Title, s.ShortTitle, r.Line})
		}
		if s.Priority != "" && r.Priority != "" && !strings.EqualFold(leadWord(r.Priority), leadWord(s.Priority)) {
			out = append(out, Drift{d.Record, "priority", r.Priority, s.Priority, r.Line})
		}
	}
	for _, r := range rows {
		if !seen[r.Record] && byRow[r.Record].Line == r.Line {
			out = append(out, Drift{Record: r.Record, Kind: "extra-row", Readme: r.Title, Line: r.Line})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Record != out[j].Record {
			return out[i].Record < out[j].Record
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// leadWord is the first word of a value, letters only, so `Draft
// (unblocked…)` and `Draft [revised…]` both read as Draft.
func leadWord(s string) string {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && (s[end] >= 'A' && s[end] <= 'Z' || s[end] >= 'a' && s[end] <= 'z' || s[end] == '/') {
		end++
	}
	return s[:end]
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// LiteralIntersect finds every pair of records whose CONTRACTS share a
// backticked literal. It is the contract half of the joint-decision
// check, and the analogue of AnchorIntersect: same pair shape, same
// uncited-first ranking, same in-flight default.
//
// WHY NOT THE ELEMENT HASH. The obvious query — two contracts with the
// same content hash — was tried against the live corpus and finds
// nothing: 0 duplicate hashes across 316 contracts, because the hash is
// exact-text identity and no two authors write a contract the same way.
// The question the check actually asks is narrower and much commoner: do
// two open records name the same error code, flag, field or sentinel
// inside otherwise-different contract text. That is a shared TOKEN, not a
// shared document, and on the same corpus it finds 36 pairs of which 4
// are uncited — the fire shape the anchor arm produces.
//
// WHAT COUNTS AS A LITERAL. A backticked span inside a contract element,
// at least minLiteral bytes long, that TEMPLATE.md does not also write.
// The length floor drops `-q`-scale tokens that collide by accident; the
// template subtraction drops the schema's own vocabulary, which every
// record that kept its guidance carries and which would otherwise link
// every pair (model.TemplateLiterals).
func LiteralIntersect(docs []*Document, openOnly bool) []Overlap {
	tmpl := model.TemplateLiterals()
	type rec struct {
		doc      *Document
		literals map[string]bool
		cites    map[string]bool
	}
	var recs []rec
	for _, d := range docs {
		if openOnly {
			s := d.MetadataField("Status")
			if s == nil || s.Status == nil || !IsInFlight(s.Status.Value) {
				continue
			}
		}
		r := rec{doc: d, literals: map[string]bool{}, cites: map[string]bool{}}
		for _, e := range d.Elements {
			if e.Kind != ident.Contract {
				continue
			}
			for _, lit := range contractLiterals(d, e) {
				if len(lit) >= minLiteral && !tmpl[lit] {
					r.literals[lit] = true
				}
			}
		}
		for _, e := range d.Edges {
			if t := recordOf(e.To); t != "" {
				r.cites[t] = true
			}
		}
		if len(r.literals) == 0 {
			continue // nothing to intersect on
		}
		recs = append(recs, r)
	}
	var out []Overlap
	for i := range recs {
		for j := i + 1; j < len(recs); j++ {
			a, b := recs[i], recs[j]
			var shared []string
			for lit := range a.literals {
				if b.literals[lit] {
					shared = append(shared, lit)
				}
			}
			if len(shared) == 0 {
				continue
			}
			sort.Strings(shared)
			out = append(out, Overlap{A: a.doc.Record, B: b.doc.Record,
				Records: [2]string{a.doc.Record, b.doc.Record}, Anchors: shared,
				Cited: a.cites[b.doc.Record] || b.cites[a.doc.Record]})
		}
	}
	sortOverlaps(out)
	return out
}

// minLiteral is the shortest backticked span counted as a contract
// literal. Two records both writing `id` or `-q` share a word, not a
// decision; three bytes is where a token starts being specific enough
// that two contracts naming it are plausibly naming one thing.
//
// A FREQUENCY CEILING WAS TRIED HERE AND REMOVED. The idea was to drop a
// literal too many records carry — `snapshot`, `migrate`, `code` — as the
// corpus's vocabulary rather than a coupling. Measured, it does not earn
// its place. The breadth it would key on is shallow (the widest literal
// is in 13 of 144 records, 9%), so no threshold separates vocabulary from
// a real shared type: `BandAlways` sits in six records and IS the
// coupling, `code` sits in seven and is not. Worse, it changed no pair on
// the reference corpus while making scope widening LOSE a fire — `--all`
// moved the denominator and suppressed a pair the in-flight scope had
// reported.
//
// What is left is the two subtractions that are defensible on their own
// terms: a length floor, and TEMPLATE.md's own literals. Both say
// something true about the token regardless of corpus. The remaining
// breadth shows up as a longer evidence list on a pair that fires anyway
// — the FIRE is the pair, and the literals are what the reader checks it
// against — which costs a line to read and cannot hide a coupling.
const minLiteral = 3

// contractLiterals reads the backticked spans inside one contract
// element, from the lines the scan already holds — the projection carries
// the element's span, so this re-reads no file.
func contractLiterals(d *Document, e Element) []string {
	if e.LineStart < 1 || e.LineEnd > len(d.lines) || e.LineStart > e.LineEnd {
		return nil
	}
	return model.Backticked(strings.Join(d.lines[e.LineStart-1:e.LineEnd], "\n"))
}
