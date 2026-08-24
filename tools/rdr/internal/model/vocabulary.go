package model

import (
	"regexp"
	"strings"
)

// Tier says how a value stands relative to a Vocabulary. It has two
// members, and the split is doctrine, not convenience.
//
// A new record may only write a Canonical value: that is what TEMPLATE.md
// and README.md sanction. But terminal records (Implemented, Rejected,
// Abandoned, Superseded, Demoted) are never amended — they are frozen at
// the template epoch that produced them, forever. So the reader must
// accept values the writer may no longer produce. ObservedAccepted is that
// set: values found in the frozen corpus that are legitimate practice the
// template simply never listed. They classify with no warning, and a gate
// or lint consumer treats them as valid.
//
// Anything else is OffVocabulary. There is deliberately no third
// "tolerated but wrong" tier: a value that is neither sanctioned nor
// legitimate practice is a typo, and a typo is exactly what the gate
// exists to report. Naming it in the model would launder it.
type Tier int

const (
	// OffVocabulary means the value is in neither set.
	OffVocabulary Tier = iota
	// Canonical means a new record may write this value.
	Canonical
	// ObservedAccepted means the value is absent from the template but
	// present in the frozen corpus as legitimate practice. Read it, do not
	// warn on it, and treat it as valid at the gate.
	ObservedAccepted
)

func (t Tier) String() string {
	switch t {
	case Canonical:
		return "canonical"
	case ObservedAccepted:
		return "observed-accepted"
	case OffVocabulary:
		return "off-vocabulary"
	}
	return "unknown"
}

// Vocabulary is a closed value set in two tiers. Canonical is what a new
// record may write; ObservedAccepted is what the frozen corpus additionally
// contains and the reader must accept. See Tier.
type Vocabulary struct {
	// Field is the metadata or Evidence Record label this vocabulary
	// governs, for use in messages.
	Field string
	// Canonical is the template-sanctioned set, in template order.
	Canonical []string
	// ObservedAccepted is the corpus-only set, in no meaningful order.
	// Empty is the normal case; a non-empty entry is a recorded finding.
	ObservedAccepted []string
}

// Classify places one already-trimmed value in a tier. Matching is exact
// and case-sensitive: every corpus occurrence of every Status, Type,
// Profile and Method label matches its canonical spelling exactly, so
// loosening the match would only hide a real defect.
func (v Vocabulary) Classify(value string) Tier {
	for _, c := range v.Canonical {
		if c == value {
			return Canonical
		}
	}
	for _, o := range v.ObservedAccepted {
		if o == value {
			return ObservedAccepted
		}
	}
	return OffVocabulary
}

// --- The four closed vocabularies ------------------------------------

// StatusVocabulary is the RDR lifecycle status set.
//
// Canonical is TEMPLATE.md's Status line. ObservedAccepted carries
// Rejected and Deferred: both are real terminal dispositions written by
// records in the frozen corpus (Rejected on four, Deferred on one) that
// TEMPLATE.md never listed. Since those records can never be amended, a
// reader that called them off-vocabulary would be permanently wrong about
// five records.
var StatusVocabulary = Vocabulary{
	Field: "Status",
	Canonical: []string{
		"Draft", "Final", "Implemented", "Reverted",
		"Abandoned", "Superseded", "Demoted",
	},
	ObservedAccepted: []string{"Rejected", "Deferred"},
}

// TerminalStatuses are the statuses after which a record is never amended.
// They are why the ObservedAccepted tier exists at all.
var TerminalStatuses = []string{
	"Implemented", "Reverted", "Abandoned", "Superseded", "Demoted", "Rejected",
}

// TypeVocabulary is TEMPLATE.md's Type line. No corpus value falls outside
// it.
var TypeVocabulary = Vocabulary{
	Field: "Type",
	Canonical: []string{
		"Feature", "Bug Fix", "Technical Debt",
		"Framework Workaround", "Architecture",
	},
}

// ProfileVocabulary is TEMPLATE.md's Profile line: the Stage 5 routing
// latch, sized by blast radius. The field's value is the label plus one
// clause naming the contracts behind it, so a consumer takes the leading
// label and classifies that.
var ProfileVocabulary = Vocabulary{
	Field:     "Profile",
	Canonical: []string{"small", "mid", "large", "foundational"},
}

// MethodVocabulary is the Evidence Record Method set: the eight labels
// from README.md's "Verifying load-bearing claims", which that section
// declares authoritative precisely so the guidance does not ship inside
// the template body.
//
// ObservedAccepted is empty, and that is a finding rather than an
// omission: every Method value in the frozen corpus resolves to one of
// these eight once compounds are split and parenthetical glosses are
// stripped. The eight are the whole vocabulary.
var MethodVocabulary = Vocabulary{
	Field: "Method",
	Canonical: []string{
		"Source Search", "Spike", "Prior Art", "Derivation",
		"Design Decision", "Peer RDR", "MVV Test", "Docs Only",
	},
}

// --- Compound Method parsing -----------------------------------------

// methodSeparator splits a compound Method into its members. The corpus
// writes ` + ` between them.
const methodSeparator = " + "

// trailingPunctuation matches sentence punctuation left on a member
// (`Source Search.`), which several records write.
var trailingPunctuation = regexp.MustCompile(`[.,;:]+$`)

// MethodMember is one member of a possibly-compound Method value.
type MethodMember struct {
	// Label is the member with its gloss, punctuation, markdown emphasis
	// and surrounding space removed — the thing to classify.
	Label string
	// Gloss is the parenthetical text stripped from the member, without
	// its delimiters, or "" if there was none. It is free text.
	Gloss string
	// Raw is the member exactly as written.
	Raw string
	// Trailing is the author's commentary after the label's clause
	// boundary, or "" if there was none. Like Gloss, it is free text.
	Trailing string
	// Tier is Label's standing in MethodVocabulary.
	Tier Tier
}

// MethodValue is a parsed Method field value, compound or not.
type MethodValue struct {
	// Raw is the input, unmodified.
	Raw string
	// Members are the member labels in written order. A non-compound
	// value yields exactly one.
	Members []MethodMember
	// Compound reports whether the value named more than one method.
	Compound bool
	// Valid reports whether every member is in vocabulary. A compound
	// built entirely from sanctioned labels is valid — combining methods
	// is normal practice, written by a sixth of the corpus — and a
	// compound with one bad member is invalid on that member alone.
	Valid bool
	// OffVocabulary lists the members that failed, in written order, so a
	// gate consumer can name which one is wrong rather than rejecting the
	// whole value. Empty when Valid.
	OffVocabulary []MethodMember
}

// ParseMethod parses a Method field value.
//
// Order matters, and it is the opposite of the obvious one. Parenthetical
// glosses are excised BEFORE the value is split on ` + `, because a gloss
// is free text that frequently contains the separator itself — `Source
// Search (carrier + decode package boundary)` is ONE method with a gloss,
// not two methods, and splitting first would manufacture two off-vocabulary
// members out of a conformant record. Excising first, on balanced
// delimiters, leaves only the real separators behind.
//
// An empty or placeholder-only value yields a MethodValue with no members
// and Valid false.
func ParseMethod(raw string) MethodValue {
	v := MethodValue{Raw: raw}

	body := strings.TrimSpace(raw)
	if body == "" {
		return v
	}

	// Excise balanced glosses, remembering each one so the member that
	// carried it can report it.
	stripped, glosses := exciseGlosses(body)
	// Markdown emphasis written ACROSS the separator (`Docs Only **+
	// Spike**`) would otherwise glue the asterisks onto a member.
	stripped = emphasisAroundSeparator.ReplaceAllString(stripped, methodSeparator)

	parts := strings.Split(stripped, methodSeparator)
	v.Compound = len(parts) > 1

	allValid := true
	gi := 0
	for _, part := range parts {
		m := parseMethodMember(part)
		if m.Label == "" {
			continue
		}
		// Reattach the glosses that fell inside this member.
		n := strings.Count(part, glossPlaceholder)
		if n > 0 && gi < len(glosses) {
			m.Gloss = strings.Join(glosses[gi:min(gi+n, len(glosses))], "; ")
			gi += n
		}
		v.Members = append(v.Members, m)
		if m.Tier == OffVocabulary {
			allValid = false
			v.OffVocabulary = append(v.OffVocabulary, m)
		}
	}

	v.Valid = allValid && len(v.Members) > 0
	return v
}

// emphasisAroundSeparator matches a ` + ` separator wrapped in markdown
// emphasis, which records write when they bold the whole compound.
var emphasisAroundSeparator = regexp.MustCompile(`\s*\*{1,2}\s*\+\s*\*{0,2}\s*|\s*\*{0,2}\s*\+\s*\*{1,2}\s*`)

// glossPlaceholder stands in for an excised gloss while the value is
// split, so a member can tell how many glosses belonged to it. It uses a
// rune no RDR writes.
const glossPlaceholder = "\x00"

// exciseGlosses removes every balanced (...) and *(...)* run from s,
// replacing each with glossPlaceholder, and returns the removed texts in
// order. Nesting is counted, so an inner pair does not close the outer.
// An UNBALANCED opener — a gloss that a line-truncation cut in half — is
// treated as running to the end of the value, which keeps the label in
// front of it readable.
func exciseGlosses(s string) (string, []string) {
	var out strings.Builder
	var glosses []string

	for i := 0; i < len(s); i++ {
		if s[i] != '(' {
			out.WriteByte(s[i])
			continue
		}
		depth, j := 1, i+1
		for ; j < len(s) && depth > 0; j++ {
			switch s[j] {
			case '(':
				depth++
			case ')':
				depth--
			}
		}
		inner := s[i+1 : max(i+1, j-1)]
		if depth > 0 {
			inner = s[i+1:] // unterminated: the rest is gloss
		}
		glosses = append(glosses, strings.TrimSpace(inner))
		out.WriteString(glossPlaceholder)
		i = j - 1
	}
	return out.String(), glosses
}

// clauseBreak marks where a member's LABEL ends and the author's
// commentary begins. Records routinely continue a Method value into prose
// — `Source Search, plus the Stage-4 I/O round`, `MVV Test — runs during
// Phase 3` — and that prose is a note about the method, not part of its
// name. Cutting at the first clause boundary keeps the label recoverable;
// without it every such record reads as off-vocabulary on a label it
// actually spelled correctly.
var clauseBreak = regexp.MustCompile(`\s+—\s|\s+--\s|,\s|;\s|\.\s|:\s`)

func parseMethodMember(part string) MethodMember {
	m := MethodMember{Raw: strings.ReplaceAll(part, glossPlaceholder, "")}

	label := strings.ReplaceAll(part, glossPlaceholder, " ")
	label = strings.TrimSpace(label)
	if loc := clauseBreak.FindStringIndex(label); loc != nil {
		m.Trailing = strings.TrimSpace(label[loc[1]:])
		label = label[:loc[0]]
	}
	label = trailingPunctuation.ReplaceAllString(strings.TrimSpace(label), "")
	// Markdown emphasis and code ticks around or beside the label, and
	// emphasis that CLOSES mid-member (`Peer RDR** for the member set`).
	label = strings.Trim(strings.TrimSpace(label), "`*_ ")
	label = strings.ReplaceAll(label, "**", " ")
	label = strings.Join(strings.Fields(label), " ")

	if label != "" {
		// A member may be followed by a qualifying phrase with no
		// punctuation to cut on — `Peer RDR 0042-frame-grammar
		// §Normative Contracts`, `Source Search for the field`. Take the longest
		// canonical label that prefixes it, on a word boundary, and treat
		// the remainder as trailing commentary. A member that does not
		// start with a canonical label is left whole, so a genuinely
		// wrong label still reports as itself.
		if MethodVocabulary.Classify(label) == OffVocabulary {
			if pre, rest := longestMethodPrefix(label); pre != "" {
				if m.Trailing == "" {
					m.Trailing = rest
				} else {
					m.Trailing = rest + " " + m.Trailing
				}
				label = pre
			}
		}
		m.Tier = MethodVocabulary.Classify(label)
	}
	m.Label = label
	return m
}

// longestMethodPrefix returns the longest canonical Method label that
// prefixes s at a word boundary, and the remainder after it.
func longestMethodPrefix(s string) (prefix, rest string) {
	for _, c := range MethodVocabulary.Canonical {
		if len(s) <= len(c) || !strings.EqualFold(s[:len(c)], c) {
			continue
		}
		if next := s[len(c)]; next != ' ' && next != '\t' {
			continue
		}
		if len(c) > len(prefix) {
			prefix, rest = c, strings.TrimSpace(s[len(c):])
		}
	}
	return prefix, rest
}

// TypeValue is a parsed Type field value.
type TypeValue struct {
	// Raw is the input, unmodified.
	Raw string
	// Members are the type labels in written order. Records write
	// combined types — `Feature / Architecture` — where two apply.
	Members []string
	// Valid reports whether every member is in vocabulary.
	Valid bool
	// OffVocabulary lists the members that failed, in written order.
	OffVocabulary []string
}

// ParseType parses a Type field value. Like Method it may be combined,
// though records join Types with ` / ` rather than ` + `, and it may
// carry a trailing parenthetical or clause naming what the type covers.
func ParseType(raw string) TypeValue {
	v := TypeValue{Raw: raw}

	body, _ := exciseGlosses(strings.TrimSpace(raw))
	body = strings.ReplaceAll(body, glossPlaceholder, "")
	if loc := clauseBreak.FindStringIndex(body); loc != nil {
		body = body[:loc[0]]
	}

	allValid := true
	for _, part := range strings.Split(body, "/") {
		label := strings.Trim(strings.TrimSpace(part), "`*_.,;: ")
		if label == "" {
			continue
		}
		v.Members = append(v.Members, label)
		if TypeVocabulary.Classify(label) == OffVocabulary {
			allValid = false
			v.OffVocabulary = append(v.OffVocabulary, label)
		}
	}
	v.Valid = allValid && len(v.Members) > 0
	return v
}

// --- Status value parsing --------------------------------------------

// StatusValue is a parsed Status field value: a vocabulary label plus an
// optional qualifier.
type StatusValue struct {
	// Raw is the input, unmodified.
	Raw string
	// Label is the bare status word, with any qualifier removed.
	Label string
	// Qualifier is the qualifier text without its brackets or
	// parentheses, or "" if there was none.
	Qualifier string
	// QualifierForm names which grammar the qualifier matched.
	QualifierForm QualifierForm
	// Tier is Label's standing in StatusVocabulary.
	Tier Tier
}

// ParseStatus parses a Status field value, which the caller has already
// line-joined per ValueContinues.
func ParseStatus(raw string) StatusValue {
	s := StatusValue{Raw: raw}

	body := strings.TrimSpace(raw)
	if body == "" {
		return s
	}

	label, qualifier, form := splitQualifier(body)
	s.Label = strings.Trim(strings.TrimSpace(label), "`*_ ")
	s.Qualifier = qualifier
	s.QualifierForm = form
	s.Tier = StatusVocabulary.Classify(s.Label)
	return s
}

// ParseProfile takes the leading label off a Profile value, whose template
// form is "value, plus one clause naming the contract(s) behind it".
func ParseProfile(raw string) (label string, tier Tier) {
	body := strings.TrimSpace(raw)
	// The value is "label, plus one clause naming the contract(s)". The
	// clause may be introduced by punctuation OR, in many records, by a
	// bare parenthetical, so both are cut.
	if i := strings.Index(body, "("); i > 0 {
		body = body[:i]
	}
	if i := strings.Index(body, "_("); i > 0 {
		body = body[:i]
	}
	if loc := clauseBreak.FindStringIndex(body); loc != nil {
		body = body[:loc[0]]
	}
	// A remaining space-separated tail is commentary too: every canonical
	// Profile label is a single word.
	if i := strings.IndexByte(strings.TrimSpace(strings.Trim(body, "`*_ ")), ' '); i > 0 {
		body = strings.TrimSpace(strings.Trim(body, "`*_ "))[:i]
	}
	label = strings.Trim(strings.TrimSpace(body), "`*_.,;: ")
	return label, ProfileVocabulary.Classify(label)
}
