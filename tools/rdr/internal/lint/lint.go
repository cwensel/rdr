// Package lint is the conformance authority over a projected record:
// one pass, three severities, and a rule about which records each
// severity is allowed to speak about.
//
// The severities exist because the corpus is not uniform and never will
// be. Most records are terminal — by doctrine an RDR is never amended
// after it locks — so a check that fires on them produces noise no one
// is permitted to act on. A live record, by contrast, is going to be
// rewritten anyway at its next stage, so a migration hint delivered
// there costs nothing. And a small class of defects is about what a
// record CLAIMS rather than how it is shaped: a citation that names a
// target which does not exist is wrong on any record, of any age, and
// is the one thing worth blocking a lock over.
//
//	TierParse       always emitted, every record. The projector could
//	                not classify something. On a terminal record this is
//	                a projector bug: the file cannot have changed, so the
//	                scanner is what is wrong. Fix with a fixture.
//	TierConformance advisory, LIVE records only. "The current
//	                template carries Load-Bearing
//	                Decisions." Phrased as a migration hint for the stage
//	                already rewriting the file. Never blocks. Terminal
//	                records never generate it.
//	TierResolution  BLOCKING at lock, every record. Every typed edge
//	                resolves; Peer-RDR Evidence names an element, not just
//	                a record; contracts are labelled on records written
//	                after the labelling rule landed. This tier judges what
//	                a record EMITS, never what its targets look like.
//
// The asymmetry between conformance and resolution is the whole design.
// Conformance is about a record's own shape, which its age excuses.
// Resolution is about a record's outbound claims, which nothing excuses:
// a dangling reference in a frozen record is a data error, and the
// sanctioned repair is a minimal pointer correction — the reference
// text, nothing else. That is why a terminal record still gets a
// resolution finding, delivered as a fix-pointer advisory naming the
// line range to open.
package lint

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// Tier is a finding's severity and, with it, the rule about which
// records the check may speak about.
type Tier string

const (
	// TierParse is a projector-owned warning: something the scanner could
	// not classify. Always emitted, on every record.
	TierParse Tier = "parse"
	// TierConformance is advisory, live records only.
	TierConformance Tier = "conformance"
	// TierResolution is blocking at lock, on every record.
	TierResolution Tier = "resolution"
)

// Finding is one lint result.
type Finding struct {
	Tier Tier `json:"tier"`
	// Code is the stable check id, `<area>:<rule>`, so a consumer can
	// filter without matching prose.
	Code string `json:"code"`
	// Blocking is whether this finding, on this record, prohibits a lock.
	// It is a property of the finding and not of its tier alone: a
	// resolution failure on a terminal record is reported as a fix
	// pointer, not as a block, because the record is not the one locking.
	Blocking bool `json:"blocking"`
	// Element is the element or edge the finding is about, or "" when it
	// is about the record.
	Element   string `json:"element,omitempty"`
	Message   string `json:"message"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	// Fix is the repair, when there is a mechanical one to name.
	Fix string `json:"fix,omitempty"`
	// Patch is the repair as bytes, when the repair is mechanical AND
	// exact. It is the machine-applicable half of Fix: Fix says what to
	// do in prose for a reader, Patch says it in lines for a script.
	//
	// It is present only on the rules whose repair is computed rather
	// than judged. A finding that needs a
	// decision — which of two records owns a relation, which element a
	// bare peer citation meant, which section a colliding label belongs
	// to — carries Fix and no Patch, because a guessed patch applied in
	// bulk is exactly the corpus-wide damage the projector's never-guess
	// rule exists to prevent.
	Patch *Patch `json:"patch,omitempty"`
}

// Patch is a machine-applicable repair: the lines it replaces and the
// text to write in their place.
//
// The grammar is deliberately three ops over a line range and nothing
// else. Every migration this tool names — re-levelling a heading,
// renaming a legacy section, writing the id the projector already
// derives, spelling a citation in the colon form — is expressible as
// whole-line surgery, and a grammar that could express more would invite
// a rule whose repair is not actually mechanical.
//
// An applier walks a record's patches BOTTOM-UP by LineStart, so earlier
// ranges keep their line numbers as later ones change length.
//
// Two properties an applier may rely on, and both are tested:
//
//   - Ranges within one record do not overlap. Where several findings
//     repair the same line — every citation written on it — they SHARE
//     one patch rather than proposing one each, so an applier must
//     deduplicate by pointer identity before applying. Computing a
//     patch per citation against the original line would make the
//     second overwrite the first, silently.
//   - Applying the full set is a FIXPOINT: a second pass over the
//     result proposes nothing. Conformance is reached in one pass; there
//     is no iterate-until-clean step and nothing to bridge between runs.
type Patch struct {
	// LineStart and LineEnd are 1-based and inclusive, over the record as
	// the projector read it.
	LineStart int `json:"line_start"`
	LineEnd   int `json:"line_end"`
	// Op is `replace` (Text stands in for lines LineStart..LineEnd),
	// `prepend` (Text goes immediately above LineStart, which is
	// unchanged) or `insert` (Text goes immediately below LineEnd).
	Op string `json:"op"`
	// Text is the replacement or inserted lines, newline-separated and
	// without a trailing newline.
	Text string `json:"text"`
}

// The three patch ops.
const (
	OpReplace = "replace"
	OpPrepend = "prepend"
	OpInsert  = "insert"
)

// Report is one record's lint result.
type Report struct {
	Schema string `json:"schema"`
	Record string `json:"record"`
	Path   string `json:"path"`
	// Status is the record's Status label, and Terminal whether that
	// status is one after which the record is never amended.
	Status   string    `json:"status,omitempty"`
	Terminal bool      `json:"terminal"`
	Findings []Finding `json:"findings"`
	// Verdict is PASS when no finding blocks, BLOCK when one does.
	Verdict string `json:"verdict"`
}

// LabelRuleDate is the day the contract-labelling rule landed. A record
// DATED on or after it is expected to label its contracts; one dated
// before it is grandfathered and gets advice at most.
//
// The boundary is the record's own `Date` field rather than a template
// fingerprint, because the fingerprint would beg the question: the
// signal that would place a record in a "labels its contracts" era is
// the presence of labelled contracts, so a new record that labelled
// nothing would fingerprint as legacy and escape the very check the rule
// exists to apply. `Date` is written by Seed on every record, is already
// read by the metadata scan, and says when the record entered the flow —
// which is exactly the question the grandfathering rule asks.
const LabelRuleDate = "2026-08-24"

// Options tunes a lint pass.
type Options struct {
	// Locking says the record is at a lock gate. It does not change which
	// findings are produced — only whether resolution findings on a
	// non-terminal record are reported as blocking. A record is linted
	// mid-flow far more often than at lock, and a blocking verdict there
	// would be read as a stop when it is a to-do.
	Locking bool
	// Now overrides the clock for the label-rule boundary in tests.
	Now string
	// Corpus is every record in the dir, when the caller has it, so a
	// finding can read a peer: nil leaves peer-dependent checks unrun.
	Corpus []*scan.Document
}

// Run lints one projected record. The document must already have been
// resolved against its records dir — an edge with no `resolved` verdict
// was never checked, and lint reports nothing about it rather than
// guessing.
func Run(d *scan.Document, opts Options) Report {
	status := model.ParseStatus(d.MetadataValue("Status"))
	terminal := isTerminal(status.Label)

	r := Report{
		Schema:   scan.SchemaVersion,
		Record:   d.Record,
		Path:     d.Path,
		Status:   status.Label,
		Terminal: terminal,
		Findings: []Finding{},
	}

	// Conformance speaks on EVERY record, terminal ones included. A
	// terminal record's CONTENT is never amended, but its STRUCTURE may
	// be brought to the current template by tooling with ids and content
	// bytes preserved (README §Identifiers) — so the advice is
	// actionable, and withholding it only hid what a migration costs.
	// It still blocks nothing: `terminal` continues to govern DELIVERY
	// in resolutionFindings, which is where the distinction belongs.
	r.Findings = append(r.Findings, parseFindings(d)...)
	r.Findings = append(r.Findings, conformanceFindings(d, opts)...)
	r.Findings = append(r.Findings, templateFindings(d)...)
	r.Findings = append(r.Findings, evidenceBudgetFindings(d)...)
	r.Findings = append(r.Findings, proseVocabularyFindings(d)...)
	r.Findings = append(r.Findings, scaffoldRowFindings(d)...)
	r.Findings = append(r.Findings, resolutionFindings(d, terminal, opts)...)

	sort.SliceStable(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if a.LineStart != b.LineStart {
			return a.LineStart < b.LineStart
		}
		return a.Code < b.Code
	})

	r.Verdict = "PASS"
	for _, f := range r.Findings {
		if f.Blocking {
			r.Verdict = "BLOCK"
			break
		}
	}
	return r
}

func isTerminal(label string) bool {
	for _, t := range model.TerminalStatuses() {
		if strings.EqualFold(t, label) {
			return true
		}
	}
	return false
}

// parseFindings republishes the projector's own warnings channel as tier
// 1. They are not re-derived here: the scanner already recorded
// everything it could not classify, and lint's contribution is to say
// what a warning MEANS on this record. On a terminal record it means the
// scanner is wrong, because the file is not going to change.
func parseFindings(d *scan.Document) []Finding {
	out := make([]Finding, 0, len(d.Warnings))
	for _, w := range d.Warnings {
		out = append(out, Finding{
			Tier:      TierParse,
			Code:      "parse:" + w.Code,
			Message:   w.Message,
			LineStart: w.LineStart,
			LineEnd:   w.LineEnd,
		})
	}
	return out
}

// conformanceFindings are the migration hints for a live record: what
// the current template asks for that this record does not carry. They
// are advisory by construction — every one carries Blocking false — and
// they are phrased for the stage that is already rewriting the file.
//
// Evidence-field length drift is deliberately absent. A 34-line Evidence
// field is prose drift inside a conforming structure, which is the
// tooling pass's advisory C9, not a structural non-conformance.
func conformanceFindings(d *scan.Document, opts Options) []Finding {
	var out []Finding

	// The contract-labelling advice. Under the rule date this is the
	// fix-forward path: lint names the labels to write, and the stage
	// that is rewriting the file writes them in-pass.
	if n := unlabelled(d); n > 0 && !subjectToLabelRule(d, opts) {
		sec, start, end := contractSpan(d)
		out = append(out, Finding{
			Tier:      TierConformance,
			Code:      "label:contracts",
			Element:   sec,
			Message:   labelMessage(d, n),
			LineStart: start,
			LineEnd:   end,
			Fix:       "label contracts C1..Cn in document order, one `**Cn**` on the line above each ```normative fence",
		})
	}

	// Sections the current template carries that this record
	// predates. Only Required ones: a Conditional section is omitted by
	// design, and saying so on every record would drown the real hints.
	for _, s := range missingRequired(d) {
		out = append(out, Finding{
			Tier:      TierConformance,
			Code:      "template:missing-section",
			Element:   s.id,
			Message:   "the current template carries a Required section this record does not: " + s.name,
			LineStart: s.line,
			LineEnd:   s.line,
			Fix:       "add the section at the stage that next rewrites this record",
		})
	}

	return out
}

// subjectToLabelRule reports whether the record was written after the
// labelling rule landed, and so owes labels as a blocking rule rather
// than as advice.
func subjectToLabelRule(d *scan.Document, opts Options) bool {
	raw := strings.TrimSpace(d.MetadataValue("Date"))
	if raw == "" {
		// A record with no Date cannot be placed against the boundary.
		// It is grandfathered: the alternative is blocking a lock on a
		// missing metadata field, which is a different finding entirely
		// and is not this rule's to make.
		return false
	}
	boundary := opts.Now
	if boundary == "" {
		boundary = LabelRuleDate
	}
	dt, err := time.Parse("2006-01-02", firstDate(raw))
	if err != nil {
		return false
	}
	b, err := time.Parse("2006-01-02", boundary)
	if err != nil {
		return false
	}
	return !dt.Before(b)
}

// firstDate takes the leading YYYY-MM-DD off a Date value, which the
// corpus writes with trailing prose ("2026-08-24 (revised …)").
func firstDate(raw string) string {
	raw = strings.TrimLeft(raw, "`*_ ")
	if len(raw) < 10 {
		return raw
	}
	return raw[:10]
}

// unlabelled counts the record's contracts carrying a derived id — the
// ones with no `**Cn**` label above the fence.
func unlabelled(d *scan.Document) int {
	n := 0
	for _, e := range d.Elements {
		if e.Kind == ident.Contract && e.Derived {
			n++
		}
	}
	return n
}

// contractSpan locates where the labelling advice points: the record's
// Normative Contracts section when it has one, else the first contract
// itself, because a record may write its contracts under its own
// headings and there is no section to name.
func contractSpan(d *scan.Document) (id string, start, end int) {
	for _, n := range d.Outline {
		if n.Canonical == "Normative Contracts" {
			return n.ID, n.LineStart, n.LineEnd
		}
	}
	for _, e := range d.Elements {
		if e.Kind == ident.Contract {
			return e.ID, e.LineStart, e.LineEnd
		}
	}
	return "", 0, 0
}

func labelMessage(d *scan.Document, n int) string {
	total := d.Counts.Elements[ident.Contract]
	plural := "contracts"
	if n == 1 {
		plural = "contract"
	}
	return itoa(n) + " of " + itoa(total) + " " + plural +
		" carry a derived id; labelling them makes them citable by a stable id that survives a heading rewrite or a move"
}

// missing is a Required section the current template carries and the
// record does not.
type missing struct {
	id, name string
	line     int
}

// missingRequired compares the record's outline against the CURRENT
// template, not the one that produced it. That is the point of the advice: the
// record is behind, and the hint says by how much.
//
// Two whole classes are excluded, because a section absent BY DESIGN is
// not a section the record is behind on:
//
// A section whose parent is absent is not separately missing. Naming
// `Contradiction Check` on a record that has no Finalization Gate body
// reports the same fact five times and buries the one hint that matters.
//
// A subsection of a gate-pointer section is not missing at all. From
// at lock, the Finalization Gate body is REPLACED with a one-line
// pointer to gate.md — the responses moved out of the record on purpose.
// Advising a locked record to restore the five gate subsections would be
// advising it to undo the current process.
func missingRequired(d *scan.Document) []missing {
	present := map[string]bool{}
	for _, n := range d.Outline {
		if n.Canonical != "" {
			present[n.Canonical] = true
		}
	}
	// The gate's responses move to gate.md at lock, so its sub-sections
	// are not missing when absent. The template names them — GateItems is
	// read from its own markers — and their shared parent is the section
	// the pointer replaces.
	pointer := map[string]bool{}
	for _, g := range model.GateItems() {
		if s, ok := model.Template().SectionByName(g.Section); ok && s.Parent != "" {
			pointer[s.Parent] = true
		}
	}
	var out []missing
	for _, s := range model.Template().Sections {
		if s.Class != model.Required || present[s.Name] {
			continue
		}
		// A scaffold section names a slot, not a section every record
		// owes: `Alternative 1: [Name]` and `Step 1: [Title]` are
		// Conditional already, but the guard is cheap and keeps a future
		// scaffold from becoming noise.
		if strings.Contains(s.Name, "[") {
			continue
		}
		if s.Parent != "" && (pointer[s.Parent] || !present[s.Parent]) {
			continue
		}
		out = append(out, missing{name: s.Name, line: 1})
	}
	return out
}

// resolutionFindings judge what the record claims: every typed edge
// resolves, and a Peer-RDR citation names an element rather than a bare
// record.
//
// These are the findings that apply to terminal records too, because
// they are not about the record's shape. The DELIVERY differs: on a
// terminal record the finding is a fix pointer — the line range to open
// and the instruction that the repair is the reference text and nothing
// else — and it does not block, because a frozen record is not the one
// locking. On a live record at a lock gate it blocks.
func resolutionFindings(d *scan.Document, terminal bool, opts Options) []Finding {
	var out []Finding
	blocks := opts.Locking && !terminal

	for _, e := range d.Edges {
		if !e.Kind.Typed() || e.Resolved == nil || *e.Resolved {
			continue
		}
		f := Finding{
			Tier:      TierResolution,
			Code:      "edge:unresolved",
			Blocking:  blocks,
			Element:   e.From,
			Message:   string(e.Kind) + " edge names " + e.To + ", which does not resolve",
			LineStart: e.Line,
			LineEnd:   e.LineEnd,
			Fix:       "correct the reference text in this range — the pointer only, no prose or structure changes",
		}
		if terminal {
			f.Code = "edge:unresolved-terminal"
			f.Message += "; the record is " + strings.ToLower(d.MetadataValue("Status")) + ", so this is a data error, not drift"
		}
		out = append(out, f)
	}

	// Ownership must be a DAG. A record that overrides a peer while the
	// peer overrides it back names no authority; the same for mutual
	// predecessors and a moved-to that comes home. The full cycle walk is
	// `index --cycles`; lint sees the pair its own record is half of.
	for _, e := range d.Edges {
		if !ownershipKind(e.Kind) {
			continue
		}
		t := refRecord(e.To)
		if t == "" || t == d.Record {
			continue
		}
		for _, p := range opts.Corpus {
			if p.Record != t {
				continue
			}
			for _, back := range p.Edges {
				if back.Kind == e.Kind && refRecord(back.To) == d.Record {
					out = append(out, Finding{
						Tier:      TierResolution,
						Code:      "ownership:mutual",
						Blocking:  blocks,
						Element:   e.From,
						Message:   string(e.Kind) + " names " + t + ", and " + t + " " + string(e.Kind) + " this record back — one of the two is wrong",
						LineStart: e.Line,
						LineEnd:   e.LineEnd,
						Fix:       "decide which record holds the relation and remove the other side's line",
					})
					break
				}
			}
		}
	}

	// A Peer-RDR assumption rests its claim on a peer. Citing the peer
	// RECORD says which file to read; citing the peer ELEMENT says what
	// in it the claim rests on — and it is the only form that survives
	// the peer being reorganised, which is the failure this rule exists
	// to stop.
	for _, e := range d.Edges {
		if e.Kind != edge.PeerEvidence || !ident.IsRecord(e.To) {
			continue
		}
		out = append(out, Finding{
			Tier:      TierResolution,
			Code:      "peer-evidence:no-element",
			Blocking:  blocks,
			Element:   e.From,
			Message:   "Peer-RDR Evidence cites the record " + e.To + " but no element in it",
			LineStart: e.Line,
			LineEnd:   e.LineEnd,
			Fix:       "cite the element the claim rests on (" + e.To + ":A3, " + e.To + ":C4); `rdr inspect " + e.To + "` lists them",
		})
	}

	out = append(out, reentryNearMiss(d, blocks)...)

	// Contracts on a record written after the rule landed.
	if n := unlabelled(d); n > 0 && subjectToLabelRule(d, opts) {
		sec, start, end := contractSpan(d)
		out = append(out, Finding{
			Tier:      TierResolution,
			Code:      "label:contracts-required",
			Blocking:  blocks,
			Element:   sec,
			Message:   labelMessage(d, n) + "; this record postdates the labelling rule",
			LineStart: start,
			LineEnd:   end,
			Fix:       "label contracts C1..Cn in document order, one `**Cn**` on the line above each ```normative fence",
		})
	}

	return out
}

// reentryNearMiss reports a bracketed Status qualifier that begins
// `revised from` and fails RevisedFromGrammar. Re-entry routing reads the
// FORM, never the prose: a near-miss degrades to a free-text note, the
// re-entry fact reads false, and the flow silently skips the scoped
// re-verification the qualifier exists to trigger — which is why this
// sits in the resolution tier with the record's other consumed claims
// rather than with the advisory shape hints. A live demote pass proved
// the failure mode: one extra word before the semicolon blinded every
// re-entry rule, and lint said nothing.
func reentryNearMiss(d *scan.Document, blocks bool) []Finding {
	s := model.ParseStatus(d.MetadataValue("Status"))
	if s.QualifierForm != model.QualifierBracketed ||
		!strings.HasPrefix(s.Qualifier, "revised from") {
		return nil
	}
	line := 0
	for _, f := range d.Metadata {
		if f.Canonical == "Status" {
			line = f.LineStart
			break
		}
	}
	return []Finding{{
		Tier:      TierResolution,
		Code:      "status:reentry-near-miss",
		Blocking:  blocks,
		Element:   "Status",
		Message:   "the Status qualifier begins `revised from` but does not parse as the re-entry form, so routing reads it as a free-text note and no re-entry rule fires",
		LineStart: line,
		LineEnd:   line,
		Fix:       "spell it `revised from Final YYYY-MM-DD; re-verify <IDs> — <reason>`: the semicolon immediately after the date (at most a short stage token between), `re-verify none` for an empty set",
	}}
}

// itoa is strconv.Itoa without the import, kept local because the only
// numbers this package formats are small counts.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func ownershipKind(k edge.Kind) bool {
	return k == edge.Predecessor || k == edge.Overrides || k == edge.MovedTo
}

var recordRef = regexp.MustCompile(`\b(\d{4})\b`)

// refRecord reads the record number out of an edge target.
func refRecord(s string) string {
	if m := recordRef.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}
