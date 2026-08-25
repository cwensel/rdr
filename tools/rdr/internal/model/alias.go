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
	// does not. It is not a defect: sections have been re-levelled
	// between template epochs, most visibly Critical Assumptions, and a
	// frozen record keeps the level of the epoch that produced it.
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
	// addition is a corpus-level question (`rdr index --coverage`
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
// Every entry is a heading observed in the frozen corpus. They fall into
// three groups.
//
// LEGACY PREDECESSORS of the Evidence Record apparatus. Before Critical
// Assumptions carried Evidence Records, records verified dependency claims
// under their own headings; those headings are the same content under an
// older name.
//
// FOLDED SECTIONS. The premortem used to have its own section; the
// current template folds its verdict into Decision Rationale as one
// greppable line. The heading is legacy, its content is not.
//
// LOCAL SPELLINGS. Case variants and small rewordings of canonical names.
// These are matched by the case-insensitive lookup rather than needing an
// entry, and are listed here only where the wording differs too.
//
// A handful of headings are recognised with no canonical home: sections
// authors invented that the template never adopted. They map to "" and
// classify as MatchRecognizedUnmapped, which keeps them out of
// `unknown-to-template` without pretending they project onto anything.
var SectionAliases = []sectionAlias{
	// Legacy predecessors of the Evidence Record apparatus.
	{"API Verification", "Critical Assumptions",
		"pre-Evidence-Record heading for verified dependency API claims"},
	{"Dependency Source Verification", "Critical Assumptions",
		"pre-Evidence-Record heading for Source Search evidence"},

	// The premortem, before its verdict line moved into Decision Rationale.
	{"Premortem", "Decision Rationale",
		"legacy section; the current template carries the premortem verdict as a line in Decision Rationale"},
	{"Premortem (chosen approach)", "Decision Rationale",
		"legacy section; the current template carries the premortem verdict as a line in Decision Rationale"},

	// Local spellings that differ by more than case.
	{"Context / Background", "Background",
		"combined heading for the Context/Background pair"},
	{"Day 2 Operations / New Dependencies", "Day 2 Operations",
		"combined heading for two Implementation Plan sub-sections"},
	{"Finalization Gate — Verdict", "Finalization Gate",
		"gate heading carrying its verdict inline"},
	{"Out of scope", "Scope Fences",
		"local spelling of the scope-boundary section"},
	{"Decisions", "Load-Bearing Decisions",
		"shortened spelling of Load-Bearing Decisions; the bullets under it are the same D-elements"},

	// Recognised, with no canonical home in any epoch.
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
// It is the alias table read WITHOUT an epoch, for a reader that wants
// the section a heading names rather than the section the record's own
// template offered. LookupSection is the epoch-aware classifier and stays
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

// LookupSection classifies an observed heading against a template epoch.
//
// The order of attempts is exact, then level-variant, then case-variant,
// then the alias table, then unknown. Level variance is checked before
// case variance because it is by far the commoner divergence and the more
// benign one — a re-levelled section is the same section written by an
// older template.
func LookupSection(te TemplateEpoch, name string, level int) Match {
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
			Note:          "written at a level other than the canonical one; template epochs re-levelled this section",
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

	if canonical, note, ok := matchScaffold(name); ok {
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
		if !strings.EqualFold(a.name, name) {
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
		// The alias names a canonical section this epoch does not have,
		// which is itself a recognised-but-unmapped situation.
		return Match{Kind: MatchRecognizedUnmapped, ObservedLevel: level, Note: a.note}
	}

	return Match{Kind: MatchUnknown, ObservedLevel: level}
}

// LookupField classifies an observed Metadata block field label against a
// template epoch, on the same principle as LookupSection. There is no
// level to vary, so the kinds in play are exact, case-variant,
// legacy-alias, recognized-unmapped and unknown.
func LookupField(te TemplateEpoch, label string) Match {
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
