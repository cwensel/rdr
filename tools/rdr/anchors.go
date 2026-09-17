package main

// `recs anchors` reads the element ids a findings file cites, so two
// ledgers are diffed by `comm` rather than re-read side by side.
//
// A lens re-run had to be reconciled against its origin ledger — which
// findings are still open, which are net-new — and the prompts said how
// in prose: "reconcile by passage anchor", "set intersection on those
// ids". Every persona, critique and cove finding already anchors to the
// id the projector mints (`NNNN:C4`, `NNNN:A3`, `NNNN:§approach`), so the
// diff is a set operation over tokens, and a model re-walking two files to
// perform it is the re-derivation the doctrine bans. This prints the set;
// `comm -12` / `comm -13` over two of them is the reconciliation.
//
//	0  the ids, sorted, unique, one per line (possibly none)
//	2  bad usage, an unresolvable record, or an unreadable file
//
// MEMBERSHIP IS EXACT. A token counts only when it is an id the record's
// projection mints — the same outline, element and anchor ids `inspect`
// lists — so a persona citing a peer record, or a section that was
// reworded away, contributes nothing to the ledger diff; `--unresolved`
// prints those instead, so the author sees what a ledger row points at
// that no longer exists. A project prefix is ignored on both sides: the
// corpus writes `NNNN:C4` and `proj/NNNN:C4` for the same element.

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// idToken is the widest span an id can occupy in prose: the grammar's
// own shapes, read leniently and then handed to ident.Parse for the
// exact test. Trailing punctuation a sentence adds is trimmed before
// the parse, so `(0021:C4).` and `| 0021:C4 |` both read.
var idToken = regexp.MustCompile(`(?:[A-Za-z0-9][A-Za-z0-9_.-]*/)?\d{4}:(?:§[a-z0-9-]+|[A-Z]{1,3}-?[A-Za-z0-9-]*)`)

func anchorsCmd(args []string, f *flags, stdout, stderr io.Writer) int {
	if deref(f.record) == "" || len(args) == 0 {
		fmt.Fprintln(stderr, "stopped:usage (anchors takes --record NNNN and one or more files)")
		return 2
	}
	// The record is read the way `status` reads it: resolved, scanned,
	// and never edge-resolved — the ids it mints need no corpus.
	path, err := resolve(deref(f.record), *f.records)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	doc, err := scan.File(path, scan.Options{Project: *f.project})
	if err != nil {
		fmt.Fprintf(stderr, "stopped:unreadable (%v)\n", err)
		return 2
	}
	if doc.Record == "" {
		fmt.Fprintln(stderr, "stopped:no-record-number (neither the title nor the filename carries NNNN)")
		return 2
	}
	minted := map[string]string{} // bare form -> the projection's spelling
	add := func(id string) {
		if k := bareID(id); k != "" {
			minted[k] = id
		}
	}
	for _, n := range doc.Outline {
		add(n.ID)
	}
	for _, el := range doc.Elements {
		add(el.ID)
	}
	for _, a := range doc.Anchors {
		add(a.ID)
	}

	unresolved := f.unresolved != nil && *f.unresolved
	seen := map[string]bool{}
	for _, path := range args {
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "stopped:unreadable (%v)\n", err)
			return 2
		}
		for _, tok := range idToken.FindAllString(string(raw), -1) {
			tok = strings.TrimRight(tok, ".,;:)]")
			id, err := ident.Parse(tok)
			if err != nil {
				continue
			}
			if id.Record != doc.Record {
				// A peer record's id is a citation, not an anchor into
				// this record; it is neither found nor unresolved here.
				continue
			}
			bare := bareID(tok)
			if canonical, ok := minted[bare]; ok {
				if !unresolved {
					seen[canonical] = true
				}
			} else if unresolved {
				seen[bare] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	for _, id := range out {
		fmt.Fprintln(stdout, id)
	}
	return 0
}

// bareID is an id without its project prefix, the form membership is
// compared in. "" for a string that is not an id.
func bareID(s string) string {
	id, err := ident.Parse(s)
	if err != nil {
		return ""
	}
	id.Project = ""
	return id.String()
}
