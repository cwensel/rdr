package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture reads a synthetic testdata record and reduces it to the signals
// the model consumes. It is a deliberately minimal stand-in for the real
// scanner, which lands separately: enough to prove the model's predicates
// hold against a whole document, not enough to be a second scanner.
// signals are the shape facts a fixture carries. They used to be an
// epoch fingerprint; with one template they are simply what the fixture
// set must keep exercising, so a fixture edit that drops a tolerance path
// fails loudly instead of silently weakening the tests below.
type signals struct {
	HasProfile                bool
	HasSeamLineage            bool
	HasLoadBearingDecisions   bool
	HasMethodField            bool
	HasGatePointer            bool
	CriticalAssumptionsLevel  int
	HasJointDecisionQualifier bool
	HasMetadataBlock          bool
}

type fixture struct {
	name     string
	signals  signals
	headings []struct {
		name  string
		level int
	}
	fields  map[string]string
	methods []string
}

func readFixture(t *testing.T, name string) fixture {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	lines := strings.Split(string(raw), "\n")

	f := fixture{name: name, fields: map[string]string{}}
	inFence, inMeta := false, false

	for i, line := range lines {
		if FenceDelimiter.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if m := Heading.FindStringSubmatch(line); m != nil {
			level, name := len(m[1]), m[2]
			if level == 1 {
				continue
			}
			f.headings = append(f.headings, struct {
				name  string
				level int
			}{name, level})
			inMeta = name == "Metadata"

			switch name {
			case "Critical Assumptions":
				f.signals.CriticalAssumptionsLevel = level
			case "Load-Bearing Decisions":
				f.signals.HasLoadBearingDecisions = true
			}
			continue
		}

		if inMeta {
			if m := MetadataFieldBullet.FindStringSubmatch(line); m != nil {
				label := strings.TrimSpace(m[1])
				f.fields[label] = joinFixtureValue(lines, i, m[2])
				switch label {
				case "Profile":
					f.signals.HasProfile = true
				case "Seam Lineage":
					f.signals.HasSeamLineage = true
				case "Status":
					f.signals.HasMetadataBlock = true
					if ParseStatus(f.fields[label]).QualifierForm == QualifierJointDecision {
						f.signals.HasJointDecisionQualifier = true
					}
				}
			}
		}

		if m := EvidenceFieldBullet.FindStringSubmatch(line); m != nil && strings.TrimSpace(m[1]) == "Method" {
			f.signals.HasMethodField = true
			f.methods = append(f.methods, joinFixtureValue(lines, i, m[2]))
		}
		if GatePointer.MatchString(line) {
			f.signals.HasGatePointer = true
		}
	}
	return f
}

func joinFixtureValue(lines []string, i int, first string) string {
	out := first
	for j := i + 1; j < len(lines) && ValueContinues(lines[j]); j++ {
		out += " " + strings.TrimSpace(lines[j])
	}
	return strings.TrimSpace(out)
}

// TestFixturesKeepTheirShapeSignals is the fixture half of the
// same-commit rule. There is one template now, so there is no epoch to
// detect; what still has to hold is that the fixture SET spans the shapes
// the corpus contains — gate pointer and inlined gate, Critical
// Assumptions at both levels, with and without the Profile / Seam Lineage
// / Load-Bearing Decisions apparatus. A fixture edit that flattens the
// set to one shape fails here, before it can silently weaken the
// tolerance tests below.
func TestFixturesKeepTheirShapeSignals(t *testing.T) {
	a := readFixture(t, "legacy-shape.md")
	b := readFixture(t, "assumptions-nested.md")
	c := readFixture(t, "gate-inline.md")
	d := readFixture(t, "current-shape.md")

	if a.signals.HasProfile || a.signals.HasSeamLineage || a.signals.HasMethodField {
		t.Error("legacy-shape.md must carry no Profile, Seam Lineage or Method field")
	}
	if a.signals.HasGatePointer {
		t.Error("legacy-shape.md must inline its gate responses, not point at gate.md")
	}
	if !b.signals.HasProfile || !b.signals.HasSeamLineage || !b.signals.HasLoadBearingDecisions {
		t.Error("assumptions-nested.md must carry Profile, Seam Lineage and Load-Bearing Decisions")
	}
	if b.signals.HasGatePointer {
		t.Error("assumptions-nested.md must inline its gate responses, not point at gate.md")
	}
	if !c.signals.HasGatePointer {
		t.Error("gate-inline.md must point at gate.md")
	}
	if c.signals.CriticalAssumptionsLevel != 3 {
		t.Errorf("gate-inline.md Critical Assumptions at level %d, want 3", c.signals.CriticalAssumptionsLevel)
	}
	if d.signals.CriticalAssumptionsLevel != 2 {
		t.Errorf("current-shape.md Critical Assumptions at level %d, want 2", d.signals.CriticalAssumptionsLevel)
	}
	if !d.signals.HasJointDecisionQualifier {
		t.Error("current-shape.md must carry a joint-decision status qualifier")
	}
	for _, f := range []fixture{a, b, c, d} {
		if !f.signals.HasMetadataBlock {
			t.Errorf("%s must carry a Metadata block with a Status field", f.name)
		}
	}
}

// TestFixturesClassifyClean is the whole point of the tolerant reading: a
// record of any age, read against the CURRENT template, produces no
// unknown headings, no unknown metadata fields and no off-vocabulary
// values. That is the whole job now that the epochs are gone: the older
// shapes must still classify through level-variance, case-variance,
// scaffold patterns and the alias table.
func TestFixturesClassifyClean(t *testing.T) {
	for _, name := range []string{"legacy-shape.md", "assumptions-nested.md", "gate-inline.md", "current-shape.md"} {
		t.Run(name, func(t *testing.T) {
			f := readFixture(t, name)
			te := Template()

			for _, h := range f.headings {
				if m := LookupSection(te, h.name, h.level); m.Kind == MatchUnknown {
					t.Errorf("heading %q (level %d) classified unknown; a conformant record must not", h.name, h.level)
				}
			}
			for label := range f.fields {
				if m := LookupField(te, label); m.Kind == MatchUnknown {
					t.Errorf("metadata field %q classified unknown; a conformant record must not", label)
				}
			}
			if s, ok := f.fields["Status"]; ok {
				if st := ParseStatus(s); st.Tier == OffVocabulary {
					t.Errorf("Status %q classified off-vocabulary (label %q)", s, st.Label)
				}
			}
			if ty, ok := f.fields["Type"]; ok {
				if tv := ParseType(ty); !tv.Valid {
					t.Errorf("Type %q classified off-vocabulary on %v", ty, tv.OffVocabulary)
				}
			}
			if p, ok := f.fields["Profile"]; ok {
				if label, tier := ParseProfile(p); tier == OffVocabulary {
					t.Errorf("Profile %q classified off-vocabulary (label %q)", p, label)
				}
			}
			for _, mv := range f.methods {
				if parsed := ParseMethod(mv); !parsed.Valid {
					t.Errorf("Method %q classified off-vocabulary on %v", mv, parsed.OffVocabulary)
				}
			}
		})
	}
}

// TestFixturesExerciseTolerancePaths checks each fixture actually reaches
// the tolerant-reading code, rather than only the happy path. A fixture
// set that exercised nothing but exact matches would pass every test above
// while proving none of what the model exists to do.
func TestFixturesExerciseTolerancePaths(t *testing.T) {
	kinds := map[MatchKind]bool{}
	forms := map[QualifierForm]bool{}
	sawCompound, sawWrappedValue := false, false

	for _, name := range []string{"legacy-shape.md", "assumptions-nested.md", "gate-inline.md", "current-shape.md"} {
		f := readFixture(t, name)
		te := Template()

		for _, h := range f.headings {
			kinds[LookupSection(te, h.name, h.level).Kind] = true
		}
		for label, value := range f.fields {
			kinds[LookupField(te, label).Kind] = true
			if strings.Contains(value, "  ") || len(value) > 120 {
				sawWrappedValue = true
			}
		}
		if s, ok := f.fields["Status"]; ok {
			forms[ParseStatus(s).QualifierForm] = true
		}
		for _, mv := range f.methods {
			if ParseMethod(mv).Compound {
				sawCompound = true
			}
		}
	}

	for _, want := range []MatchKind{MatchExact, MatchLegacyAlias, MatchScaffoldInstance} {
		if !kinds[want] {
			t.Errorf("no fixture exercises the %s match path", want)
		}
	}
	for _, want := range []QualifierForm{QualifierJointDecision, QualifierDemotedTarget, QualifierParenthetical} {
		if !forms[want] {
			t.Errorf("no fixture exercises the %s qualifier form", want)
		}
	}
	if !sawCompound {
		t.Error("no fixture carries a compound Method")
	}
	if !sawWrappedValue {
		t.Error("no fixture wraps a metadata value onto a continuation line")
	}
}

// TestFixturesCarryNoConsumerIdentifiers is a genericity guard. The
// fixtures are synthetic because this repo is public and the corpus the
// model was built against is not; a fixture that acquired a real path or
// identifier would publish it. This checks the shapes such a leak would
// take rather than any particular name, so it keeps working as the repo
// gains fixtures.
func TestFixturesCarryNoConsumerIdentifiers(t *testing.T) {
	// Absolute paths and home-relative paths cannot appear in an invented
	// record; either is a copy from a real one. A `cli/NNNN` citation is
	// the consumer corpus's own record-reference shape.
	banned := []string{"/Users/", "/home/", "~/", "file:///", "cli/0"}

	root := filepath.Join("..", "..", "testdata")
	seen := 0
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		seen++
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		text := string(raw)
		for _, b := range banned {
			if strings.Contains(text, b) {
				t.Errorf("%s contains %q; fixtures must be synthetic, with no real paths or consumer identifiers", p, b)
			}
		}
		return nil
	})
	if err != nil || seen < 9 {
		t.Fatalf("walking testdata: %v (%d fixtures)", err, seen)
	}
}
