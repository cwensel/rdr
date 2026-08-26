package scan

import (
	"regexp"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// This file is the resilience contract's second half. The outline covers
// every line; this pass classifies every LABELLED BULLET — `- **Label**:
// value` at any indent, outside fences — against the template model, so
// that each one is a metadata field, an element's field, a section's
// field, or a warning. Nothing is dropped: a label the model does not know
// is recorded as the author's (match `author`), and only a label in a
// closed block (Metadata) warns.

// Field is one labelled bullet, classified.
type Field struct {
	// Label is the bold text as written, whitespace collapsed.
	Label string `json:"label"`
	// Canonical is the template label it matched, or "" for an author's.
	Canonical string `json:"canonical,omitempty"`
	// Match says how (model.MatchKind): exact, case-variant, prefix,
	// legacy-alias, recognized-unmapped, scaffold-instance, author.
	Match string `json:"match"`
	// Value is the text after the colon, joined across wrapped lines per
	// model.ValueContinues. A nested bullet is the field's own
	// sub-structure and is not part of the value.
	Value string `json:"value"`
	// Section is the innermost outline node the bullet sits in; Element
	// is the element, when it sits inside one.
	Section   string `json:"section"`
	Element   string `json:"element,omitempty"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	// Status is the normalised form of a Status value: the record's
	// lifecycle status in Metadata, the Evidence Record's in an assumption.
	Status *Status `json:"status,omitempty"`
	// Method is the parsed form of an Evidence Record Method value.
	Method *Method `json:"method,omitempty"`
}

// Status is a Status value normalised to {value, qualifier, raw}.
type Status struct {
	Value     string `json:"value"`
	Qualifier string `json:"qualifier,omitempty"`
	// Form names the qualifier grammar a lifecycle status matched
	// (model.QualifierForm); absent for an assumption status.
	Form string `json:"form,omitempty"`
	// Tier is the value's standing in its vocabulary (model.Tier).
	Tier string `json:"tier"`
	// Placeholder marks the template legend left unfilled.
	Placeholder bool `json:"placeholder,omitempty"`
	// OpenJointDecisions names the joint decisions this record is still
	// waiting on, and ONLY those. It is present whenever the record's
	// status is a lifecycle status, empty list included, so a consumer can
	// tell "none open" from "this field does not apply".
	//
	// It exists because the qualifier is prose and prose lies to a regex.
	// A record that has FINISHED answering its joint decisions writes them
	// down — `Final [all joint decisions answered — JDR 0001 §D8 (§JD-9),
	// §D9 (§JD-19)…]` — so scraping `JD-\d+` out of the qualifier reports
	// five open decisions on a record with none. Only the routing form,
	// `Final [joint decision → <home>: …]`, means one is open, and the
	// projector already knows which form it matched.
	OpenJointDecisions []string `json:"open_joint_decisions,omitempty"`
	Raw                string   `json:"raw"`
}

// Method is a parsed Method value.
type Method struct {
	// Members are the sanctioned-or-not member labels, glosses stripped.
	Members []string `json:"members"`
	// OffVocabulary names the members that are in no tier.
	OffVocabulary []string `json:"off_vocabulary,omitempty"`
	Raw           string   `json:"raw"`
}

// Coverage is the zero-silent-drop metric: how many of the record's
// non-blank lines the projector could not classify. Unclassified lines
// are exactly the lines a warning covers — a warning is the itemised
// form, this is the rate — so the number is 0 when the record is
// template-conformant, and a rise across a corpus after a
// TEMPLATE.md change is the drift alarm.
type Coverage struct {
	// Lines is the count of non-blank lines.
	Lines int `json:"lines"`
	// Unclassified is the count of non-blank lines inside a warning's range.
	Unclassified int `json:"unclassified"`
	// Rate is Unclassified / Lines, 0 for an empty record.
	Rate float64 `json:"rate"`
}

// labelledBullet matches a labelled bullet at any indent: `- **Label**:
// value` and the `- **Label:** value` variant with the colon inside the
// bold. The label may not contain `*`.
var labelledBullet = regexp.MustCompile(`^(\s*)(?:[-*+]|\d+[.)])\s+\*\*([^*]+?)(?::\*\*|\*\*\s*:)\s*(.*)$`)

// fields classifies every labelled bullet. It runs after extract, so
// elements exist to own their fields, and before count.
func (d *Document) fields() {
	d.Metadata = []Field{}
	d.Fields = []Field{}
	meta := d.metadataNode()
	for i := 1; i <= len(d.lines); i++ {
		if d.fenced[i-1] {
			continue
		}
		m := labelledBullet.FindStringSubmatch(d.lines[i-1])
		if m == nil {
			continue
		}
		if el := d.elementAt(i); el != nil && el.LineStart == i {
			continue // the bullet is the element's own lead line
		}
		f := Field{Label: strings.TrimSpace(spaces.ReplaceAllString(m[2], " ")), LineStart: i, LineEnd: i}
		f.Label = strings.TrimSpace(strings.TrimSuffix(f.Label, ":"))
		val := m[3]
		for j := i + 1; j <= len(d.lines) && model.ValueContinues(d.lines[j-1]); j++ {
			val += " " + strings.TrimSpace(d.lines[j-1])
			f.LineEnd = j
		}
		f.Value = strings.TrimSpace(val)
		node := d.nodeAt(i)
		f.Section = node.ID

		switch el := d.elementAt(i); {
		case meta != nil && node == meta && len(m[1]) == 0:
			d.metadataField(&f)
			d.Metadata = append(d.Metadata, f)
		case el != nil && el.Kind == ident.Assumption:
			f.Element = el.ID
			lm := model.LookupLabel(model.FieldSetOf("Critical Assumptions"), f.Label)
			f.Match, f.Canonical = lm.Kind.String(), lm.Canonical
			switch lm.Canonical {
			case "Status":
				f.Status = assumptionStatus(f.Value)
			case "Method":
				f.Method = method(f.Value)
			}
			el.Fields = append(el.Fields, f)
		default:
			lm := model.LookupLabel(model.FieldSetOf(d.templateSectionOf(node)), f.Label)
			f.Match, f.Canonical = lm.Kind.String(), lm.Canonical
			if el != nil {
				f.Element = el.ID
				el.Fields = append(el.Fields, f)
			} else {
				d.Fields = append(d.Fields, f)
			}
		}
		i = f.LineEnd
	}
}

var spaces = regexp.MustCompile(`\s+`)

// metadataField classifies a column-zero bullet of the Metadata block.
// Metadata is the one closed block: a label in no tier and no alias is a
// warning, because the flow reads these fields by name and a misspelt
// one is a field the flow will not find.
func (d *Document) metadataField(f *Field) {
	te := model.Template
	m := model.LookupField(te, f.Label)
	f.Match = m.Kind.String()
	switch m.Kind {
	case model.MatchExact, model.MatchCaseVariant:
		f.Canonical = m.Note
	case model.MatchLegacyAlias:
		f.Canonical = model.FieldAliasCanonical(f.Label)
	case model.MatchUnknown:
		// Not an alias either: try the prefix rule before warning, so
		// `Status (as of lock)` is still Status.
		if lm := model.LookupLabel(model.FieldSetOf("Metadata"), f.Label); lm.Kind == model.MatchPrefix {
			f.Match, f.Canonical = lm.Kind.String(), lm.Canonical
		} else {
			f.Match = model.MatchAuthor.String()
			d.warn("field:unknown-to-template", f.LineStart, f.LineEnd,
				"metadata field %q is in neither the template nor the alias table", f.Label)
		}
	}
	if f.Canonical == "Status" {
		f.Status = lifecycleStatus(f.Value)
	}
}

func lifecycleStatus(raw string) *Status {
	s := model.ParseStatus(raw)
	out := &Status{Value: s.Label, Qualifier: s.Qualifier, Tier: s.Tier.String(), Raw: raw}
	if s.QualifierForm != model.NoQualifier {
		out.Form = s.QualifierForm.String()
	}
	out.OpenJointDecisions = openJointDecisions(s)
	return out
}

// jdRef matches a joint-decision anchor as the corpus writes it: `§JD-18`,
// `JD-18`, with or without the section mark.
var jdRef = regexp.MustCompile(`§?\bJD-(\d+[a-z]?)\b`)

// openJointDecisions reads the anchors out of a qualifier, but only when
// the qualifier's FORM says one is open.
//
// The form is the whole rule. `joint-decision` is the routing grammar —
// `Final [joint decision → JDR 0001 §JD-18: …]` — and names what this
// record waits on. Every other form that mentions a JD is talking ABOUT
// them, usually to say they are done, and a reader that does not check
// the form turns a finished record into five open questions.
func openJointDecisions(s model.StatusValue) []string {
	if s.QualifierForm != model.QualifierJointDecision {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, m := range jdRef.FindAllStringSubmatch(s.Qualifier, -1) {
		id := "JD-" + m[1]
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func assumptionStatus(raw string) *Status {
	s := model.ParseAssumptionStatus(raw)
	return &Status{Value: s.Value, Qualifier: s.Qualifier, Tier: s.Tier.String(), Placeholder: s.Placeholder, Raw: raw}
}

func method(raw string) *Method {
	mv := model.ParseMethod(raw)
	out := &Method{Members: []string{}, Raw: raw}
	for _, m := range mv.Members {
		out.Members = append(out.Members, m.Label)
	}
	for _, m := range mv.OffVocabulary {
		out.OffVocabulary = append(out.OffVocabulary, m.Label)
	}
	return out
}

// metadataNode is the node classified as the Metadata section, or nil.
func (d *Document) metadataNode() *Node {
	for _, n := range d.nodes {
		if n.Canonical == "Metadata" {
			return n
		}
	}
	return nil
}

// nodeAt returns the innermost node containing a line.
func (d *Document) nodeAt(line int) *Node {
	var out *Node
	for _, n := range d.nodes {
		if n.LineStart <= line && line <= n.LineEnd {
			out = n
		}
	}
	return out
}

// elementAt returns the element whose range contains the line, or nil.
// Elements of different kinds do not nest, and the outer one — an
// Alternative section holding a Risk bullet — is the one a field
// belongs to, so the first match in document order wins.
func (d *Document) elementAt(line int) *Element {
	for i := range d.Elements {
		e := &d.Elements[i]
		if e.LineStart <= line && line <= e.LineEnd {
			return e
		}
	}
	return nil
}

// templateSectionOf returns the canonical name of the nearest
// template-mapped node at or above n, or "" when there is none.
func (d *Document) templateSectionOf(n *Node) string {
	for n != nil {
		if n.Canonical != "" {
			return n.Canonical
		}
		n = d.nodeByID(n.Parent)
	}
	return ""
}

func (d *Document) nodeByID(id string) *Node {
	if id == "" {
		return nil
	}
	for _, n := range d.nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// coverage computes the unclassified-line rate from the warnings.
//
// EDGE WARNINGS ARE EXCLUDED. The rate measures what the projector could
// not STRUCTURALLY classify — a heading, a bullet, a field it could not
// place — and it is the drift alarm for TEMPLATE.md changes (README §The
// resilience contract). A line carrying a reference form the edge grammar
// cannot type is fully classified as structure; only the relation inside
// it is unread. Counting it would move the alarm for a reason that has
// nothing to do with template drift, and would mask a real rise behind
// prose the corpus was always free to write.
func (d *Document) coverage() {
	flagged := make([]bool, len(d.lines)+1)
	for _, w := range d.Warnings {
		if strings.HasPrefix(w.Code, "edge:") {
			continue
		}
		for i := w.LineStart; i <= w.LineEnd && i <= len(d.lines); i++ {
			if i >= 1 {
				flagged[i] = true
			}
		}
	}
	for i := 1; i <= len(d.lines); i++ {
		if strings.TrimSpace(d.lines[i-1]) == "" {
			continue
		}
		d.Coverage.Lines++
		if flagged[i] {
			d.Coverage.Unclassified++
		}
	}
	if d.Coverage.Lines > 0 {
		d.Coverage.Rate = float64(d.Coverage.Unclassified) / float64(d.Coverage.Lines)
	}
}

// MetadataValue returns the value of a Metadata field by its canonical
// name, or "" when the record does not carry it. Canonical is what a
// caller means: `Status` should find `**Status**` and the case- and
// alias-variants the model maps to it, which is exactly what
// classification already decided. A field the model does not know is
// matched by its written label, so an author's own metadata is reachable
// too.
func (d *Document) MetadataValue(name string) string {
	for _, f := range d.Metadata {
		if strings.EqualFold(f.Canonical, name) {
			return f.Value
		}
	}
	for _, f := range d.Metadata {
		if strings.EqualFold(f.Label, name) {
			return f.Value
		}
	}
	return ""
}
