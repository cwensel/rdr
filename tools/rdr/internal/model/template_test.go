package model

import (
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

// observedSection is what TEMPLATE.md declares about one heading, read
// back through the PRODUCTION parser. The anti-drift tests below compare
// the model against the same reading the binary does, so a divergence is
// a real disagreement rather than two parsers differing.
type observedSection = templateSection

// elementSection reports whether a canonical section is one an element
// kind is projected from, and which kind.
func elementSection(name string) (string, bool) {
	for kind, section := range ElementSections() {
		if section == name {
			return kind, true
		}
	}
	return "", false
}

func parseTemplate(t *testing.T, path string) []observedSection {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	got, err := parseSections(string(raw))
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return got
}

// TestEpochDParentsResolve checks the section table is internally
// coherent: every named parent exists, and sits at a shallower level.
func TestEpochDParentsResolve(t *testing.T) {
	for _, s := range Template().Sections {
		if s.Parent == "" {
			if s.Level != 2 {
				t.Errorf("section %q has no parent but sits at level %d; only level-2 sections are top-level", s.Name, s.Level)
			}
			continue
		}
		p, ok := Template().SectionByName(s.Parent)
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

	if len(found) != len(MethodVocabulary().Canonical) {
		t.Fatalf("README.md defines %d Method labels %v, the model has %d %v.\n"+
			"FIX: update MethodVocabulary in tools/rdr/internal/model/vocabulary.go in the same commit.",
			len(found), found, len(MethodVocabulary().Canonical), MethodVocabulary().Canonical)
	}
	for i, f := range found {
		if f != MethodVocabulary().Canonical[i] {
			t.Errorf("Method label %d: README.md says %q, the model says %q.\n"+
				"FIX: update MethodVocabulary in the same commit.", i+1, f, MethodVocabulary().Canonical[i])
		}
	}
}

// TestGateKeysComeFromMarkers binds the G-keys to TEMPLATE.md's own
// `[Gate key: …]` markers. The key is not the heading slugified —
// `Contradiction Check` keys `contradiction` — and peers cite gate
// responses as `cli/NNNN:G-<key>`, so the spelling is a stable id that a
// reworded heading must not move. Declaring it in the template keeps the
// reader from being a second source for it.
func TestGateKeysComeFromMarkers(t *testing.T) {
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatal(err)
	}
	secs, err := parseSections(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	got, err := gateItems(secs)
	if err != nil {
		t.Fatalf("%v\nFIX: every Finalization Gate sub-section carries a [Gate key: <key>] marker.", err)
	}
	if len(got) != len(GateItems()) {
		t.Fatalf("TEMPLATE.md declares %d gate items, the model has %d: %+v", len(got), len(GateItems()), got)
	}
	for i := range got {
		if got[i] != GateItems()[i] {
			t.Errorf("gate item %d: TEMPLATE.md declares %+v, the model has %+v.\n"+
				"Update GateItems() in template.go in the same commit as the marker.", i, got[i], GateItems()[i])
		}
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
