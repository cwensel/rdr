package lint

// The current-template reading of a record: every record judged against
// the one TEMPLATE.md, each mechanical finding carrying the bytes that
// would repair it.
//
// This was `--strict`, an opt-in beside a narrower default that spoke
// only about live records. The narrower reading was a kindness aimed at
// the wrong thing: a terminal record's CONTENT is never amended, but its
// STRUCTURE may be brought to the current TEMPLATE.md by tooling with ids
// and content bytes preserved (README §Identifiers), so the advice was
// always actionable and withholding it only hid what a migration costs.
// What the flag actually bought — a quieter report — is the lint tier's
// job, and the tier already does it: everything here is advisory and
// blocks nothing. So there is one reading, and this is it.
//
// WHY A PATCH FIELD AND NOT A `migrate` VERB. The computation is the
// same either way. A patch is inspectable before it touches a file, it
// diffs against the previous run, and it stays useful for every later
// fix-pass on a live record, where the stage rewriting the file wants to
// be told what to write rather than to have it written underneath it.
// Applying is a throwaway script; deriving is the part worth keeping.
// This tool still writes no record.
//
// THE ID A LABEL WRITES IS THE ID THE PROJECTOR ALREADY DERIVED. That is
// the whole safety property of the label rule: `label:missing` proposes
// exactly `Element.Key`, so a citation that resolves before the patch
// resolves to the same element after it. Nothing in the graph moves.
//
// WHAT GETS NO PATCH. A rule earns a patch only when its repair is
// computed. Three kinds of finding are deliberately left with prose
// advice alone:
//
//   - Judgment. Which of two records owns a mutual relation, which
//     element a bare peer citation meant, which of two colliding labels
//     yields. These are decisions, and a bulk-applied guess at them is
//     the corpus-wide damage the never-guess rule exists to prevent.
//   - Ambiguity inside a mechanical rule. A legacy heading with no
//     canonical home, a citation whose text appears twice in its range,
//     a target that resolves to more than one element: the rule is
//     mechanical but THIS instance is not, so the finding is reported
//     and the patch withheld.
//   - Repairs the grammar cannot express. Externalizing an inlined gate
//     moves content to `artifacts/gate.md`, which is a second file; a
//     line-range op over one record cannot say that. It is reported with
//     the range to move and no patch.

import (
	"regexp"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// templateFindings are the current-template conformance rules, run on
// every record, each mechanical finding carrying the bytes that would
// repair it.
func templateFindings(d *scan.Document) []Finding {
	var out []Finding
	out = append(out, headingFindings(d)...)
	out = append(out, labelFindings(d)...)
	out = append(out, citationFindings(d)...)
	out = append(out, vocabularyFindings(d)...)
	out = append(out, gateFindings(d)...)
	return out
}

// --- headings ---------------------------------------------------------

// headingFindings re-reads every heading against the CURRENT template: a
// heading a frozen record wrote may be a level-variant or a legacy alias
// today, and that gap is precisely the migration.
//
// Two codes, because they are two different edits with very different
// risks, and only one of them is safe to apply mechanically.
//
// RE-LEVELLING IS FREE. A canonical section takes the CANONICAL SLUG
// whatever level it is written at — `### Critical Assumptions` and `##
// Critical Assumptions` both project as `NNNN:§critical-assumptions` —
// so re-levelling moves no id, mints no element and breaks no citation.
// It is the largest rule in the pass and the least dangerous one.
//
// RENAMING IS NOT, AND IT CARRIES NO PATCH. A section's id is a slug of
// its HEADING TEXT, so a rename moves the id: `§api-verification`
// ceases to exist and `§critical-assumptions` appears, and any citation
// of the old name dangles. Worse, the canonical name comes with a
// canonical LEVEL, and writing both at once can re-parent the record's
// content. Promoting `#### API Verification` to `## Critical
// Assumptions` widens the section from four lines to twenty-eight, so
// the `#### Normative Contracts` that followed it is swallowed — and its
// contract bullets are then read as ASSUMPTIONS. A structural patch that
// silently refiles a record's contracts is content corruption, whatever
// the alias table says about the heading.
//
// So the rename is reported and left to a hand pass with lint as the
// checker. The corpus makes that cheap: fourteen renames across ten
// records, and not one of the sections involved is cited by anything.
// The alias table is also MANY-TO-ONE — `API Verification` and
// `Dependency Source Verification` are both predecessors of Critical
// Assumptions, and one record carries both — so the repair is
// sometimes a MERGE of two bodies, which is a judgment about ordering
// that no line-range op can express in the first place.
func headingFindings(d *scan.Document) []Finding {
	var out []Finding
	for _, n := range d.Outline {
		if n.Level < 2 {
			continue // the title is the record's, not the template's
		}
		m := model.LookupSection(model.Template, n.Heading, n.Level)
		if m.Canonical == nil {
			continue
		}
		switch m.Kind {
		case model.MatchLevelVariant:
			f := Finding{
				Tier:    TierConformance,
				Code:    "heading:level",
				Element: n.ID,
				Message: "the current template writes " + n.Heading + " at level " +
					itoa(m.Canonical.Level) + "; this record writes it at " + itoa(n.Level),
				LineStart: n.LineStart,
				LineEnd:   n.LineStart,
				Fix:       "re-level the heading; the name and everything under it are unchanged",
			}
			// A PROMOTION MAY SWALLOW THE SECTION'S SIBLINGS. Re-levelling
			// moves no id — a canonical section takes the canonical slug
			// at any level — but it does change what the section CONTAINS.
			// Record 0110 writes `### Critical Assumptions` followed by a
			// sibling `### Resolve Deviations`; promoting the first to
			// `##` makes the second its CHILD, and that section's bullets
			// are then read as three more assumptions the record never
			// wrote. The heading is right and the extent is wrong, which
			// is worse than leaving the level alone.
			//
			// So a promotion is patched only where nothing follows the
			// section at a level the promotion would capture.
			if !promotionCaptures(d, n, m.Canonical.Level) {
				f.Patch = &Patch{
					LineStart: n.LineStart, LineEnd: n.LineStart, Op: OpReplace,
					Text: strings.Repeat("#", m.Canonical.Level) + " " + n.Heading,
				}
			} else {
				f.Fix = "re-level by hand: at level " + itoa(m.Canonical.Level) +
					" this heading would capture a following section that is its sibling today"
			}
			out = append(out, f)
		case model.MatchLegacyAlias, model.MatchCaseVariant:
			out = append(out, Finding{
				Tier:    TierConformance,
				Code:    "section:legacy-name",
				Element: n.ID,
				Message: "heading " + n.Heading + " is a recognised predecessor of the current template's " +
					m.Canonical.Name,
				LineStart: n.LineStart,
				LineEnd:   n.LineStart,
				Fix: "rename to " + m.Canonical.Name + " by hand, with lint as the checker: " +
					"the rename moves this section's id and can re-parent the content below it",
			})
		}
	}
	return out
}

// promotionCaptures reports whether writing the section at want would
// pull a following sibling under it.
//
// The test is the next heading after this section's current extent: if it
// is deeper than want, the promotion captures it and everything below it
// down to the next heading at want or shallower. A demotion — want deeper
// than the level written — captures nothing, because the section only
// shrinks.
func promotionCaptures(d *scan.Document, n scan.Node, want int) bool {
	if want >= n.Level {
		return false
	}
	for _, m := range d.Outline {
		if m.LineStart <= n.LineEnd {
			continue
		}
		// The first heading past this section's current end.
		return m.Level > want
	}
	return false
}

// --- element labels ---------------------------------------------------

// labelFindings propose the id the projector already derived, written
// into the record as a label.
//
// Only ORDINAL-derived kinds are patchable, and that distinction is the
// rule's whole correctness argument. An assumption, a contract, a
// scenario, a round-trip invariant and an alternative all derive their
// key from a position in a list, so writing that position down is a pure
// gain: the id stops depending on the neighbours. A decision derives its
// key from its own LABEL TEXT (`D-refusal-envelope-across-tiers`), so
// there is no ordinal to write; the template's decision label is a
// decision CLASS, and a decision whose label is the author's own prose
// has no template-sanctioned label to migrate to. Those are reported
// with advice and no patch.
//
// BR and F carry no ids after migration at all (README §Identifiers), so
// they are not reported.
func labelFindings(d *scan.Document) []Finding {
	var out []Finding
	mixed := mixedSections(d)
	for _, e := range d.Elements {
		if !e.Derived {
			continue
		}
		// A POSITIONALLY-READ element is never labelled. Where a section
		// labels none of its items, the projector reads the items as
		// elements BY POSITION — a tolerance, not the record's claim —
		// and writing those ids down would convert a guess into an
		// authored fact.
		//
		// The corpus shows exactly how that goes wrong. Record 0030
		// carries `#### Scoping notes (not Critical Assumptions)`, whose
		// heading says in words that its bullets are not assumptions;
		// the projector reads them positionally as A7..A10 anyway.
		// Labelling them asserts the opposite of the heading, and it
		// also flips the section's labelled flag — at which point the
		// scanner drops every UNLABELLED item in the section, and the
		// record's real A1..A6 disappear from the projection. A patch
		// that deletes six assumptions is not a migration.
		//
		// So an element is labelled only where its section already
		// labels some sibling: there the flag is already set, the
		// element is already a real element of that kind, and the id is
		// the one citations resolve to today.
		// Contracts are exempt from the section test: a contract is a
		// ```normative FENCE, read document-wide wherever it sits, so its
		// section is incidental and there is no positional reading to
		// protect. A fence is a contract because it is a fence.
		if e.Kind != ident.Contract && e.Section != "" && !mixed[e.Section] {
			continue
		}
		f := Finding{
			Tier:      TierConformance,
			Code:      "label:missing",
			Element:   e.ID,
			Message:   string(e.Kind) + " element carries the derived id " + e.ID + "; the current template labels it",
			LineStart: e.LineStart,
			LineEnd:   e.LineStart,
		}
		switch e.Kind {
		case ident.Contract:
			// A contract's label is its own line above the fence, so the
			// repair adds a line rather than editing one. e.LineStart is
			// the fence itself when the contract is unlabelled.
			f.Fix = "write **" + string(e.Kind) + e.Key + "** on the line above the fence"
			// An INDENTED fence sits inside another element — the corpus
			// writes ```normative blocks inside an assumption's Evidence
			// — and a label line prepended at column zero ENDS that list
			// item. Record 0119's A16 loses 138 of its 220 lines and two
			// dozen of its edges that way, all of them re-attributed to
			// the document. The contract is real and its id is right;
			// there is simply nowhere at column zero to write the label,
			// so the finding is reported and the patch withheld.
			if labelSiteIsFlat(d, e.LineStart) {
				f.Patch = &Patch{
					LineStart: e.LineStart, LineEnd: e.LineStart, Op: OpPrepend,
					Text: "**" + string(e.Kind) + e.Key + "**",
				}
			}
		case ident.Assumption, ident.Scenario, ident.RoundTrip:
			// These are list items: the label goes inside the item's
			// existing bold lead, or opens one when the item has none.
			line := d.Line(e.LineStart)
			f.Fix = "write the label " + string(e.Kind) + e.Key + " into the item's lead"
			text, ok := spliceLabel(line, string(e.Kind)+e.Key)
			if ok && readsBackAs(e.Kind, text, e.Key) {
				f.Patch = &Patch{LineStart: e.LineStart, LineEnd: e.LineStart, Op: OpReplace, Text: text}
			}
		case ident.Alternative:
			// An alternative is a scaffold HEADING, and its label is the
			// ordinal in the heading text. It is derived only when the
			// heading does not open `Alternative N`, which means the
			// author wrote something else there; rewriting it would
			// change the author's title, so this one is advice.
			f.Fix = "title the scaffold `Alternative " + e.Key + ": <name>` so the ordinal is as written"
		default:
			// Decisions and anything else: content-derived, no ordinal.
			continue
		}
		out = append(out, f)
	}
	return out
}

// mixedSections names the sections that label SOME of their elements —
// the part-way-through state a label patch is safe in. A section that
// labels all of its elements has nothing to propose; one that labels none
// is read positionally and must not be written to (see labelFindings).
func mixedSections(d *scan.Document) map[string]bool {
	labelled := map[string]bool{}
	for _, e := range d.Elements {
		if !e.Derived && e.Section != "" && labelKind(e.Kind) {
			labelled[e.Section] = true
		}
	}
	return labelled
}

// labelKind is the set of kinds that carry a written id after migration.
// BR and F carry none (README §Identifiers), so a labelled BR does not
// make its section safe for an A.
func labelKind(k ident.Kind) bool {
	switch k {
	case ident.Assumption, ident.Contract, ident.Scenario, ident.RoundTrip, ident.Alternative:
		return true
	}
	return false
}

// labelSiteIsFlat reports whether a label line may be written above the
// fence at LineStart without landing inside another element.
//
// Both the fence AND the non-blank line above it must be unindented. The
// fence alone is not enough: a record writes an indented paragraph of a
// list item and then opens the fence at column zero underneath it, and a
// label prepended between the two still sits inside that item. The
// scanner then reads neither the label nor the item as it did before.
func labelSiteIsFlat(d *scan.Document, fence int) bool {
	if indent(d.Line(fence)) != 0 {
		return false
	}
	// The line the label would occupy must not itself be fenced. Where a
	// record's fences nest or a delimiter is unbalanced, the scanner reads
	// the region above a fence as still inside one, and it will not accept
	// a label from fenced lines — so the label would be written and never
	// read back, and the finding would return on every pass. Record 0138's
	// fourth contract is the corpus's one instance.
	if fence > 1 && d.Fenced(fence-1) {
		return false
	}
	for i := fence - 1; i >= 1; i-- {
		l := d.Line(i)
		if strings.TrimSpace(l) == "" {
			continue
		}
		return indent(l) == 0
	}
	return true
}

// indent counts the leading spaces of a line, tabs expanded to one.
func indent(line string) int {
	n := 0
	for _, r := range line {
		if r != ' ' && r != '\t' {
			break
		}
		n++
	}
	return n
}

// boldLead matches a list item whose text opens with a bold run, through
// any leading indent, bullet and task-list checkbox:
//
//   - [x] **Descriptor/instance as separate types is correct.** Every…
//     1. **The values.** …
//
// The three capture groups are the prefix to keep, the bold run's opening
// delimiter, and everything after it.
var boldLead = regexp.MustCompile(`^(\s*(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?)(\*\*)(.*)$`)

// plainItem matches a list item with no bold lead at all.
var plainItem = regexp.MustCompile(`^(\s*(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?)(\S.*)$`)

// alreadyLabelled matches a bold run that already opens with an element
// label, so a second pass over a patched record proposes nothing.
var alreadyLabelled = regexp.MustCompile(`^\*\*(?:A|C|S|RT|D|ALT|BR|F)\d`)

// spliceLabel writes an element label into a list item's lead, returning
// the rewritten line.
//
// It edits ONE line and it never reflows: the item's continuation lines
// are the author's prose and are not this rule's to touch. It declines —
// returning ok false, which withholds the patch — whenever the item's
// shape is not one of the two it understands, because a label written
// into the wrong place is worse than a label not written.
func spliceLabel(line, label string) (string, bool) {
	if m := boldLead.FindStringSubmatch(line); m != nil {
		if alreadyLabelled.MatchString(m[2] + m[3]) {
			return "", false
		}
		// `- **Statement.**` becomes `- **A1 Statement.**`, which is the
		// template's form (`- **A1 [Statement]**`) and the form the
		// scanner's own AssumptionBullet already reads back.
		return m[1] + m[2] + label + " " + m[3], true
	}
	if m := plainItem.FindStringSubmatch(line); m != nil {
		if strings.Contains(m[2], "**") {
			// Bold somewhere other than the lead: the item's shape is
			// not one this rule models, so it declines.
			return "", false
		}
		return m[1] + "**" + label + "** " + m[2], true
	}
	return "", false
}

// readsBackAs asks the SCANNER whether the line it is about to be handed
// would in fact yield the key being written. It is the rule's convergence
// guarantee, and it is asked rather than assumed on purpose.
//
// The label grammars are narrower than the list grammars. A record writes
// its assumptions as a NUMBERED list — `1. **Check gate** (x):` — and
// AssumptionBullet reads a label only off a DASH bullet, so splicing
// `A8` into the numbered item produces a line that still projects as
// derived. The label would be written, the finding would return on the
// next pass, and an applier looping until clean would never terminate.
// A hundred and five corpus assumptions are written that way.
//
// So the rule declines rather than writing a label the reader cannot see.
// Those elements keep their ordinal ids, which are already stable on a
// terminal record, and the finding still names them for a hand pass.
func readsBackAs(kind ident.Kind, line, key string) bool {
	switch kind {
	case ident.Assumption:
		m := model.AssumptionBullet.FindStringSubmatch(line)
		return m != nil && strings.NewReplacer(".", "", "-", "").Replace(strings.TrimPrefix(m[1], "A")) == key
	}
	// Scenarios and round-trip invariants key off a unique list NUMBER or
	// the labeller's own lead, never off a written `S3`, so there is no
	// spelling of the label that would read back. They are reported and
	// left alone.
	return false
}

// --- citations --------------------------------------------------------

// citationFindings propose the colon form for a cross-record citation
// written in one of the older spellings the grammar also reads.
//
// The colon form is what the citation rule asks new references to use
// (README §The citation form). The spaced form and the `§Section Name`
// form resolve identically today, so this is a spelling migration and
// never a repair: only edges that ALREADY RESOLVE are considered, which
// is what makes the rewrite safe. An unresolved citation is a different
// finding entirely (edge:unresolved) and is not this rule's to touch.
//
// ONE PATCH PER LINE, NOT PER CITATION. Thirty-three lines in the corpus
// carry more than one citation:
//
//	inherits cli/0035 D2+D4; drop-safety inherits cli/0035 D6
//
// A patch per citation computes each replacement against the ORIGINAL
// line, so applying the second discards the first and one citation stays
// unmigrated — silently, because both patches applied cleanly. The fix
// is not to order them: it is to make the patch's unit the line, so a
// line is rewritten once with every citation on it already substituted.
// The findings stay per-citation, because each is its own fact; they
// share the one patch that repairs their line.
func citationFindings(d *scan.Document) []Finding {
	var out []Finding
	// Per line, the substitutions that line needs, in the order the
	// edges were read.
	pending := map[int][]substitution{}
	var lines []int

	for _, e := range d.Edges {
		if e.Resolved == nil || !*e.Resolved {
			continue
		}
		if !e.Kind.Typed() && e.Kind != edge.Mentions {
			continue
		}
		if !strings.Contains(e.To, ":") || strings.Contains(e.Evidence, ":") {
			continue // names a whole record, or is already the colon form
		}
		// SECTION AND ANCHOR CITATIONS ARE NOT REWRITTEN. A `§Name`
		// citation is a bounded FRAGMENT of the target's heading or bold
		// lead — the grammar reads at most six words (edge.sectionName) —
		// while the id it resolves to is the slug of the WHOLE text.
		// Rewriting `cli/0051 §Normative Contracts contract 2 straddle
		// rejection` to the full `…-envelope` slug replaces what the
		// author wrote with something longer, and across the corpus 23
		// such citations stop resolving. Every one of the losses is a
		// section citation and not one is an element citation, which is
		// the line this rule now draws: element ids are exact and safe to
		// spell differently; section ids are derived from prose and are
		// not this rule's to restate.
		if strings.Contains(e.To, ":"+string(ident.Section)) {
			continue
		}
		want := colonForm(e.Evidence, e.To)
		if want == "" {
			continue
		}
		f := Finding{
			Tier:      TierConformance,
			Code:      "citation:form",
			Element:   e.From,
			Message:   "citation " + e.Evidence + " resolves to " + e.To + "; the current form is the colon id",
			LineStart: e.Line,
			LineEnd:   e.LineEnd,
			Fix:       "write the citation as " + want,
		}
		if line, ok := uniqueLine(d, e.Line, e.LineEnd, e.Evidence); ok {
			f.LineStart, f.LineEnd = line, line
			if _, seen := pending[line]; !seen {
				lines = append(lines, line)
			}
			pending[line] = append(pending[line], substitution{from: e.Evidence, to: want})
		}
		out = append(out, f)
	}

	// Build one patch per line and hand the SAME pointer to every finding
	// on it. An applier that walks findings applies the line once however
	// many findings name it, and a reader sees the whole repair rather
	// than one citation's share of it.
	patches := map[int]*Patch{}
	for _, line := range lines {
		text := d.Line(line)
		for _, sub := range pending[line] {
			text = strings.Replace(text, sub.from, sub.to, 1)
		}
		patches[line] = &Patch{LineStart: line, LineEnd: line, Op: OpReplace, Text: text}
	}
	for i := range out {
		if p, ok := patches[out[i].LineStart]; ok && out[i].LineStart == out[i].LineEnd {
			out[i].Patch = p
		}
	}
	return out
}

// substitution is one citation's rewrite within a line.
type substitution struct{ from, to string }

// colonForm renders the resolved target in the colon spelling, keeping
// whatever record prefix the author wrote.
//
// The author's prefix is kept deliberately. `cli/0055 A5` becomes
// `cli/0055:A5` and a bare `RDR 0055 A5` becomes `RDR 0055:A5`: the
// migration is the SEPARATOR, and adding or dropping a project prefix
// would be a second change this rule has no mandate to make.
func colonForm(evidence, to string) string {
	i := strings.LastIndex(to, ":")
	if i < 0 {
		return ""
	}
	element := to[i+1:]
	// The record half of the citation, as the author wrote it: everything
	// up to the separator that precedes the ELEMENT half.
	//
	// The separator is found from the RIGHT, not the left, because the
	// record half can itself contain a space: the corpus writes `RDR
	// 0055 C4` and `RDR cli/0055 §Approach` as well as `cli/0055 A5`.
	// Cutting at the first space would render `RDR 0055 C4` as `RDR:C4`,
	// which names no record at all.
	loc := recordHalf.FindStringSubmatchIndex(evidence)
	if loc == nil {
		return ""
	}
	record := strings.TrimSpace(evidence[:loc[3]])
	if record == "" {
		return ""
	}
	return record + ":" + element
}

// recordHalf matches the record half of a citation — an optional `RDR`
// or `R-` lead, an optional `project/` prefix, and the four digits — so
// everything after it is the element half whatever separator was used.
var recordHalf = regexp.MustCompile(`^((?:RDR[ -]|R-)?(?:[a-z][a-z0-9_-]*/)?\d{4})`)

// uniqueLine finds the one line in start..end that carries the text
// exactly once, and reports whether the text occurs exactly once across
// the whole range.
//
// Both conditions are required, and this is where the rule is at its most
// careful. A field-borne citation's range is the WHOLE FIELD, several
// lines of wrapped prose, and the edge does not record which of them the
// citation was read from — the field's value is joined and whitespace-
// normalised before the reference scanner sees it, so its offsets do not
// map back to source bytes. Rather than plumb offsets through that join,
// the rule asks the question a patch actually needs answered: is there
// exactly one place in this range where the text stands? If the answer is
// no — the citation repeats, or it is split across a wrap — the finding
// is reported without a patch.
//
// A line inside a fence is never a candidate: fenced text is quoted, and
// rewriting a citation there edits the quote rather than the claim.
func uniqueLine(d *scan.Document, start, end int, text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	found, total := 0, 0
	for i := start; i <= end; i++ {
		if d.Fenced(i) {
			continue
		}
		n := strings.Count(d.Line(i), text)
		total += n
		if n > 0 {
			found = i
		}
	}
	if total != 1 {
		return 0, false
	}
	return found, true
}

// --- vocabularies -----------------------------------------------------

// vocabularyFindings propose the canonical spelling of a Status, Type or
// Method value the corpus wrote in an accepted-but-not-canonical form.
//
// Matching in the vocabularies is exact and case-sensitive by design, so
// an OffVocabulary value is a value the template has no spelling for —
// a judgment, not a migration, and it gets no finding here. Only the
// ObservedAccepted tier is mechanical: those values are recorded in the
// vocabulary precisely because a real record wrote them, which is what
// makes the mapping known rather than guessed.
func vocabularyFindings(d *scan.Document) []Finding {
	var out []Finding
	for _, f := range d.Metadata {
		v, ok := vocabularyFor(f.Canonical)
		if !ok {
			continue
		}
		raw := strings.TrimSpace(f.Value)
		if raw == "" || v.Classify(raw) != model.ObservedAccepted {
			continue
		}
		want := canonicalFor(v, raw)
		if want == "" {
			continue
		}
		fd := Finding{
			Tier:      TierConformance,
			Code:      "vocabulary:spelling",
			Element:   f.Canonical,
			Message:   f.Canonical + " is written " + raw + "; the current template spells it " + want,
			LineStart: f.LineStart,
			LineEnd:   f.LineStart,
			Fix:       "write the canonical spelling",
		}
		if line, ok := uniqueLine(d, f.LineStart, f.LineEnd, raw); ok {
			fd.LineStart, fd.LineEnd = line, line
			fd.Patch = &Patch{
				LineStart: line, LineEnd: line, Op: OpReplace,
				Text: strings.Replace(d.Line(line), raw, want, 1),
			}
		}
		out = append(out, fd)
	}
	return out
}

func vocabularyFor(field string) (model.Vocabulary, bool) {
	switch field {
	case "Status":
		return model.StatusVocabulary, true
	case "Type":
		return model.TypeVocabulary, true
	case "Profile":
		return model.ProfileVocabulary, true
	}
	return model.Vocabulary{}, false
}

// canonicalFor names the one canonical spelling an observed value maps
// to, or "" when more than one could be meant. Case is the only variation
// the mapping reads: anything further apart than that is a rewording, and
// a rewording is the author's, not the tool's.
func canonicalFor(v model.Vocabulary, observed string) string {
	var hit string
	for _, c := range v.Canonical {
		if strings.EqualFold(c, observed) {
			if hit != "" {
				return "" // ambiguous; say nothing
			}
			hit = c
		}
	}
	return hit
}

// --- the gate ---------------------------------------------------------

// gateFindings report an inlined Finalization Gate: at lock,
// moves the responses out of the record into `artifacts/gate.md` and
// leaves a one-line pointer behind.
//
// It carries NO PATCH, and that is a property of the repair rather than
// of the rule. The migration moves content into a SECOND FILE, and the
// patch grammar is whole-line surgery over one record — it has no way to
// say "these lines become that file, and this line stands in for them".
// Extending the grammar for the one rule that needs it would give the
// applier a special case to carry forever, so the finding names the range
// to move and leaves the move to the pass that can do it.
func gateFindings(d *scan.Document) []Finding {
	var out []Finding
	for _, n := range d.Outline {
		if n.Canonical != "Finalization Gate" {
			continue
		}
		if hasGatePointer(d, n) {
			continue
		}
		if !hasGateElements(d, n) {
			continue
		}
		out = append(out, Finding{
			Tier:      TierConformance,
			Code:      "gate:inline",
			Element:   n.ID,
			Message:   "the gate responses are inlined; they belong in artifacts/gate.md",
			LineStart: n.LineStart,
			LineEnd:   n.LineEnd,
			Fix:       "move these responses to the record's artifacts/gate.md and leave the one-line pointer (a cross-file move; no patch)",
		})
	}
	return out
}

// hasGatePointer reports whether the gate body is already the pointer:
// the first non-blank line under the heading, before any sub-heading.
func hasGatePointer(d *scan.Document, n scan.Node) bool {
	for i := n.LineStart + 1; i <= n.LineEnd; i++ {
		l := strings.TrimSpace(d.Line(i))
		if l == "" {
			continue
		}
		if model.Heading.MatchString(l) {
			return false
		}
		return model.GatePointer.MatchString(l)
	}
	return false
}

// hasGateElements reports whether the projector read gate responses under
// the gate node — which is the evidence that the responses are in fact
// inlined, rather than the section merely being empty.
//
// An item the template RETAINS at lock does not count: it is in the record
// on purpose, and a correctly locked gate that keeps it would otherwise be
// reported as unmigrated forever, with nothing to repair. Which items
// those are is `model.GateItems`' to answer, from the template's own
// marker — never a name written down here.
//
// A gate element's Section is its OWN sub-heading (`NNNN:§contradiction-check`),
// never the gate's, because the projector reads one element per sub-heading
// under the gate and stamps each with the node it came from. Matching the
// gate's ID against that section — by prefix or by equality — therefore
// never fires. The relation to test is the one the projector itself used to
// mint these elements: the element's section node is a child of the gate.
func hasGateElements(d *scan.Document, gate scan.Node) bool {
	under := map[string]bool{}
	for _, n := range d.Outline {
		if n.Parent == gate.ID {
			under[n.ID] = true
		}
	}
	for _, e := range d.Elements {
		if e.Kind == ident.Gate && under[e.Section] && !model.GateItemRetained(e.Key) {
			return true
		}
	}
	return false
}
