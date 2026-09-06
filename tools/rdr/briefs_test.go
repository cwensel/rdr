package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestBriefTemplatesAreFilledNotAuthored pins the contract launch.md's
// BRIEFS block makes with prompts/implementation/briefs/: seven templates,
// each under 2KB (the orchestrator sends a path and fields, so the size is
// the leaf's read, not the parent's output); every `{FIELD}` a template
// uses is declared on its `Fields:` line, so the orchestrator can fill it
// from that line alone; and every launch.md block a template cites by its
// heading line still starts a line there — a renamed PHASE heading would
// otherwise silently hand a leaf an empty task.
func TestBriefTemplatesAreFilledNotAuthored(t *testing.T) {
	home, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	launch, err := os.ReadFile(filepath.Join(home, "prompts", "implementation", "launch.md"))
	if err != nil {
		t.Fatal(err)
	}
	lineLeading := map[string]bool{}
	for _, l := range strings.Split(string(launch), "\n") {
		lineLeading[l] = true
	}
	startsLine := func(anchor string) bool {
		for l := range lineLeading {
			if strings.HasPrefix(l, anchor) {
				return true
			}
		}
		return false
	}

	field := regexp.MustCompile(`\{([A-Z_]+)\}`)
	cited := regexp.MustCompile("`(PHASE [0-9][a-d]?( —)?|COMPLETION GATE)`")
	dir := filepath.Join(home, "prompts", "implementation", "briefs")
	want := []string{"phase-0.md", "phase-1.md", "phase-2.md", "phase-3a.md", "phase-3b.md", "phase-3c.md", "phase-grounder.md"}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	sort.Strings(got)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("briefs/ holds %v, want exactly %v", got, want)
	}
	for _, name := range want {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if len(body) > 2048 {
			t.Errorf("%s is %d bytes; the cap is 2048", name, len(body))
		}
		text := string(body)
		var declared []string
		for _, l := range strings.Split(text, "\n") {
			if strings.HasPrefix(l, "Fields: ") {
				declared = strings.Fields(strings.TrimPrefix(l, "Fields: "))
			}
		}
		if declared == nil {
			t.Errorf("%s has no `Fields:` line", name)
			continue
		}
		isDeclared := map[string]bool{}
		for _, d := range declared {
			isDeclared[d] = true
		}
		for _, m := range field.FindAllStringSubmatch(text, -1) {
			if m[1] == "NAME" { // the field-note's own `{NAME}`, the placeholder for a placeholder
				continue
			}
			if !isDeclared[m[1]] {
				t.Errorf("%s uses {%s} but its Fields line does not declare it", name, m[1])
			}
		}
		for _, m := range cited.FindAllStringSubmatch(text, -1) {
			if !startsLine(m[1]) {
				t.Errorf("%s cites launch.md block %q, which no line there starts with", name, m[1])
			}
		}
		if !strings.Contains(text, "verdict: ") || !strings.Contains(text, "summary_50w: ") {
			t.Errorf("%s does not carry the §return-packet", name)
		}
	}
}
