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
// grammar, ships in the same commit as its Template table entry and a synthetic
// fixture. TemplateTest in template_test.go enforces the section half
// mechanically.
//
// READ, NEVER JUDGE. Terminal records (Implemented, Rejected, Abandoned,
// Superseded, Demoted) are never amended. They are frozen at the template
// that produced them, forever. The model therefore reads what the
// corpus contains, which is a strictly larger language than what a new
// record may write — see Vocabulary and its three tiers.
package model

import (
	"regexp"
	"strconv"
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

// Section is one entry in the template's ordered section table.
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
	// Keys records whether the section's own body shows its items
	// carrying a key the projector can read back as an id — a numbered
	// list (`1. **Scenario**:`), a labelled lead (`- **A<N> …**`), a
	// keyed table row. Where the template shows an unnumbered bullet
	// (`- **[Alternative N]**:` under Briefly Rejected) or plain prose
	// (Failure Modes), there is no key to write and the projector's
	// ordinal IS the element's identity.
	//
	// This is the template's to say, for the reason Retained is: a rule
	// the reader spells out for itself is a second source for one fact,
	// and the two drift. Nothing coarser answers it — Testing Strategy,
	// Briefly Rejected and Failure Modes all carry prose, and only the
	// first numbers its items — so it is read from the body itself.
	Keys bool
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

// --- Seam Lineage ------------------------------------------------------
//
// TEMPLATE.md fixes the field's form: `<seam> — Nth point-fix; trail: …`,
// or "no prior accretion", with an optional `Accretion disposition: …`
// line as the floor's only escape. The count and the disposition are the
// two facts the accretion floor routes on (rdr-status.toml `floor`), so
// the grammar that reads them lives here, once, beside the wrap rule.
//
// A count is READ, never inferred: the spellings below are the ones the
// corpus writes unambiguously, each anchored on the literal `point-fix`
// (or the template's own `no prior accretion`). Anything else — `Nth
// point-fix (N≥3)`, `would be point-fixes #3–#5` — is UNREAD, which the
// scanner reports as such rather than guessing a number. The first form
// to match wins, in this order, so a disposition sentence quoting "the 3
// point-fixes" never outranks the field's own `3rd point-fix`.
var seamLineageCounts = []struct {
	form string
	re   *regexp.Regexp
}{
	{"ordinal", regexp.MustCompile(`(?i)\b(\d+)(?:st|nd|rd|th)\+?\s+point-fix`)},
	{"prior-count", regexp.MustCompile(`(?i)\b(\d+)\s+prior\s+(?:closed\s+)?(?:code\s+)?point-fix`)},
	{"count-eq", regexp.MustCompile(`(?i)\bcount\s*=\s*(\d+)\b`)},
	{"none-declared", regexp.MustCompile(`(?i)\bno prior (?:closed )?(?:accretion|point-fix)`)},
}

// SeamLineageCount reads the point-fix count out of a Seam Lineage value.
// It returns the count, the form that carried it, and false when no
// declared form is present (the value is unread, not zero).
func SeamLineageCount(value string) (n int, form string, ok bool) {
	for _, c := range seamLineageCounts {
		m := c.re.FindStringSubmatch(value)
		if m == nil {
			continue
		}
		if c.form == "none-declared" {
			return 0, c.form, true
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		return n, c.form, true
	}
	return 0, "", false
}

// accretionDisposition matches the escape line where the template puts
// it — opening a sentence or a nested bullet, optionally bold, with the
// colon optional (`**Accretion disposition (written):**` is written). It
// does NOT match the template's own guidance when an author copies it
// into the field (`… of the form: \`Accretion disposition: the N …`),
// because there the label sits mid-sentence after a colon.
var accretionDisposition = regexp.MustCompile(`(?i)(?:^|[.;!?]\s+|^\s*-\s+)[*_]*accretion disposition\b`)

// AccretionDispositionLine reports whether text carries the written
// accretion disposition. Callers pass the field's value and each of its
// nested bullets separately — the value's wrap rule cuts a nested bullet
// off, and the corpus writes the disposition both ways.
func AccretionDispositionLine(text string) bool {
	return accretionDisposition.MatchString(text)
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
// Finalization Gate body at lock.
var GatePointer = regexp.MustCompile(`gate\.md`)

// AuthoringMarker matches TEMPLATE.md's own guidance blocks — the
// bracketed clauses that tell an author what a section owes. They declare
// the section's class while the template is being read (parse.go), and in
// a RECORD they are text the author was meant to delete: the template's
// words standing where the author's should be.
//
// It used to be `^\[(Required|Conditional)\b`, a literal that named two
// of the template's lead words. TEMPLATE.md writes about forty bracketed
// leads and only three begin that way, so the rule saw a fraction of its
// own subject: `[Instructions]`, `[Architecture, component relationships,
// …`, `[Shape only — …`, `[What was analyzed? …` and the rest were
// invisible, and 24 records in the reference corpus carry one that lint
// never reported. The fix is the one the template already affords —
// ASK THE TEMPLATE — so a lead the template stops writing stops being a
// marker, and one it starts writing is matched without a code change.
//
// It is deliberately NOT every bracketed lead. `[Gate key: …]` and
// `[Retained at lock — …]` are SCHEMA markers the reader parses as data
// and a record legitimately carries them — matching those would report
// the template's own grammar as a defect, which is the false finding the
// never-guess rule exists to prevent. They are excluded by name because
// the template gives no other signal that they differ in kind.
//
// Anchored at column zero because a guidance block opens its own line; an
// indented bracket is a field's placeholder and has an owner.
var AuthoringMarker = authoringMarker{}

// schemaMarkers are the bracketed leads the reader consumes as DATA
// rather than as guidance an author should have deleted.
var schemaMarkers = []string{"Gate key", "Retained at lock"}

// authoringMarker answers whether a line opens one of the template's
// guidance blocks. It carries no state: the set is read from the bound
// schema on each call, so a test that rebinds a different template sees
// that template's markers.
type authoringMarker struct{}

// MatchString reports whether the line opens a guidance block.
//
// A record's copy is matched by its LEAD — the words up to the first
// sentence break — rather than by the whole block, because records
// wrapped and trimmed these blocks as they were pasted. The most common
// divergence is a dropped `[Required — never omit.` prefix, which leaves
// `[Load-bearing — implementers must match exactly.` standing on its own
// in thirteen records; matching the whole clause would miss every one.
func (authoringMarker) MatchString(line string) bool {
	if !strings.HasPrefix(line, "[") {
		return false
	}
	lead := markerLead(line)
	if lead == "" {
		return false
	}
	for _, m := range TemplateMarkerLeads() {
		if strings.HasPrefix(lead, m) || strings.HasPrefix(m, lead) {
			return true
		}
	}
	return false
}

// markerLead reduces a bracketed opening to the words a template lead and
// a record's copy of it have in common: the text up to the first sentence
// break, lower-cased.
func markerLead(line string) string {
	t := strings.TrimPrefix(strings.TrimSpace(line), "[")
	return sentenceLead(t)
}

// sentenceLead is the first sentence of a clause, lower-cased. Below four
// characters there is nothing left to compare and the answer is empty.
func sentenceLead(t string) string {
	for _, cut := range []string{".", "?", "]", ":"} {
		if i := strings.Index(t, cut); i >= 0 {
			t = t[:i]
		}
	}
	t = strings.TrimSpace(strings.ToLower(t))
	if len(t) < 4 {
		return ""
	}
	return t
}

// TemplateMarkerLeads are the guidance-block leads TEMPLATE.md writes,
// schema markers excluded. Derived on each call from the bound schema.
func TemplateMarkerLeads() []string {
	var out []string
	for _, line := range strings.Split(current().TemplateSource, "\n") {
		if !strings.HasPrefix(line, "[") {
			continue
		}
		// Every sentence of the opening line is a candidate lead, not
		// just the first. Records trimmed these blocks as they pasted
		// them, and the commonest trim drops a leading
		// `[Required — never omit.` — which leaves the SECOND sentence
		// standing as the record's own opening. Indexing only the first
		// would miss every such copy.
		body := strings.TrimPrefix(strings.TrimSpace(line), "[")
		for _, part := range strings.Split(body, ".") {
			lead := sentenceLead(part)
			if lead == "" {
				continue
			}
			skip := false
			for _, sm := range schemaMarkers {
				if strings.HasPrefix(lead, strings.ToLower(sm)) {
					skip = true
				}
			}
			if !skip {
				out = append(out, lead)
			}
		}
		continue
	}
	return out
}

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
func DecisionClasses() []string { return decisionClasses(current().Sections) }

// DecisionClassOf returns the template class a decision label opens with,
// or "". `Selection / predicate (remainder)` and `Selection` both resolve
// to `Selection / predicate`; the match is case-insensitive and takes
// the longest class that is a prefix of the label, so the shorter
// `Selection` cannot shadow the full name.
func DecisionClassOf(label string) string {
	l := strings.ToLower(strings.TrimSpace(label))
	best := ""
	for _, c := range DecisionClasses() {
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
// inlined gate response is addressable as `NNNN:G-scope`
// regardless of the heading's exact wording.
//
// Retained records the template's `[Retained at lock — …]` marker: at lock
// the gate's responses move to gate.md, but a retained item STAYS in the
// record. Which item that is, and whether there is one at all, is the
// template's to say — a rule the reader spelled out for itself would be a
// second source for one fact, and the two would drift. The same-commit
// test binds this column to the marker.
// GateItem is one Finalization Gate sub-section: its heading, the key it
// is cited by, and whether the template keeps it in the record at lock.
type GateItem struct {
	Section, Key string
	Retained     bool
}

func GateItems() []GateItem {
	g, _ := gateItems(current().Sections)
	return g
}

// SectionKeys reports whether the named canonical section's body shows
// its items carrying a readable key. An unknown name reports false: a
// section the template does not model cannot be said to key anything.
func SectionKeys(name string) bool {
	for _, s := range current().Table.Sections {
		if s.Name == name {
			return s.Keys
		}
	}
	return false
}

// ElementSections names the canonical section each element kind is
// projected from, by the kind's ID token. It is the one place the reader
// says which section answers for which kind, so a question about a kind
// — does the template key it? — is answerable from the template table
// rather than from a list restated beside it.
//
// The kinds absent here are keyed by something other than a section's
// list shape: G by the gate item, JC by a line anywhere in the record,
// § by the heading itself.
func ElementSections() map[string]string { return current().Sidecar.ElementSections }

// KindKeys reports whether the template gives the element kind — named by
// its ID token — somewhere to write a key. A kind with no section here
// reports false.
//
// This is what separates the two things a derived id can mean. Where the
// template keys the kind, a derived id is a BACKLOG: the slot is empty
// and writing the label pins the id against a sibling being inserted
// before it. Where the template keys nothing, the ordinal IS the
// element's identity — the standing MVV has always had — and no edit
// would improve it. Counting the second as backlog reports work that
// cannot be done: BR 436/436 and F 540/540 read as 976 pending edits
// corpus-wide against a real backlog of zero, which made "stop minting
// them" look like the way to clear the queue.
//
// Both are minted, and neither is a defect. This says only which one an
// author could ever act on — and it says it by reading the template,
// so a template that starts numbering Briefly Rejected changes one table
// row and the reader follows.
func KindKeys(kind string) bool {
	s, ok := ElementSections()[kind]
	return ok && SectionKeys(s)
}

// GateItemRetained reports whether a G-key names an item the template
// keeps in the record at lock.
func GateItemRetained(key string) bool {
	for _, g := range GateItems() {
		if g.Key == key {
			return g.Retained
		}
	}
	return false
}

// GateItemKey returns the G-key for a canonical gate sub-section, or "".
func GateItemKey(section string) string {
	for _, g := range GateItems() {
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
	for _, g := range GateItems() {
		if g.Key == key {
			return true
		}
	}
	return false
}

// TemplateLiterals are the backticked literals TEMPLATE.md itself writes.
//
// A record that kept the template's guidance keeps its literals too, so
// two records can share `path::Symbol` or `stopped:usage` by having
// copied the same schema rather than by touching the same contract. The
// anchor intersection suppresses one such literal by name
// (scan.anchorPlaceholder); a literal intersection cannot, because there
// are ~50 of them and the set changes whenever the template does. Reading
// them off the bound schema keeps the suppression correct by
// construction — this is the same rule the anchor facet applies, stated
// as data rather than as a constant.
func TemplateLiterals() map[string]bool {
	out := map[string]bool{}
	for _, lit := range Backticked(current().TemplateSource) {
		out[lit] = true
	}
	return out
}

// Backticked reads the single-backtick spans out of markdown source.
// Fenced blocks are not special here: a literal inside a ``` fence is
// still a literal the author wrote, and the template's fences are
// exactly where its example contracts live.
func Backticked(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		for {
			i := strings.IndexByte(line, '`')
			if i < 0 {
				break
			}
			rest := line[i+1:]
			j := strings.IndexByte(rest, '`')
			if j < 0 {
				break
			}
			if lit := strings.TrimSpace(rest[:j]); lit != "" {
				out = append(out, lit)
			}
			line = rest[j+1:]
		}
	}
	return out
}
