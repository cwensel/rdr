// Package model is the machine-readable model of TEMPLATE.md.
//
// TEMPLATE.md is the only schema an RDR has, and it is prose: section
// names, levels and classes live in bracket text, the Method vocabulary
// lives in README prose, and the qualifier grammars live in HTML comments.
// Every consumer that needs them restated them by hand, and they drifted.
// This package states them once, as ordered Go tables, with a test that
// fails if TEMPLATE.md and the tables disagree.
//
// Two rules govern everything here.
//
// SAME-COMMIT RULE. A TEMPLATE.md change that adds, removes, renames or
// re-levels a section, or changes a class, field, label or qualifier
// grammar, ships in the same commit as its epoch entry and a synthetic
// fixture. TemplateTest in template_test.go enforces the section half
// mechanically.
//
// READ, NEVER JUDGE. Terminal records (Implemented, Rejected, Abandoned,
// Superseded, Demoted) are never amended. They are frozen at the template
// epoch that produced them, forever. The model therefore reads what the
// corpus contains, which is a strictly larger language than what a new
// record may write — see Vocabulary and its three tiers.
package model

import (
	"regexp"
	"strings"
)

// Class is a section's omission rule, per TEMPLATE.md's legend at the top
// of the file: "Section classes: **Required** (never omit). **Conditional**
// (delete the whole section if N/A — do NOT leave it blank or N/A-bulleted)."
type Class int

const (
	// Required means the section may never be omitted.
	Required Class = iota
	// Conditional means the whole section is deleted when it does not
	// apply. A cleanly deleted Conditional section is conformant; it is
	// never present-but-N/A-bulleted.
	Conditional
)

func (c Class) String() string {
	switch c {
	case Required:
		return "Required"
	case Conditional:
		return "Conditional"
	}
	return "Unknown"
}

// Grammar names the element grammar a section's body carries. It tells a
// scanner which extractor to run inside the section, and it is what makes
// the section table more than a list of names.
type Grammar int

const (
	// GrammarProse is free text with no extractable element structure.
	GrammarProse Grammar = iota
	// GrammarMetadataFields is the bold-label bullet list of the Metadata
	// block: `- **Label**: value`, values that may wrap (see ValueContinues).
	GrammarMetadataFields
	// GrammarEvidenceRecords is the assumption bullet tree: an
	// `- **A<N> [Statement]**` parent with the EvidenceFields sub-bullets.
	GrammarEvidenceRecords
	// GrammarNormativeBlocks is prose plus ```normative fences and the
	// Transient contract marker.
	GrammarNormativeBlocks
	// GrammarTable is a markdown pipe table with a fixed header row.
	GrammarTable
	// GrammarScaffold is a repeated per-instance sub-structure whose own
	// headings are instance-named (Alternative N, Step N) rather than
	// template-drawn.
	GrammarScaffold
	// GrammarGatePointer is the Finalization Gate body: at lock it is
	// replaced by a one-line pointer to gate.md. Epoch A and B inline the
	// responses instead; that difference is an epoch fingerprint.
	GrammarGatePointer
)

func (g Grammar) String() string {
	switch g {
	case GrammarProse:
		return "prose"
	case GrammarMetadataFields:
		return "metadata-fields"
	case GrammarEvidenceRecords:
		return "evidence-records"
	case GrammarNormativeBlocks:
		return "normative-blocks"
	case GrammarTable:
		return "table"
	case GrammarScaffold:
		return "scaffold"
	case GrammarGatePointer:
		return "gate-pointer"
	}
	return "unknown"
}

// Section is one entry in a template epoch's ordered section table.
type Section struct {
	// Name is the canonical heading text, verbatim, without the leading
	// hashes.
	Name string
	// Level is the canonical heading level: 2 for `##`, 3 for `###`,
	// 4 for `####`. Instances that write the section at another level are
	// level-variants, not unknown sections — see alias.go.
	Level int
	// Class is the omission rule.
	Class Class
	// Parent is the Name of the enclosing section, or "" for a top-level
	// (`##`) section.
	Parent string
	// Grammar is the element grammar the section's body carries.
	Grammar Grammar
}

// SectionClassRule documents how Class is derived, so a reader of the
// tables below does not have to reconstruct it.
//
// A section's class is read from its own bracket text in TEMPLATE.md:
// `[Required — never omit.` marks Required, `[Conditional — ...]` marks
// Conditional. Most sections carry neither marker. The rule for those:
//
//   - A section with no marker is Required. TEMPLATE.md's legend states
//     the default by exception — it defines what Conditional means and
//     tells the author to delete those sections — so an unmarked section
//     is part of the spine that is always present. The explicit
//     `[Required — never omit.` markers are emphasis on the six sections
//     the implementation prompt reads directly, not the whole of the
//     Required set.
//   - The exception is a scaffold sub-heading whose own bracket text is
//     carried by its parent. `### Alternative 1: [Name]` is Conditional
//     because Alternatives Considered says so on its behalf ("Conditional
//     scaffold — omit ... the `Alternative 1` block below if no
//     alternative warranted full analysis"). The same holds for the
//     `#### Step N` and `#### Activation Step N` placeholders, which are
//     per-instance scaffolding, and for `### Phase 2: Operational
//     Activation`, whose body says "Omit if not applicable."
//
// The anti-drift test reads the markers back out of TEMPLATE.md and
// asserts this table agrees, so the rule cannot rot silently.
const SectionClassRule = "unmarked template sections are Required; Conditional requires either an explicit [Conditional marker or a parent scaffold clause"

// EvidenceFields is the Critical Assumptions Evidence Record field set:
// the bold-label sub-bullets under an `- **A<N> [Statement]**` bullet, in
// template order. TEMPLATE.md states all four; a record missing one is
// incomplete, not foreign.
var EvidenceFields = []string{"Status", "Method", "Evidence", "If wrong"}

// AssumptionBullet matches the Evidence Record's parent bullet, capturing
// the assumption label. The template writes `**A1 [Statement]**`; the
// corpus also writes `**A1 — Statement.**`, `**A1** Statement` and
// `**A1 Statement**`, and splits labels (`A4b`), so only the label is
// matched and a scanner reads the statement by its own rule. A split
// label is written `A4b` or `A1.b`; the capture keeps the author's
// punctuation and the scanner normalises it.
var AssumptionBullet = regexp.MustCompile(`^\s*-\s+\*\*(A\d+(?:[.-]?[a-z])?)\b`)

// EvidenceFieldBullet matches one Evidence Record sub-bullet, capturing
// the label and the first line of its value.
var EvidenceFieldBullet = regexp.MustCompile(`^\s*-\s+\*\*([^*]+)\*\*:\s*(.*)$`)

// MetadataFieldBullet matches one Metadata block bullet, capturing the
// label and the first line of its value. Metadata bullets sit at column
// zero; the Evidence Record's are indented under their assumption.
var MetadataFieldBullet = regexp.MustCompile(`^- \*\*([^*]+)\*\*:\s*(.*)$`)

// MetadataFields is the Metadata block's canonical field set in template
// order. Presence is not uniform across epochs — Profile, Seam Lineage,
// Overrides and Cluster arrive later (see epoch.go), and Predecessors,
// Overrides, Seam Lineage and Cluster are omitted when they have no value.
var MetadataFields = []string{
	"Date",
	"Status",
	"Type",
	"Profile",
	"Priority",
	"Related Issues",
	"Predecessors",
	"Overrides",
	"Seam Lineage",
	"Cluster",
}

// ValueContinues reports whether next is a continuation of the metadata or
// Evidence Record field value whose first line has already been consumed.
//
// This exists because 80 of the 156 corpus records wrap at least one
// metadata value onto a following line, most often Seam Lineage,
// Predecessors and Overrides — long values that a formatter reflowed.
// Truncating at the newline loses the back half of those values, which is
// precisely the part that carries the accretion trail and the override
// list. The scanner does the line joining; the model owns the predicate
// that says where a value ends, so the rule has one home.
//
// A value continues onto next when next is indented and is not itself a
// new bullet. Three things end it:
//
//   - a blank line, or an unindented line: the block is over;
//   - a new `- **Label**:` bullet, at any indent: the next field;
//   - an HTML comment opener. TEMPLATE.md's own comments are copied into
//     instances verbatim and sit indented directly under the field they
//     annotate — 41 records carry one right under Status. Those comments
//     are template guidance, not part of anyone's value.
//
// A list continuation (`  - ...` nested under the field, as Seam Lineage's
// accretion-disposition clause is written) is a bullet, so it ends the
// scalar value. That is correct: such a field's value is its first
// paragraph, and the nested bullets are its own sub-structure.
func ValueContinues(next string) bool {
	trimmed := trimLeftSpace(next)
	if trimmed == "" {
		return false // blank line
	}
	if len(trimmed) == len(next) {
		return false // not indented
	}
	if hasPrefix(trimmed, "<!--") {
		return false // template guidance comment
	}
	if hasPrefix(trimmed, "- ") || hasPrefix(trimmed, "-\t") || trimmed == "-" {
		return false // a new bullet, including a nested one
	}
	return true
}

// --- Markers ---------------------------------------------------------

// TransientMarker matches the bridge-surface contract marker from
// Normative Contracts: `Transient — scheduled deletion by <sibling
// NNNN-slug>, <phase/anchor>; <one-clause disposition>`. A marked contract
// counts toward neither the Profile contract axis nor the split signal,
// so a consumer must be able to find it. The em dash is required; the
// sibling, anchor and disposition are captured.
var TransientMarker = regexp.MustCompile(`Transient\s+—\s+scheduled deletion by\s+([^,]+),\s*([^;]+);\s*(.*)`)

// NormativeFenceOpen matches the opening of a ```normative block. Every
// external API call inside such a block owes an Evidence Record above it.
var NormativeFenceOpen = regexp.MustCompile("^\\s*```normative\\s*$")

// FenceDelimiter matches any fenced-code delimiter, so a scanner can skip
// block content rather than reading headings and bullets out of it.
var FenceDelimiter = regexp.MustCompile("^\\s*(```|~~~)")

// GatePointer matches the one-line gate.md pointer that replaces the
// Finalization Gate body at lock. Its presence versus an inlined gate body
// separates epoch C onward from epochs A and B.
var GatePointer = regexp.MustCompile(`gate\.md`)

// --- Heading -------------------------------------------------------

// Heading matches an ATX heading, capturing its hashes and its text.
var Heading = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*$`)

// --- small helpers, kept local so the package stays stdlib-light ------

func trimLeftSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func hasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

// --- element keys ------------------------------------------------------

// DecisionClasses are the Load-Bearing Decisions classes TEMPLATE.md
// names, in template order. A decision bullet whose bold label starts
// with one of these is keyed by the class (`D-identity`); any other label
// is an author's own class and gets a derived key.
var DecisionClasses = []string{
	"Identity",
	"Wire / byte format",
	"Naming",
	"Selection / predicate",
}

// DecisionClassOf returns the template class a decision label opens with,
// or "". `Selection / predicate (remainder)` and `Selection` both resolve
// to `Selection / predicate`; the match is case-insensitive and takes
// the longest class that is a prefix of the label, so the shorter
// `Selection` cannot shadow the full name.
func DecisionClassOf(label string) string {
	l := strings.ToLower(strings.TrimSpace(label))
	best := ""
	for _, c := range DecisionClasses {
		lc := strings.ToLower(c)
		if hasPrefix(l, lc) && len(c) > len(best) {
			best = c
		}
		// A one-word lead of a multi-word class (`Selection`, `Wire`) is
		// the same class abbreviated.
		if head, _, ok := strings.Cut(lc, " "); ok && head == firstWord(l) && len(c) > len(best) {
			best = c
		}
	}
	return best
}

func firstWord(s string) string {
	head, _, _ := strings.Cut(s, " ")
	return strings.TrimRight(head, ":—-")
}

// GateItems maps each Finalization Gate sub-section onto its G-key, so an
// inlined gate response (epochs A and B) is addressable as `NNNN:G-scope`
// regardless of the heading's exact wording.
//
// Retained records the template's `[Retained at lock — …]` marker: at lock
// the gate's responses move to gate.md, but a retained item STAYS in the
// record. Which item that is, and whether there is one at all, is the
// template's to say — a rule the reader spelled out for itself would be a
// second source for one fact, and the two would drift. The same-commit
// test binds this column to the marker.
var GateItems = []struct {
	Section, Key string
	Retained     bool
}{
	{"Contradiction Check", "contradiction", false},
	{"Assumption Verification", "assumptions", false},
	{"Scope Verification", "scope", false},
	{"Cross-Cutting Concerns", "cross-cutting", true},
	{"Proportionality", "proportionality", false},
}

// GateItemRetained reports whether a G-key names an item the template
// keeps in the record at lock.
func GateItemRetained(key string) bool {
	for _, g := range GateItems {
		if g.Key == key {
			return g.Retained
		}
	}
	return false
}

// GateItemKey returns the G-key for a canonical gate sub-section, or "".
func GateItemKey(section string) string {
	for _, g := range GateItems {
		if g.Section == section {
			return g.Key
		}
	}
	return ""
}

// IsGateItemKey reports whether a key is one of the five gate keys.
//
// The gate namespace is closed by construction — every `G-` element the
// projector mints comes from a Finalization Gate sub-heading — so this is
// the whole of it. A citation grammar asks so that a `G-<slug>` an author
// coined for their own guard table is not read as a gate response the
// target can never have; see edge.slugKind.
func IsGateItemKey(key string) bool {
	for _, g := range GateItems {
		if g.Key == key {
			return true
		}
	}
	return false
}
