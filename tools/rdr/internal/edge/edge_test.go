package edge

import (
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
)

// TestReferenceForms pins one case per citation SHAPE the corpus writes.
// Every string here is synthetic — invented numbers, invented names, a
// neutral `proj/` prefix — for the same reason the fixtures are
// (testdata/README.md): this repo is public and the corpus the grammars
// were built against is not. The shapes are what the grammar reads; the
// content is made up.
func TestReferenceForms(t *testing.T) {
	cases := []struct {
		name, in string
		bare     bool
		want     []string
	}{
		{"project-qualified", "proj/0014's exit-code table", false, []string{"proj/0014"}},
		{"rdr-prefixed", "RDR proj/0009 owns the model", false, []string{"proj/0009"}},
		{"rdr-bare-number", "cross-check against RDR 0008", false, []string{"0008"}},
		{"r-dash", "R-0037 is a sibling, not a predecessor", false, []string{"0037"}},
		{"filename-slug", "`0001-frame-length` (width model)", false, []string{"0001"}},
		{"assumption", "proj/0055 A5 settles it", false, []string{"proj/0055:A5"}},
		{"assumption-sectioned", "proj/0032 § A8 rejected it", false, []string{"proj/0032:A8"}},
		{"legacy-CA", "proj/0055 CA-12, confirmed", false, []string{"proj/0055:A12"}},
		{"contract", "proj/0117 C3 binds the writer", false, []string{"proj/0117:C3"}},
		{"decision", "proj/0054 D-4 deferred it", false, []string{"proj/0054:D-4"}},
		{"split-label", "proj/0060 A4b narrows it", false, []string{"proj/0060:A4b"}},
		{"section", "proj/0055 §Normative Contracts", false, []string{"proj/0055:§normative-contracts"}},
		{"section-quoted", `proj/0092 §"The values" list`, false, []string{"proj/0092:§the-values"}},
		{"record-list", "0001, 0002, 0004, 0011", true, []string{"0001", "0002", "0004", "0011"}},
		{"list-with-slugs", "`0001-catalog`, `0014-import`", true, []string{"0001", "0014"}},

		// The forms that must NOT become references.
		{"rfc", "RFC 6962 binary Merkle tree", false, nil},
		{"counts", "upper bound for 2000 rows × 1000 elements", false, nil},
		{"line-number", "walker.go:1306 and :1312", false, nil},
		{"iso-date-in-list", "0035-keyed-bodies (Final 2026-05-19)", true, []string{"0035"}},
		{"version", "since 1.24 the encoder", false, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, r := range FindRefs(c.in, c.bare) {
				got = append(got, r.ID())
			}
			if len(got) != len(c.want) {
				t.Fatalf("FindRefs(%q, %v) = %v, want %v", c.in, c.bare, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("ref %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestSectionNameIsBounded: a `§Name` citation is a heading fragment, not
// the sentence that follows it. Over-reading slugs the prose into the key
// and the edge resolves against nothing.
func TestSectionNameIsBounded(t *testing.T) {
	cases := map[string]string{
		"proj/0009 §Identity stack §1: ElementIDs are randomly minted at birth": "proj/0009:§identity-stack",
		"proj/0035 § Table B line 187 classifies the body":                      "proj/0035:§table-b-line-187-classifies-the",
		"proj/0112 §Normative Contracts L-4":                                    "proj/0112:§normative-contracts",
		"proj/0106 §Normative Contracts I-3/I-4":                                "proj/0106:§normative-contracts",
		"proj/0062 §Critical Assumptions A3/A6":                                 "proj/0062:§critical-assumptions",
	}
	for in, want := range cases {
		refs := FindRefs(in, false)
		if len(refs) == 0 {
			t.Errorf("%q: no reference found", in)
			continue
		}
		if got := refs[0].ID(); got != want {
			t.Errorf("%q\n got %q\nwant %q", in, got, want)
		}
	}
}

// TestREQIsNotAnElement: `REQ-13` is minted per implementation run into a
// req-list artifact over every clause of the record. It is not an element
// of the record, so the citation targets the DOCUMENT and never invents a
// contract the record does not have.
func TestREQIsNotAnElement(t *testing.T) {
	refs := FindRefs("proj/0135 REQ-3 and REQ-4, by named supersession", false)
	if len(refs) == 0 {
		t.Fatal("no reference found")
	}
	if refs[0].Kind != "" {
		t.Errorf("REQ-3 read as element kind %q; it names no element", refs[0].Kind)
	}
	if got := refs[0].ID(); got != "proj/0135" {
		t.Errorf("target = %q, want the document proj/0135", got)
	}
}

// TestNonRecordReferences pins the grammars for the targets that are not
// records: code anchors, artifact paths, trackers and RFDs.
func TestNonRecordReferences(t *testing.T) {
	t.Run("source anchor", func(t *testing.T) {
		for _, in := range []string{
			"`frame::Encode` returns", "internal/proj/drift.go::walk is the arm",
			"see walker.go:319::absorb for the skip",
		} {
			if !SourceAnchorRe.MatchString(in) {
				t.Errorf("%q: no anchor matched", in)
			}
		}
		// A record's element written with the path separator is a record
		// citation, not code.
		m := SourceAnchorRe.FindStringSubmatch("proj/0092::A1 holds")
		if m == nil {
			t.Fatal("expected a raw match to test the guard against")
		}
		if IsSourceAnchor(m[1]) {
			t.Errorf("%q read as a code anchor; it is a record citation", m[0])
		}
		if !IsSourceAnchor("internal/proj/drift.go") {
			t.Error("a real path rejected as an anchor")
		}
	})

	t.Run("artifact", func(t *testing.T) {
		for in, want := range map[string]string{
			"{SPIKE_DIR}/RESULTS.md":          "RESULTS.md",
			"{EVIDENCE_DIR}research/prior.md": "research/prior.md",
			"{ARTIFACT_DIR}/gate.md":          "gate.md",
		} {
			m := ArtifactRe.FindStringSubmatch(in)
			if m == nil {
				t.Errorf("%q: no artifact matched", in)
				continue
			}
			if m[2] != want {
				t.Errorf("%q: path = %q, want %q", in, m[2], want)
			}
		}
	})

	t.Run("issue", func(t *testing.T) {
		for _, in := range []string{"`_issues/0022` § E", "kata **ahg1** tracks it", "kata #71 (tracker)", "kata `54ws` (related)"} {
			if !IssueRe.MatchString(in) {
				t.Errorf("%q: no issue matched", in)
			}
		}
		// A bare `#N` is prose numbering — 635 of the corpus's 1,615 are
		// `principle #N` — and must never be read as a tracker.
		for _, in := range []string{"principle #7: codes are wire format", "item #3", "step #2"} {
			if IssueRe.MatchString(in) {
				t.Errorf("%q read as an issue reference", in)
			}
		}
	})

	t.Run("rfd", func(t *testing.T) {
		for in, want := range map[string]string{
			"per RFD 0004 §3":        "0004",
			"see rfd/0007/README.md": "0007",
		} {
			m := RFDRe.FindStringSubmatch(in)
			if m == nil {
				t.Errorf("%q: no RFD matched", in)
				continue
			}
			if got := m[1] + m[2]; got != want {
				t.Errorf("%q: number = %q, want %q", in, got, want)
			}
		}
	})
}

// TestKindTable: every kind states its target class, and only mentions is
// untyped. A kind added without a class would silently resolve as an
// element reference.
func TestKindTable(t *testing.T) {
	if len(Kinds) != 16 {
		t.Errorf("Kinds has %d entries; the issue's enum has 14, plus surface-of and jdr", len(Kinds))
	}
	seen := map[Kind]bool{}
	for _, k := range Kinds {
		if seen[k] {
			t.Errorf("%s listed twice", k)
		}
		seen[k] = true
		if k.Class() == "" {
			t.Errorf("%s has no target class", k)
		}
		if k.Typed() != (k != Mentions) {
			t.Errorf("%s: Typed() disagrees with the mentions rule", k)
		}
	}
	for _, k := range []Kind{SourceAnchor, Artifact, Issue, RFD} {
		if k.Class() == TargetElement {
			t.Errorf("%s resolves as an element; its target is outside the records dir", k)
		}
	}
}

// TestUnmapped: a reference-shaped span no grammar claimed is reported,
// and a claimed one is not. This is the acceptance criterion that an
// unmapped form lands in warnings rather than in silence.
func TestUnmapped(t *testing.T) {
	v := "proj/0103 §Approach item 3 supersedes; renamed → SplitTable"
	if got := Unmapped(v, nil); len(got) != 2 {
		t.Errorf("unclaimed value: %d unmapped spans, want 2 (the § and the arrow): %q", len(got), got)
	}
	// With the whole string claimed, nothing is unmapped.
	if got := Unmapped(v, [][2]int{{0, len(v)}}); len(got) != 0 {
		t.Errorf("fully claimed value reported %q", got)
	}
	// Ordinary prose with no citation shape is never a warning.
	if got := Unmapped("The encoder writes the length prefix first.", nil); len(got) != 0 {
		t.Errorf("prose reported as unmapped: %q", got)
	}
}

// TestRefID: a reference renders as the ID grammar writes it, and drops
// to its document form on demand.
func TestRefID(t *testing.T) {
	r := Ref{Project: "proj", Record: "0055", Kind: ident.Assumption, Key: "3"}
	if got := r.ID(); got != "proj/0055:A3" {
		t.Errorf("ID = %q", got)
	}
	if got := r.Document(); got != "proj/0055" {
		t.Errorf("Document = %q", got)
	}
	if got := (Ref{Record: "0055"}).ID(); got != "0055" {
		t.Errorf("bare document ID = %q", got)
	}
}

// TestColonIDCitationIsRead: the canonical ID form is the citation the
// linking rule asks authors to write, so the grammar must read it. Before
// element IDs existed the corpus had no way to spell one, and a `0055:C4`
// that parsed as a bare reference to the whole of 0055 would lose the
// element half silently — the exact failure cite-don't-restate exists to
// prevent.
func TestColonIDCitationIsRead(t *testing.T) {
	for _, tc := range []struct {
		in, id string
	}{
		{"see cli/0055:C4 for the shape", "cli/0055:C4"},
		{"rests on cli/0055:A3", "cli/0055:A3"},
		{"0055-frame-widths:C2 fixes it", "0055:C2"},
		{"RDR 0055:MVV covers it", "0055:MVV"},
		{"cli/0055:D-identity is the key", "cli/0055:D-identity"},
		// The older spellings keep working: the colon is an addition,
		// not a replacement.
		{"cli/0055 A5 says so", "cli/0055:A5"},
		{"cli/0055 §Normative Contracts", "cli/0055:§normative-contracts"},
		// A colon that is punctuation rather than an ID separator leaves
		// a document reference, because what follows names no element.
		{"cli/0055: the record that started it", "cli/0055"},
	} {
		refs := FindRefs(tc.in, false)
		if len(refs) == 0 {
			t.Errorf("%q: no reference found", tc.in)
			continue
		}
		if got := refs[0].ID(); got != tc.id {
			t.Errorf("%q: got %s, want %s", tc.in, got, tc.id)
		}
	}
}

// TestSectionSeparatorRuns: a heading's words are separated by more than
// a single space. `Semantic / Per-op — precondition replay` writes a
// spaced slash and an em dash between them, and a one-character
// separator class ends the citation at `Semantic` — which names a
// DIFFERENT thing from what the author wrote, and an ambiguous one where
// the whole name is unique.
func TestSectionSeparatorRuns(t *testing.T) {
	for in, want := range map[string]string{
		"proj/0007 §Semantic / Per-op":             "proj/0007:§semantic-per-op",
		"proj/0007 §Semantic / Fork — merge order": "proj/0007:§semantic-fork-merge-order",
		"proj/0007 §Wire format – the byte layout": "proj/0007:§wire-format-the-byte-layout",
	} {
		refs := FindRefs(in, false)
		if len(refs) == 0 {
			t.Fatalf("%q: no reference found", in)
		}
		if got := refs[0].ID(); got != want {
			t.Errorf("%q\n got %q\nwant %q", in, got, want)
		}
	}
}

// TestSubLocatorIsTrimmed: `point 6` / `item 5` / `step 2` name an item
// INSIDE a section, in the record's own per-section numbering, which is
// below anything this grammar addresses. It is the same thing sectionTail
// already trims in its `L-4` label form, written out in words.
//
// Left in, the words slug into the key and a section that plainly exists
// is reported missing.
func TestSubLocatorIsTrimmed(t *testing.T) {
	for in, want := range map[string]string{
		"proj/0009 § Identity stack point 6":  "proj/0009:§identity-stack",
		"proj/0009 § Identity stack point 1":  "proj/0009:§identity-stack",
		"proj/0013 § Technical Design item 5": "proj/0013:§technical-design",
		"proj/0055 §Approach step 2":          "proj/0055:§approach",
		"proj/0055 §Failure Modes row 3":      "proj/0055:§failure-modes",
		// A label list and a word-form sub-locator at once: both go.
		"proj/0055 §Normative Contracts L-4 point 2": "proj/0055:§normative-contracts",
	} {
		refs := FindRefs(in, false)
		if len(refs) == 0 {
			t.Fatalf("%q: no reference found", in)
		}
		if got := refs[0].ID(); got != want {
			t.Errorf("%q\n got %q\nwant %q", in, got, want)
		}
	}
}

// TestSubLocatorVocabularyIsClosed: the trim fires on the sub-locator
// words and on nothing else. A heading that ENDS in a number is real —
// `Step 2: Layer assignment`, `Phase 1` — and a rule that ate any
// `<word> <number>` tail would clip the last word off it.
func TestSubLocatorVocabularyIsClosed(t *testing.T) {
	for in, want := range map[string]string{
		"proj/0112 §Step 2":            "proj/0112:§step-2",
		"proj/0112 §Phase 1":           "proj/0112:§phase-1",
		"proj/0112 §Encoding Format 2": "proj/0112:§encoding-format-2",
	} {
		refs := FindRefs(in, false)
		if len(refs) == 0 {
			t.Fatalf("%q: no reference found", in)
		}
		if got := refs[0].ID(); got != want {
			t.Errorf("%q\n got %q\nwant %q", in, got, want)
		}
	}
}

// TestGateNamespaceIsClosed: `ident.Gate` keys are the five Finalization
// Gate sub-sections and can never be anything else, because that is where
// every G-element is minted from. A record's OWN `G-a`…`G-j` guard table,
// or a `G-faithful` mode name, is an author namespace that happens to
// share the letter.
//
// Reading those as gate items asserts an element the target cannot have
// under any spelling, and reports a correct citation broken forever. So
// the citation targets the document instead — the answer `REQ-N` already
// gets, for the same reason.
func TestGateNamespaceIsClosed(t *testing.T) {
	t.Run("template gate items resolve as gate elements", func(t *testing.T) {
		for in, want := range map[string]string{
			"proj/0055 G-scope narrows it":     "proj/0055:G-scope",
			"proj/0055 G-contradiction is met": "proj/0055:G-contradiction",
			"proj/0055 G-cross-cutting":        "proj/0055:G-cross-cutting",
		} {
			refs := FindRefs(in, false)
			if len(refs) == 0 {
				t.Fatalf("%q: no reference found", in)
			}
			if refs[0].Kind != ident.Gate {
				t.Errorf("%q: kind = %q, want the gate kind", in, refs[0].Kind)
			}
			if got := refs[0].ID(); got != want {
				t.Errorf("%q: got %q, want %q", in, got, want)
			}
		}
	})
	t.Run("an author's own G-key names the document", func(t *testing.T) {
		for _, in := range []string{
			"a proj/0133 G-c framing refusal",
			"proj/0133 G-a and G-j both hold",
			"proj/0133 G-faithful is the default mode",
		} {
			refs := FindRefs(in, false)
			if len(refs) == 0 {
				t.Fatalf("%q: no reference found", in)
			}
			if refs[0].Kind != "" {
				t.Errorf("%q: read as element kind %q; it names no element of this grammar", in, refs[0].Kind)
			}
			if got := refs[0].ID(); got != "proj/0133" {
				t.Errorf("%q: target = %q, want the document proj/0133", in, got)
			}
		}
	})
	t.Run("a decision slug key is unaffected", func(t *testing.T) {
		refs := FindRefs("proj/0055 D-identity settles it", false)
		if len(refs) == 0 || refs[0].ID() != "proj/0055:D-identity" {
			t.Errorf("D- key changed: %v", refs)
		}
	})
}

// TestRecordSlugIsNotAFilenameSegment: the `NNNN-slug` alternative is the
// one grammar with no marker of its own, and a path in the RDR's own
// evidence tree is that shape one segment in —
// `{EVIDENCE_DIR}research/a5-0118-clause-spans.md`. `\b` does not stop it,
// because a hyphen is a non-word byte.
//
// The edge it minted was a FALSE relation carrying a slug that cannot
// resolve, which reads to a consumer as a broken pointer in a record that
// is not broken.
func TestRecordSlugIsNotAFilenameSegment(t *testing.T) {
	for _, in := range []string{
		"`{EVIDENCE_DIR}research/a5-0118-clause-spans.md`",
		"see {SPIKE_DIR}/c2-0118-probe-results.md",
		"the file v2-0055-widths.json",
	} {
		if refs := FindRefs(in, false); len(refs) != 0 {
			t.Errorf("%q: read %d references; the digits are mid-token in a filename: %v", in, len(refs), refs[0].ID())
		}
	}
	// A record filename has no segment before the number — that is what
	// makes the shape a citation — so these still read.
	for in, want := range map[string]string{
		"`0055-frame-widths.md` covers it":   "0055",
		"{EVIDENCE_DIR}0055-frame-widths.md": "0055",
		"see research/0118-clause-spans.md":  "0118",
	} {
		refs := FindRefs(in, false)
		if len(refs) == 0 {
			t.Errorf("%q: no reference found", in)
			continue
		}
		if got := refs[0].Record; got != want {
			t.Errorf("%q: record = %q, want %q", in, got, want)
		}
	}
}

// TestClauseIsReadInTheColonFormOnly: a contract clause label is an
// element reference in its canonical colon form — the author's exact id,
// the one the template prescribes — and the document or section in every
// spaced spelling the corpus writes, which stay byte-for-byte as they
// were so no record is asked to rewrite a citation that was never wrong.
// The precedence mirrors ident.Parse: decisions, gate keys and the
// ordinal kinds are read first, so `:F-1` is a clause and `:F1` a failure
// mode, and a closed-namespace `:G-a` stays the document.
func TestClauseIsReadInTheColonFormOnly(t *testing.T) {
	t.Run("colon form is the clause", func(t *testing.T) {
		for in, want := range map[string]string{
			"cli/0112:L-3 binds it":            "cli/0112:L-3",
			"see cli/0113:REQ-12a":             "cli/0113:REQ-12a",
			"cli/0112:NC-5 holds":              "cli/0112:NC-5",
			"cli/0112:F-1 is the clause":       "cli/0112:F-1",
			"cli/0112:S-1 is the clause":       "cli/0112:S-1",
			"0112-some-slug:L-3 by slug":       "0112:L-3",
			"RDR cli/0112:I-4 by prefix":       "cli/0112:I-4",
			"cli/0112:L-3, then cli/0112:L-4.": "cli/0112:L-3",
		} {
			refs := FindRefs(in, false)
			if len(refs) == 0 {
				t.Fatalf("%q: no reference found", in)
			}
			if refs[0].Kind != ident.Clause {
				t.Errorf("%q: kind = %q, want clause", in, refs[0].Kind)
			}
			if got := refs[0].ID(); got != want {
				t.Errorf("%q: got %q, want %q", in, got, want)
			}
			if !strings.Contains(refs[0].Raw, ":") {
				t.Errorf("%q: raw %q dropped the element half", in, refs[0].Raw)
			}
		}
	})
	t.Run("the earlier grammars keep precedence", func(t *testing.T) {
		for in, want := range map[string]struct {
			id   string
			kind ident.Kind
		}{
			"cli/0035:D-6 decided it":     {"cli/0035:D-6", ident.Decision},
			"cli/0055:G-scope narrows it": {"cli/0055:G-scope", ident.Gate},
			"cli/0133:G-a is a guard":     {"cli/0133", ""},
			"cli/0112:F1 fails":           {"cli/0112:F1", ident.Failure},
			"cli/0112:S1 scenario":        {"cli/0112:S1", ident.Scenario},
			"cli/0055:CA-5 legacy":        {"cli/0055:A5", ident.Assumption},
		} {
			refs := FindRefs(in, false)
			if len(refs) == 0 {
				t.Fatalf("%q: no reference found", in)
			}
			if refs[0].Kind != want.kind || refs[0].ID() != want.id {
				t.Errorf("%q: got %s (%q), want %s (%q)", in, refs[0].ID(), refs[0].Kind, want.id, want.kind)
			}
		}
	})
	t.Run("spaced spellings are unchanged", func(t *testing.T) {
		for in, want := range map[string]struct {
			id, raw string
		}{
			"cli/0112 L-3 binds it":                       {"cli/0112", "cli/0112"},
			"cli/0119 REQ-89 is minted per run":           {"cli/0119", "cli/0119 REQ-89"},
			"cli/0112: REQ-3 with punctuation":            {"cli/0112", "cli/0112: REQ-3"},
			"cli/0112 §Normative Contracts L-3, the home": {"cli/0112:§normative-contracts", "cli/0112 §Normative Contracts L-3"},
		} {
			refs := FindRefs(in, false)
			if len(refs) != 1 {
				t.Fatalf("%q: refs = %v, want one", in, refs)
			}
			if refs[0].ID() != want.id || refs[0].Raw != want.raw {
				t.Errorf("%q: got %s raw %q, want %s raw %q", in, refs[0].ID(), refs[0].Raw, want.id, want.raw)
			}
			if refs[0].Kind == ident.Clause {
				t.Errorf("%q: a spaced clause spelling was promoted", in)
			}
		}
	})
}
