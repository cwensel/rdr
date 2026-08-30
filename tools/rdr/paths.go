package main

// `rdr paths` binds the evidence directory and the iteration number once,
// so a skill stops rebuilding them in prose.
//
// Six sites hand-built these: rdr-common §evidence, rdr-prelock's re-entry
// step, 05-prelock.md's paste-2 block, 05-prelock-resolve.prompt.md's bind
// block, and Stage 7.1's doc and prompt. Each restated the same path shape
// and the same "N = 1 + highest existing" rule, and a restatement is a copy
// that can go stale on its own. Two of them did: they named `<lens>/<slug>/`
// after the migration moved it to `<slug>/evidence/<lens>/`, and nothing
// noticed, because a lens directory that cannot exist reads exactly like a
// lens that never ran.
//
// It answers from the SAME declarations the fact evaluator reads —
// `models/rdr-facts.toml`'s roots plus its `[iteration]` block — rather
// than from knowledge of its own. That is the point: a probe and a path
// that disagree about where evidence lives is the defect above, and one
// source cannot disagree with itself.
//
//	0  the answer, text or JSON
//	1  nothing to bind: no root, no such tree
//	2  bad usage, an unreadable table, or an unresolvable record
//
// UNBOUND IS ABSENT, never a fabricated path. If `$RDR_EVIDENCE` names no
// directory the answer says so and exits 1; it does not print a path
// rooted at `/` that a caller would then `mkdir -p`. The fact table's own
// header argues this for probes — false says "looked, not there", absent
// says "nothing looked" — and a path verb has the same duty.
//
// It reads no record body and opens no template, so it is schema-bind
// exempt (main.go), like `env` and `receipt`: the record's SLUG is the
// whole input, and that comes off the filename.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// pathsAnswer is the emitted shape. Every field is omitted when it has
// nothing to say, so a caller can tell "no iteration was asked for" from
// "the iteration is 1" without reading a sentinel.
type pathsAnswer struct {
	Record string `json:"record"`
	Slug   string `json:"slug"`
	// Roots are the trees the table declares and the seam bound, by
	// name. An unbound root is absent from the map rather than empty.
	Roots map[string]string `json:"roots,omitempty"`
	// Tree is the iteration tree named by --lens/--cluster/--tree, and
	// Dir is its base — the directory a FIRST pass writes to.
	Tree string `json:"tree,omitempty"`
	Dir  string `json:"dir,omitempty"`
	// NextIter and the rest answer --next-iter. Found lists the
	// segments actually on disk, so a gap is visible; Note names one
	// when the segments are not contiguous.
	NextIter int    `json:"next_iter,omitempty"`
	NextDir  string `json:"next_dir,omitempty"`
	Found    []int  `json:"found"`
	Note     string `json:"note,omitempty"`
}

func pathsCmd(args []string, f *flags, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "stopped:usage (paths takes exactly one NNNN, slug or path)")
		return 2
	}
	tblPath, err := factTablePath(deref(f.facts))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	tbl, err := LoadFactTable(tblPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	recPath, err := resolve(args[0], *f.records)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	slug := recordSlug(recPath)
	num := recordNumberOfPath(recPath)
	if num == "" {
		fmt.Fprintf(stderr, "stopped:no-record-number (%s)\n", filepath.Base(recPath))
		return 2
	}

	// The roots, bound exactly as a probe binds them — same var, same
	// existence rule, same {slug}. NewFactEnv is reused rather than
	// reimplemented so a path and a probe can never disagree.
	env := NewFactEnv(tbl, nil, slug)
	ans := pathsAnswer{Record: num, Slug: slug, Roots: map[string]string{}, Found: []int{}}
	for name := range tbl.Roots {
		if dir, ok := env.Roots[name]; ok {
			ans.Roots[name] = dir
		}
	}

	// Which tree, if any. --lens and --cluster are the two the flow
	// names; --tree reaches any the table declares, so a new one needs
	// no new flag.
	treeName, operand, err := pathsTree(f)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if treeName != "" {
		tree, ok := tbl.Iter.Trees[treeName]
		if !ok {
			fmt.Fprintf(stderr, "stopped:no-such-tree (%s; the table declares %s)\n",
				treeName, strings.Join(sortedTreeNames(tbl.Iter.Trees), " "))
			return 2
		}
		base, ok := env.Roots[tree.Root]
		if !ok {
			// Absent, not guessed. The root's var names no directory, so
			// there is no honest path to print.
			fmt.Fprintf(stderr, "stopped:unbound-root (%s is unset or names no directory, so %s has no path)\n",
				tbl.Roots[tree.Root].Var, treeName)
			return 1
		}
		under := strings.ReplaceAll(tree.Under, "{slug}", slug)
		// Only a NON-EMPTY operand substitutes. Filling `{key}` with ""
		// would silently collapse `cluster-reconcile/{key}` to the parent
		// of every cluster — a real directory, so nothing would error, and
		// a caller would write one run's report over the whole tree. An
		// unfilled placeholder has to survive to be refused below.
		if operand != "" {
			under = strings.ReplaceAll(under, "{lens}", operand)
			under = strings.ReplaceAll(under, "{key}", operand)
		}
		if i := strings.Index(under, "{"); i >= 0 {
			fmt.Fprintf(stderr, "stopped:unbound-placeholder (tree %s needs %s; pass it as the flag's value)\n",
				treeName, under[i:])
			return 2
		}
		ans.Tree, ans.Dir = treeName, filepath.Join(base, filepath.FromSlash(under))

		if f.nextIter != nil && *f.nextIter {
			next, found, note := nextIteration(ans.Dir, tbl.Iter)
			ans.NextIter, ans.Found, ans.Note = next, found, note
			ans.NextDir = filepath.Join(ans.Dir,
				strings.ReplaceAll(tbl.Iter.Segment, "{n}", strconv.Itoa(next)))
			// Iteration 1 is the loose set: it IS the base, and saying
			// `.../iter-1` would invent a directory the corpus does not
			// use and the probes do not look for.
			if next == 1 {
				ans.NextDir = ans.Dir
			}
		}
	}

	if f.json != nil && *f.json {
		return emit(ans, stdout, stderr)
	}
	return emitPathsText(ans, stdout)
}

// pathsTree reads which tree the caller asked for. The three flags are
// mutually exclusive: each names a different tree, and a call that set
// two would have to be answered by picking one silently.
func pathsTree(f *flags) (name, operand string, err error) {
	set := 0
	if v := deref(f.lens); v != "" {
		name, operand, set = "lens", v, set+1
	}
	if v := deref(f.cluster); v != "" {
		name, operand, set = "cluster", v, set+1
	}
	if v := deref(f.tree); v != "" {
		// --tree takes `<name>[=<operand>]`, so a tree whose `under`
		// carries a placeholder is reachable without its own flag.
		n, o, _ := strings.Cut(v, "=")
		name, operand, set = n, o, set+1
	}
	if set > 1 {
		return "", "", fmt.Errorf("stopped:usage (--lens, --cluster and --tree name different trees; pass one)")
	}
	return name, operand, nil
}

// iterSegmentRE turns the declared segment into the matcher that reads it
// back. The segment is data, so the pattern is built from it rather than
// written out here — a table that spells the segment differently is then
// read the way it is declared, which is the whole reason it is data.
func iterSegmentRE(segment string) *regexp.Regexp {
	if segment == "" {
		return nil
	}
	i := strings.Index(segment, "{n}")
	if i < 0 {
		return nil
	}
	return regexp.MustCompile("^" + regexp.QuoteMeta(segment[:i]) + `(\d+)` +
		regexp.QuoteMeta(segment[i+len("{n}"):]) + "$")
}

// nextIteration LISTS the base and reports what it found. It does not
// match a guessed name against the tree — that is the glob the fact
// table's probe rule bans — it asks the directory and reads the answer.
//
// 1 + the HIGHEST found, never the lowest absent. `0122-0123-0130-0131-0132`
// on the reference corpus holds `iter-3` with no `iter-2`; filling that gap
// would file a later report under an earlier number and put a reader's
// history backwards. The gap is reported instead, so a caller sees it.
func nextIteration(base string, it Iteration) (next int, found []int, note string) {
	found = []int{}
	re := iterSegmentRE(it.Segment)
	entries, err := os.ReadDir(base)
	if err != nil {
		// No base at all: nothing has been written here, so the next
		// pass is the first one. That is a real answer — the directory
		// was asked and said "empty" — not a guess.
		return 1, found, "no such directory: nothing written yet"
	}
	loose := 0
	for _, ent := range entries {
		if !ent.IsDir() {
			if !strings.HasPrefix(ent.Name(), ".") {
				loose++
			}
			continue
		}
		if re == nil {
			continue
		}
		if m := re.FindStringSubmatch(ent.Name()); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				found = append(found, n)
			}
		}
	}
	sort.Ints(found)
	if len(found) == 0 {
		if loose > 0 {
			// Loose files ARE iteration 1 (rdr-common §evidence), so the
			// next pass is 2. Say so: an ITER=2 with nothing under
			// ITER_FOUND/ITER_NOTE reads as a hidden gap, and a caller
			// then lists the dir by hand to learn why.
			return 2, found, fmt.Sprintf("loose files are iteration 1 (%d on disk, no %s segments)",
				loose, strings.ReplaceAll(it.Segment, "{n}", "N"))
		}
		return 1, found, "empty: nothing written yet"
	}
	next = found[len(found)-1] + 1
	// Contiguity is checked against the loose set being iteration 1: the
	// segments should run 2..max.
	for want, i := 2, 0; i < len(found); i, want = i+1, want+1 {
		if found[i] != want {
			note = fmt.Sprintf("non-contiguous: %s absent",
				strings.ReplaceAll(it.Segment, "{n}", strconv.Itoa(want)))
			break
		}
	}
	if loose == 0 && note == "" {
		note = "no loose iteration-1 files beside the segments"
	}
	return next, found, note
}

// emitPathsText is the shell form: `k=v` per line, single-quoted, so the
// whole answer survives an `eval` the way `env`'s does. A skill binds
// {EVIDENCE_DIR} from it in one call instead of composing a path.
func emitPathsText(a pathsAnswer, stdout io.Writer) int {
	for _, name := range sortedKeys(a.Roots) {
		fmt.Fprintf(stdout, "RDR_ROOT_%s=%s\n",
			strings.ToUpper(strings.ReplaceAll(name, "-", "_")), shellQuote(a.Roots[name]))
	}
	if a.Dir != "" {
		fmt.Fprintf(stdout, "EVIDENCE_DIR=%s\n", shellQuote(a.Dir))
	}
	if a.NextIter > 0 {
		fmt.Fprintf(stdout, "ITER=%d\n", a.NextIter)
		fmt.Fprintf(stdout, "ITER_DIR=%s\n", shellQuote(a.NextDir))
		if len(a.Found) > 0 {
			fmt.Fprintf(stdout, "ITER_FOUND=%s\n", shellQuote(joinInts(a.Found)))
		}
		if a.Note != "" {
			fmt.Fprintf(stdout, "ITER_NOTE=%s\n", shellQuote(a.Note))
		}
	}
	return 0
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedTreeNames(m map[string]IterTree) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinInts(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
