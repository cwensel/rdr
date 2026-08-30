package model

import "testing"

func TestParseStatusBare(t *testing.T) {
	for _, v := range StatusVocabulary().Canonical {
		s := ParseStatus(v)
		if s.Label != v || s.Tier != Canonical || s.QualifierForm != NoQualifier {
			t.Errorf("ParseStatus(%q) = %+v, want a bare canonical label", v, s)
		}
	}
}

func TestParseStatusQualifiers(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		label string
		tier  Tier
		form  QualifierForm
		qual  string
	}{
		{
			name:  "demoted target",
			raw:   "Demoted [→ tracker#412]",
			label: "Demoted", tier: Canonical,
			form: QualifierDemotedTarget, qual: "→ tracker#412",
		},
		{
			name:  "revised from Final with re-verify list",
			raw:   "Draft [revised from Final 2026-03-04; re-verify A2,A4 — the frame width was never pinned]",
			label: "Draft", tier: Canonical,
			form: QualifierRevisedFrom,
			qual: "revised from Final 2026-03-04; re-verify A2,A4 — the frame width was never pinned",
		},
		{
			name:  "revised from Final without a re-verify list",
			raw:   "Draft [revised from Final 2026-03-04; — the sibling withdrew its half]",
			label: "Draft", tier: Canonical,
			form: QualifierRevisedFrom,
			qual: "revised from Final 2026-03-04; — the sibling withdrew its half",
		},
		{
			// The exact spelling a live demote pass wrote on seven
			// records: a stage token between the date and the semicolon.
			// Rejecting it degraded the form to `bracketed` and every
			// re-entry routing rule went dark.
			name:  "revised from Final with a stage token after the date",
			raw:   "Draft [revised from Final 2026-08-29 cluster-reconcile; re-verify none — wording/cross-reference fixes only]",
			label: "Draft", tier: Canonical,
			form: QualifierRevisedFrom,
			qual: "revised from Final 2026-08-29 cluster-reconcile; re-verify none — wording/cross-reference fixes only",
		},
		{
			// Missing date: the tolerance is for a stamp after the date,
			// never for the date's absence. This stays a free-text note.
			name:  "revised from Final without a date is not the form",
			raw:   "Draft [revised from Final; re-verify A2 — the date went missing]",
			label: "Draft", tier: Canonical,
			form: QualifierBracketed,
			qual: "revised from Final; re-verify A2 — the date went missing",
		},
		{
			name:  "joint decision on Final",
			raw:   "Final [joint decision → 0042-frame-grammar § A3: who owns the trailing pad byte]",
			label: "Final", tier: Canonical,
			form: QualifierJointDecision,
			qual: "joint decision → 0042-frame-grammar § A3: who owns the trailing pad byte",
		},
		{
			name:  "joint decision survives onto a terminal status",
			raw:   "Implemented [joint decision → 0042-frame-grammar § A3: the pad-byte owner]",
			label: "Implemented", tier: Canonical,
			form: QualifierJointDecision,
			qual: "joint decision → 0042-frame-grammar § A3: the pad-byte owner",
		},
		{
			name:  "legacy parenthetical commit pin",
			raw:   "Implemented (`main` c1926e1)",
			label: "Implemented", tier: Canonical,
			form: QualifierParenthetical, qual: "`main` c1926e1",
		},
		{
			name:  "legacy parenthetical on an observed-accepted status",
			raw:   "Rejected (scope shipped elsewhere; no design fork remained)",
			label: "Rejected", tier: ObservedAccepted,
			form: QualifierParenthetical, qual: "scope shipped elsewhere; no design fork remained",
		},
		{
			name:  "bracketed free-text note",
			raw:   "Draft [unblocked — the predecessor reached Implemented]",
			label: "Draft", tier: Canonical,
			form: QualifierBracketed, qual: "unblocked — the predecessor reached Implemented",
		},
		{
			name:  "undelimited em-dash clause",
			raw:   "Deferred — no solution decided; revisit when the upstream knob lands",
			label: "Deferred", tier: Canonical,
			form: QualifierDash, qual: "no solution decided; revisit when the upstream knob lands",
		},
		{
			name:  "deferred revisit trigger",
			raw:   "Deferred [revisit when upstream exposes a public pool knob]",
			label: "Deferred", tier: Canonical,
			form: QualifierRevisitWhen, qual: "revisit when upstream exposes a public pool knob",
		},
		{
			// The template writes `revisit when`; an author reaching for
			// this status writes whichever verb fits the sentence, and the
			// grammar accepts the family rather than manufacturing a
			// bracketed free-text note out of a conformant trigger.
			name:  "deferred re-open spelling",
			raw:   "Deferred [re-open if the project's stance on forks changes]",
			label: "Deferred", tier: Canonical,
			form: QualifierRevisitWhen, qual: "re-open if the project's stance on forks changes",
		},
		{
			name:  "superseded target",
			raw:   "Superseded [→ 0140-successor-slug]",
			label: "Superseded", tier: Canonical,
			form: QualifierDemotedTarget, qual: "→ 0140-successor-slug",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := ParseStatus(c.raw)
			if s.Label != c.label {
				t.Errorf("Label = %q, want %q", s.Label, c.label)
			}
			if s.Tier != c.tier {
				t.Errorf("Tier = %s, want %s", s.Tier, c.tier)
			}
			if s.QualifierForm != c.form {
				t.Errorf("QualifierForm = %s, want %s", s.QualifierForm, c.form)
			}
			if s.Qualifier != c.qual {
				t.Errorf("Qualifier = %q, want %q", s.Qualifier, c.qual)
			}
		})
	}
}

// TestParseStatusNestedQualifier covers a qualifier whose text contains its
// own bracket or paren pair. Scanning for the first closer would cut the
// value short; several corpus qualifiers nest this way.
func TestParseStatusNestedQualifier(t *testing.T) {
	raw := "Implemented [joint decision → 0133-window § A4: whether \"zero committed frames\" (the empty case) counts]"
	s := ParseStatus(raw)
	if s.Label != "Implemented" {
		t.Errorf("Label = %q, want \"Implemented\"", s.Label)
	}
	want := "joint decision → 0133-window § A4: whether \"zero committed frames\" (the empty case) counts"
	if s.Qualifier != want {
		t.Errorf("Qualifier = %q,\n            want %q", s.Qualifier, want)
	}
}

// TestParseStatusTruncatedQualifier covers a qualifier whose closing
// delimiter is missing, which happens when an upstream reader truncated a
// wrapped value. The label must still come out right.
func TestParseStatusTruncatedQualifier(t *testing.T) {
	s := ParseStatus("Draft [unblocked — the predecessor is Implemented, so the")
	if s.Label != "Draft" || s.Tier != Canonical {
		t.Errorf("ParseStatus of a truncated qualifier = %+v, want label Draft", s)
	}
}

func TestQualifierGrammarCaptures(t *testing.T) {
	m := RevisedFromGrammar.FindStringSubmatch(
		"revised from Final 2026-03-04; re-verify A2,A4 — the frame width was never pinned")
	if m == nil {
		t.Fatal("RevisedFromGrammar did not match the template form")
	}
	if m[1] != "2026-03-04" {
		t.Errorf("date capture = %q, want \"2026-03-04\"", m[1])
	}
	if m[2] != "A2,A4" {
		t.Errorf("re-verify capture = %q, want \"A2,A4\"", m[2])
	}
	if m[3] != "" {
		t.Errorf("target capture = %q, want empty when no @<stage> is written", m[3])
	}
	if m[4] != "the frame width was never pinned" {
		t.Errorf("reason capture = %q", m[4])
	}

	// The tolerated stage token rides between the date and the semicolon
	// without entering any capture, and `re-verify none` — the sanctioned
	// empty set — leaves the assumption capture empty rather than reading
	// `none` as a label the scanner would emit a dangling edge for.
	m = RevisedFromGrammar.FindStringSubmatch(
		"revised from Final 2026-08-29 cluster-reconcile; re-verify none — wording/cross-reference fixes only")
	if m == nil {
		t.Fatal("RevisedFromGrammar did not match the tolerated stage-token spelling")
	}
	if m[1] != "2026-08-29" {
		t.Errorf("date capture = %q, want \"2026-08-29\"", m[1])
	}
	if m[2] != "" {
		t.Errorf("re-verify capture = %q, want empty for `none`", m[2])
	}
	if m[4] != "wording/cross-reference fixes only" {
		t.Errorf("reason capture = %q", m[4])
	}
	if RevisedFromGrammar.MatchString(
		"revised from Final 2026-08-29 the whole reason written before the semicolon; — drift") {
		t.Error("the stage-token run is capped at two; a sentence before the semicolon must not match")
	}
}

// TestRevisedFromTargetAndTail covers the `@<stage>` slot and the
// optional reason: the target rides between the ID list and the dash
// without entering the ID capture, sits on `re-verify none` and on a
// qualifier with no re-verify clause at all, refuses a word outside
// ReentryTargets, and a qualifier that forgot its `— <reason>` is still
// the form (a stale-lock gate keys on the form, not the prose).
func TestRevisedFromTargetAndTail(t *testing.T) {
	for _, tc := range []struct {
		q                   string
		ids, target, reason string
		form                QualifierForm
	}{
		{"revised from Final 2026-08-29; re-verify A15, A17 @refine — a contract edit", "A15, A17", "refine", "a contract edit", QualifierRevisedFrom},
		{"revised from Final 2026-08-29 cluster-reconcile; re-verify none @propose — approach changed", "", "propose", "approach changed", QualifierRevisedFrom},
		{"revised from Final 2026-08-29; @resolve — no assumption reopened", "", "resolve", "no assumption reopened", QualifierRevisedFrom},
		{"revised from Final 2026-08-29; re-verify A2,A4", "A2,A4", "", "", QualifierRevisedFrom},
		{"revised from Final 2026-08-29; re-verify A2 @refine", "A2", "refine", "", QualifierRevisedFrom},
		{"revised from Final 2026-08-29; re-verify A2 @verify — not a stage", "", "", "", QualifierBracketed},
		{"revised from Final 2026-08-29; re-verify A2 @refinement — not a stage", "", "", "", QualifierBracketed},
	} {
		s := ParseStatus("Draft [" + tc.q + "]")
		if s.QualifierForm != tc.form {
			t.Errorf("%q: form = %s, want %s", tc.q, s.QualifierForm, tc.form)
			continue
		}
		if got := ReentryTarget(tc.q); got != tc.target {
			t.Errorf("%q: ReentryTarget = %q, want %q", tc.q, got, tc.target)
		}
		if tc.form != QualifierRevisedFrom {
			continue
		}
		m := RevisedFromGrammar.FindStringSubmatch(tc.q)
		if m[2] != tc.ids || m[4] != tc.reason {
			t.Errorf("%q: ids=%q reason=%q, want ids=%q reason=%q", tc.q, m[2], m[4], tc.ids, tc.reason)
		}
	}
	for _, want := range ReentryTargets {
		if ReentryTarget("revised from Final 2026-01-01; @"+want+" — x") != want {
			t.Errorf("ReentryTargets names %q but the grammar does not accept it", want)
		}
	}

	j := JointDecisionGrammar.FindStringSubmatch(
		"joint decision → 0042-frame-grammar § A3: who owns the trailing pad byte")
	if j == nil {
		t.Fatal("JointDecisionGrammar did not match the template form")
	}
	if j[1] != "0042-frame-grammar § A3" {
		t.Errorf("home capture = %q", j[1])
	}
	if j[2] != "who owns the trailing pad byte" {
		t.Errorf("question capture = %q", j[2])
	}

	d := DemotedTargetGrammar.FindStringSubmatch("→ tracker#412")
	if d == nil || d[1] != "tracker#412" {
		t.Errorf("DemotedTargetGrammar capture = %v, want \"tracker#412\"", d)
	}

	c := CommitPinGrammar.FindStringSubmatch("`main` c1926e1")
	if c == nil || c[1] != "main" || c[2] != "c1926e1" {
		t.Errorf("CommitPinGrammar capture = %v, want [main c1926e1]", c)
	}
}

func TestTransientMarker(t *testing.T) {
	line := "Transient — scheduled deletion by 0044-reader-retire, Phase 2; the legacy reader goes with it"
	m := TransientMarker.FindStringSubmatch(line)
	if m == nil {
		t.Fatal("TransientMarker did not match the template form")
	}
	if m[1] != "0044-reader-retire" {
		t.Errorf("sibling capture = %q", m[1])
	}
	if m[2] != "Phase 2" {
		t.Errorf("anchor capture = %q", m[2])
	}
	if m[3] != "the legacy reader goes with it" {
		t.Errorf("disposition capture = %q", m[3])
	}

	if TransientMarker.MatchString("Transient - scheduled deletion by x, y; z") {
		t.Error("the marker requires an em dash; a hyphen must not match")
	}
}

func TestNormativeFence(t *testing.T) {
	if !NormativeFenceOpen.MatchString("```normative") {
		t.Error("NormativeFenceOpen must match a bare normative fence")
	}
	if NormativeFenceOpen.MatchString("```go") {
		t.Error("NormativeFenceOpen must not match a language fence")
	}
	if !FenceDelimiter.MatchString("```normative") || !FenceDelimiter.MatchString("~~~") {
		t.Error("FenceDelimiter must match both fence styles")
	}
}

func TestAssumptionBullet(t *testing.T) {
	// The template form and the three corpus forms all yield the label;
	// the statement is the scanner's to read.
	for line, want := range map[string]string{
		"- **A3 [The reader tolerates a short final frame]**": "A3",
		"- **A1 — A collision-free end line is choosable.**":  "A1",
		"- **A2** The second statement":                       "A2",
		"- **A7 The seventh statement holds**":                "A7",
		"  - **A4b PostgreSQL scopes constraint names**":      "A4b",
		"- **A1.b — the split half of A1**":                   "A1.b",
	} {
		m := AssumptionBullet.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("AssumptionBullet did not match %q", line)
			continue
		}
		if m[1] != want {
			t.Errorf("label capture for %q = %q, want %q", line, m[1], want)
		}
	}
	for _, line := range []string{"- **Status**: Verified", "- [ ] **A claim with no label**", "- **AB1 [x]**"} {
		if AssumptionBullet.MatchString(line) {
			t.Errorf("AssumptionBullet matched %q", line)
		}
	}
}

func TestEvidenceFieldBullet(t *testing.T) {
	for _, label := range EvidenceFields() {
		line := "  - **" + label + "**: some value"
		m := EvidenceFieldBullet.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("EvidenceFieldBullet did not match %q", line)
		}
		if m[1] != label {
			t.Errorf("label capture = %q, want %q", m[1], label)
		}
		if m[2] != "some value" {
			t.Errorf("value capture = %q", m[2])
		}
	}
}
