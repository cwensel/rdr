package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newProject makes a git repo with a marker, so a test exercises the same
// topology the flow resolves against: git-common-dir -> project -> ws.
func newProject(t *testing.T, markerAt, body string) (project, records string) {
	t.Helper()
	// EvalSymlinks to match what the resolver does (`pwd -P` in the shell
	// block): on macOS t.TempDir() hands back /var/... which is a symlink
	// to /private/var/..., and the comparison would fail on the link, not
	// on the logic.
	ws := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(ws); err == nil {
		ws = resolved
	}
	project = filepath.Join(ws, "consumer")
	records = filepath.Join(project, "docs", "rdr")
	if err := os.MkdirAll(records, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := "# Recommendation 0007: Seam\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(records, "0007-seam.md"), []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = project
	if err := cmd.Run(); err != nil {
		t.Skipf("git unavailable: %v", err)
	}

	var path string
	switch markerAt {
	case "local":
		path = filepath.Join(project, ".rdr", "workspace")
	case "shared":
		path = filepath.Join(ws, ".rdr-workspace")
	default:
		t.Fatalf("bad markerAt %q", markerAt)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return project, records
}

// TestSeamBindsFromTheMarker: the vars live in a file the flow already
// maintains, so the binary reads them itself. Before this, every call
// site carried the seam through the turn — a re-run resolver block or an
// `export RDR_HOME=… RDR_RECORDS=…` prefix on invocation after
// invocation, because shell state dies between tool calls.
func TestSeamBindsFromTheMarker(t *testing.T) {
	// The marker expands $PROJECT internally and guards on it, exactly
	// as the real ones do.
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_SOURCE_REPO="$PROJECT"
export RDR_RECORDS RDR_SOURCE_REPO
`
	project, records := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_SOURCE_REPO", "")

	got := bindSeam()
	if got["RDR_RECORDS"] != records {
		t.Errorf("RDR_RECORDS = %q, want %q", got["RDR_RECORDS"], records)
	}
	if got["RDR_SOURCE_REPO"] != project {
		t.Errorf("RDR_SOURCE_REPO = %q, want %q", got["RDR_SOURCE_REPO"], project)
	}
}

// TestNearestMarkerWins: a repo-local marker overrides the shared
// workspace one, the way the closest .git or .editorconfig governs.
func TestNearestMarkerWins(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
export RDR_RECORDS
`
	project, records := newProject(t, "local", body)
	// A shared marker beside it, naming somewhere else entirely.
	ws := filepath.Dir(project)
	shared := `: "${WS:?needs the canonical resolver}"
RDR_RECORDS="$WS/not-this-one"
export RDR_RECORDS
`
	if err := os.WriteFile(filepath.Join(ws, ".rdr-workspace"), []byte(shared), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")

	if got := bindSeam()["RDR_RECORDS"]; got != records {
		t.Errorf("nearest marker lost: got %q, want the repo-local %q", got, records)
	}
}

// TestEnvironmentOutranksTheMarker: an inherited value is a decision
// someone made; the marker is only the fallback.
func TestEnvironmentOutranksTheMarker(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
export RDR_RECORDS
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "/somewhere/deliberate")

	if got := envOrSeam("RDR_RECORDS"); got != "/somewhere/deliberate" {
		t.Errorf("the marker overrode an explicit env var: %q", got)
	}
}

// TestNoMarkerBindsNothing: discovery, never invention. Without a marker
// the seam is empty and callers fail exactly as they did before.
func TestNoMarkerBindsNothing(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	t.Chdir(dir)
	t.Setenv("RDR_RECORDS", "")

	if got := bindSeam(); len(got) != 0 {
		t.Errorf("bound %v with no marker present", got)
	}
}

// TestMarkerThatRefusesToSourceBindsNothing: the real markers guard
// themselves with `: "${WS:?…}"` so that sourcing them without the
// resolver fails loudly rather than binding half a seam. A marker that
// exits non-zero must bind nothing, not a partial map.
func TestMarkerThatRefusesToSourceBindsNothing(t *testing.T) {
	body := `: "${DEFINITELY_NOT_SET:?this marker refuses}"
RDR_RECORDS="/never/reached"
export RDR_RECORDS
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")

	if got := bindSeam()["RDR_RECORDS"]; got != "" {
		t.Errorf("a refusing marker still bound %q", got)
	}
}

// TestSeamIsUsedByTheCommands: end to end, with no environment at all —
// the shape the screenshot showed re-exporting on every single call.
func TestSeamIsUsedByTheCommands(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
export RDR_RECORDS
`
	// The fact table is named absolutely because the test chdirs away
	// from the repo; the seam is what is under test here, not the
	// table's own lookup.
	table := factTableForTest(t)
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_SOURCE_REPO", "")

	code, out, errb := runCapture(t, "status", "--facts", table)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0007-seam") {
		t.Errorf("the seam did not bind the records dir:\n%s", out)
	}
}

// allowMarkerLog opts one test back into marker-decided logging, which
// usageMarkerFallback refuses under `go test` so a fixture run can never
// reach a real workspace's log. The tests below build their own marker
// in a temp workspace, and the marker's say is the thing under test.
func allowMarkerLog(t *testing.T) {
	t.Helper()
	prev := usageMarkerFallback
	usageMarkerFallback = true
	t.Cleanup(func() { usageMarkerFallback = prev })
}

// TestUsageLogDefaultsUnderTheProjectDotDir: `/rdr-init` turns logging
// on for a project by writing a bare truthy value into the marker, and
// the binary decides where. `.rdr/` is this flow's repo-local run-output
// directory — the counterpart of the sibling codebase's `$REPO/.retrofit/`
// — and it self-ignores, so nothing written here can reach a commit.
func TestUsageLogDefaultsUnderTheProjectDotDir(t *testing.T) {
	allowMarkerLog(t)
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_USAGE_LOG="true"
export RDR_RECORDS RDR_USAGE_LOG
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_USAGE_LOG", "") // the marker alone must turn it on

	code, _, errb := runCapture(t, "inspect", "--json", "--filter", "path", "7")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	log := filepath.Join(project, ".rdr", "usage.jsonl")
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("no log at the default location %s: %v", log, err)
	}
	if !strings.Contains(string(raw), `"cmd":"inspect"`) {
		t.Errorf("log does not carry the invocation: %s", raw)
	}
}

// TestUsageLogExplicitPathBeatsTheDefault: a caller naming a file means
// that file, marker or no marker.
func TestUsageLogExplicitPathBeatsTheDefault(t *testing.T) {
	allowMarkerLog(t)
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_USAGE_LOG="true"
export RDR_RECORDS RDR_USAGE_LOG
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	explicit := filepath.Join(t.TempDir(), "elsewhere.jsonl")
	t.Setenv("RDR_USAGE_LOG", explicit)

	if code, _, errb := runCapture(t, "inspect", "--json", "--filter", "path", "7"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if _, err := os.ReadFile(explicit); err != nil {
		t.Errorf("the explicit path was not used: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".rdr", "usage.jsonl")); err == nil {
		t.Error("the default location was written to as well")
	}
}

// TestUsageLogOffSwitchBeatsTheMarker: a project that logs by default
// must be silenceable for one run without editing the marker.
func TestUsageLogOffSwitchBeatsTheMarker(t *testing.T) {
	allowMarkerLog(t)
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_USAGE_LOG="true"
export RDR_RECORDS RDR_USAGE_LOG
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_USAGE_LOG", "off")

	if code, _, errb := runCapture(t, "inspect", "--json", "--filter", "path", "7"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if _, err := os.Stat(filepath.Join(project, ".rdr", "usage.jsonl")); err == nil {
		t.Error("`off` still wrote a log")
	}
}

// TestUsageLogOnWithNoProjectStaysOff: discovery, never invention. A
// truthy setting with nowhere to put the file writes nothing rather than
// picking a directory nobody asked for.
func TestUsageLogOnWithNoProjectStaysOff(t *testing.T) {
	allowMarkerLog(t)
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("RDR_USAGE_LOG", "true")

	if got := usageLogPath(); got != "" {
		t.Errorf("with no project, logging resolved to %q", got)
	}
}

// TestUsageLogUnderAWorkspaceMarkerStaysOutOfTheRepo: the log goes beside
// the marker that turned it on. Under a workspace-scope marker that is
// `$WS`, above sibling repos which deliberately have no `.rdr/` — putting
// it in `$PROJECT/.rdr/` would invent seam structure the consumer opted
// out of, and scatter one shared setting into per-repo files nobody
// ignored.
func TestUsageLogUnderAWorkspaceMarkerStaysOutOfTheRepo(t *testing.T) {
	allowMarkerLog(t)
	body := `: "${WS:?needs the canonical resolver}"
RDR_RECORDS="$WS/consumer/docs/rdr"
RDR_USAGE_LOG="true"
export RDR_RECORDS RDR_USAGE_LOG
`
	project, _ := newProject(t, "shared", body)
	ws := filepath.Dir(project)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_USAGE_LOG", "")

	if code, _, errb := runCapture(t, "inspect", "--json", "--filter", "path", "7"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if _, err := os.ReadFile(filepath.Join(ws, "usage.jsonl")); err != nil {
		t.Errorf("no log beside the workspace marker: %v", err)
	}
	// The whole point: no .rdr/ was conjured inside the repo.
	if _, err := os.Stat(filepath.Join(project, ".rdr")); err == nil {
		t.Error("a .rdr/ was created in a workspace-scope project")
	}
}

// TestUsageLogIsOnePerWorkspace: siblings sharing a marker share its log,
// rather than each growing an untracked file of its own.
func TestUsageLogIsOnePerWorkspace(t *testing.T) {
	allowMarkerLog(t)
	body := `: "${WS:?needs the canonical resolver}"
RDR_RECORDS="$WS/consumer/docs/rdr"
RDR_USAGE_LOG="true"
export RDR_RECORDS RDR_USAGE_LOG
`
	project, _ := newProject(t, "shared", body)
	ws := filepath.Dir(project)

	// A second repo under the same workspace, inheriting the same marker.
	sibling := filepath.Join(ws, "sibling")
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = sibling
	if err := cmd.Run(); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_USAGE_LOG", "")

	for _, dir := range []string{project, sibling} {
		t.Chdir(dir)
		runCapture(t, "inspect", "--json", "--filter", "path", "7")
	}

	raw, err := os.ReadFile(filepath.Join(ws, "usage.jsonl"))
	if err != nil {
		t.Fatalf("no shared log: %v", err)
	}
	if n := strings.Count(strings.TrimSpace(string(raw)), "\n") + 1; n != 2 {
		t.Errorf("want both siblings' lines in one log, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(sibling, ".rdr")); err == nil {
		t.Error("the sibling grew its own .rdr/")
	}
}

// TestUsageLogMarkerFallbackIsRefusedUnderGoTest: the guarantee, proved
// the way it was broken. A marker with logging on, an env nobody set,
// a refusal-path run — the exact shape that used to write fixture rows
// into the workspace's production log on every `go test` — and no log
// may appear. No allowMarkerLog here: this test IS the default.
func TestUsageLogMarkerFallbackIsRefusedUnderGoTest(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_USAGE_LOG="true"
export RDR_RECORDS RDR_USAGE_LOG
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_USAGE_LOG", "") // the old contaminating shape: nothing explicit, marker on

	if code, _, _ := runCapture(t, "inspect", "--select", "0007:Z9", "7"); code == 0 {
		t.Fatal("the refusal path did not refuse")
	}
	if code, _, errb := runCapture(t, "inspect", "--json", "--filter", "path", "7"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if _, err := os.Stat(filepath.Join(project, ".rdr", "usage.jsonl")); err == nil {
		t.Error("go test wrote a row via marker fallback")
	}
}

// TestCommitRefusesARecordWithoutALintReceipt: §commit is the one
// mechanical choke point a stage passes through, so it is where a gate
// closed without lint is caught. The script is sourced exactly as a
// skill sources it, against a built binary, in a real repo.
func TestCommitRefusesARecordWithoutALintReceipt(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(home, "bin", "recs"), ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "skills", "rdr-commit.sh"))
	if err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(repo); err == nil {
		repo = resolved
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "commit.gpgsign", "false") // this test is about receipts; signing has its own
	git("-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "root")
	rec := filepath.Join(repo, "0007-receipt.md")
	body := "# Recommendation 0007: Receipt\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Draft\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(rec, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "usage.jsonl")

	commit := func() (int, string) {
		cmd := exec.Command("sh", "-c", `. "$1"; rdr_commit "docs(rdr): test" "$2"`, "_", script, rec)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "RDR_HOME="+home, "RDR_USAGE_LOG="+log, "RDR_RECORDS="+repo,
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("sh: %v\n%s", err, out)
		}
		return code, string(out)
	}

	if code, out := commit(); code != 1 || !strings.Contains(out, "stopped:commit-unlinted") {
		t.Fatalf("unlinted record: exit %d %q, want 1 stopped:commit-unlinted", code, out)
	}
	lint := exec.Command(filepath.Join(home, "bin", "recs"), "lint", "--records", repo, "0007")
	lint.Env = append(os.Environ(), "RDR_USAGE_LOG="+log)
	if out, err := lint.CombinedOutput(); err != nil {
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() > 1 {
			t.Fatalf("lint: %v\n%s", err, out)
		}
	}
	if code, out := commit(); code != 0 || !strings.Contains(out, "committed ") {
		t.Fatalf("linted record: exit %d %q, want a commit", code, out)
	}
	// With no log bound the check cannot vouch either way: note, and commit.
	if err := os.WriteFile(rec, []byte(body+"\nEdited.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	log = "off"
	if code, out := commit(); code != 0 || !strings.Contains(out, "note:stopped:no-usage-log") {
		t.Fatalf("no log: exit %d %q, want a commit with a note", code, out)
	}
}

// TestSeamBindsTheFactRoots: $RDR_EVIDENCE and $RDR_HOME come off the
// marker like every other var.
//
// They were deliberately absent while the projector read records and
// nothing else — §seam-bind names them as what this tool does NOT read.
// The fact table changed that: a fact about whether a lens ran is a fact
// about a directory, so the tool answering it has to know where the
// evidence tree is, and where its own table lives.
func TestSeamBindsTheFactRoots(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_EVIDENCE="$PROJECT/evidence"
RDR_HOME="$PROJECT/engine"
export RDR_RECORDS RDR_EVIDENCE RDR_HOME
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_EVIDENCE", "")
	t.Setenv("RDR_HOME", "")

	got := bindSeam()
	if want := filepath.Join(project, "evidence"); got["RDR_EVIDENCE"] != want {
		t.Errorf("RDR_EVIDENCE = %q, want %q", got["RDR_EVIDENCE"], want)
	}
	if want := filepath.Join(project, "engine"); got["RDR_HOME"] != want {
		t.Errorf("RDR_HOME = %q, want %q", got["RDR_HOME"], want)
	}
}

// TestUnboundEvidenceRootIsNotAnError: a marker that binds no evidence
// root binds nothing, and that is a legitimate consumer — the probes
// that would hang under it go absent rather than answering false.
func TestUnboundEvidenceRootIsNotAnError(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
export RDR_RECORDS
`
	project, records := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_EVIDENCE", "")

	got := bindSeam()
	if got["RDR_RECORDS"] != records {
		t.Errorf("RDR_RECORDS = %q, want %q", got["RDR_RECORDS"], records)
	}
	if v, ok := got["RDR_EVIDENCE"]; ok {
		t.Errorf("RDR_EVIDENCE = %q; an unbound var must stay unbound, not bind empty "+
			"(an empty root would anchor every probe at the filesystem root)", v)
	}
}

// TestCommitHonorsTheTargetRepoSigningPolicy: rdr_commit builds its commit
// with `commit-tree`, which ignores `commit.gpgsign` (that config drives
// only the `git commit` porcelain), so the helper must read the TARGET
// repo's policy itself and pass `-S`. A fake `gpg.program` stands in for
// the signer: it emits the SIG_CREATED status git looks for, or refuses
// when told to. Three claims: a repo that signs gets a gpgsig header; a
// signer that fails stops the commit and moves no ref; a repo-local
// `false` wins over a global `true` — the policy is the target's.
func TestCommitHonorsTheTargetRepoSigningPolicy(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "skills", "rdr-commit.sh"))
	if err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(t.TempDir(), "fake-gpg")
	if err := os.WriteFile(fake, []byte(`#!/bin/sh
# git invokes: gpg --status-fd=2 -bsau <key>; payload on stdin, signature on stdout.
if [ -n "$FAKE_GPG_FAIL" ]; then echo "fake gpg: refusing to sign" >&2; exit 2; fi
cat >/dev/null
echo "[GNUPG:] SIG_CREATED D 1 8 00 0 0000000000000000000000000000000000000000" >&2
printf -- '-----BEGIN PGP SIGNATURE-----\n\nZmFrZQ==\n-----END PGP SIGNATURE-----\n'
`), 0o755); err != nil {
		t.Fatal(err)
	}

	newRepo := func(sign string) (string, func(...string) string) {
		repo := t.TempDir()
		if resolved, err := filepath.EvalSymlinks(repo); err == nil {
			repo = resolved
		}
		git := func(args ...string) string {
			t.Helper()
			cmd := exec.Command("git", args...)
			cmd.Dir = repo
			cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
				"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
			return string(out)
		}
		git("init", "-q")
		git("config", "user.signingkey", "FAKEKEY")
		git("config", "gpg.program", fake)
		git("config", "commit.gpgsign", sign)
		git("commit", "-q", "--allow-empty", "--no-gpg-sign", "-m", "root")
		return repo, git
	}
	commit := func(repo, path string, extraEnv ...string) (int, string) {
		cmd := exec.Command("sh", "-c", `. "$1"; rdr_commit "docs(rdr): test" "$2"`, "_", script, path)
		cmd.Dir = repo
		// A global `commit.gpgsign=true` on the developer's machine must not
		// leak in: the helper reads the repo it commits TO, and the last
		// case below sets that to false on purpose.
		cmd.Env = append(os.Environ(), "RDR_USAGE_LOG=off", "RDR_RECORDS="+repo,
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		cmd.Env = append(cmd.Env, extraEnv...)
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("sh: %v\n%s", err, out)
		}
		return code, string(out)
	}
	write := func(repo, name, body string) string {
		p := filepath.Join(repo, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// 1. The target repo signs: the commit carries a signature.
	repo, git := newRepo("true")
	p := write(repo, "note.md", "signed\n")
	if code, out := commit(repo, p); code != 0 || !strings.Contains(out, "committed ") {
		t.Fatalf("signing repo: exit %d %q, want a commit", code, out)
	}
	if head := git("cat-file", "-p", "HEAD"); !strings.Contains(head, "gpgsig") {
		t.Errorf("the target repo's commit.gpgsign=true was not honored; commit has no gpgsig header:\n%s", head)
	}

	// 2. The signer fails: the commit stops, names the failure, and HEAD is unmoved.
	before := strings.TrimSpace(git("rev-parse", "HEAD"))
	write(repo, "note.md", "second edit\n")
	code, out := commit(repo, p, "FAKE_GPG_FAIL=1")
	if code != 1 || !strings.Contains(out, "stopped:commit-sign-failed") {
		t.Errorf("failed signer: exit %d %q, want 1 stopped:commit-sign-failed", code, out)
	}
	if after := strings.TrimSpace(git("rev-parse", "HEAD")); after != before {
		t.Errorf("a failed signature still moved HEAD %s -> %s", before, after)
	}
	if entries, _ := filepath.Glob(filepath.Join(repo, ".git", "rdr-skillidx-*")); len(entries) != 0 {
		t.Errorf("private index left behind after the refusal: %v", entries)
	}

	// 3. The target repo says false, whatever the global config says: no signature.
	repo2, git2 := newRepo("false")
	p2 := write(repo2, "note.md", "unsigned by policy\n")
	global := filepath.Join(t.TempDir(), "gitconfig-signs")
	if err := os.WriteFile(global, []byte("[commit]\n\tgpgsign = true\n[gpg]\n\tprogram = "+fake+"\n[user]\n\tsigningkey = FAKEKEY\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := commit(repo2, p2, "GIT_CONFIG_GLOBAL="+global); code != 0 {
		t.Fatalf("non-signing repo: exit %d %q, want a commit", code, out)
	}
	if head := git2("cat-file", "-p", "HEAD"); strings.Contains(head, "gpgsig") {
		t.Errorf("a repo with commit.gpgsign=false was signed anyway:\n%s", head)
	}
}

// newWorktreeFixture builds the topology a real git worktree has, without
// invoking git: a main checkout P with a `.git` DIRECTORY and the
// worktree's `commondir` file beneath it, and a worktree W — deliberately
// nested at `P/.claude/worktrees/wt`, mirroring the harness's own layout —
// whose `.git` is a FILE pointing back at P's private worktree git dir.
// `findMarker`'s `gitCommonProject` must resolve W's project to P by
// reading only these two files, no `git` process spawned.
func newWorktreeFixture(t *testing.T, markerBody string) (mainProject, worktree string) {
	t.Helper()
	root := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	mainProject = filepath.Join(root, "main")
	worktree = filepath.Join(mainProject, ".claude", "worktrees", "wt")
	privateGitDir := filepath.Join(mainProject, ".git", "worktrees", "wt")

	for _, dir := range []string{
		filepath.Join(mainProject, ".git"),
		privateGitDir,
		filepath.Join(worktree, "docs", "rdr"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The main checkout's own git dir IS the common dir; a real repo's
	// commondir file for its own git dir would be ".", but this fixture
	// only needs the worktree's pointer chain, which commondir supplies
	// as a path relative to the PRIVATE worktree git dir.
	if err := os.WriteFile(filepath.Join(privateGitDir, "commondir"), []byte("../..\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"),
		[]byte("gitdir: "+privateGitDir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := "# Recommendation 0007: Seam\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(worktree, "docs", "rdr", "0007-seam.md"), []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	markerDir := filepath.Join(mainProject, ".rdr")
	if err := os.MkdirAll(markerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(markerDir, "workspace"), []byte(markerBody), 0o600); err != nil {
		t.Fatal(err)
	}
	return mainProject, worktree
}

// TestWorktreeFollowsTheGitdirPointer: a worktree's `.git` is a FILE, not
// a directory, so the pre-fix `findMarker` took the worktree ITSELF as
// `project` and looked for `<worktree>/.rdr/workspace` — which is never
// there, because the marker lives in the main checkout. The fix follows
// the pointer (via `commondir`, no `git` spawned) so the marker still
// resolves, and anchors the bound records on `$TOPLEVEL` — the worktree
// being edited — rather than `$PROJECT`, the main checkout beside it.
func TestWorktreeFollowsTheGitdirPointer(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
: "${TOPLEVEL:?needs the canonical resolver}"
RDR_RECORDS="$TOPLEVEL/docs/rdr"
RDR_SOURCE_REPO="$TOPLEVEL"
export RDR_RECORDS RDR_SOURCE_REPO
`
	mainProject, worktree := newWorktreeFixture(t, body)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_SOURCE_REPO", "")

	t.Run("from the worktree root", func(t *testing.T) {
		t.Chdir(worktree)
		got := bindSeam()
		want := filepath.Join(worktree, "docs", "rdr")
		if got["RDR_RECORDS"] != want {
			t.Errorf("RDR_RECORDS = %q, want the worktree's %q", got["RDR_RECORDS"], want)
		}
		if got["RDR_SOURCE_REPO"] != worktree {
			t.Errorf("RDR_SOURCE_REPO = %q, want %q", got["RDR_SOURCE_REPO"], worktree)
		}
	})

	t.Run("from a subdirectory of the worktree", func(t *testing.T) {
		sub := filepath.Join(worktree, "docs")
		t.Chdir(sub)
		got := bindSeam()
		want := filepath.Join(worktree, "docs", "rdr")
		if got["RDR_RECORDS"] != want {
			t.Errorf("RDR_RECORDS = %q, want %q", got["RDR_RECORDS"], want)
		}
	})

	t.Run("from the main checkout, no worktree involved", func(t *testing.T) {
		mainRecords := filepath.Join(mainProject, "docs", "rdr")
		if err := os.MkdirAll(mainRecords, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(mainProject)
		got := bindSeam()
		if got["RDR_RECORDS"] != mainRecords {
			t.Errorf("RDR_RECORDS = %q, want %q — outside a worktree TOPLEVEL == PROJECT", got["RDR_RECORDS"], mainRecords)
		}
	})
}

// TestStaleMarkerBindsMainCheckoutRefuses: a marker written before TOPLEVEL
// existed anchors `<CONSUMER>_ROOT="$PROJECT"`, so from a worktree it binds
// the MAIN checkout's records — a real corpus, just the wrong one, which is
// the false pass this fix removes. The refusal fires only when the bound
// records land under `project` and NOT under `toplevel`, so a worktree that
// happens to sit inside the main checkout is not itself mistaken for the
// stale case.
func TestStaleMarkerBindsMainCheckoutRefuses(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
export RDR_RECORDS
`
	mainProject, worktree := newWorktreeFixture(t, body)
	t.Setenv("RDR_RECORDS", "")

	t.Run("from the worktree the stale marker refuses", func(t *testing.T) {
		t.Chdir(worktree)
		got := bindSeam()
		if v, ok := got["RDR_RECORDS"]; ok {
			t.Errorf("a stale marker still bound RDR_RECORDS = %q", v)
		}
		why := markerRefusal()
		if !strings.Contains(why, "stopped:marker-binds-main-checkout") {
			t.Errorf("wrong or missing refusal: %q", why)
		}
	})

	t.Run("from the main checkout the same marker still binds", func(t *testing.T) {
		mainRecords := filepath.Join(mainProject, "docs", "rdr")
		if err := os.MkdirAll(mainRecords, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(mainProject)
		got := bindSeam()
		if got["RDR_RECORDS"] != mainRecords {
			t.Errorf("RDR_RECORDS = %q, want %q — the marker is correct outside a worktree", got["RDR_RECORDS"], mainRecords)
		}
		if why := markerRefusal(); why != "" {
			t.Errorf("the main checkout's own bind refused: %q", why)
		}
	})
}

// TestEngineCwdRefuses: a workspace marker admits the engine checkout as a
// member, so a call made with the engine as cwd bound the marker's
// consumer — a leg that `cd $RDR_HOME && ./bin/rdr-gate ground 0029`'d
// resolved another project's 0029 and ground against it. From the engine
// the seam refuses and names the fix; from the consumer the same marker
// binds as before.
func TestEngineCwdRefuses(t *testing.T) {
	project, records := newProject(t, "shared", `: "${WS:?needs the canonical resolver}"
RDR_HOME="$WS/engine"
RDR_RECORDS="$WS/consumer/docs/rdr"
export RDR_HOME RDR_RECORDS
`)
	engine := filepath.Join(filepath.Dir(project), "engine")
	if err := os.MkdirAll(filepath.Join(engine, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RDR_RECORDS", "")

	t.Run("from the engine the shared marker refuses every consumer var", func(t *testing.T) {
		t.Chdir(engine)
		got := bindSeam()
		if v, ok := got["RDR_RECORDS"]; ok {
			t.Errorf("the engine cwd bound RDR_RECORDS = %q", v)
		}
		if got["RDR_HOME"] != engine {
			t.Errorf("RDR_HOME = %q, want %q — the one var the engine cwd vouches for", got["RDR_HOME"], engine)
		}
		why := markerRefusal()
		if !strings.Contains(why, "stopped:engine-cwd") || !strings.Contains(why, records) {
			t.Errorf("wrong or missing refusal: %q", why)
		}
	})

	t.Run("a bare record number from the engine stops on the refusal", func(t *testing.T) {
		t.Chdir(engine)
		code, _, errb := runCapture(t, "inspect", "0007")
		if code != 2 || !strings.Contains(errb, "stopped:engine-cwd") {
			t.Errorf("exit %d: %s", code, errb)
		}
	})

	t.Run("from the consumer the same marker binds", func(t *testing.T) {
		t.Chdir(project)
		got := bindSeam()
		if got["RDR_RECORDS"] != records {
			t.Errorf("RDR_RECORDS = %q, want %q", got["RDR_RECORDS"], records)
		}
		if why := markerRefusal(); why != "" {
			t.Errorf("the consumer's own bind refused: %q", why)
		}
	})
}

// recordsTreeWithArtifacts builds one records dir holding record `num`
// and, when withArtifacts is true, an artifacts folder beside it whose
// four files give every impl-artifact fact a non-trivial value: a
// COMPLETE capsule, one REQ line with its coverage row, and one still-open
// deviation. Without artifacts, every impl_* fact is absent — the shape
// tree B needs to prove --records didn't just happen to agree with the
// env by coincidence.
func recordsTreeWithArtifacts(t *testing.T, num string, withArtifacts bool) string {
	t.Helper()
	dir := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	slug := num + "-seam-flag-bind"
	rec := "# Recommendation " + num + ": Seam Flag Bind\n\n## Metadata\n\n" +
		"- **Date**: 2026-08-01\n- **Status**: Final\n- **Profile**: standard\n\n" +
		"## Problem Statement\n\nSynthetic.\n"
	if err := os.WriteFile(filepath.Join(dir, slug+".md"), []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	if !withArtifacts {
		return dir
	}
	artifacts := filepath.Join(dir, slug, "artifacts")
	if err := os.MkdirAll(artifacts, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"status.md":     "phase: done\nstate: COMPLETE\n",
		"req-list.md":   "- **[REQ-1]** \"first.\"\n",
		"coverage.md":   "| Requirement | Test |\n| --- | --- |\n| `REQ-1` | `TestOne` |\n",
		"deviations.md": "- **Status: needs author decision (recorded, run continued)**\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(artifacts, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestRecordsFlagRebindsTheArtifactRoots is defect 2's acceptance:
// `--records DIR` picks the record file from DIR, but NewFactEnv's roots
// (artifacts, evidence) used to keep reading $RDR_RECORDS from the
// environment or marker regardless — so `status --records X --tags NNNN`
// could read the record in X and its artifacts from a different tree,
// silently. Tree A carries a real artifact ledger; tree B carries none.
// All three invocations below must read tree A's artifacts, or the tag
// vectors would disagree on impl_state/impl_open_decisions/etc.
func TestRecordsFlagRebindsTheArtifactRoots(t *testing.T) {
	num := "0058"
	treeA := recordsTreeWithArtifacts(t, num, true)
	treeB := recordsTreeWithArtifacts(t, num, false)
	table := factTableForTest(t)

	// Run from a cwd unrelated to either tree, so no marker or cwd-relative
	// resolution can accidentally supply the right answer.
	elsewhere := t.TempDir()
	t.Chdir(elsewhere)

	tags := func(label string, env map[string]string, args ...string) string {
		t.Helper()
		for k, v := range env {
			t.Setenv(k, v)
		}
		full := append([]string{"status", "--tags", "--facts", table}, args...)
		full = append(full, num)
		code, out, errb := runCapture(t, full...)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", label, code, errb)
		}
		return out
	}

	viaEnv := tags("env only", map[string]string{"RDR_RECORDS": treeA, "RDR_SOURCE_REPO": ""})
	viaFlag := tags("flag only", map[string]string{"RDR_RECORDS": "", "RDR_SOURCE_REPO": ""}, "--records", treeA)
	viaFlagOverEnv := tags("flag overrides a different env", map[string]string{"RDR_RECORDS": treeB, "RDR_SOURCE_REPO": ""}, "--records", treeA)

	if viaEnv != viaFlag {
		t.Errorf("env-bound and flag-bound tag vectors disagree:\nenv:  %s\nflag: %s", viaEnv, viaFlag)
	}
	if viaFlag != viaFlagOverEnv {
		t.Errorf("--records did not outrank a conflicting $RDR_RECORDS for the artifact roots:\n"+
			"flag alone:      %s\nflag over env B: %s", viaFlag, viaFlagOverEnv)
	}
	for _, want := range []string{"impl_state=COMPLETE", "impl_open_decisions=1+"} {
		if !strings.Contains(viaFlagOverEnv, want) {
			t.Errorf("tag vector missing %q (artifacts not read from the --records tree):\n%s", want, viaFlagOverEnv)
		}
	}
}
