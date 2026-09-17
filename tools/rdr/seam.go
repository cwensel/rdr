package main

// The seam binds itself.
//
// Every var this binary reads — $RDR_RECORDS, $RDR_SOURCE_REPO,
// $RDR_EVIDENCE, $RDR_HOME — is written in a marker file the flow
// already maintains. Until now only a shell could read it, so every call
// site had to carry the seam through the turn: either a fifteen-line
// resolver block re-run verbatim, or an
// `export RDR_HOME=… RDR_RECORDS=… RDR_EVIDENCE=… RDR_ENV=…` prefix
// repeated on invocation after invocation, because shell state dies
// between tool calls and the harness starts each one fresh.
//
// That is baggage on the one resource that actually costs: the turn. A
// re-exported prefix is bytes on every call and a fresh chance to get a
// path wrong; a re-run resolver is bytes plus a second failure surface.
// Neither carries information the marker does not already hold.
//
// So the binary does what the shell block did. From the working
// directory it finds the git project, applies the flow's own
// nearest-marker-wins rule, sources the marker with `sh` (markers are
// plain assignments, and expand `$WS`/`$PROJECT` internally, so they
// need a real shell, not a regex), and reads the values back. An
// explicit flag or an inherited environment variable always wins — this
// only fills a gap that would otherwise have been an error.
//
// It is discovery, never invention: no marker means nothing is bound and
// the caller fails exactly as before.
//
// A git worktree complicates "the project": its `.git` is a FILE, not a
// directory, pointing at the main checkout, so the marker LOOKUP still
// resolves there — but a repo-local marker's records must bind to the
// worktree being edited, not the main checkout beside it. `findMarker`
// tells the two apart as `project` (the main checkout, for lookup) and
// `toplevel` (the worktree, for anchoring); a marker written before this
// distinction existed refuses rather than silently binding the wrong
// tree.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// seamVars are the marker values this binary can use. The marker exports
// more — path maps, per-consumer conveniences — and none of it is this
// tool's business.
//
// RDR_EVIDENCE and RDR_HOME joined the list when the fact table landed
// (facts.go). Neither is a records path: RDR_EVIDENCE roots the exact-path
// probes a fact declares, and RDR_HOME is where the fact table itself
// lives. Both were previously "what the seam binds and this tool does
// not" — the projector read records and nothing else. A fact about
// whether a lens ran is a fact about a directory, so the tool that
// answers it has to know where that directory is.
// RDR_ENV, RDR_RESOURCES and RDR_AUTOCOMMIT joined last, and none of them
// is read by this tool at all. They are here because `recs env` publishes
// the seam CONTRACT rather than this binary's own appetite — the three
// vars a skill still needed a shell resolver for were exactly the three
// the projector never opened, so the resolver survived for them alone.
// RDR_MODEL_CEILING joined for the same reason RDR_AUTOCOMMIT did: it is
// a behavioural var this binary never reads, published because `recs env`
// carries the seam CONTRACT. rdr-common §model-ceiling names it as the
// middle rung of the resolution (`--model-ceiling` arg > marker var >
// unset), and without it here that rung was documented and unreachable —
// a marker could set it, `recs env` would not emit it, and §seam-bind's
// `eval` would leave it unset in the shell that spawns.
var seamVars = []string{
	"RDR_RECORDS", "RDR_SOURCE_REPO", "RDR_USAGE_LOG", "RDR_EVIDENCE", "RDR_HOME",
	"RDR_ENV", "RDR_RESOURCES", "RDR_AUTOCOMMIT", "RDR_MODEL_CEILING",
}

// seam resolves once per working directory. A projection may consult it
// several times and the marker cannot change mid-run, so the common case
// binds once — but the cwd IS the lookup key, so a process that moves
// (a test binary, or any long-lived caller) must not keep answering from
// the directory it started in.
//
// seamRefusal rides with the cache, under the same lock: it carries why a
// marker that EXISTS bound nothing — the message the marker itself
// printed. `env` is the one caller that must tell that apart from an
// unconfigured repo, and it is per-cwd for the same reason the cache is.
var (
	seamMu      sync.Mutex
	seamDir     string
	seamCache   map[string]string
	seamRefusal string
)

func seam() map[string]string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	seamMu.Lock()
	defer seamMu.Unlock()
	if seamCache != nil && seamDir == cwd {
		return seamCache
	}
	seamRefusal = ""
	seamDir, seamCache = cwd, bindSeam()
	return seamCache
}

// markerRefusal reports why the bound marker refused, or "" when none did.
// It forces the bind first, so a caller need not have read a var to ask.
func markerRefusal() string {
	seam()
	seamMu.Lock()
	defer seamMu.Unlock()
	return seamRefusal
}

// bindSeam finds the marker and returns the vars it exports. An empty
// map means no marker was found, which is not an error: the flow works
// without one, and callers already handle unbound vars.
func bindSeam() map[string]string {
	out := map[string]string{}
	marker, project, ws, toplevel := findMarker()
	if marker == "" {
		return out
	}
	// The marker guards itself twice, and the two guards fail for
	// different reasons. `: "${WS:?…}"` / `"${PROJECT:?…}"` prove the
	// RESOLVER ran — supplying both is what makes this the canonical read
	// rather than the direct source the marker refuses. The anchor check
	// proves the resolver resolved THIS marker's project: a marker whose
	// `$RDR_PROJECT_ANCHOR` names another project refuses rather than
	// deriving every path below it from a foreign `$PROJECT`.
	//
	// A refusal is NOT "no marker", and the difference is the whole point.
	// Swallowing it would hand back an empty map, which reads as an
	// unconfigured repo — so a caller would proceed with unset vars
	// instead of stopping, which is the silent bind this guard exists to
	// end. The marker's own message is captured and carried out, because
	// it names the two paths (described, handed) that a caller needs to
	// see and this binary cannot reconstruct.
	script := `set -e
. "$1" >/dev/null || exit 1
for v in ` + strings.Join(seamVars, " ") + `; do
  eval "val=\$$v"
  [ -n "$val" ] && printf '%s=%s\n' "$v" "$val"
done
exit 0`
	cmd := exec.Command("sh", "-c", script, "_", marker)
	cmd.Env = append(os.Environ(), "PROJECT="+project, "WS="+ws, "TOPLEVEL="+toplevel)
	cmd.Dir = project
	var refusal strings.Builder
	cmd.Stderr = &refusal
	stdout, err := cmd.Output()
	if err != nil {
		seamRefusal = strings.TrimSpace(refusal.String())
		if seamRefusal == "" {
			seamRefusal = "the marker could not be sourced"
		}
		return out
	}
	for _, line := range strings.Split(string(stdout), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || v == "" {
			continue
		}
		out[k] = v
	}

	// The engine checkout is every consumer's. A workspace marker admits
	// it as a member (2pb4: the skills are read from it), so `cd $RDR_HOME
	// && ./bin/recs …` binds whichever consumer the SHARED marker names —
	// one leg ground a record number from there against another project's
	// corpus, and that project had a record of the same number. From the
	// engine there is no cwd to bind a consumer by; refuse every consumer
	// var and say where to run from. RDR_HOME alone survives: the engine
	// cwd vouches for it by construction, and the fact table and template
	// sidecar an explicit `--records` read still live there.
	if home := out["RDR_HOME"]; home != "" {
		if resolved, err := filepath.EvalSymlinks(home); err == nil {
			home = resolved
		}
		if filepath.Clean(home) == project {
			seamRefusal = "stopped:engine-cwd marker=" + marker + " records=" + out["RDR_RECORDS"] +
				" -- rdr ran from the engine checkout (" + project + "), which is every consumer's and binds only the marker's; run it from the consumer repo, or pass --records"
			return map[string]string{"RDR_HOME": out["RDR_HOME"]}
		}
	}

	// A repo-local marker anchors `<CONSUMER>_ROOT="$PROJECT"` before this
	// fix, so from a worktree it would bind the MAIN checkout's records —
	// the exact false pass this fix removes. Once toplevel and project can
	// differ, the only way to tell a stale marker from a bound one is to
	// check where the bound records actually landed: inside toplevel, or
	// only inside project. The worktree may sit INSIDE the main checkout
	// (`main/.claude/worktrees/x`), so the test is "under project AND not
	// under toplevel", not a simple inequality.
	if marker == filepath.Join(project, ".rdr", "workspace") && toplevel != project {
		if records := out["RDR_RECORDS"]; records != "" &&
			strings.HasPrefix(records, project+string(filepath.Separator)) &&
			!strings.HasPrefix(records, toplevel+string(filepath.Separator)) {
			seamRefusal = "stopped:marker-binds-main-checkout marker=" + marker +
				" records=" + records + " toplevel=" + toplevel +
				" -- a repo-local marker must anchor its records on $TOPLEVEL; re-run /rdr-init --reconfigure"
			return map[string]string{}
		}
	}
	return out
}

// findMarker applies the flow's nearest-marker-wins rule: a repo-local
// `$PROJECT/.rdr/workspace` beats the shared `$WS/.rdr-workspace`, the
// way the closest .git or .editorconfig governs.
//
// `toplevel` is the directory holding whatever stopped the upward walk —
// a `.git` directory or a `.git` file, exactly what
// `git rev-parse --show-toplevel` would answer, without paying a process
// spawn on every invocation. In a worktree `.git` is a FILE pointing at
// the main repo, and `toplevel` is the worktree itself, never the main
// checkout.
//
// `project` is where the git COMMON dir lives — the main checkout — so
// the marker lookup still finds a repo-local marker that was written
// once, in the main checkout, and never duplicated per worktree. Outside
// a worktree `project == toplevel`. Inside one, the `.git` file is
// followed: its `gitdir:` line names the worktree's private git dir, and
// that dir's `commondir` file names the shared one, whose parent is
// `project`. Either read failing falls back to `project = toplevel` —
// today's behavior — so a `.git` file this binary cannot parse degrades
// to "marker not found" rather than a wrong bind.
func findMarker() (marker, project, ws, toplevel string) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", ""
	}
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}
	toplevel = ""
	for dir := cwd; ; {
		if pathExists(filepath.Join(dir, ".git")) {
			toplevel = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if toplevel == "" {
		return "", "", "", ""
	}
	project = gitCommonProject(toplevel)
	if project == "" {
		project = toplevel
	}
	ws = filepath.Dir(project)

	if local := filepath.Join(project, ".rdr", "workspace"); fileExists(local) {
		return local, project, ws, toplevel
	}
	if shared := filepath.Join(ws, ".rdr-workspace"); fileExists(shared) {
		return shared, project, ws, toplevel
	}
	return "", project, ws, toplevel
}

// gitCommonProject answers the directory holding the git COMMON dir, or
// "" when `.git` is a plain directory (no following needed) or the
// pointer chain cannot be read. Pure file reads, no `git` process: a
// worktree's `.git` is `gitdir: <path>`, and `<path>/commondir` names the
// shared git dir, usually as `../..` relative to `<path>`.
func gitCommonProject(toplevel string) string {
	dotGit := filepath.Join(toplevel, ".git")
	fi, err := os.Stat(dotGit)
	if err != nil || fi.IsDir() {
		return ""
	}
	raw, err := os.ReadFile(dotGit)
	if err != nil {
		return ""
	}
	const prefix = "gitdir:"
	line := strings.TrimSpace(string(raw))
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	gitdir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(toplevel, gitdir)
	}
	commondirFile := filepath.Join(gitdir, "commondir")
	raw, err = os.ReadFile(commondirFile)
	if err != nil {
		return ""
	}
	commondir := strings.TrimSpace(string(raw))
	if commondir == "" {
		return ""
	}
	if !filepath.IsAbs(commondir) {
		commondir = filepath.Join(gitdir, commondir)
	}
	commondir = filepath.Clean(commondir)
	if resolved, err := filepath.EvalSymlinks(commondir); err == nil {
		commondir = resolved
	}
	return filepath.Dir(commondir)
}

// seamValue returns a marker value, or "" when no marker bound it. The
// environment is not consulted here: callers check it first, because an
// inherited value is a decision someone made and the marker is only the
// fallback.
func seamValue(name string) string { return seam()[name] }

// pathExists reports whether anything is at p — a directory or, for a
// worktree's `.git`, a file.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// flagBound holds the roots a resolved flag has bound for this process —
// `--records`/`--repo`, once main.go knows their final absolute value.
// `declareFlags` reads `envOrSeam` for a flag's DEFAULT before the flag
// itself is parsed, so the flag cannot outrank env-or-marker from inside
// its own default; this map is how it does so anyway, for every OTHER
// reader of the same var (NewFactEnv's roots, resolveRecordsDir's
// $RDR_RECORDS fallback) that would otherwise keep reading the old tree
// after `--records` named a new one.
var (
	flagBoundMu sync.Mutex
	flagBound   = map[string]string{}
)

// bindFlag records a resolved flag value under the var name it stands
// in for, so envOrSeam's later callers see it first.
func bindFlag(name, value string) {
	flagBoundMu.Lock()
	defer flagBoundMu.Unlock()
	flagBound[name] = value
}

// envOrSeam is the binding order every flag default uses: a resolved
// flag outranks an explicit environment variable, which outranks the
// marker — a caller who spells out a path means that path, over anything
// ambient.
func envOrSeam(name string) string {
	flagBoundMu.Lock()
	v, ok := flagBound[name]
	flagBoundMu.Unlock()
	if ok && v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return seamValue(name)
}
