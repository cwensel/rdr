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
type fixture struct {
	name        string
	fingerprint Fingerprint
	headings    []struct {
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
				f.fingerprint.CriticalAssumptionsLevel = level
			case "Load-Bearing Decisions":
				f.fingerprint.HasLoadBearingDecisions = true
			}
			continue
		}

		if inMeta {
			if m := MetadataFieldBullet.FindStringSubmatch(line); m != nil {
				label := strings.TrimSpace(m[1])
				f.fields[label] = joinFixtureValue(lines, i, m[2])
				switch label {
				case "Profile":
					f.fingerprint.HasProfile = true
				case "Seam Lineage":
					f.fingerprint.HasSeamLineage = true
				case "Status":
					f.fingerprint.HasMetadataBlock = true
					if ParseStatus(f.fields[label]).QualifierForm == QualifierJointDecision {
						f.fingerprint.HasJointDecisionQualifier = true
					}
				}
			}
		}

		if m := EvidenceFieldBullet.FindStringSubmatch(line); m != nil && strings.TrimSpace(m[1]) == "Method" {
			f.fingerprint.HasMethodField = true
			f.methods = append(f.methods, joinFixtureValue(lines, i, m[2]))
		}
		if GatePointer.MatchString(line) {
			f.fingerprint.HasGatePointer = true
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

// TestFixturesDetectTheirEpoch is the fixture half of the same-commit
// rule: one synthetic record per epoch, each of which must fingerprint as
// the epoch it was written for.
func TestFixturesDetectTheirEpoch(t *testing.T) {
	cases := []struct {
		file string
		want Epoch
	}{
		{"epoch-a.md", EpochA},
		{"epoch-b.md", EpochB},
		{"epoch-c.md", EpochC},
		{"epoch-d.md", EpochD},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			f := readFixture(t, c.file)
			if got := DetectEpoch(f.fingerprint); got != c.want {
				t.Errorf("DetectEpoch = %s, want %s (fingerprint %+v)", got, c.want, f.fingerprint)
			}
		})
	}
}

// TestFixtureEpochSignals pins what makes each fixture its epoch, so a
// fixture edit that quietly removes a signal fails here rather than
// silently weakening the detection test above.
func TestFixtureEpochSignals(t *testing.T) {
	a := readFixture(t, "epoch-a.md")
	if a.fingerprint.HasProfile || a.fingerprint.HasSeamLineage || a.fingerprint.HasMethodField {
		t.Error("epoch-a must carry no Profile, no Seam Lineage and no Evidence Record Method field")
	}
	if a.fingerprint.HasGatePointer {
		t.Error("epoch-a must inline its gate responses, not point at gate.md")
	}

	b := readFixture(t, "epoch-b.md")
	if !b.fingerprint.HasProfile || !b.fingerprint.HasSeamLineage || !b.fingerprint.HasLoadBearingDecisions {
		t.Error("epoch-b must carry Profile, Seam Lineage and Load-Bearing Decisions")
	}
	if b.fingerprint.HasGatePointer {
		t.Error("epoch-b must still inline its gate responses")
	}

	c := readFixture(t, "epoch-c.md")
	if !c.fingerprint.HasGatePointer {
		t.Error("epoch-c must replace the gate body with a gate.md pointer")
	}
	if c.fingerprint.CriticalAssumptionsLevel != 3 {
		t.Errorf("epoch-c Critical Assumptions at level %d, want 3", c.fingerprint.CriticalAssumptionsLevel)
	}

	d := readFixture(t, "epoch-d.md")
	if d.fingerprint.CriticalAssumptionsLevel != 2 {
		t.Errorf("epoch-d Critical Assumptions at level %d, want 2", d.fingerprint.CriticalAssumptionsLevel)
	}
	if !d.fingerprint.HasJointDecisionQualifier {
		t.Error("epoch-d must carry a joint-decision status qualifier")
	}
}

// TestFixturesClassifyClean is the whole point of the tolerant reading: a
// conformant record of ANY epoch produces no unknown headings, no unknown
// metadata fields and no off-vocabulary values.
func TestFixturesClassifyClean(t *testing.T) {
	for _, name := range []string{"epoch-a.md", "epoch-b.md", "epoch-c.md", "epoch-d.md"} {
		t.Run(name, func(t *testing.T) {
			f := readFixture(t, name)
			te := EpochOf(DetectEpoch(f.fingerprint))

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

	for _, name := range []string{"epoch-a.md", "epoch-b.md", "epoch-c.md", "epoch-d.md"} {
		f := readFixture(t, name)
		te := EpochOf(DetectEpoch(f.fingerprint))

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
