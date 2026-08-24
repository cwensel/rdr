package model

import "strings"

// Epoch identifies which template generation produced a record.
//
// Drift across the corpus is epochal, not chaotic: within an epoch,
// conformance is near-total and every heading is template-drawn. Four
// generations of TEMPLATE.md are distinguishable, and the differences
// between them are additive — each epoch's section set contains its
// predecessor's.
//
// Epochs are detected by FINGERPRINT, not by a stamp in the record. A
// `Template-version:` metadata line was considered and rejected: it is one
// more field to drift, it is copyable-wrong from a neighbouring record
// (the corpus shows metadata being copied wholesale between siblings), and
// it would be absent from every record already frozen — which is most of
// them. The fingerprint separates the four cleanly with no cooperation
// from the author. Add a stamp only if a future epoch is fingerprint-
// ambiguous against its predecessor.
type Epoch int

const (
	// EpochUnknown means no epoch fingerprint matched.
	EpochUnknown Epoch = iota
	// EpochA is the original template: no Profile, no Evidence Record
	// Method field, Finalization Gate responses inlined in the record.
	EpochA
	// EpochB adds the Profile field, Seam Lineage, and the Load-Bearing
	// Decisions section.
	EpochB
	// EpochC externalises the gate: the Finalization Gate body is
	// replaced at lock by a one-line pointer to gate.md.
	EpochC
	// EpochD promotes Critical Assumptions from `###` to `##` and puts
	// the joint-decision status qualifier into use.
	EpochD
)

func (e Epoch) String() string {
	switch e {
	case EpochA:
		return "A"
	case EpochB:
		return "B"
	case EpochC:
		return "C"
	case EpochD:
		return "D"
	}
	return "unknown"
}

// TemplateEpoch is one generation of TEMPLATE.md.
type TemplateEpoch struct {
	// Epoch is the generation this entry describes.
	Epoch Epoch
	// Summary is the one-line description of what this epoch introduced.
	Summary string
	// Sections is the epoch's section table in template order.
	Sections []Section
	// MetadataFields is the epoch's Metadata block field set, in order.
	MetadataFields []string
	// EvidenceFields is the epoch's Evidence Record field set, in order.
	// Empty for EpochA, which has no Evidence Record apparatus.
	EvidenceFields []string
	// Vocabularies are the closed value sets in force. They do not vary
	// by epoch — the vocabularies only ever gained members, and the
	// ObservedAccepted tier already carries what the frozen records
	// wrote — so every epoch points at the same four.
	Vocabularies []Vocabulary
}

// Fingerprint is what a scanner observes about a record, reduced to the
// handful of signals that separate the epochs.
type Fingerprint struct {
	// HasProfile reports a `**Profile**` metadata field.
	HasProfile bool
	// HasSeamLineage reports a `**Seam Lineage**` metadata field.
	HasSeamLineage bool
	// HasLoadBearingDecisions reports a Load-Bearing Decisions section.
	HasLoadBearingDecisions bool
	// HasMethodField reports at least one Evidence Record `**Method**`
	// sub-bullet.
	HasMethodField bool
	// HasGatePointer reports that the Finalization Gate body is a
	// pointer to gate.md rather than inlined responses.
	HasGatePointer bool
	// CriticalAssumptionsLevel is the heading level Critical Assumptions
	// was written at: 2, 3, or 0 when the section is absent.
	CriticalAssumptionsLevel int
	// HasJointDecisionQualifier reports a joint-decision status
	// qualifier on the live Status value.
	HasJointDecisionQualifier bool
	// HasMetadataBlock reports a Metadata section carrying at least the
	// Status field. It is the weakest signal — it says only "this is an
	// RDR" — and it is what places the oldest records, which predate the
	// Evidence Record apparatus entirely and carry no Critical
	// Assumptions section to read a level from.
	HasMetadataBlock bool
}

// DetectEpoch places a record in a template generation.
//
// The signals are read in newest-first order, because the epochs are
// additive: a record showing a later epoch's signal is of that epoch even
// though it also shows every earlier one.
//
//   - D is Critical Assumptions at `##`, or a joint-decision qualifier.
//   - C is a gate.md pointer in place of an inlined gate body.
//   - B is a Profile field, Seam Lineage, or Load-Bearing Decisions.
//   - A is anything else that is still recognisably an RDR — including
//     the oldest records, which have no Critical Assumptions section at
//     all and verified their claims under the legacy `API Verification`
//     and `Dependency Source Verification` headings that alias.go maps.
//
// Critical Assumptions at `##` is the strongest D signal because it is a
// structural change the author cannot omit; the joint-decision qualifier
// is a second D signal because it only exists from D onward, but it is
// written by a minority of records, so it cannot carry the detection
// alone.
//
// EpochUnknown is returned only when the fingerprint shows nothing at all
// — no metadata block, no Critical Assumptions section, no Evidence
// Records — which means the input is not an RDR rather than that it is of
// an unrecognised epoch.
func DetectEpoch(f Fingerprint) Epoch {
	switch {
	case f.CriticalAssumptionsLevel == 2 || f.HasJointDecisionQualifier:
		return EpochD
	case f.HasGatePointer:
		return EpochC
	case f.HasProfile || f.HasSeamLineage || f.HasLoadBearingDecisions:
		return EpochB
	case f.CriticalAssumptionsLevel == 3 || f.HasMethodField || f.HasMetadataBlock:
		return EpochA
	}
	return EpochUnknown
}

// EpochOf returns the table entry for an epoch. It returns the epoch-D
// entry for EpochUnknown, on the principle that an unrecognised record is
// read against the current template rather than not read at all.
func EpochOf(e Epoch) TemplateEpoch {
	for _, te := range Epochs {
		if te.Epoch == e {
			return te
		}
	}
	return EpochDTable
}

// Epochs is the epoch table, oldest first.
var Epochs = []TemplateEpoch{EpochATable, EpochBTable, EpochCTable, EpochDTable}

// vocabularies is the shared vocabulary set. See TemplateEpoch.Vocabularies
// for why it does not vary by epoch.
var vocabularies = []Vocabulary{
	StatusVocabulary, TypeVocabulary, ProfileVocabulary, MethodVocabulary,
}

// EpochDTable is the CURRENT template. It is generated from nothing — it
// is hand-maintained — and TemplateModelMatchesFile in template_test.go
// asserts, against TEMPLATE.md itself, that every name, level, class and
// position below is right. A TEMPLATE.md edit that changes a section
// updates this table in the same commit or that test fails.
var EpochDTable = TemplateEpoch{
	Epoch:   EpochD,
	Summary: "Critical Assumptions promoted to ##; joint-decision status qualifier in use",
	Sections: []Section{
		{Name: "Metadata", Level: 2, Class: Required, Grammar: GrammarMetadataFields},
		{Name: "Problem Statement", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Critical Assumptions", Level: 2, Class: Required, Grammar: GrammarEvidenceRecords},
		{Name: "Proposed Solution", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Approach", Level: 3, Class: Required, Parent: "Proposed Solution", Grammar: GrammarProse},
		{Name: "Technical Design", Level: 3, Class: Required, Parent: "Proposed Solution", Grammar: GrammarProse},
		{Name: "Normative Contracts", Level: 4, Class: Required, Parent: "Technical Design", Grammar: GrammarNormativeBlocks},
		{Name: "Load-Bearing Decisions", Level: 4, Class: Conditional, Parent: "Technical Design", Grammar: GrammarProse},
		{Name: "Round-Trip / Inverse Invariants", Level: 4, Class: Conditional, Parent: "Technical Design", Grammar: GrammarProse},
		{Name: "Illustrative Code", Level: 4, Class: Required, Parent: "Technical Design", Grammar: GrammarProse},
		{Name: "Capability Dependencies", Level: 3, Class: Conditional, Parent: "Proposed Solution", Grammar: GrammarTable},
		{Name: "Existing Infrastructure Audit", Level: 3, Class: Conditional, Parent: "Proposed Solution", Grammar: GrammarTable},
		{Name: "Decision Rationale", Level: 3, Class: Required, Parent: "Proposed Solution", Grammar: GrammarProse},
		{Name: "Alternatives Considered", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Alternative 1: [Name]", Level: 3, Class: Conditional, Parent: "Alternatives Considered", Grammar: GrammarScaffold},
		{Name: "Briefly Rejected", Level: 3, Class: Required, Parent: "Alternatives Considered", Grammar: GrammarProse},
		{Name: "Context", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Background", Level: 3, Class: Required, Parent: "Context", Grammar: GrammarProse},
		{Name: "Technical Environment", Level: 3, Class: Required, Parent: "Context", Grammar: GrammarProse},
		{Name: "Research Findings", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Investigation", Level: 3, Class: Required, Parent: "Research Findings", Grammar: GrammarProse},
		{Name: "Key Discoveries", Level: 3, Class: Required, Parent: "Research Findings", Grammar: GrammarProse},
		{Name: "Trade-offs", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Consequences", Level: 3, Class: Required, Parent: "Trade-offs", Grammar: GrammarProse},
		{Name: "Risks and Mitigations", Level: 3, Class: Required, Parent: "Trade-offs", Grammar: GrammarProse},
		{Name: "Failure Modes", Level: 3, Class: Required, Parent: "Trade-offs", Grammar: GrammarProse},
		{Name: "Implementation Plan", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Prerequisites", Level: 3, Class: Required, Parent: "Implementation Plan", Grammar: GrammarProse},
		{Name: "Minimum Viable Validation", Level: 3, Class: Required, Parent: "Implementation Plan", Grammar: GrammarProse},
		{Name: "Phase 1: Code Implementation", Level: 3, Class: Required, Parent: "Implementation Plan", Grammar: GrammarScaffold},
		{Name: "Step 1: [Title]", Level: 4, Class: Conditional, Parent: "Phase 1: Code Implementation", Grammar: GrammarScaffold},
		{Name: "Step 2: [Title]", Level: 4, Class: Conditional, Parent: "Phase 1: Code Implementation", Grammar: GrammarScaffold},
		{Name: "Phase 2: Operational Activation", Level: 3, Class: Conditional, Parent: "Implementation Plan", Grammar: GrammarScaffold},
		{Name: "Activation Step 1: [Title]", Level: 4, Class: Conditional, Parent: "Phase 2: Operational Activation", Grammar: GrammarScaffold},
		{Name: "Day 2 Operations", Level: 3, Class: Conditional, Parent: "Implementation Plan", Grammar: GrammarTable},
		{Name: "New Dependencies", Level: 3, Class: Conditional, Parent: "Implementation Plan", Grammar: GrammarProse},
		{Name: "Validation", Level: 2, Class: Required, Grammar: GrammarProse},
		{Name: "Testing Strategy", Level: 3, Class: Required, Parent: "Validation", Grammar: GrammarProse},
		{Name: "Performance Expectations", Level: 3, Class: Conditional, Parent: "Validation", Grammar: GrammarProse},
		{Name: "Finalization Gate", Level: 2, Class: Required, Grammar: GrammarGatePointer},
		{Name: "Contradiction Check", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Assumption Verification", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Scope Verification", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Cross-Cutting Concerns", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "Proportionality", Level: 3, Class: Required, Parent: "Finalization Gate", Grammar: GrammarProse},
		{Name: "References", Level: 2, Class: Required, Grammar: GrammarProse},
	},
	MetadataFields: MetadataFields,
	EvidenceFields: EvidenceFields,
	Vocabularies:   vocabularies,
}

// EpochCTable differs from D in one place: Critical Assumptions sits at
// `###` under Proposed Solution rather than at `##`. The gate.md pointer
// that names this epoch is a body change inside Finalization Gate, not a
// section change, so it does not appear in the section table — it is a
// Grammar and a Fingerprint signal.
var EpochCTable = TemplateEpoch{
	Epoch:          EpochC,
	Summary:        "Finalization Gate body externalised to gate.md; Critical Assumptions still at ###",
	Sections:       withCriticalAssumptionsAtLevel3(EpochDTable.Sections),
	MetadataFields: MetadataFields,
	EvidenceFields: EvidenceFields,
	Vocabularies:   vocabularies,
}

// EpochBTable has the Profile / Seam Lineage / Load-Bearing Decisions
// apparatus but keeps the Finalization Gate responses inlined in the
// record.
var EpochBTable = TemplateEpoch{
	Epoch:          EpochB,
	Summary:        "Profile, Seam Lineage and Load-Bearing Decisions added; gate responses still inlined",
	Sections:       withGatePointer(EpochCTable.Sections, GrammarProse),
	MetadataFields: MetadataFields,
	EvidenceFields: EvidenceFields,
	Vocabularies:   vocabularies,
}

// EpochATable is the original template: no Profile or Seam Lineage
// metadata, no Load-Bearing Decisions section, no Evidence Record field
// set, gate responses inlined. Records of this epoch carry Critical
// Assumptions as prose or as the legacy API Verification / Dependency
// Source Verification sub-sections that alias.go maps.
var EpochATable = TemplateEpoch{
	Epoch:    EpochA,
	Summary:  "original template: no Profile, no Evidence Record apparatus, gate responses inlined",
	Sections: without(EpochBTable.Sections, "Load-Bearing Decisions", "Round-Trip / Inverse Invariants"),
	// Overrides is NOT stripped: the corpus shows it in records that
	// predate Profile and Seam Lineage entirely, so it belongs to the
	// original field set even though it is written less often than
	// Profile (it is omitted when a record overrides nothing).
	MetadataFields: without1(MetadataFields, "Profile", "Seam Lineage", "Cluster"),
	EvidenceFields: nil,
	Vocabularies:   vocabularies,
}

// SectionByName finds a section in an epoch by its canonical name.
func (te TemplateEpoch) SectionByName(name string) (Section, bool) {
	for _, s := range te.Sections {
		if s.Name == name {
			return s, true
		}
	}
	return Section{}, false
}

// --- derivation helpers ----------------------------------------------
//
// The older epoch tables are DERIVED from the current one rather than
// spelled out. That is deliberate: the epochs are additive, so writing
// each in full would mean four copies of the same 46 rows drifting apart,
// and only the current table has a mechanical check against TEMPLATE.md.
// Deriving keeps that one checked table authoritative and states each
// older epoch as the delta that actually defines it.

func withCriticalAssumptionsAtLevel3(in []Section) []Section {
	out := make([]Section, len(in))
	copy(out, in)
	for i := range out {
		if out[i].Name == "Critical Assumptions" {
			out[i].Level = 3
			out[i].Parent = "Proposed Solution"
		}
	}
	return out
}

func withGatePointer(in []Section, g Grammar) []Section {
	out := make([]Section, len(in))
	copy(out, in)
	for i := range out {
		if out[i].Name == "Finalization Gate" {
			out[i].Grammar = g
		}
	}
	return out
}

func without(in []Section, names ...string) []Section {
	out := make([]Section, 0, len(in))
	for _, s := range in {
		if !contains(names, s.Name) {
			out = append(out, s)
		}
	}
	return out
}

func without1(in []string, names ...string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !contains(names, s) {
			out = append(out, s)
		}
	}
	return out
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if strings.EqualFold(h, needle) {
			return true
		}
	}
	return false
}
