package model

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// templatePath resolves TEMPLATE.md at the repo root, four levels above
// this file's directory (tools/rdr/internal/model).
func templatePath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "TEMPLATE.md"))
	if err != nil {
		t.Fatalf("resolving TEMPLATE.md: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("TEMPLATE.md not found at %s: %v\n"+
			"The anti-drift test reads the real template. If the repo layout moved, "+
			"update templatePath in this file.", p, err)
	}
	return p
}

// observedSection is a heading read straight out of TEMPLATE.md, with the
// class its own bracket text declares.
type observedSection struct {
	Name  string
	Level int
	Class Class
	// ClassExplicit records whether the class came from a bracket marker
	// or from the unmarked-means-Required default.
	ClassExplicit bool
}

var (
	requiredMarker    = regexp.MustCompile(`\[Required\b`)
	conditionalMarker = regexp.MustCompile(`\[Conditional\b`)
	// delegatedMarker matches a `[Conditional scaffold — ...]` clause,
	// which declares a CHILD block's class rather than the section's own.
	// Alternatives Considered carries one on behalf of `Alternative 1`.
	// Treating it as a self-declaration would make a Required spine
	// section read as Conditional.
	delegatedMarker = regexp.MustCompile(`\[Conditional scaffold\b`)
)

// parseTemplate extracts TEMPLATE.md's heading tree and each section's
// declared class.
//
// A section's class is read from the bracket text in its body — the text
// between its own heading and the next heading — skipping fenced code and
// HTML comments so that a marker quoted in guidance is not mistaken for a
// declaration. A section with no marker takes the Required default per
// SectionClassRule; the caller reconciles the scaffold exceptions.
func parseTemplate(t *testing.T, path string) []observedSection {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	lines := strings.Split(string(raw), "\n")

	var out []observedSection
	var body []string

	flush := func() {
		if len(out) == 0 {
			return
		}
		text := strings.Join(body, "\n")
		if delegatedMarker.MatchString(text) {
			// Strip the delegated clause before reading the section's own
			// class, so the clause governs only the child it names.
			text = delegatedMarker.ReplaceAllString(text, "")
		}
		s := &out[len(out)-1]
		switch {
		case conditionalMarker.MatchString(text):
			s.Class, s.ClassExplicit = Conditional, true
		case requiredMarker.MatchString(text):
			s.Class, s.ClassExplicit = Required, true
		default:
			s.Class, s.ClassExplicit = Required, false
		}
		body = nil
	}

	inFence, inComment := false, false
	for _, line := range lines {
		if FenceDelimiter.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if !inComment && strings.Contains(line, "<!--") && !strings.Contains(line, "-->") {
			inComment = true
			continue
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}
		if strings.Contains(line, "<!--") && strings.Contains(line, "-->") {
			continue
		}

		if m := Heading.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			if level == 1 {
				continue // the record title, not a section
			}
			flush()
			out = append(out, observedSection{Name: m[2], Level: level})
			continue
		}
		body = append(body, line)
	}
	flush()

	if len(out) == 0 {
		t.Fatalf("no sections parsed from %s — the heading grammar or the file layout changed", path)
	}
	return out
}

// TestEpochDMatchesTemplateFile is the anti-drift test. TEMPLATE.md and
// the epoch-D table cannot disagree without this failing.
//
// It compares, position by position, the section names, levels and
// classes that TEMPLATE.md actually declares against EpochDTable. It fails
// on a gained, lost, renamed, re-levelled or re-classed section.
func TestEpochDMatchesTemplateFile(t *testing.T) {
	path := templatePath(t)
	observed := parseTemplate(t, path)
	model := EpochDTable.Sections

	const fixHint = "\n\nFIX: update EpochDTable in tools/rdr/internal/model/epoch.go in the SAME commit " +
		"as the TEMPLATE.md change, and add a synthetic fixture under tools/rdr/testdata/ if the " +
		"change affects how a record is read. See tools/rdr/README.md, 'The same-commit rule'."

	for i := 0; i < len(observed) && i < len(model); i++ {
		o, m := observed[i], model[i]
		if o.Name != m.Name {
			t.Fatalf("section %d: TEMPLATE.md declares %q but the model has %q.\n"+
				"A section was added, removed or renamed."+fixHint, i+1, o.Name, m.Name)
		}
		if o.Level != m.Level {
			t.Errorf("section %q: TEMPLATE.md writes it at level %d, the model has level %d.\n"+
				"The section was re-levelled."+fixHint, o.Name, o.Level, m.Level)
		}
		if o.Class != m.Class {
			// The unmarked default and a scaffold clause carried by the
			// parent are both legitimate reasons for the model to say
			// Conditional where the section's own body has no marker.
			// Anything else is a real class flip.
			if !o.ClassExplicit && m.Class == Conditional && scaffoldConditional(m.Name) {
				continue
			}
			t.Errorf("section %q: TEMPLATE.md declares it %s, the model has %s.\n"+
				"The section's class flipped."+fixHint, o.Name, o.Class, m.Class)
		}
	}

	if len(observed) != len(model) {
		var extra []string
		if len(observed) > len(model) {
			for _, o := range observed[len(model):] {
				extra = append(extra, fmt.Sprintf("TEMPLATE.md has %q (level %d), the model does not", o.Name, o.Level))
			}
		} else {
			for _, m := range model[len(observed):] {
				extra = append(extra, fmt.Sprintf("the model has %q (level %d), TEMPLATE.md does not", m.Name, m.Level))
			}
		}
		t.Fatalf("TEMPLATE.md declares %d sections, the model has %d:\n  %s"+fixHint,
			len(observed), len(model), strings.Join(extra, "\n  "))
	}
}

// scaffoldConditional names the sections whose Conditional class is
// declared by a parent's scaffold clause rather than by their own bracket
// text. Keeping the list here, rather than loosening the class comparison,
// means a class flip anywhere else still fails.
func scaffoldConditional(name string) bool {
	switch name {
	case "Alternative 1: [Name]",
		"Step 1: [Title]",
		"Step 2: [Title]",
		"Phase 2: Operational Activation",
		"Activation Step 1: [Title]":
		return true
	}
	return false
}

// TestEpochDParentsResolve checks the section table is internally
// coherent: every named parent exists, and sits at a shallower level.
func TestEpochDParentsResolve(t *testing.T) {
	for _, s := range EpochDTable.Sections {
		if s.Parent == "" {
			if s.Level != 2 {
				t.Errorf("section %q has no parent but sits at level %d; only level-2 sections are top-level", s.Name, s.Level)
			}
			continue
		}
		p, ok := EpochDTable.SectionByName(s.Parent)
		if !ok {
			t.Errorf("section %q names parent %q, which is not in the table", s.Name, s.Parent)
			continue
		}
		if p.Level >= s.Level {
			t.Errorf("section %q (level %d) names parent %q at level %d; a parent must be shallower",
				s.Name, s.Level, p.Name, p.Level)
		}
	}
}

// TestMethodVocabularyMatchesREADME checks the eight Method labels against
// README.md, which declares itself the authoritative Method vocabulary.
// The template points at README for this list precisely so guidance does
// not ship inside the template body, so this is the second half of the
// same-commit rule.
func TestMethodVocabularyMatchesREADME(t *testing.T) {
	p, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "README.md"))
	if err != nil {
		t.Fatalf("resolving README.md: %v", err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	text := string(raw)

	start := strings.Index(text, "### Verifying load-bearing claims")
	if start < 0 {
		t.Fatal("README.md has no 'Verifying load-bearing claims' section; the Method vocabulary lost its home")
	}
	section := text[start:]
	if end := strings.Index(section, "\n## "); end > 0 {
		section = section[:end]
	}

	label := regexp.MustCompile(`(?m)^- \*\*([^*]+)\*\* —`)
	var found []string
	for _, m := range label.FindAllStringSubmatch(section, -1) {
		found = append(found, strings.TrimSpace(m[1]))
	}

	if len(found) != len(MethodVocabulary.Canonical) {
		t.Fatalf("README.md defines %d Method labels %v, the model has %d %v.\n"+
			"FIX: update MethodVocabulary in tools/rdr/internal/model/vocabulary.go in the same commit.",
			len(found), found, len(MethodVocabulary.Canonical), MethodVocabulary.Canonical)
	}
	for i, f := range found {
		if f != MethodVocabulary.Canonical[i] {
			t.Errorf("Method label %d: README.md says %q, the model says %q.\n"+
				"FIX: update MethodVocabulary in the same commit.", i+1, f, MethodVocabulary.Canonical[i])
		}
	}
}

// TestStatusVocabularyMatchesTemplate checks the canonical Status set
// against TEMPLATE.md's Status line. ObservedAccepted is deliberately not
// checked against the template: those values exist precisely because the
// template never listed them.
func TestStatusVocabularyMatchesTemplate(t *testing.T) {
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatalf("reading TEMPLATE.md: %v", err)
	}
	text := string(raw)

	i := strings.Index(text, "- **Status**:")
	if i < 0 {
		t.Fatal("TEMPLATE.md has no Status metadata field")
	}
	// The value spans to the HTML comment that follows it.
	rest := text[i+len("- **Status**:"):]
	if j := strings.Index(rest, "<!--"); j > 0 {
		rest = rest[:j]
	}
	value := strings.Join(strings.Fields(rest), " ")

	var declared []string
	for _, part := range strings.Split(value, "|") {
		if p := strings.TrimSpace(part); p != "" {
			declared = append(declared, p)
		}
	}

	if len(declared) != len(StatusVocabulary.Canonical) {
		t.Fatalf("TEMPLATE.md declares %d statuses %v, the model has %d %v.\n"+
			"FIX: update StatusVocabulary in tools/rdr/internal/model/vocabulary.go in the same commit.",
			len(declared), declared, len(StatusVocabulary.Canonical), StatusVocabulary.Canonical)
	}
	for i, d := range declared {
		if d != StatusVocabulary.Canonical[i] {
			t.Errorf("status %d: TEMPLATE.md says %q, the model says %q.\n"+
				"FIX: update StatusVocabulary in the same commit.", i+1, d, StatusVocabulary.Canonical[i])
		}
	}
}

// TestTypeVocabularyMatchesTemplate does the same for the Type line.
func TestTypeVocabularyMatchesTemplate(t *testing.T) {
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatalf("reading TEMPLATE.md: %v", err)
	}
	text := string(raw)

	i := strings.Index(text, "- **Type**:")
	if i < 0 {
		t.Fatal("TEMPLATE.md has no Type metadata field")
	}
	rest := text[i+len("- **Type**:"):]
	if j := strings.Index(rest, "\n- **"); j > 0 {
		rest = rest[:j]
	}
	value := strings.Join(strings.Fields(rest), " ")

	var declared []string
	for _, part := range strings.Split(value, "|") {
		if p := strings.TrimSpace(part); p != "" {
			declared = append(declared, p)
		}
	}
	if len(declared) != len(TypeVocabulary.Canonical) {
		t.Fatalf("TEMPLATE.md declares %d types %v, the model has %d %v.\n"+
			"FIX: update TypeVocabulary in the same commit.",
			len(declared), declared, len(TypeVocabulary.Canonical), TypeVocabulary.Canonical)
	}
	for i, d := range declared {
		if d != TypeVocabulary.Canonical[i] {
			t.Errorf("type %d: TEMPLATE.md says %q, the model says %q.\nFIX: update TypeVocabulary in the same commit.",
				i+1, d, TypeVocabulary.Canonical[i])
		}
	}
}

// TestDecisionClassesMatchTemplate reads the Load-Bearing Decisions
// bullets out of TEMPLATE.md and asserts DecisionClasses lists exactly
// them, in order — the same-commit rule for the D-key vocabulary.
func TestDecisionClassesMatchTemplate(t *testing.T) {
	body := sectionBody(t, "#### Load-Bearing Decisions")
	var got []string
	for _, l := range strings.Split(body, "\n") {
		if m := regexp.MustCompile(`^- \*\*([^*]+)\*\*`).FindStringSubmatch(l); m != nil {
			got = append(got, strings.TrimSpace(m[1]))
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(DecisionClasses) {
		t.Fatalf("TEMPLATE.md Load-Bearing Decisions classes %v != model.DecisionClasses %v\n"+
			"Update DecisionClasses in template.go in the same commit as the template.", got, DecisionClasses)
	}
}

// TestGateItemsMatchTemplate asserts every `###` under Finalization Gate
// has a G-key and no G-key names a section the template lacks.
func TestGateItemsMatchTemplate(t *testing.T) {
	body := sectionBody(t, "## Finalization Gate")
	var got []string
	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "### ") {
			got = append(got, strings.TrimSpace(strings.TrimPrefix(l, "### ")))
		}
	}
	var want []string
	for _, g := range GateItems {
		want = append(want, g.Section)
		if g.Key == "" || GateItemKey(g.Section) != g.Key {
			t.Errorf("GateItems entry %q has no usable key", g.Section)
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("TEMPLATE.md Finalization Gate sub-sections %v != model.GateItems %v\n"+
			"Update GateItems in template.go in the same commit as the template.", got, want)
	}
}

func TestDecisionClassOf(t *testing.T) {
	for label, want := range map[string]string{
		"Identity":                           "Identity",
		"Identity — what makes two the same": "Identity",
		"Selection / predicate (remainder)":  "Selection / predicate",
		"Selection":                          "Selection / predicate",
		"Wire":                               "Wire / byte format",
		"wire / byte format":                 "Wire / byte format",
		"Naming":                             "Naming",
		"Verdict":                            "",
		"The one immutability exemption":     "",
		"Identification of the carrier":      "",
	} {
		if got := DecisionClassOf(label); got != want {
			t.Errorf("DecisionClassOf(%q) = %q, want %q", label, got, want)
		}
	}
}

// sectionBody returns the lines of TEMPLATE.md from the given heading to
// the next heading of the same or a higher level.
func sectionBody(t *testing.T, heading string) string {
	t.Helper()
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatal(err)
	}
	level := len(heading) - len(strings.TrimLeft(heading, "#"))
	var out []string
	in := false
	for _, l := range strings.Split(string(raw), "\n") {
		if l == heading {
			in = true
			continue
		}
		if in && strings.HasPrefix(l, "#") {
			if len(l)-len(strings.TrimLeft(l, "#")) <= level {
				break
			}
		}
		if in {
			out = append(out, l)
		}
	}
	if !in {
		t.Fatalf("heading %q not found in TEMPLATE.md", heading)
	}
	return strings.Join(out, "\n")
}
