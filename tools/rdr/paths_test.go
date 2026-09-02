package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pathsFixture builds an evidence tree with the iteration shapes the
// reference corpus actually holds, and points the seam at it. The shapes
// are the point: each one is a case the hand-built prose got to decide
// per site, and each is a case this verb now has to decide once.
func pathsFixture(t *testing.T) (records, evidence string) {
	t.Helper()
	recs, ev := t.TempDir(), t.TempDir()

	body := "# Recommendation 0007: Iteration shapes\n\n## Metadata\n\n" +
		"- **Status**: Draft\n- **Profile**: mid\n"
	if err := os.WriteFile(filepath.Join(recs, "0007-iteration-shapes.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	mk := func(rel string) {
		if err := os.MkdirAll(filepath.Join(ev, filepath.FromSlash(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	touch := func(rel string) {
		full := filepath.Join(ev, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const slug = "0007-iteration-shapes"
	// critique: deep and contiguous — loose iteration 1 plus iter-2..4.
	touch(slug + "/evidence/critique/critique.md")
	mk(slug + "/evidence/critique/iter-2")
	mk(slug + "/evidence/critique/iter-3")
	mk(slug + "/evidence/critique/iter-4")
	// 3amigo: loose files only. That set IS iteration 1, so next is 2.
	touch(slug + "/evidence/3amigo/consolidation.md")
	// cove: the directory exists but is empty — nothing written yet.
	mk(slug + "/evidence/cove")
	// A cluster with a GAP: iter-3 and no iter-2, which the reference
	// corpus really holds at 0122-0123-0130-0131-0132.
	touch("cluster-reconcile/0007-0008/reconcile-report.md")
	mk("cluster-reconcile/0007-0008/iter-3")

	t.Setenv("RDR_RECORDS", recs)
	t.Setenv("RDR_EVIDENCE", ev)
	t.Setenv("RDR_SOURCE_REPO", "")
	return recs, ev
}

// pathsJSON runs the verb and decodes it, so assertions read fields
// rather than parse text.
func pathsJSON(t *testing.T, args ...string) pathsAnswer {
	t.Helper()
	full := append([]string{"paths", "--json", "--facts", factTableForTest(t)}, args...)
	code, out, errb := runCapture(t, full...)
	if code != 0 {
		t.Fatalf("rdr %s: exit %d: %s", strings.Join(full, " "), code, errb)
	}
	var a pathsAnswer
	if err := json.Unmarshal([]byte(out), &a); err != nil {
		t.Fatalf("undecodable answer: %v\n%s", err, out)
	}
	return a
}

// TestPathsBindsTheSameDirsTheProbesRead is the whole reason this verb
// exists rather than a helper in a skill: the path a lens WRITES and the
// path a probe READS have to be one string. They were two, restated per
// site, and two of the restatements named a shape the migration had
// moved — which is silent, because a lens directory that cannot exist
// reads exactly like a lens that never ran.
//
// So this asserts the identity directly: the dir `paths --lens critique`
// hands a writer is the dir the `lens_critique` probe hangs its path
// under, both taken from the same table.
func TestPathsBindsTheSameDirsTheProbesRead(t *testing.T) {
	_, ev := pathsFixture(t)
	a := pathsJSON(t, "--lens", "critique", "0007")

	want := filepath.Join(ev, "0007-iteration-shapes", "evidence", "critique")
	if a.Dir != want {
		t.Errorf("paths --lens critique = %q, want %q", a.Dir, want)
	}
	// The probe's own view of the same tree, via the fact evaluator.
	tbl, err := LoadFactTable(factTableForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	env := NewFactEnv(tbl, nil, "0007-iteration-shapes")
	probeBase, ok := env.Roots["evidence"]
	if !ok {
		t.Fatal("the evidence root did not bind for the probe")
	}
	if got := filepath.Join(probeBase, "critique"); got != a.Dir {
		t.Errorf("the writer's dir %q and the probe's %q disagree — this is the "+
			"defect the verb exists to make impossible", a.Dir, got)
	}
}

// TestNextIterReadsTheTreeRatherThanGuessing covers the four shapes the
// corpus holds. `--next-iter` LISTS the directory; it never matches a
// guessed name, which is the glob the fact table's probe rule bans.
func TestNextIterReadsTheTreeRatherThanGuessing(t *testing.T) {
	pathsFixture(t)
	for _, c := range []struct {
		name  string
		args  []string
		next  int
		found []int
		note  string
		why   string
	}{
		{"contiguous segments", []string{"--lens", "critique"}, 5, []int{2, 3, 4}, "",
			"1 + the highest segment on disk"},
		{"loose files are iteration 1", []string{"--lens", "3amigo"}, 2, nil,
			"loose files are iteration 1",
			"rdr-common §evidence: the loose set IS iteration 1, so the next pass is 2 — and the note says why, so the caller need not list the dir"},
		{"empty dir", []string{"--lens", "cove"}, 1, nil, "empty",
			"the directory exists and holds nothing, so the next pass is the first"},
		{"no dir at all", []string{"--lens", "grounding"}, 1, nil, "no such directory",
			"nothing was ever written here"},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pathsJSON(t, append(c.args, "--next-iter", "0007")...)
			if a.NextIter != c.next {
				t.Errorf("next_iter = %d, want %d — %s", a.NextIter, c.next, c.why)
			}
			if len(a.Found) != len(c.found) {
				t.Errorf("found = %v, want %v", a.Found, c.found)
			}
			if !strings.Contains(a.Note, c.note) {
				t.Errorf("note = %q, want it to say %q — ITER_FOUND/ITER_NOTE say what was on disk; a gap is named, not hidden", a.Note, c.note)
			}
		})
	}
}

// TestFirstPassWritesTheBaseNotAnIterOne: iteration 1 is the loose set,
// so the first pass's directory is the base itself. Emitting `iter-1`
// would invent a directory the corpus does not use and no probe looks
// for — the same class of mistake as the migrated-away `<lens>/<slug>/`.
func TestFirstPassWritesTheBaseNotAnIterOne(t *testing.T) {
	pathsFixture(t)
	a := pathsJSON(t, "--lens", "grounding", "--next-iter", "0007")
	if a.NextIter != 1 {
		t.Fatalf("next_iter = %d, want 1", a.NextIter)
	}
	if a.NextDir != a.Dir {
		t.Errorf("a first pass writes %q, want the base %q", a.NextDir, a.Dir)
	}
	if strings.Contains(a.NextDir, "iter-1") {
		t.Errorf("invented an iter-1 directory: %q", a.NextDir)
	}
}

// TestNextIterReportsAGapRatherThanFillingIt pins the answer on the
// non-contiguous case the corpus really holds (0122-…-0132: iter-3, no
// iter-2). Next is 1 + the HIGHEST, never the lowest absent: reusing the
// gap's number would file a later report under an earlier one and put a
// reader's history backwards. The gap is reported so it stays visible.
func TestNextIterReportsAGapRatherThanFillingIt(t *testing.T) {
	pathsFixture(t)
	a := pathsJSON(t, "--cluster", "0007-0008", "--next-iter", "0007")
	if a.NextIter != 4 {
		t.Errorf("next_iter = %d, want 4 (1 + the highest found, not the gap)", a.NextIter)
	}
	if len(a.Found) != 1 || a.Found[0] != 3 {
		t.Errorf("found = %v, want [3] — the caller has to see what was there", a.Found)
	}
	if !strings.Contains(a.Note, "iter-2") {
		t.Errorf("the gap is not named in the note: %q", a.Note)
	}
}

// TestPathsRefusesRatherThanFabricating: the two ways this verb could
// invent a directory, both refused.
//
// An unbound root is the fact table's own three-valued rule applied to a
// path: absent says "nothing looked", and a path rooted at / would be a
// guess a caller might `mkdir -p`. An unfilled `{key}` is subtler and
// was a real bug here — substituting "" collapses
// `cluster-reconcile/{key}` to the parent of EVERY cluster, which exists,
// so nothing would error and a run would write over the whole tree.
func TestPathsRefusesRatherThanFabricating(t *testing.T) {
	pathsFixture(t)
	table := factTableForTest(t)

	t.Run("unbound root", func(t *testing.T) {
		t.Setenv("RDR_EVIDENCE", filepath.Join(t.TempDir(), "nope"))
		code, out, errb := runCapture(t, "paths", "--facts", table, "--lens", "critique", "0007")
		if code != 1 || !strings.Contains(errb, "stopped:unbound-root") {
			t.Errorf("exit %d, stdout %q, stderr %q — want 1 and a stated absence", code, out, errb)
		}
		if strings.Contains(out, "EVIDENCE_DIR") {
			t.Errorf("a path was printed for an unbound root: %q", out)
		}
	})

	t.Run("unfilled placeholder", func(t *testing.T) {
		code, out, errb := runCapture(t, "paths", "--facts", table, "--tree", "cluster", "0007")
		if code != 2 || !strings.Contains(errb, "stopped:unbound-placeholder") {
			t.Errorf("exit %d, stdout %q, stderr %q — want 2; an empty key must not "+
				"collapse to the parent of every cluster", code, out, errb)
		}
		if strings.Contains(out, "EVIDENCE_DIR") {
			t.Errorf("a path was printed for an unfilled key: %q", out)
		}
	})
}

// TestPathsCreatesNothing: this is a read-only tool, and the one thing a
// path verb is most tempting to do — make the directory it just named —
// is the thing that would turn a wrong answer into a wrong tree.
func TestPathsCreatesNothing(t *testing.T) {
	_, ev := pathsFixture(t)
	before := treeSnapshot(t, ev)
	pathsJSON(t, "--lens", "grounding", "--next-iter", "0007")
	pathsJSON(t, "--cluster", "0007-0008", "--next-iter", "0007")
	if after := treeSnapshot(t, ev); after != before {
		t.Errorf("the evidence tree changed:\nbefore %s\nafter  %s", before, after)
	}
}

func treeSnapshot(t *testing.T, root string) string {
	t.Helper()
	var seen []string
	err := filepath.Walk(root, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		seen = append(seen, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(seen, "\n")
}

// TestPathsIsSchemaBindExempt: `paths` reads a record's SLUG off its
// filename and never opens the body, so requiring TEMPLATE.md would make
// a skill find the engine before it can ask where the engine's evidence
// goes. Same exemption as `env`, and it is asserted the same way: with
// RDR_HOME pointed somewhere holding no template.
func TestPathsIsSchemaBindExempt(t *testing.T) {
	pathsFixture(t)
	t.Setenv("RDR_HOME", t.TempDir())
	code, _, errb := runCapture(t, "paths", "--facts", factTableForTest(t), "--lens", "critique", "0007")
	if code != 0 {
		t.Errorf("exit %d: %s — paths must not require a template it never reads", code, errb)
	}
}

// TestPathsNamesItselfInTheUsageLog is the v12r lesson: a facet the log
// cannot name reads as never called, and this verb's whole claim is that
// it retired six hand-built constructions. The iteration half is named
// separately because it is the half that reads the disk.
func TestPathsNamesItselfInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "bind"},
		{[]string{"-json"}, "bind:json"},
		{[]string{"-next-iter"}, "next-iter"},
		{[]string{"-next-iter", "-json"}, "next-iter:json"},
	} {
		fs := flag.NewFlagSet("paths", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("paths", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if got := usageFacet("paths", f, "0007"); got != c.want {
			t.Errorf("paths %v logs as %q, want %q", c.args, got, c.want)
		}
	}
}

// TestIterationTreeMustNameADeclaredRoot: the tree/root link is checked
// at LOAD, so a table that names a root nobody declared is wrong in this
// repo rather than in a consumer's evidence dir — where it would resolve
// under the filesystem root and report a first pass on every record.
func TestIterationTreeMustNameADeclaredRoot(t *testing.T) {
	for _, c := range []struct {
		name, body, want string
	}{
		{"undeclared root",
			"[facts]\nversion = 1\n\n[root.evidence]\nvar = \"RDR_EVIDENCE\"\nsuffix = \"{slug}\"\n\n" +
				"[iteration]\nsegment = \"iter-{n}\"\n\n[iteration.tree.lens]\nroot = \"nope\"\nunder = \"{lens}\"\n\n" +
				"[fact.x]\nkind = \"bool\"\nsource = \"probe\"\nroot = \"evidence\"\npath = \"x\"\n",
			"undeclared root"},
		{"segment carrying no {n}",
			"[facts]\nversion = 1\n\n[root.evidence]\nvar = \"RDR_EVIDENCE\"\nsuffix = \"{slug}\"\n\n" +
				"[iteration]\nsegment = \"iter\"\n\n[iteration.tree.lens]\nroot = \"evidence\"\nunder = \"{lens}\"\n\n" +
				"[fact.x]\nkind = \"bool\"\nsource = \"probe\"\nroot = \"evidence\"\npath = \"x\"\n",
			"carries no {n}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "t.toml")
			if err := os.WriteFile(p, []byte(c.body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadFactTable(p)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("LoadFactTable = %v, want an error naming %q", err, c.want)
			}
		})
	}
}

// TestIterSegmentIsReadTheWayItIsDeclared is the genericity claim, as a
// test: the matcher is BUILT from the declared segment, so a document
// family that spells its iterations differently is read correctly by
// editing the table rather than by patching this binary.
func TestIterSegmentIsReadTheWayItIsDeclared(t *testing.T) {
	for _, c := range []struct {
		segment, dir string
		want         bool
	}{
		{"iter-{n}", "iter-2", true},
		{"iter-{n}", "iter-x", false},
		{"iter-{n}", "iteration-2", false},
		{"round{n}", "round7", true},
		{"v{n}.d", "v3.d", true},
		{"v{n}.d", "v3.e", false},
	} {
		re := iterSegmentRE(c.segment)
		if re == nil {
			t.Fatalf("segment %q built no matcher", c.segment)
		}
		if got := re.MatchString(c.dir); got != c.want {
			t.Errorf("segment %q against %q = %v, want %v", c.segment, c.dir, got, c.want)
		}
	}
	if iterSegmentRE("no-placeholder") != nil {
		t.Error("a segment with no {n} cannot match an iteration; want no matcher")
	}
}

// TestPathsTextFormSurvivesEval: the text form is what a skill evals, so
// every value is one quoted shell word — the rule `env` already follows,
// and the rule a path with a space would otherwise break.
func TestPathsTextFormSurvivesEval(t *testing.T) {
	pathsFixture(t)
	code, out, errb := runCapture(t, "paths", "--facts", factTableForTest(t),
		"--lens", "critique", "--next-iter", "0007")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"EVIDENCE_DIR=", "ITER=", "ITER_DIR="} {
		if !strings.Contains(out, want) {
			t.Errorf("the text form is missing %s:\n%s", want, out)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			t.Errorf("not a k=v line: %q", line)
			continue
		}
		// ITER is a bare integer; every other value is a quoted path or
		// string, so a space in a consumer's tree cannot split it.
		if k == "ITER" {
			continue
		}
		if !strings.HasPrefix(v, "'") || !strings.HasSuffix(v, "'") {
			t.Errorf("%s is not a single quoted word: %q", k, v)
		}
	}
}

// TestNextIterBucketsAgainstTheDeclaredCap: ITER_BUCKET is ITER against
// the tree's `cap`, "1".."cap" verbatim and "over" past it — the loop tag
// rdr-loop.toml routes on, so a stage never compares N to a number of its
// own. PRIOR_DIR is where the last pass wrote: the highest segment, the
// base when only loose files are there, absent when nothing is. A tree
// that declares no cap gets no bucket, so a caller cannot route on a
// cap the table never stated.
func TestNextIterBucketsAgainstTheDeclaredCap(t *testing.T) {
	_, ev := pathsFixture(t)
	const slug = "0007-iteration-shapes"
	for _, c := range []struct {
		name   string
		args   []string
		bucket string
		prior  string
	}{
		{"past the cap", []string{"--lens", "critique"}, "over",
			filepath.Join(ev, slug, "evidence", "critique", "iter-4")},
		{"loose files only", []string{"--lens", "3amigo"}, "2",
			filepath.Join(ev, slug, "evidence", "3amigo")},
		{"nothing written", []string{"--lens", "cove"}, "1", ""},
		{"cluster past the cap with a gap", []string{"--cluster", "0007-0008"}, "over",
			filepath.Join(ev, "cluster-reconcile", "0007-0008", "iter-3")},
		{"no cap declared", []string{"--tree", "spikes"}, "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pathsJSON(t, append(c.args, "--next-iter", "0007")...)
			if a.IterBucket != c.bucket {
				t.Errorf("iter_bucket = %q, want %q (next_iter %d)", a.IterBucket, c.bucket, a.NextIter)
			}
			if a.PriorDir != c.prior {
				t.Errorf("prior_dir = %q, want %q", a.PriorDir, c.prior)
			}
		})
	}
}

// TestNextIterBucketOnTheStatusFixture pins the in-cap case on the
// checked-in tree: cluster 0021-0022-0023 holds a loose report and
// iter-2, so the next pass is 3 — inside the cap, bucketed verbatim —
// and the prior pass is iter-2. The text form carries both as quoted
// words, so a skill evals them beside ITER.
func TestNextIterBucketOnTheStatusFixture(t *testing.T) {
	_, table := bindStatusFixture(t)
	code, out, errb := runCapture(t, "paths", "--facts", table, "--cluster", "0021-0022-0023", "--next-iter", "0021")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"ITER=3\n", "ITER_BUCKET='3'\n",
		"PRIOR_DIR='" + filepath.Join(os.Getenv("RDR_EVIDENCE"), "cluster-reconcile", "0021-0022-0023", "iter-2") + "'\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("the text form is missing %q:\n%s", want, out)
		}
	}
}

// TestIterationCapMustBePositive: a cap the table spells but the reader
// cannot count with would bucket every pass as "over" and stop every
// loop on its first pass — refused at load, like an undeclared root.
func TestIterationCapMustBePositive(t *testing.T) {
	body := "[facts]\nversion = 1\n\n[root.evidence]\nvar = \"RDR_EVIDENCE\"\nsuffix = \"{slug}\"\n\n" +
		"[iteration]\nsegment = \"iter-{n}\"\n\n[iteration.tree.lens]\nroot = \"evidence\"\nunder = \"{lens}\"\ncap = \"three\"\n\n" +
		"[fact.x]\nkind = \"bool\"\nsource = \"probe\"\nroot = \"evidence\"\npath = \"x\"\n"
	p := filepath.Join(t.TempDir(), "t.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFactTable(p); err == nil || !strings.Contains(err.Error(), "cap") {
		t.Errorf("LoadFactTable = %v, want an error naming the cap", err)
	}
}
