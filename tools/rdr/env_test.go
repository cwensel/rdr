package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runEnv drives the verb the way run() does, with the seam cache cleared
// so a previous test's cwd cannot answer for this one.
func runEnv(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	seamMu.Lock()
	seamDir, seamCache, seamRefusal = "\x00unset", nil, ""
	seamMu.Unlock()
	flagBoundMu.Lock()
	flagBound = map[string]string{}
	flagBoundMu.Unlock()
	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := declareFlags("env", fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out, errb bytes.Buffer
	code := envCmd(f, &out, &errb)
	return out.String(), errb.String(), code
}

// parseEnvText reads the emitted k='v' lines back, unquoting, so a test
// asserts on values rather than on formatting.
func parseEnvText(s string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		v = strings.TrimSuffix(strings.TrimPrefix(v, "'"), "'")
		m[k] = strings.ReplaceAll(v, `'\''`, "'")
	}
	return m
}

const envMarkerBody = `: "${PROJECT:?needs the canonical resolver}"
RDR_HOME="$PROJECT/engine"
RDR_RECORDS="$PROJECT/docs/rdr"
RDR_EVIDENCE="$PROJECT/docs/rdr"
RDR_ENV="$PROJECT/.rdr/env.md"
RDR_RESOURCES="$PROJECT/.rdr/resources.md"
RDR_SOURCE_REPO="$PROJECT"
RDR_AUTOCOMMIT="true"
RDR_MODEL_CEILING="test-ceiling-model"
export RDR_HOME RDR_RECORDS RDR_EVIDENCE RDR_ENV RDR_RESOURCES RDR_SOURCE_REPO RDR_AUTOCOMMIT RDR_MODEL_CEILING
`

// TestEnvPublishesTheWholeContract: the verb exists because three vars —
// RDR_ENV, RDR_RESOURCES, RDR_AUTOCOMMIT — were the only reason a skill
// still carried a shell resolver. This tool reads none of them, so a
// regression that drops them from seamVars would be invisible to every
// other test and would silently resurrect the block.
func TestEnvPublishesTheWholeContract(t *testing.T) {
	project, _ := newProject(t, "local", envMarkerBody)
	t.Chdir(project)
	for _, v := range seamVars {
		t.Setenv(v, "")
	}

	out, errb, code := runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	got := parseEnvText(out)
	for _, want := range []string{
		"RDR_HOME", "RDR_RECORDS", "RDR_EVIDENCE", "RDR_ENV",
		"RDR_RESOURCES", "RDR_SOURCE_REPO", "RDR_AUTOCOMMIT", "RDR_MODEL_CEILING",
		envMarkerVar, envProjectVar,
	} {
		if got[want] == "" {
			t.Errorf("env omits %s:\n%s", want, out)
		}
	}
	if got[envProjectVar] != project {
		t.Errorf("%s = %q, want %q", envProjectVar, got[envProjectVar], project)
	}
	if got["RDR_RECORDS"] != filepath.Join(project, "docs", "rdr") {
		t.Errorf("RDR_RECORDS = %q", got["RDR_RECORDS"])
	}
}

// TestEnvAnswersFromTheMarkerNotTheEnvironment is the reason this verb
// does not use envOrSeam, and it is the one test that must never be
// relaxed into agreement with TestEnvironmentOutranksTheMarker.
//
// Those two look contradictory and are not. `--records` names ONE dir for
// ONE invocation, so an exported value there is a decision. Publishing the
// whole seam is different: in a workspace where a repo-local marker sits
// beside a shared one the two seams are disjoint, and an inherited RDR_*
// is far likelier a leak from the previous call than an intention. If the
// verb echoed it back the caller would eval it, and the leak would
// outlive the turn instead of dying with it — which is exactly the
// property §seam-bind's `. "$MARKER"` had for free, since sourcing
// overwrites.
//
// The failure it prevents is silent: records read from one project while
// RDR_EVIDENCE resolves from another makes every lens fact read false, so
// a record with four finished lenses reports as never lensed and the
// router re-runs them over the evidence already on disk.
func TestEnvAnswersFromTheMarkerNotTheEnvironment(t *testing.T) {
	project, _ := newProject(t, "local", envMarkerBody)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "/leaked/from/another/project")
	t.Setenv("RDR_EVIDENCE", "/leaked/from/another/project")

	out, errb, code := runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	got := parseEnvText(out)
	want := filepath.Join(project, "docs", "rdr")
	if got["RDR_RECORDS"] != want {
		t.Errorf("a leaked RDR_RECORDS survived: got %q, want the marker's %q", got["RDR_RECORDS"], want)
	}
	if got["RDR_EVIDENCE"] != want {
		t.Errorf("a leaked RDR_EVIDENCE survived: got %q, want the marker's %q", got["RDR_EVIDENCE"], want)
	}
	// The flag path keeps the opposite rule, deliberately.
	if got := envOrSeam("RDR_RECORDS"); got != "/leaked/from/another/project" {
		t.Errorf("envOrSeam changed meaning: %q", got)
	}
}

// TestEnvReportsWhichSeamItBound: coexisting scopes are the norm, not the
// exception — one repo carries a local marker while its siblings share the
// workspace one. RDR_PROJECT is what lets a caller assert the seam belongs
// to the repo it is standing in, so a foreign bind stops instead of
// proceeding on the wrong corpus.
func TestEnvReportsWhichSeamItBound(t *testing.T) {
	project, _ := newProject(t, "local", envMarkerBody)
	ws := filepath.Dir(project)
	shared := `: "${WS:?needs the canonical resolver}"
RDR_RECORDS="$WS/somewhere-else"
export RDR_RECORDS
`
	if err := os.WriteFile(filepath.Join(ws, ".rdr-workspace"), []byte(shared), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	for _, v := range seamVars {
		t.Setenv(v, "")
	}

	out, errb, code := runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	got := parseEnvText(out)
	if want := filepath.Join(project, ".rdr", "workspace"); got[envMarkerVar] != want {
		t.Errorf("%s = %q, want the nearest %q", envMarkerVar, got[envMarkerVar], want)
	}
	if got["RDR_RECORDS"] == filepath.Join(ws, "somewhere-else") {
		t.Error("the shared marker won over the repo-local one")
	}
}

// TestEnvStopsWithoutAMarker keeps §seam-bind's wording and the two
// directories it named. A caller who has read this message before must not
// have to learn a new one because the mechanism moved into the binary.
func TestEnvStopsWithoutAMarker(t *testing.T) {
	dir := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	t.Chdir(dir)

	out, errb, code := runEnv(t)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if out != "" {
		t.Errorf("emitted a seam with no marker:\n%s", out)
	}
	if !strings.Contains(errb, "stopped:no-marker") {
		t.Errorf("wrong reason: %s", errb)
	}
	// Both directories, so the fix is actionable without a second call.
	if !strings.Contains(errb, filepath.Join(dir, ".rdr")) ||
		!strings.Contains(errb, filepath.Dir(dir)) {
		t.Errorf("the message drops a directory it looked in: %s", errb)
	}
}

// TestEnvStopsOutsideAProject: no git topology, no seam. The shell block
// stopped here too, with this reason.
func TestEnvStopsOutsideAProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	_, errb, code := runEnv(t)
	// A temp dir under a git checkout would resolve a project; only assert
	// the pairing of reason and code.
	if code == 0 {
		t.Skip("temp dir sits inside a git project")
	}
	if code != 1 || !strings.Contains(errb, "stopped:") {
		t.Errorf("exit %d, err %q", code, errb)
	}
}

// TestEnvQuotesForEval: the values are paths someone typed, so a space is
// ordinary. Unquoted, `eval` would split one path into two words and bind
// a truncated seam — the same trap `status --tags` already paid for once,
// where a prose fact arrived cut at its first space.
func TestEnvQuotesForEval(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/my records"
RDR_HOME="$PROJECT/engine"
export RDR_RECORDS RDR_HOME
`
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	for _, v := range seamVars {
		t.Setenv(v, "")
	}

	out, errb, code := runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	want := filepath.Join(project, "docs", "my records")
	if got := parseEnvText(out)["RDR_RECORDS"]; got != want {
		t.Errorf("RDR_RECORDS = %q, want %q", got, want)
	}
	// Prove it against a real shell, not only against the parser above.
	script := `eval "$(cat)"
printf '%s' "$RDR_RECORDS"`
	cmd := exec.Command("sh", "-c", script)
	cmd.Stdin = strings.NewReader(out)
	cmd.Env = []string{}
	got, err := cmd.Output()
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if string(got) != want {
		t.Errorf("after eval RDR_RECORDS = %q, want %q", got, want)
	}
}

// TestEnvJSONCarriesTheSameSeam: the neutral form for a caller that will
// not eval — same values, so the two renderings cannot drift.
func TestEnvJSONCarriesTheSameSeam(t *testing.T) {
	project, _ := newProject(t, "local", envMarkerBody)
	t.Chdir(project)
	for _, v := range seamVars {
		t.Setenv(v, "")
	}

	text, _, code := runEnv(t)
	if code != 0 {
		t.Fatal("text form failed")
	}
	out, errb, code := runEnv(t, "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	want := parseEnvText(text)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: json %q, text %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("json has %d keys, text %d", len(got), len(want))
	}
}

// TestEnvNamesItselfInTheUsageLog is the v12r lesson applied to a new
// verb: a facet the log cannot name reads as never called, and an audit
// that prunes on "no calls recorded" would delete the one call every
// skill run makes.
func TestEnvNamesItselfInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "text"},
		{[]string{"--json"}, "json"},
	} {
		fs := flag.NewFlagSet("env", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("env", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if got := usageFacet("env", f, ""); got != c.want {
			t.Errorf("env %v logs as %q, want %q", c.args, got, c.want)
		}
	}
}

// anchoredMarkerBody is the marker /rdr-init writes: the same contract as
// envMarkerBody, plus the self-identifying anchor that makes a foreign
// bind refuse instead of fabricating paths from whatever $PROJECT held.
func anchoredMarkerBody(anchor string) string {
	return `: "${PROJECT:?needs the canonical resolver}"
RDR_PROJECT_ANCHOR="` + anchor + `"
[ "$PROJECT" = "$RDR_PROJECT_ANCHOR" ] || {
  echo "stopped:foreign-project marker=$RDR_PROJECT_ANCHOR/.rdr/workspace describes=$RDR_PROJECT_ANCHOR handed=$PROJECT" >&2
  return 1 2>/dev/null || exit 1
}
` + envMarkerBody
}

// TestEnvBindsAnAnchoredMarkerUnchanged is the compatibility half of the
// anchor guard: a marker that DOES describe this project must bind exactly
// as it did before the guard existed. The guard is only allowed to cost a
// foreign caller something.
func TestEnvBindsAnAnchoredMarkerUnchanged(t *testing.T) {
	project, records := newProject(t, "local", "")
	body := anchoredMarkerBody(project)
	if err := os.WriteFile(filepath.Join(project, ".rdr", "workspace"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	out, errb, code := runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d, want 0 — an anchored marker on its OWN project must bind: %s", code, errb)
	}
	got := parseEnvText(out)
	if got["RDR_RECORDS"] != records {
		t.Errorf("RDR_RECORDS = %q, want %q", got["RDR_RECORDS"], records)
	}
	if got[envProjectVar] != project {
		t.Errorf("%s = %q, want %q", envProjectVar, got[envProjectVar], project)
	}
}

// TestEnvStopsOnAMarkerThatRefuses is 2pb4's acceptance, at the layer that
// would otherwise defeat it.
//
// A marker whose anchor names another project refuses to bind. Before this,
// that refusal was swallowed: bindSeam returned an empty map, which is
// indistinguishable from "no marker here", so `env` exited 0 publishing
// only RDR_MARKER and RDR_PROJECT. The caller's own §seam-bind assertion
// then PASSED — this binary computes RDR_PROJECT from git topology, not
// from the marker, so it matched — and the flow proceeded with every
// contract var unset. That is the same silent bind one layer later.
func TestEnvStopsOnAMarkerThatRefuses(t *testing.T) {
	project, _ := newProject(t, "local", "")
	// The anchor names a project this one is not.
	body := anchoredMarkerBody(filepath.Join(filepath.Dir(project), "other-project"))
	if err := os.WriteFile(filepath.Join(project, ".rdr", "workspace"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	out, errb, code := runEnv(t)
	if code != 1 {
		t.Errorf("exit %d, want 1 — a refusing marker must stop, not publish a partial seam", code)
	}
	if out != "" {
		t.Errorf("published a seam from a marker that refused:\n%s", out)
	}
	if !strings.Contains(errb, "stopped:marker-refused") {
		t.Errorf("the refusal is not named as one: %s", errb)
	}
	// The marker's OWN message must survive: it names the project the
	// marker describes and the one it was handed, which is the whole
	// diagnosis and is not reconstructable from this side.
	if !strings.Contains(errb, "stopped:foreign-project") ||
		!strings.Contains(errb, "other-project") ||
		!strings.Contains(errb, project) {
		t.Errorf("the marker's own reason did not survive: %s", errb)
	}
}

// TestEnvRefusalIsNotAMissingMarker pins the distinction itself. The two
// stop for different reasons and a caller acts differently on each: no
// marker means run /rdr-init, a refusal means this repo is not the one the
// marker describes. Collapsing them sends the caller to the wrong fix.
func TestEnvRefusalIsNotAMissingMarker(t *testing.T) {
	project, _ := newProject(t, "local", "")
	body := anchoredMarkerBody(filepath.Join(filepath.Dir(project), "elsewhere"))
	if err := os.WriteFile(filepath.Join(project, ".rdr", "workspace"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	_, errb, _ := runEnv(t)
	if strings.Contains(errb, "stopped:no-marker") {
		t.Errorf("a marker that refused is reported as absent — the caller would run /rdr-init "+
			"and rewrite a marker that is correct for its own project: %s", errb)
	}
}

// TestEnvFromAWorktreeBindsTheWorktreesRecords is `env`'s acceptance of the
// worktree fix: a repo-local marker written with $TOPLEVEL in its body
// binds the worktree being edited, and RDR_PROJECT still names the main
// checkout — the anchor a caller uses to assert the seam belongs to the
// repo it is standing in.
func TestEnvFromAWorktreeBindsTheWorktreesRecords(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
: "${TOPLEVEL:?needs the canonical resolver}"
RDR_RECORDS="$TOPLEVEL/docs/rdr"
export RDR_RECORDS
`
	mainProject, worktree := newWorktreeFixture(t, body)
	t.Setenv("RDR_RECORDS", "")

	t.Chdir(worktree)
	out, errb, code := runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	got := parseEnvText(out)
	if want := filepath.Join(worktree, "docs", "rdr"); got["RDR_RECORDS"] != want {
		t.Errorf("RDR_RECORDS = %q, want the worktree's %q", got["RDR_RECORDS"], want)
	}
	if got[envProjectVar] != mainProject {
		t.Errorf("%s = %q, want the main checkout %q", envProjectVar, got[envProjectVar], mainProject)
	}

	// From the main checkout itself, TOPLEVEL == PROJECT and the same
	// marker binds the same way it always did.
	mainRecords := filepath.Join(mainProject, "docs", "rdr")
	if err := os.MkdirAll(mainRecords, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(mainProject)
	out, errb, code = runEnv(t)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if got := parseEnvText(out)["RDR_RECORDS"]; got != mainRecords {
		t.Errorf("RDR_RECORDS = %q, want %q", got, mainRecords)
	}
}

// TestEnvRefusesAStaleMarkerFromAWorktree is the `env` layer of the same
// stale-marker refusal seam_test.go proves at bindSeam: a pre-fix marker
// anchors records on $PROJECT, so from a worktree `env` must stop rather
// than publish the main checkout's corpus under the worktree's name.
func TestEnvRefusesAStaleMarkerFromAWorktree(t *testing.T) {
	body := `: "${PROJECT:?needs the canonical resolver}"
RDR_RECORDS="$PROJECT/docs/rdr"
export RDR_RECORDS
`
	mainProject, worktree := newWorktreeFixture(t, body)
	t.Setenv("RDR_RECORDS", "")

	t.Chdir(worktree)
	out, errb, code := runEnv(t)
	if code != 1 {
		t.Errorf("exit %d, want 1 — a stale marker must not bind the main checkout's records", code)
	}
	if out != "" {
		t.Errorf("published a seam from a stale marker:\n%s", out)
	}
	if !strings.Contains(errb, "stopped:marker-binds-main-checkout") {
		t.Errorf("wrong reason: %s", errb)
	}

	mainRecords := filepath.Join(mainProject, "docs", "rdr")
	if err := os.MkdirAll(mainRecords, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(mainProject)
	if _, errb, code := runEnv(t); code != 0 {
		t.Errorf("exit %d, want 0 — the same marker binds normally outside a worktree: %s", code, errb)
	}
}

// TestStatusArityNamesItselfInTheUsageLog is the same v12r lesson for the
// set arity. `status NNNN` and `status NNNN NNNN` are one verb at two
// costs, and the worklist is a third — a log that could not tell them
// apart would average a 47ms call with a 2.0s scan and hide whichever one
// a skill actually pays.
func TestStatusArityNamesItselfInTheUsageLog(t *testing.T) {
	for _, c := range []struct {
		name   string
		args   []string
		target string
		argc   int
		want   string
	}{
		{"worklist", nil, "", 0, "worklist"},
		{"worklist json", []string{"--json"}, "", 0, "worklist:json"},
		{"one record", nil, "0106", 1, "record"},
		{"one record tags", []string{"--tags"}, "0106", 1, "record:tags"},
		{"a set", nil, "0122 0123", 2, "records"},
		{"a set json", []string{"--json"}, "0122 0123", 2, "records:json"},
	} {
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := declareFlags("status", fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%s: parse: %v", c.name, err)
		}
		f.argc = c.argc
		if got := usageFacet("status", f, c.target); got != c.want {
			t.Errorf("%s logs as %q, want %q", c.name, got, c.want)
		}
	}
}
