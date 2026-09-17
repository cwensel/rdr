package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// receipt answers one question from the binary's own usage log: was this
// record linted since it was last written? It exists because the log's
// first real audit showed four stages closing gates on one record with
// no lint call at all, and nothing noticed. The skills say "lint at stage
// exit"; this is the check that says whether they did.
//
// It reads the log rather than the record because the record cannot vouch
// for itself — a gate line is prose the same model wrote. The log line was
// written by lint, on the way out, whatever the exit; that is the receipt.
//
//	0  a lint of this record (or of the whole dir) ran at or after the
//	   record's mtime — or after --since — and its log line is printed
//	1  no such lint: stopped:no-lint-receipt, with what to run
//	2  no usage log is bound, so nothing can vouch either way
//
// A lint that exited 1 still counts: it ran, and its findings are the
// stage's problem, not this check's. A whole-dir lint (no target) covers
// every record. The caller decides what 2 means — §commit proceeds with a
// note, because a project that never opted into the log did not opt into
// this either.
func receipt(args []string, f *flags, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "stopped:usage (receipt takes exactly one NNNN or path)")
		return 2
	}
	path, err := resolve(args[0], *f.records)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	num := recordNumberOfPath(path)
	if num == "" {
		fmt.Fprintf(stderr, "stopped:no-record-number (%s)\n", filepath.Base(path))
		return 2
	}
	logPath := usageLogPath()
	if logPath == "" {
		fmt.Fprintln(stderr, "stopped:no-usage-log (RDR_USAGE_LOG is off; nothing can vouch for a lint)")
		return 2
	}

	var since time.Time
	if f.since != nil && *f.since != "" {
		since, err = time.Parse(time.RFC3339, *f.since)
		if err != nil {
			fmt.Fprintf(stderr, "stopped:bad-since (%s is not RFC3339)\n", *f.since)
			return 2
		}
	} else {
		st, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(stderr, "stopped:unreadable (%v)\n", err)
			return 2
		}
		since = st.ModTime()
	}
	// Log stamps are whole seconds; a write and its lint inside the same
	// second must not read as "lint came first".
	since = since.Truncate(time.Second)

	last, latest, err := lastLint(logPath, num, since)
	if err != nil {
		fmt.Fprintf(stderr, "stopped:unreadable (%v)\n", err)
		return 2
	}
	if last != "" {
		fmt.Fprintln(stdout, last)
		return 0
	}
	was := "never"
	if !latest.IsZero() {
		was = latest.Format(time.RFC3339)
	}
	fmt.Fprintf(stderr, "stopped:no-lint-receipt (%s written %s; last lint %s) — run: recs lint %s\n",
		num, since.Format(time.RFC3339), was, num)
	return 1
}

// lastLint scans the log for the newest lint line covering num at or
// after since. It also returns the newest lint of num at any time, so the
// refusal can say "last lint 06:58" rather than just "no".
func lastLint(logPath, num string, since time.Time) (line string, latest time.Time, err error) {
	fh, err := os.Open(logPath)
	if os.IsNotExist(err) {
		return "", latest, nil // a log that was never written has no lint in it
	}
	if err != nil {
		return "", latest, err
	}
	defer fh.Close()
	var best time.Time
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 0, 64<<10), lineHardMaxBytes)
	for sc.Scan() {
		raw := sc.Bytes()
		var u struct {
			Cmd    string `json:"cmd"`
			Target string `json:"target"`
			TS     string `json:"ts"`
		}
		if json.Unmarshal(raw, &u) != nil || u.Cmd != "lint" || !lintCovers(u.Target, num) {
			continue
		}
		ts, e := time.Parse(time.RFC3339, u.TS)
		if e != nil {
			continue
		}
		if ts.After(latest) {
			latest = ts
		}
		if !ts.Before(since) && !ts.Before(best) {
			best, line = ts, string(raw)
		}
	}
	return line, latest, sc.Err()
}

var recordNumberPrefix = regexp.MustCompile(`^(\d{4})-`)

// recordNumberOfPath reads NNNN off a record's filename.
func recordNumberOfPath(path string) string {
	m := recordNumberPrefix.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return ""
	}
	return m[1]
}

// lintCovers decides whether a logged lint target names num: the whole
// dir (no target), the number in any of the spellings resolve accepts
// (3, 03, 0003), or a path/slug whose basename starts with NNNN-.
func lintCovers(target, num string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return true
	}
	if isDigits(target) {
		return strings.TrimLeft(target, "0") == strings.TrimLeft(num, "0")
	}
	return recordNumberOfPath(target) == num
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
