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

