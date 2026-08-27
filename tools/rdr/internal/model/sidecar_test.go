package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sidecarPath resolves models/rdr-template.toml at the repo root, beside
// the other data tables the binary reads.
func sidecarPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "models", "rdr-template.toml"))
	if err != nil {
		t.Fatalf("resolving rdr-template.toml: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("rdr-template.toml not found at %s: %v", p, err)
	}
	return p
}

func loadSidecar(t *testing.T) *Sidecar {
	t.Helper()
	p := sidecarPath(t)
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading %s: %v", p, err)
	}
	sc, err := ParseSidecar(string(raw), p)
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// TestSidecarDeclaresWhatTheReaderNeeds checks the shipped file against
// the values it must carry, not against the accessors that read it — the
// accessors return the sidecar, so comparing to them proves nothing.
//
// What is worth pinning is that every element kind the projector mints
// has a section, and that the lifecycle sets are the ones lint and the
// status skill depend on.
func TestSidecarDeclaresWhatTheReaderNeeds(t *testing.T) {
	sc := loadSidecar(t)

	for _, kind := range []string{"A", "C", "D", "RT", "ALT", "BR", "S", "MVV", "F"} {
		if sc.ElementSections[kind] == "" {
			t.Errorf("element kind %s names no section", kind)
		}
	}
	// G, JC and § are keyed by something other than a section's list
	// shape, so their absence is meaningful rather than an omission.
	for _, kind := range []string{"G", "JC", "§"} {
		if s, ok := sc.ElementSections[kind]; ok {
			t.Errorf("element kind %s should not name a section, got %q", kind, s)
		}
	}

	if strings.Join(sc.Terminal, "|") != "Implemented|Reverted|Abandoned|Superseded|Demoted|Rejected" {
		t.Errorf("terminal statuses = %v", sc.Terminal)
	}
	// Deferred is PARKED, not terminal: it owes no post-mortem and
	// re-enters the flow when its trigger fires. Listing it as terminal
	// would freeze a record that is expected to be amended.
	if strings.Join(sc.Parked, "|") != "Deferred" {
		t.Errorf("parked statuses = %v", sc.Parked)
	}
	for _, s := range sc.Terminal {
		if s == "Deferred" {
			t.Error("Deferred is listed as terminal; it is parked")
		}
	}

	// Rejected is the reason the observed tier exists: four frozen
	// records write it and TEMPLATE.md never listed it.
	if got := sc.Observed["Status"]; strings.Join(got, "|") != "Rejected" {
		t.Errorf("observed Status = %v, want [Rejected]", got)
	}
	// An empty set is a finding, not an omission — it must be DECLARED.
	for _, field := range []string{"Type", "Profile", "Method"} {
		got, ok := sc.Observed[field]
		if !ok {
			t.Errorf("%s declares no observed tier; write values = [] to state it is empty", field)
		}
		if len(got) != 0 {
			t.Errorf("observed %s = %v, want empty", field, got)
		}
	}
}

// TestSidecarNamesNoUnknownSection is the check a self-comparison cannot
// make: rename a section in TEMPLATE.md without updating the sidecar and
// this fails, which is what keeps the two shipping together.
func TestSidecarNamesNoUnknownSection(t *testing.T) {
	sc := loadSidecar(t)
	if err := sc.CheckAgainst(Template().Sections); err != nil {
		t.Fatalf("%v\nFIX: update models/rdr-template.toml in the same commit as the TEMPLATE.md change.", err)
	}
}

// TestSidecarRefusesRatherThanSkips pins the property the file's own
// header claims: an unrecognised shape is an error naming it, never a
// silent default. A skipped key would make a kind read as keying nothing
// — a real answer to a question nobody asked.
func TestSidecarRefusesRatherThanSkips(t *testing.T) {
	const good = "[template]\nversion = 1\n\n[elements]\nA = \"Critical Assumptions\"\n"
	if _, err := ParseSidecar(good, "t.toml"); err != nil {
		t.Fatalf("the minimal valid sidecar was refused: %v", err)
	}
	for _, c := range []struct{ name, src, want string }{
		{"unknown table", good + "\n[nope]\nx = \"y\"\n", "unknown table"},
		{"wrong version", "[template]\nversion = 2\n\n[elements]\nA = \"Critical Assumptions\"\n", "unsupported"},
		{"no elements", "[template]\nversion = 1\n", "no [elements]"},
		{"element names no section", "[template]\nversion = 1\n\n[elements]\nA = \"\"\n", "names no section"},
		{"observed without field", good + "\n[observed.x]\nvalues = []\n", "declares no field"},
		{"observed without values", good + "\n[observed.x]\nfield = \"Status\"\n", "declares no values"},
		{"lifecycle without terminal", good + "\n[lifecycle]\nparked = [\"Deferred\"]\n", "no terminal statuses"},
		{"malformed toml", good + "\na.b = \"c\"\n", "malformed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseSidecar(c.src, "t.toml")
			if err == nil {
				t.Fatalf("ParseSidecar accepted %q; it must refuse rather than skip", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

// TestSidecarSkewIsRefused: a section the template no longer has is a
// refusal, because a missed lookup would silently change an answer.
func TestSidecarSkewIsRefused(t *testing.T) {
	sc := &Sidecar{ElementSections: map[string]string{"A": "No Such Section"}, Source: "t.toml"}
	err := sc.CheckAgainst(Template().Sections)
	if err == nil {
		t.Fatal("a sidecar naming an absent section was accepted")
	}
	if !strings.Contains(err.Error(), "template-sidecar-skew") {
		t.Errorf("error = %v, want a template-sidecar-skew refusal", err)
	}
}

// TestObservedSectionsResolve: the Observed tier is keyed by section
// name, so a rename in TEMPLATE.md would silently orphan its entry and
// the labels would start reading as the author's own. This is the same
// check CheckAgainst makes for the sidecar, for the one table left in Go.
func TestObservedSectionsResolve(t *testing.T) {
	for _, fs := range sectionObserved {
		if _, ok := Template().SectionByName(fs.Section); !ok {
			t.Errorf("sectionObserved names %q, which TEMPLATE.md does not have.\n"+
				"Update fields.go in the same commit as the TEMPLATE.md change.", fs.Section)
		}
	}
}
