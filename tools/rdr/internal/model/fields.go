package model

import (
	"regexp"
	"strings"
)

// This file is the labelled-bullet half of the template model: which
// `- **Label**:` bullets each section carries, how an observed label is
// matched against them, and the Evidence Record's own Status vocabulary.
//
// It exists for the RESILIENCE CONTRACT (README §The resilience
// contract): the scanner classifies every labelled bullet in a record,
// and a bullet it cannot place is recorded rather than dropped. That
// needs a vocabulary to classify against, per section, and a matching
// rule tolerant of how authors actually write labels — `Evidence — the
// two channel reductions`, `If wrong (a refusal is owed)`, `Evidence
// (plan)` — which is by PREFIX, not by literal.

// FieldSet is the labelled-bullet vocabulary of one canonical section.
type FieldSet struct {
	// Section is the canonical section name (Section.Name).
	Section string
	// Canonical are the `- **Label**:` bullets TEMPLATE.md writes inside
	// the section, at any indent, in template order. The anti-drift test
	// reads them back out of TEMPLATE.md.
	Canonical []string
	// Observed are labels the section's own prose names and authors
	// bulletise — TEMPLATE.md's Cross-Cutting Concerns lists its
	// candidate concerns as prose, and records write each as a bold
	// label. They are template-drawn in substance, so a reader accepts
	// them without a warning; the same two-tier doctrine as Vocabulary.
	Observed []string
	// Scaffold marks a set whose one canonical label is a placeholder
	// (`[Alternative N]`) every instance replaces with its own name, so
	// any label in the section is a filled-in scaffold.
	Scaffold bool
}

// SectionFields is the per-section labelled-bullet vocabulary. A section
// absent from this table carries no template-drawn labels: every labelled
// bullet written under it is the author's own (MatchAuthor).
// SectionFields is the per-section labelled-bullet vocabulary. A section
// absent from this table carries no template-drawn labels: every labelled
// bullet written under it is the author's own (MatchAuthor).
//
// Canonical comes from TEMPLATE.md — the `- **Label**:` bullets it writes
// under each section, read back at load. Observed is the reader's own
// two-tier allowance for labels a section's PROSE names and authors
// bulletise; it stays in Go because the template states it as prose, not
// as bullets, and inferring a set from prose is guessing.
func SectionFields() []FieldSet {
	canonical := templateLabels(current().TemplateSource)
	out := make([]FieldSet, 0, len(sectionObserved))
	for _, fs := range sectionObserved {
		fs.Canonical = canonical[fs.Section]
		out = append(out, fs)
	}
	return out
}

// sectionObserved carries the Observed and Scaffold columns, which the
// template does not write as bullets.
var sectionObserved = []FieldSet{
	{Section: "Metadata"},
	{Section: "Critical Assumptions"},
	{Section: "Normative Contracts", Observed: []string{"Preview / dry-run"}},
	{Section: "Briefly Rejected", Scaffold: true},
	// TEMPLATE.md writes `**Mitigation**:` as the Risk bullet's second
	// line, not as its own bullet; records promote it to one.
	{Section: "Risks and Mitigations", Observed: []string{"Mitigation"}},
	// "Positive and negative consequences of the chosen approach."
	{Section: "Consequences", Observed: []string{"Positive", "Negative"}},
	// "What breaks visibly? What fails silently? Recovery path? How does
	// a developer diagnose the problem?"
	{Section: "Failure Modes", Observed: []string{
		"Visible", "Silent", "Recovery", "Diagnosis",
		"What breaks visibly", "What fails silently", "Recovery path",
	}},
	// The candidate-concern list and the determinism checklist, verbatim
	// from the section's prose.
	{Section: "Cross-Cutting Concerns", Observed: []string{
		"Versioning", "Build tool compatibility", "Licensing", "Deployment model",
		"IDE compatibility", "Incremental adoption", "Secret/credential lifecycle",
		"Memory management", "Concurrency model", "Character encoding",
		"Canonical-form / determinism", "Determinism", "Concurrency",
		"Hash function + library", "Pre-image byte layout", "Primitive encodings",
		"Map iteration order", "Whitespace policy", "Case folding",
		"Empty/null/absent distinguishability", "Version marker",
	}},
}

// FieldSetOf returns the vocabulary of a canonical section, or an empty
// set when the section carries no template-drawn labels.
func FieldSetOf(section string) FieldSet {
	for _, fs := range SectionFields() {
		if fs.Section == section {
			return fs
		}
	}
	return FieldSet{Section: section}
}

// LabelMatch is the result of classifying one observed label.
type LabelMatch struct {
	// Kind says how it matched. MatchAuthor is not a finding: the label
	// is the author's own, recorded so nothing is dropped.
	Kind MatchKind
	// Canonical is the vocabulary label matched, in its canonical
	// spelling, or "" for MatchAuthor.
	Canonical string
	// Tier says which tier of the set the label matched.
	Tier Tier
}

// LookupLabel classifies an observed bullet label against a field set.
//
// The order is exact, case-variant, then PREFIX, first over the canonical
// tier and then over the observed one. Prefix matching is the resilience
// rule the corpus demands: authors extend a label into a clause —
// `Evidence — the two source-checkable channel reductions`, `If wrong (a
// gate is owed after all)`, `Evidence (plan)` — and the label is the
// field followed by a boundary, not the whole bold text. The longest
// vocabulary label that is a word-prefix of the observed one wins, so
// `Status / sentinel errors` is never read as `Status` where both are in
// the set.
func LookupLabel(fs FieldSet, label string) LabelMatch {
	label = normaliseLabel(label)
	if label == "" {
		return LabelMatch{Kind: MatchAuthor}
	}
	if fs.Scaffold {
		if len(fs.Canonical) > 0 {
			return LabelMatch{Kind: MatchScaffoldInstance, Canonical: fs.Canonical[0], Tier: Canonical}
		}
		return LabelMatch{Kind: MatchScaffoldInstance, Tier: Canonical}
	}
	tiers := []struct {
		tier Tier
		set  []string
	}{{Canonical, fs.Canonical}, {ObservedAccepted, fs.Observed}}
	for _, t := range tiers {
		for _, c := range t.set {
			if c == label {
				return LabelMatch{Kind: MatchExact, Canonical: c, Tier: t.tier}
			}
		}
		for _, c := range t.set {
			if strings.EqualFold(c, label) {
				return LabelMatch{Kind: MatchCaseVariant, Canonical: c, Tier: t.tier}
			}
		}
	}
	best, bestTier := "", OffVocabulary
	for _, t := range tiers {
		for _, c := range t.set {
			if HasWordPrefix(label, c) && len(c) > len(best) {
				best, bestTier = c, t.tier
			}
		}
	}
	if best != "" {
		return LabelMatch{Kind: MatchPrefix, Canonical: best, Tier: bestTier}
	}
	return LabelMatch{Kind: MatchAuthor}
}

// labelBoundary is what may follow a vocabulary label inside a longer
// observed label: a space, a dash or em dash, a slash, a colon, a comma,
// a full stop, a semicolon, or a parenthesis or opening bracket.
var labelBoundary = regexp.MustCompile(`^[\s—–\-/:,.;)(\[]`)

// HasWordPrefix reports whether label opens with name, case-insensitively,
// followed by the end of the label or a clause boundary. `Evidence plan`
// and `Evidence — ...` open with `Evidence`; `Evidenced` does not.
func HasWordPrefix(label, name string) bool {
	if len(label) < len(name) || !strings.EqualFold(label[:len(name)], name) {
		return false
	}
	rest := label[len(name):]
	return rest == "" || labelBoundary.MatchString(rest)
}

var spaces = regexp.MustCompile(`\s+`)

// normaliseLabel trims a label and collapses internal whitespace — a
// wrapped bold label re-joins with a newline's worth of space — and drops
// a trailing colon an author put inside the bold.
func normaliseLabel(label string) string {
	label = strings.TrimSpace(spaces.ReplaceAllString(label, " "))
	return strings.TrimSpace(strings.TrimSuffix(label, ":"))
}

// --- the Evidence Record's Status ---------------------------------------

// AssumptionStatusVocabulary is the Evidence Record's Status line,
// `Verified | Pending | Unverified`, plus the dispositions the frozen
// corpus writes for an assumption the flow settled some other way:
// Refuted, Resolved (settled by a decision rather than evidence),
// Accepted and Downgraded (a Stage 6 reconcile verdict carried onto the
// record). Read, never judge: the records carrying them are terminal.
func AssumptionStatusVocabulary() Vocabulary {
	const field = "Status (Evidence Record)"
	v, err := assumptionStatuses(current().TemplateSource)
	if err != nil {
		return Vocabulary{Field: field}
	}
	return Vocabulary{
		Field:            field,
		Canonical:        v,
		ObservedAccepted: current().Sidecar.Observed[field],
	}
}

// AssumptionStatus is a parsed Evidence Record Status value, normalised
// to {value, qualifier, raw}.
type AssumptionStatus struct {
	// Raw is the input, unmodified.
	Raw string
	// Value is the vocabulary label, in canonical spelling, or "" when
	// the value is the template placeholder or opens with no label.
	Value string
	// Qualifier is whatever the author wrote after the label — `as
	// narrowed`, `at Stage 6 reconcile`, `-by-derivation`, a sentence —
	// with its leading separator removed. Free text.
	Qualifier string
	// Tier is Value's standing in AssumptionStatusVocabulary().
	Tier Tier
	// Placeholder is true when the value is the template's own legend
	// (`Verified | Pending | Unverified`) left unfilled. Reading it as
	// `Verified` by prefix would be confident and wrong, which is the one
	// outcome the projector must never produce.
	Placeholder bool
}

// ParseAssumptionStatus parses a line-joined Evidence Record Status value.
// The label is matched by word-prefix, case-insensitively (`REFUTED`),
// longest label first; everything after it is the qualifier.
func ParseAssumptionStatus(raw string) AssumptionStatus {
	s := AssumptionStatus{Raw: raw}
	// Emphasis is presentation: `**Verified**`, `_Pending_` and
	// `Verified.** The rest` all open with the label.
	body := strings.TrimSpace(strings.NewReplacer("**", "", "*", "", "_", "", "`", "").Replace(raw))
	if body == "" {
		return s
	}
	if isStatusPlaceholder(body) {
		s.Placeholder = true
		return s
	}
	best := ""
	for _, set := range [][]string{AssumptionStatusVocabulary().Canonical, AssumptionStatusVocabulary().ObservedAccepted} {
		for _, c := range set {
			if HasWordPrefix(body, c) && len(c) > len(best) {
				best = c
			}
		}
	}
	if best == "" {
		// No vocabulary label opens the value: the first clause is the
		// value, off-vocabulary, and the reader reports it as such.
		head := body
		if loc := offVocabularyBreak.FindStringIndex(body); loc != nil {
			head = body[:loc[0]]
		}
		s.Value = trailingPunctuation.ReplaceAllString(strings.TrimSpace(head), "")
		s.Qualifier = trimQualifier(body[len(head):])
		s.Tier = OffVocabulary
		return s
	}
	s.Value = best
	s.Qualifier = trimQualifier(body[len(best):])
	s.Tier = AssumptionStatusVocabulary().Classify(best)
	return s
}

// offVocabularyBreak ends an off-vocabulary value: a clause boundary or
// an opening parenthesis.
var offVocabularyBreak = regexp.MustCompile(`\s+—\s|\s+--\s|,\s|;\s|\.\s|:\s|\s*\(`)

// isStatusPlaceholder recognises the template legend left in place: two
// or more vocabulary labels joined by `|`.
func isStatusPlaceholder(body string) bool {
	parts := strings.Split(body, "|")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if AssumptionStatusVocabulary().Classify(strings.TrimSpace(p)) == OffVocabulary {
			return false
		}
	}
	return true
}

// trimQualifier strips the separator an author put between the label and
// the qualifier, and one pair of enclosing parentheses or brackets.
func trimQualifier(q string) string {
	q = strings.TrimSpace(q)
	q = strings.TrimLeft(q, "—–-:;,.")
	q = strings.TrimSpace(strings.TrimRight(q, "*_ "))
	q = strings.TrimSpace(q)
	if n := len(q); n >= 2 {
		if (q[0] == '(' && q[n-1] == ')') || (q[0] == '[' && q[n-1] == ']') {
			q = strings.TrimSpace(q[1 : n-1])
		}
	}
	return q
}
