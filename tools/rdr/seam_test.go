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
	project, _ := newProject(t, "local", body)
	t.Chdir(project)
	t.Setenv("RDR_RECORDS", "")
	t.Setenv("RDR_SOURCE_REPO", "")

	code, out, errb := runCapture(t, "index", "--in-flight")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(out, "0007-seam") {
		t.Errorf("the seam did not bind the records dir:\n%s", out)
	}
}
