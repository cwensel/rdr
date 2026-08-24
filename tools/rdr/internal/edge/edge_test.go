package edge

import (
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
			"{SPIKE_DIR}/RESULTS.md":         "RESULTS.md",
			"{EVIDENCE_DIR}research/prior.md": "research/prior.md",
			"{ARTIFACT_DIR}/gate.md":         "gate.md",
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
	if len(Kinds) != 14 {
		t.Errorf("Kinds has %d entries; the issue's enum has 14", len(Kinds))
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
