package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// installLaunchHelpers builds this package as `rdr` in a temp dir and copies
// bin/rdr-gate and bin/rdr-leg-commit beside it — the layout the scripts
// bind (`$(dirname "$0")/rdr`), with RDR_HOME pointed at the engine for the
// models. Same shape as TestRdrNextRendersTheWorklist; skips without a Go
// toolchain, and the callers skip without intrastate.
func installLaunchHelpers(t *testing.T) string {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain to build the projector the scripts bind")
	}
	home, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if out, err := exec.Command(goBin, "build", "-o", filepath.Join(dir, "rdr"), ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, s := range []string{"rdr-gate", "rdr-leg-commit", "rdr-leg-budget", "rdr-leg-test"} {
		script, err := os.ReadFile(filepath.Join(home, "bin", s))
		if err != nil {
			t.Fatalf("bin/%s is not in the tree: %v", s, err)
		}
		if err := os.WriteFile(filepath.Join(dir, s), script, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("RDR_HOME", home)
	return dir
}

// runScript runs one helper through `sh` from cwd, returning its exit code
// and combined output; only a failure to start is fatal.
func runScript(t *testing.T, cwd, script string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{script}, args...)...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), string(out)
		}
		t.Fatalf("%s: %v", script, err)
	}
	return 0, string(out)
}

func firstLine(s string) string { return strings.SplitN(s, "\n", 2)[0] }

// TestRdrGateAnswersEachOutcome is launch.md's every gate as the one
// command the prompt now writes: the script renders the outcome's facts,
// adds the caller's tags, resolves the row, and prints `next:`/`why:`.
// The expected rows are the fixture's known cells (0030 COMPLETE and
// small/short, 0022 with no verification.md, 0029 with no impact.md,
// 0021 clustered); `ground` reads rdr-write.toml through the same table.
func TestRdrGateAnswersEachOutcome(t *testing.T) {
	t.Setenv("RDR_INTRASTATE", intrastateBinary(t))
	bindStatusFixture(t)
	dir := installLaunchHelpers(t)
	gate := filepath.Join(dir, "rdr-gate")

	for _, c := range []struct {
		args []string
		next string
	}{
		{[]string{"complete", "0030", "--tag", "suite_green=true"}, "COMPLETE"},
		{[]string{"complete", "0030", "--tag", "suite_green=false"}, "stopped:suite-red"},
		{[]string{"complete", "0022", "--tag", "suite_green=true"}, "stopped:verification-unrun"},
		{[]string{"size", "0030", "--tag", "files=0-3", "--tag", "suite=quick", "--tag", "pressure=false"}, "inline"},
		{[]string{"size", "0030", "--tag", "files=4+", "--tag", "suite=quick", "--tag", "pressure=false"}, "delegated"},
		{[]string{"shard", "0030"}, "sharded"},
		{[]string{"shard", "0029"}, "stopped:impact-unread"},
		{[]string{"precheck", "0029", "--tag", "baseline=none"}, "run-baseline"},
		{[]string{"precheck", "0029", "--tag", "baseline=green"}, "proceed"},
		{[]string{"budget", "--tag", "commits=6+", "--tag", "elapsed=0-30", "--tag", "suite_green=false"}, "return-partial"},
		{[]string{"budget", "--tag", "commits=0-5", "--tag", "elapsed=0-30", "--tag", "suite_green=true"}, "return-green"},
		{[]string{"ground", "0030", "--tag", "searched=none", "--tag", "found=false"}, "code"},
		{[]string{"ground", "0021", "--tag", "searched=code", "--tag", "found=false"}, "cluster"},
		{[]string{"ground", "0030", "--tag", "searched=code", "--tag", "found=true"}, "apply"},
	} {
		code, out := runScript(t, dir, gate, c.args...)
		if code != 0 {
			t.Errorf("rdr-gate %v: exit %d\n%s", c.args, code, out)
			continue
		}
		if got := firstLine(out); got != "next: "+c.next {
			t.Errorf("rdr-gate %v: first line %q, want %q", c.args, got, "next: "+c.next)
		}
		if !strings.Contains(out, "\nwhy: ") {
			t.Errorf("rdr-gate %v: no why: line\n%s", c.args, out)
		}
		if c.args[0] == "ground" && !strings.Contains(out, "\nsurface: ") {
			t.Errorf("rdr-gate ground %v: no surface: line\n%s", c.args, out)
		}
	}
}

// TestRdrGateRefusesRatherThanGuesses: a missing observed tag, a missing or
// extra record, an unknown record, an off-domain value and an unknown
// outcome are exit 2 with the reason and NO `next:` line — the answer is
// never substituted (rdr-common §intrastate).
func TestRdrGateRefusesRatherThanGuesses(t *testing.T) {
	t.Setenv("RDR_INTRASTATE", intrastateBinary(t))
	bindStatusFixture(t)
	dir := installLaunchHelpers(t)
	gate := filepath.Join(dir, "rdr-gate")

	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"complete", "0030"}, "--tag suite_green=<true|false>"},
		{[]string{"size", "0030", "--tag", "files=0-3"}, "--tag suite=<quick,long>"},
		{[]string{"complete", "--tag", "suite_green=true"}, "needs the record"},
		{[]string{"budget", "0030", "--tag", "commits=0-5"}, "reads no record"},
		{[]string{"complete", "9999", "--tag", "suite_green=true"}, "stopped:no-such-record"},
		{[]string{"complete", "0030", "--tag", "suite_green=maybe"}, "stopped:gate-refused"},
		{[]string{"frob", "0030"}, "unknown outcome"},
		{[]string{}, "usage:"},
	} {
		code, out := runScript(t, dir, gate, c.args...)
		if code != 2 {
			t.Errorf("rdr-gate %v: exit %d, want 2\n%s", c.args, code, out)
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("rdr-gate %v: output lacks %q\n%s", c.args, c.want, out)
		}
		if strings.Contains(out, "next: ") {
			t.Errorf("rdr-gate %v: a refusal printed an answer\n%s", c.args, out)
		}
	}
}

// TestRdrLegCommitCommitsThenAsksTheBudget walks a leg's commits in a
// scratch repo: every call commits (a red tree with a ` [wip]` suffix — a
// suffix because a consumer's commit-msg hook may enforce `type(scope): …`),
// reads the count since the start SHA and the minutes since the start
// epoch, and prints the `budget` row — the 6th commit and the 31st minute
// each cut the leg; a clean tree still asks; a missing suite verdict and a
// refusing hook ask nothing, since nothing moved.
func TestRdrLegCommitCommitsThenAsksTheBudget(t *testing.T) {
	t.Setenv("RDR_INTRASTATE", intrastateBinary(t))
	dir := installLaunchHelpers(t)
	leg := filepath.Join(dir, "rdr-leg-commit")
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.email", "leg@test")
	git("config", "user.name", "leg")
	git("config", "commit.gpgsign", "false")
	git("config", "core.hooksPath", ".githooks") // local beats a global hooksPath, so the refusing hook below fires
	git("commit", "-q", "--allow-empty", "-m", "init")
	start := git("rev-parse", "--short", "HEAD")
	since := strconv.FormatInt(time.Now().Unix(), 10)
	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	expect := func(out string, wants ...string) {
		t.Helper()
		for _, w := range wants {
			if !strings.Contains(out, w) {
				t.Errorf("output lacks %q:\n%s", w, out)
			}
		}
	}

	write("a")
	code, out := runScript(t, repo, leg, "--start", start, "--since", since, "--suite-green", "false", "-m", "add a")
	if code != 0 {
		t.Fatalf("first commit: exit %d\n%s", code, out)
	}
	expect(out, "committed ", "  add a [wip]", "commits: 1 (0-5)", "elapsed: 0 min (0-30)", "next: continue")
	if subj := git("log", "-1", "--format=%s"); subj != "add a [wip]" {
		t.Errorf("red commit subject %q, want the [wip] suffix", subj)
	}

	code, out = runScript(t, repo, leg, "--start", start, "--since", since, "--suite-green", "false")
	if code != 0 {
		t.Fatalf("clean tree: exit %d\n%s", code, out)
	}
	expect(out, "nothing to commit", "commits: 1 (0-5)", "next: continue")

	write("b")
	code, out = runScript(t, repo, leg, "--start", start, "--since", since, "--suite-green", "true", "-m", "feat: b")
	if code != 0 {
		t.Fatalf("green commit: exit %d\n%s", code, out)
	}
	expect(out, "committed ", "  feat: b", "commits: 2 (0-5)", "next: return-green")
	if strings.Contains(out, "[wip]") {
		t.Errorf("a green commit took the [wip] suffix\n%s", out)
	}

	for _, n := range []string{"c", "d", "e"} {
		write(n)
		if code, out = runScript(t, repo, leg, "--start", start, "--since", since, "--suite-green", "false", "-m", n); code != 0 {
			t.Fatalf("commit %s: exit %d\n%s", n, code, out)
		}
	}
	write("f")
	code, out = runScript(t, repo, leg, "--start", start, "--since", since, "--suite-green", "false", "-m", "sixth")
	if code != 0 {
		t.Fatalf("sixth commit: exit %d\n%s", code, out)
	}
	expect(out, "commits: 6 (6+)", "next: return-partial")

	late := strconv.FormatInt(time.Now().Unix()-31*60, 10)
	code, out = runScript(t, repo, leg, "--start", git("rev-parse", "--short", "HEAD"), "--since", late, "--suite-green", "false")
	if code != 0 {
		t.Fatalf("elapsed ask: exit %d\n%s", code, out)
	}
	expect(out, "commits: 0 (0-5)", "elapsed: 31 min (31+)", "next: return-partial")

	write("g")
	code, out = runScript(t, repo, leg, "--start", start, "--since", since, "-m", "no verdict")
	if code != 2 || strings.Contains(out, "next: ") || strings.Contains(out, "committed ") {
		t.Errorf("missing --suite-green: exit %d, want 2 and no commit, no answer\n%s", code, out)
	}
	expect(out, "--suite-green <true|false>")

	if err := os.MkdirAll(filepath.Join(repo, ".githooks"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".githooks", "pre-commit"), []byte("#!/bin/sh\necho 'vet: boom' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	head := git("rev-parse", "HEAD")
	code, out = runScript(t, repo, leg, "--start", start, "--since", since, "--suite-green", "false", "-m", "hooked")
	if code == 0 || strings.Contains(out, "next: ") {
		t.Errorf("a refusing hook: exit %d, want non-zero and no answer\n%s", code, out)
	}
	expect(out, "vet: boom")
	if git("rev-parse", "HEAD") != head {
		t.Errorf("a refused commit moved HEAD")
	}
	if git("status", "--porcelain") == "" {
		t.Errorf("a refused commit left the tree clean")
	}
	if err := os.Remove(filepath.Join(repo, ".githooks", "pre-commit")); err != nil {
		t.Fatal(err)
	}

	// -C: the same commit from an unrelated cwd, the shape a caller that never cd's uses.
	code, out = runScript(t, t.TempDir(), leg, "-C", repo, "--start", start, "--since", since, "--suite-green", "false", "-m", "hooked")
	if code != 0 {
		t.Fatalf("-C commit: exit %d\n%s", code, out)
	}
	expect(out, "committed ", "commits: 7 (6+)", "next: return-partial")
}

// TestRdrLegTestRunsThenAsksTheBudget: a run under the elapsed cap executes
// the command and asks the budget with suite_green = (--full && exit 0); a
// run at or past the cap is refused BEFORE it starts — the command's own
// output (which would prove it ran) must never appear — and the budget is
// asked with suite_green=false. -C runs the command inside that dir rather
// than the test's cwd.
func TestRdrLegTestRunsThenAsksTheBudget(t *testing.T) {
	t.Setenv("RDR_INTRASTATE", intrastateBinary(t))
	dir := installLaunchHelpers(t)
	legTest := filepath.Join(dir, "rdr-leg-test")
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.email", "leg@test")
	git("config", "user.name", "leg")
	git("config", "commit.gpgsign", "false")
	git("commit", "-q", "--allow-empty", "-m", "init")
	start := git("rev-parse", "--short", "HEAD")
	since := strconv.FormatInt(time.Now().Unix(), 10)
	expect := func(out string, wants ...string) {
		t.Helper()
		for _, w := range wants {
			if !strings.Contains(out, w) {
				t.Errorf("output lacks %q:\n%s", w, out)
			}
		}
	}

	// a. under the cap, a package run (no --full) never reports green
	code, out := runScript(t, repo, legTest, "--start", start, "--since", since, "--", "sh", "-c", "echo ran; exit 0")
	if code != 0 {
		t.Fatalf("package run: exit %d\n%s", code, out)
	}
	expect(out, "ran", "run: exit 0", "next: continue")

	// b. --full, exit 0 -> return-green
	code, out = runScript(t, repo, legTest, "--start", start, "--since", since, "--full", "--", "sh", "-c", "exit 0")
	if code != 0 {
		t.Fatalf("full green run: exit %d\n%s", code, out)
	}
	expect(out, "run: exit 0", "next: return-green")

	// c. --full, exit 1 -> a red full run, still under caps
	code, out = runScript(t, repo, legTest, "--start", start, "--since", since, "--full", "--", "sh", "-c", "exit 1")
	if code != 0 {
		t.Fatalf("full red run: exit %d\n%s", code, out)
	}
	expect(out, "run: exit 1", "next: continue")

	// d. over the cap: the run must not start, so its output must not appear
	late := strconv.FormatInt(time.Now().Unix()-40*60, 10)
	code, out = runScript(t, repo, legTest, "--start", start, "--since", late, "--", "sh", "-c", "echo MUST-NOT-RUN")
	if code != 0 {
		t.Fatalf("over-cap ask: exit %d\n%s", code, out)
	}
	expect(out, "not run: over the leg's cap")
	if strings.Contains(out, "MUST-NOT-RUN") {
		t.Errorf("a run over the cap still ran the command\n%s", out)
	}
	expect(out, "next: return-partial")

	// e. -C: the command runs inside the repo, not the test's cwd
	code, out = runScript(t, t.TempDir(), legTest, "-C", repo, "--start", start, "--since", since, "--", "sh", "-c", "pwd")
	if code != 0 {
		t.Fatalf("-C run: exit %d\n%s", code, out)
	}
	wantDir, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	gotDir := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "/") {
			gotDir = line
			break
		}
	}
	if resolved, err := filepath.EvalSymlinks(gotDir); err != nil || resolved != wantDir {
		t.Errorf("-C run: pwd %q (resolved %q, err %v), want %q\n%s", gotDir, resolved, err, wantDir, out)
	}

	// f. usage refusals: exit 2, nothing after --, and --since not epoch seconds
	code, out = runScript(t, repo, legTest, "--start", start, "--since", since, "--")
	if code != 2 || strings.Contains(out, "next: ") {
		t.Errorf("no command after --: exit %d, want 2 and no answer\n%s", code, out)
	}
	code, out = runScript(t, repo, legTest, "--start", start, "--since", "now", "--", "sh", "-c", "exit 0")
	if code != 2 || strings.Contains(out, "next: ") {
		t.Errorf("--since now: exit %d, want 2 and no answer\n%s", code, out)
	}
}

// TestRdrLegBudgetBucketsGitAndTheClock: the budget ask reads commits since
// the start SHA from git and minutes since the start epoch from the clock,
// buckets each over the model's declared domain, and resolves the row —
// the same ask rdr-leg-commit and rdr-leg-test both end in.
func TestRdrLegBudgetBucketsGitAndTheClock(t *testing.T) {
	t.Setenv("RDR_INTRASTATE", intrastateBinary(t))
	dir := installLaunchHelpers(t)
	budget := filepath.Join(dir, "rdr-leg-budget")
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.email", "leg@test")
	git("config", "user.name", "leg")
	git("config", "commit.gpgsign", "false")
	git("commit", "-q", "--allow-empty", "-m", "init")
	start := git("rev-parse", "--short", "HEAD")
	git("commit", "-q", "--allow-empty", "-m", "one")
	git("commit", "-q", "--allow-empty", "-m", "two")
	since := strconv.FormatInt(time.Now().Unix(), 10)
	expect := func(out string, wants ...string) {
		t.Helper()
		for _, w := range wants {
			if !strings.Contains(out, w) {
				t.Errorf("output lacks %q:\n%s", w, out)
			}
		}
	}

	code, out := runScript(t, repo, budget, "--start", start, "--since", since, "--suite-green", "false")
	if code != 0 {
		t.Fatalf("commits ask: exit %d\n%s", code, out)
	}
	expect(out, "commits: 2 (0-5)", "next: continue")

	late := strconv.FormatInt(time.Now().Unix()-40*60, 10)
	code, out = runScript(t, repo, budget, "--start", start, "--since", late, "--suite-green", "false")
	if code != 0 {
		t.Fatalf("elapsed ask: exit %d\n%s", code, out)
	}
	expect(out, "elapsed: 40 min (31+)", "next: return-partial")

	code, out = runScript(t, repo, budget, "--start", start, "--since", late, "--suite-green", "true")
	if code != 0 {
		t.Fatalf("green ask: exit %d\n%s", code, out)
	}
	expect(out, "next: return-green")
}
