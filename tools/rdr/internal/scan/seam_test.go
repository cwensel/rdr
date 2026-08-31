package scan

import "testing"

// TestSeamLineageReadsTheDeclaredForms pins the count grammar: the
// template's `Nth point-fix`, the corpus's `N prior point-fixes` and
// `Count = N`, and `no prior accretion` as zero. Everything else is
// UNREAD — a nil count with a form that says so — because a number the
// grammar did not read is a number a reader would have had to guess.
func TestSeamLineageReadsTheDeclaredForms(t *testing.T) {
	for _, c := range []struct {
		name, value string
		count       int
		form        string
		read        bool
	}{
		{"ordinal", "`frame::Encode` — 3rd point-fix; trail: a1b2c3d + tracker#204.", 3, "ordinal", true},
		{"ordinal, lower bound", "`snap::Run` — 5th+ point-fix at this seam; trail: x + y.", 5, "ordinal", true},
		{"prior count", "`mitosis.go` — 4 prior point-fixes in the target function; trail: k1 + k2.", 4, "prior-count", true},
		{"prior count, bold, qualified", "`envelope.go::warn` — **1 prior closed code point-fix**: k9.", 1, "prior-count", true},
		{"at least", "`index.go` — ≥2 prior point-fixes at this locus (a deferral; a truncation fix).", 2, "prior-count", true},
		{"count equals", "`plan.go::compute` — accretion emitted at Propose. **Count = 5** closed prior fixes.", 5, "count-eq", true},
		{"no prior accretion", "`area:genealogy` — no prior accretion.", 0, "none-declared", true},
		{"no prior closed point-fixes", "idea-sourced. No prior closed point-fixes at this locus.", 0, "none-declared", true},
		{"a letter is not a count", "`emit.go` — Nth point-fix (N≥3); trail: k1.", 0, "unread", false},
		{"a hypothetical is not a count", "`down.go::classify` — two open fixes would be point-fixes #3–#5 at that locus.", 0, "unread", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := synth(t, "0031", head("0031", "Seam", "Draft")+"- **Seam Lineage**: "+c.value+"\n\n## Problem Statement\n\nSynthetic.\n")
			f := metadataField(t, d, "Seam Lineage")
			if f.Seam == nil {
				t.Fatal("no seam sub-parse on the Seam Lineage field")
			}
			if f.Seam.Form != c.form {
				t.Errorf("form = %q, want %q", f.Seam.Form, c.form)
			}
			if (f.Seam.Count != nil) != c.read {
				t.Fatalf("count read = %v, want %v", f.Seam.Count != nil, c.read)
			}
			if c.read && *f.Seam.Count != c.count {
				t.Errorf("count = %d, want %d", *f.Seam.Count, c.count)
			}
		})
	}
}

// TestSeamLineageDispositionIsReadWhereverItIsWritten: the template
// writes the escape as a nested bullet, the corpus writes it inline as
// the value's own sentence, bold, with or without a parenthetical — and
// an author who copies the template's guidance into the field has NOT
// written one, because there the label sits mid-sentence.
func TestSeamLineageDispositionIsReadWhereverItIsWritten(t *testing.T) {
	const lead = "- **Seam Lineage**: `frame::Encode` — 3rd point-fix; trail: a1b2c3d +\n  tracker#204, tracker#331.\n"
	for _, c := range []struct {
		name, tail string
		want       bool
	}{
		{"inline sentence", "  Accretion disposition: the 3 point-fixes at `frame::Encode` are NOT one\n  missing design decision; cite: 0003.\n", true},
		{"inline, bold, parenthetical", "  **Accretion disposition (written):** the locus carries three siting fixes.\n", true},
		{"nested bullet", "  - Accretion disposition: the prior point-fixes are two halves of one test.\n", true},
		{"nested bullet after a labelled one", "  - **Disposition owner**: this record.\n  - Accretion disposition: one decision, unified here.\n", true},
		{"template guidance copied in", "  - If the count is ≥2 the Profile is floored at `foundational`; the only\n    escape is an accretion disposition written here, of the form:\n    `Accretion disposition: the N point-fixes at <seam> are NOT …`.\n", false},
		{"none written", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := synth(t, "0031", head("0031", "Seam", "Draft")+lead+c.tail+"- **Cluster**: 0032-peer\n\n## Problem Statement\n\nSynthetic.\n")
			f := metadataField(t, d, "Seam Lineage")
			if f.Seam == nil {
				t.Fatal("no seam sub-parse")
			}
			if f.Seam.Disposition != c.want {
				t.Errorf("disposition = %v, want %v", f.Seam.Disposition, c.want)
			}
			if f.Seam.Count == nil || *f.Seam.Count != 3 {
				t.Errorf("the count must survive the tail: got %v", f.Seam.Count)
			}
		})
	}
}

// A seed's bracketed placeholder is not a field to read: no count, no
// disposition, and a form that says why.
func TestSeamLineagePlaceholderIsNotRead(t *testing.T) {
	d := synth(t, "0031", head("0031", "Seam", "Draft")+"- **Seam Lineage**: [seed placeholder — populate from the scope review at propose]\n\n## Problem Statement\n\nSynthetic.\n")
	f := metadataField(t, d, "Seam Lineage")
	if f.Seam == nil || f.Seam.Form != "placeholder" || f.Seam.Count != nil || f.Seam.Disposition {
		t.Errorf("placeholder parsed as %+v", f.Seam)
	}
}
