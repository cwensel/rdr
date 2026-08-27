package model

import "testing"

func TestLookupSectionKinds(t *testing.T) {
	d := Template()

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

		// Legacy alias: the pre-Evidence-Record verification heading, the
		// one mapped entry with records still writing it.
		{"Dependency Source Verification at ####", "Dependency Source Verification", 4, MatchLegacyAlias, "Critical Assumptions"},
		{"Dependency Source Verification at ###", "Dependency Source Verification", 3, MatchLegacyAlias, "Critical Assumptions"},

		// The premortem is recognised and deliberately unmapped: prospective
		// hindsight is not an old spelling of the argument that a choice was
		// sound. See SectionAliases.
		{"Premortem", "Premortem", 3, MatchRecognizedUnmapped, ""},
		{"Premortem with its parenthetical", "Premortem (chosen approach)", 3, MatchRecognizedUnmapped, ""},

		// Recognised but with no canonical home: author-added sections
		// the template never adopted. Not foreign, so not a warning, but
		// nothing to project them onto either.
		{"Escaped-Defect Ledger", "Escaped-Defect Ledger", 2, MatchRecognizedUnmapped, ""},
		{"Open Questions", "Open Questions", 2, MatchRecognizedUnmapped, ""},
		{"Joint-Decision Check", "Joint-Decision Check (Stage 2)", 2, MatchRecognizedUnmapped, ""},
		{"Decision", "Decision", 2, MatchRecognizedUnmapped, ""},
		{"Consumers", "Consumers", 3, MatchRecognizedUnmapped, ""},
		{"Scope Fences", "Scope Fences", 3, MatchRecognizedUnmapped, ""},
		{"Out of scope", "Out of scope", 3, MatchRecognizedUnmapped, ""},
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
		"Critical Assumptions", "Briefly rejected",
		"Dependency Source Verification",
		"Premortem", "Escaped-Defect Ledger", "Open Questions",
		"Alternative 4: Reuse the existing walker", "Step 7: Backfill",
		"Phase 3: Rollout",
	}
	for _, h := range quiet {
		if m := LookupSection(Template(), h, 3); m.Kind == MatchUnknown {
			t.Errorf("%q classified unknown; unknown-to-template must fire only on foreign sections", h)
		}
	}
	for _, h := range []string{"Snapshot rendering", "Why this needs a record", "The gap"} {
		if m := LookupSection(Template(), h, 4); m.Kind != MatchUnknown {
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
		if m := LookupField(Template(), c.label); m.Kind != c.want {
			t.Errorf("LookupField(%q).Kind = %s, want %s", c.label, m.Kind, c.want)
		}
	}
}
