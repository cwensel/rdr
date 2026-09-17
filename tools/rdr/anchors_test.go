package main

import (
	"flag"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// anchorsFixture binds the status corpus (record 0021 mints A1, C1 and
// seven outline sections) and names the ledger fixtures: an iteration-1
// critique ledger, its delta-scoped pass 2, and three persona files.
func anchorsFixture(t *testing.T) (records string, file func(string) string) {
	t.Helper()
	records, _ = bindStatusFixture(t)
	base, err := filepath.Abs(filepath.Join("testdata", "anchors"))
	if err != nil {
		t.Fatal(err)
	}
	return records, func(name string) string { return filepath.Join(base, name) }
}

func anchorsLines(t *testing.T, args ...string) []string {
	t.Helper()
	code, out, errb := runCapture(t, append([]string{"anchors"}, args...)...)
	if code != 0 {
		t.Fatalf("recs anchors %s: exit %d: %s", strings.Join(args, " "), code, errb)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// TestAnchorsMembershipIsExact: a token counts only when the record's
// projection mints it. The ledger cites C1, A1 and §proposed-solution
// (minted), C7 (shaped like an id, nothing minted), and 0022:A1 (a peer
// record's id — a citation, not an anchor into this record). The diff
// this feeds is a set operation, so a near-miss must contribute nothing.
func TestAnchorsMembershipIsExact(t *testing.T) {
	_, file := anchorsFixture(t)
	got := anchorsLines(t, "--record", "0021", file("ledger.md"))
	want := []string{"0021:A1", "0021:C1", "0021:§proposed-solution"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ledger anchors = %v, want %v (sorted, unique, minted ids only)", got, want)
	}
	for _, bad := range []string{"0021:C7", "0022:A1"} {
		if hasString(got, bad) {
			t.Errorf("%s was printed; it names nothing this record mints", bad)
		}
	}
}

// TestAnchorsUnresolvedNamesTheDanglingRows: --unresolved is the other
// half of the same read — the tokens shaped like this record's ids that
// name no element it mints, which is what a ledger row points at after
// a reword. A peer record's id is still neither.
func TestAnchorsUnresolvedNamesTheDanglingRows(t *testing.T) {
	_, file := anchorsFixture(t)
	got := anchorsLines(t, "--record", "0021", "--unresolved", file("ledger.md"), file("pass-2.md"))
	want := []string{"0021:C7", "0021:S9"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("unresolved = %v, want %v", got, want)
	}
}

// TestAnchorsDiffIsTheLedgerReconciliation is the composition the prompts
// perform: `comm -13 ledger pass` is the net-new scope and `comm -12` the
// still-open set. Both are computed here over the verb's own output so
// the fixture's shape is pinned: pass 2 keeps A1 and C1 open, raises
// §decision-rationale and §metadata, and §proposed-solution closes.
func TestAnchorsDiffIsTheLedgerReconciliation(t *testing.T) {
	_, file := anchorsFixture(t)
	ledger := anchorsLines(t, "--record", "0021", file("ledger.md"))
	pass := anchorsLines(t, "--record", "0021", file("pass-2.md"))
	inLedger := map[string]bool{}
	for _, id := range ledger {
		inLedger[id] = true
	}
	var netNew, stillOpen []string
	for _, id := range pass {
		if inLedger[id] {
			stillOpen = append(stillOpen, id)
		} else {
			netNew = append(netNew, id)
		}
	}
	if got, want := strings.Join(netNew, ","), "0021:§decision-rationale,0021:§metadata"; got != want {
		t.Errorf("net-new (comm -13) = %s, want %s", got, want)
	}
	if got, want := strings.Join(stillOpen, ","), "0021:A1,0021:C1"; got != want {
		t.Errorf("still-open (comm -12) = %s, want %s", got, want)
	}
	// Byte order, so two outputs comm without a locale in the way.
	for _, lines := range [][]string{ledger, pass} {
		for i := 1; i < len(lines); i++ {
			if lines[i-1] >= lines[i] {
				t.Errorf("not sorted unique bytewise: %q before %q", lines[i-1], lines[i])
			}
		}
	}
}

// TestAnchorsHotspotsAreACount: the 3amigo consolidation is "which ids
// two or more personas named". Over the three persona files, C1 is named
// by all three and nothing else by more than one — the set intersection
// the prompt used to describe, as a count over this verb's lines.
func TestAnchorsHotspotsAreACount(t *testing.T) {
	_, file := anchorsFixture(t)
	count := map[string]int{}
	for _, p := range []string{"persona-1.md", "persona-2.md", "persona-3.md"} {
		for _, id := range anchorsLines(t, "--record", "0021", file(p)) {
			count[id]++
		}
	}
	var hot []string
	for id, n := range count {
		if n >= 2 {
			hot = append(hot, id)
		}
	}
	if len(hot) != 1 || hot[0] != "0021:C1" || count["0021:C1"] != 3 {
		t.Errorf("hotspots = %v (C1 named %d times), want exactly 0021:C1 named 3 times", hot, count["0021:C1"])
	}
}

// TestAnchorsRefusesRatherThanGuessing: no record, no files, or a file
// that cannot be read is a stated stop, never an empty success — an
// empty set here would read as "converged".
func TestAnchorsRefusesRatherThanGuessing(t *testing.T) {
	_, file := anchorsFixture(t)
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"no record", []string{file("ledger.md")}, "stopped:usage"},
		{"no files", []string{"--record", "0021"}, "stopped:usage"},
		{"unreadable file", []string{"--record", "0021", file("nope.md")}, "stopped:unreadable"},
		{"unresolvable record", []string{"--record", "0999", file("ledger.md")}, "stopped:"},
	} {
		code, out, errb := runCapture(t, append([]string{"anchors"}, c.args...)...)
		if code != 2 || !strings.Contains(errb, c.want) {
			t.Errorf("%s: exit %d, stderr %q, want 2 and %s", c.name, code, errb, c.want)
		}
		if strings.Contains(out, "0021:") {
			t.Errorf("%s: ids were printed on a refused call: %q", c.name, out)
		}
	}
}

// TestAnchorsNamesItselfInTheUsageLog: the verb exists to retire the
// prose "reconcile by passage anchor", and a facet the log cannot name
// reads as never called.
func TestAnchorsNamesItselfInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		args []string
		argc int
		want string
	}{
		{[]string{"-record", "0021"}, 1, "found"},
		{[]string{"-record", "0021"}, 3, "found:files"},
		{[]string{"-record", "0021", "-unresolved"}, 1, "unresolved"},
		{[]string{"-record", "0021", "-unresolved"}, 2, "unresolved:files"},
	} {
		fs := flag.NewFlagSet("anchors", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("anchors", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		f.argc = c.argc
		if got := usageFacet("anchors", f, "x.md"); got != c.want {
			t.Errorf("anchors %v over %d files logs as %q, want %q", c.args, c.argc, got, c.want)
		}
	}
}
