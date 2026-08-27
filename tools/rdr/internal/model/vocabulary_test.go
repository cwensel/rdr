package model

import "testing"

func TestStatusTiers(t *testing.T) {
	cases := []struct {
		value string
		want  Tier
	}{
		{"Draft", Canonical},
		{"Final", Canonical},
		{"Implemented", Canonical},
		{"Reverted", Canonical},
		{"Abandoned", Canonical},
		{"Superseded", Canonical},
		{"Demoted", Canonical},
		// Promoted from ObservedAccepted when TEMPLATE.md gained the
		// parked-with-a-revisit-trigger spelling the corpus had already
		// improvised. It is Canonical for writing and, being parked
		// rather than closed, is NOT in TerminalStatuses().
		{"Deferred", Canonical},

		// Present in the frozen corpus, absent from TEMPLATE.md, and a
		// legitimate terminal disposition. A reader that called it
		// off-vocabulary would be permanently wrong about records that
		// can never be amended.
		{"Rejected", ObservedAccepted},

		{"Parked", OffVocabulary},
		{"draft", OffVocabulary}, // matching is case-sensitive by design
		{"", OffVocabulary},
	}
	for _, c := range cases {
		if got := StatusVocabulary().Classify(c.value); got != c.want {
			t.Errorf("Status %q: got %s, want %s", c.value, got, c.want)
		}
	}
}

func TestTypeAndProfileTiers(t *testing.T) {
	for _, v := range TypeVocabulary().Canonical {
		if got := TypeVocabulary().Classify(v); got != Canonical {
			t.Errorf("Type %q: got %s, want canonical", v, got)
		}
	}
	if got := TypeVocabulary().Classify("Refactor"); got != OffVocabulary {
		t.Errorf("Type \"Refactor\": got %s, want off-vocabulary", got)
	}
	if len(TypeVocabulary().ObservedAccepted) != 0 {
		t.Errorf("Type has observed-accepted members %v; the corpus census found none",
			TypeVocabulary().ObservedAccepted)
	}
}

func TestParseProfile(t *testing.T) {
	cases := []struct {
		raw   string
		label string
		tier  Tier
	}{
		{"small", "small", Canonical},
		{"mid — locks the on-disk record layout", "mid", Canonical},
		{"large, locks the frame grammar", "large", Canonical},
		{"foundational: spans the reader and the writer", "foundational", Canonical},
		{"`mid` — one contract plus a user-facing surface", "mid", Canonical},
		{"enormous — not a label", "enormous", OffVocabulary},
	}
	for _, c := range cases {
		label, tier := ParseProfile(c.raw)
		if label != c.label || tier != c.tier {
			t.Errorf("ParseProfile(%q) = (%q, %s), want (%q, %s)", c.raw, label, tier, c.label, c.tier)
		}
	}
}

func TestMethodVocabularyHasNoObservedMembers(t *testing.T) {
	// This is a recorded finding, not an oversight. Every Method value in
	// the frozen corpus resolves to one of the eight sanctioned labels
	// once compounds are split and parenthetical glosses are stripped, so
	// the eight are the whole vocabulary. A future entry here needs the
	// same corpus evidence Status's two carry.
	if len(MethodVocabulary().ObservedAccepted) != 0 {
		t.Errorf("Method gained observed-accepted members %v; the eight sanctioned labels are the whole vocabulary",
			MethodVocabulary().ObservedAccepted)
	}
}

func TestParseMethodSimple(t *testing.T) {
	cases := []struct {
		raw      string
		label    string
		gloss    string
		tier     Tier
		compound bool
		valid    bool
	}{
		{"Source Search", "Source Search", "", Canonical, false, true},
		{"Spike", "Spike", "", Canonical, false, true},
		{"Docs Only", "Docs Only", "", Canonical, false, true},

		// Trailing sentence punctuation is written by several records.
		{"Source Search.", "Source Search", "", Canonical, false, true},
		{"Peer RDR.", "Peer RDR", "", Canonical, false, true},

		// A parenthetical gloss names what the method covered. It is free
		// text and is never part of the label.
		{"Spike (repro)", "Spike", "repro", Canonical, false, true},
		{"MVV Test (pending impl)", "MVV Test", "pending impl", Canonical, false, true},

		// Markdown ticks around the label.
		{"`Design Decision`", "Design Decision", "", Canonical, false, true},

		{"Benchmark", "Benchmark", "", OffVocabulary, false, false},
	}
	for _, c := range cases {
		got := ParseMethod(c.raw)
		if len(got.Members) != 1 {
			t.Fatalf("ParseMethod(%q): got %d members, want 1", c.raw, len(got.Members))
		}
		m := got.Members[0]
		if m.Label != c.label {
			t.Errorf("ParseMethod(%q).Label = %q, want %q", c.raw, m.Label, c.label)
		}
		if m.Gloss != c.gloss {
			t.Errorf("ParseMethod(%q).Gloss = %q, want %q", c.raw, m.Gloss, c.gloss)
		}
		if m.Tier != c.tier {
			t.Errorf("ParseMethod(%q).Tier = %s, want %s", c.raw, m.Tier, c.tier)
		}
		if got.Compound != c.compound {
			t.Errorf("ParseMethod(%q).Compound = %v, want %v", c.raw, got.Compound, c.compound)
		}
		if got.Valid != c.valid {
			t.Errorf("ParseMethod(%q).Valid = %v, want %v", c.raw, got.Valid, c.valid)
		}
	}
}

func TestParseMethodCompound(t *testing.T) {
	cases := []struct {
		raw    string
		labels []string
		valid  bool
		// bad names the member expected to fail, or "" when all pass.
		bad string
	}{
		// A compound built from sanctioned labels is VALID. Combining
		// methods is normal practice, not a defect.
		{"Source Search + Spike", []string{"Source Search", "Spike"}, true, ""},
		{"Peer RDR + Source Search", []string{"Peer RDR", "Source Search"}, true, ""},
		{"Source Search + Peer RDR + Spike",
			[]string{"Source Search", "Peer RDR", "Spike"}, true, ""},
		{"Source Search + MVV Test", []string{"Source Search", "MVV Test"}, true, ""},
		{"Docs Only + Spike", []string{"Docs Only", "Spike"}, true, ""},

		// Members carrying glosses.
		{"Spike (repro) + Source Search", []string{"Spike", "Source Search"}, true, ""},
		{"Spike (precondition) + MVV Test (forward outcome)",
			[]string{"Spike", "MVV Test"}, true, ""},
		{"Design Decision (arm choice) + Source Search + Spike",
			[]string{"Design Decision", "Source Search", "Spike"}, true, ""},
		{"Source Search + Peer RDR (settled) + MVV Test",
			[]string{"Source Search", "Peer RDR", "MVV Test"}, true, ""},

		// A compound with one bad member is off-vocabulary ON THAT
		// MEMBER, and the result must name which one so a gate consumer
		// can report it rather than rejecting the whole value.
		{"Spike + Corpus Probe", []string{"Spike", "Corpus Probe"}, false, "Corpus Probe"},
		{"Source Search + Vibes", []string{"Source Search", "Vibes"}, false, "Vibes"},
	}
	for _, c := range cases {
		got := ParseMethod(c.raw)
		if !got.Compound {
			t.Errorf("ParseMethod(%q).Compound = false, want true", c.raw)
		}
		if len(got.Members) != len(c.labels) {
			t.Fatalf("ParseMethod(%q): got %d members, want %d", c.raw, len(got.Members), len(c.labels))
		}
		for i, want := range c.labels {
			if got.Members[i].Label != want {
				t.Errorf("ParseMethod(%q) member %d = %q, want %q", c.raw, i, got.Members[i].Label, want)
			}
		}
		if got.Valid != c.valid {
			t.Errorf("ParseMethod(%q).Valid = %v, want %v", c.raw, got.Valid, c.valid)
		}
		if c.bad == "" {
			if len(got.OffVocabulary) != 0 {
				t.Errorf("ParseMethod(%q).OffVocabulary = %v, want empty", c.raw, got.OffVocabulary)
			}
			continue
		}
		if len(got.OffVocabulary) != 1 || got.OffVocabulary[0].Label != c.bad {
			t.Errorf("ParseMethod(%q).OffVocabulary = %v, want exactly [%q]", c.raw, got.OffVocabulary, c.bad)
		}
	}
}

// TestParseMethodBareCorpusName pins the one case where a record wrote an
// external reference-corpus name as a bare compound member instead of the
// `Docs Only` label it meant. It is correctly off-vocabulary: a gate
// consumer SHOULD flag it. The model grants no exception — naming it as an
// accepted label would launder a typo into a ninth Method.
//
// The same name appears far more often INSIDE parenthetical glosses on
// perfectly good Methods, where it is free text. Those must classify
// clean, which is what makes gloss-stripping load-bearing rather than
// cosmetic: without it, every one of those conformant records would be
// reported as off-vocabulary.
func TestParseMethodBareCorpusName(t *testing.T) {
	bare := ParseMethod("Spike (RUN) + SomeDocCorpus")
	if bare.Valid {
		t.Error("a bare external-corpus name as a compound member must be off-vocabulary")
	}
	if len(bare.OffVocabulary) != 1 || bare.OffVocabulary[0].Label != "SomeDocCorpus" {
		t.Errorf("OffVocabulary = %v, want exactly the bare member named", bare.OffVocabulary)
	}
	if bare.Members[0].Label != "Spike" || bare.Members[0].Tier != Canonical {
		t.Error("the good member of a mixed compound must still classify canonical")
	}

	inGloss := ParseMethod("Design Decision (corroborated by SomeDocCorpus § 36.17)")
	if !inGloss.Valid {
		t.Errorf("a corpus name inside a gloss must classify clean, got off-vocabulary %v", inGloss.OffVocabulary)
	}
	if inGloss.Members[0].Label != "Design Decision" {
		t.Errorf("gloss stripping failed: label = %q", inGloss.Members[0].Label)
	}
}

func TestValueContinues(t *testing.T) {
	cases := []struct {
		next string
		want bool
		why  string
	}{
		{"  the rest of a wrapped value", true, "indented continuation"},
		{"\tthe rest of a wrapped value", true, "tab-indented continuation"},
		{"", false, "blank line ends the value"},
		{"- **Type**: Feature", false, "a new field bullet ends the value"},
		{"  - nested bullet", false, "a nested bullet ends the scalar value"},
		{"  <!-- template guidance -->", false, "a template comment is not part of the value"},
		{"## Problem Statement", false, "an unindented line ends the value"},
	}
	for _, c := range cases {
		if got := ValueContinues(c.next); got != c.want {
			t.Errorf("ValueContinues(%q) = %v, want %v (%s)", c.next, got, c.want, c.why)
		}
	}
}

// TestParkedIsNotTerminal pins the distinction that justifies Deferred
// existing at all. A Deferred RDR is PAUSED, not closed: it owes no
// post-mortem, keeps its trackers, and re-enters the flow when its
// revisit trigger fires. Every consumer that branches on "is this record
// finished?" reads TerminalStatuses(), so Deferred appearing there would
// close a record that is waiting to be re-opened — the exact conflation
// the corpus record that improvised this status wrote itself to avoid.
func TestParkedIsNotTerminal(t *testing.T) {
	for _, p := range ParkedStatuses() {
		if StatusVocabulary().Classify(p) != Canonical {
			t.Errorf("parked status %q must be canonical: a status a record may not write cannot park it", p)
		}
		for _, term := range TerminalStatuses() {
			if p == term {
				t.Errorf("%q is in both ParkedStatuses() and TerminalStatuses(); a record cannot be both "+
					"paused and closed. FIX: a parked status owes no post-mortem and has a next stage.", p)
			}
		}
	}
	// The converse: nothing terminal may claim to be parked.
	if len(ParkedStatuses()) == 0 {
		t.Error("ParkedStatuses() is empty; Deferred should be in it")
	}
}
