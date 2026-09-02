package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// TestHunkOverlapIsInclusiveAtBothEnds pins the overlap rule the prose
// used to state: a hunk touches an element when any post-image line
// falls inside the element's closed range. A blank length is 1, and a
// pure deletion (`+c,0`) is the point between c and c+1, so it touches
// the element ending at c or beginning at c+1 and nothing in between.
func TestHunkOverlapIsInclusiveAtBothEnds(t *testing.T) {
	el := func(id string, a, b int) scan.Element { return scan.Element{ID: id, LineStart: a, LineEnd: b} }
	doc := &scan.Document{Elements: []scan.Element{
		el("X:E5", 5, 10), el("X:E10", 10, 12), el("X:E15", 15, 19), el("X:E20", 20, 20), el("X:E21", 21, 25),
	}}
	cases := []struct {
		name string
		diff string
		want []string
	}{
		{"boundary line shared by two ranges", "@@ -10,1 +10,1 @@", []string{"X:E5", "X:E10"}},
		{"interior", "@@ -7 +7,2 @@", []string{"X:E5"}},
		{"pure deletion between 20 and 21", "@@ -20,3 +20,0 @@", []string{"X:E20", "X:E21"}},
		{"no overlap", "@@ -13 +13,2 @@", []string{}},
		{"one hunk spanning two elements", "@@ -18 +18,3 @@", []string{"X:E15", "X:E20"}},
		{"blank length is 1", "@@ -12 +12 @@", []string{"X:E10"}},
		{"two hunks, ids once each in line order", "@@ -24 +24 @@\n@@ -6 +6 @@", []string{"X:E5", "X:E21"}},
	}
	for _, c := range cases {
		hunks, err := parseHunks(c.diff)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := touchedIDs(doc, hunks); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestMalformedHunkHeaderIsAnError: a header the pattern does not read
// stops the call rather than dropping the hunk — a silently shorter set
// would scope a re-entry short.
func TestMalformedHunkHeaderIsAnError(t *testing.T) {
	if _, err := parseHunks("@@ -3,2 +x,1 @@"); err == nil {
		t.Error("a malformed @@ header must be an error")
	}
	if _, err := parseHunks("@@ -3,2 +4,1 @@ context\n@@ garbage"); err == nil {
		t.Error("a malformed second header must be an error")
	}
	hunks, err := parseHunks("diff --git a/x b/x\n--- a/x\n+++ b/x\n")
	if err != nil || len(hunks) != 0 {
		t.Errorf("a diff with no hunks is the empty set, not an error: %v %v", hunks, err)
	}
}

// TestTouchedIsReadAgainstTheFixture pins the facet over a real element
// table: the synthetic diff in testdata/touched names an interior edit
// (A1), a pure deletion at a decision's first line (D-naming, not the
// decision that ends the line before), and a blank-length add (S1);
// every section holding a touched line is reported beside its elements.
func TestTouchedIsReadAgainstTheFixture(t *testing.T) {
	diff, err := os.ReadFile(fixturePath("touched/hunks.diff"))
	if err != nil {
		t.Fatal(err)
	}
	hunks, err := parseHunks(string(diff))
	if err != nil {
		t.Fatal(err)
	}
	if want := []hunk{{38, 3}, {87, 0}, {210, 1}}; !reflect.DeepEqual(hunks, want) {
		t.Fatalf("hunks %v, want %v", hunks, want)
	}
	doc, err := scan.File(fixturePath("current-shape.md"), scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0004:A1", "0004:D-naming", "0004:S1",
		"0004:§title", "0004:§critical-assumptions", "0004:§proposed-solution", "0004:§technical-design",
		"0004:§load-bearing-decisions", "0004:§validation", "0004:§testing-strategy"}
	if got := touchedIDs(doc, hunks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

// TestTouchedNamesTheClauseAndItsContract: a clause's range lies inside
// its contract's, so an edit to the clause line reports both ids — the
// caller reads either without re-walking.
func TestTouchedNamesTheClauseAndItsContract(t *testing.T) {
	fence := "```"
	body := "# Recommendation 0022: Bands\n\n## Metadata\n\n- **Status**: Draft\n\n" +
		"#### Normative Contracts\n\n**C1**\n\n" +
		fence + "normative\nL-1  input := owned rows\nL-2  the BandHold rule keeps every row\n" +
		"     in its band until release\n" + fence + "\n"
	path := filepath.Join(t.TempDir(), "0022-bands.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := scan.File(path, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	hunks, _ := parseHunks("@@ -13 +13 @@")
	got := touchedIDs(doc, hunks)
	if !hasString(got, "0022:L-2") || !hasString(got, "0022:C1") || hasString(got, "0022:L-1") {
		t.Errorf("want the clause and its contract, not the sibling clause: %v", got)
	}
}

// TestTouchedSinceReadsGit runs the facet end to end over a temp
// repository: the committed fixture, one edit in the working tree, and
// the ids that edit touched — with the hunks beside them as evidence. A
// rev git cannot diff is a stop, never an empty set, and crossing the
// flag with --select has no one answer.
func TestTouchedSinceReadsGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	src, err := os.ReadFile(fixturePath("current-shape.md"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "0004-frame-checksum-algorithm.md")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", ".")
	git("commit", "-q", "-m", "docs(rdr): finalize 0004")

	// Rewrite one line inside A2 (44-50) and append a line after S1.
	lines := strings.Split(string(src), "\n")
	lines[46] = lines[46] + " (edited)"
	edited := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errb := runCapture(t, "inspect", "--json", "--touched-since", "HEAD", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var rep struct {
		Record  string `json:"record"`
		Rev     string `json:"rev"`
		Touched []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"touched"`
		Hunks []hunk `json:"hunks"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Record != "0004" || rep.Rev != "HEAD" || len(rep.Hunks) != 1 || rep.Hunks[0] != (hunk{47, 1}) {
		t.Errorf("identity and hunks: %s", out)
	}
	if len(rep.Touched) == 0 || rep.Touched[0].ID != "0004:A2" || rep.Touched[0].Kind != "A" {
		t.Errorf("the edited assumption leads the touched rows: %s", out)
	}
	if !strings.Contains(out, `"0004:§critical-assumptions"`) {
		t.Errorf("the section holding the edit is reported beside it: %s", out)
	}

	code, out, _ = runCapture(t, "inspect", "--touched-since", "HEAD", path)
	if code != 0 || !strings.HasPrefix(out, "  0004:A2 ") {
		t.Errorf("text rows lead with the element id: exit %d\n%s", code, out)
	}

	// Committed and unchanged: an empty set beside the evidence, exit 0.
	git("commit", "-q", "-am", "docs(rdr): refine 0004")
	code, out, _ = runCapture(t, "inspect", "--json", "--touched-since", "HEAD", path)
	if code != 0 || !strings.Contains(out, `"touched": []`) || !strings.Contains(out, `"hunks": []`) {
		t.Errorf("no change: exit %d\n%s", code, out)
	}
	// From the parent, the same edit is touched again: rev is the floor.
	code, out, _ = runCapture(t, "inspect", "--json", "--touched-since", "HEAD^", path)
	if code != 0 || !strings.Contains(out, `"0004:A2"`) {
		t.Errorf("HEAD^ sees the committed edit: exit %d\n%s", code, out)
	}

	code, _, errb = runCapture(t, "inspect", "--touched-since", "no-such-rev", path)
	if code != 2 || !strings.Contains(errb, "stopped:no-diff") {
		t.Errorf("a rev git cannot diff stops: exit %d, stderr %q", code, errb)
	}
	code, _, errb = runCapture(t, "inspect", "--touched-since", "HEAD", "--select", "0004:A2", path)
	if code != 2 || !strings.Contains(errb, "stopped:usage") {
		t.Errorf("--touched-since with --select: exit %d, stderr %q", code, errb)
	}
	code, _, errb = runCapture(t, "inspect", "--touched-since", "HEAD", "--filter", "metadata", path)
	if code != 2 || !strings.Contains(errb, "stopped:usage") {
		t.Errorf("--touched-since with --filter metadata: exit %d, stderr %q", code, errb)
	}
}
