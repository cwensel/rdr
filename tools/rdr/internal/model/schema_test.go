package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoFile(t *testing.T, parts ...string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join(append([]string{"..", "..", "..", ".."}, parts...)...))
	if err != nil {
		t.Fatalf("resolving %v: %v", parts, err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("%s not found: %v", p, err)
	}
	return p
}

func loadSchema(t *testing.T) *Schema {
	t.Helper()
	s, err := Load(
		repoFile(t, "TEMPLATE.md"),
		repoFile(t, "README.md"),
		repoFile(t, "models", "rdr-template.toml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestLoadReadsWhatTheTemplateStates checks the schema against the
// DOCUMENT rather than against another copy of itself.
//
// The hand tables this once compared to are gone, so a comparison with
// the loader's own output would prove nothing. What is still checkable is
// that the values the template visibly writes are the values that come
// out: the assertions below quote TEMPLATE.md.
func TestLoadReadsWhatTheTemplateStates(t *testing.T) {
	s := loadSchema(t)

	// The spine the template opens and closes with.
	if got := s.Table.Sections[0].Name; got != "Metadata" {
		t.Errorf("first section = %q, want Metadata", got)
	}
	if got := s.Table.Sections[len(s.Table.Sections)-1].Name; got != "References" {
		t.Errorf("last section = %q, want References", got)
	}

	// Nesting comes from heading levels, so a level-4 section under
	// Technical Design must name it as its parent.
	nc, ok := s.Table.SectionByName("Normative Contracts")
	if !ok {
		t.Fatal("no Normative Contracts section")
	}
	if nc.Level != 4 || nc.Parent != "Technical Design" {
		t.Errorf("Normative Contracts = level %d under %q, want level 4 under Technical Design", nc.Level, nc.Parent)
	}
	if !nc.Keys {
		t.Error("Normative Contracts should key its items: the template shows **C1** leads")
	}

	// A `[Conditional` marker in a section's own body, and a scaffold
	// declaring itself one.
	for name, want := range map[string]Class{
		"Load-Bearing Decisions": Conditional,
		"Alternative 1: [Name]":  Conditional,
		"Problem Statement":      Required,
		"Finalization Gate":      Required,
	} {
		sec, ok := s.Table.SectionByName(name)
		if !ok {
			t.Errorf("no %q section", name)
			continue
		}
		if sec.Class != want {
			t.Errorf("%s class = %v, want %v", name, sec.Class, want)
		}
	}

	// The vocabularies the template writes as pipe-separated lines. The
	// Profile line ends in prose, so its last member is a real parse
	// question rather than a formality.
	for field, want := range map[string][]string{
		"Status":  {"Draft", "Final", "Implemented", "Reverted", "Abandoned", "Superseded", "Demoted", "Deferred"},
		"Type":    {"Feature", "Bug Fix", "Technical Debt", "Framework Workaround", "Architecture"},
		"Profile": {"small", "mid", "large", "foundational"},
	} {
		var got []string
		for _, v := range s.Table.Vocabularies {
			if v.Field == field {
				got = v.Canonical
			}
		}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s vocabulary = %v, want %v", field, got, want)
		}
	}

	if strings.Join(s.Table.EvidenceFields, "|") != "Status|Method|Evidence|If wrong" {
		t.Errorf("evidence fields = %v", s.Table.EvidenceFields)
	}

	// The gate's keys are declared, not derived: `Contradiction Check`
	// keys `contradiction`, which no slugification produces.
	items, err := gateItems(s.Sections)
	if err != nil {
		t.Fatal(err)
	}
	want := []GateItem{
		{"Contradiction Check", "contradiction", false},
		{"Assumption Verification", "assumptions", false},
		{"Scope Verification", "scope", false},
		{"Cross-Cutting Concerns", "cross-cutting", true},
		{"Proportionality", "proportionality", false},
	}
	if fmt.Sprint(items) != fmt.Sprint(want) {
		t.Errorf("gate items = %v, want %v", items, want)
	}
}

// TestLoadRefusesAMissingFile: the binary stops rather than reading a
// half-schema. A zero schema would classify every value off-vocabulary
// and every heading unknown-to-template — a corpus-wide defect reported
// for what is really a missing install.
func TestLoadRefusesAMissingFile(t *testing.T) {
	tmpl, readme, side := repoFile(t, "TEMPLATE.md"), repoFile(t, "README.md"), repoFile(t, "models", "rdr-template.toml")
	for _, c := range []struct{ name, tmpl, readme, side, want string }{
		{"no template", filepath.Join(t.TempDir(), "nope.md"), readme, side, "stopped:no-template"},
		{"no readme", tmpl, filepath.Join(t.TempDir(), "nope.md"), side, "stopped:no-readme"},
		{"no sidecar", tmpl, readme, filepath.Join(t.TempDir(), "nope.toml"), "stopped:no-template-sidecar"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Load(c.tmpl, c.readme, c.side)
			if err == nil {
				t.Fatal("Load succeeded without a required file")
			}
			if !strings.HasPrefix(err.Error(), c.want) {
				t.Errorf("error = %v, want it to start %q", err, c.want)
			}
		})
	}
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
