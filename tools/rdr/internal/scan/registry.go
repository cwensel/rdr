package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

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
	//
	// The value is the anchor THIS registry answers the citation with,
	// which is the source id itself for an identity alias and the
	// declared target for a rename.
	InheritsFrom map[string]map[string]string
	// InheritsSections holds the section aliases, keyed by source
	// document: `0007` -> {`decision-1`: `d1`}. They are held apart from
	// InheritsFrom because a section source is matched by the resolver's
	// whole-word prefix rule rather than by equality — a citation spelled
	// `§Decision 1 — Classified by…` names the heading `Decision 1`, and
	// only the target's own headings can say which one it means. An id
	// alias is an exact string and stays one.
	InheritsSections map[string]map[string]string
	// InheritsDecls are the `inherits:` items as written, in order, for a
	// lint that judges the declaration rather than resolving through it.
	InheritsDecls []InheritsDecl
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

// InheritsDecl is one `inherits:` item, parsed.
//
// THE GRAMMAR IS `<source> [-> <target>]`. A target left off means the
// anchor did not move, which is what the DX range has always meant and
// stays the common case: `RFD 0004 DX-1..DX-18` and `RFD 0007 §4a..§4e`
// both declare that a citation resolves under the id it already names.
// An arrow declares a RENAME the registry performed on the way in —
// `RFD 0007 Decision 1..4 -> §D1..§D4`, where the class's entry form is
// `## D1 — …` and the old home wrote `## Decision 1 — …`.
//
// A heading label is structure and migrates; the body under it does not.
// That asymmetry is why the rename is DECLARED rather than inferred: a
// scanner taught that "Decision N" means "DN" would be guessing at a
// convention, and the next registry to hoist a differently-named fork
// would inherit the guess. The declaration costs one line and says
// exactly what happened.
type InheritsDecl struct {
	// Raw is the item as written, for a finding to quote.
	Raw string
	// From is the RFD number the anchors were taken from, or "" when the
	// item names none.
	From string
	// Sources are the anchors the OLD home spelled, normalized: entry ids
	// (`dx13`) or section slugs (`decision-1`).
	Sources []string
	// Targets are the anchors THIS registry answers with, positionally
	// paired with Sources. An identity declaration repeats the source.
	Targets []string
	// Section reports whether the sources are section anchors, matched by
	// the whole-word prefix rule rather than by equality.
	Section bool
	// Err names why the item could not be read as a mapping — an unequal
	// range pairing, or a side that named no anchor at all. It is the
	// `jdr:inherits-unanchored` message's own text.
	Err string
}

// inheritsArrow splits an `inherits:` item on the mapping arrow. Only the
// ASCII `->` and the unicode `→` are arrows: an em-dash is prose ("DX-1
// — the band"), and reading one as a mapping would silently halve a
// declaration.
var inheritsArrow = regexp.MustCompile(`\s*(?:->|→)\s*`)

// inheritsSectionRange matches a section range on either side of the
// arrow: `§4a..§4e`, `§Decision 1..3`, `Decision 1..4`.
//
// TWO AXES VARY, NEVER BOTH. `Decision 1..4` counts the ORDINAL under a
// fixed label; `§4a..§4e` counts the LETTER under a fixed number. A
// grammar that let both move at once would have to invent an ordering
// across them, and no citation spells one.
//
// The label before the ordinal is carried onto every member, so
// `§Decision 1..3` names `decision-1`, `decision-2`, `decision-3` and not
// three bare numbers.
var inheritsSectionRange = regexp.MustCompile(
	`(?i)^§?\s*(.*?)(\d+)([a-z]?)\s*(?:\.\.|–|—)\s*§?\s*(.*?)(\d+)?([a-z]?)$`)

// entryPrefix matches a range label that is an entry-id prefix rather
// than a section label: the `D` of `§D1..§D4`, the `JD` of `JD-1..JD-5`.
var entryPrefix = regexp.MustCompile(`(?i)^(?:DX|JD|D)-?$`)

// inheritsEntryID matches a single entry id on either side of the arrow.
var inheritsEntryID = regexp.MustCompile(`(?i)^§?\s*((?:DX|JD|D)-?\d+[a-z]?)$`)

// parseInherits reads one `inherits:` item into a declaration.
//
// It is deliberately total: an item it cannot read yields a decl carrying
// Err rather than nothing, because a declaration the projector silently
// dropped is a registry that believes it aliased an anchor and a corpus
// whose citations quietly dangle. `jdr:inherits-unanchored` is that Err
// surfaced.
func parseInherits(raw string) InheritsDecl {
	d := InheritsDecl{Raw: strings.TrimSpace(raw), From: inheritedFrom(raw)}
	// The source half keeps the `RFD NNNN` prefix off: it names the
	// document, not an anchor, and From already holds it.
	body := strings.TrimSpace(inheritsSource.ReplaceAllString(d.Raw, ""))
	lhs, rhs := body, ""
	if parts := inheritsArrow.Split(body, 2); len(parts) == 2 {
		lhs, rhs = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	d.Sources, d.Section = inheritsAnchors(lhs)
	if len(d.Sources) == 0 {
		d.Err = "names no anchor on the left of the mapping"
		return d
	}
	if rhs == "" {
		// Identity: the anchor did not move. Targets repeat Sources so
		// every consumer reads one shape.
		d.Targets = append([]string(nil), d.Sources...)
		return d
	}
	targets, _ := inheritsAnchors(rhs)
	if len(targets) == 0 {
		d.Err = "names no anchor on the right of the mapping"
		return d
	}
	if len(targets) != len(d.Sources) {
		d.Err = "maps " + strconv.Itoa(len(d.Sources)) + " anchors onto " + strconv.Itoa(len(targets)) +
			"; a range mapping pairs positionally and must be equal length"
		return d
	}
	d.Targets = targets
	return d
}

// inheritsAnchors reads one side of a mapping into its anchors, and
// reports whether they are sections.
//
// A side is a range or a single anchor, and an entry id is tried before a
// section: `D1` is an id in the class's own entry grammar, and reading it
// as the section slug `d1` would make the alias miss the entry it names.
func inheritsAnchors(side string) (out []string, section bool) {
	if side == "" {
		return nil, false
	}
	if m := inheritsRange.FindStringSubmatch(side); m != nil && looksLikeEntryRange(side) {
		lo, _ := strconv.Atoi(m[2])
		hi, _ := strconv.Atoi(m[3])
		if lo > 0 && hi >= lo && hi-lo < 512 {
			for i := lo; i <= hi; i++ {
				out = append(out, normalizeEntry(m[1]+strconv.Itoa(i)))
			}
			return out, false
		}
	}
	if m := inheritsEntryID.FindStringSubmatch(side); m != nil {
		return []string{normalizeEntry(m[1])}, false
	}
	if m := inheritsSectionRange.FindStringSubmatch(side); m != nil {
		label, loN, loL := strings.TrimSpace(m[1]), m[2], m[3]
		hiN, hiL := m[5], m[6]
		switch {
		case loL == "" && hiL == "" && hiN != "":
			// The ordinal varies under a fixed label: `Decision 1..4`,
			// and `§D1..§D4`.
			lo, _ := strconv.Atoi(loN)
			hi, _ := strconv.Atoi(hiN)
			if lo > 0 && hi >= lo && hi-lo < 512 {
				// A LABEL THAT IS AN ENTRY PREFIX MAKES THIS AN ID RANGE.
				// `§D1..§D4` wears the section spelling — the citation
				// grammar puts `§` on both — but `D1` is an id in the
				// class's own entry grammar, and slugging it to `d-1`
				// would miss the `d1` the entry table is keyed under.
				id := entryPrefix.MatchString(label)
				for i := lo; i <= hi; i++ {
					if id {
						out = append(out, normalizeEntry(label+strconv.Itoa(i)))
						continue
					}
					out = append(out, sectionKey(label+" "+strconv.Itoa(i)))
				}
				return out, !id
			}
		case loL != "" && hiL != "" && (hiN == "" || hiN == loN):
			// The letter varies under a fixed number: `§4a..§4e`.
			if hiL[0] >= loL[0] && hiL[0]-loL[0] < 26 {
				for c := loL[0]; c <= hiL[0]; c++ {
					out = append(out, sectionKey(label+loN+string(c)))
				}
				return out, true
			}
		}
	}
	// A single section anchor: `§4b`, `§Facts of record`, `§Decision 1`.
	if k := sectionKey(strings.TrimPrefix(strings.TrimSpace(side), "§")); k != "" {
		return []string{k}, true
	}
	return nil, false
}

// looksLikeEntryRange reports whether a range's endpoints are entry ids
// rather than sections. `DX-1..DX-18` is an id range; `4a..4e` and
// `Decision 1..4` are not, and inheritsRange's own grammar would match
// the `D` of `Decision` as an id prefix if nothing asked.
func looksLikeEntryRange(side string) bool {
	return regexp.MustCompile(`(?i)^§?\s*(?:DX|JD|D)-?\d`).MatchString(strings.TrimSpace(side))
}

// sectionKey normalizes a section anchor the way the resolver's section
// keys are: a slug, so `§Facts of record` and `§facts-of-record` are one
// anchor. A bare section NUMBER keeps its digits, which is the form
// rfd.go records (`4b`).
func sectionKey(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "§"))
	if s == "" {
		return ""
	}
	return ident.Slug(s)
}

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
		InheritsFrom: map[string]map[string]string{},
		InheritsSections: map[string]map[string]string{}, Binds: map[string][]string{}}
	front, rest := splitFrontmatter(body)
	reg.Seam = frontmatterList(front, "seam")
	for _, raw := range frontmatterList(front, "inherits") {
		d := parseInherits(raw)
		reg.InheritsDecls = append(reg.InheritsDecls, d)
		if d.Err != "" {
			continue // the lint reports it; resolving through it would guess
		}
		into := reg.InheritsFrom
		if d.Section {
			into = reg.InheritsSections
		}
		for i, src := range d.Sources {
			// Inherits is the UNSCOPED table, and it holds what a citation
			// SPELLS, not what the registry answers with — a bare `DX-13`
			// names the old id, and the unscoped arm has no document to
			// check the rename against.
			reg.Inherits[src] = true
			if d.From == "" {
				continue
			}
			if into[d.From] == nil {
				into[d.From] = map[string]string{}
			}
			into[d.From][src] = d.Targets[i]
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
	_, ok := r.inheritedTarget(num, anchor)
	return ok
}

// inheritedTarget resolves an old-home anchor to the anchor this registry
// answers it with, and reports whether the registry claimed it at all.
//
// The id table is tried first and by equality, then the section table by
// the whole-word prefix rule — an id is an exact string, and a section
// anchor is a citation of a heading whose tail the author may have
// carried into the reference. Trying the prefix rule on ids would let
// `§DX-1` reach `dx-18`, which is a different decision.
func (r *Registry) inheritedTarget(num, anchor string) (string, bool) {
	a := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(anchor), "§"))
	if t, ok := r.InheritsFrom[num][normalizeEntry(a)]; ok {
		return t, true
	}
	if t, ok := r.InheritsSections[num][sectionKey(a)]; ok {
		return t, true
	}
	// The over-read case: `§Decision 1 — Classified by…` carries the
	// heading plus a tail of the sentence. Shorten a whole word at a time,
	// longest first, and take the first length that names exactly one
	// declared source — the same rule, and the same uniqueness guard, that
	// uniqueSectionPrefix applies to a record's own headings.
	for key := sectionKey(a); key != ""; key = dropLastSegment(key) {
		t, n := "", 0
		for src, tgt := range r.InheritsSections[num] {
			if src != key && !slugPrefix(key, src) && !slugPrefix(src, key) {
				continue
			}
			t, n = tgt, n+1
		}
		if n == 1 {
			return t, true
		}
		if n > 1 {
			return "", false // ambiguous; a tiebreak would be a guess
		}
	}
	return "", false
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
//
// TWO KEYINGS, BOTH TRIED. An entry id is keyed with its hyphen dropped,
// so `DX-13`, `dx-13` and `DX13` are one anchor; a prose section is keyed
// as its slug, where the hyphens are the word boundaries and dropping
// them destroys the key. Normalizing every anchor the entry way turned
// `§Facts of record` into `factsofrecord` and missed the
// `facts-of-record` heading the registry has — so a registry's prose
// anchors were addressable only when they were a single word.
//
// Nothing in the live corpus cited one yet, which is why this stayed
// latent: `jdr/cli/0001`'s citations are all `§DX-n`. It is load-bearing
// the moment a registry inherits a named section.
func (r *Registry) Has(anchor string) bool {
	a := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(anchor), "§"))
	if id := normalizeEntry(a); r.Entries[id] || r.Inherits[id] {
		return true
	}
	k := sectionKey(a)
	return r.Entries[k] || r.Inherits[k]
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

// RegistryAt reads the registry a path names, or nil when the path is not
// a readable registry.
//
// It reads the ONE file rather than the tree. A caller that has a
// registry's path in hand — lint, given a file to judge — wants that
// file's own declarations, and walking the JDR root to find it again
// would make a single-file lint depend on a bound root it does not need.
func RegistryAt(path string) *Registry {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	reg := parseRegistry(string(body))
	reg.Number, reg.Path = registryNumber(filepath.Base(path)), path
	return reg
}

// EntryIDIn reads the entry id a heading or bullet leads with, normalized
// for lookup, or "" when it leads with none. `## DX-13 — the band` yields
// `dx13`, which is the key Entries and Inherits are held under.
func EntryIDIn(s string) string {
	if m := entryHeading.FindStringSubmatch("## " + strings.TrimLeft(s, "#- *\t")); m != nil {
		return normalizeEntry(m[1])
	}
	return ""
}

// RegistryInheriting names the one registry that took `anchor` over from
// RFD `num`, or "" when none did or more than one claims it.
//
// Same exclusivity as the resolver's alias arm, and for the same reason:
// two claimants make the anchor ambiguous, and a migration that picked
// one would rewrite a corpus of citations toward a guess. The registries
// come from the bound JDR root, so an unbound root claims nothing and the
// migration simply has no proposal to make.
func RegistryInheriting(num, anchor string) string {
	key, _ := RegistryInheritingAnchor(num, anchor)
	return key
}

// RegistryInheritingAnchor is RegistryInheriting with the LANDED anchor
// reported beside the registry: the id or slug that registry answers the
// citation with, which differs from the anchor cited whenever the
// registry declared a rename.
//
// The migration needs both halves. The registry key says which document
// to name; the landed anchor says which id to name inside it, and a
// rewrite that carried the old id across would point at an anchor the new
// home does not have — the dangling citation the alias exists to prevent,
// reintroduced by the tool meant to retire it.
func RegistryInheritingAnchor(num, anchor string) (key, landed string) {
	regs, read := cachedRegistries()
	if !read {
		return "", ""
	}
	var claimant *Registry
	for _, reg := range regs {
		t, ok := reg.inheritedTarget(num, anchor)
		if !ok {
			continue
		}
		if claimant != nil {
			return "", ""
		}
		claimant, landed = reg, t
	}
	if claimant == nil {
		return "", ""
	}
	// The same guard the resolver applies: a rename onto an anchor the
	// registry never grew is a declaration to fix, not a rewrite to
	// propose.
	if !claimant.Has(landed) {
		return "", ""
	}
	return claimant.Key(), landed
}

// cachedRegistries reads the bound JDR tree once per root. The alias
// question is asked once per CITATION — 375 of them on the live corpus —
// and walking the registry tree for each would turn a lint pass into a
// few hundred tree walks for an answer that cannot change mid-run.
var regCache struct {
	mu   sync.Mutex
	root string
	regs []*Registry
	read bool
	done bool
}

func cachedRegistries() ([]*Registry, bool) {
	root := JDRRoot()
	regCache.mu.Lock()
	defer regCache.mu.Unlock()
	if regCache.done && regCache.root == root {
		return regCache.regs, regCache.read
	}
	regCache.regs, regCache.read = LoadRegistries(root)
	regCache.root, regCache.done = root, true
	return regCache.regs, regCache.read
}
