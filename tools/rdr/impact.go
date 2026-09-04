package main

// `rdr impact` predicts which predecessor tests a locked record's contract
// changes will turn red, so the implementer meets the list before the
// loop rather than one red test at a time inside it.
//
// The baseline precheck proves the suite green, so a predecessor test
// that goes red under Stage 8 is this change's doing — either a
// regression to fix or a contract the record retires, and telling the
// two apart is a per-test decision the implementer used to make mid-loop
// with no list. This projects the list from two sources: the records
// the RDR states it overrides or succeeds — its own Metadata edges, read
// the way `status` reads a record, never resolved — and the literals the
// change retires, which the caller names because deciding what a REQ
// retires is judgement. Which files are tests, and how a test's name
// carries the record it pins, is a convention of the SOURCE repo and is
// declared in `models/rdr-impact.toml`, not here (README §Impact).
//
//	0  the projection, markdown or JSON — possibly `rows: 0`
//	2  bad usage, no model, an unresolvable record, an unbound repo, or
//	   a repo no declared convention detects
//
// A SKIPPED LOOK IS NEVER AN EMPTY ANSWER. An unbound `--repo` and a
// tree no convention detects are both stops: `rows: 0` says "looked, and
// nothing pins these records", and only a walk can say that. The one
// empty answer that needs no walk is a record with no override or
// predecessor edge and no literal — there is nothing to look for, and
// the header says so with `records: none`.
//
// Only files the convention's glob names are opened, and each once. A
// source file that carries a retired literal is not a test, and reading
// it would cost the walk the whole tree; the read count is what a test
// asserts, because the output cannot show which files were NOT read.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
	"github.com/cwensel/rdr/tools/rdr/internal/toml"
)

const impactModelName = "rdr-impact.toml"

// ImpactModel is `models/rdr-impact.toml` as read: the conventions in
// `order`, each compiled once.
type ImpactModel struct {
	Version     int
	Description string
	Conventions []ImpactConvention
	Source      string
}

// ImpactConvention is one `[convention.<name>]` table. Glob is the
// basename pattern after `**/`; Func names a record-named test (group 1
// the name, group 2 the record), Tests any test (group 1 the name),
// Family the key a test name groups under (group 1).
type ImpactConvention struct {
	Name   string
	Order  int
	Detect string
	Glob   string
	Func   *regexp.Regexp
	Tests  *regexp.Regexp
	Family *regexp.Regexp
}

// LoadImpactModel reads and compiles the model. Every defect is a named
// stop: a convention missing a key or carrying a regex that does not
// compile is refused, because a convention that half-loads would walk
// the tree and predict nothing, which reads exactly like a clean tree.
func LoadImpactModel(path string) (*ImpactModel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("stopped:no-impact-model (%v)", err)
	}
	doc, err := toml.Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("stopped:malformed-impact-model (%s: %v)", path, err)
	}
	m := &ImpactModel{Source: path}
	for _, tbl := range doc {
		switch {
		case tbl.Name == "impact":
			m.Version = tbl.Int("version")
			m.Description = tbl.Str("description")
		case strings.HasPrefix(tbl.Name, "convention."):
			c := ImpactConvention{Name: strings.TrimPrefix(tbl.Name, "convention."), Order: tbl.Int("order")}
			c.Detect, c.Glob = tbl.Str("detect"), tbl.Str("glob")
			for _, key := range []string{"detect", "glob", "func", "tests", "family"} {
				if tbl.Str(key) == "" {
					return nil, fmt.Errorf("stopped:malformed-impact-model (%s: convention %q names no %s)", path, c.Name, key)
				}
			}
			if !strings.HasPrefix(c.Glob, "**/") || strings.ContainsRune(c.Glob[3:], '/') {
				return nil, fmt.Errorf("stopped:malformed-impact-model (%s: convention %q glob %q is not **/ then a basename pattern)", path, c.Name, c.Glob)
			}
			if _, err := filepath.Match(c.Glob[3:], "x"); err != nil {
				return nil, fmt.Errorf("stopped:malformed-impact-model (%s: convention %q glob %q: %v)", path, c.Name, c.Glob, err)
			}
			for key, dst := range map[string]**regexp.Regexp{"func": &c.Func, "tests": &c.Tests, "family": &c.Family} {
				re, err := regexp.Compile(tbl.Str(key))
				if err != nil {
					return nil, fmt.Errorf("stopped:malformed-impact-model (%s: convention %q %s: %v)", path, c.Name, key, err)
				}
				*dst = re
			}
			if c.Func.NumSubexp() < 2 || c.Tests.NumSubexp() < 1 || c.Family.NumSubexp() < 1 {
				return nil, fmt.Errorf("stopped:malformed-impact-model (%s: convention %q: func needs two groups, tests and family one)", path, c.Name)
			}
			m.Conventions = append(m.Conventions, c)
		default:
			return nil, fmt.Errorf("stopped:malformed-impact-model (%s: line %d: unknown table [%s])", path, tbl.Line, tbl.Name)
		}
	}
	if len(m.Conventions) == 0 {
		return nil, fmt.Errorf("stopped:malformed-impact-model (%s: declares no [convention.<name>])", path)
	}
	sort.SliceStable(m.Conventions, func(i, j int) bool { return m.Conventions[i].Order < m.Conventions[j].Order })
	return m, nil
}

// impactRow is one predicted test. Arm lists what predicted its FILE —
// `record:NNNN` for a record-named test in the same file, `literal:<tok>`
// for a literal the file carries — so a row that is only there because
// its neighbour pins a record says so.
type impactRow struct {
	Test string   `json:"test"`
	File string   `json:"file"`
	Arm  []string `json:"arm"`
}

// impactFamily groups rows under the family key: the record-named prefix
// (`TestCorpus0103`) or, for a test whose name pins nothing, the file's
// basename.
type impactFamily struct {
	Family string      `json:"family"`
	Files  []string    `json:"files"`
	Tests  []impactRow `json:"tests"`
}

// impactAnswer is the `--json` shape. Keys are the contract:
//
//	record        the record number
//	slug          its filename slug
//	families      distinct family keys
//	rows          predicted tests
//	records       the override + predecessor record set, sorted
//	literals      the --literal tokens, as given
//	convention    the convention that detected the repo
//	repo          the repo as bound
//	head          the repo's short HEAD sha, when `<repo>/.git` exists
//	families_list the families, by row count desc then name, each with
//	              its files and rows (rows by file then name)
type impactAnswer struct {
	Record     string         `json:"record"`
	Slug       string         `json:"slug"`
	Families   int            `json:"families"`
	Rows       int            `json:"rows"`
	Records    []string       `json:"records"`
	Literals   []string       `json:"literals"`
	Convention string         `json:"convention"`
	Repo       string         `json:"repo"`
	Head       string         `json:"head,omitempty"`
	List       []impactFamily `json:"families_list"`
}

func impactCmd(args []string, f *flags, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "stopped:usage (impact takes exactly one NNNN, slug or path)")
		return 2
	}
	modelPath, err := beside("models", impactModelName, deref(f.model))
	if err != nil {
		fmt.Fprintln(stderr, "stopped:no-impact-model (looked in $RDR_HOME/models and beside the binary; --model names one)")
		return 2
	}
	m, err := LoadImpactModel(modelPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	recPath, err := resolve(args[0], *f.records)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	// The record is read the way `status` reads it: scanned, never
	// edge-resolved. An override or predecessor edge names a record by
	// number in the Metadata block, and the number is all this needs.
	doc, err := scan.File(recPath, scan.Options{Project: *f.project})
	if err != nil {
		fmt.Fprintf(stderr, "stopped:unreadable (%v)\n", err)
		return 2
	}
	if doc.Record == "" {
		fmt.Fprintln(stderr, "stopped:no-record-number (neither the title nor the filename carries NNNN)")
		return 2
	}
	set := map[string]bool{}
	for _, e := range doc.Edges {
		if e.Kind != edge.Overrides && e.Kind != edge.Predecessor {
			continue
		}
		if n := edgeRecord(e.To); n != "" && n != doc.Record {
			set[n] = true
		}
	}
	records := make([]string, 0, len(set))
	for n := range set {
		records = append(records, n)
	}
	sort.Strings(records)
	literals := dedupe(f.literal.values)

	repo := strings.TrimSpace(*f.repo)
	if repo == "" {
		fmt.Fprintln(stderr, "stopped:no-repo (bind $RDR_SOURCE_REPO or pass --repo)")
		return 2
	}
	repo = filepath.Clean(repo)
	var conv *ImpactConvention
	detects := make([]string, 0, len(m.Conventions))
	for i := range m.Conventions {
		c := &m.Conventions[i]
		detects = append(detects, c.Detect)
		if conv == nil && fileExists(filepath.Join(repo, c.Detect)) {
			conv = c
		}
	}
	if conv == nil {
		fmt.Fprintf(stderr, "stopped:no-convention (%s: none of %s present)\n", repo, strings.Join(detects, ", "))
		return 2
	}

	ans := impactAnswer{
		Record: doc.Record, Slug: recordSlug(recPath),
		Records: records, Literals: literals,
		Convention: conv.Name, Repo: repo, Head: repoHead(repo),
		List: []impactFamily{},
	}
	// Nothing to look for is the one empty answer that needs no walk:
	// the header carries `records: none` so the reader can see why.
	if len(records) > 0 || len(literals) > 0 {
		ans.List = impactWalk(repo, conv, set, literals)
	}
	for _, fam := range ans.List {
		ans.Rows += len(fam.Tests)
	}
	ans.Families = len(ans.List)

	if *f.json {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(ans); err != nil {
			fmt.Fprintf(stderr, "stopped:encode (%v)\n", err)
			return 2
		}
		return 0
	}
	emitImpact(ans, stdout)
	return 0
}

// impactWalk reads every file the glob names, once, and predicts by file:
// a file is predicted when any of its record-named tests pins a member
// of the set (name arm) or its body carries a literal (literal arm), and
// then every test in it is a row.
func impactWalk(repo string, conv *ImpactConvention, set map[string]bool, literals []string) []impactFamily {
	pattern := conv.Glob[3:]
	needles := make([][]byte, len(literals))
	for i, l := range literals {
		needles[i] = []byte(l)
	}
	byFamily := map[string]*impactFamily{}
	filepath.WalkDir(repo, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			if p != repo && scan.SkipDir(e.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if ok, _ := filepath.Match(pattern, e.Name()); !ok {
			return nil
		}
		info, err := e.Info()
		if err != nil || info.Size() > scan.MaxSearchBytes {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		scan.SourceReads.Add(1)

		var arm []string
		var tests []string
		pinned := map[string]bool{}
		for _, line := range bytes.Split(body, []byte("\n")) {
			// Both regexes are anchored at `^func`; the prefix check
			// keeps the regex off every line that cannot match.
			if !bytes.HasPrefix(line, []byte("func ")) {
				continue
			}
			if m := conv.Tests.FindSubmatch(line); m != nil {
				tests = append(tests, string(m[1]))
			}
			if m := conv.Func.FindSubmatch(line); m != nil && set[string(m[2])] {
				pinned[string(m[2])] = true
			}
		}
		for n := range pinned {
			arm = append(arm, "record:"+n)
		}
		sort.Strings(arm)
		for i, needle := range needles {
			if bytes.Contains(body, needle) {
				arm = append(arm, "literal:"+literals[i])
			}
		}
		if len(arm) == 0 {
			return nil
		}
		rel, err := filepath.Rel(repo, p)
		if err != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		for _, name := range tests {
			key := filepath.Base(rel)
			if m := conv.Family.FindStringSubmatch(name); m != nil {
				key = m[1]
			}
			fam := byFamily[key]
			if fam == nil {
				fam = &impactFamily{Family: key}
				byFamily[key] = fam
			}
			fam.Tests = append(fam.Tests, impactRow{Test: name, File: rel, Arm: arm})
		}
		return nil
	})

	out := make([]impactFamily, 0, len(byFamily))
	for _, fam := range byFamily {
		sort.Slice(fam.Tests, func(i, j int) bool {
			a, b := fam.Tests[i], fam.Tests[j]
			if a.File != b.File {
				return a.File < b.File
			}
			return a.Test < b.Test
		})
		seen := map[string]bool{}
		for _, r := range fam.Tests {
			if !seen[r.File] {
				seen[r.File] = true
				fam.Files = append(fam.Files, r.File)
			}
		}
		out = append(out, *fam)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Tests) != len(out[j].Tests) {
			return len(out[i].Tests) > len(out[j].Tests)
		}
		return out[i].Family < out[j].Family
	})
	return out
}

// emitImpact renders the markdown form — the body of `<art>/impact.md`,
// which the caller writes from stdout; this binary writes no file.
func emitImpact(ans impactAnswer, w io.Writer) {
	fmt.Fprintf(w, "# Impact — %s\n\n", ans.Slug)
	fmt.Fprintf(w, "families: %d\n", ans.Families)
	fmt.Fprintf(w, "rows: %d\n", ans.Rows)
	fmt.Fprintf(w, "records: %s\n", orNone(strings.Join(ans.Records, " ")))
	lits := make([]string, len(ans.Literals))
	for i, l := range ans.Literals {
		lits[i] = "`" + l + "`"
	}
	fmt.Fprintf(w, "literals: %s\n", orNone(strings.Join(lits, " ")))
	if ans.Head != "" {
		fmt.Fprintf(w, "repo: %s @%s\n", ans.Repo, ans.Head)
	} else {
		fmt.Fprintf(w, "repo: %s\n", ans.Repo)
	}
	fmt.Fprintf(w, "convention: %s\n", ans.Convention)
	for _, fam := range ans.List {
		fmt.Fprintf(w, "\n## %s (%d tests, %d files)\n", fam.Family, len(fam.Tests), len(fam.Files))
		fmt.Fprintln(w, "| test | file | arm |")
		fmt.Fprintln(w, "|---|---|---|")
		for _, r := range fam.Tests {
			fmt.Fprintf(w, "| %s | %s | %s |\n", r.Test, r.File, strings.Join(r.Arm, ";"))
		}
	}
}

// edgeRecord is the record number an edge target names: a whole-record
// target (`0103`, `cli/0103`) or an element inside one (`0103:C4`, which
// an Overrides value may cite). "" for anything else.
func edgeRecord(to string) string {
	if id, err := ident.Parse(to); err == nil {
		return id.Record
	}
	if i := strings.LastIndex(to, "/"); i >= 0 {
		to = to[i+1:]
	}
	if ident.IsRecord(to) {
		return ident.RecordOf(to)
	}
	return ""
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// dedupe keeps the first occurrence of each literal, in the order given,
// so a token typed twice predicts once and the header lists it once.
func dedupe(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// repoHead is the short HEAD sha when the repo root itself holds `.git`;
// "" otherwise. The check comes before the git call so a repo that is a
// subtree of some other checkout does not report that checkout's HEAD
// as its own.
func repoHead(repo string) string {
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return ""
	}
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
