package main

// The seam binds itself.
//
// Every var this binary reads — $RDR_RECORDS, $RDR_SOURCE_REPO — is
// written in a marker file the flow already maintains. Until now only a
// shell could read it, so every call site had to carry the seam through
// the turn: either a fifteen-line resolver block re-run verbatim, or an
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

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// seamVars are the marker values this binary can use. The marker exports
// more — evidence roots, path maps, per-consumer conveniences — and none
// of it is this tool's business.
var seamVars = []string{"RDR_RECORDS", "RDR_SOURCE_REPO"}

// seam resolves once per working directory. A projection may consult it
// several times and the marker cannot change mid-run, so the common case
// binds once — but the cwd IS the lookup key, so a process that moves
// (a test binary, or any long-lived caller) must not keep answering from
// the directory it started in.
var (
	seamMu    sync.Mutex
	seamDir   string
	seamCache map[string]string
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
	seamDir, seamCache = cwd, bindSeam()
	return seamCache
}

// bindSeam finds the marker and returns the vars it exports. An empty
// map means no marker was found, which is not an error: the flow works
// without one, and callers already handle unbound vars.
func bindSeam() map[string]string {
	out := map[string]string{}
	marker, project, ws := findMarker()
	if marker == "" {
		return out
	}
	// The marker guards itself with `: "${WS:?…}"` / `"${PROJECT:?…}"` so
	// that sourcing it directly, without the resolver that derives them
	// from git topology, fails loudly rather than binding half a seam.
	// Supplying both is what makes this the canonical read rather than
	// the direct source the marker refuses.
	script := `set -e
. "$1" >/dev/null 2>&1 || exit 1
for v in ` + strings.Join(seamVars, " ") + `; do
  eval "val=\$$v"
  [ -n "$val" ] && printf '%s=%s\n' "$v" "$val"
done
exit 0`
	cmd := exec.Command("sh", "-c", script, "_", marker)
	cmd.Env = append(os.Environ(), "PROJECT="+project, "WS="+ws)
	cmd.Dir = project
	stdout, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(stdout), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// findMarker applies the flow's nearest-marker-wins rule: a repo-local
// `$PROJECT/.rdr/workspace` beats the shared `$WS/.rdr-workspace`, the
// way the closest .git or .editorconfig governs.
//
// `$PROJECT` is the directory holding the git common dir, so a worktree
// resolves its main repo's marker rather than missing it — the same
// worktree-invariance the shell resolver is careful about.
func findMarker() (marker, project, ws string) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", ""
	}
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}
	// Walk up for `.git`, which is the project root by the same rule
	// `git rev-parse --git-common-dir` applies — without paying a process
	// spawn on every invocation. In a worktree `.git` is a FILE pointing
	// at the main repo; the directory holding it is still this project's
	// root, and the marker lookup wants that, so the pointer needs no
	// following.
	project = ""
	for dir := cwd; ; {
		if pathExists(filepath.Join(dir, ".git")) {
			project = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if project == "" {
		return "", "", ""
	}
	ws = filepath.Dir(project)

	if local := filepath.Join(project, ".rdr", "workspace"); fileExists(local) {
		return local, project, ws
	}
	if shared := filepath.Join(ws, ".rdr-workspace"); fileExists(shared) {
		return shared, project, ws
	}
	return "", project, ws
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

// envOrSeam is the binding order every flag default uses: an explicit
// environment variable wins, then the marker. The flag itself outranks
// both, since a caller who spells out a path means that path.
func envOrSeam(name string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return seamValue(name)
}
