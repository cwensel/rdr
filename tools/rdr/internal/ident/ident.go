// Package ident is the globally-qualified element ID grammar.
//
// An RDR number is almost a global identifier and an assumption label is
// almost a local one; nothing else below the document level has an
// identity. This package gives every element one, in a single grammar:
//
//	[<project>/]<NNNN>:<KIND><key>
//
//	0055:A3            assumption               as written
//	0055:C4            normative contract       as written once labelled, derived until then
//	0055:D-identity    load-bearing decision    keyed by decision class
//	0055:RT1           round-trip invariant
//	0055:ALT2          alternative              keyed by the scaffold ordinal
//	0055:BR3           briefly-rejected item
//	0055:S5            validation scenario      keyed by the list number
//	0055:MVV           minimum viable validation (one per record)
//	0055:F2            failure mode
//	0055:G-contradiction  gate response         keyed by the gate item
//	0055:L-3           clause inside a contract fence, keyed by its label
//	0055:§normative-contracts  outline section  keyed by canonical slug
//	cli/0055:C4        the same, qualified across records dirs
//
// The project prefix is written only when a reference crosses records
// dirs; inside one dir it is omitted, exactly as `cli/NNNN` is today.
//
// An ID is AS WRITTEN when the author labelled the element (A-numbers;
// C-numbers once the labelling rule lands; scenario and alternative
// numbers; decision classes) and DERIVED when the projector had to mint
// it from the element's ordinal. A derived ID is flagged so the corpus
// can report its labelling backlog, and every element carries a short
// content hash so a consumer can tell "same ID, same content" from "same
// ID, moved under it". Ordinal-derived IDs are exactly as stable as
// author-written ones on a terminal record, because the file never
// changes again; on a live record they hold until a sibling element is
// inserted before them, which is the reason labels exist.
package ident

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Kind is an element class. Its string form is the ID's kind token.
type Kind string

const (
	Assumption  Kind = "A"
	Contract    Kind = "C"
	Decision    Kind = "D"
	RoundTrip   Kind = "RT"
	Alternative Kind = "ALT"
	Rejected    Kind = "BR"
	Scenario    Kind = "S"
	MVV         Kind = "MVV"
	Failure     Kind = "F"
	Gate        Kind = "G"
	JointCheck  Kind = "JC"
	Section     Kind = "§"
	// Clause is a labelled clause INSIDE a normative fence — `L-3`,
	// `I-4`, `REQ-12` — the grain the corpus cites below the contract.
	// Its kind token is the author's own letters, so the ID is written
	// `0055:L-3` and the kind name appears only in JSON.
	Clause Kind = "clause"
)

// Kinds lists every kind in the order projections and counts report them.
var Kinds = []Kind{
	Assumption, Contract, Decision, RoundTrip, Alternative, Rejected,
	Scenario, MVV, Failure, Gate, Clause, Section,
}

// Keyed reports whether the kind takes a slug key (`D-identity`,
// `G-scope`, `§approach`) rather than an ordinal or nothing.
func (k Kind) Keyed() bool {
	return k == Decision || k == Gate || k == Section
}

// ID is one parsed element identifier.
type ID struct {
	// Project is the records-dir qualifier, or "" inside one dir.
	Project string `json:"project,omitempty"`
	// Record is the four-digit RDR number.
	Record string `json:"record"`
	// Kind is the element class.
	Kind Kind `json:"kind"`
	// Key is the ordinal (`3`, or `4b` where an author split a label),
	// the class slug (`identity`), or "" for MVV.
	Key string `json:"key,omitempty"`
}

// grammar is the whole ID form. Kind tokens are tried longest-first where
// one is a prefix of another (ALT before A, RT before nothing, MVV alone).
var grammar = regexp.MustCompile(
	`^(?:([A-Za-z0-9][A-Za-z0-9_.-]*)/)?(\d{4}):(?:` +
		`(ALT|BR|RT|JC|S|F|A|C)(\d+[a-z]?)` + // ordinal kinds; A4b is written
		`|(D|G)-([a-z0-9]+(?:-[a-z0-9]+)*)` + // slug kinds
		`|(§)([a-z0-9]+(?:-[a-z0-9]+)*)` + // outline sections
		`|(MVV)` +
		`|([A-Z]{1,3}-\d+[a-z]?)` + // a contract clause label, as written; D-/G- are taken above
		`)$`)

// ErrSyntax is returned for a string that is not an element ID.
var ErrSyntax = errors.New("not an element id")

// Parse reads an ID in its canonical form. It does not accept the legacy
// prose forms (`cli/NNNN A5`, `§Normative Contracts`); those are edge
// extraction's business and are reported there as mentions.
func Parse(s string) (ID, error) {
	m := grammar.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return ID{}, fmt.Errorf("%w: %q", ErrSyntax, s)
	}
	id := ID{Project: m[1], Record: m[2]}
	switch {
	case m[3] != "":
		id.Kind, id.Key = Kind(m[3]), m[4]
	case m[5] != "":
		id.Kind, id.Key = Kind(m[5]), m[6]
	case m[7] != "":
		id.Kind, id.Key = Section, m[8]
	case m[10] != "":
		id.Kind, id.Key = Clause, m[10]
	default:
		id.Kind = MVV
	}
	return id, nil
}

// IsID reports whether s parses as an element ID.
func IsID(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// String writes the ID in canonical form, with its project prefix when it
// has one.
func (id ID) String() string {
	var b strings.Builder
	if id.Project != "" {
		b.WriteString(id.Project)
		b.WriteByte('/')
	}
	b.WriteString(id.Record)
	b.WriteByte(':')
	if id.Kind != Clause {
		b.WriteString(string(id.Kind))
	}
	if id.Kind.Keyed() && id.Kind != Section {
		b.WriteByte('-')
	}
	b.WriteString(id.Key)
	return b.String()
}

// Local is the ID without its project prefix — the form used inside one
// records dir.
func (id ID) Local() string {
	id.Project = ""
	return id.String()
}

// Qualified is the ID with the given project prefix — the form used when
// a reference crosses records dirs.
func (id ID) Qualified(project string) string {
	id.Project = project
	return id.String()
}

// New builds an ID from its parts. Key is normalised for slug kinds so an
// author-written class label (`Wire / byte format`) and its slug key
// (`wire-byte-format`) meet in one place.
func New(project, record string, kind Kind, key string) ID {
	if kind.Keyed() {
		key = Slug(key)
	}
	return ID{Project: project, Record: record, Kind: kind, Key: key}
}

// Slug reduces heading or label text to an ID key: lower-case ASCII
// letters and digits, runs of anything else collapsed to one hyphen,
// markdown emphasis and code marks dropped. `Round-Trip / Inverse
// Invariants` becomes `round-trip-inverse-invariants`.
func Slug(text string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Non-ASCII letters are kept: a slug must round-trip a
			// heading a human wrote, and dropping them would fold two
			// distinct headings together.
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// Hash is the short content hash carried by every element: the first
// eight hex digits of SHA-256 over the element's lines with surrounding
// whitespace trimmed and blank lines dropped. Reflowing or re-indenting
// an element therefore leaves its hash alone; changing a word does not.
func Hash(lines []string) string {
	h := sha256.New()
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		h.Write([]byte(t))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))[:8]
}

// RecordNumber matches the four-digit record number at the head of a
// record filename (`0055-some-slug.md`) or a records-dir basename.
var RecordNumber = regexp.MustCompile(`^(\d{4})(?:[-.]|$)`)

// RecordOf reads the record number off a filename's base, or "" if the
// base does not start with one.
func RecordOf(base string) string {
	if m := RecordNumber.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return ""
}

// documentID is a whole-record target: `0055` or `cli/0055`, with no
// element after it.
var documentID = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9_.-]*/)?\d{4}$`)

// IsRecord reports whether s names a record and nothing inside it. It is
// what separates "read this file" from "this claim rests on that
// element": a citation that resolves to a whole 4,000-line record has
// named a document, not a reason.
func IsRecord(s string) bool {
	return documentID.MatchString(strings.TrimSpace(s))
}
