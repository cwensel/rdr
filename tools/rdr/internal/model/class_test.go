package model

import (
	"path/filepath"
	"strings"
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

// TestClassFromTitleWhenNoRootBound: a copy read outside its bindings is
// still judged against its own tier's template.
//
// The root stays the primary authority — it is a fact the seam states,
// where a title is a string an author typed. But an unbound root is not an
// absent document class; it is a caller that has not bound the seam, and
// every file then reads as an RDR. Right for a consumer that never adopted
// the other tiers, wrong for a COPY of this repo: a registry judged
// against the record template reports eleven findings about sections it is
// defined by not having.
func TestClassFromTitleWhenNoRootBound(t *testing.T) {
	SetClassRoots("", "")
	t.Cleanup(func() { SetClassRoots("", "") })

	for _, tc := range []struct {
		path, title string
		want        DocClass
		why         string
	}{
		{"/tmp/copy/jdr/cli/0002-refusals.md", "# JDR cli/0002 What does retrofit promise?",
			ClassJDR, "the title declares the tier the unbound root cannot"},
		{"/tmp/copy/rfd/0001/README.md", "# RFD 0001 Document Tiers: RFD, JDR, RDR",
			ClassRFD, "same, for a capability"},
		{"/tmp/copy/rdr/cli/0138-advisory.md", "# RDR 0138 Advisory identity",
			ClassRDR, "an RDR title agrees with the default"},
		{"/tmp/copy/rdr/cli/0138-advisory.md", "# 0138 Advisory identity",
			ClassRDR, "a title naming no tier leaves the root to answer"},
		{"/tmp/copy/rdr/cli/0138-advisory.md", "",
			ClassRDR, "no title at all is the pre-existing answer"},
	} {
		got, err := ClassOfDoc(tc.path, tc.title)
		if err != nil {
			t.Errorf("ClassOfDoc(%q, %q) errored: %v", tc.path, tc.title, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ClassOfDoc(%q, %q) = %q, want %q — %s", tc.path, tc.title, got, tc.want, tc.why)
		}
	}
}

// TestBoundRootDisagreeingWithTitleStops: two authorities claiming one
// file, and the projector refuses to choose.
//
// The file is in the wrong tree or its title is wrong. Either way judging
// it means believing one authority over the other, which is the guess the
// never-guess rule forbids — and guessing wrong here mis-judges a whole
// document rather than one reference. The reason names both readings.
func TestBoundRootDisagreeingWithTitleStops(t *testing.T) {
	SetClassRoots("/repo/jdr", "/repo/rfd")
	t.Cleanup(func() { SetClassRoots("", "") })

	// A registry sitting in the RFD tree.
	_, err := ClassOfDoc("/repo/rfd/0006/README.md", "# JDR cli/0002 What does retrofit promise?")
	if err == nil {
		t.Fatal("a JDR title under the RFD root must stop; the projector cannot judge it either way")
	}
	for _, want := range []string{"stopped:class-disagreement", "rfd", "jdr"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("reason %q does not name %q; the author must see both readings", err, want)
		}
	}

	// Agreement is silent, under either root.
	for _, tc := range []struct{ path, title string }{
		{"/repo/rfd/0001/README.md", "# RFD 0001 Document Tiers"},
		{"/repo/jdr/cli/0001-data-corpus.md", "# JDR cli/0001 What classifies a row?"},
	} {
		if _, err := ClassOfDoc(tc.path, tc.title); err != nil {
			t.Errorf("ClassOfDoc(%q, %q) errored on agreement: %v", tc.path, tc.title, err)
		}
	}

	// A record outside both roots keeps its default, whatever its title
	// says, because no root is claiming it.
	if c, err := ClassOfDoc("/repo/rdr/cli/0138-a.md", "# RDR 0138 A record"); err != nil || c != ClassRDR {
		t.Errorf("ClassOfDoc(record) = %q, %v; want rdr with no error", c, err)
	}
}
