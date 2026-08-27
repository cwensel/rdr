package model

// The sidecar: what TEMPLATE.md cannot say about itself.
//
// Everything the template states, the binary reads out of the template
// (parse.go). Where a fact can be carried as a bracket marker beside the
// section it describes, it is carried there. What is left is three
// things with no home in the document at all — the element-kind map,
// which is a reader fact; the observed vocabulary tiers, which exist
// precisely because the template never listed them; and the status
// lifecycle, which the template discusses only in English prose.
//
// This reader REFUSES what it does not recognise. A parser that skips an
// unknown key turns a typo into a missing fact, and a missing fact reads
// as absent when it was only misspelled — which is the failure the whole
// package is built to avoid.

import (
	"fmt"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/toml"
)

// Sidecar is a parsed rdr-template.toml.
type Sidecar struct {
	Version         int
	Description     string
	ElementSections map[string]string
	// Observed maps a vocabulary's field name onto the corpus-only
	// values it accepts. An entry present with an empty list is a
	// finding — see the file's own comment — not an omission.
	Observed map[string][]string
	Terminal []string
	Parked   []string
	// Source is the path it was read from, so an error names the file
	// that disagreed.
	Source string
}

// ParseSidecar reads the sidecar, refusing anything outside its shape.
func ParseSidecar(src, path string) (*Sidecar, error) {
	tables, err := toml.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: %v)", path, err)
	}

	sc := &Sidecar{
		ElementSections: map[string]string{},
		Observed:        map[string][]string{},
		Source:          path,
	}
	for _, t := range tables {
		switch {
		case t.Name == "template":
			sc.Version = t.Int("version")
			sc.Description = t.Str("description")
		case t.Name == "elements":
			for _, k := range t.Keys() {
				v := t.Str(k)
				if v == "" {
					return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: [elements] %s names no section)", path, k)
				}
				sc.ElementSections[k] = v
			}
		case strings.HasPrefix(t.Name, "observed."):
			field := t.Str("field")
			if field == "" {
				return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: [%s] declares no field)", path, t.Name)
			}
			if _, dup := sc.Observed[field]; dup {
				return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: two observed tiers for %q)", path, field)
			}
			// An absent `values` and an empty one are different: the
			// empty list is a declaration that the tier is empty.
			if !hasKey(t, "values") {
				return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: [%s] declares no values; write values = [] to state the tier is empty)", path, t.Name)
			}
			sc.Observed[field] = t.List("values")
		case t.Name == "lifecycle":
			sc.Terminal, sc.Parked = t.List("terminal"), t.List("parked")
			if len(sc.Terminal) == 0 {
				return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: [lifecycle] declares no terminal statuses)", path)
			}
		default:
			return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: unknown table [%s])", path, t.Name)
		}
	}

	if sc.Version != 1 {
		return nil, fmt.Errorf("stopped:unsupported-template-sidecar (%s: version %d; this binary reads version 1)", path, sc.Version)
	}
	if len(sc.ElementSections) == 0 {
		return nil, fmt.Errorf("stopped:malformed-template-sidecar (%s: no [elements] table)", path)
	}
	return sc, nil
}

// hasKey reports whether the table wrote the key at all, which is not the
// same as its value being empty.
func hasKey(t toml.Table, key string) bool {
	for _, k := range t.Keys() {
		if k == key {
			return true
		}
	}
	return false
}

// CheckAgainst reports a sidecar entry that names a section the template
// does not have.
//
// This is the same-commit rule pointed at this file: the sidecar and the
// template ship together, and a name that no longer resolves means one of
// them moved without the other. It is a refusal rather than a warning
// because a silently-missed lookup would make a kind read as keying
// nothing, which is a real answer to a question nobody asked.
func (sc *Sidecar) CheckAgainst(sections []Section) error {
	have := map[string]bool{}
	for _, s := range sections {
		have[s.Name] = true
	}
	for kind, section := range sc.ElementSections {
		if !have[section] {
			return fmt.Errorf("stopped:template-sidecar-skew (%s: [elements] %s names section %q, which TEMPLATE.md does not have)",
				sc.Source, kind, section)
		}
	}
	return nil
}
