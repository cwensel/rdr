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
// each under the cap (the orchestrator sends a path and fields, so the size
// is the leaf's read, not the parent's output — the cap is a per-leaf token
// budget, not a protocol limit, and rose to 2304 when Phase 3b took its
// mark/clear pair: 3b is verifier, test author and committer in one leaf,
// so it carries strictly more contract than its siblings); every `{FIELD}` a template
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
		if len(body) > 2304 {
			t.Errorf("%s is %d bytes; the cap is 2304", name, len(body))
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
		// Every leaf marks its worktree and clears it, in the role launch.md's
		// BRIEFS block assigns. The marker is what arms a consumer's guard, so
		// an unmarked leaf is one the guard cannot refuse a raw `cat`, `go
		// test` or `git commit` in: three runs' scorecards split exactly on
		// this line, every marking brief using the helpers and every
		// non-marking one reaching for the raw command.
		role := map[string]string{
			"phase-0.md":        "verifier",
			"phase-1.md":        "test-author",
			"phase-2.md":        "",
			"phase-3a.md":       "verifier",
			"phase-3b.md":       "verifier",
			"phase-3c.md":       "",
			"phase-grounder.md": "verifier",
		}[name]
		want := "rdr-leg-mark {WORKTREE}"
		if role != "" {
			want = "rdr-leg-mark --role " + role + " {WORKTREE}"
		}
		if !strings.Contains(text, want) {
			t.Errorf("%s does not mark its worktree with `%s`; without the mark a consumer's guard is inert for this leaf", name, want)
		}
		if !strings.Contains(text, "`--clear`") {
			t.Errorf("%s marks its worktree but never clears it", name)
		}
	}
}
