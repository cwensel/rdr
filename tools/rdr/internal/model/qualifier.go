package model

import (
	"regexp"
	"strings"
)

// QualifierForm names which qualifier grammar a Status value carried.
type QualifierForm int

const (
	// NoQualifier means the value was a bare status label.
	NoQualifier QualifierForm = iota
	// QualifierDemotedTarget is `Demoted [→ <issue link>]`: the
	// destination issue a not-RDR-shaped record was refiled as.
	QualifierDemotedTarget
	// QualifierRevisedFrom is `Draft [revised from Final YYYY-MM-DD;
	// re-verify A2,A4 — <reason>]`, written by the cluster gate's
	// Final→Draft flip. The value is still a Draft for every binary gate;
	// only the scoped re-verify and re-lock stages parse the qualifier,
	// and the re-lock's flip to Final overwrites the whole value, so it
	// self-clears.
	QualifierRevisedFrom
	// QualifierJointDecision is `Final [joint decision → <home §-anchor>:
	// <question>]`. Unlike QualifierRevisedFrom this is an open
	// obligation, not a coherence claim, so it does NOT self-clear: a
	// re-lock carries it forward unchanged until the named home answers.
	// The corpus also writes it on Implemented.
	QualifierJointDecision
	// QualifierRevisitWhen is `Deferred [revisit when <condition>]`: the
	// condition that un-parks a Deferred RDR. Unlike the terminal
	// qualifiers it names a FUTURE event, and the record it sits on is
	// still live — when the condition fires the RDR re-enters the flow at
	// the stage it stopped, and the flip overwrites the whole value.
	QualifierRevisitWhen
	// QualifierBracketed is a bracketed qualifier matching none of the
	// named grammars — a free-text note on the live value.
	QualifierBracketed
	// QualifierDash is an undelimited em-dash clause on the live value,
	// `Deferred — no solution decided; revisit when ...`. No template
	// form specifies it, but a frozen record writes it, and reading the
	// whole sentence as the status label would be plainly wrong.
	QualifierDash
	// QualifierParenthetical is the legacy `(...)` form: `Implemented
	// (`main` c1926e1)`, `Rejected (...)`, `Abandoned (date — reason)`.
	// The template only ever specified brackets, but ten frozen records
	// carry parentheses, most often to pin the implementing commit.
	QualifierParenthetical
)

func (q QualifierForm) String() string {
	switch q {
	case NoQualifier:
		return "none"
	case QualifierDemotedTarget:
		return "demoted-target"
	case QualifierRevisedFrom:
		return "revised-from"
	case QualifierJointDecision:
		return "joint-decision"
	case QualifierRevisitWhen:
		return "revisit-when"
	case QualifierBracketed:
		return "bracketed"
	case QualifierDash:
		return "dash"
	case QualifierParenthetical:
		return "parenthetical"
	}
	return "unknown"
}

// Qualifier grammars, as compiled regexes over a qualifier's inner text
// (brackets or parentheses already removed). They are anchored at the
// start and deliberately loose at the end: the trailing free-text half of
// each form is prose an author writes in their own words, and several
// corpus records wrap it across lines.
var (
	// DemotedTargetGrammar matches `→ <issue link>`, capturing the link.
	DemotedTargetGrammar = regexp.MustCompile(`^→\s*(.+)$`)

	// RevisedFromGrammar matches `revised from Final YYYY-MM-DD;
	// re-verify A2,A4 — <reason>`, capturing the date, the assumption
	// list and the reason. The re-verify clause is optional: a demotion
	// that reopens no specific assumption omits it.
	RevisedFromGrammar = regexp.MustCompile(`^revised from Final\s+(\d{4}-\d{2}-\d{2})\s*;\s*(?:re-verify\s+([A-Za-z0-9,\s]+?)\s*)?[—-]\s*(.+)$`)

	// JointDecisionGrammar matches `joint decision → <home §-anchor>:
	// <question>`, capturing the home anchor and the open question.
	JointDecisionGrammar = regexp.MustCompile(`^joint decision\s*→\s*([^:]+?)\s*:\s*(.+)$`)

	// RevisitWhenGrammar matches `revisit when <condition>`, capturing the
	// condition. It also accepts the `revisit if` and `re-open when/if`
	// spellings: the one corpus record that predates the template form
	// wrote its trigger as prose, and an author reaching for this status
	// reaches for whichever verb fits the sentence. The condition itself
	// is free text — it names an external event, so nothing here can
	// validate it beyond requiring that it was written.
	RevisitWhenGrammar = regexp.MustCompile(`(?i)^(?:re-?open|revisit)\s+(?:when|if)\s+(.+)$`)

	// CommitPinGrammar matches the legacy parenthetical commit pin,
	// ``main` <sha>`, capturing the branch and the revision.
	CommitPinGrammar = regexp.MustCompile("^`?([A-Za-z0-9._/-]+)`?\\s+([0-9a-f]{7,40})\\s*$")
)

// splitQualifier separates a status value's leading label from its
// qualifier and reports which grammar the qualifier matched.
//
// It accepts both delimiters. The template specifies brackets; the corpus
// also uses parentheses on frozen terminal records, which can never be
// corrected, so parentheses are read as a qualifier rather than as part of
// the label.
//
// The closing delimiter may be absent: a qualifier that wrapped across
// lines and was joined can still be truncated by an upstream reader, and a
// truncated qualifier should still yield the right label.
func splitQualifier(body string) (label, qualifier string, form QualifierForm) {
	open := strings.IndexAny(body, "[(")

	// A third form the corpus writes: an em-dash clause with no delimiter
	// at all, `Deferred — no solution decided; revisit when ...`. It is a
	// qualifier by any reading, so the label must not swallow the whole
	// sentence. It is checked BEFORE the delimited forms whenever the em
	// dash comes first, because such a clause frequently goes on to
	// contain a parenthesis of its own — a date, an aside — and keying on
	// that parenthesis would put the entire clause inside the label.
	dash := strings.Index(body, " — ")
	if dash > 0 && (open < 0 || dash < open) {
		return strings.TrimSpace(body[:dash]),
			strings.Trim(strings.TrimSpace(body[dash+len(" — "):]), "*_ "),
			QualifierDash
	}

	if open < 0 {
		return body, "", NoQualifier
	}

	label = strings.TrimSpace(body[:open])
	if label == "" {
		return body, "", NoQualifier
	}

	var closer byte = ']'
	fallback := QualifierBracketed
	if body[open] == '(' {
		closer = ')'
		fallback = QualifierParenthetical
	}

	inner := body[open+1:]
	if end := matchingClose(inner, body[open], closer); end >= 0 {
		inner = inner[:end]
	}
	qualifier = strings.TrimSpace(inner)

	switch {
	case DemotedTargetGrammar.MatchString(qualifier):
		form = QualifierDemotedTarget
	case RevisedFromGrammar.MatchString(qualifier):
		form = QualifierRevisedFrom
	case JointDecisionGrammar.MatchString(qualifier):
		form = QualifierJointDecision
	case RevisitWhenGrammar.MatchString(qualifier):
		form = QualifierRevisitWhen
	default:
		form = fallback
	}
	return label, qualifier, form
}

// matchingClose finds the delimiter that closes the one just opened,
// counting nested pairs. Qualifiers in the corpus nest — a joint-decision
// question quotes a parenthetical, a commit note carries an aside — so a
// scan for the first closer would cut the value short. Returns -1 when the
// qualifier is unterminated.
func matchingClose(s string, open, closer byte) int {
	depth := 1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case open:
			depth++
		case closer:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
