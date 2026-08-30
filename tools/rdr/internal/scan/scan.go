// Package scan is the line scanner: it reads one RDR record and projects
// it as an outline, a set of addressable elements, and a warnings channel.
//
// It is deliberately a line scanner rather than a markdown AST walk. RDR
// markdown is line-oriented — ATX headings, fences, bullet trees with
// bold labels — and every element the projection needs is recoverable
// from line shapes, which keeps the scanner small and its line ranges
// exact: an element's ID round-trips to the bytes it names.
//
// Nothing here writes. If the projection and the record disagree the
// record is right; see the package model's READ, NEVER JUDGE rule.
package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// SchemaVersion is the envelope contract consumers pin. It moves only when
// the envelope's shape changes. 2 adds edges[].
const SchemaVersion = "2"

// Document is the projection of one record.
type Document struct {
	Schema  string `json:"schema"`
	Record  string `json:"record"`
	Project string `json:"project,omitempty"`
	Path    string `json:"path"`
	Title   string `json:"title"`
	Lines   int    `json:"lines"`

	Outline  []Node    `json:"outline"`
	Elements []Element `json:"elements"`
	// Anchors are the bold paragraph leads a citation can name with `§`.
	// They are addressable text, not structure: see anchors().
	Anchors []Anchor `json:"anchors,omitempty"`
	// Metadata is the Metadata block's fields, classified; Fields is every
	// other labelled bullet that sits inside no element (an element's own
	// fields nest under it). See fields.go.
	Metadata []Field `json:"metadata"`
	Fields   []Field `json:"fields"`
	// Edges are the typed relations this record states — to other
	// records and their elements, to code, to artifacts and to trackers.
	// See edges.go.
	Edges    []Edge    `json:"edges"`
	Warnings []Warning `json:"warnings"`
	Coverage Coverage  `json:"coverage"`
	Counts   Counts    `json:"counts"`

	lines  []string
	fenced []bool
	nodes  []*Node
	// fieldLines are the lines a field pass owns, which the body edge
	// passes skip. See edges.go.
	fieldLines map[int]bool
}

// Node is one heading and the lines it governs, subsections included.
type Node struct {
	// ID is the section's element ID: `NNNN:§<slug>`.
	ID string `json:"id"`
	// Heading is the text as written.
	Heading string `json:"heading"`
	Level   int    `json:"level"`
	// Canonical is the template section this heading maps to, or "".
	Canonical string `json:"canonical,omitempty"`
	Class     string `json:"class,omitempty"`
	// Match says how the heading matched the template (model.MatchKind).
	Match string `json:"match"`
	// Derived is true when the slug came from the heading text rather
	// than a canonical section name, so rewording the heading moves it.
	Derived   bool   `json:"derived"`
	Hash      string `json:"hash"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	// Parent is the enclosing node's ID, or "" for the title.
	Parent string `json:"parent,omitempty"`
}

// Anchor is a bold paragraph lead a `§` citation can name: `**Attribute
// resolution is as-authored, byte-preserving.**` opening a paragraph.
//
// It is NOT an outline node. A node is structure — it governs a range of
// lines, nests, and every non-blank line of the record lies inside one —
// and a bold lead governs nothing; putting it in the outline would break
// the nesting and coverage invariants that make the outline worth having.
// It is not an element either: it has no class, no fields and no
// lifecycle. It is a piece of text with a name, recorded so a citation
// that reaches for it by that name lands somewhere.
type Anchor struct {
	// ID is the anchor's ID in the section grammar: `NNNN:§<slug>`. The
	// section grammar is deliberate — `§` is how the corpus cites a named
	// piece of a record, and the author writing `§"The values"` is not
	// asserting the target is a heading, only that it is called that.
	ID string `json:"id"`
	// Text is the lead as written, without its bold markers.
	Text string `json:"text"`
	// Section is the ID of the node the anchor sits in.
	Section string `json:"section"`
	// Line is where the lead is written.
	Line int `json:"line"`
}

// Element is one addressable semantic object.
type Element struct {
	ID   string     `json:"id"`
	Kind ident.Kind `json:"kind"`
	Key  string     `json:"key,omitempty"`
	// Label is the author's handle for the element: the assumption
	// statement, the decision class as written, the alternative's title,
	// the first line of a list item.
	Label string `json:"label,omitempty"`
	// Derived is true when the projector minted the key from the
	// element's ordinal rather than reading a label.
	Derived bool `json:"derived"`
	// Backlog is true when a derived id is one an author could write
	// down: the template keys the kind (model.KindKeys) and the slot was
	// left empty. A derived id that is NOT a backlog is the element's
	// identity — the template keys the kind nowhere, so no edit improves
	// it. Whether a derived id is work is the TEMPLATE's answer, not the
	// element's; this field carries it so a consumer need not re-derive it.
	Backlog bool `json:"backlog,omitempty"`
	// Hash is the short content hash (ident.Hash) over the element's
	// lines; it separates "same ID, same content" from "same ID, changed".
	Hash string `json:"hash"`
	// Section is the ID of the node the element was read from.
	Section   string `json:"section"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	// Transient marks a contract carrying the Transient marker.
	Transient bool `json:"transient,omitempty"`
	// Parent is the contract a clause was read from (`0055:C1` for
	// `0055:L-3`); empty on every other kind. Clauses are the one kind
	// that nests inside another element.
	Parent string `json:"parent,omitempty"`
	// Fields are the labelled bullets inside the element: an assumption's
	// Evidence Record, a failure mode's Visible/Silent/Recovery, a step's
	// Risk. Each is classified against the template (see fields.go).
	Fields []Field `json:"fields,omitempty"`
	// Joint is set on a JC element: the parsed `Joint-check:` line.
	Joint *JointCheck `json:"joint,omitempty"`
}

// JointCheck is one `Joint-check:` line, read as data. The line is the
// record's own account of a shared decision with its peers, and the one
// fact a gate needs from it — is it still OPEN — was being scraped out of
// prose with grep, truncated by `| head`, and misread. The field cannot be.
type JointCheck struct {
	// Verdict is `fired`, `clear` or `re-run`, as written after the label.
	Verdict string `json:"verdict"`
	// Targets are the peer records a fired check names, as written.
	Targets []string `json:"targets,omitempty"`
	// Home is the text inside `(home: …)`: where the decision is ruled.
	Home string `json:"home,omitempty"`
	// Open is true when the home is OPEN — the decision has no ruling yet.
	Open bool `json:"open"`
}

// Warning is anything the scanner could not classify, with where it was.
// Nothing is dropped silently: an unknown heading, a duplicate label or a
// contracts section with no fenced block all land here.
type Warning struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// Counts summarises elements per kind and, per kind, how many carry a
// derived ID.
//
// Derived counts only the kinds the template labels (ident.Kind.Labelled)
// — the labelling backlog, the elements an author could pin. Structural
// counts the rest: BR, F and MVV, whose ordinal is their identity because
// the template gives them nowhere to write one. Both are minted ids;
// only the first names work.
type Counts struct {
	Elements   map[ident.Kind]int `json:"elements"`
	Derived    map[ident.Kind]int `json:"derived"`
	Structural map[ident.Kind]int `json:"structural"`
	// Fields counts every labelled bullet by how it matched the template;
	// `author` is the count the model does not know.
	Fields map[string]int `json:"fields"`
}

// Options tunes a scan.
type Options struct {
	// Project qualifies every ID (`cli/0055:C4`). Empty inside one dir.
	Project string
	// Record overrides the record number read from the title / filename.
	Record string
}

// File reads and scans one record.
func File(path string, opts Options) (*Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc := Bytes(raw, opts)
	doc.Path = path
	if doc.Record == "" {
		doc.Record = ident.RecordOf(filepath.Base(path))
		doc.reassign()
	}
	return doc, nil
}

// Bytes scans a record held in memory. The record number is read from the
// title line; File falls back to the filename.
func Bytes(raw []byte, opts Options) *Document {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1] // a trailing newline is not a line
	}
	doc := &Document{
		Schema:   SchemaVersion,
		Project:  opts.Project,
		Record:   opts.Record,
		Lines:    len(lines),
		lines:    lines,
		Warnings: []Warning{},
		Elements: []Element{},
	}
	doc.markFences()
	doc.outline()
	doc.classify()
	doc.extract()
	doc.fields()
	doc.edges()
	doc.coverage()
	doc.count()
	return doc
}

// Line returns the 1-based line, or "" out of range.
func (d *Document) Line(n int) string {
	if n < 1 || n > len(d.lines) {
		return ""
	}
	return d.lines[n-1]
}

// Fenced reports whether the 1-based line lies inside a fenced block.
// A consumer proposing a byte-level repair needs it: text inside a fence
// is quoted content — a code sample, a transcript, a record quoted by
// another record — and rewriting a citation there would edit the quote
// rather than the claim.
func (d *Document) Fenced(n int) bool {
	if n < 1 || n > len(d.fenced) {
		return false
	}
	return d.fenced[n-1]
}

// Slice returns lines start..end inclusive, 1-based.
func (d *Document) Slice(start, end int) []string {
	if start < 1 {
		start = 1
	}
	if end > len(d.lines) {
		end = len(d.lines)
	}
	if start > end {
		return nil
	}
	return d.lines[start-1 : end]
}

// Select resolves an element or section ID to its line range. The ID may
// be local or carry this document's project prefix.
func (d *Document) Select(id string) (start, end int, ok bool) {
	parsed, err := ident.Parse(id)
	if err != nil {
		return 0, 0, false
	}
	if parsed.Project != "" && parsed.Project != d.Project {
		return 0, 0, false
	}
	if parsed.Record != d.Record {
		return 0, 0, false
	}
	want := parsed.Qualified(d.Project)
	for _, e := range d.Elements {
		if e.ID == want {
			return e.LineStart, e.LineEnd, true
		}
	}
	for _, n := range d.Outline {
		if n.ID == want {
			return n.LineStart, n.LineEnd, true
		}
	}
	// Anchors are searched last: a heading of the same slug is the
	// section, and the lead beneath it adds nothing to select.
	for _, a := range d.Anchors {
		if a.ID == want {
			return a.Line, a.Line, true
		}
	}
	return 0, 0, false
}

func (d *Document) warn(code string, start, end int, format string, args ...any) {
	d.Warnings = append(d.Warnings, Warning{
		Code: code, Message: fmt.Sprintf(format, args...), LineStart: start, LineEnd: end,
	})
}

func (d *Document) id(kind ident.Kind, key string) string {
	return ident.ID{Project: d.Project, Record: d.Record, Kind: kind, Key: key}.String()
}

// --- fences ------------------------------------------------------------

// markFences flags every line that is a fence delimiter or sits inside a
// fence, so headings and bullets are never read out of code.
func (d *Document) markFences() {
	d.fenced = make([]bool, len(d.lines))
	open := ""
	for i, l := range d.lines {
		m := model.FenceDelimiter.FindStringSubmatch(l)
		switch {
		case open == "" && m != nil:
			open = m[1]
			d.fenced[i] = true
		case open != "" && m != nil && m[1] == open && strings.TrimSpace(l) == open:
			// a closing delimiter is bare; an opening one may carry an
			// info string
			open = ""
			d.fenced[i] = true
		case open != "":
			d.fenced[i] = true
		}
	}
	if open != "" {
		d.warn("fence:unclosed", len(d.lines), len(d.lines), "a %s fence is never closed", open)
	}
}

// --- outline -----------------------------------------------------------

var titleRecord = regexp.MustCompile(`^#\s+Recommendation\s+(\d{4})\b`)

// outline builds the heading tree. A node's range runs from its heading
// to the line before the next heading of its level or higher, trailing
// blank lines trimmed; the title node's range is the whole document.
func (d *Document) outline() {
	type head struct {
		level int
		text  string
		line  int // 1-based
	}
	var heads []head
	for i, l := range d.lines {
		if d.fenced[i] {
			continue
		}
		if m := model.Heading.FindStringSubmatch(l); m != nil {
			heads = append(heads, head{len(m[1]), m[2], i + 1})
		}
	}

	if len(heads) == 0 || heads[0].level != 1 {
		d.warn("title:missing", 1, 1, "no level-1 title heading at the top of the record")
	}
	var stack []*Node
	for i, h := range heads {
		end := len(d.lines)
		for _, later := range heads[i+1:] {
			if later.level <= h.level {
				end = later.line - 1
				break
			}
		}
		for end > h.line && strings.TrimSpace(d.lines[end-1]) == "" {
			end--
		}
		n := &Node{Heading: h.text, Level: h.level, LineStart: h.line, LineEnd: end}
		if h.level == 1 {
			d.Title = h.text
			n.LineStart, n.LineEnd = 1, len(d.lines)
			if d.Record == "" {
				if m := titleRecord.FindStringSubmatch(l1(d.lines, h.line)); m != nil {
					d.Record = m[1]
				}
			}
		}
		for len(stack) > 0 && stack[len(stack)-1].Level >= h.level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			n.Parent = stack[len(stack)-1].ID // filled after classify; see reassign
		}
		stack = append(stack, n)
		d.nodes = append(d.nodes, n)
	}
}

func l1(lines []string, n int) string {
	if n < 1 || n > len(lines) {
		return ""
	}
	return lines[n-1]
}

// parentOf returns the nearest enclosing node index, or -1.
func (d *Document) parentOf(i int) int {
	for j := i - 1; j >= 0; j-- {
		if d.nodes[j].Level < d.nodes[i].Level {
			return j
		}
	}
	return -1
}

// --- headings ----------------------------------------------------------

var (
	listItem   = regexp.MustCompile(`^(\s*)(?:[-*+]|\d+[.)])\s+(.*)$`)
	numberItem = regexp.MustCompile(`^(\s*)(\d+)[.)]\s+(.*)$`)
	// tableRowLead is a table row whose first cell is a lettered ordinal
	// (`T-6`, `**T-34**`, `F-2`); header and rule rows have none.
	tableRowLead = regexp.MustCompile(`^\|\s*(?:\*\*)?[A-Z]{1,3}-?(\d+[a-z]?)(?:\*\*)?\s*\|`)
	// jointCheckLine is a `Joint-check:` line in any of its three verdicts,
	// at line start or after a sentence (`Premortem: hardened. Joint-check: …`);
	// a mention inside backticks or mid-sentence is prose, not a check.
	jointCheckLine = regexp.MustCompile(`^(?:\s*(?:[-*]\s+)?|.*[.!]\s+)Joint-check:\s*(fired|clear|re-run)\s*(?:→\s*)?(.*)$`)
	jointHome      = regexp.MustCompile(`\(home:\s*([^)]*)\)`)
	jointTarget    = regexp.MustCompile(`^(?:[a-z][a-z0-9-]*/)?\d{4}$`)
	boldLead       = regexp.MustCompile(`^\*\*(.+?)\*\*`)
)

// metadata reads the Metadata block's fields, values line-joined per
// model.ValueContinues. It is keyed by the label as written.
func (d *Document) metadata() map[string]string {
	fields := map[string]string{}
	n := d.findHeading("Metadata")
	if n == nil {
		return fields
	}
	for i := n.LineStart + 1; i <= n.LineEnd; i++ {
		m := model.MetadataFieldBullet.FindStringSubmatch(d.lines[i-1])
		if m == nil {
			continue
		}
		val := m[2]
		for j := i + 1; j <= n.LineEnd && model.ValueContinues(d.lines[j-1]); j++ {
			val += " " + strings.TrimSpace(d.lines[j-1])
			i = j
		}
		fields[strings.TrimSpace(m[1])] = strings.TrimSpace(val)
	}
	return fields
}

// IsRecord reports whether the file is an RDR at all.
//
// It is the weakest possible test, and deliberately so: a Metadata block
// carrying a Status field, or a Critical Assumptions section. A file
// showing neither is not a record read against a stricter template — it
// is not a record, and the callers skip it rather than report it as a
// malformed one. This is what the epoch fingerprint's "unknown" verdict
// used to answer, kept as its own question now that the epochs are gone.
func (d *Document) IsRecord() bool {
	if md := d.metadata(); md != nil {
		if _, ok := md["Status"]; ok {
			return true
		}
	}
	return d.findHeading("Critical Assumptions") != nil
}

// findHeading finds the first node whose heading text equals name,
// case-insensitively, before template classification has run.
func (d *Document) findHeading(name string) *Node {
	for _, n := range d.nodes {
		if strings.EqualFold(strings.TrimSpace(n.Heading), name) {
			return n
		}
	}
	return nil
}

// --- classification ----------------------------------------------------

// classify maps each heading onto the template and
// assigns section IDs. Canonical sections take the canonical slug, so a
// case- or level-variant heading, or a legacy alias, resolves to the same
// ID as the section it stands for. Everything else slugs its own text and
// is flagged derived.
func (d *Document) classify() {
	te := model.Template()
	seen := map[string]int{}
	for i, n := range d.nodes {
		var key string
		if n.Level == 1 {
			n.Match = "title"
			key = "title"
		} else {
			m := model.LookupSection(te, n.Heading, n.Level)
			n.Match = m.Kind.String()
			if m.Canonical != nil {
				n.Canonical = m.Canonical.Name
				n.Class = m.Canonical.Class.String()
			}
			switch {
			case m.Canonical != nil && m.Kind != model.MatchScaffoldInstance && m.Kind != model.MatchLegacyAlias:
				key = ident.Slug(m.Canonical.Name)
			default:
				// A scaffold instance or a legacy alias slugs its own
				// text: the heading is the author's, and a legacy name
				// must never take the canonical slug away from the
				// section that is actually written under it (a record
				// can carry both `### Premortem` and `### Decision
				// Rationale`).
				key = ident.Slug(n.Heading)
				n.Derived = true
			}
			if m.Kind == model.MatchUnknown {
				switch {
				case d.underUnknownSection(i):
					// Inside a foreign section already reported: the
					// ancestor's warning covers these lines.
				case d.underTemplateSection(i):
					// An author's own sub-heading inside a template
					// section is that section's content, not a foreign
					// section: no warning. Whether it is really an
					// unmodelled template addition is a corpus-level
					// question (index --coverage).
					n.Match = model.MatchAuthorSubsection.String()
				default:
					d.warn("section:unknown-to-template", n.LineStart, n.LineEnd,
						"heading %q (level %d) matches no section of the template", n.Heading, n.Level)
				}
			}
		}
		if key == "" {
			key = "untitled"
			n.Derived = true
		}
		seen[key]++
		if c := seen[key]; c > 1 {
			// A second heading with the same slug — two `Step 1`s under
			// two phases, or a canonical section written twice — gets a
			// numbered slug so both stay addressable, and a warning
			// because only the first is the one a citation means.
			key = key + "-" + strconv.Itoa(c)
			n.Derived = true
			d.warn("section:duplicate", n.LineStart, n.LineEnd,
				"heading %q repeats a section slug; addressed as §%s", n.Heading, key)
		}
		n.ID = d.id(ident.Section, key)
		n.Hash = ident.Hash(d.Slice(n.LineStart, n.LineEnd))
		if p := d.parentOf(i); p >= 0 {
			n.Parent = d.nodes[p].ID
		} else {
			n.Parent = ""
		}
	}
	d.Outline = make([]Node, len(d.nodes))
	for i, n := range d.nodes {
		d.Outline[i] = *n
	}
}

// underTemplateSection reports whether node i has an enclosing node,
// below the title, that the template recognises — mapped to a canonical
// section, or a recognised-unmapped one (`Rationale`, `Open Questions`),
// whose own sub-headings are equally the author's.
func (d *Document) underTemplateSection(i int) bool {
	for p := d.parentOf(i); p >= 0; p = d.parentOf(p) {
		n := d.nodes[p]
		if n.Level > 1 && (n.Canonical != "" || n.Match == model.MatchRecognizedUnmapped.String()) {
			return true
		}
	}
	return false
}

// underUnknownSection reports whether node i sits inside a heading
// already classified unknown.
func (d *Document) underUnknownSection(i int) bool {
	for p := d.parentOf(i); p >= 0; p = d.parentOf(p) {
		if d.nodes[p].Match == model.MatchUnknown.String() {
			return true
		}
	}
	return false
}

// reassign re-derives every ID after the record number changed (File
// learned it from the filename because the title carried none). The
// interim IDs were minted with an empty record, so their keys are read
// back positionally rather than parsed.
func (d *Document) reassign() {
	keyOf := func(id string) string {
		_, key, _ := strings.Cut(id, ":§")
		return key
	}
	for i := range d.nodes {
		n := d.nodes[i]
		n.ID = d.id(ident.Section, keyOf(n.ID))
	}
	for i := range d.nodes {
		if p := d.parentOf(i); p >= 0 {
			d.nodes[i].Parent = d.nodes[p].ID
		}
	}
	for i := range d.Outline {
		d.Outline[i] = *d.nodes[i]
	}
	for i := range d.Elements {
		e := &d.Elements[i]
		e.ID = d.id(e.Kind, e.Key)
		e.Section = d.id(ident.Section, keyOf(e.Section))
		for j := range e.Fields {
			e.Fields[j].Section = d.id(ident.Section, keyOf(e.Fields[j].Section))
			e.Fields[j].Element = e.ID
		}
	}
	for _, fs := range [][]Field{d.Metadata, d.Fields} {
		for j := range fs {
			fs[j].Section = d.id(ident.Section, keyOf(fs[j].Section))
		}
	}
	for i := range d.Anchors {
		a := &d.Anchors[i]
		a.ID = d.id(ident.Section, keyOf(a.ID))
		a.Section = d.id(ident.Section, keyOf(a.Section))
	}
	// Edges carry element IDs on both ends, and a self-edge's target is
	// this record's own. They are re-read rather than patched: the pass
	// is pure over the now-correct record number, and re-reading cannot
	// drift from what a first-pass scan would have produced.
	d.dropEdgeWarnings()
	d.edges()
	// Edge warnings count toward the unclassified-line rate, so the
	// metric is recomputed over the re-read set.
	d.Coverage = Coverage{}
	d.coverage()
}

// dropEdgeWarnings removes the warnings the edge pass raised, so a
// re-read does not double them.
func (d *Document) dropEdgeWarnings() {
	kept := d.Warnings[:0]
	for _, w := range d.Warnings {
		if !strings.HasPrefix(w.Code, "edge:") {
			kept = append(kept, w)
		}
	}
	d.Warnings = kept
}

// --- elements ----------------------------------------------------------

// canonicalNodes returns the nodes mapped to a canonical section, in
// document order.
//
// A heading whose ALIAS names the section counts. `Canonical` is set by
// classify() against the template, and an alias to a section the template
// spells differently resolves to MatchRecognizedUnmapped — correctly,
// because the heading is not the canonical one. But the SECTION IS
// WRITTEN: a record with `### Decisions` and `- **D1**` bullets under it
// has those elements whatever it called the heading, and reading zero of
// them makes every citation into that record dangle against a section
// that is plainly there. Nine records cite `cli/0035:D-*` through exactly
// this path.
//
// So the reader asks what the record wrote, not what the template spells.
// This is the model's READ, NEVER JUDGE rule applied to a section: the
// classification stays exactly as it was — the heading is still reported
// as recognised-and-unmapped, and the template table is never bent to
// accommodate one record — while the elements underneath become
// addressable.
//
// Every alias kept here is a REFORMAT IN THE WAITING, not a permanent
// tolerance: the heading should eventually be migrated to its canonical
// spelling, and this table is what keeps the citations resolving until it
// is. Deleting an entry before the records are rewritten silently drops
// the elements under it.
// kindNodes returns the canonical nodes of the section that answers for
// an element kind, per the sidecar's [elements] map. A kind the map does
// not name projects from no section — that is a statement, not an
// omission (G is keyed by the gate item, JC by a line anywhere, § by the
// heading), so an unnamed kind reads as no nodes rather than a guess.
func (d *Document) kindNodes(kind ident.Kind) []*Node {
	section := model.ElementSections()[string(kind)]
	if section == "" {
		return nil
	}
	return d.canonicalNodes(section)
}

func (d *Document) canonicalNodes(name string) []*Node {
	var out []*Node
	for _, n := range d.nodes {
		if n.Canonical == name || (n.Canonical == "" && model.SectionAliasCanonical(n.Heading) == name) {
			out = append(out, n)
		}
	}
	return out
}

// body is the node's own lines below its heading, excluding any
// sub-heading's range.
func (d *Document) body(n *Node) (start, end int) {
	start, end = n.LineStart+1, n.LineEnd
	for _, m := range d.nodes {
		if m.LineStart > n.LineStart && m.LineStart <= n.LineEnd {
			end = m.LineStart - 1
			break
		}
	}
	for end >= start && strings.TrimSpace(d.lines[end-1]) == "" {
		end--
	}
	return start, end
}

// itemEnd finds the last line of the list item starting at line s (1-based)
// within [s, limit]. An item continues over indented lines, over lazy
// continuation lines that directly follow a non-blank line, and across a
// blank line only when what follows the blank is indented deeper than
// the item's marker. It ends at a heading, at a list item at the same or
// a shallower indent, and at a new column-zero paragraph.
func (d *Document) itemEnd(s, limit int) int {
	indent := indentOf(d.lines[s-1])
	end := s
	for i := s + 1; i <= limit; i++ {
		l := d.lines[i-1]
		fenced := d.fenced[i-1]
		trim := strings.TrimSpace(l)
		if trim == "" {
			next := i + 1
			for next <= limit && strings.TrimSpace(d.lines[next-1]) == "" {
				next++
			}
			if next > limit || indentOf(d.lines[next-1]) <= indent {
				return end
			}
			continue
		}
		if !fenced && model.Heading.MatchString(l) {
			return end
		}
		if !fenced && listItem.MatchString(l) && indentOf(l) <= indent {
			return end
		}
		if indentOf(l) <= indent && strings.TrimSpace(d.lines[i-2]) == "" {
			return end
		}
		end = i
	}
	return end
}

func indentOf(l string) int {
	return len(l) - len(strings.TrimLeft(l, " \t"))
}

// topItems lists the top-level list items in [start, end]: the items at
// the shallowest indent seen, outside fences.
func (d *Document) topItems(start, end int) []int {
	min := -1
	var all []int
	for i := start; i <= end; i++ {
		if d.fenced[i-1] {
			continue
		}
		if m := listItem.FindStringSubmatch(d.lines[i-1]); m != nil {
			ind := len(m[1])
			if min < 0 || ind < min {
				min = ind
			}
			all = append(all, i)
		}
	}
	var out []int
	for _, i := range all {
		if indentOf(d.lines[i-1]) == min {
			out = append(out, i)
		}
	}
	return out
}

func (d *Document) add(e Element) {
	e.ID = d.id(e.Kind, e.Key)
	e.Hash = ident.Hash(d.Slice(e.LineStart, e.LineEnd))
	d.Elements = append(d.Elements, e)
}

// itemLabel is an item's text with its marker removed and any leading
// bold label extracted; the whole first line stands in when there is no
// bold lead. A bold lead that is only the template's own field name
// (`**Scenario**:`) is skipped in favour of what follows it.
func itemLabel(line string) string {
	m := listItem.FindStringSubmatch(line)
	if m == nil {
		return strings.TrimSpace(line)
	}
	text := strings.TrimSpace(m[2])
	if b := boldLead.FindStringSubmatch(text); b != nil {
		lead := strings.TrimSpace(b[1])
		if strings.EqualFold(strings.TrimRight(lead, ":"), "Scenario") {
			rest := strings.TrimSpace(strings.TrimPrefix(text[len(b[0]):], ":"))
			if rest != "" {
				return rest
			}
		}
		return lead
	}
	// A bold lead that closes on a later line leaves its opener behind.
	return strings.TrimSpace(strings.TrimLeft(text, "*"))
}

func (d *Document) extract() {
	d.assumptions()
	d.contracts()
	d.clauses()
	d.decisions()
	d.listKind(ident.RoundTrip, roundTripLabel)
	d.alternatives()
	d.listKind(ident.Rejected, nil)
	d.listKind(ident.Scenario, nil)
	d.mvv()
	d.listKind(ident.Failure, nil)
	d.gate()
	d.jointChecks()
	d.anchors()
	sort.SliceStable(d.Elements, func(i, j int) bool {
		return d.Elements[i].LineStart < d.Elements[j].LineStart
	})
}

// statement recovers an assumption's statement from its bullet, joined
// across wrapped lines: the bracketed text when the label opens a
// bracket, otherwise the text up to the closing `**` — or, when the bold
// closes on the label itself, the rest of the first line.
func (d *Document) statement(start, end int) string {
	head := strings.TrimSpace(d.lines[start-1])
	head = head[model.AssumptionBullet.FindStringIndex(head)[1]:] // drop the label
	var b strings.Builder
	b.WriteString(head)
	for i := start + 1; i <= end && !strings.Contains(b.String(), "**"); i++ {
		b.WriteByte(' ')
		b.WriteString(strings.TrimSpace(d.lines[i-1]))
	}
	text := b.String()
	switch {
	case strings.HasPrefix(strings.TrimSpace(text), "["):
		text = strings.TrimSpace(text)[1:]
		if j := strings.Index(text, "]"); j >= 0 {
			text = text[:j]
		}
	case strings.HasPrefix(text, "**"):
		text = strings.TrimSpace(text[2:])
		if j := strings.Index(text, "\n"); j >= 0 {
			text = text[:j]
		}
	default:
		if j := strings.Index(text, "**"); j >= 0 {
			text = text[:j]
		}
	}
	text = strings.TrimSpace(text)
	text = strings.TrimLeft(text, "—–-: ")
	return strings.TrimSpace(text)
}

// assumptions reads the top-level bullets of Critical Assumptions —
// wherever it lives: at `##`, at `###`, or under a legacy alias. A bullet
// carrying an A-label is as written; an unlabelled one (the older
// checkbox bullets) is derived by ordinal, so every assumption in the
// corpus is addressable and the unlabelled ones are counted as backlog.
func (d *Document) assumptions() {
	var items []keyed
	labelled := false
	for _, n := range d.kindNodes(ident.Assumption) {
		for _, i := range d.topItems(n.LineStart+1, n.LineEnd) {
			it := keyed{section: n.ID, start: i, end: d.itemEnd(i, n.LineEnd)}
			if m := model.AssumptionBullet.FindStringSubmatch(d.lines[i-1]); m != nil {
				// `A1.b` and `A1-b` are the split label `A1b`.
				it.key = strings.NewReplacer(".", "", "-", "").Replace(strings.TrimPrefix(m[1], "A"))
				it.label = d.statement(i, it.end)
				labelled = true
			} else {
				it.label = itemLabel(checkbox.ReplaceAllString(d.lines[i-1], "$1"))
			}
			items = append(items, it)
		}
	}
	if labelled {
		// A section that labels its assumptions also carries other
		// bullets — the template's Method-vocabulary legend, copied in
		// verbatim by a whole cohort of records — and those are not
		// assumptions. Only a label-free section (the older checkbox
		// list) has its bullets read as assumptions by position.
		kept := items[:0]
		for _, it := range items {
			if it.key != "" {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	d.assign(ident.Assumption, items)
}

// checkbox strips a task-list marker (`- [ ] `, `- [x] `) down to its
// bullet, so the label reads from the text.
var checkbox = regexp.MustCompile(`^(\s*-\s+)\[[ xX]\]\s+`)

// keyed is an element candidate before its key is settled.
type keyed struct {
	section    string
	start, end int
	key        string // as written, or "" when the projector must derive one
	label      string
	transient  bool
}

// assign settles keys for one kind and adds the elements. Author-written
// keys are claimed first, first writer wins; a repeated label is
// reported and the later element yields. Unlabelled elements take their
// ordinal among all candidates of the kind; an ordinal a label already
// claims — the mixed state of a live record part-way through labelling
// — takes the first free number above the candidate count, with a
// warning, so IDs stay unique and the author's numbering is never
// overridden.
func (d *Document) assign(kind ident.Kind, items []keyed) {
	claimed := map[string]bool{}
	tag := strings.ToLower(string(kind))
	for i := range items {
		it := &items[i]
		if it.key == "" {
			continue
		}
		if claimed[it.key] {
			d.warn(tag+":duplicate", it.start, it.end, "%s%s labels more than one element; the later one is addressed by ordinal", kind, it.key)
			it.key = ""
			continue
		}
		claimed[it.key] = true
	}
	next := len(items) + 1
	for ord, it := range items {
		e := Element{Kind: kind, Section: it.section, Label: it.label, Key: it.key,
			LineStart: it.start, LineEnd: it.end, Transient: it.transient}
		if e.Key == "" {
			key := strconv.Itoa(ord + 1)
			if claimed[key] {
				for claimed[strconv.Itoa(next)] {
					next++
				}
				d.warn(tag+":collision", it.start, it.end,
					"unlabelled %s %d shares its ordinal with a labelled %s%s; addressed as %s%d until labelled", tag, ord+1, kind, key, kind, next)
				key = strconv.Itoa(next)
				next++
			}
			e.Key, e.Derived = key, true
		}
		claimed[e.Key] = true
		d.add(e)
	}
}

// contractLabel matches an author-written contract label on the line
// before a normative fence. The label is the bold token alone — `**C4**`,
// `- **C4** title`, `**C4**: title` — or a `##### C4` heading. A `C4`
// buried in bold prose (`**C2 — the second claim**`) is not a label:
// the corpus writes such leads for claims and checks that are not
// contracts, and reading them as labels would mint IDs the author never
// meant.
var contractLabel = regexp.MustCompile(`^\s*(?:-\s+)?(?:\*\*C(\d+)\*\*|#{5,6}\s+C(\d+)\b)`)

// contracts reads every ```normative fence in the record, in document
// order. Fences are read document-wide, not only under Normative
// Contracts: records put them under the contract's own `#####`
// sub-heading, or beside the design they specify, and a fence is a
// contract wherever it sits; the element's Section says where.
//
// A fence is labelled when the nearest non-blank line above it carries a
// C-label, and derived (its document ordinal) otherwise. Labelled keys
// are claimed first; a derived ordinal that a label already claims — the
// mixed state of a live record part-way through labelling — takes the
// first free number above the fence count, with a warning, so IDs stay
// unique and the author's own numbering is never overridden.
//
// Contracts written as prose rather than fenced are not addressable;
// that shows as a zero count, not a warning, because older records
// wrote them that way and a terminal record is never wrong for its age.
func (d *Document) contracts() {
	var items []keyed
	for i := 1; i <= len(d.lines); i++ {
		if !model.NormativeFenceOpen.MatchString(d.lines[i-1]) || !d.fenced[i-1] {
			continue
		}
		end := i
		for end < len(d.lines) && !(d.fenced[end] && strings.TrimSpace(d.lines[end]) == "```") {
			end++
		}
		if end < len(d.lines) {
			end++ // include the closing delimiter
		}
		// The template's own example is not the author's contract.
		// A seeded record carries it verbatim until the section is
		// authored, and counting it mints a C1 the author never wrote —
		// which is the same class as a surviving guidance block, and is
		// reported as one rather than projected as an element.
		if body := fenceBody(d.lines, i, end); model.TemplateNormativeBodies()[body] {
			d.warn("contract:template-example", i, end,
				"this ```normative fence is TEMPLATE.md's own example, not an authored contract")
			continue
		}
		it := keyed{section: d.sectionAt(i), start: i, end: end}
		for k := i - 1; k >= 1; k-- {
			if strings.TrimSpace(d.lines[k-1]) == "" {
				continue
			}
			if m := contractLabel.FindStringSubmatch(d.lines[k-1]); m != nil && !d.fenced[k-1] {
				it.key, it.start = m[1]+m[2], k
				// The label's own text is what follows the token.
				rest := strings.TrimSpace(d.lines[k-1][len(m[0]):])
				it.label = strings.TrimSpace(strings.TrimLeft(rest, "*:—–- "))
			}
			break
		}
		if it.label == "" {
			// A derived contract is labelled by its first line of
			// content — the signature, usually — so a listing can tell
			// the blocks apart.
			for k := i + 1; k <= end; k++ {
				if t := strings.TrimSpace(d.lines[k-1]); t != "" && t != "```" {
					it.label = t
					break
				}
			}
		}
		for k := it.start; k <= it.end; k++ {
			if model.TransientMarker.MatchString(d.lines[k-1]) {
				it.transient = true
			}
		}
		items = append(items, it)
	}
	d.assign(ident.Contract, items)
}

// clauseDef matches a clause DEFINITION inside a normative fence: a label
// of one to three capitals, a hyphen and a number (`L-3`, `I-4`, `REQ-12`,
// `NC-5a`) at column zero, followed by the definition's own separator —
// two or more spaces (`L-1  input := …`), a space and an opening
// parenthesis (`REQ-2 (single surface): …`), a colon, or the close of a
// bold lead (`**REQ-4b (the emission shape).**`).
//
// The separator is what tells a definition from prose that merely opens
// a wrapped line with the label: `REQ-5 serves two arities`, `NC-5(2),
// that its naming request…`, `I-4(b) compares`. Measured over the corpus
// the column-zero rule alone reads 112 lines with six labels defined
// twice in one record; with the separator it reads 90 with none. A
// continuation line is indented, so column zero is the other half of
// the rule.
var clauseDef = regexp.MustCompile(`^(?:\*\*)?([A-Z]{1,3}-\d+[a-z]?)(?:\*\*|\s{2,}|\s\(|:)`)

// clauseDefWide matches the alignment casualty the two-space arm creates:
// once a label's digits reach two, a column-aligned list closes the gap to
// a single space (`R-9  determinism:` but `R-10 cascade guard.`), and the
// strict grammar reads R-10's lines as R-9's. Alone this would also match
// prose that opens a line with a wide label, so clauses() accepts it only
// inside a fence where a sibling with the same letter prefix already
// defined at two or more spaces — the alignment the collapsed gap is
// evidence of. No bold form: bold labels are not column-aligned.
var clauseDefWide = regexp.MustCompile(`^([A-Z]{1,3}-\d{2,}[a-z]?) `)

// clauseAligned captures the letter prefix of a definition written in the
// aligned style — clauseDef's two-or-more-space separator arm.
var clauseAligned = regexp.MustCompile(`^(?:\*\*)?([A-Z]{1,3})-\d+[a-z]?\s{2,}`)

// clauseKeyAt is the label token opening a definition line found by
// clauseDef or, failing that, clauseDefWide.
func clauseKeyAt(line string) string {
	if m := clauseDef.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return clauseDefWide.FindStringSubmatch(line)[1]
}

// clauses reads the labelled clauses inside every contract fence, in
// document order, and mints each as an element whose Parent is the
// contract. A contract is the projector's grain; the corpus cites one
// grain finer — every joint-decision home is written `cli/0112
// §Normative Contracts L-3` — and a 530-line C1 answers a citation of
// its L-3 with 530 lines.
//
// Labels are keyed as written and are UNIQUE PER RECORD, which is what
// the corpus does (ten records, twenty fences, ninety clauses, no
// label defined twice); a record whose contracts each restart at L-1
// would have no bare id for either. So a label defined in more than one
// place mints nothing and warns `clause:duplicate` naming every
// candidate, and `--select` refuses it as ambiguous rather than
// answering one of them. A clause runs from its definition line to the
// line before the next definition or the fence's close, trailing blank
// lines dropped.
func (d *Document) clauses() {
	type cand struct {
		el    Element
		where string
	}
	byKey := map[string][]cand{}
	var order []string
	for _, c := range d.Elements {
		if c.Kind != ident.Contract {
			continue
		}
		// The fence's close is the last line of the contract; a
		// labelled contract starts on its label line, and the fence
		// opener is inside the range either way.
		var defs []int
		aligned := map[string]bool{}
		for i := c.LineStart; i <= c.LineEnd; i++ {
			if d.fenced[i-1] && !model.NormativeFenceOpen.MatchString(d.lines[i-1]) && clauseDef.MatchString(d.lines[i-1]) {
				defs = append(defs, i)
				if m := clauseAligned.FindStringSubmatch(d.lines[i-1]); m != nil {
					aligned[m[1]] = true
				}
			}
		}
		// The second pass admits the alignment casualty: a wide label
		// whose single space is the collapsed two-space gap, accepted
		// only where a same-prefix sibling proved the aligned style.
		if len(aligned) > 0 {
			for i := c.LineStart; i <= c.LineEnd; i++ {
				if !d.fenced[i-1] || model.NormativeFenceOpen.MatchString(d.lines[i-1]) || clauseDef.MatchString(d.lines[i-1]) {
					continue
				}
				if m := clauseDefWide.FindStringSubmatch(d.lines[i-1]); m != nil && aligned[strings.SplitN(m[1], "-", 2)[0]] {
					defs = append(defs, i)
				}
			}
			sort.Ints(defs)
		}
		for j, start := range defs {
			end := c.LineEnd - 1 // the line before the closing delimiter
			if j+1 < len(defs) {
				end = defs[j+1] - 1
			}
			for end > start && strings.TrimSpace(d.lines[end-1]) == "" {
				end--
			}
			// The label is the rest of the definition line after the
			// token: the separator stays, since `(chain state)` is the
			// author's title and the parenthesis is part of it.
			key := clauseKeyAt(d.lines[start-1])
			label := d.lines[start-1][strings.Index(d.lines[start-1], key)+len(key):]
			label = strings.TrimSpace(strings.TrimLeft(strings.ReplaceAll(label, "**", ""), ":—–- "))
			e := Element{Kind: ident.Clause, Key: key, Label: label, Section: c.Section,
				Parent: c.ID, LineStart: start, LineEnd: end}
			if _, seen := byKey[key]; !seen {
				order = append(order, key)
			}
			byKey[key] = append(byKey[key], cand{e, fmt.Sprintf("%s %d-%d", c.ID, start, end)})
		}
	}
	for _, key := range order {
		cs := byKey[key]
		if len(cs) == 1 {
			d.add(cs[0].el)
			continue
		}
		var where []string
		for _, c := range cs {
			where = append(where, c.where)
		}
		d.warn("clause:duplicate", cs[0].el.LineStart, cs[len(cs)-1].el.LineEnd,
			"%s is defined in more than one contract (%s); no clause id is minted until the labels are unique within the record", key, strings.Join(where, ", "))
	}
}

// Ambiguity reports why an element id resolves to nothing when the
// reason is a label defined more than once: the `clause:duplicate`
// warning's message, naming every candidate, or "" when the id is
// simply absent. It lets a refusal say which of two answers it declined
// to pick instead of claiming the element does not exist.
func (d *Document) Ambiguity(id string) string {
	parsed, err := ident.Parse(id)
	if err != nil || parsed.Kind != ident.Clause || parsed.Record != d.Record {
		return ""
	}
	for _, w := range d.Warnings {
		if w.Code == "clause:duplicate" && strings.HasPrefix(w.Message, parsed.Key+" ") {
			return w.Message
		}
	}
	return ""
}

// fenceBody is the text inside a fence, trimmed, for comparison against
// the template's own examples. The delimiters are excluded: `i` is the
// opening line and `end` the line after the closing one.
func fenceBody(lines []string, i, end int) string {
	if i >= end-1 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[i:end-1], "\n"))
}

// anchorLead matches a bold run opening a paragraph at column zero:
// `**The values.** the rest of the sentence`, `**Attribute resolution is
// as-authored, byte-preserving.**` on a line of its own.
//
// Column zero is the test that separates a lead from everything else the
// corpus bolds. A `- **Evidence**:` is a field, a `- **A3 [...]**` is an
// assumption, a `  **Note.**` under a bullet is part of that bullet's
// body: all of them are indented, all of them already have an owner, and
// all of them would collide with the identities that owner mints. Only an
// unindented one starts a paragraph.
var anchorLead = regexp.MustCompile(`^\*\*([^*\n]{2,120}?)\*\*(?:$|[ .,;:—–-])`)

// anchors records the bold paragraph leads of the record.
//
// The corpus names them in citations — `cli/0009 § "Attribute resolution
// is as-authored, byte-preserving"`, `cli/0092 §"The values"` — and both
// forms above are QUOTED and verbatim: the most precise citation the
// grammar offers, naming a string that is in the target exactly once.
// Indexing only `#`-headings left those citations resolving against
// nothing, which reports the most careful references in the corpus as the
// broken ones.
//
// A lead is addressable text, not structure, so it goes in its own list
// (see Anchor) and the resolver consults it after the outline. Structure
// wins ties: if a heading and a lead slug alike, the heading is the
// section and the lead adds nothing.
//
// The lead is read WHEREVER IT SITS, like a contract fence, because a
// paragraph lead is a paragraph lead under any section.
func (d *Document) anchors() {
	seen := map[string]bool{}
	for i := 1; i <= len(d.lines); i++ {
		if d.fenced[i-1] {
			continue
		}
		// A lead OPENS a paragraph: the line above it is blank, a
		// heading, or the top of the file. A bold run mid-paragraph is
		// emphasis, and emphasis names nothing.
		if i > 1 {
			if prev := strings.TrimSpace(d.lines[i-2]); prev != "" && !model.Heading.MatchString(d.lines[i-2]) {
				continue
			}
		}
		if contractLabel.MatchString(d.lines[i-1]) {
			// A `**C4**` above a fence is a contract's label. The
			// contract is already an element with an ID of its own, and
			// a second identity for the same bytes in the section
			// namespace is noise at best and a collision at worst.
			continue
		}
		m := anchorLead.FindStringSubmatch(d.lines[i-1])
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[1])
		key := ident.Slug(strings.TrimRight(text, ".,;:"))
		if key == "" || seen[key] {
			// A slug written twice names neither occurrence. Dropping
			// both is the same refusal to guess the resolver makes for an
			// ambiguous prefix.
			seen[key] = true
			d.dropAnchor(key)
			continue
		}
		seen[key] = true
		d.Anchors = append(d.Anchors, Anchor{
			ID: d.id(ident.Section, key), Text: text, Section: d.sectionAt(i), Line: i})
	}
}

// dropAnchor removes an anchor whose slug turned out not to be unique.
func (d *Document) dropAnchor(key string) {
	want := d.id(ident.Section, key)
	kept := d.Anchors[:0]
	for _, a := range d.Anchors {
		if a.ID != want {
			kept = append(kept, a)
		}
	}
	d.Anchors = kept
}

// sectionAt returns the ID of the innermost node containing a line.
func (d *Document) sectionAt(line int) string {
	id := ""
	for _, n := range d.nodes {
		if n.LineStart <= line && line <= n.LineEnd {
			id = n.ID
		}
	}
	return id
}

// decisionNumber reads a `D1` / `D-1` bold lead as an as-written key.
//
// TEMPLATE.md keys decisions by CLASS, and the classes are what
// DecisionClassOf reads. But a cohort of records numbered theirs instead
// — `- **D1** Bodies inline on the op.`, `- **D4 (\`Seal.prev\`)** …` —
// and numbering is a label like any other: the author named the element,
// so the name is the key and the ID is as-written, not derived.
//
// It is also the spelling the citation grammar already reads: `§D6` maps
// to `D-6` through elementKind, which had nothing to land on while a
// numbered bullet slugged its whole sentence into a derived key.
var decisionNumber = regexp.MustCompile(`^D-?(\d+[a-z]?)\b`)

func decisionKey(label string) string {
	if m := decisionNumber.FindStringSubmatch(strings.Trim(label, "`* ")); m != nil {
		return m[1]
	}
	return ""
}

// decisions reads Load-Bearing Decisions bullets. A bullet whose bold
// label starts with a template decision class is keyed by that class
// (`D-identity`) or by its own number (`D-6`); any other label slugs its
// own text and is derived.
func (d *Document) decisions() {
	seen := map[string]int{}
	for _, n := range d.kindNodes(ident.Decision) {
		for _, i := range d.topItems(n.LineStart+1, n.LineEnd) {
			label := itemLabel(d.lines[i-1])
			end := d.itemEnd(i, n.LineEnd)
			e := Element{Kind: ident.Decision, Label: label, Section: n.ID, LineStart: i, LineEnd: end}
			if class := model.DecisionClassOf(label); class != "" {
				e.Key = ident.Slug(class)
			} else if num := decisionKey(label); num != "" {
				e.Key = num
			} else {
				e.Key, e.Derived = ident.Slug(label), true
			}
			if e.Key == "" {
				e.Key, e.Derived = "decision", true
			}
			seen[e.Key]++
			if c := seen[e.Key]; c > 1 {
				d.warn("decision:duplicate", i, end, "decision class %q is answered more than once", e.Key)
				e.Key, e.Derived = e.Key+"-"+strconv.Itoa(c), true
			}
			d.add(e)
		}
	}
}

// roundTripLabel reads an `INV-1` / `RT1` / `RT-1` lead as an as-written
// key.
var roundTripLabelRe = regexp.MustCompile(`^(?:RT|INV)-?(\d+)\b`)

func roundTripLabel(label string) string {
	if m := roundTripLabelRe.FindStringSubmatch(strings.Trim(label, "`* ")); m != nil {
		return m[1]
	}
	return ""
}

// listKind projects the top-level list items of a section as elements of
// one kind. The key is as written when the item carries a label the
// kind's labeller recognises, or when the section is a numbered list
// whose numbers are unique; otherwise it is the item's ordinal, derived.
//
// WHICH section answers for the kind is not written here: it is read
// from the sidecar's [elements] map, the one place that already states
// it. Naming the section again in Go would be a second source for one
// fact, and the two would drift — the same reason GateItems() reads its
// keys off the template rather than restating them. It also means a
// second document family retargets the projector by editing a table,
// not by editing this function.
func (d *Document) listKind(kind ident.Kind, labeller func(string) string) {
	nodes := d.kindNodes(kind)
	if len(nodes) == 0 {
		return
	}
	type item struct {
		n          *Node
		line, end  int
		label, key string
	}
	var items []item
	for _, n := range nodes {
		for _, i := range d.topItems(n.LineStart+1, n.LineEnd) {
			it := item{n: n, line: i, end: d.itemEnd(i, n.LineEnd), label: itemLabel(d.lines[i-1])}
			if labeller != nil {
				it.key = labeller(it.label)
			}
			if it.key == "" {
				if m := numberItem.FindStringSubmatch(d.lines[i-1]); m != nil {
					it.key = strings.TrimLeft(m[2], "0")
					if it.key == "" {
						it.key = "0"
					}
				}
			}
			items = append(items, it)
		}
		// A section written as a table — `| T-6 | …` — is the same list in
		// another shape: one row, one element, keyed by the row's lead.
		for i := n.LineStart + 1; i <= n.LineEnd; i++ {
			if m := tableRowLead.FindStringSubmatch(d.lines[i-1]); m != nil {
				items = append(items, item{n: n, line: i, end: i,
					label: collapse(strings.Trim(d.lines[i-1], "| ")), key: strings.TrimLeft(m[1], "0")})
			}
		}
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].line < items[b].line })
	// Author-written numbers are keys only if they are unique; a list
	// that restarts at 1 (two phases of scenarios) is addressed by
	// ordinal, with a warning naming the clash.
	counts := map[string]int{}
	for _, it := range items {
		if it.key != "" {
			counts[it.key]++
		}
	}
	// An id names one element's bytes, so the two branches below must not
	// draw from one namespace: an author's unique `8` and a positional
	// ord+1 of 8 are different elements and were both minted as S8. The
	// ordinal is disambiguated against every key already taken, the way
	// decisions() disambiguates a repeated class.
	taken := map[string]int{}
	for _, it := range items {
		if it.key != "" && counts[it.key] == 1 {
			taken[it.key]++
		}
	}
	for ord, it := range items {
		e := Element{Kind: kind, Label: it.label, Section: it.n.ID, LineStart: it.line, LineEnd: it.end}
		switch {
		case it.key != "" && counts[it.key] == 1:
			e.Key = it.key
		case it.key != "":
			d.warn(strings.ToLower(string(kind))+":duplicate", it.line, it.end,
				"%s item number %s repeats; addressed by ordinal", it.n.Canonical, it.key)
			fallthrough
		default:
			// The suffix is a letter, not `-N`: the ID grammar's
			// ordinal key is `\d+[a-z]?`, so `S8a` parses and `S8-1`
			// does not. Minting an id nothing can cite would be worse
			// than the collision it avoids.
			key := strconv.Itoa(ord + 1)
			for n := 0; taken[key] > 0 && n < 26; n++ {
				key = strconv.Itoa(ord+1) + string(rune('a'+n))
			}
			taken[key]++
			e.Key, e.Derived = key, true
		}
		d.add(e)
	}
}

// altOrdinal reads the author's own ordinal off an Alternative
// heading. The trailing letter is part of the ordinal — `Alt 3b` is a
// sibling of `Alt 3`, not a second writing of it — exactly as
// tableRowLead reads `T-5b`. Dropping it collided the two and sent
// every later Alternative in the section to a positional ordinal.
var altOrdinal = regexp.MustCompile(`(?i)^Alt(?:ernative)?\s+(\d+[a-z]?)`)

// alternatives projects each filled-in `### Alternative N:` scaffold as
// ALT<N>, keyed by the author's own ordinal.
func (d *Document) alternatives() {
	seen := map[string]int{}
	ord := 0
	for _, n := range d.nodes {
		if n.Canonical != "Alternative 1: [Name]" || n.Match != model.MatchScaffoldInstance.String() {
			continue
		}
		ord++
		e := Element{Kind: ident.Alternative, Label: n.Heading, Section: n.ID, LineStart: n.LineStart, LineEnd: n.LineEnd}
		if m := altOrdinal.FindStringSubmatch(n.Heading); m != nil && seen[m[1]] == 0 {
			e.Key = m[1]
		} else {
			if m != nil {
				d.warn("alt:duplicate", n.LineStart, n.LineEnd, "Alternative %s is written more than once; addressed by ordinal", m[1])
			}
			e.Key, e.Derived = strconv.Itoa(ord), true
		}
		seen[e.Key]++
		d.add(e)
	}
}

// mvv projects the Minimum Viable Validation body as the record's one MVV
// element. It is never derived: the template fixes its identity.
func (d *Document) mvv() {
	for _, n := range d.kindNodes(ident.MVV) {
		s, e := d.body(n)
		if s > e {
			continue
		}
		d.add(Element{Kind: ident.MVV, Label: n.Heading, Section: n.ID, LineStart: s, LineEnd: e})
		return
	}
}

// gate projects a record's inlined gate responses as
// G-<item> elements. With a gate.md pointer the responses live outside the
// record and there is nothing to address here.
func (d *Document) gate() {
	gates := d.canonicalNodes("Finalization Gate")
	if len(gates) == 0 {
		return
	}
	g := gates[0]
	// A gate.md pointer moves the four lock-time judgements out of the
	// record, but a locked gate KEEPS `### Cross-Cutting Concerns`: it
	// states a project-wide policy other RDRs cite, and an element that
	// is not projected cannot be cited. So the pointer is not a reason to
	// stop reading — whatever sub-sections remain below it are still the
	// record's own, and are projected as usual. A gate that is nothing
	// but the pointer simply has no sub-sections to find.
	found := false
	for _, n := range d.nodes {
		if n.Parent != g.ID || n.Level != g.Level+1 {
			continue
		}
		s, e := d.body(n)
		if s > e {
			continue
		}
		found = true
		el := Element{Kind: ident.Gate, Label: n.Heading, Section: n.ID, LineStart: s, LineEnd: e}
		if key := model.GateItemKey(n.Canonical); key != "" {
			el.Key = key
		} else {
			el.Key, el.Derived = ident.Slug(n.Heading), true
		}
		d.add(el)
	}
	if !found {
		d.gateItems(g)
	}
}

// gateItems projects a gate written as a labelled LIST rather than as
// sub-headings — `- **Contradiction Check**: …`. Two of the oldest
// records answer the gate that way, and reading only sub-headings made
// their five responses project as nothing: `counts.elements.G` was 0 and
// `coverage.unclassified` was 0 too, so the items were classified and
// then dropped without a word. A bullet the projector cannot place is
// supposed to be a warning, never silence.
//
// It is the same gate, in the other shape markdown offers for a list of
// labelled things, so it reads through the same table: the label is
// matched against the sub-sections TEMPLATE.md declares, and the key is
// the `[Gate key: …]` marker beside the one it names. Nothing about the
// item set is written here — a template that renames a gate item, adds
// one, or drops one moves both shapes together.
//
// A label naming no declared item still projects, derived, exactly as an
// author's own gate sub-heading does: `API Verification` is a real
// response those two records wrote, and dropping it would be the silence
// this function exists to end.
func (d *Document) gateItems(g *Node) {
	for _, i := range d.topItems(g.LineStart+1, g.LineEnd) {
		label := itemLabel(d.lines[i-1])
		if label == "" {
			continue
		}
		el := Element{
			Kind: ident.Gate, Label: label, Section: g.ID,
			LineStart: i, LineEnd: d.itemEnd(i, g.LineEnd),
		}
		if key := gateKeyForLabel(label); key != "" {
			el.Key = key
		} else {
			el.Key, el.Derived = ident.Slug(label), true
		}
		d.add(el)
	}
}

// gateKeyForLabel resolves a bullet's label onto a declared gate item.
//
// A heading is canonicalised before it reaches GateItemKey; a bullet
// label is not, so this asks the same table directly. The match is by
// name, case-insensitively, and also accepts a label that is the
// declared name's own lead — `Cross-Cutting` for `Cross-Cutting
// Concerns` — because that is how those records abbreviate it and the
// key it resolves to (`cross-cutting`) is the one peers already cite.
// Anything looser would be guessing, and a wrong gate key is worse than
// a derived one.
func gateKeyForLabel(label string) string {
	l := strings.ToLower(strings.TrimSpace(strings.Trim(label, "`*: ")))
	if l == "" {
		return ""
	}
	for _, g := range model.GateItems() {
		name := strings.ToLower(g.Section)
		if l == name {
			return g.Key
		}
		// The lead must end on a word boundary, so `Scope` matches
		// `Scope Verification` but `Scoped` does not.
		if strings.HasPrefix(name, l) && len(name) > len(l) && name[len(l)] == ' ' {
			return g.Key
		}
	}
	return ""
}

func (d *Document) count() {
	d.Counts = Counts{Elements: map[ident.Kind]int{}, Derived: map[ident.Kind]int{}, Structural: map[ident.Kind]int{}, Fields: map[string]int{}}
	for _, f := range d.Metadata {
		d.Counts.Fields[f.Match]++
	}
	for _, f := range d.Fields {
		d.Counts.Fields[f.Match]++
	}
	for _, e := range d.Elements {
		for _, f := range e.Fields {
			d.Counts.Fields[f.Match]++
		}
	}
	for _, k := range ident.Kinds {
		d.Counts.Elements[k] = 0
		d.Counts.Derived[k] = 0
		d.Counts.Structural[k] = 0
	}
	// A minted id lands in exactly one of the two columns: Derived where
	// an author could write the key down, Structural where the ordinal is
	// the identity. Elements[k] stays the total either way.
	tally := func(k ident.Kind, derived, backlog bool) {
		d.Counts.Elements[k]++
		switch {
		case !derived:
		case backlog:
			d.Counts.Derived[k]++
		default:
			d.Counts.Structural[k]++
		}
	}
	for i, e := range d.Elements {
		// A minted id is BACKLOG only where the template gives the kind
		// somewhere to write one; otherwise the ordinal is the identity.
		d.Elements[i].Backlog = e.Derived && model.KindKeys(string(e.Kind))
		tally(e.Kind, e.Derived, d.Elements[i].Backlog)
	}
	for _, n := range d.Outline {
		// A section's id is derived for several reasons, and only one is
		// work: a LEGACY ALIAS is a reformat in the waiting, retired when
		// the record is migrated to the canonical heading. A scaffold
		// instance (`### Alternative 2: …`), the author's own sub-heading,
		// and a heading recognised with no canonical home are the author's
		// text, permanent, and no template edit labels them.
		tally(ident.Section, n.Derived, n.Match == model.MatchLegacyAlias.String())
	}
}

// jointChecks projects every `Joint-check:` line as a JC element with the
// line parsed: verdict, the peers it names, and the home — OPEN or ruled.
// They are keyed by document ordinal; the line carries no label of its own.
func (d *Document) jointChecks() {
	ord := 0
	for j, raw := range d.lines {
		m := jointCheckLine.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		ord++
		line := j + 1
		jc := &JointCheck{Verdict: m[1]}
		rest := m[2]
		if h := jointHome.FindStringSubmatch(rest); h != nil {
			jc.Home = strings.TrimSpace(h[1])
			jc.Open = strings.HasPrefix(strings.ToUpper(jc.Home), "OPEN")
		}
		if jc.Verdict == "fired" {
			head := rest
			if k := strings.Index(head, "(home:"); k >= 0 {
				head = head[:k]
			}
			for _, t := range strings.Split(head, ",") {
				t = strings.TrimSpace(t)
				if jointTarget.MatchString(t) {
					jc.Targets = append(jc.Targets, t)
				}
			}
		}
		e := Element{Kind: ident.JointCheck, Key: strconv.Itoa(ord), Label: collapse(strings.TrimSpace(raw)),
			LineStart: line, LineEnd: line, Joint: jc}
		if n := d.nodeAt(line); n != nil {
			e.Section = n.ID
		}
		d.add(e)
	}
}
