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
	Epoch   string `json:"epoch"`
	Lines   int    `json:"lines"`

	Outline  []Node    `json:"outline"`
	Elements []Element `json:"elements"`
	// Metadata is the Metadata block's fields, classified; Fields is every
	// other labelled bullet that sits inside no element (an element's own
	// fields nest under it). See fields.go.
	Metadata []Field   `json:"metadata"`
	Fields   []Field   `json:"fields"`
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
	// element's ordinal because the author wrote no label.
	Derived bool `json:"derived"`
	// Hash is the short content hash (ident.Hash) over the element's
	// lines; it separates "same ID, same content" from "same ID, changed".
	Hash string `json:"hash"`
	// Section is the ID of the node the element was read from.
	Section   string `json:"section"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	// Transient marks a contract carrying the Transient marker.
	Transient bool `json:"transient,omitempty"`
	// Fields are the labelled bullets inside the element: an assumption's
	// Evidence Record, a failure mode's Visible/Silent/Recovery, a step's
	// Risk. Each is classified against the template (see fields.go).
	Fields []Field `json:"fields,omitempty"`
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
// derived ID — the labelling backlog.
type Counts struct {
	Elements map[ident.Kind]int `json:"elements"`
	Derived  map[ident.Kind]int `json:"derived"`
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
	doc.detectEpoch()
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

// --- epoch -------------------------------------------------------------

var (
	listItem   = regexp.MustCompile(`^(\s*)(?:[-*+]|\d+[.)])\s+(.*)$`)
	numberItem = regexp.MustCompile(`^(\s*)(\d+)[.)]\s+(.*)$`)
	boldLead   = regexp.MustCompile(`^\*\*(.+?)\*\*`)
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

func (d *Document) detectEpoch() {
	f := model.Fingerprint{}
	md := d.metadata()
	_, f.HasProfile = md["Profile"]
	_, f.HasSeamLineage = md["Seam Lineage"]
	if status, ok := md["Status"]; ok {
		f.HasMetadataBlock = true
		f.HasJointDecisionQualifier = model.ParseStatus(status).QualifierForm == model.QualifierJointDecision
	}
	f.HasLoadBearingDecisions = d.findHeading("Load-Bearing Decisions") != nil
	if ca := d.findHeading("Critical Assumptions"); ca != nil {
		f.CriticalAssumptionsLevel = ca.Level
		for i := ca.LineStart; i <= ca.LineEnd; i++ {
			if m := model.EvidenceFieldBullet.FindStringSubmatch(d.lines[i-1]); m != nil && strings.TrimSpace(m[1]) == "Method" {
				f.HasMethodField = true
				break
			}
		}
	}
	if g := d.findHeading("Finalization Gate"); g != nil {
		// The pointer replaces the body: it is the first non-blank line
		// after the heading, before any sub-heading.
		for i := g.LineStart + 1; i <= g.LineEnd; i++ {
			l := strings.TrimSpace(d.lines[i-1])
			if l == "" {
				continue
			}
			if model.Heading.MatchString(l) {
				break
			}
			f.HasGatePointer = model.GatePointer.MatchString(l)
			break
		}
	}
	d.Epoch = model.DetectEpoch(f).String()
}

// --- classification ----------------------------------------------------

// classify maps each heading onto the template of the detected epoch and
// assigns section IDs. Canonical sections take the canonical slug, so a
// case- or level-variant heading, or a legacy alias, resolves to the same
// ID as the section it stands for. Everything else slugs its own text and
// is flagged derived.
func (d *Document) classify() {
	te := model.EpochOf(epochOf(d.Epoch))
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
						"heading %q (level %d) matches no section of epoch %s", n.Heading, n.Level, d.Epoch)
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

func epochOf(s string) model.Epoch {
	for _, e := range model.Epochs {
		if e.Epoch.String() == s {
			return e.Epoch
		}
	}
	return model.EpochUnknown
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
func (d *Document) canonicalNodes(name string) []*Node {
	var out []*Node
	for _, n := range d.nodes {
		if n.Canonical == name {
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
	d.decisions()
	d.listKind(ident.RoundTrip, "Round-Trip / Inverse Invariants", roundTripLabel)
	d.alternatives()
	d.listKind(ident.Rejected, "Briefly Rejected", nil)
	d.listKind(ident.Scenario, "Testing Strategy", nil)
	d.mvv()
	d.listKind(ident.Failure, "Failure Modes", nil)
	d.gate()
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
// carrying an A-label is as written; an unlabelled one (epoch A's
// checkbox bullets) is derived by ordinal, so every assumption in the
// corpus is addressable and the unlabelled ones are counted as backlog.
func (d *Document) assumptions() {
	var items []keyed
	labelled := false
	for _, n := range d.canonicalNodes("Critical Assumptions") {
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
		// assumptions. Only a label-free section (epoch A's checkbox
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
// that shows as a zero count, not a warning, because older epochs wrote
// them that way and a terminal record is never wrong for being of its
// epoch.
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

// decisions reads Load-Bearing Decisions bullets. A bullet whose bold
// label starts with a template decision class is keyed by that class
// (`D-identity`); any other label slugs its own text and is derived.
func (d *Document) decisions() {
	seen := map[string]int{}
	for _, n := range d.canonicalNodes("Load-Bearing Decisions") {
		for _, i := range d.topItems(n.LineStart+1, n.LineEnd) {
			label := itemLabel(d.lines[i-1])
			end := d.itemEnd(i, n.LineEnd)
			e := Element{Kind: ident.Decision, Label: label, Section: n.ID, LineStart: i, LineEnd: end}
			if class := model.DecisionClassOf(label); class != "" {
				e.Key = ident.Slug(class)
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
func (d *Document) listKind(kind ident.Kind, section string, labeller func(string) string) {
	nodes := d.canonicalNodes(section)
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
	}
	// Author-written numbers are keys only if they are unique; a list
	// that restarts at 1 (two phases of scenarios) is addressed by
	// ordinal, with a warning naming the clash.
	counts := map[string]int{}
	for _, it := range items {
		if it.key != "" {
			counts[it.key]++
		}
	}
	for ord, it := range items {
		e := Element{Kind: kind, Label: it.label, Section: it.n.ID, LineStart: it.line, LineEnd: it.end}
		switch {
		case it.key != "" && counts[it.key] == 1:
			e.Key = it.key
		case it.key != "":
			d.warn(strings.ToLower(string(kind))+":duplicate", it.line, it.end,
				"%s item number %s repeats; addressed by ordinal", section, it.key)
			fallthrough
		default:
			e.Key, e.Derived = strconv.Itoa(ord+1), true
		}
		d.add(e)
	}
}

var altOrdinal = regexp.MustCompile(`(?i)^Alt(?:ernative)?\s+(\d+)`)

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
	for _, n := range d.canonicalNodes("Minimum Viable Validation") {
		s, e := d.body(n)
		if s > e {
			continue
		}
		d.add(Element{Kind: ident.MVV, Label: n.Heading, Section: n.ID, LineStart: s, LineEnd: e})
		return
	}
}

// gate projects the inlined gate responses of an epoch A or B record as
// G-<item> elements. With a gate.md pointer the responses live outside the
// record and there is nothing to address here.
func (d *Document) gate() {
	gates := d.canonicalNodes("Finalization Gate")
	if len(gates) == 0 {
		return
	}
	g := gates[0]
	for i := g.LineStart + 1; i <= g.LineEnd; i++ {
		t := strings.TrimSpace(d.lines[i-1])
		if t == "" {
			continue
		}
		if model.GatePointer.MatchString(t) {
			return
		}
		break
	}
	for _, n := range d.nodes {
		if n.Parent != g.ID || n.Level != g.Level+1 {
			continue
		}
		s, e := d.body(n)
		if s > e {
			continue
		}
		el := Element{Kind: ident.Gate, Label: n.Heading, Section: n.ID, LineStart: s, LineEnd: e}
		if key := model.GateItemKey(n.Canonical); key != "" {
			el.Key = key
		} else {
			el.Key, el.Derived = ident.Slug(n.Heading), true
		}
		d.add(el)
	}
}

func (d *Document) count() {
	d.Counts = Counts{Elements: map[ident.Kind]int{}, Derived: map[ident.Kind]int{}, Fields: map[string]int{}}
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
	}
	for _, e := range d.Elements {
		d.Counts.Elements[e.Kind]++
		if e.Derived {
			d.Counts.Derived[e.Kind]++
		}
	}
	for _, n := range d.Outline {
		d.Counts.Elements[ident.Section]++
		if n.Derived {
			d.Counts.Derived[ident.Section]++
		}
	}
}
