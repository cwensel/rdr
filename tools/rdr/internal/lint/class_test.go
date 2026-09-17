package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/model"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// jdrFixture writes a registry under a JDR root and binds the class
// roots to it, restoring them when the test ends.
func jdrFixture(t *testing.T, name, body string) *scan.Document {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "jdr", "cli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	model.SetClassRoots(filepath.Join(root, "jdr"), filepath.Join(root, "rfd"))
	t.Cleanup(func() { model.SetClassRoots("", "") })
	d, err := scan.File(p, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestJDRIsNotJudgedByTheRDRTemplate: the category error this rule
// exists to stop. jdr/cli/0001 conforms to jdr/TEMPLATE.md exactly and
// was reported as missing eleven RDR sections — Critical Assumptions,
// Proposed Solution, Implementation Plan and eight more it is DEFINED by
// not having. Every one of those findings was wrong.
func TestJDRIsNotJudgedByTheRDRTemplate(t *testing.T) {
	d := jdrFixture(t, "0001-data-corpus.md", `---
state: open
rfd: 0004
inherits: RFD 0004 DX-1..DX-18
---

# JDR cli/0001 What classifies a row?

## Problem statement

Synthetic.

## Principles

- **P-1** MUST name the chain key.

## Interface record

## DX-13 — the band

The band is the chain key's.

## What this does not decide

Local items.
`)
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "template:missing-section" {
			t.Errorf("template:missing-section %q on a JDR — the RDR template is not this document's", f.Element)
		}
	}
}

// TestJDRMissingItsOwnSectionIsStillReported: selecting the template by
// class must not turn the rule off. A registry that omits a section
// jdr/TEMPLATE.md requires is still behind its own template, and saying
// nothing would trade eleven wrong findings for zero right ones.
func TestJDRMissingItsOwnSectionIsStillReported(t *testing.T) {
	d := jdrFixture(t, "0002-thin.md", `---
state: open
---

# JDR cli/0002 A registry with no problem statement

## Principles

- **P-1** MUST name something.
`)
	var missing []string
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "template:missing-section" {
			missing = append(missing, f.Element)
		}
	}
	if len(missing) == 0 {
		t.Fatal("no template:missing-section on a registry missing its own template's sections")
	}
	// Whatever jdr/TEMPLATE.md requires, it is the JDR's sections that
	// are named and never the RDR's.
	for _, name := range missing {
		for _, rdrOnly := range []string{"Critical Assumptions", "Proposed Solution", "Implementation Plan", "Metadata"} {
			if name == rdrOnly {
				t.Errorf("missing section %q is the RDR template's, not the JDR's", name)
			}
		}
	}
}

// TestRDRStillJudgedByTheRDRTemplate: the default is unchanged. A file
// under no other tier's root is a record, which is the answer every
// caller saw before the class existed and the answer every consumer that
// binds no JDR root still gets.
func TestRDRStillJudgedByTheRDRTemplate(t *testing.T) {
	model.SetClassRoots("", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "0001-thin.md")
	if err := os.WriteFile(p, []byte("# Recommendation 0001: Thin\n\n## Problem Statement\n\nSynthetic.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := scan.File(p, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "template:missing-section" {
			n++
		}
	}
	if n == 0 {
		t.Fatal("a bare record produced no template:missing-section — the RDR rule stopped running")
	}
}

// TestInheritedTextIsNotRestyled: the citation:form rule is a migration
// for text the author wrote. Text a registry HOISTED is someone else's,
// preserved verbatim by declaration, and rewriting a citation inside it
// edits the one document whose job is to have not changed it.
func TestInheritedTextIsNotRestyled(t *testing.T) {
	d := jdrFixture(t, "0003-hoisted.md", `---
state: open
rfd: 0004
inherits: RFD 0004 DX-1..DX-18
---

# JDR cli/0003 A registry with hoisted text

## Problem statement

Synthetic.

## Principles

- **P-1** MUST name the chain key.

## Interface record

## DX-13 — the band

Hoisted: the floor cli/0145 C7 states is the band's.

### Hoisted from RFD 0004 §3c — decisions recorded at cluster reconcile

- **DX-19** ` + "`open`" + ` — the screen grain. See cli/0143 C2 for staleness.

## Author's own reading

This paragraph is the registry's own, and cli/0142 A10 in it is the
author's citation to restyle.
`)
	// citation:form only ever speaks about an edge that ALREADY resolves,
	// so the cited record has to exist for the positive half of this test
	// to be reachable at all.
	target := scan.Bytes([]byte("# Recommendation 0142: The Target\n\n## Metadata\n\n"+
		"- **Status**: Implemented\n- **Date**: 2026-08-01\n\n"+
		"## Critical Assumptions\n\n- **A10 Tenth.**\n"), scan.Options{Project: "cli"})
	docs := []*scan.Document{d, target}
	scan.NewResolver(docs, "").ResolveAll(docs)

	byLine := map[int]bool{}
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "citation:form" {
			byLine[f.LineStart] = true
		}
	}
	src, err := os.ReadFile(d.Path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(src), "\n")
	for i, text := range lines {
		line := i + 1
		switch {
		case strings.Contains(text, "cli/0145 C7"), strings.Contains(text, "cli/0143 C2"):
			if byLine[line] {
				t.Errorf("line %d (%q) got citation:form — it is hoisted text", line, text)
			}
		case strings.Contains(text, "cli/0142 A10"):
			if !byLine[line] {
				t.Errorf("line %d (%q) got no citation:form — the registry's own prose is migratable", line, text)
			}
		}
	}
}


// rfdJdrFixture writes an RFD and a registry that inherits from it, binds
// both roots, and returns a record scanned against them.
func rfdJdrFixture(t *testing.T, body string) *scan.Document {
	t.Helper()
	root := t.TempDir()
	rfd := filepath.Join(root, "rfd", "0004")
	jdr := filepath.Join(root, "jdr", "cli")
	for _, d := range []string{rfd, jdr} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(rfd, "README.md"),
		[]byte("# RFD 0004 Getting data in\n\n## 3 Mechanisms\n\n| DX-6 | a row |\n| DX-13 | another |\n| DX-18 | a third |\n"),
		0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jdr, "0001-data-corpus.md"), []byte(`---
state: open
inherits: RFD 0004 DX-1..DX-18
---

# JDR cli/0001 What classifies a row?

## DX-6 — the domain
`), 0o600); err != nil {
		t.Fatal(err)
	}
	scan.SetJDRRoot(filepath.Join(root, "jdr"))
	scan.SetRFDRoot(filepath.Join(root, "rfd"))
	model.SetClassRoots(filepath.Join(root, "jdr"), filepath.Join(root, "rfd"))
	t.Cleanup(func() {
		scan.SetJDRRoot("")
		scan.SetRFDRoot("")
		model.SetClassRoots("", "")
	})
	d := scan.Bytes([]byte(body), scan.Options{Project: "cli"})
	docs := []*scan.Document{d}
	scan.NewResolver(docs, "").ResolveAll(docs)
	return d
}

// TestRegistryCitationFix: the migration the alias makes optional. A
// citation that spells out the RFD is substitutable and carries a patch
// with the byte span it rewrites; the spellings that are not are reported
// and left to a hand pass.
func TestRegistryCitationFix(t *testing.T) {
	d := rfdJdrFixture(t, `# Recommendation 0107: Registry Citations

## Metadata

- **Status**: Implemented
- **Date**: 2026-08-01

## Problem Statement

The band RFD 0004 DX-6 fixes is the chain key's.
The clause RFD 0004 DX-6/8/9/14 names four at once.
An enumeration names DX-13 bare after its head.
`)
	byLine := map[int]Finding{}
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "citation:registry-form" {
			byLine[f.LineStart] = f
		}
	}
	if len(byLine) == 0 {
		t.Fatal("no citation:registry-form findings — the rule did not run")
	}

	// The substitutable spelling: patched, with the span it rewrites.
	f, ok := byLine[10]
	if !ok {
		t.Fatal("no finding on the substitutable citation")
	}
	if f.Patch == nil {
		t.Fatal("the substitutable citation carries no patch")
	}
	if !strings.Contains(f.Patch.Text, "JDR cli/0001 §DX-6") {
		t.Errorf("patch text = %q, want the JDR spelling", f.Patch.Text)
	}
	if strings.Contains(f.Patch.Text, "RFD 0004 DX-6") {
		t.Errorf("patch text still carries the old spelling: %q", f.Patch.Text)
	}
	// The span names the bytes the repair touches, and nothing more.
	line := d.Line(10)
	if got := line[f.Patch.ByteStart:f.Patch.ByteEnd]; got != "RFD 0004 DX-6" {
		t.Errorf("byte span covers %q, want the citation exactly", got)
	}

	// THE COMPOUND FORM IS NOT SUBSTITUTABLE. Replacing its first anchor
	// leaves `JDR cli/0001 §DX-6/8/9/14`, which erases three citations
	// into one anchor.
	if f := byLine[11]; f.Patch != nil {
		t.Errorf("compound citation was patched to %q — the tail is not part of the anchor", f.Patch.Text)
	}

	// THE BARE FORM IS NOT SUBSTITUTABLE either: the replacement names a
	// document, and the enumeration's head already named one.
	if f := byLine[12]; f.Patch != nil {
		t.Errorf("bare citation was patched to %q", f.Patch.Text)
	}
}

// TestRegistryCitationPatchIsOnePerLine: two citations on one line share
// ONE patch, already carrying both substitutions.
//
// A patch computed per citation against the original line makes the
// second discard the first when both are applied, so one citation stays
// unmigrated — silently, because both applied cleanly. On the live corpus
// that cost record 0113 a citation and made the patch set fail to be a
// fixpoint: a second run found what the first had dropped.
func TestRegistryCitationPatchIsOnePerLine(t *testing.T) {
	d := rfdJdrFixture(t, `# Recommendation 0108: Two On One Line

## Metadata

- **Status**: Implemented
- **Date**: 2026-08-01

## Problem Statement

The home is RFD 0004 DX-6 and also RFD 0004 DX-18 on one line.
`)
	var patches []*Patch
	for _, f := range Run(d, Options{}).Findings {
		if f.Code == "citation:registry-form" && f.Patch != nil {
			patches = append(patches, f.Patch)
		}
	}
	if len(patches) != 2 {
		t.Fatalf("got %d patched findings, want 2 (one per citation)", len(patches))
	}
	if patches[0] != patches[1] {
		t.Error("the two findings carry different patch objects; an applier would apply the line twice")
	}
	text := patches[0].Text
	for _, want := range []string{"JDR cli/0001 §DX-6", "JDR cli/0001 §DX-18"} {
		if !strings.Contains(text, want) {
			t.Errorf("shared patch %q is missing %q — applying it would drop a citation", text, want)
		}
	}
	if strings.Contains(text, "RFD 0004") {
		t.Errorf("shared patch %q still carries an unmigrated citation", text)
	}
}
