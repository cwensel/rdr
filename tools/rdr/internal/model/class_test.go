package model

import (
	"path/filepath"
	"testing"
)

// TestClassOfReadsTheRoot: the class is a fact about where a file sits,
// never a guess from its headings. An RFD's `Problem Statement` and a
// JDR's `Problem statement` differ by one letter, so content would be a
// coin flip where the root is a declaration.
func TestClassOfReadsTheRoot(t *testing.T) {
	base := t.TempDir()
	jdr := filepath.Join(base, "jdr")
	rfd := filepath.Join(base, "rfd")
	SetClassRoots(jdr, rfd)
	t.Cleanup(func() { SetClassRoots("", "") })

	for _, tc := range []struct {
		path string
		want DocClass
		why  string
	}{
		{filepath.Join(jdr, "cli", "0001-data-corpus.md"), ClassJDR, "under the JDR root"},
		{filepath.Join(rfd, "0004", "README.md"), ClassRFD, "under the RFD root"},
		{filepath.Join(base, "rdr", "cli", "0138-x.md"), ClassRDR, "under neither: a record"},
		{filepath.Join(base, "jdrs", "0001-x.md"), ClassRDR, "`jdrs` is not inside `jdr` — the match is component-aligned"},
	} {
		if got := ClassOf(tc.path); got != tc.want {
			t.Errorf("ClassOf(%q) = %q, want %q — %s", tc.path, got, tc.want, tc.why)
		}
	}

	// UNBOUND ROOTS MEAN EVERYTHING IS A RECORD. That is the answer every
	// caller saw before the class existed, and the answer a consumer that
	// has not adopted the other two tiers still gets — not a new failure.
	SetClassRoots("", "")
	if got := ClassOf(filepath.Join(jdr, "cli", "0001-data-corpus.md")); got != ClassRDR {
		t.Errorf("unbound roots gave %q, want %q", got, ClassRDR)
	}
}

// TestClassRequiredSectionsPerTier: each tier's list comes from its own
// TEMPLATE.md, and the three lists are genuinely different documents'.
func TestClassRequiredSectionsPerTier(t *testing.T) {
	rdr := ClassRequiredSections(ClassRDR)
	jdr := ClassRequiredSections(ClassJDR)
	rfd := ClassRequiredSections(ClassRFD)

	for _, tc := range []struct {
		name string
		got  []string
	}{{"rdr", rdr}, {"jdr", jdr}, {"rfd", rfd}} {
		if len(tc.got) == 0 {
			t.Errorf("%s required sections empty — the template was not read", tc.name)
		}
	}

	// The RDR spine is the RDR's. A registry that lacked these was told
	// eleven times over that it was incomplete.
	for _, name := range []string{"Critical Assumptions", "Proposed Solution", "Implementation Plan"} {
		if has(jdr, name) {
			t.Errorf("JDR requires %q, which is the RDR template's section", name)
		}
		if has(rfd, name) {
			t.Errorf("RFD requires %q, which is the RDR template's section", name)
		}
	}

	// A SCAFFOLD IS NOT A REQUIRED SECTION. jdr/TEMPLATE.md writes `## D1
	// — [The fork, as a question]` as one entry's shape; a registry whose
	// entries are D2 and D3 is not missing D1.
	for _, tc := range []struct {
		name string
		got  []string
	}{{"jdr", jdr}, {"rfd", rfd}} {
		for _, s := range tc.got {
			if containsBracket(s) {
				t.Errorf("%s required section %q is a scaffold slot, not a section every document owes", tc.name, s)
			}
		}
	}
}

func has(list []string, name string) bool {
	for _, s := range list {
		if s == name {
			return true
		}
	}
	return false
}

func containsBracket(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '[' {
			return true
		}
	}
	return false
}
