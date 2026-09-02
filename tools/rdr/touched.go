package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// hunk is one post-image range of a unified diff: the `+start,len` pair
// of an `@@` header. Len 0 is a pure deletion at the point between
// Start and Start+1.
type hunk struct {
	Start int `json:"start"`
	Len   int `json:"len"`
}

// hunkHeader is the `@@ -a,b +c,d @@` line; a blank length is 1.
var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// parseHunks reads every `@@` header of a unified diff. Any header the
// pattern does not match is an error, never a skipped hunk — a set that
// silently dropped one would scope a re-entry short.
func parseHunks(diff string) ([]hunk, error) {
	hunks := []hunk{}
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "@@") {
			continue
		}
		m := hunkHeader.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("malformed hunk header %q", line)
		}
		start, _ := strconv.Atoi(m[1])
		n := 1
		if m[2] != "" {
			n, _ = strconv.Atoi(m[2])
		}
		hunks = append(hunks, hunk{Start: start, Len: n})
	}
	return hunks, nil
}

// span is the closed line range a hunk touches in the working tree. A
// pure deletion (`+c,0`) sits between lines c and c+1, so it touches
// whichever element ends at c or begins at c+1.
func (h hunk) span() (lo, hi int) {
	if h.Len == 0 {
		return h.Start, h.Start + 1
	}
	return h.Start, h.Start + h.Len - 1
}

func overlaps(hunks []hunk, start, end int) bool {
	for _, h := range hunks {
		lo, hi := h.span()
		if start <= hi && end >= lo {
			return true
		}
	}
	return false
}

// touchedRow is one element or section a hunk overlaps.
type touchedRow struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// touched answers which of the record's ids a hunk set overlaps: every
// element (a nested clause and its parent contract both, since both
// ranges hold the line), then every section, each in line order. An
// overlap is inclusive at both ends.
func touched(doc *scan.Document, hunks []hunk) []touchedRow {
	rows := []touchedRow{}
	for _, e := range doc.Elements {
		if overlaps(hunks, e.LineStart, e.LineEnd) {
			rows = append(rows, touchedRow{ID: e.ID, Kind: string(e.Kind), LineStart: e.LineStart, LineEnd: e.LineEnd})
		}
	}
	for _, n := range doc.Outline {
		if overlaps(hunks, n.LineStart, n.LineEnd) {
			rows = append(rows, touchedRow{ID: n.ID, Kind: "§", LineStart: n.LineStart, LineEnd: n.LineEnd})
		}
	}
	return rows
}

// touchedIDs is the id list alone, the value a caller scopes by.
func touchedIDs(doc *scan.Document, hunks []hunk) []string {
	ids := []string{}
	for _, r := range touched(doc, hunks) {
		ids = append(ids, r.ID)
	}
	return ids
}

// gitHunks is the diff from rev to the working tree for one file, as
// post-image hunks. The binary issues -U0 itself so the headers are the
// changed lines and nothing around them. A renamed record diffs as a
// whole-file add, so every element reads as touched — the honest answer
// when the path has no history at rev.
func gitHunks(path, rev string) ([]hunk, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "-C", filepath.Dir(abs), "diff", "-U0", "--no-color", rev, "--", filepath.Base(abs))
	var errb strings.Builder
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		why := strings.TrimSpace(errb.String())
		if why == "" {
			why = err.Error()
		}
		return nil, fmt.Errorf("%s", why)
	}
	return parseHunks(string(out))
}

// touchedSince answers `inspect --touched-since REV`: the ids whose
// lines changed between REV and the working tree, with the hunks as
// evidence, so an empty answer is `[]` beside the diff that produced
// it. A git failure is a stop, never an empty set — a bad rev that read
// as "nothing touched" would pass a re-entry with no scope.
func touchedSince(doc *scan.Document, rev string, jsonOut bool) (projection, error) {
	hunks, err := gitHunks(doc.Path, rev)
	if err != nil {
		return projection{}, fmt.Errorf("stopped:no-diff (%s)", strings.ReplaceAll(err.Error(), "\n", " "))
	}
	rows := touched(doc, hunks)
	r := projection{doc: doc}
	if jsonOut {
		r.value = map[string]any{"schema": schemaVersion, "record": doc.Record, "path": doc.Path,
			"rev": rev, "touched": rows, "hunks": hunks}
		return r, nil
	}
	if len(rows) == 0 {
		r.lines = []string{fmt.Sprintf("no-change  %s since %s (%d hunks)", doc.Record, rev, len(hunks))}
		return r, nil
	}
	for _, t := range rows {
		mark := " "
		if t.Kind == "§" {
			mark = "§"
		}
		r.lines = append(r.lines, fmt.Sprintf("%s %-24s %5d-%d", mark, t.ID, t.LineStart, t.LineEnd))
	}
	return r, nil
}
