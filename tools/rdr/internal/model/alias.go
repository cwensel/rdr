package model

import "strings"

// MatchKind says how an observed heading or field label was matched
// against the template. A consumer decides from it whether to warn: the
// point of the table is that `unknown-to-template` fires only on
// genuinely foreign sections, so only MatchUnknown is a finding.
type MatchKind int

const (
	// MatchUnknown means nothing in the template or the alias table
	// matched. This is the only kind that warrants an
	// `unknown-to-template` warning.
	MatchUnknown MatchKind = iota
	// MatchExact means the name and the level both match the canonical
	// section.
	MatchExact
	// MatchLevelVariant means the name matches but the heading level
	// does not. It is not a defect: sections have been re-levelled over
	// the template's life, most visibly Critical Assumptions, and a frozen
	// record keeps the level it was written at. It is a reformat in the
	// waiting — the fix is to re-level the record, by hand where a
	// promotion would swallow a following sibling.
	MatchLevelVariant
	// MatchCaseVariant means the name matches only case-insensitively.
	MatchCaseVariant
	// MatchLegacyAlias means the name is a recognised predecessor of a
	// canonical section, mapped through the alias table.
	MatchLegacyAlias
	// MatchScaffoldInstance means the heading is a template scaffold with
	// its placeholder filled in — `### Alternative 2: Buffer the whole
	// frame` for the template's `### Alternative 1: [Name]`. It is
	// template-drawn, so it is not a finding; it is distinguished from
	// MatchExact because the name is the author's, not the template's.
	MatchScaffoldInstance
	// MatchRecognizedUnmapped means the name is recognised as something
	// authors demonstrably wrote, but it has no canonical section to map
	// to — the template never had one and no longer wants one. It is not
	// foreign, so it is not `unknown-to-template`, but there is nothing
	// to project it onto either. Kept distinct so a consumer can report
	// "recognised, no home" separately from "never seen this".
	MatchRecognizedUnmapped
	// MatchAuthorSubsection means an unknown heading nested under a
	// template-mapped section. It is the author's own sub-structure —
	// `### Why not a cache` under Approach — not a foreign section, so
	// it is not a finding; its lines belong to the section above it.
	// Whether such a heading is in fact an unmodelled TEMPLATE.md
	// addition is a corpus-level question (`recs index --coverage`
	// reports headings that recur across records), not a per-record one.
	MatchAuthorSubsection
	// MatchPrefix means an observed bullet label opens with a vocabulary
	// label at a word boundary — `Evidence — the two channel reductions`
	// for `Evidence`. Labels are matched by prefix, not literal, because
	// authors extend them into clauses; see LookupLabel.
	MatchPrefix
	// MatchAuthor means a bullet label in no vocabulary: the author's
	// own field, recorded so it is never dropped, and not a finding.
	MatchAuthor
)

func (m MatchKind) String() string {
	switch m {
	case MatchExact:
		return "exact"
	case MatchLevelVariant:
		return "level-variant"
	case MatchCaseVariant:
		return "case-variant"
	case MatchLegacyAlias:
		return "legacy-alias"
	case MatchScaffoldInstance:
		return "scaffold-instance"
	case MatchRecognizedUnmapped:
		return "recognized-unmapped"
	case MatchAuthorSubsection:
		return "author-subsection"
	case MatchPrefix:
		return "prefix"
	case MatchAuthor:
		return "author"
	case MatchUnknown:
		return "unknown"
	}
	return "unknown"
}

// Match is the result of looking one observed heading up.
type Match struct {
	// Kind says how it matched.
	Kind MatchKind
	// Canonical is the canonical section, or nil when the heading has no
	// canonical home (MatchRecognizedUnmapped and MatchUnknown).
	Canonical *Section
	// ObservedLevel is the level the heading was written at.
	ObservedLevel int
	// Note explains a legacy or unmapped match, for a consumer's message.
	Note string
}

// sectionAlias is one legacy heading and where it maps.
type sectionAlias struct {
	// name is the observed heading text, matched case-insensitively.
	name string
	// canonical is the canonical section it maps to, or "" when the
	// heading is recognised but has no canonical home.
	canonical string
	// note explains the mapping.
	note string
}

// SectionAliases maps legacy heading names onto canonical sections.
//
// Every entry is a heading observed in the frozen corpus, in one of two
// groups: it either has a canonical home or it does not.
//
// MAPPED ENTRIES are genuine predecessors — the same content under the
// name the template used to give it. Both survivors are pre-Evidence-Record
// headings, retired by the commit that introduced the Evidence Record
// apparatus.
//
// A mapped entry is NOT automatically a reformat in the waiting. Whether
// renaming a record's heading to the canonical name is an improvement is a
// question about that record's CONTENT, and for these two the answer is no
// — see PERMANENTLY RETAINED below. The table's job is to keep a frozen
// record's headings classified and its elements addressable; migrating the
// record is a separate decision that the mapping does not license.
//
// UNMAPPED ENTRIES map to "" and classify as MatchRecognizedUnmapped:
// recognised as something authors demonstrably wrote, with no canonical
// section to project onto. That keeps them out of `unknown-to-template`
// without pretending they have a home. Two reasons a heading lands here —
// the template never adopted it, or the template deliberately rehomed its
// content elsewhere (Open Questions -> the Status qualifier, Rationale ->
// Decision Rationale, Premortem -> a verdict line plus Risks and
// Mitigations / Failure Modes).
//
// An entry earns deletion only when no record in any consuming corpus
// writes the heading any more. At that point it classifies nothing and
// costs a lookup. Deleting an entry a record still writes reclassifies
// that heading as MatchUnknown and fires `unknown-to-template` on a
// section that is merely old — which is why the audit is per-corpus, not
// per-intuition.
//
// WHAT AN ENTRY DOES NOT DO. It does not hold ids up. A section's id is a
// slug of its own heading text, and the elements under it carry their own
// authored keys — the 21 citations onto `cli/0035:D-*` from six records
// resolve through the authored `D-N` bullet labels, not through any entry
// here. An earlier version of this comment claimed emptying the table drops
// 18 elements and 24 edges because nine records cite `cli/0035:D-*` through
// the `Decisions` entry; that was measured wrong on both counts and is what
// made the retirement look blocked. Renaming a heading moves that heading's
// own `§` id and nothing else.
//
// PERMANENTLY RETAINED — the two mapped entries, and why neither record
// set should be migrated:
//
//   - `Dependency Source Verification` (17 records) is a dependency SURVEY
//     TABLE under Research Findings — `| Dependency | Source Searched? |
//     Key Findings |`. Critical Assumptions is a top-level section of
//     per-assumption Evidence Records (Status / Method / Evidence / If
//     wrong). Renaming re-parents the table across a section boundary and
//     reads a survey as assumptions the record never made.
//
//   - `Finalization Gate — Verdict` (2 records) carries its verdict inline,
//     at the same level and position as the canonical gate. The current
//     template puts gate responses in `{ARTIFACT_DIR}/gate.md` and expresses
//     the verdict as the Status flip, so there is no verdict subsection to
//     rename toward.
var SectionAliases = []sectionAlias{
	// Legacy predecessors of the Evidence Record apparatus.
	// Both permanently retained; see PERMANENTLY RETAINED above.
	{"Dependency Source Verification", "Critical Assumptions",
		"pre-Evidence-Record heading for Source Search evidence"},

	// Local spellings that differ by more than case.
	{"Finalization Gate — Verdict", "Finalization Gate",
		"gate heading carrying its verdict inline"},

	// Recognised, with no canonical home in the template.
	//
	// The premortem is the load-bearing case. It is NOT an old spelling of
	// Decision Rationale: a premortem is prospective hindsight — assume the
	// approach shipped and failed, enumerate why (Klein 2007, cited in
	// RESEARCH.md §Critique / premortem lens) — where rationale argues the
	// choice was sound. They point in opposite directions and the template
	// homes them separately: Stage 2 keeps only the greppable `Premortem:`
	// verdict line closing Decision Rationale, sends the ledger to
	// `evidence/propose-premortem/critic.md`, and leaves the failure content
	// to Risks and Mitigations / Failure Modes. So the heading has no
	// canonical section to rename toward, and mapping it onto Decision
	// Rationale would relocate a failure narrative into an argument for
	// soundness — for 8 of the 15 records, across a top-level boundary from
	// Trade-offs. Three records already write an unlisted premortem heading
	// (`Premortem outcome`, `Premortem (chosen approach A)`); they classify
	// as MatchAuthorSubsection, report nothing, and stay addressable. That
	// is the healthy outcome these entries now share.
	{"Premortem", "",
		"author-added; the template carries the premortem as the `Premortem:` verdict line closing Decision Rationale, its ledger in evidence/, and its failure content in Risks and Mitigations / Failure Modes"},
	{"Premortem (chosen approach)", "",
		"author-added; see Premortem"},
	{"Escaped-Defect Ledger", "",
		"author-added ledger; never a template section"},
	{"Open Questions", "",
		"author-added; the template carries open obligations on the Status qualifier instead"},
	{"Joint-Decision Check (Stage 2)", "",
		"author-added; the current template carries the joint-check verdict as a line in Decision Rationale"},
	{"Decision", "",
		"author-added summary heading; the template has no standalone Decision section"},
	{"Rationale", "",
		"author-added; the template's rationale lives in Decision Rationale"},
	{"Consumers", "",
		"author-added; downstream consumers are named in Normative Contracts"},
	{"Scope Fences", "",
		"author-added scope-boundary section; never a template section"},
	{"Out of scope", "",
		"author-added scope-boundary section; the same thing under its commoner spelling. It mapped to `Scope Fences` until this was audited, but that is not a template section either — it is the entry two lines up, which maps to \"\". A mapping onto a non-existent canonical is a lookup that can never project."},
	{"What this RDR locks", "",
		"author-added summary of the locked contracts"},
	{"Pre-release scope", "",
		"author-added release-scoping section"},
	{"Downstream effects", "",
		"author-added; the template carries these under Consequences"},
	{"Normative Fixtures", "",
		"author-added; approved fixtures are recorded in Normative Contracts"},
	{"Scored matrix (Questions-Options-Criteria)", "",
		"author-added decision matrix under Alternatives Considered"},
	{"Prior Art Survey", "",
		"author-added; prior art is normally cited in Research Findings"},
}

// fieldAlias is one legacy metadata field label and where it maps.
type fieldAlias struct {
	name      string
	canonical string
	note      string
}

// FieldAliases maps legacy Metadata block field labels onto canonical
// ones, on the same three-group principle as SectionAliases.
//
// `Related` is a genuine legacy spelling of `Related Issues`. The rest are
// one-off fields individual records invented — a coordination note, a
// pointer at a normative home — which map to "" and classify as
// MatchRecognizedUnmapped: recognised as an author's field, not a
// canonical one, and distinct from a label nobody has ever written.
var FieldAliases = []fieldAlias{
	{"Related", "Related Issues", "legacy spelling of Related Issues"},

	{"Normative home", "", "author-added pointer at the RDR owning a shared contract"},
	{"Dependents", "", "author-added inverse of Predecessors"},
	{"Peers", "", "author-added sibling list; the template's field is Cluster"},
	{"Joint decisions", "", "author-added; joint decisions ride the Status qualifier"},
	{"Joint decision (owed at Propose)", "", "author-added; joint decisions ride the Status qualifier"},
	{"Coordination peers", "", "author-added sibling list; the template's field is Cluster"},
	{"Conforms to (not overridden)", "", "author-added counterpart to Overrides"},
	{"Sibling (not a predecessor)", "", "author-added sibling note"},
	{"Split-out (sibling work item, NOT this RDR)", "", "author-added scope note"},
	{"Release scope", "", "author-added release-scoping field"},
	{"Visible", "", "author-added visibility note"},
}

// SectionAliasCanonical returns the canonical section a legacy heading
// maps to, or "" when the heading is not a mapped alias or maps to no
// canonical section.
//
// It is the alias table read for a reader that wants
// the section a heading names rather than the section the record's own
// heading names. LookupSection is the full classifier and stays
// the authority on how a heading is REPORTED; this answers the narrower
// question of what it is called.
func SectionAliasCanonical(heading string) string {
	heading = strings.TrimSpace(heading)
	for _, a := range SectionAliases {
		if strings.EqualFold(a.name, heading) {
			return a.canonical
		}
	}
	return ""
}

// FieldAliasCanonical returns the canonical field a legacy label maps to,
// or "" when the label is not a mapped alias.
func FieldAliasCanonical(label string) string {
	for _, a := range FieldAliases {
		if strings.EqualFold(a.name, strings.TrimSpace(label)) {
			return a.canonical
		}
	}
	return ""
}

// LookupSection classifies an observed heading against the template.
//
// The order of attempts is exact, then level-variant, then case-variant,
// then the alias table, then unknown. Level variance is checked before
// case variance because it is by far the commoner divergence and the more
// benign one — a re-levelled section is the same section written by an
// older template.
func LookupSection(te TemplateTable, name string, level int) Match {
	return lookupSection(te, name, level, ClassRDR)
}

// LookupSectionIn is LookupSection for a document class: the class's own
// template table, and the RDR alias table consulted only for the RDR.
//
// The alias table holds the RDR's OWN history — `Problem statement` is a
// recognised predecessor of the RDR's `Problem Statement`. It is also a
// registry's current and correct spelling, so consulting the table for a
// registry turned a conforming heading into a legacy-name finding with a
// rename fix attached: the projector telling a document to migrate
// toward a template that does not govern it.
func LookupSectionIn(c DocClass, name string, level int) Match {
	return lookupSection(ClassTemplateTable(c), name, level, c)
}

func lookupSection(te TemplateTable, name string, level int, c DocClass) Match {
	name = strings.TrimSpace(name)

	for i := range te.Sections {
		s := te.Sections[i]
		if s.Name != name {
			continue
		}
		if s.Level == level {
			return Match{Kind: MatchExact, Canonical: &te.Sections[i], ObservedLevel: level}
		}
		return Match{
			Kind:          MatchLevelVariant,
			Canonical:     &te.Sections[i],
			ObservedLevel: level,
			Note:          "written at a level other than the canonical one; the template re-levelled this section",
		}
	}

	for i := range te.Sections {
		if strings.EqualFold(te.Sections[i].Name, name) {
			return Match{
				Kind:          MatchCaseVariant,
				Canonical:     &te.Sections[i],
				ObservedLevel: level,
				Note:          "differs from the canonical name only by case",
			}
		}
	}

	if canonical, note, ok := matchScaffoldIn(c, name); ok {
		for i := range te.Sections {
			if te.Sections[i].Name == canonical {
				return Match{
					Kind:          MatchScaffoldInstance,
					Canonical:     &te.Sections[i],
					ObservedLevel: level,
					Note:          note,
				}
			}
		}
	}

	for _, a := range SectionAliases {
		if !ClassUsesRDRAliases(c) || !strings.EqualFold(a.name, name) {
			continue
		}
		if a.canonical == "" {
			return Match{Kind: MatchRecognizedUnmapped, ObservedLevel: level, Note: a.note}
		}
		if s, ok := te.SectionByName(a.canonical); ok {
			for i := range te.Sections {
				if te.Sections[i].Name == s.Name {
					return Match{
						Kind:          MatchLegacyAlias,
						Canonical:     &te.Sections[i],
						ObservedLevel: level,
						Note:          a.note,
					}
				}
			}
		}
		// The alias names a canonical section the template does not have,
		// which is itself a recognised-but-unmapped situation.
		return Match{Kind: MatchRecognizedUnmapped, ObservedLevel: level, Note: a.note}
	}

	return Match{Kind: MatchUnknown, ObservedLevel: level}
}

// LookupField classifies an observed Metadata block field label against a
// template, on the same principle as LookupSection. There is no
// level to vary, so the kinds in play are exact, case-variant,
// legacy-alias, recognized-unmapped and unknown.
func LookupField(te TemplateTable, label string) Match {
	label = strings.TrimSpace(label)

	for _, f := range te.MetadataFields {
		if f == label {
			return Match{Kind: MatchExact, Note: f}
		}
	}
	for _, f := range te.MetadataFields {
		if strings.EqualFold(f, label) {
			return Match{Kind: MatchCaseVariant, Note: f}
		}
	}
	for _, a := range FieldAliases {
		if !strings.EqualFold(a.name, label) {
			continue
		}
		if a.canonical == "" {
			return Match{Kind: MatchRecognizedUnmapped, Note: a.note}
		}
		return Match{Kind: MatchLegacyAlias, Note: a.note}
	}
	return Match{Kind: MatchUnknown}
}
