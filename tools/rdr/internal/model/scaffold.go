package model

import "regexp"

// Scaffold headings are the template sections whose names an instance
// SUBSTITUTES rather than copies.
//
// TEMPLATE.md writes them with a bracketed placeholder — `### Alternative
// 1: [Name]`, `#### Step 1: [Title]`, `#### Activation Step 1: [Title]` —
// and every real record fills the placeholder in and repeats the block as
// many times as it needs. So the literal template heading almost never
// appears in a record, and the substituted headings are template-drawn
// even though no fixed string matches them.
//
// Matching these by name alone is the single largest source of false
// `unknown-to-template` findings: they account for the clear majority of
// the headings a name-only lookup cannot place, across the whole corpus.
// They are not foreign sections — they are the template's own scaffolding
// wearing an author's title — so they are matched by pattern here.
//
// The ordinal is deliberately unbounded: records write Alternative 9 and
// Step 12, and the template's `1` / `2` are examples of a repeated block,
// not a limit. Suffixes after the ordinal (`Alternative 1 (B)`,
// `Alternative 3b`, `Alt 1`) are the same block with an author's
// disambiguator.
type scaffoldPattern struct {
	// re matches the substituted heading.
	re *regexp.Regexp
	// canonical is the template section the pattern belongs to.
	canonical string
	// note explains the match.
	note string
}

var scaffoldPatterns = []scaffoldPattern{
	{
		// `Alternative 1: Name`, `Alternative 2 (B): Name`, `Alt 3b: Name`,
		// `Alternative 1 — Name`. The template's own `Alternative 1: [Name]`
		// matches too, so the literal placeholder needs no special case.
		re:        regexp.MustCompile(`(?i)^Alt(ernative)?\s+\d+[a-z]?\b`),
		canonical: "Alternative 1: [Name]",
		note:      "an instance of the Alternatives Considered scaffold, with the placeholder filled in",
	},
	{
		// `Alternative 0: ...` — a record numbering from zero.
		re:        regexp.MustCompile(`(?i)^Alt(ernative)?\s+0\b`),
		canonical: "Alternative 1: [Name]",
		note:      "an instance of the Alternatives Considered scaffold, numbered from zero",
	},
	{
		// `Activation Step 1: Title`. Checked before the plain Step
		// pattern, which would otherwise not match it at all but is
		// listed first here for clarity of intent.
		re:        regexp.MustCompile(`(?i)^Activation Step\s+\d+\b`),
		canonical: "Activation Step 1: [Title]",
		note:      "an instance of the Phase 2 activation-step scaffold",
	},
	{
		// `Step 1: Title`, `Step 12: Title`.
		re:        regexp.MustCompile(`(?i)^Step\s+\d+\b`),
		canonical: "Step 1: [Title]",
		note:      "an instance of the Phase 1 implementation-step scaffold",
	},
	{
		// `Phase 1: Code`, `Phase 3: Rollout`, `Phase 2a: ...`. The
		// template ships Phase 1 and Phase 2 by name; records add further
		// phases and reword the titles, which is the same scaffold.
		re:        regexp.MustCompile(`(?i)^Phase\s+\d+[a-z]?\b`),
		canonical: "Phase 1: Code Implementation",
		note:      "an instance of the Implementation Plan phase scaffold",
	},
}

// matchScaffold reports whether a heading is a filled-in template
// scaffold, and which canonical section it belongs to.
func matchScaffold(name string) (canonical, note string, ok bool) {
	for _, p := range scaffoldPatterns {
		if p.re.MatchString(name) {
			return p.canonical, p.note, true
		}
	}
	return "", "", false
}
