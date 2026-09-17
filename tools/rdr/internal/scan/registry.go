package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
)

// Registry is a joint decision registry as the projector reads it: the
// entries a record may cite, the seam those entries bind, and the legacy
// anchors this registry took over.
//
// It is NOT a Document. A registry is a different document class with a
// different lifecycle — per-entry, append-only — and reading it through
// the record scanner would mint record ids for a numbering that is not
// the records dir's. Only what a citation must resolve against is read.
type Registry struct {
	// Project is the instance dir's project (`cli` in jdr/cli/0001-…),
	// or "" when the tree holds no project level.
	Project string
	// Number is the four-digit registry number.
	Number string
	// Path is the file read.
	Path string
	// Entries holds every citable anchor, lowercased: entry ids
	// (`dx-13`, `jd-5`, `d2`) and prose section slugs (`principles`).
	Entries map[string]bool
	// Inherits holds the legacy anchors this registry took over, from
	// `inherits:` frontmatter, lowercased. A citation that hits one is
	// resolved, not a finding: the alias is what lets a spelling the
	// migration did not reach keep working (jdr/README.md §Citation).
	Inherits map[string]bool
	// InheritsFrom holds the same anchors keyed by the document they were
	// taken FROM: `0004` -> {`dx1`, …}, read out of the `RFD NNNN` half
	// of the `inherits:` value.
	//
	// Scoping matters because the alias answers a citation that still
	// names the OLD home. `RFD 0004 DX-13` is a claim about RFD 0004, and
	// a registry that inherited DX-13 from some other RFD has not made
	// that claim true. Inherits alone would resolve it anyway — the id is
	// the same string — which is a guess wearing a map hit.
	InheritsFrom map[string]map[string]bool
	// Seam is the declared loci, from `seam:` frontmatter.
	Seam []string
	// Binds maps an entry id to the records that entry names. An entry
	// states them on a `**Binds:**` line; every bound record must be a
	// member, and one that is not is the `jdr:bound-off-seam` finding.
	Binds map[string][]string
}

// entryHeading matches an entry's own heading or bullet: `## D1 — …`,
// `## JD-5: …`, `- **JD-12** `decided` — …`. The id is what a record
// cites, so the grammar is the id and what may surround it, never the
// prose after.
var entryHeading = regexp.MustCompile(
	`(?m)^(?:#{2,4}\s+|[-*]\s+(?:\*\*)?)((?:DX|JD|D)-?\d+[a-z]?)\b`)

// registrySection matches a prose heading a record may cite by slug,
// `## Principles` → `principles`.
var registrySection = regexp.MustCompile(`(?m)^#{2,4}\s+(.+?)\s*$`)

// inheritsRange matches an `inherits:` range, `RFD 0004 DX-1..DX-18`, and
// the singular form. A range is expanded so every id it names resolves;
// the migration's whole point is that the ids did not move.
var inheritsRange = regexp.MustCompile(
	`((?:DX|JD|D)-?)(\d+)\s*(?:\.\.|–|—|-)\s*(?:(?:DX|JD|D)-?)?(\d+)`)

// LoadRegistries reads every registry under root. A root that is unset or
// unreadable yields nothing AND reports false, which is what keeps an
// unbound root reading as nothing-looked rather than as a corpus with no
// registries in it — the same distinction `consulted` carries for records.
func LoadRegistries(root string) ([]*Registry, bool) {
	if strings.TrimSpace(root) == "" {
		return nil, false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	var out []*Registry
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		base := filepath.Base(path)
		num := registryNumber(base)
		if num == "" {
			return nil // README.md and anything else unnumbered
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		reg := parseRegistry(string(body))
		reg.Number, reg.Path = num, path
		if parent := filepath.Base(filepath.Dir(path)); parent != filepath.Base(root) {
			reg.Project = parent
		}
		out = append(out, reg)
		return nil
	})
	return out, true
}

var registryFile = regexp.MustCompile(`^(\d{3,4})-`)

// bindsLine matches an entry's `**Binds:** 0113, 0120.` line — the
// records that entry binds.
var bindsLine = regexp.MustCompile(`(?i)\*{0,2}Binds:?\*{0,2}\s*([^\n]*)`)

// recordRefIn matches a record number in a Binds list, with or without a
// project prefix.
var recordRefIn = regexp.MustCompile(`\b(?:[a-z][a-z0-9-]*/)?(\d{4})\b`)

func registryNumber(base string) string {
	if m := registryFile.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return ""
}

// parseRegistry reads the citable surface out of a registry's text.
func parseRegistry(body string) *Registry {
	reg := &Registry{Entries: map[string]bool{}, Inherits: map[string]bool{},
		InheritsFrom: map[string]map[string]bool{}, Binds: map[string][]string{}}
	front, rest := splitFrontmatter(body)
	reg.Seam = frontmatterList(front, "seam")
	for _, raw := range frontmatterList(front, "inherits") {
		from := inheritedFrom(raw)
		for _, id := range expandInherits(raw) {
			reg.Inherits[id] = true
			if from == "" {
				continue
			}
			if reg.InheritsFrom[from] == nil {
				reg.InheritsFrom[from] = map[string]bool{}
			}
			reg.InheritsFrom[from][id] = true
		}
	}
	for _, m := range entryHeading.FindAllStringSubmatchIndex(rest, -1) {
		// Two spellings, on purpose: the key is normalized so `DX-13`,
		// `dx-13` and `DX13` are one anchor for lookup, and the WRITTEN
		// form is what a finding prints. A normalized id in output reads
		// as a typo of the citation it is reporting on.
		written := strings.ToLower(rest[m[2]:m[3]])
		id := normalizeEntry(written)
		reg.Entries[id] = true
		// The entry's own `Binds:` line, to the end of its block. An
		// entry is a bullet or a heading, so the block ends at the next
		// one — a bound record named after that belongs to its entry,
		// not this one.
		tail := rest[m[1]:]
		if nx := entryHeading.FindStringIndex(tail); nx != nil {
			tail = tail[:nx[0]]
		}
		if bm := bindsLine.FindStringSubmatch(tail); bm != nil {
			for _, r := range recordRefIn.FindAllString(bm[1], -1) {
				reg.Binds[written] = append(reg.Binds[written], strings.TrimPrefix(r, "cli/"))
			}
		}
	}
	for _, m := range registrySection.FindAllStringSubmatch(rest, -1) {
		// A heading that IS an entry is already recorded under its id;
		// recording its slug too would let `§d1-how-does-the-guard-seam`
		// resolve, which is a spelling no citation uses.
		if entryHeading.MatchString(m[0]) {
			continue
		}
		if s := ident.Slug(m[1]); s != "" {
			reg.Entries[s] = true
		}
	}
	return reg
}

// normalizeEntry lowercases an id and drops the hyphen the corpus writes
// inconsistently, so `DX-13`, `dx-13` and `DX13` are one anchor. The
// citation form the README fixes is `§DX-13`; the others are read, never
// written.
func normalizeEntry(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "-", ""))
}

// inheritsSource matches the `RFD NNNN` half of an `inherits:` value —
// the document these anchors were taken over FROM. Absent is legal: a
// registry may inherit from a home the frontmatter does not name, and
// that alias then answers no RFD-scoped citation, only a bare one.
var inheritsSource = regexp.MustCompile(`(?i)\bRFD\s+(\d{3,4})\b`)

// inheritedFrom reads the RFD number an `inherits:` value names, or ""
// when it names none.
func inheritedFrom(s string) string {
	if m := inheritsSource.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// InheritsAnchorFrom reports whether this registry took over `anchor`
// from RFD `num`. It is the scoped half of Has: a citation that still
// names the old home resolves here, and only against the registry that
// actually claimed that home's anchor.
func (r *Registry) InheritsAnchorFrom(num, anchor string) bool {
	return r.InheritsFrom[num][normalizeEntry(strings.TrimPrefix(anchor, "§"))]
}

// expandInherits turns `RFD 0004 DX-1..DX-18` into every id it names.
// The range is expanded rather than stored, so a lookup is a map hit and
// the alias table says exactly which anchors it covers.
func expandInherits(s string) []string {
	var out []string
	if m := inheritsRange.FindStringSubmatch(s); m != nil {
		lo, _ := strconv.Atoi(m[2])
		hi, _ := strconv.Atoi(m[3])
		if lo > 0 && hi >= lo && hi-lo < 512 {
			for i := lo; i <= hi; i++ {
				out = append(out, normalizeEntry(m[1]+strconv.Itoa(i)))
			}
			return out
		}
	}
	for _, m := range entryHeading.FindAllStringSubmatch("- "+s, -1) {
		out = append(out, normalizeEntry(m[1]))
	}
	if len(out) == 0 {
		if m := regexp.MustCompile(`((?:DX|JD|D)-?\d+[a-z]?)`).FindAllStringSubmatch(s, -1); m != nil {
			for _, x := range m {
				out = append(out, normalizeEntry(x[1]))
			}
		}
	}
	return out
}

// Key is how a citation addresses this registry: `cli/0001`, or the bare
// number when the tree has no project level.
func (r *Registry) Key() string {
	if r.Project != "" {
		return r.Project + "/" + r.Number
	}
	return r.Number
}

// Has reports whether an anchor resolves against this registry, by entry
// or by an inherited alias. An alias hit is not a finding.
func (r *Registry) Has(anchor string) bool {
	a := normalizeEntry(strings.TrimPrefix(anchor, "§"))
	return r.Entries[a] || r.Inherits[a]
}

// splitFrontmatter separates a leading `---` block from the body. A file
// with no frontmatter is all body, which is the common case for a
// registry drafted before the class existed.
func splitFrontmatter(body string) (front, rest string) {
	if !strings.HasPrefix(body, "---\n") {
		return "", body
	}
	if i := strings.Index(body[4:], "\n---"); i >= 0 {
		end := 4 + i + len("\n---")
		return body[4 : 4+i], strings.TrimPrefix(body[end:], "\n")
	}
	return "", body
}

// frontmatterList reads a key's value as a list, in either YAML spelling
// the template admits: an inline `key: a, b` or a block of `- ` items.
func frontmatterList(front, key string) []string {
	lines := strings.Split(front, "\n")
	var out []string
	for i, ln := range lines {
		name, val, ok := strings.Cut(ln, ":")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		if v := strings.TrimSpace(val); v != "" {
			for _, part := range strings.Split(v, ",") {
				if p := strings.TrimSpace(part); p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		for _, item := range lines[i+1:] {
			t := strings.TrimSpace(item)
			if !strings.HasPrefix(t, "- ") {
				break
			}
			if v := strings.TrimSpace(strings.TrimPrefix(t, "- ")); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}
