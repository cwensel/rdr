package main

// The usage log answers one question the flow could previously only
// estimate: did routing a structural read through this binary actually
// cost fewer tokens than having a model read the record?
//
// The consumer-integration pass had to answer it from byte counts —
// "estimated, measurement method chosen deliberately, not deferred" —
// because nothing recorded what the tool was asked for or how much it
// emitted. This records exactly that, per invocation, and nothing else.
// Real numbers accumulate over ordinary use; no dashboard, no session
// instrumentation, no second tool to run.
//
// # Format
//
// One JSON object per line, `\n`-terminated, UTF-8 — the sorted-key
// NDJSON the sibling codebase writes for its own record streams. Records
// are built as a map and marshalled, not encoded from a struct, because
// `json.Marshal` sorts map keys: the byte order is then stable across
// runs and across Go versions, which is what makes a log diffable.
//
// Keys are snake_case. The field set is APPEND-ONLY: a new key may join,
// an existing one never changes meaning and is never removed, and a
// consumer must tolerate keys it does not recognise. Optional fields are
// omitted when empty, so an older record's bytes stay identical as the
// schema grows.
//
// # The timestamp is a deliberate exception
//
// The house rule next door is that outputs carry no clock: the same
// input produces the same bytes forever. That rule is right for build
// artifacts and wrong here — a record of *when* work happened is
// worthless without a clock, and this log's whole purpose is to compare
// a before against an after. So `ts` exists, in RFC3339 with an explicit
// offset. It is confined to the log; the projection itself never reads a
// clock, and `stamp` is injectable so tests stay deterministic.
//
// # Never writes near a record
//
// This is the ONLY thing this binary ever writes, and it is opt-in:
// $RDR_USAGE_LOG unset — the default — writes nothing at all. A
// projector that silently created files beside a corpus would be a
// different tool.
//
// Opting in is a project decision, not a per-call one. `/rdr-init`
// writes the var into the marker, the binary binds it the way it binds
// every other seam var, and the log lands in `$PROJECT/.rdr/` — this
// flow's repo-local run-output directory, self-ignored by git. A caller
// that wants it somewhere else names a path instead.

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// usageEnvVar names the log. It is bound like every other seam var —
// explicit environment first, then the marker — so `/rdr-init` can turn
// logging on once for a project instead of every call setting it.
const usageEnvVar = "RDR_USAGE_LOG"

// usageDefaultName is the log's filename. Where it lands is decided by
// the marker that turned logging on — see usageLogPath.
const usageDefaultName = "usage.jsonl"

// usageLogPath decides where a line goes, or "" for nowhere.
//
// Three states, and the middle one is the point:
//
//	unset            off — the tool writes nothing at all, the default
//	a path           that file
//	"1"/"true"/"on"  on, beside the marker that said so
//
// The bare-truthy form is what `/rdr-init` writes into a marker: a
// project opts in once, and no call site has to know a path.
//
// **Beside the marker** is the whole rule, and it follows the seam's own
// scope rather than assuming one. A repo-local marker lives at
// `$PROJECT/.rdr/workspace`, so the log joins it in that already-
// self-ignoring directory. A workspace marker lives at
// `$WS/.rdr-workspace`, above repos that deliberately have no `.rdr/` —
// creating one there would invent seam structure the user opted out of,
// and scatter a shared setting into per-repo files nobody ignored. The
// log goes beside the shared marker instead, one file for the workspace
// that turned it on, in a directory that is not a repo.
//
// If no marker can be found, logging stays off rather than inventing a
// location: the same discovery-never-invention rule the seam follows.
func usageLogPath() string {
	v := strings.TrimSpace(os.Getenv(usageEnvVar))
	if v == "" {
		v = seamValue(usageEnvVar)
	}
	switch strings.ToLower(v) {
	case "":
		return ""
	case "0", "false", "off", "no":
		return ""
	case "1", "true", "on", "yes":
		marker, _, _ := findMarker()
		if marker == "" {
			return ""
		}
		return filepath.Join(filepath.Dir(marker), usageDefaultName)
	}
	return v
}

// lineHardMaxBytes caps one record, mirroring the sibling serializer's
// ceiling. A line this long means a field is carrying something it
// should not; the line is dropped rather than written, because a log
// that can wedge a reader is worse than a log with a gap.
const lineHardMaxBytes = 1 << 20

// usageRecord is one invocation. The fields are the ones a before/after
// comparison actually needs, and deliberately no more: what was asked,
// what it cost to answer, and how big the answer was. No record content
// is ever logged — a projection of prose must not leak the prose into a
// log that outlives it.
type usageRecord struct {
	TS        string
	Cmd       string
	Facet     string
	Target    string
	BytesOut  int
	ElapsedMS int64
	Exit      int
}

// payload projects a record to the wire shape. It is the one place the
// JSON spelling of a field is decided, so the on-disk log cannot fork
// from what this type means. Empty optional fields are omitted.
func (u usageRecord) payload() map[string]any {
	m := map[string]any{
		"ts":         u.TS,
		"cmd":        u.Cmd,
		"bytes_out":  u.BytesOut,
		"elapsed_ms": u.ElapsedMS,
		"exit":       u.Exit,
	}
	if u.Facet != "" {
		m["facet"] = u.Facet
	}
	if u.Target != "" {
		m["target"] = u.Target
	}
	return m
}

// countingWriter tallies what a subcommand emitted without buffering it:
// the projection of a large record is measured, not held in memory.
type countingWriter struct {
	w io.Writer
	n int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += n
	return n, err
}

// logUsage appends one line. Every failure is silent by design: a log
// that cannot be written must never change what the tool prints, what it
// exits with, or whether an answer is delivered. Measurement is strictly
// subordinate to the projection.
//
// Append, not the atomic tempfile-rename the sibling repo uses for its
// manifests: those replace a whole document per run, while this one
// accumulates across runs, and a rename would drop every earlier line.
// A single write of a line well under the pipe-buffer size is atomic on
// POSIX, so the parallel pre-lock lenses interleave whole lines rather
// than corrupting each other's — which is why the size ceiling above is
// enforced before the write, not after.
func logUsage(rec usageRecord) {
	path := usageLogPath()
	if path == "" {
		return
	}
	line, err := json.Marshal(rec.payload())
	if err != nil || len(line)+1 > lineHardMaxBytes {
		return
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}

// usageFacet names which query was run, so the log distinguishes a whole
// index build from a one-record projection. It reads the parsed flags
// rather than re-scanning argv: the facet is whichever branch run() took.
func usageFacet(cmd string, f *flags) string {
	if f == nil {
		return ""
	}
	switch cmd {
	case "inspect":
		if f.sel != nil && *f.sel != "" {
			return "select:" + selectorClass(*f.sel)
		}
		if f.json != nil && *f.json {
			if f.filter != nil && *f.filter != "" {
				// The filter is the whole difference between the 150KB
				// envelope and an 800-byte answer; a log that cannot tell
				// them apart cannot audit what --filter saves.
				return "json:" + strings.ReplaceAll(*f.filter, " ", "")
			}
			return "json"
		}
		return "text"
	case "index":
		for _, c := range []struct {
			on   bool
			name string
		}{
			{f.derived != nil && *f.derived, "derived"},
			{f.coverage != nil && *f.coverage, "coverage"},
			{f.backlinks.set, "backlinks"},
			{f.unresolved != nil && *f.unresolved, "unresolved"},
			{f.clusterOf != nil && *f.clusterOf != "", "cluster-of"},
			{f.inFlight != nil && *f.inFlight, "in-flight"},
			{f.status != nil && *f.status, "status"},
			{f.anchors != nil && *f.anchors, "anchor-intersect"},
			{f.readme.set, "readme"},
		} {
			if c.on {
				return c.name
			}
		}
		return "graph"
	case "lint":
		// Strict wins over locking: it is the whole-corpus migration
		// pass, an order of magnitude more work than either ordinary
		// lint, and an audit that could not tell it from a gate check
		// would misread the cost of both.
		if f.strict != nil && *f.strict {
			return "strict"
		}
		if f.locking != nil && *f.locking {
			return "locking"
		}
		return "advisory"
	}
	return ""
}

// selectorClass reduces `--select` to its kind. The selector can name an
// element id, which is record-identifying; the log keeps the shape
// (`element`) and leaves the id to the target field, which is already a
// record number the caller passed on the command line.
func selectorClass(sel string) string {
	switch sel {
	case "outline", "elements", "edges", "warnings", "metadata", "fields":
		return sel
	}
	return "element"
}

// stamp is time.Now indirected so a test can pin it. The projection
// itself never reads a clock — only the log does.
var stamp = func() time.Time { return time.Now() }
