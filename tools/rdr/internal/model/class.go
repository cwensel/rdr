package model

// The document class: which template a file is judged against.
//
// The projector reads three document tiers and only one of them is an
// RDR. Judging a JDR against the RDR template is not a near miss, it is a
// category error: the registry is told it lacks Critical Assumptions,
// Proposed Solution, Implementation Plan and eight more sections it is
// defined by not having. Eleven findings, every one of them wrong, on a
// document that conforms to its own template exactly.
//
// The class is read from the ROOT the file sits under, not from its
// content. A heading set is a guess — an RFD with a Problem Statement and
// a JDR with a Problem statement differ by a letter — while the root is a
// fact the seam already states: RDR_RECORDS, RDR_JDRS and RDR_RFDS each
// name one tier's tree. A file under none of them is an RDR, which is the
// answer that was right before this existed and stays right for every
// caller that never binds the other two roots.
//
// Only the required-section list is per class. The rest of the schema —
// the Method vocabulary, the element grammar, the metadata field set — is
// the RDR's and is not asked about a registry, because no rule that reads
// it runs on one.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// DocClass is which of the three tiers a file belongs to.
type DocClass string

const (
	// ClassRDR is one seam's contract and implementation plan. It is the
	// default: a file under no other tier's root is judged as a record,
	// which is what every caller saw before the other two classes existed.
	ClassRDR DocClass = "rdr"
	// ClassJDR is a joint decision registry.
	ClassJDR DocClass = "jdr"
	// ClassRFD is a capability document.
	ClassRFD DocClass = "rfd"
)

// classRoots are the bound trees, one per non-default class. They are
// package vars for the same reason scan's roots are: the seam lives in
// package main and threading a path through every reader would let one
// of them drift.
var (
	classMu   sync.RWMutex
	jdrRoot   string
	rfdRoot   string
	classTmpl = map[DocClass]*classTemplate{}
)

// SetClassRoots binds the JDR and RFD trees. Called once, from the seam.
// An unbound root leaves its class unreachable, so every file reads as an
// RDR — the pre-existing answer, not a new failure mode.
func SetClassRoots(jdr, rfd string) {
	classMu.Lock()
	defer classMu.Unlock()
	jdrRoot, rfdRoot = jdr, rfd
	classTmpl = map[DocClass]*classTemplate{} // roots moved; re-read
}

// ClassOf reports which tier a file belongs to, from the root it sits
// under.
func ClassOf(path string) DocClass {
	c, _ := classOf(path, "")
	return c
}

// titleClass matches the class a document's own H1 declares: `# RFD 0001
// Document Tiers`, `# JDR cli/0002 What does retrofit promise?`. The
// number is not read — only which of the three words opens the title.
//
// It is a NARROW grammar on purpose. A title is the document naming
// itself, which is a strong signal, and a body heuristic over section
// names is a weak one: an RFD's `Problem Statement` and a JDR's `Problem
// statement` differ by a letter. So the fallback reads the one line that
// says the tier outright and nothing else.
var titleClass = regexp.MustCompile(`(?im)^#\s+(RFD|JDR|RDR)\s`)

// ClassOfDoc reports a file's tier from the root it sits under, falling
// back to the document's own title when no root is bound.
//
// WHY A FALLBACK AT ALL. The root is the better authority and stays
// primary: it is a fact the seam states, where a title is a string an
// author typed. But an unbound root is not an absent document class — it
// is a caller that has not bound the seam, and every file then read as an
// RDR. That is the right default for a consumer who never adopted the
// other two tiers, and the wrong one for a COPY of this repo: a registry
// verified outside its bindings is judged against the record template and
// reports eleven findings about sections it is defined by not having.
// Reading the title recovers the class in exactly that case and changes
// nothing when a root is bound.
//
// A BOUND ROOT THAT DISAGREES WITH THE TITLE IS A STOP. Both authorities
// are then claiming the file and only one can be right — the file is in
// the wrong tree, or its title is wrong — and either way the projector
// cannot judge it without choosing which to believe. Choosing is the guess
// the never-guess rule forbids, and guessing wrong here mis-judges a whole
// document rather than one reference. The reason names both readings so
// the author can see which to fix.
func ClassOfDoc(path, title string) (DocClass, error) {
	c, err := classOf(path, title)
	return c, err
}

func classOf(path, title string) (DocClass, error) {
	classMu.RLock()
	jdr, rfd := jdrRoot, rfdRoot
	classMu.RUnlock()
	rooted, bound := ClassRDR, false
	switch {
	case under(path, jdr):
		rooted, bound = ClassJDR, true
	case under(path, rfd):
		rooted, bound = ClassRFD, true
	}
	declared := titleDeclares(title)
	switch {
	case declared == "":
		// No title to read, or a title that names no tier: the root
		// answers, which is every caller that existed before this.
		return rooted, nil
	case !bound:
		// Neither root is bound OR the file sits under neither. The title
		// is the only authority present, and for an RDR title it agrees
		// with the default anyway.
		return declared, nil
	case declared != rooted:
		return rooted, fmt.Errorf(
			"stopped:class-disagreement (%s sits under the %s root but its title declares %s; "+
				"move the file to the %s tree or correct the title — the projector will not choose between them)",
			path, rooted, declared, declared)
	}
	return rooted, nil
}

// titleDeclares reads the tier a document's title names, or "" when it
// names none.
func titleDeclares(title string) DocClass {
	m := titleClass.FindStringSubmatch(title)
	if m == nil {
		return ""
	}
	switch strings.ToUpper(m[1]) {
	case "JDR":
		return ClassJDR
	case "RFD":
		return ClassRFD
	}
	return ClassRDR
}

// under reports whether path sits inside root, component-aligned so
// `/a/jdrs` is not read as inside `/a/jdr`. Both sides are made absolute
// first: a caller may name either with a relative path, and comparing a
// relative path to an absolute root never matches.
func under(path, root string) bool {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(path) == "" {
		return false
	}
	ap, err1 := filepath.Abs(path)
	ar, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(ar, ap)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// classTemplate is a non-RDR tier's template, reduced to the one thing a
// rule asks of it.
type classTemplate struct {
	// required is the Required section names in template order. A
	// nil slice from a template that could not be read is deliberate:
	// the rule then reports nothing rather than reporting every section
	// as missing, which is the never-guess rule applied to the schema
	// itself. A skipped check must not read as a passed one, and it does
	// not — it reads as no finding, the same as a conforming document.
	required []string
	// sections is EVERY section the class's template declares, Required
	// or not, in template order. The required list answers "what does
	// this document owe"; this one answers "is this heading one of its
	// template's" — a different question, and the one the heading
	// classifier asks.
	sections []Section
}

// ClassRequiredSections is the Required section names for a class, in
// template order.
//
// For ClassRDR it is the bound schema's, unchanged. For the other two it
// is read from `<root>/TEMPLATE.md` under the class's own tree in the
// ENGINE, not the consumer instance: a template is the engine's to state,
// the same way TEMPLATE.md is, and a consumer directory holds instances.
//
// A class whose template is unreadable yields nothing. That is the honest
// answer and the safe one: the alternative is reporting every section of
// a conforming registry as missing because the engine could not open a
// file.
func ClassRequiredSections(c DocClass) []string {
	if c == ClassRDR {
		var out []string
		for _, s := range Template().Sections {
			if s.Class == Required && !strings.Contains(s.Name, "[") {
				out = append(out, s.Name)
			}
		}
		return out
	}
	return classTemplateFor(c).required
}

// ClassTemplateTable is the section table a class's headings are read
// against: the RDR's bound table, or the class's own template.
//
// It is the heading classifier's half of the class split. The required
// list says what a document OWES; this says what its headings ARE, and
// reading a registry's `Interface record` against the RDR table reported
// a section its template requires as unknown-to-template.
//
// A class whose template could not be read yields an EMPTY table, not
// the RDR's. Empty means every heading is unknown, which is visible and
// wrong-looking; falling back to the RDR's means every heading is
// confidently misclassified, which is the failure this split exists to
// end.
func ClassTemplateTable(c DocClass) TemplateTable {
	if c == ClassRDR {
		return Template()
	}
	return TemplateTable{Sections: classTemplateFor(c).sections}
}

// ClassUsesRDRAliases reports whether the RDR alias table applies to a
// class. Only the RDR's own history is in it: `Problem statement` is a
// recognised PREDECESSOR of the RDR's `Problem Statement` and is a
// registry's current, correct spelling, so consulting the table for a
// registry turns a conforming heading into a legacy-name finding with a
// rename fix attached.
func ClassUsesRDRAliases(c DocClass) bool { return c == ClassRDR }

// classTemplateFor reads and caches a non-RDR class's template.
func classTemplateFor(c DocClass) *classTemplate {
	classMu.Lock()
	defer classMu.Unlock()
	if t, ok := classTmpl[c]; ok {
		return t
	}
	t := readClassTemplate(c)
	classTmpl[c] = t
	return t
}

// ClassOfTemplate reports which class a TEMPLATE.md path states, from the
// directory holding it: `<engine>/jdr/TEMPLATE.md` is the JDR's. It is
// the mirror of ClassOf, and reads a path for the same reason — a
// template's own headings are the sections it PRESCRIBES, so they cannot
// say which tier prescribes them.
//
// ClassRDR means "the root template", which is the answer for
// `<engine>/TEMPLATE.md` and for any file a caller names directly.
func ClassOfTemplate(path string) DocClass {
	if strings.TrimSpace(path) == "" {
		return ClassRDR
	}
	switch strings.ToLower(filepath.Base(filepath.Dir(path))) {
	case string(ClassJDR):
		return ClassJDR
	case string(ClassRFD):
		return ClassRFD
	default:
		return ClassRDR
	}
}

// SetClassTemplate installs a class's required-section list from an
// explicit template file, overriding the engine copy this process would
// otherwise read.
//
// It is what `--template <class>/TEMPLATE.md` means. The flag cannot
// replace the BOUND schema for a class template: that schema is the
// reader's — the element grammar, the metadata vocabulary, the Method
// list — and a registry's template declares none of them, so loading one
// as the reader's schema stopped with `malformed-template (the Metadata
// block declares no fields)` about a template defined by not having one.
// Only the required-section list is per class, so only it is overridden.
func SetClassTemplate(c DocClass, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("stopped:no-template (%s: %v)", path, err)
	}
	if _, err := parseSections(string(src)); err != nil {
		return fmt.Errorf("stopped:malformed-template (%s: %v)", path, err)
	}
	classMu.Lock()
	defer classMu.Unlock()
	classTmpl[c] = parseClassTemplate(string(src))
	return nil
}

// readClassTemplate parses `<engine>/<class>/TEMPLATE.md` for its
// Required sections, using the same heading-and-bracket grammar the RDR
// template is read with. One grammar across the three tiers is the point:
// a template that marks a section `[Conditional` means the same thing
// whichever tier writes it, and an unmarked section means Required
// everywhere.
func readClassTemplate(c DocClass) *classTemplate {
	path := classTemplatePath(c)
	if path == "" {
		return &classTemplate{}
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return &classTemplate{}
	}
	return parseClassTemplate(string(src))
}

// parseClassTemplate reads a class template's sections into both lists.
func parseClassTemplate(src string) *classTemplate {
	secs, err := parseSections(src)
	if err != nil {
		return &classTemplate{}
	}
	t := &classTemplate{}
	parents := parentOf(secs)
	for i, s := range secs {
		t.sections = append(t.sections, Section{
			Name: s.Name, Level: s.Level, Class: s.Class, Parent: parents[i], Keys: s.Keys})
		// A scaffold names a per-instance slot, never a section every
		// document owes. The JDR template's `## D1 — [The fork, as a
		// question]` is one entry's shape, not a heading a registry with
		// entries named D2 and D3 is missing. It stays in `sections`,
		// because a heading matching that shape IS the template's.
		if s.Class != Required || strings.Contains(s.Name, "[") {
			continue
		}
		t.required = append(t.required, s.Name)
	}
	return t
}

// classTemplatePath locates a class's TEMPLATE.md beside the RDR one, at
// the engine root. The bound schema's Source is the RDR template's path,
// so its directory is the engine root whichever of the three ways the
// schema was found — an explicit --template, $RDR_HOME, or beside the
// binary. Deriving it keeps the three tiers found the same way rather
// than adding a fourth resolution order to drift.
func classTemplatePath(c DocClass) string {
	if !Bound() {
		return ""
	}
	src := current().Source
	if src == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(src), string(c), "TEMPLATE.md")
}
