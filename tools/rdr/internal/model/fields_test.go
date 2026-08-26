package model

import (
	"os"
	"strings"
	"testing"
)

// templateLabels reads every `- **Label**:` bullet out of TEMPLATE.md, at
// any indent, keyed by the nearest heading above it. This is the
// labelled-bullet half of the same-commit rule: SectionFields.Canonical
// must equal what the template writes.
func templateLabels(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatalf("reading TEMPLATE.md: %v", err)
	}
	out := map[string][]string{}
	section, fenced := "", false
	for _, line := range strings.Split(string(raw), "\n") {
		if FenceDelimiter.MatchString(line) {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if m := Heading.FindStringSubmatch(line); m != nil {
			section = m[2]
			continue
		}
		if m := EvidenceFieldBullet.FindStringSubmatch(line); m != nil {
			out[section] = append(out[section], strings.TrimSpace(m[1]))
		}
	}
	return out
}

// TestSectionFieldsMatchTemplate: each FieldSet's canonical labels are
// exactly the labelled bullets TEMPLATE.md writes under that section, and
// every section TEMPLATE.md gives labelled bullets has a FieldSet.
func TestSectionFieldsMatchTemplate(t *testing.T) {
	want := templateLabels(t)
	const hint = "\n  Update SectionFields in fields.go in the same commit as the TEMPLATE.md change, " +
		"and add a synthetic fixture exercising the label."
	for _, fs := range SectionFields {
		if _, ok := Template.SectionByName(fs.Section); !ok {
			t.Errorf("FieldSet %q names no epoch D section", fs.Section)
		}
		got := want[fs.Section]
		if strings.Join(got, "|") != strings.Join(fs.Canonical, "|") {
			t.Errorf("section %q: TEMPLATE.md writes labels %v, the model has %v"+hint, fs.Section, got, fs.Canonical)
		}
	}
	for section, labels := range want {
		if len(FieldSetOf(section).Canonical) == 0 {
			t.Errorf("TEMPLATE.md writes labels %v under %q; the model has no FieldSet for it"+hint, labels, section)
		}
	}
}

// TestAssumptionStatusVocabularyMatchesTemplate: the canonical Evidence
// Record statuses are TEMPLATE.md's `Verified | Pending | Unverified`.
func TestAssumptionStatusVocabularyMatchesTemplate(t *testing.T) {
	raw, err := os.ReadFile(templatePath(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		m := EvidenceFieldBullet.FindStringSubmatch(line)
		if m == nil || strings.TrimSpace(m[1]) != "Status" || !strings.HasPrefix(line, " ") {
			continue
		}
		var got []string
		for _, p := range strings.Split(m[2], "|") {
			got = append(got, strings.TrimSpace(p))
		}
		if strings.Join(got, "|") != strings.Join(AssumptionStatusVocabulary.Canonical, "|") {
			t.Errorf("TEMPLATE.md's Evidence Record Status line is %v, the model has %v", got, AssumptionStatusVocabulary.Canonical)
		}
		if !isStatusPlaceholder(m[2]) {
			t.Errorf("the template legend %q is not recognised as the placeholder", m[2])
		}
		return
	}
	t.Fatal("TEMPLATE.md has no indented Evidence Record Status bullet")
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
