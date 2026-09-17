// Package edge is the typed relation model: the closed set of edge kinds
// an RDR can carry, and the reference grammars that recover a target out
// of the prose the corpus actually writes.
//
// An RDR's relations to other records, to code and to artifacts are
// expressed in a dozen syntaxes, none of them machine-checked: Metadata
// fields (`Predecessors`, `Overrides`, `Cluster`), status qualifiers
// (`Final [joint decision → …]`, `Demoted [→ …]`), Evidence citations
// (`cli/0055 A5`, `cli/0055 §Normative Contracts`, `path::Symbol`,
// `{SPIKE_DIR}/…`), the Transient contract marker, Cross-Cutting
// ownership prose, and thousands of bare `cli/NNNN` mentions with no type
// at all. Typing them is what turns "who cites this contract?" from an
// LLM reading two files into a lookup.
//
// TWO RULES GOVERN THE GRAMMARS HERE.
//
// A REFERENCE FORM IS MAPPED OR IT IS A WARNING. Every syntax the corpus
// writes maps to a Kind. A reference-shaped string in an edge-bearing
// field that no grammar claims is recorded in the document's warnings
// channel, never dropped — the same zero-silent-drop contract the scanner
// holds itself to, because an edge silently missing is a query that
// answers "no relation" when the relation is right there in the file.
//
// MENTIONS ARE A KIND, NOT A FALLBACK. A bare `cli/NNNN` in prose is real
// — it is how most of the corpus cross-references — but it carries no
// stated relation, so it is its own weak kind. Typed queries filter it
// out; an impact query that wants everything includes it. Folding
// mentions into the typed kinds would make every typed answer wrong by
// thousands of edges.
package edge

import (
	"regexp"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// Kind is the closed enum of typed relations. Its string form is the
// `kind` field of an emitted edge.
type Kind string

const (
	// Predecessor is a `- **Predecessors**:` entry: a record whose design
	// this one builds on, and whose implementation must land first.
	Predecessor Kind = "predecessor"
	// Overrides is an `- **Overrides**:` entry: a contract of another
	// record this one narrows, retires or replaces.
	Overrides Kind = "overrides"
	// MovedTo is the `Demoted [→ <target>]` qualifier: the issue or record
	// the work was refiled as when it turned out not to be RDR-shaped.
	MovedTo Kind = "moved-to"
	// Cluster is a `- **Cluster**:` entry: a sibling record the 7.1
	// cross-RDR gate reconciles this one against.
	Cluster Kind = "cluster"
	// PeerEvidence is a citation in a `Method: Peer RDR` assumption's
	// Evidence: the peer element this record's claim rests on.
	PeerEvidence Kind = "peer-evidence"
	// JointDecisionHome is the `Final [joint decision → <home §anchor>: …]`
	// qualifier: the single normative home a hoisted decision was moved
	// to, and which this record waits on.
	JointDecisionHome Kind = "joint-decision-home"
	// Reverify is the `Draft [revised from Final …; re-verify A2,A4]`
	// qualifier: a self-edge to this record's own assumptions, naming
	// what the cluster gate reopened.
	Reverify Kind = "reverify"
	// TransientDeletedBy is the Normative Contracts `Transient — scheduled
	// deletion by <sibling>, <phase>` marker: the record that retires this
	// contract.
	TransientDeletedBy Kind = "transient-deleted-by"
	// SurfaceOf is the Normative Contracts `Surface — of Cn; …` marker: a
	// self-edge to the contract this fence only enforces, so the split
	// signal counts the root and not the fence.
	SurfaceOf Kind = "surface-of"
	// CrossCuttingOwner is a Cross-Cutting Concerns citation: the peer
	// record that owns the project-wide policy this one conforms to.
	CrossCuttingOwner Kind = "cross-cutting-owner"
	// SourceAnchor is a `path::Symbol` code anchor.
	SourceAnchor Kind = "source-anchor"
	// Artifact is a path into the RDR's own artifact or spike directory.
	Artifact Kind = "artifact"
	// Issue is a tracker reference: a kata id, a `#NNN`, an `_issues/NNNN`.
	Issue Kind = "issue"
	// RFD is a reference to an RFD by number or path.
	RFD Kind = "rfd"
	// JDR is a reference to a joint decision registry, by entry
	// (`JDR cli/0001 §DX-13`) or to the document (`JDR cli/0001`).
	JDR Kind = "jdr"
	// Mentions is the weak kind: a bare record reference in prose, with no
	// stated relation. Kept separate so typed queries are not polluted.
	Mentions Kind = "mentions"
)

// Kinds lists every kind in the order projections report them: the
// record-to-record relations first, then the out-of-corpus targets, with
// the weak kind last.
var Kinds = []Kind{
	Predecessor, Overrides, MovedTo, Cluster, PeerEvidence,
	JointDecisionHome, Reverify, TransientDeletedBy, SurfaceOf, CrossCuttingOwner,
	SourceAnchor, Artifact, Issue, RFD, JDR, Mentions,
}

// Typed reports whether the kind states a relation. Mentions does not:
// it records that one record named another, nothing more.
func (k Kind) Typed() bool { return k != Mentions }

// TargetClass says what a kind's target names, so a consumer knows how to
// read `to` without a table of its own.
type TargetClass string

const (
	// TargetElement is an element or document ID (`0055`, `0055:A3`,
	// `cli/0055:§normative-contracts`) inside a records dir.
	TargetElement TargetClass = "element"
	// TargetSymbol is a `path::Symbol` code anchor.
	TargetSymbol TargetClass = "symbol"
	// TargetPath is a filesystem path — an artifact, a spike output.
	TargetPath TargetClass = "path"
	// TargetRegistry is a JDR entry or document (`jdr:cli/0001:§dx-13`).
	// Its own class, not TargetElement, because a registry lives in its
	// own numbering: a JDR 0001 and a record cli/0001 are different
	// documents, and merging them is the defect the `jdr:` prefix exists
	// to prevent. Not TargetExternal either — unlike an RFD number, a
	// registry entry is resolvable once the JDR root is bound.
	TargetRegistry TargetClass = "registry"
	// TargetExternal is an identifier outside both the records dir and
	// the repo: a tracker id, an RFD number.
	TargetExternal TargetClass = "external"
)

// Class returns what a kind's target names.
func (k Kind) Class() TargetClass {
	switch k {
	case SourceAnchor:
		return TargetSymbol
	case Artifact:
		return TargetPath
	case Issue, RFD:
		return TargetExternal
	case JDR:
		return TargetRegistry
	}
	return TargetElement
}

// --- record and element references --------------------------------------

// Ref is one reference recovered from prose: the record it names and,
// when the citation reaches inside the record, the element.
type Ref struct {
	// Project is the records-dir qualifier written in the reference
	// (`cli/0055` → `cli`), or "" when the author wrote a bare number.
	Project string
	// Record is the four-digit number.
	Record string
	// Kind is the element class the citation reached for, or "" when the
	// reference names the whole document.
	Kind ident.Kind
	// Key is the element key (`3` for `A3`, `normative-contracts` for
	// `§Normative Contracts`), or "" for a document reference.
	Key string
	// Slug is the filename slug when the author wrote one
	// (`0003-frame-checksum-placement`), for the resolver to check
	// against the file it finds.
	Slug string
	// Raw is the matched text, for the edge's evidence field.
	Raw string
	// Start and End are byte offsets of Raw within the string it was
	// found in, so a caller can rewrite or exclude the span.
	Start, End int
}

// ID renders the reference as an element ID, or as a bare record number
// when the citation names the whole document.
func (r Ref) ID() string {
	if r.Kind == "" {
		if r.Project != "" {
			return r.Project + "/" + r.Record
		}
		return r.Record
	}
	return ident.ID{Project: r.Project, Record: r.Record, Kind: r.Kind, Key: r.Key}.String()
}

// Document is the reference with its element part dropped: the record ID
// a resolver looks up.
func (r Ref) Document() string {
	r.Kind, r.Key = "", ""
	return r.ID()
}

// recordRef matches every way the corpus names a record, and captures the
// element citation that may follow it.
//
// The record half is one of `cli/0055`, `RDR cli/0055`, `RDR 0055`,
// `R-0055`, `0055-some-slug` or, inside an edge-bearing field where a
// bare number is unambiguous, `0055`. The element half is optional and
// covers the citation shapes authors write: `§Section Name`,
// `§A5` / `A5` / `CA-5` / `C2` / `D-4` (a labelled element), and
// `REQ-13`, the implementation-time requirement id minted from a
// contract.
//
// The separator before the element half is a space, a `§`, or a COLON.
// The colon is the canonical ID form — `0055:C4`, `cli/0055:A3` — which
// is what the citation rule asks authors to write, and which the corpus
// did not have a way to express before element IDs existed. Reading it
// here is what makes a cite-don't-restate reference resolvable rather
// than merely well-intentioned: without it `0055:C4` parses as a bare
// reference to the whole of 0055 and the element half is silently lost.
//
// EVERY ALTERNATIVE REQUIRES A MARKER — an `RDR`/`R-` prefix, a `cli/`
// records-dir prefix, or a filename slug. A bare four-digit number is
// never a record here, because in prose four digits is a year, an RFC
// number, a line number or a count at least as often: the corpus writes
// `RFC 6962`, `2000 SMOs × 1000 elements`, `drift_walker.go:1306`. An
// edge to record `6962` from an RFC citation is a false relation a query
// cannot tell from a true one, and false edges are worse than absent
// ones. Bare numbers are read only by FindRefs(s, true), for the three
// Metadata fields whose whole value is a record list.
var recordRef = regexp.MustCompile(
	`(?:` +
		`\b(?:RDR|R)[ -](?:([a-z][a-z0-9_-]*)/)?(\d{4})()` + // `RDR 0055`, `R-0055`, `RDR cli/0055`
		`|\b([a-z][a-z0-9_-]*)/(\d{4})()` + // `cli/0055`
		`|\b()(\d{4})(-[a-z][a-z0-9-]*)` + // `0055-some-slug`; see hyphenContinuation
		`)` +
		`(?:\s*(?::|§\s*)?\s*` + // an optional separator: the ID colon or a section mark
		`(?:(§)\s*` + sectionName + // §Section Name
		`|(CA-|A|C|D-|D|S|F|RT|ALT|BR|G-|REQ-)(\d+[a-z]?)\b` + // an element label
		`|(D|G)-([a-z0-9]+(?:-[a-z0-9]+)*)\b` + // a slug-keyed element: D-identity, G-scope
		`|(MVV)\b)` + // the one keyed-by-nothing element
		`|:([A-Z]{1,3}-\d+[a-z]?)\b` + // a clause label in the colon form only; see clauseColonOnly
		`)?`,
)

// clauseColonOnly is the rule for a contract clause label — `L-3`,
// `NC-5`, `REQ-12a` — in a citation: it is an element reference ONLY in
// the canonical colon form, `cli/0112:L-3`, and is the document in every
// spaced spelling.
//
// The colon form is the author's exact clause id, the one TEMPLATE.md
// prescribes and ident's grammar parses, so it resolves against the
// target's minted clauses and a miss is the dangling reference that any
// exact id gets. Its precedence mirrors ident.Parse: the decision and
// gate grammars (`D-6`, `G-scope`) and the ordinal kinds (`F1`, `S1`)
// are tried first, so `:F-1` is a clause while `:F1` is a failure mode,
// and `:G-a` stays the document because the gate namespace is closed.
// `:CA-5` keeps its legacy reading as an assumption.
//
// The spaced forms — `cli/0112 L-3`, `cli/0119 REQ-89`, `§Normative
// Contracts L-3` — are the corpus's prose and are NOT promoted: they keep
// the document or section edge they have today, so no record is asked
// to rewrite a citation that was never wrong. The clause alternative is
// therefore anchored on the colon and nothing else.
func clauseColonOnly(s string, m []int, recordEnd int) bool {
	start := m[2*14]
	return start == recordEnd+1 && s[recordEnd] == ':'
}

// sectionName bounds a `§Section Name` citation. A section citation is a
// heading fragment, not a sentence: the corpus writes `§Identity stack §1:
// ElementIDs are randomly minted…` and `§ Table B line 187 classifies…`,
// where only the first words name the section and the rest is the
// author's prose about it. Running to the end of the clause would slug
// the whole sentence and produce a section ID that resolves against
// nothing.
//
// So the name is a quoted string when the author quoted it — the
// unambiguous form — and otherwise at most six words, stopping at any
// punctuation that ends a citation.
//
// The bound reads the CITATION GRAMMAR; it does not soften resolution.
// Resolution is exact: a citation that lands on no section of the target
// is reported, whether the author under-specified it (`§Semantic`, where
// the target has five such headings) or this bound clipped it. The
// parser never guesses which section was meant — a dangling reference is
// record data to correct, not parser tolerance to add.
//
// THE WORD SEPARATOR IS A RUN, NOT A CHARACTER. A heading is written
// `Semantic / Per-op — precondition replay`: the slash carries spaces
// around it and the dash is an em dash. A single-character `[ /]` class
// ends the name at `Semantic`, which is a DIFFERENT citation from the one
// the author wrote — and one that is genuinely ambiguous where the whole
// one is not. So the separator is the run of space, slash and dash the
// corpus writes between heading words. It reads more of the citation; it
// still reads only the citation, because the six-word ceiling and the
// stop-punctuation set are unchanged.
const sectionSep = `(?:\s*[/\x{2014}\x{2013}]\s*|\s)`

const sectionName = `(?:"([^"\n]{1,80})"|` + "`" + `([^` + "`" + `\n]{1,80})` + "`" +
	`|([A-Za-z][A-Za-z0-9-]*(?:` + sectionSep + `[A-Za-z0-9][A-Za-z0-9-]*){0,5}))`

// hyphenContinuation reports whether the match at off is a `NNNN-slug`
// that is really the TAIL of a longer hyphenated token — `a5-0118-clause-
// spans` inside `{EVIDENCE_DIR}research/a5-0118-clause-spans.md`.
//
// The slug alternative is the one grammar with no marker of its own: it
// says "four digits followed by a lowercase slug is a record", and a
// filename in the RDR's own evidence tree is exactly that shape one
// segment in. `\b` does not stop it, because a hyphen is a non-word byte
// and the number therefore opens a word wherever it sits.
//
// A record filename never has a segment before the number — that is what
// makes the shape a record reference at all. So a preceding hyphen whose
// own left neighbour is alphanumeric says the digits are mid-token, and
// the match is declined. The edge it would mint is a FALSE relation with
// a slug that cannot resolve: worse than an absent one, because a
// consumer chases it as a broken pointer in a record that is not broken.
func hyphenContinuation(s string, off int) bool {
	return off >= 2 && s[off-1] == '-' && identByte(s[off-2])
}

func identByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// bareNumber matches a record number written alone, for the fields whose
// whole value is a record list.
//
// A number followed by `-DD-DD` is excluded: it is the year of an ISO
// date, and record-list fields are full of them — `0035-… (Final
// 2026-05-19)`, `the drift half (68805e21, 2026-06-03)`. Reading the
// year as a record mints an edge to `2026` in seven corpus records.
var bareNumber = regexp.MustCompile(`\b(\d{4})(-[a-z][a-z0-9-]*)?\b`)

// isoDateYear matches the year position of an ISO date, so the bare-number
// sweep can decline it.
var isoDateYear = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// hasFourDigits reports whether s holds four consecutive ASCII digits —
// the cheapest necessary condition for any record reference.
func hasFourDigits(s string) bool {
	run := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			if run++; run == 4 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

// FindRefs recovers every record reference in a string. Bare four-digit
// numbers are read only when bare is true — which is correct inside
// `Predecessors`, `Overrides` and `Cluster`, whose values are record
// lists, and wrong in free prose, where four digits is as likely a year.
func FindRefs(s string, bare bool) []Ref {
	if !hasFourDigits(s) {
		// Every grammar here needs a four-digit run; most lines have none,
		// and the regexp's per-position backtracking is the scan's cost.
		return nil
	}
	var out []Ref
	claimed := make([]bool, len(s)+1)
	// A FOREIGN-TIER reference is claimed FIRST, so the record grammar
	// never sees it. `JDR 0001 §JD-18` and `RFD 0004 §3c` are four digits
	// and an anchor — a record reference by shape — so without this the
	// joint-decision home minted an edge to record 0001 or 0004, which
	// resolves TRUE wherever that record exists, against a document that
	// is not the one cited. A registry and an RFD live in their own
	// namespaces and are read by FindJDRRefs and FindRFDRefs; here they
	// are only spans to keep out of the record grammar.
	for _, re := range foreignTierRefs {
		for _, m := range re.FindAllStringIndex(s, -1) {
			for i := m[0]; i < m[1]; i++ {
				claimed[i] = true
			}
		}
	}
	for _, m := range recordRef.FindAllStringSubmatchIndex(s, -1) {
		if claimed[m[0]] {
			continue
		}
		// The markerless slug alternative fires inside a filename that
		// merely contains a record-shaped segment; decline it there.
		if m[16] >= 0 && hyphenContinuation(s, m[16]) {
			continue
		}
		r := Ref{Start: m[0], End: m[1], Raw: s[m[0]:m[1]]}
		recordEnd := 0
		for _, base := range []int{1, 4, 7} {
			if n := group(s, m, base+1); n != "" {
				r.Project, r.Record = group(s, m, base), n
				if slug := group(s, m, base+2); slug != "" {
					r.Slug = n + slug
				}
				recordEnd = m[2*(base+2)+1]
				break
			}
		}
		switch {
		case group(s, m, 10) == "§":
			name := group(s, m, 11) + group(s, m, 12) + group(s, m, 13)
			r.Kind, r.Key = ident.Section, ident.Slug(trimSectionTail(name))
		case group(s, m, 14) == "REQ-" && clauseColonOnly(s, m, recordEnd):
			r.Kind, r.Key = ident.Clause, group(s, m, 14)+group(s, m, 15)
		case group(s, m, 14) != "":
			r.Kind, r.Key = elementKind(group(s, m, 14)), group(s, m, 15)
		case group(s, m, 16) != "":
			r.Kind, r.Key = slugKind(group(s, m, 16), group(s, m, 17)), group(s, m, 17)
		case group(s, m, 18) != "":
			r.Kind = ident.MVV
		case group(s, m, 19) != "":
			r.Kind, r.Key = ident.Clause, group(s, m, 19)
		}
		if r.Kind == ident.Section && r.Key == "" {
			// `§` with nothing readable after it names the document.
			r.Kind = ""
		}
		// A kind that takes a key and got none names nothing inside the
		// record, so it falls back to a document reference. MVV is the
		// exception: it is one per record and carries no key by design.
		if r.Kind != "" && r.Kind != ident.Section && r.Kind != ident.MVV && r.Key == "" {
			r.Kind = ""
		}
		for i := m[0]; i < m[1]; i++ {
			claimed[i] = true
		}
		out = append(out, r)
	}
	if !bare {
		return out
	}
	for _, m := range bareNumber.FindAllStringSubmatchIndex(s, -1) {
		if claimed[m[0]] || isoDateYear.MatchString(s[m[0]:]) {
			continue
		}
		r := Ref{Record: s[m[2]:m[3]], Raw: s[m[0]:m[1]], Start: m[0], End: m[1]}
		if m[4] >= 0 {
			r.Slug = s[m[0]:m[1]]
		}
		out = append(out, r)
	}
	sortRefs(out)
	return out
}

// elementKind maps a citation's label token onto the ID grammar's kind.
//
// `CA-5` is the legacy spelling of `A5`, so it resolves as an assumption.
//
// `REQ-13` deliberately maps to NOTHING, and the citation is kept as a
// document reference. A REQ number is not an element of the record: the
// implementation prompt's Phase 0 mints one per testable clause into a
// per-run `req-list.md` artifact, over every section of the record, and
// the numbering belongs to that run rather than to the record. Reading
// `cli/0119 REQ-89` as `cli/0119:C89` would assert an element the record
// does not have and report it unresolved — 27 such false findings over
// the corpus, each one a real citation the projector mislabelled. The
// record is still the target; only the clause is out of this grammar's
// reach. That holds for the SPACED form; the colon form `cli/0113:REQ-12a`
// is the author's exact clause id and does resolve (clauseColonOnly).
// slugKind maps the slug-keyed citation forms — `D-identity`, `G-scope` —
// onto their kind. `D-` is unconditional: a decision's key IS the label's
// slug, so any slug is a decision key the target may or may not have, and
// a miss is the dangling reference resolution exists to report.
//
// `G-` IS NOT. The gate namespace is CLOSED: `ident.Gate` keys are the
// five Finalization Gate sub-sections (`model.GateItems`), minted from
// headings the template writes, and nothing else can ever be one. But
// `G-<slug>` is also how records name their OWN guards — a table of
// `G-a`…`G-j` conditions, a `G-faithful` mode — an author namespace with
// no relation to the gate.
//
// Reading those as gate items asserts an element the target cannot have
// under any spelling and reports it unresolved forever: a false finding
// on a citation that is correct, pointing a reader at a record with
// nothing to fix. So a `G-` key outside the closed set names no element
// in this grammar, and the citation targets the DOCUMENT — the same
// answer, for the same reason, that `REQ-N` already gets. The record is
// still the target; only the author's own item is out of reach.
func slugKind(tok, key string) ident.Kind {
	if tok == "G" && !model.IsGateItemKey(key) {
		return ""
	}
	return ident.Kind(tok)
}

func elementKind(tok string) ident.Kind {
	switch tok {
	case "CA-", "A":
		return ident.Assumption
	case "C":
		return ident.Contract
	case "D-", "D":
		return ident.Decision
	case "S":
		return ident.Scenario
	case "F":
		return ident.Failure
	case "RT":
		return ident.RoundTrip
	case "ALT":
		return ident.Alternative
	case "BR":
		return ident.Rejected
	case "G-":
		return ident.Gate
	}
	return ""
}

// sectionTail matches the label list an author appends to a section
// citation: `§Normative Contracts L-4`, `§Normative Contracts I-3/I-4`,
// `§Critical Assumptions A3/A6`. The section is the heading; what follows
// names items INSIDE it, in the record's own per-section labelling
// (`L-4`, `O-2`, `F-5`) rather than in any grammar this package knows.
//
// Left in, the labels slug into the section key and the citation resolves
// against nothing — 14 corpus citations to sections that plainly exist.
// Trimmed, the edge lands on the section, which is as deep as the
// reference is machine-readable. The finer target is the linking-
// enforcement issue's business, not a reason to report a real section
// missing.
var sectionTail = regexp.MustCompile(`\s+(?:[A-Z]+-?\d+[a-z]?)(?:\s*/\s*[A-Z]*-?\d+[a-z]?)*\s*$`)

// subLocator matches the WORD-FORM sub-locator authors append to a
// section citation — `§Identity stack point 6`, `§Technical Design item
// 5`, `§Approach step 2`. It is the same thing sectionTail trims in its
// label form (`L-4`), written out in words instead.
//
// It names an item INSIDE the section, in the record's own per-section
// numbering, which is below what any grammar here addresses: the section
// is the deepest target the citation is machine-readable to. Left in, the
// words slug into the key and the citation resolves against nothing,
// reporting a section that plainly exists as missing.
//
// The vocabulary is closed on purpose — `point`, `item`, `step`, `line`,
// `note`, `bullet`, `row`, `§` — because these are sub-locators and
// nothing else. Trimming any `<word> <number>` tail would eat the last
// word of a heading that ends in a number, and headings do (`Step 2:
// Layer assignment`, `Phase 1`). A closed list cannot make that mistake.
var subLocator = regexp.MustCompile(
	`(?i)\s+(?:point|item|step|line|note|bullet|row|§)\s*\d+[a-z]?\s*$`)

// trimSectionTail drops the label list and the word-form sub-locator from
// a section citation's name, repeatedly: the corpus writes both at once
// (`§Normative Contracts L-4 point 2`), and one pass would leave whichever
// came first.
func trimSectionTail(name string) string {
	name = strings.TrimSpace(name)
	for {
		trimmed := strings.TrimSpace(subLocator.ReplaceAllString(
			strings.TrimSpace(sectionTail.ReplaceAllString(name, "")), ""))
		if trimmed == name || trimmed == "" {
			return trimmed
		}
		name = trimmed
	}
}

func group(s string, m []int, n int) string {
	if 2*n+1 >= len(m) || m[2*n] < 0 {
		return ""
	}
	return strings.TrimSpace(s[m[2*n]:m[2*n+1]])
}

func sortRefs(refs []Ref) {
	for i := 1; i < len(refs); i++ {
		for j := i; j > 0 && refs[j].Start < refs[j-1].Start; j-- {
			refs[j], refs[j-1] = refs[j-1], refs[j]
		}
	}
}

// --- non-record references ----------------------------------------------

// SourceAnchorRe matches a `path::Symbol` code anchor — the form
// tooling-pass CHECK 5 requires of every Source Search evidence line,
// because a symbol greps and a line number rots. The path may be bare
// (`frame::Encode`, a package-qualified symbol) or a real path
// (`internal/cli/drift.go::walk`).
//
// A `path:LINE::Symbol` form, where an author pinned a line as well, is
// matched on its symbol half: the line number is exactly what CHECK 5
// says not to check.
var SourceAnchorRe = regexp.MustCompile(
	`\b([A-Za-z0-9_][A-Za-z0-9_./-]*?)(?::\d+)?::([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\b`)

// recordDoubleColon matches the `0092::A1` form — a record's element
// written with the path separator. It is a record citation an author
// spelled like an anchor, and reading it as code would emit a
// source-anchor edge to a symbol no repo has.
var recordDoubleColon = regexp.MustCompile(`^(?:[a-z][a-z0-9_-]*/)?\d{4}$`)

// IsSourceAnchor reports whether a matched `a::b` is really a code
// anchor. The path half of a genuine anchor is a package or a file, never
// a bare record number.
func IsSourceAnchor(path string) bool { return !recordDoubleColon.MatchString(path) }

// ArtifactRe matches a path into the RDR's own evidence tree, written
// with one of the flow's directory variables.
var ArtifactRe = regexp.MustCompile(`\{(SPIKE_DIR|ARTIFACT_DIR|EVIDENCE_DIR|RDR_PATH)\}/?([A-Za-z0-9_./-]*)`)

// IssueRe matches the tracker forms the corpus writes: an `_issues/NNNN`
// path, and a tracker id spelled with its tracker — `kata #71`, `kata
// ahg1`, and the same with the id itself backticked. Both id shapes the
// corpus uses are read: the numeric one the tracker started with, and the
// short alphanumeric one it mints now.
//
// A BARE `#N` IS NOT READ. It is the corpus's ordinary way of numbering
// prose, not of citing a tracker: of 1,615 `#N` occurrences, 635 are
// `principle #N` and the rest are items, steps and clauses. Reading them
// as issues produced edges to `#1` and `#5` from a dozen records — false
// relations to trackers that were never referenced. The tracker forms
// that name their tracker are unambiguous, and they are the ones read.
var IssueRe = regexp.MustCompile(
	`(?:\b_issues/(?:archive/)?([0-9A-Za-z][A-Za-z0-9_.-]*)` +
		`|\bkata\s+[#*` + "`" + `]*(#?[0-9a-z]{2,6})\b)`)

// JDRRe matches a joint decision registry: `JDR cli/0001 §DX-13`,
// `JDR 0001 §JD-18`, or the bare document `JDR cli/0001`. The optional
// project prefix mirrors the record grammar, and the `§` anchor names an
// entry.
//
// IT IS MATCHED BEFORE THE RECORD GRAMMAR, AND THAT ORDER IS THE POINT.
// `JDR 0001 §JD-18` is four digits followed by a `§` anchor, which is
// exactly what a record reference looks like — so before this existed the
// qualifier's home minted an edge to RECORD 0001. Where such a record
// exists the home resolved true against the wrong document, and the
// propose gate clears a lock on that edge. Claiming the span here is what
// keeps the record grammar from seeing it.
var JDRRe = regexp.MustCompile(
	`\bJDR\s+(?:([a-z][a-z0-9-]*)/)?(\d{3,4})(?:\s*§\s*([A-Za-z][A-Za-z0-9-]*))?`)

// JDRRef is one registry reference: the document, and the entry when the
// citation reaches inside it.
type JDRRef struct {
	// Project is the qualifier written in the reference (`cli`), or ""
	// when the author wrote a bare number.
	Project string
	// Registry is the four-digit number.
	Registry string
	// Entry is the entry id (`DX-13`, `JD-18`), or "" for a reference to
	// the document.
	Entry string
	// Start and End bound the span in the source string.
	Start, End int
	// Raw is the reference as written.
	Raw string
}

// ID renders the reference as a target, in the registry's own namespace.
// The `jdr:` prefix is what keeps a JDR 0001 from colliding with a record
// cli/0001 — two different documents that share a number.
func (r JDRRef) ID() string {
	id := "jdr:"
	if r.Project != "" {
		id += r.Project + "/"
	}
	id += r.Registry
	if r.Entry != "" {
		id += ":§" + strings.ToLower(r.Entry)
	}
	return id
}

// FindJDRRefs recovers every registry reference in a string.
func FindJDRRefs(s string) []JDRRef {
	var out []JDRRef
	for _, m := range JDRRe.FindAllStringSubmatchIndex(s, -1) {
		r := JDRRef{Start: m[0], End: m[1], Raw: s[m[0]:m[1]]}
		r.Project, r.Registry, r.Entry = group(s, m, 1), group(s, m, 2), group(s, m, 3)
		if r.Registry == "" {
			continue
		}
		out = append(out, r)
	}
	return out
}

// RFDRe matches an RFD by number (`RFD 0004`) or by path (`rfd/0004/…`).
var RFDRe = regexp.MustCompile(`\b(?:RFD\s+(\d{3,4})|rfd/(\d{3,4})(?:/([A-Za-z0-9_./-]*))?)`)

// RFDAnchorRe matches an RFD citation that reaches INSIDE the document:
// `RFD 0004 §3c` (a section), `RFD 0004 P-2` (a principle), `RFD 0004
// DX-13` (a legacy decision row), `RFD 0007 Decision 1` (the other legacy
// spelling). The corpus writes the decision rows WITHOUT a `§`, which is
// why the anchor alternation carries them bare.
//
// Only the number was read before, so the anchor half fell out as
// `edge:unmapped-reference` — twelve warnings on one record, six of them
// RFD section lists. Reading the anchor is what turns those into edges
// that can be checked.
var RFDAnchorRe = regexp.MustCompile(
	`\bRFD\s+(\d{3,4})\s+(?:§\s*([0-9]+[a-z]?)` +
		`|(P-\d+[a-z]?)` +
		`|(DX-\d+[a-z]?)` +
		`|Decision\s+(\d+[a-z]?))`)

// RFDRef is one RFD citation: the document, and the anchor when the
// citation reaches inside it.
type RFDRef struct {
	// Number is the four-digit RFD number.
	Number string
	// Anchor is the section number, principle or decision id,
	// lowercased (`3c`, `p-2`, `dx-13`, `decision-1`), or "" for a
	// reference to the document.
	Anchor string
	// Start and End bound the span in the source string.
	Start, End int
	// Raw is the reference as written.
	Raw string
}

// ID renders the reference as a target, in the RFD's own namespace —
// `rfd/0004`, `rfd/0004:§3c` — the namespace the `rfd` edge already used
// for the document.
func (r RFDRef) ID() string {
	id := "rfd/" + r.Number
	if r.Anchor != "" {
		id += ":§" + r.Anchor
	}
	return id
}

// rfdContinuation matches a bare `, §2a` following an RFD citation: the
// corpus writes SECTION LISTS — `RFD 0004 §1, §1a, §2, §2a` — where one
// `RFD NNNN` prefix governs several anchors. Reading only the first left
// the rest falling out as `edge:unmapped-reference`, which is most of
// what that warning was reporting on the live corpus.
var rfdContinuation = regexp.MustCompile(`^[,;]\s*§\s*([0-9]+[a-z]?|P-\d+[a-z]?|DX-\d+[a-z]?)`)

// FindRFDRefs recovers every RFD citation that names an anchor, including
// the continuations of a section list. The document-only form stays with
// RFDRe, which issueEdges already reads.
func FindRFDRefs(s string) []RFDRef {
	var out []RFDRef
	for _, m := range RFDAnchorRe.FindAllStringSubmatchIndex(s, -1) {
		r := RFDRef{Start: m[0], End: m[1], Raw: s[m[0]:m[1]]}
		r.Number = group(s, m, 1)
		switch {
		case group(s, m, 2) != "":
			r.Anchor = strings.ToLower(group(s, m, 2))
		case group(s, m, 3) != "":
			r.Anchor = strings.ToLower(group(s, m, 3))
		case group(s, m, 4) != "":
			r.Anchor = strings.ToLower(group(s, m, 4))
		case group(s, m, 5) != "":
			r.Anchor = "decision-" + strings.ToLower(group(s, m, 5))
		}
		if r.Number == "" || r.Anchor == "" {
			continue
		}
		out = append(out, r)
	}
	// A GOVERNING PREFIX: one `RFD NNNN` opens a value and every bare
	// `§x` / `DX-n` after it names an anchor of that RFD. The corpus
	// writes whole fields this way —
	//
	//   Related Issues: RFD 0004 (…) — locked: DX-5 (…), DX-1 (…);
	//   §1, §1a, §2, §2a Rejected (…), §3 chain key, §6 journeys
	//
	// and reading only the anchors glued to the prefix left the rest
	// falling out as `edge:unmapped-reference`, which is most of what
	// that warning reported on the live corpus.
	//
	// Scoped to ONE value and to the LAST RFD named, so a bare `§3` in
	// free prose mints nothing (there is no prefix to govern it) and a
	// value naming two RFDs attributes each anchor to the nearer one.
	// A bare `§x` is otherwise a section of the RECORD, so the prefix is
	// what makes it an RFD anchor rather than a guess.
	if len(out) == 0 && !RFDRe.MatchString(s) {
		return out
	}
	claimed := make([]bool, len(s)+1)
	for _, r := range out {
		for i := r.Start; i < r.End; i++ {
			claimed[i] = true
		}
	}
	var governing string
	var spans [][]int
	for _, m := range RFDRe.FindAllStringSubmatchIndex(s, -1) {
		spans = append(spans, m)
	}
	for _, m := range rfdBareAnchor.FindAllStringSubmatchIndex(s, -1) {
		if claimed[m[0]] {
			continue
		}
		governing = ""
		for _, sp := range spans {
			if sp[1] <= m[0] {
				if n := groupAt(s, sp, 1); n != "" {
					governing = n
				} else if n := groupAt(s, sp, 2); n != "" {
					governing = n
				}
			}
		}
		if governing == "" {
			continue
		}
		anchor := groupAt(s, m, 1) + groupAt(s, m, 2)
		if anchor == "" {
			continue
		}
		out = append(out, RFDRef{
			Number: governing,
			Anchor: strings.ToLower(anchor),
			Start:  m[0], End: m[1], Raw: s[m[0]:m[1]],
		})
	}
	return out
}

// foreignTierRefs are the grammars whose spans the RECORD grammar must
// not read: a reference to a document in another tier, carrying that
// tier's own marker.
//
// Only the MARKED forms are here, and that is the whole rule. `RFD 0004
// §3c` and `rfd/0004/README.md` say which document they mean, so
// claiming their spans costs the record grammar nothing it was right
// about. FindRFDRefs' governing-prefix arm is NOT here: it attributes a
// bare `§2a` to the last RFD a value named, which is a reading of that
// value's grammar and not a marker on the text — claiming those spans
// would silence record references that sit in the same field.
//
// RFDAnchorRe is listed beside RFDRe because it is the longer match:
// RFDRe alone claims `RFD 0004` and leaves `§3c` for the record grammar,
// which reads it as a section of the RECORD. The anchor half belongs to
// the RFD or to nothing.
var foreignTierRefs = []*regexp.Regexp{JDRRe, RFDAnchorRe, RFDRe}

// rfdBareAnchor matches an anchor with no `RFD NNNN` of its own: `§3c`,
// `§1a`, `DX-13`. Only meaningful under a governing prefix.
//
// `P-n` IS NOT HERE, and that is a finding rather than an omission. The
// corpus uses `P-n` for premortem points from a critic pass ("critic.md
// P-1…P-16") at least as often as for a principle, and under a governing
// prefix those were attributed to whatever RFD the value named earlier —
// 96 false unresolved edges on the live corpus, every one of them a
// terminal record reporting a dangling reference it never wrote. A
// principle is read only when it carries its own `RFD NNNN P-n`, which
// RFDAnchorRe handles. An ambiguous reference stays unresolved.
// The section arm REQUIRES the `§`. A bare digit under a governing
// prefix reads "RFD 0007 Follow-on 3" as §3 — a section that RFD does not
// have, reported as a dangling citation on a terminal record. The `§` is
// what distinguishes a section citation from a numbered noun, and the
// corpus always writes it.
var rfdBareAnchor = regexp.MustCompile(`§\s*([0-9]+[a-z]?)\b|\b(DX-\d+[a-z]?)\b`)

// groupAt is group() over an explicit match slice.
func groupAt(s string, m []int, n int) string {
	if 2*n+1 < len(m) && m[2*n] >= 0 {
		return s[m[2*n]:m[2*n+1]]
	}
	return ""
}

// referenceShaped matches a string that looks like a reference to
// something — the test for AC-1's unmapped-form warning. A token in an
// edge-bearing field that looks like a citation and matches no grammar is
// the thing the warning is for: a syntax the corpus grew and this package
// has not learned.
//
// It is deliberately narrow. A false warning on ordinary prose would
// bury the real ones, so it fires only on the two shapes that are
// citations and nothing else: an arrow to a target, and a `§` anchor.
var referenceShaped = regexp.MustCompile(`(?:→|-->)\s*\S|§\s*\S`)

// Unmapped reports the reference-shaped spans of s that no grammar
// claimed, given the spans that were. It is how an unknown syntax
// becomes a warning instead of a silence.
func Unmapped(s string, claimed [][2]int) []string {
	var out []string
	for _, loc := range referenceShaped.FindAllStringIndex(s, -1) {
		covered := false
		for _, c := range claimed {
			if loc[0] >= c[0] && loc[0] < c[1] {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		end := loc[0] + 60
		if end > len(s) {
			end = len(s)
		}
		out = append(out, strings.TrimSpace(s[loc[0]:end]))
	}
	return out
}
