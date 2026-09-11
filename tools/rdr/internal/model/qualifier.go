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
	// QualifierRoutedBack is `Draft [routed back from resolve
	// YYYY-MM-DD; re-verify A5,A6 @propose — <reason>]`: the Draft-side
	// sibling of QualifierRevisedFrom, written when a mid-flow stage
	// sends a Draft BACKWARD.
	//
	// It exists because the route-back used to be decide-only. The packet
	// named the stage and nothing reached the record, so a Draft sent back
	// from Stage 4 looked — to the very next skill — exactly like a Draft
	// that had reached Stage 4 on its own: the evidence on disk says
	// FORWARD, only the transcript said BACKWARD, and the transcript is
	// not what the next stage reads. The receiving stage refused as
	// already-passed and the record deadlocked.
	//
	// Like the revised-from form it rides on the LIVE Status value, so a
	// history scrub cannot strip it, and the receiving stage clears it by
	// overwriting the whole value. Unlike it, the form names the ORIGIN
	// stage as well as the target: "back to Propose from Resolve" and
	// "back to Propose from Stage 8" owe different rework.
	//
	// It is appended LAST rather than filed beside QualifierRevisedFrom
	// on purpose — see the note above String().
	QualifierRoutedBack
)

// The const block's numeric values are not persisted anywhere: every
// crossing spells the form as String() — the projector writes
// `status.form` as text, the fact table's `status_form` domain lists the
// same words, and no comparison in the tree reads the int. So an
// insertion would be safe on the letter of it.
//
// A new form is still APPENDED, because one reader walks the enum by
// ordinal from zero until String() returns "unknown" (facts_test's
// allQualifierForms, which pins the fact domain to this list). Appending
// keeps that walk's output a superset of what it was; inserting reorders
// a list a test compares as a set today and could compare as a sequence
// tomorrow, for readability alone.

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
	case QualifierRoutedBack:
		return "routed-back"
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
	// re-verify A2,A4 @refine — <reason>`, capturing the date, the
	// assumption list, the target re-entry stage and the reason (groups
	// 1–4). The re-verify clause is optional: a demotion that reopens no
	// specific assumption omits it. `re-verify none` is the sanctioned
	// spelling of that same empty set, so `none` matches without being
	// captured — a captured `none` would read as an assumption label and
	// emit a re-verify edge to nothing.
	//
	// `@<stage>` names the stage the record re-enters at — one of
	// ReentryTargets, the verb of the skill that runs it — and is what
	// the navigator routes on. Without it a re-entry routes to the
	// resolve fallback, which is the historical reading: records written
	// before the slot existed parse unchanged.
	//
	// The `— <reason>` tail is optional too. It is the human half of the
	// qualifier, and a demote that forgot it must still READ as a
	// re-entry: the form is what every routing and staleness gate keys
	// on, and a missing reason degrading the value to a free-text note
	// would let a stale lock through unrefused.
	//
	// Between the date and the semicolon the grammar tolerates a short
	// run of lowercase stage tokens (`2026-08-29 cluster-reconcile;`):
	// a live demote pass wrote the flipping stage's name there, and
	// rejecting it silently degraded the qualifier to a free-text note
	// that blinded every re-entry routing rule. The run is capped at two
	// tokens so the slot stays a stamp, not a sentence.
	RevisedFromGrammar = regexp.MustCompile(`^revised from Final\s+(\d{4}-\d{2}-\d{2})(?:\s+[a-z][a-z0-9-]*){0,2}\s*;\s*(?:re-verify\s+(?:none|([A-Za-z0-9,\s]+?))\s*)?(?:@(propose|refine|resolve|finalize)\b\s*)?(?:[—-]\s*(.+))?$`)

	// RoutedBackGrammar matches `routed back from resolve YYYY-MM-DD;
	// re-verify A5,A6 @propose — <reason>`, capturing the ORIGIN stage,
	// the date, the assumption list, the target stage and the reason
	// (groups 1–5). It is RevisedFromGrammar's shape with one group
	// prepended, and every tolerance there is deliberate here too: the
	// re-verify clause is optional, `none` is the sanctioned empty set
	// and matches without being captured, and a qualifier that forgot its
	// `— <reason>` is still the form, because a routing gate keys on the
	// form and degrading the value to a free-text note is exactly the
	// blindness this qualifier exists to end.
	//
	// The ORIGIN vocabulary is the stages that can send a Draft back —
	// the `return` group's callers — which is a different list from the
	// TARGET vocabulary below: implement and cluster-reconcile route
	// records back without being stages a record re-enters at.
	//
	// It does NOT carry the revised-from grammar's two-token tolerance
	// after the date. That tolerance is a concession to seven frozen
	// records; this form has no corpus yet, so the stamp stays a stamp.
	RoutedBackGrammar = regexp.MustCompile(`^routed back from\s+(resolve|prelock|reconcile|finalize|implement|cluster-reconcile)\s+(\d{4}-\d{2}-\d{2})\s*;\s*(?:re-verify\s+(?:none|([A-Za-z0-9,\s]+?))\s*)?(?:@(propose|refine|resolve|prelock|reconcile|finalize)\b\s*)?(?:[—-]\s*(.+))?$`)

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

// ReentryTargets is the vocabulary of the `@<stage>` slot in
// RevisedFromGrammar: the stages a demoted record can re-enter at, spelled
// as the verb of the skill that runs each (Stage 2 propose, Stage 3 refine,
// Stage 4 resolve, and Stage 7 finalize for a RE-LOCK-ONLY re-entry whose
// wording fixes are applied in the lock pass itself). The regexp above and
// the fact table's `reentry_target` domain both spell this list; this is
// the one Go copy.
var ReentryTargets = []string{"propose", "refine", "resolve", "finalize"}

// RouteBackTargets is the vocabulary of the `@<stage>` slot in
// RoutedBackGrammar: a SUPERSET of ReentryTargets, adding the two stages
// a route-back can name that a demotion cannot — prelock (the `return`
// group's determinacy row, `/rdr-prelock repeatability 1`) and reconcile
// (its spike and disturbed-assumption rows).
//
// They are two lists and not one on purpose. ReentryTargets is a
// CONTRACT, not just a spelling: it is pinned to the demote rows'
// `[emit.stage]` domain, and widening it would claim the write model
// emits prelock and reconcile DEMOTIONS, which it cannot — a Final has
// closed Stages 5 and 6, so there is nothing there to send it back to.
// Only the route-back, which acts on a live Draft, can name them.
//
// The `reentry_target` fact's domain is the UNION, because the slot is
// one slot: both forms write it and the navigator's `reentry` group
// routes both through the same rows. The relationship between the two
// lists is checked rather than asserted — see
// TestRouteBackTargetsExtendReentryTargets.
var RouteBackTargets = []string{"propose", "refine", "resolve", "prelock", "reconcile", "finalize"}

// RouteBackOrigins is the vocabulary of the origin slot in
// RoutedBackGrammar: the stages that send a Draft back. It is neither
// list above — implement and cluster-reconcile route records back
// without ever being a stage a record re-enters AT.
var RouteBackOrigins = []string{"resolve", "prelock", "reconcile", "finalize", "implement", "cluster-reconcile"}

// ReentryTarget reads the `@<stage>` off a re-entry qualifier — either
// the revised-from form or its routed-back sibling — or "" when the
// qualifier is neither form or names no target. Absent is not a default:
// the caller decides what an untargeted re-entry means.
//
// It answers for BOTH forms because every caller asks the same question
// (where does this record go next) and none of them asks it about the
// revised-from form specifically: the projector calls it under a form
// check it already made, and the fact reads the projected field. Making
// the new form a second call site would leave each of those callers one
// edit away from routing a routed-back record to nowhere.
func ReentryTarget(qualifier string) string {
	q := strings.TrimSpace(qualifier)
	if m := RevisedFromGrammar.FindStringSubmatch(q); m != nil {
		return m[3]
	}
	return RouteBackTarget(q)
}

// RouteBackTarget reads the `@<stage>` off a routed-back qualifier
// ONLY, or "" otherwise. It is the narrow accessor a caller reaches for
// when the two forms owe different handling — the origin stage is
// readable only here.
func RouteBackTarget(qualifier string) string {
	m := RoutedBackGrammar.FindStringSubmatch(strings.TrimSpace(qualifier))
	if m == nil {
		return ""
	}
	return m[4]
}

// RouteBackOrigin reads the stage that SENT a routed-back record back,
// or "" when the qualifier is not that form. The target says where the
// record goes; the origin says how much rework it owes on arrival.
func RouteBackOrigin(qualifier string) string {
	m := RoutedBackGrammar.FindStringSubmatch(strings.TrimSpace(qualifier))
	if m == nil {
		return ""
	}
	return m[1]
}

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
	// Beside its sibling, and it cannot be shadowed in either direction:
	// both grammars are anchored on a distinct opening phrase (`revised
	// from Final` / `routed back from`), so no qualifier can satisfy the
	// two. The placement is for the reader, not for the parse.
	case RoutedBackGrammar.MatchString(qualifier):
		form = QualifierRoutedBack
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
