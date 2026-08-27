package model

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
