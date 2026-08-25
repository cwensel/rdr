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
	Epoch      string `json:"epoch"`
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
		Record: d.Record, Path: d.Path, Title: d.Title, Epoch: d.Epoch, Lines: d.Lines,
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
	for _, t := range model.TerminalStatuses {
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
	Line     int       `json:"line"`
	LineEnd  int       `json:"line_end"`
	Field    string    `json:"field,omitempty"`
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
	Skipped   []string             `json:"skipped"`
}

// BuildGraph assembles the graph over already-resolved documents.
func BuildGraph(docs []*Document, skipped []string) Graph {
	g := Graph{Schema: SchemaVersion, Records: []Summary{}, Elements: []GraphElement{},
		Edges: []GraphEdge{}, Backlinks: map[string][]BackRef{}, Skipped: skipped}
	if g.Skipped == nil {
		g.Skipped = []string{}
	}
	for _, d := range docs {
		g.Records = append(g.Records, Summarize(d))
		for _, e := range d.Elements {
			g.Elements = append(g.Elements, GraphElement{d.Record, e.ID, e.Kind, e.Label, e.Hash, e.LineStart, e.LineEnd, e.Joint})
		}
		for _, e := range d.Edges {
			g.Edges = append(g.Edges, GraphEdge{d.Record, e.From, e.To, e.Kind, e.Resolved, e.Line, e.LineEnd, e.Field})
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
	if pa == pb {
		return true
	}
	long, short := pa, pb
	if len(short) > len(long) {
		long, short = short, long
	}
	return strings.HasSuffix(long, "/"+short)
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
	// Uncited first, then by how much is shared, then by pair — so the
	// top of the report is the pair most likely to be a joint decision.
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
	return out
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

// readmeRow matches `| [NNNN](file) | Title | Status | Priority |`.
var readmeRow = regexp.MustCompile(`^\|\s*\[(\d{4})\]\([^)]*\)\s*\|(.*)$`)

// ParseReadmeIndex reads the index table rows out of README lines. Only
// rows whose first cell is a linked record number are rows; the header,
// the rule and the prose are not.
func ParseReadmeIndex(lines []string) []ReadmeRow {
	var rows []ReadmeRow
	for i, l := range lines {
		m := readmeRow.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		cells := splitCells(m[2])
		row := ReadmeRow{Record: m[1], Line: i + 1}
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
