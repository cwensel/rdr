package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// impactFixture binds testdata/impact: a records dir holding 0040 (whose
// Metadata names 0103 twice, 0120 once and itself once — the set is
// {0103, 0120}) and 0041 (no edges), the engine's own convention table,
// and a Go tree whose files each exercise one rule (each file's header
// comment says which).
func impactFixture(t *testing.T) (records, model, repo string) {
	t.Helper()
	base, err := filepath.Abs(filepath.Join("testdata", "impact"))
	if err != nil {
		t.Fatal(err)
	}
	model, err = filepath.Abs(filepath.Join("..", "..", "models", impactModelName))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "records"), model, filepath.Join("testdata", "impact", "repo")
}

func impactRun(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	records, model, repo := impactFixture(t)
	return runCapture(t, append([]string{"impact", "--records", records, "--model", model, "--repo", repo}, args...)...)
}

// TestImpactGolden pins the markdown body Phase 0 writes to impact.md.
//
// Line by line: the header carries the family and row counts, the
// record set sorted and deduplicated (0103 named twice in Metadata, the
// self-reference dropped), the literals as given, the repo as bound (no
// `@sha`: the fixture tree holds no .git) and the convention that
// detected it. TestCorpus0103 leads with two rows because families sort
// by row count; the three single-row families follow by name (byte
// order, so `TestExport0120` precedes the lower-case basenames). Both
// arms fire on import_test.go, so its rows carry `record:0103;literal:
// .dml.sql` — including TestImportHelper, whose name pins nothing and
// so groups under the file's basename. export_test.go is name-only,
// plain_test.go literal-only. other_test.go (pins 0555, outside the
// set), import.go (not a test file) and vendor/ (skipped) yield nothing.
func TestImpactGolden(t *testing.T) {
	code, out, errb := impactRun(t, "--literal", ".dml.sql", "--literal", "dml-sidecar", "0040")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	goldenPath := filepath.Join("testdata", "impact", "impact.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, []byte(out), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("golden rewritten; re-run without UPDATE_GOLDEN")
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("no golden: %v (run with UPDATE_GOLDEN=1 to write it)", err)
	}
	if out != string(want) {
		t.Errorf("the projection moved. Diff it, and if the change is "+
			"intended re-run with UPDATE_GOLDEN=1.\n--- got ---\n%s", out)
	}
}

// TestImpactJSONShape pins the --json keys a consumer reads, and that the
// counts agree with the list they summarise.
func TestImpactJSONShape(t *testing.T) {
	code, out, errb := impactRun(t, "--json", "--literal", ".dml.sql", "0040")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got struct {
		Record     string   `json:"record"`
		Slug       string   `json:"slug"`
		Families   int      `json:"families"`
		Rows       int      `json:"rows"`
		Records    []string `json:"records"`
		Literals   []string `json:"literals"`
		Convention string   `json:"convention"`
		Repo       string   `json:"repo"`
		List       []struct {
			Family string   `json:"family"`
			Files  []string `json:"files"`
			Tests  []struct {
				Test string   `json:"test"`
				File string   `json:"file"`
				Arm  []string `json:"arm"`
			} `json:"tests"`
		} `json:"families_list"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if got.Record != "0040" || got.Slug != "0040-impact-fixture" || got.Convention != "go" {
		t.Errorf("identity = %q %q %q", got.Record, got.Slug, got.Convention)
	}
	if strings.Join(got.Records, ",") != "0103,0120" || strings.Join(got.Literals, ",") != ".dml.sql" {
		t.Errorf("records %v literals %v", got.Records, got.Literals)
	}
	// Without the dml-sidecar literal plain_test.go is not predicted:
	// three families, four rows.
	if got.Families != 3 || got.Rows != 4 || len(got.List) != got.Families {
		t.Errorf("families=%d rows=%d list=%d", got.Families, got.Rows, len(got.List))
	}
	rows := 0
	for _, fam := range got.List {
		rows += len(fam.Tests)
	}
	if rows != got.Rows {
		t.Errorf("rows %d but the list holds %d", got.Rows, rows)
	}
	lead := got.List[0]
	if lead.Family != "TestCorpus0103" || strings.Join(lead.Files, ",") != "internal/cli/import_test.go" {
		t.Errorf("lead family = %+v", lead)
	}
	if arm := strings.Join(lead.Tests[0].Arm, ";"); arm != "record:0103;literal:.dml.sql" {
		t.Errorf("arm = %q", arm)
	}
	if strings.Contains(out, `"head"`) {
		t.Errorf("head is present with no .git under the fixture repo")
	}
}

// TestImpactReadsOnlyGlobFiles: the walk opens exactly the files the
// convention's glob names — the four *_test.go outside vendor/ — and
// never import.go, which carries the literal and a decoy test-shaped
// line. The output cannot show a file was NOT read, so the read counter
// is the assertion.
func TestImpactReadsOnlyGlobFiles(t *testing.T) {
	before := scan.SourceReads.Load()
	code, _, errb := impactRun(t, "--literal", ".dml.sql", "0040")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if got := scan.SourceReads.Load() - before; got != 4 {
		t.Errorf("read %d files, want 4 (import_test, export_test, plain_test, other_test)", got)
	}
}

// TestImpactSkipsNestedRepo: a repo carrying nested repositories — a
// worktree (a directory whose `.git` is a plain FILE) and a nested clone
// (a directory whose `.git` is a DIRECTORY) — each holding a copy of
// import_test.go beside the real one. A path named `.git` cannot live in
// this repo's own git history (git refuses to track it, worktree or
// submodule alike), so the fixture is built fresh in a TempDir every run
// by copying testdata/impact/repo and adding the two nested trees.
// Neither copy is read (the counter proves it, the same way
// TestImpactReadsOnlyGlobFiles does), and no row's file column starts
// with the nested path: a repo that carries worktrees or nested clones
// is counted once, on its own tree.
func TestImpactSkipsNestedRepo(t *testing.T) {
	records, model, srcRepo := impactFixture(t)
	repo := t.TempDir()
	if err := copyTree(srcRepo, repo); err != nil {
		t.Fatal(err)
	}
	// A worktree: nested-file/.git is a plain file (as git itself writes
	// for `git worktree add`), holding a copy of a globbed test file.
	writeNested(t, filepath.Join(repo, "nested-file"), func(dir string) error {
		return os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /nowhere\n"), 0o600)
	})
	// A nested clone: nested-dir/.git is a directory.
	writeNested(t, filepath.Join(repo, "nested-dir"), func(dir string) error {
		return os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	})

	before := scan.SourceReads.Load()
	code, out, errb := runCapture(t, "impact", "--records", records, "--model", model, "--repo", repo, "--json", "--literal", ".dml.sql", "0040")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if got := scan.SourceReads.Load() - before; got != 4 {
		t.Errorf("read %d files, want 4: a nested .git (file or dir) must stop the walk at its border", got)
	}
	for _, bad := range []string{"nested-file/", "nested-dir/"} {
		if strings.Contains(out, bad) {
			t.Errorf("output names a path under %q; the nested repo was walked:\n%s", bad, out)
		}
	}
}

// writeNested builds one nested-repo case under dir: the real subtree
// (a copy of internal/cli/import_test.go, which the glob would otherwise
// predict), then makeGit lays the `.git` entry across its border.
func writeNested(t *testing.T, dir string, makeGit func(dir string) error) {
	t.Helper()
	cliDir := filepath.Join(dir, "internal", "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join("testdata", "impact", "repo", "internal", "cli", "import_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "import_test.go"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := makeGit(dir); err != nil {
		t.Fatal(err)
	}
}

// copyTree copies src to dst, file by file, preserving the tree shape.
// The fixture repo under testdata/impact/repo carries no path a plain
// copy cannot reproduce (no symlinks, no other `.git`).
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o600)
	})
}

// TestImpactEmptySetIsHeaderOnly: a record with no override or
// predecessor edge and no literal has nothing to look for. The header
// says so — `records: none`, `rows: 0` — and no file is opened.
func TestImpactEmptySetIsHeaderOnly(t *testing.T) {
	before := scan.SourceReads.Load()
	code, out, errb := impactRun(t, "0041")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"# Impact — 0041-impact-empty\n", "families: 0\n", "rows: 0\n", "records: none\n", "literals: none\n", "convention: go\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("header lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "## ") {
		t.Errorf("a family was printed for an empty set:\n%s", out)
	}
	if got := scan.SourceReads.Load() - before; got != 0 {
		t.Errorf("read %d files; an empty set must not walk", got)
	}
	// A literal alone is a reason to walk, even with no record set.
	code, out, _ = impactRun(t, "--literal", "dml-sidecar", "0041")
	if code != 0 || !strings.Contains(out, "records: none\n") || !strings.Contains(out, "| TestPlainRoundTrip | internal/cli/plain_test.go | literal:dml-sidecar |") {
		t.Errorf("literal with no record set:\n%s", out)
	}
}

// TestImpactRefusesAnUnlookedTree: an unbound repo and a repo no
// convention detects are stops, never `rows: 0` — a tree nothing looked
// at must not read as a tree with no impact.
func TestImpactRefusesAnUnlookedTree(t *testing.T) {
	records, model, _ := impactFixture(t)
	t.Setenv("RDR_SOURCE_REPO", "")
	// An empty var falls through to the seam marker, and the engine
	// checkout carries one; run from a directory no marker reaches.
	t.Chdir(t.TempDir())
	code, out, errb := runCapture(t, "impact", "--records", records, "--model", model, "0040")
	if code != 2 || !strings.Contains(errb, "stopped:no-repo") || !strings.Contains(out, "stopped:no-repo") {
		t.Errorf("no repo: exit %d, stdout %q, stderr %q", code, out, errb)
	}
	empty := t.TempDir()
	code, _, errb = runCapture(t, "impact", "--records", records, "--model", model, "--repo", empty, "0040")
	if code != 2 || !strings.Contains(errb, "stopped:no-convention ("+empty+": none of go.mod present)") {
		t.Errorf("no convention: exit %d, stderr %q", code, errb)
	}
}

// TestImpactModelIsANamedStop: an absent or malformed convention table
// is refused by name. A convention that half-loaded would walk the tree
// and predict nothing, which reads exactly like a clean tree.
func TestImpactModelIsANamedStop(t *testing.T) {
	records, _, repo := impactFixture(t)
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := "[impact]\nversion = 1\n[convention.go]\norder = 1\ndetect = \"go.mod\"\nglob = \"**/*_test.go\"\n"
	for _, c := range []struct{ name, model, want string }{
		{"absent", filepath.Join(dir, "missing.toml"), "stopped:no-impact-model"},
		{"not toml", write("a.toml", "[impact\n"), "stopped:malformed-impact-model"},
		{"array of tables", write("b.toml", "[[convention]]\nname = \"go\"\n"), "stopped:malformed-impact-model"},
		{"no conventions", write("c.toml", "[impact]\nversion = 1\n"), "declares no [convention.<name>]"},
		{"missing key", write("d.toml", good+"func = \"^func (Test(\\\\d{4}))\\\\(\"\ntests = \"^func (Test)\"\n"), "names no family"},
		{"bad regex", write("e.toml", good+"func = \"^func (Test(\\\\d{4})\"\ntests = \"^func (Test)\"\nfamily = \"^(Test\"\n"), "error parsing regexp"},
		{"too few groups", write("f.toml", good+"func = \"^func Test\"\ntests = \"^func (Test)\"\nfamily = \"^(Test)\"\n"), "func needs two groups"},
		{"glob shape", write("g.toml", strings.Replace(good, "**/*_test.go", "tests/*.go", 1)+"func = \"^func (Test(\\\\d{4}))\"\ntests = \"^func (Test)\"\nfamily = \"^(Test)\"\n"), "is not **/ then a basename pattern"},
		{"unknown table", write("h.toml", good+"func = \"^func (Test(\\\\d{4}))\"\ntests = \"^func (Test)\"\nfamily = \"^(Test)\"\n[other]\nx = 1\n"), "unknown table [other]"},
	} {
		code, _, errb := runCapture(t, "impact", "--records", records, "--model", c.model, "--repo", repo, "0040")
		if code != 2 || !strings.Contains(errb, c.want) {
			t.Errorf("%s: exit %d, stderr %q, want %q", c.name, code, errb, c.want)
		}
	}
}

// TestImpactUsage: the argument shape is refused by name, like every
// other verb's.
func TestImpactUsage(t *testing.T) {
	code, _, errb := impactRun(t)
	if code != 2 || !strings.Contains(errb, "stopped:usage") {
		t.Errorf("no record: exit %d, stderr %q", code, errb)
	}
	code, _, errb = impactRun(t, "0040", "0041")
	if code != 2 || !strings.Contains(errb, "stopped:usage") {
		t.Errorf("two records: exit %d, stderr %q", code, errb)
	}
}

// TestImpactNamesItselfInTheUsageLog: a facet the log cannot name reads
// as never called, and whether the auditor supplied literals is the
// half of the call worth auditing.
func TestImpactNamesItselfInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "records"},
		{[]string{"-json"}, "records:json"},
		{[]string{"-literal", "x"}, "records+literals"},
		{[]string{"-literal", "x", "-literal", "y", "-json"}, "records+literals:json"},
	} {
		fs := flag.NewFlagSet("impact", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("impact", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if got := usageFacet("impact", f, "0040"); got != c.want {
			t.Errorf("impact %v logs as %q, want %q", c.args, got, c.want)
		}
	}
}
