package model

import (
	"os"
	"testing"
)

// templateLabelsFor reads TEMPLATE.md through the production parser, so
// the anti-drift test below checks the model against the same reading the
// binary does.
func templateLabelsFor(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatalf("reading TEMPLATE.md: %v", err)
	}
	return templateLabels(string(raw))
}

func TestLookupLabel(t *testing.T) {
	ca := FieldSetOf("Critical Assumptions")
	nc := FieldSetOf("Normative Contracts")
	cases := []struct {
		set       FieldSet
		label     string
		kind      MatchKind
		canonical string
	}{
		{ca, "Status", MatchExact, "Status"},
		{ca, "status", MatchCaseVariant, "Status"},
		{ca, "Status:", MatchExact, "Status"},
		{ca, "Evidence — the two source-checkable channel reductions", MatchPrefix, "Evidence"},
		{ca, "Evidence (plan)", MatchPrefix, "Evidence"},
		{ca, "If wrong (a refusal is owed)", MatchPrefix, "If wrong"},
		{ca, "Evidence plan", MatchPrefix, "Evidence"},
		{ca, "Evidenced by", MatchAuthor, ""},
		{ca, "Note", MatchAuthor, ""},
		{ca, "If  wrong", MatchExact, "If wrong"},
		{nc, "Status / sentinel errors", MatchExact, "Status / sentinel errors"},
		{nc, "Status / sentinel errors (v2)", MatchPrefix, "Status / sentinel errors"},
		{nc, "Preview / dry-run", MatchExact, "Preview / dry-run"},
		{FieldSetOf("Failure Modes"), "Silent (guarded)", MatchPrefix, "Silent"},
		{FieldSetOf("Consequences"), "Positive", MatchExact, "Positive"},
		{FieldSetOf("Briefly Rejected"), "A 16-bit CRC", MatchScaffoldInstance, "[Alternative N]"},
		{FieldSetOf("Approach"), "Risk", MatchAuthor, ""},
		{FieldSetOf(""), "Anything", MatchAuthor, ""},
	}
	for _, c := range cases {
		got := LookupLabel(c.set, c.label)
		if got.Kind != c.kind || got.Canonical != c.canonical {
			t.Errorf("LookupLabel(%q, %q) = %s %q, want %s %q", c.set.Section, c.label, got.Kind, got.Canonical, c.kind, c.canonical)
		}
	}
}

func TestParseAssumptionStatus(t *testing.T) {
	cases := []struct {
		raw, value, qualifier string
		tier                  Tier
		placeholder           bool
	}{
		{"Verified", "Verified", "", Canonical, false},
		{"**Verified**", "Verified", "", Canonical, false},
		{"Pending", "Pending", "", Canonical, false},
		{"Verified as narrowed — total over live entries", "Verified", "as narrowed — total over live entries", Canonical, false},
		{"Verified-by-derivation", "Verified", "by-derivation", Canonical, false},
		{"**Verified** (live spike, both hosts) — **Method**: Spike", "Verified", "(live spike, both hosts) — Method: Spike", Canonical, false},
		{"Verified.** The subset is exactly two values.", "Verified", "The subset is exactly two values.", Canonical, false},
		{"REFUTED", "Refuted", "", ObservedAccepted, false},
		{"REFUTED (one test pinned the old order)", "Refuted", "one test pinned the old order", ObservedAccepted, false},
		{"Resolved — settled by the Naming decision", "Resolved", "settled by the Naming decision", ObservedAccepted, false},
		{"Accepted at Stage 6", "Accepted", "at Stage 6", ObservedAccepted, false},
		{"Verified | Pending | Unverified", "", "", OffVocabulary, true},
		{"Partially Verified", "Partially Verified", "", OffVocabulary, false},
		{"Decided (Design Decision)", "Decided", "Design Decision", OffVocabulary, false},
		{"", "", "", OffVocabulary, false},
	}
	for _, c := range cases {
		got := ParseAssumptionStatus(c.raw)
		if got.Value != c.value || got.Qualifier != c.qualifier || got.Tier != c.tier || got.Placeholder != c.placeholder {
			t.Errorf("ParseAssumptionStatus(%q) = {%q %q %s %v}, want {%q %q %s %v}",
				c.raw, got.Value, got.Qualifier, got.Tier, got.Placeholder, c.value, c.qualifier, c.tier, c.placeholder)
		}
		if got.Raw != c.raw {
			t.Errorf("raw not preserved: %q", got.Raw)
		}
	}
}
