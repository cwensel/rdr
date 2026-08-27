package model

// Template is the CURRENT TEMPLATE.md, as a table.
//
// There is exactly one. Records are read against it whatever their age:
// a frozen record is not re-judged by the template that produced it, it
// is reported as it differs from the template in force now. That is the
// whole of the reader's template model.
//
// It is hand-maintained, and TestTemplateMatchesTemplateFile in template_test.go
// asserts, against TEMPLATE.md itself, that every name, level, class and
// position below is right. A TEMPLATE.md edit that changes a section
// updates this table in the same commit or that test fails.
var Template = TemplateTable{
	Sections: []Section{
		{Name: "Metadata", Level: 2, Class: Required, Grammar: GrammarMetadataFields},
		{Name: "Problem Statement", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Critical Assumptions", Level: 2, Class: Required, Grammar: GrammarEvidenceRecords, Keys: true},
		{Name: "Proposed Solution", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Approach", Level: 3, Class: Required, Parent: "Proposed Solution", Grammar: GrammarProse},
		{Name: "Technical Design", Level: 3, Class: Required, Parent: "Proposed Solution", Grammar: GrammarProse},
		{Name: "Normative Contracts", Level: 4, Class: Required, Parent: "Technical Design", Grammar: GrammarNormativeBlocks, Keys: true},
		{Name: "Load-Bearing Decisions", Level: 4, Class: Conditional, Parent: "Technical Design", Grammar: GrammarProse, Keys: true},
		{Name: "Round-Trip / Inverse Invariants", Level: 4, Class: Conditional, Parent: "Technical Design", Grammar: GrammarProse, Keys: false},
		{Name: "Illustrative Code", Level: 4, Class: Required, Parent: "Technical Design", Grammar: GrammarProse},
		{Name: "Capability Dependencies", Level: 3, Class: Conditional, Parent: "Proposed Solution", Grammar: GrammarTable},
		{Name: "Existing Infrastructure Audit", Level: 3, Class: Conditional, Parent: "Proposed Solution", Grammar: GrammarTable},
		{Name: "Decision Rationale", Level: 3, Class: Required, Parent: "Proposed Solution", Grammar: GrammarProse},
		{Name: "Alternatives Considered", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Alternative 1: [Name]", Level: 3, Class: Conditional, Parent: "Alternatives Considered", Grammar: GrammarScaffold, Keys: true},
		{Name: "Briefly Rejected", Level: 3, Class: Required, Parent: "Alternatives Considered", Grammar: GrammarProse, Keys: false},
		{Name: "Context", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Background", Level: 3, Class: Required, Parent: "Context", Grammar: GrammarProse},
		{Name: "Technical Environment", Level: 3, Class: Required, Parent: "Context", Grammar: GrammarProse},
		{Name: "Research Findings", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Investigation", Level: 3, Class: Required, Parent: "Research Findings", Grammar: GrammarProse},
		{Name: "Key Discoveries", Level: 3, Class: Required, Parent: "Research Findings", Grammar: GrammarProse},
		{Name: "Trade-offs", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Consequences", Level: 3, Class: Required, Parent: "Trade-offs", Grammar: GrammarProse},
		{Name: "Risks and Mitigations", Level: 3, Class: Required, Parent: "Trade-offs", Grammar: GrammarProse},
		{Name: "Failure Modes", Level: 3, Class: Required, Parent: "Trade-offs", Grammar: GrammarProse, Keys: false},
		{Name: "Implementation Plan", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Prerequisites", Level: 3, Class: Required, Parent: "Implementation Plan", Grammar: GrammarProse},
		{Name: "Minimum Viable Validation", Level: 3, Class: Required, Parent: "Implementation Plan", Grammar: GrammarProse, Keys: false},
		{Name: "Phase 1: Code Implementation", Level: 3, Class: Required, Parent: "Implementation Plan", Grammar: GrammarScaffold},
		{Name: "Step 1: [Title]", Level: 4, Class: Conditional, Parent: "Phase 1: Code Implementation", Grammar: GrammarScaffold},
		{Name: "Step 2: [Title]", Level: 4, Class: Conditional, Parent: "Phase 1: Code Implementation", Grammar: GrammarScaffold},
		{Name: "Phase 2: Operational Activation", Level: 3, Class: Conditional, Parent: "Implementation Plan", Grammar: GrammarScaffold},
		{Name: "Activation Step 1: [Title]", Level: 4, Class: Conditional, Parent: "Phase 2: Operational Activation", Grammar: GrammarScaffold},
		{Name: "Day 2 Operations", Level: 3, Class: Conditional, Parent: "Implementation Plan", Grammar: GrammarTable},
		{Name: "New Dependencies", Level: 3, Class: Conditional, Parent: "Implementation Plan", Grammar: GrammarProse},
		{Name: "Validation", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Testing Strategy", Level: 3, Class: Required, Parent: "Validation", Grammar: GrammarProse, Keys: true},
		{Name: "Performance Expectations", Level: 3, Class: Conditional, Parent: "Validation", Grammar: GrammarProse},
		{Name: "Finalization Gate", Level: 2, Class: Required, Grammar: GrammarGatePointer},
		{Name: "Contradiction Check", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Assumption Verification", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Scope Verification", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Cross-Cutting Concerns", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Proportionality", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "References", Level: 2, Class: Required, Grammar: GrammarProse},
	}, MetadataFields: MetadataFields,
	EvidenceFields: EvidenceFields,
	Vocabularies:   vocabularies,
}

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

// vocabularies is the shared vocabulary set.
var vocabularies = []Vocabulary{
	StatusVocabulary, TypeVocabulary, ProfileVocabulary, MethodVocabulary,
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
