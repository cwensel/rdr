package model

import "strings"

// Template is the CURRENT TEMPLATE.md, as a table.
//
// There is exactly one. Records are read against it whatever their age: a
// frozen record is not re-judged by the template that produced it, it is
// reported as it differs from the template in force now. That is the
// whole of the reader's template model.
//
// It used to be a hand-maintained literal that a test compared against
// TEMPLATE.md at test time — one fact stated twice, with a test standing
// between the two copies to catch them drifting. The binary now reads the
// file, so there is no second copy left to disagree with it.
func Template() TemplateTable { return current().Table }

// TemplateTable is TEMPLATE.md reduced to what a reader needs.
type TemplateTable struct {
	// Sections is the section table in template order.
	Sections []Section
	// MetadataFields is the Metadata block field set, in order.
	MetadataFields []string
	// EvidenceFields is the Evidence Record field set, in order.
	EvidenceFields []string
	// Vocabularies are the closed value sets in force.
	Vocabularies []Vocabulary
}

// SectionByName finds a section by its canonical name.
func (te TemplateTable) SectionByName(name string) (Section, bool) {
	for _, s := range te.Sections {
		if s.Name == name {
			return s, true
		}
	}
	return Section{}, false
}

// MetadataFields is the Metadata block's canonical field set in template
// order. Presence is not uniform across the corpus — Profile, Seam
// Lineage, Overrides and Cluster arrived later, and Predecessors,
// Overrides, Seam Lineage and Cluster are omitted when they have no value.
func MetadataFields() []string { return current().Table.MetadataFields }

// EvidenceFields is the Critical Assumptions Evidence Record field set:
// the bold-label sub-bullets under an `- **A<N> [Statement]**` bullet, in
// template order. TEMPLATE.md states all four; a record missing one is
// incomplete, not foreign.
func EvidenceFields() []string { return current().Table.EvidenceFields }

// TemplateNormativeBodies are the bodies of the ```normative fences
// TEMPLATE.md writes as its OWN example.
//
// The template shows an author what a contract looks like, and that
// example is fenced exactly as a real contract is. A record that was
// seeded and not yet authored still carries it, and the projector — which
// reads every normative fence document-wide, correctly, because a fence
// is a contract wherever it sits — counted the template's example as the
// record's C1. Thirteen records in the reference corpus project it, and
// eleven of them are in-flight Drafts whose `contracts` count therefore
// said one when the true count is zero.
//
// The set is DERIVED, never listed. Delete the example from TEMPLATE.md
// and this empties itself; change it and this follows. A literal here
// would be the same defect one layer down — the template's words spelled
// in Go — and it would go stale the first time the example is edited.
//
// Matched on the fence BODY rather than its label: the label is `**C1**`
// in the template and in half the corpus's real contracts too, so it
// discriminates nothing. The body is the template's own text, and a
// record that has authored a real contract has by construction replaced
// it. Measured: 13 of 329 corpus fences match, and all 13 are the
// unauthored example.
func TemplateNormativeBodies() map[string]bool {
	return normativeBodies(current().TemplateSource)
}

// normativeBodies collects the text inside every ```normative fence,
// whitespace-trimmed so indentation drift is not a difference.
func normativeBodies(src string) map[string]bool {
	out := map[string]bool{}
	var cur []string
	open := false
	for _, line := range strings.Split(src, "\n") {
		if !open && NormativeFenceOpen.MatchString(line) {
			open, cur = true, nil
			continue
		}
		if open && FenceDelimiter.MatchString(line) {
			if body := strings.TrimSpace(strings.Join(cur, "\n")); body != "" {
				out[body] = true
			}
			open = false
			continue
		}
		if open {
			cur = append(cur, line)
		}
	}
	return out
}

// TemplateLine reports whether a line's text is one TEMPLATE.md itself
// writes.
//
// It is the general form of the question every "did the author write
// this?" rule asks. A guidance block, an instructional bullet list, a
// worked example, a blockquote of advice: a record that was seeded and
// not yet authored carries all of them verbatim, and each would otherwise
// need its own rule. Comparing against the template's own lines needs no
// rule per shape, and it cannot go stale — edit TEMPLATE.md and the
// answer follows.
//
// SHORT LINES ARE NOT COMPARED. A bare `**C1**`, a `---`, a `}` or a
// three-word bullet appears in the template and in genuinely authored
// prose alike, so matching them would call authored content template
// text. The floor is the length below which a line carries no
// distinguishing content of its own; measured against the corpus, twelve
// characters separates the template's sentences from the punctuation and
// labels that recur everywhere.
//
// A record's copy may be RE-WRAPPED, so this answers for the line as the
// record spells it and callers that need the wrapped case compare leads
// instead (see AuthoringMarker).
func TemplateLine(s string) bool {
	t := normalizeTemplateLine(s)
	if len(t) < templateLineFloor {
		return false
	}
	return templateLines()[t]
}

// normalizeTemplateLine is the form two copies of one template line are
// compared in. Case, surrounding emphasis and TRAILING PUNCTUATION are
// dropped: records re-wrapped these blocks as they pasted them, and the
// commonest difference is a sentence the template ends with `.` and the
// record continues with `,`. Interior text is untouched, so two lines
// that actually say different things still differ.
func normalizeTemplateLine(s string) string {
	t := strings.TrimSpace(s)
	t = strings.Trim(t, "*_>` ")
	t = strings.TrimRight(t, ".,;:—– ")
	return strings.ToLower(strings.Join(strings.Fields(t), " "))
}

// templateLineFloor is the shortest line worth comparing; below it a line
// is punctuation or a label, not the template's prose.
const templateLineFloor = 12

func templateLines() map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(current().TemplateSource, "\n") {
		if t := normalizeTemplateLine(line); len(t) >= templateLineFloor {
			out[t] = true
		}
	}
	return out
}
