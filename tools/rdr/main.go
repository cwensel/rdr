// Command rdr is the deterministic, read-only reader for RDR markdown records.
//
// It never writes to a record. Markdown remains the source of truth; this
// binary only projects it.
//
// Usage:
//
//	rdr inspect <NNNN|path> [--json] [--select outline|elements|warnings|<element-id>] [--project P] [--records DIR]
//	rdr index [--derived] [--status] [--in-flight] [--backlinks] ... [--records DIR]
//	rdr lint <NNNN|path>
//	rdr version
//
// Exit codes:
//
//	0  success
//	2  unparseable input, an unresolvable selector, or a subcommand whose
//	   scanner has not landed yet
//
// Findings never change inspect's exit code; lint owns PASS/BLOCK exits.
//
// inspect and index --derived are live. The remaining index facets and
// lint report `stopped:not-implemented` and exit 2, which is the
// contract's "degrade to a clear stopped:<reason> rather than a stack
// trace" requirement.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// version is the engine revision this binary was built from. rdr-doctor
// compares it against the engine commit. It is overridden at build time
// with -ldflags "-X main.version=<sha>"; the default marks a build that
// did not go through the install path.
var version = "dev"

// schemaVersion is the JSON envelope contract consumers pin. It moves only
// when the envelope's shape changes, independently of the engine revision.
const schemaVersion = scan.SchemaVersion

const usage = `rdr — read-only projector for RDR markdown records

usage:
  rdr inspect <NNNN|path> [--json] [--select outline|elements|warnings|<id>] [--project P] [--records DIR]
  rdr index [--derived] [--status] [--in-flight] [--backlinks] [--records DIR]
  rdr lint <NNNN|path>
  rdr version

element ids (README §identifiers):
  NNNN:A3 assumption · NNNN:C4 contract · NNNN:D-identity decision · NNNN:RT1
  invariant · NNNN:ALT2 alternative · NNNN:BR3 briefly rejected · NNNN:S5
  scenario · NNNN:MVV · NNNN:F2 failure mode · NNNN:G-scope gate response ·
  NNNN:§approach section · cli/NNNN:C4 across records dirs

exit codes:
  0  success
  2  unparseable input, unresolvable selector, or unimplemented subcommand
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "rdr %s (schema %s)\n", version, schemaVersion)
		return 0

	case "inspect", "index", "lint":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(stderr)
		f := declareFlags(args[0], fs)
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		switch args[0] {
		case "inspect":
			return inspect(fs.Args(), f, stdout, stderr)
		case "index":
			if *f.derived {
				return indexDerived(f, stdout, stderr)
			}
		}
		fmt.Fprintf(stderr, "stopped:not-implemented (%s: the record scanner has not landed)\n", args[0])
		return 2

	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0

	default:
		fmt.Fprintf(stderr, "stopped:unknown-subcommand (%s)\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}

// flags holds the values of every declared flag; a subcommand reads only
// its own.
type flags struct {
	json, all, derived          *bool
	sel, project, records       *string
	status, inFlight, backlinks *bool
	clusterOf                   *string
	unresolved                  *bool
}

// declareFlags registers each subcommand's flags. They are declared here —
// not where a facet reads them — so that `rdr <cmd> --help` states the
// invocation contract before every facet exists, and so an unknown flag
// fails loudly today rather than silently once it does.
func declareFlags(cmd string, fs *flag.FlagSet) *flags {
	f := &flags{}
	records := fs.String("records", os.Getenv("RDR_RECORDS"), "records dir for NNNN lookup (default $RDR_RECORDS, else .)")
	project := fs.String("project", "", "project prefix for ids (cli/NNNN:C4); omitted inside one records dir")
	f.records, f.project = records, project
	switch cmd {
	case "inspect":
		f.json = fs.Bool("json", false, "emit the JSON envelope")
		f.sel = fs.String("select", "", "project one facet: outline|elements|warnings|<element-id>")
		f.all = fs.Bool("all", false, "include facets omitted by default")
	case "index":
		f.json = fs.Bool("json", false, "emit the index as JSON")
		f.derived = fs.Bool("derived", false, "count derived (unlabelled) element ids per record — the labelling backlog")
		f.status = fs.Bool("status", false, "group records by status")
		f.inFlight = fs.Bool("in-flight", false, "records not in a terminal status")
		f.backlinks = fs.Bool("backlinks", false, "inbound predecessor/override edges")
		f.clusterOf = fs.String("cluster-of", "", "sibling records declaring the same cluster")
		f.unresolved = fs.Bool("unresolved", false, "edges with no resolvable target")
	case "lint":
		f.json = fs.Bool("json", false, "emit findings as JSON")
	}
	return f
}

// resolve turns a NNNN or a path into a record path. A bare number is
// looked up as NNNN-*.md in the records dir.
func resolve(arg, records string) (string, error) {
	if ident.RecordOf(arg) == arg {
		dir := records
		if dir == "" {
			dir = "."
		}
		matches, _ := filepath.Glob(filepath.Join(dir, arg+"-*.md"))
		if len(matches) == 0 {
			matches, _ = filepath.Glob(filepath.Join(dir, arg+".md"))
		}
		matches = recordFiles(matches)
		switch len(matches) {
		case 0:
			return "", fmt.Errorf("stopped:no-such-record (%s in %s)", arg, dir)
		case 1:
			return matches[0], nil
		default:
			sort.Strings(matches)
			return "", fmt.Errorf("stopped:ambiguous-record (%s: %s)", arg, strings.Join(matches, ", "))
		}
	}
	return arg, nil
}

// records drops the files beside a record that share its number but are
// not the record: the `-postmortem.md` sibling the flow writes next to a
// closed RDR.
func recordFiles(paths []string) []string {
	var out []string
	for _, p := range paths {
		if strings.HasSuffix(p, "-postmortem.md") {
			continue
		}
		out = append(out, p)
	}
	return out
}

func inspect(args []string, f *flags, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "stopped:usage (inspect takes exactly one NNNN or path)")
		return 2
	}
	path, err := resolve(args[0], *f.records)
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

	var out any = doc
	switch sel := *f.sel; sel {
	case "":
	case "outline":
		out = doc.Outline
	case "elements":
		out = doc.Elements
	case "warnings":
		out = doc.Warnings
	default:
		start, end, ok := doc.Select(sel)
		if !ok {
			fmt.Fprintf(stderr, "stopped:no-such-element (%s in %s)\n", sel, doc.Record)
			return 2
		}
		if *f.json {
			out = map[string]any{"id": sel, "line_start": start, "line_end": end,
				"lines": doc.Slice(start, end)}
			break
		}
		for _, l := range doc.Slice(start, end) {
			fmt.Fprintln(stdout, l)
		}
		return 0
	}
	if *f.json || *f.sel != "" {
		return emit(out, stdout, stderr)
	}
	return summary(doc, stdout)
}

func emit(v any, stdout, stderr io.Writer) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(stderr, "stopped:encode (%v)\n", err)
		return 2
	}
	return 0
}

// summary is the human form: one line per element, ids first, so the
// output is greppable and pipes into --select.
func summary(doc *scan.Document, w io.Writer) int {
	fmt.Fprintf(w, "%s  %s  epoch %s  %d lines\n", doc.Record, doc.Title, doc.Epoch, doc.Lines)
	for _, e := range doc.Elements {
		mark := " "
		if e.Derived {
			mark = "~"
		}
		fmt.Fprintf(w, "%s %-24s %5d-%-5d %s\n", mark, e.ID, e.LineStart, e.LineEnd, e.Label)
	}
	for _, wn := range doc.Warnings {
		fmt.Fprintf(w, "! %-24s %5d-%-5d %s\n", wn.Code, wn.LineStart, wn.LineEnd, wn.Message)
	}
	fmt.Fprintf(w, "derived: %s\n", derivedLine(doc.Counts))
	return 0
}

func derivedLine(c scan.Counts) string {
	var parts []string
	for _, k := range ident.Kinds {
		if c.Elements[k] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d/%d", k, c.Derived[k], c.Elements[k]))
		}
	}
	return strings.Join(parts, "  ")
}

// derivedRow is one record's labelling backlog.
type derivedRow struct {
	Record   string             `json:"record"`
	Path     string             `json:"path"`
	Epoch    string             `json:"epoch"`
	Elements map[ident.Kind]int `json:"elements"`
	Derived  map[ident.Kind]int `json:"derived"`
	Warnings int                `json:"warnings"`
}

// indexDerived walks the records dir and reports, per record and in
// total, how many elements carry derived ids — the backlog the labelling
// rule burns down.
func indexDerived(f *flags, stdout, stderr io.Writer) int {
	dir := *f.records
	if dir == "" {
		dir = "."
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9][0-9]-*.md"))
	paths = recordFiles(paths)
	sort.Strings(paths)
	if len(paths) == 0 {
		fmt.Fprintf(stderr, "stopped:no-records (%s holds no NNNN-*.md)\n", dir)
		return 2
	}
	total := scan.Counts{Elements: map[ident.Kind]int{}, Derived: map[ident.Kind]int{}}
	var rows []derivedRow
	var skipped []string
	for _, p := range paths {
		doc, err := scan.File(p, scan.Options{Project: *f.project})
		if err != nil {
			fmt.Fprintf(stderr, "stopped:unreadable (%s: %v)\n", p, err)
			return 2
		}
		if doc.Epoch == "unknown" {
			// Not an RDR: no metadata block, no assumptions. Listed, never
			// silently dropped.
			skipped = append(skipped, p)
			continue
		}
		rows = append(rows, derivedRow{doc.Record, p, doc.Epoch, doc.Counts.Elements, doc.Counts.Derived, len(doc.Warnings)})
		for k, n := range doc.Counts.Elements {
			total.Elements[k] += n
		}
		for k, n := range doc.Counts.Derived {
			total.Derived[k] += n
		}
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "records": rows, "total": total, "skipped": skipped}, stdout, stderr)
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%s %s %3dw  %s\n", r.Record, r.Epoch, r.Warnings, derivedLine(scan.Counts{Elements: r.Elements, Derived: r.Derived}))
	}
	fmt.Fprintf(stdout, "total %d records  %s\n", len(rows), derivedLine(total))
	for _, p := range skipped {
		fmt.Fprintf(stdout, "skipped %s (not an RDR: no epoch fingerprint)\n", p)
	}
	return 0
}
