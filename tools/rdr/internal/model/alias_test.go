package model

import "testing"

func TestLookupSectionKinds(t *testing.T) {
	d := EpochDTable

	cases := []struct {
		name      string
		heading   string
		level     int
		want      MatchKind
		canonical string // "" when the match has no canonical section
	}{
		// Exact.
		{"canonical top-level", "Metadata", 2, MatchExact, "Metadata"},
		{"canonical nested", "Normative Contracts", 4, MatchExact, "Normative Contracts"},
		{"Critical Assumptions at its canonical level", "Critical Assumptions", 2, MatchExact, "Critical Assumptions"},

		// Level variant. This is the single commonest divergence in the
		// corpus: Critical Assumptions sits at ### in the great majority
		// of records and at ## in the rest. Both are the same section,
		// and neither may produce a warning.
		{"Critical Assumptions at the legacy level", "Critical Assumptions", 3, MatchLevelVariant, "Critical Assumptions"},
		{"Failure Modes re-levelled", "Failure Modes", 4, MatchLevelVariant, "Failure Modes"},

		// Case variant. Authors write these lowercase.
		{"lowercase Briefly Rejected", "Briefly rejected", 3, MatchCaseVariant, "Briefly Rejected"},
		{"lowercase Technical Environment", "Technical environment", 3, MatchCaseVariant, "Technical Environment"},

		// Legacy alias: the pre-Evidence-Record verification headings and
		// the folded premortem.
		{"API Verification", "API Verification", 4, MatchLegacyAlias, "Critical Assumptions"},
		{"Dependency Source Verification at ####", "Dependency Source Verification", 4, MatchLegacyAlias, "Critical Assumptions"},
		{"Dependency Source Verification at ###", "Dependency Source Verification", 3, MatchLegacyAlias, "Critical Assumptions"},
		{"Premortem", "Premortem", 3, MatchLegacyAlias, "Decision Rationale"},
		{"Premortem with its parenthetical", "Premortem (chosen approach)", 3, MatchLegacyAlias, "Decision Rationale"},

		// Recognised but with no canonical home: author-added sections
		// the template never adopted. Not foreign, so not a warning, but
		// nothing to project them onto either.
		{"Escaped-Defect Ledger", "Escaped-Defect Ledger", 2, MatchRecognizedUnmapped, ""},
		{"Open Questions", "Open Questions", 2, MatchRecognizedUnmapped, ""},
		{"Joint-Decision Check", "Joint-Decision Check (Stage 2)", 2, MatchRecognizedUnmapped, ""},
		{"Decision", "Decision", 2, MatchRecognizedUnmapped, ""},
		{"Consumers", "Consumers", 3, MatchRecognizedUnmapped, ""},
		{"Scope Fences", "Scope Fences", 3, MatchRecognizedUnmapped, ""},
		{"What this RDR locks", "What this RDR locks", 3, MatchRecognizedUnmapped, ""},
		{"Pre-release scope", "Pre-release scope", 3, MatchRecognizedUnmapped, ""},

		// Scaffold instances: a template block whose placeholder the
		// author filled in. Template-drawn, so not a finding.
		{"a filled-in alternative", "Alternative 2: Buffer the whole frame", 3, MatchScaffoldInstance, "Alternative 1: [Name]"},
		{"an alternative with a disambiguator", "Alternative 1 (B): Promote the survivor", 3, MatchScaffoldInstance, "Alternative 1: [Name]"},
		{"the abbreviated spelling", "Alt 3: One file per record", 3, MatchScaffoldInstance, "Alternative 1: [Name]"},
		{"an implementation step beyond the template's two", "Step 5: Import walker", 4, MatchScaffoldInstance, "Step 1: [Title]"},
		{"an activation step", "Activation Step 2: Flip the flag", 4, MatchScaffoldInstance, "Activation Step 1: [Title]"},
		{"a phase beyond the template's two", "Phase 3: Rollout", 3, MatchScaffoldInstance, "Phase 1: Code Implementation"},

		// Genuinely foreign: prose sub-headings an author invented inside
		// a template section. This is the ONLY kind that warrants
		// unknown-to-template.
		{"an instance detail heading", "Snapshot rendering", 4, MatchUnknown, ""},
		{"a per-operator heading", "Operator: CreateTable", 5, MatchUnknown, ""},
		{"an explanatory sub-heading", "Why the literature corroborates", 3, MatchUnknown, ""},
		{"an instance summary heading", "The gap", 3, MatchUnknown, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := LookupSection(d, c.heading, c.level)
			if m.Kind != c.want {
				t.Errorf("LookupSection(%q, %d).Kind = %s, want %s", c.heading, c.level, m.Kind, c.want)
			}
			if m.ObservedLevel != c.level {
				t.Errorf("ObservedLevel = %d, want %d", m.ObservedLevel, c.level)
			}
			if c.canonical == "" {
				if m.Canonical != nil {
					t.Errorf("Canonical = %q, want nil", m.Canonical.Name)
				}
				return
			}
			if m.Canonical == nil {
				t.Fatalf("Canonical = nil, want %q", c.canonical)
			}
			if m.Canonical.Name != c.canonical {
				t.Errorf("Canonical = %q, want %q", m.Canonical.Name, c.canonical)
			}
		})
	}
}

// TestLookupSectionWarningBoundary states the contract the alias table
// exists to serve: only MatchUnknown is a finding.
func TestLookupSectionWarningBoundary(t *testing.T) {
	quiet := []string{
		"Critical Assumptions", "Briefly rejected", "API Verification",
		"Premortem", "Escaped-Defect Ledger", "Open Questions",
		"Alternative 4: Reuse the existing walker", "Step 7: Backfill",
		"Phase 3: Rollout",
	}
	for _, h := range quiet {
		if m := LookupSection(EpochDTable, h, 3); m.Kind == MatchUnknown {
			t.Errorf("%q classified unknown; unknown-to-template must fire only on foreign sections", h)
		}
	}
	for _, h := range []string{"Snapshot rendering", "Why this needs a record", "The gap"} {
		if m := LookupSection(EpochDTable, h, 4); m.Kind != MatchUnknown {
			t.Errorf("foreign heading %q classified %s, want unknown", h, m.Kind)
		}
	}
}

func TestLookupField(t *testing.T) {
	cases := []struct {
		label string
		want  MatchKind
	}{
		{"Date", MatchExact},
		{"Status", MatchExact},
		{"Related Issues", MatchExact},
		{"Seam Lineage", MatchExact},
		{"Cluster", MatchExact},

		{"related issues", MatchCaseVariant},

		// The one genuine legacy spelling.
		{"Related", MatchLegacyAlias},

		// Fields individual records invented. Recognised, so not a
		// warning, but they map onto nothing.
		{"Normative home", MatchRecognizedUnmapped},
		{"Dependents", MatchRecognizedUnmapped},
		{"Peers", MatchRecognizedUnmapped},
		{"Joint decisions", MatchRecognizedUnmapped},
		{"Coordination peers", MatchRecognizedUnmapped},

		{"Sprint", MatchUnknown},
	}
	for _, c := range cases {
		if m := LookupField(EpochDTable, c.label); m.Kind != c.want {
			t.Errorf("LookupField(%q).Kind = %s, want %s", c.label, m.Kind, c.want)
		}
	}
}

// TestEpochAHasNoLaterFields checks the epoch tables actually differ where
// the census says they do, so an epoch-A record is not read against
// fields its template never had.
func TestEpochAHasNoLaterFields(t *testing.T) {
	// Overrides is deliberately absent from this list: it predates
	// Profile and Seam Lineage and belongs to epoch A's field set.
	for _, f := range []string{"Profile", "Seam Lineage", "Cluster"} {
		if m := LookupField(EpochATable, f); m.Kind == MatchExact {
			t.Errorf("epoch A has field %q, which arrived with epoch B", f)
		}
		if m := LookupField(EpochBTable, f); m.Kind != MatchExact {
			t.Errorf("epoch B is missing field %q, which it introduced", f)
		}
	}
	if _, ok := EpochATable.SectionByName("Load-Bearing Decisions"); ok {
		t.Error("epoch A has Load-Bearing Decisions, which arrived with epoch B")
	}
	if _, ok := EpochBTable.SectionByName("Load-Bearing Decisions"); !ok {
		t.Error("epoch B is missing Load-Bearing Decisions, which it introduced")
	}
	if m := LookupField(EpochATable, "Overrides"); m.Kind != MatchExact {
		t.Error("epoch A is missing Overrides, which predates the Profile apparatus")
	}
	if len(EpochATable.EvidenceFields) != 0 {
		t.Error("epoch A has an Evidence Record field set; it predates the apparatus")
	}
}

func TestEpochLevelsDiffer(t *testing.T) {
	d, ok := EpochDTable.SectionByName("Critical Assumptions")
	if !ok || d.Level != 2 {
		t.Fatalf("epoch D Critical Assumptions = %+v, want level 2", d)
	}
	for _, te := range []TemplateEpoch{EpochATable, EpochBTable, EpochCTable} {
		s, ok := te.SectionByName("Critical Assumptions")
		if !ok {
			t.Fatalf("epoch %s has no Critical Assumptions section", te.Epoch)
		}
		if s.Level != 3 {
			t.Errorf("epoch %s Critical Assumptions at level %d, want 3", te.Epoch, s.Level)
		}
	}
}

func TestDetectEpoch(t *testing.T) {
	cases := []struct {
		name string
		fp   Fingerprint
		want Epoch
	}{
		{
			name: "original template",
			fp:   Fingerprint{HasMetadataBlock: true, CriticalAssumptionsLevel: 3},
			want: EpochA,
		},
		{
			name: "original template with early Evidence Records",
			fp:   Fingerprint{HasMetadataBlock: true, CriticalAssumptionsLevel: 3, HasMethodField: true},
			want: EpochA,
		},
		{
			name: "Profile arrives",
			fp: Fingerprint{
				HasProfile: true, HasSeamLineage: true, HasLoadBearingDecisions: true,
				HasMethodField: true, CriticalAssumptionsLevel: 3,
			},
			want: EpochB,
		},
		{
			name: "gate externalised",
			fp: Fingerprint{
				HasProfile: true, HasSeamLineage: true, HasLoadBearingDecisions: true,
				HasMethodField: true, HasGatePointer: true, CriticalAssumptionsLevel: 3,
			},
			want: EpochC,
		},
		{
			name: "Critical Assumptions promoted",
			fp: Fingerprint{
				HasProfile: true, HasSeamLineage: true, HasLoadBearingDecisions: true,
				HasMethodField: true, HasGatePointer: true, CriticalAssumptionsLevel: 2,
			},
			want: EpochD,
		},
		{
			name: "joint-decision qualifier alone carries D",
			fp: Fingerprint{
				HasProfile: true, HasGatePointer: true,
				CriticalAssumptionsLevel: 3, HasJointDecisionQualifier: true,
			},
			want: EpochD,
		},
		{
			name: "the oldest records, with no Critical Assumptions section at all",
			fp:   Fingerprint{HasMetadataBlock: true},
			want: EpochA,
		},
		{
			name: "nothing recognisable",
			fp:   Fingerprint{},
			want: EpochUnknown,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectEpoch(c.fp); got != c.want {
				t.Errorf("DetectEpoch = %s, want %s", got, c.want)
			}
		})
	}
}

// TestEpochOfUnknownFallsBackToD pins the read-never-judge posture: an
// unrecognised record is read against the current template rather than
// not read at all.
func TestEpochOfUnknownFallsBackToD(t *testing.T) {
	if EpochOf(EpochUnknown).Epoch != EpochD {
		t.Error("EpochOf(EpochUnknown) must fall back to the current template")
	}
	for _, e := range []Epoch{EpochA, EpochB, EpochC, EpochD} {
		if EpochOf(e).Epoch != e {
			t.Errorf("EpochOf(%s) returned epoch %s", e, EpochOf(e).Epoch)
		}
	}
}
