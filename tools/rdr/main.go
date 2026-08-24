// Command rdr is the deterministic, read-only reader for RDR markdown records.
//
// It never writes to a record. Markdown remains the source of truth; this
// binary only projects it.
//
// Usage:
//
//	rdr inspect <NNNN|path> [--json] [--select outline|elements|edges]
//	rdr index [--status] [--in-flight] [--backlinks] ...
//	rdr lint <NNNN|path>
//	rdr version
//
// Exit codes:
//
//	0  success
//	2  unparseable input, or a subcommand whose scanner has not landed yet
//
// Findings never change inspect's exit code; lint owns PASS/BLOCK exits.
//
// This commit ships the plumbing and the template model only. The line
// scanner that feeds inspect/index/lint is a separate change; until it
// lands those subcommands report `stopped:not-implemented` and exit 2,
// which is the contract's "degrade to a clear stopped:<reason> rather than
// a stack trace" requirement.
package main

import (
	"flag"
	"fmt"
	"os"
)

// version is the engine revision this binary was built from. rdr-doctor
// compares it against the engine commit. It is overridden at build time
// with -ldflags "-X main.version=<sha>"; the default marks a build that
// did not go through the install path.
var version = "dev"

// schemaVersion is the JSON envelope contract consumers pin. It moves only
// when the envelope's shape changes, independently of the engine revision.
const schemaVersion = "0"

const usage = `rdr — read-only projector for RDR markdown records

usage:
  rdr inspect <NNNN|path> [--json] [--select outline|elements|edges]
  rdr index [--status] [--in-flight] [--backlinks]
  rdr lint <NNNN|path>
  rdr version

exit codes:
  0  success
  2  unparseable input or unimplemented subcommand
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "rdr %s (schema %s)\n", version, schemaVersion)
		return 0

	case "inspect", "index", "lint":
		// Parse the subcommand's flags so an unknown flag is reported as a
		// flag error rather than swallowed by the not-implemented stop.
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(stderr)
		declareFlags(args[0], fs)
		if err := fs.Parse(args[1:]); err != nil {
			return 2
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

// declareFlags registers each subcommand's flags. The flags are declared
// here — not where the scanner will read them — so that `rdr <cmd> --help`
// states the invocation contract before the scanner exists, and so an
// unknown flag fails loudly today rather than silently once it does.
func declareFlags(cmd string, fs *flag.FlagSet) {
	switch cmd {
	case "inspect":
		fs.Bool("json", false, "emit the JSON envelope")
		fs.String("select", "", "project one facet: outline|elements|edges|<element-id>")
		fs.Bool("all", false, "include facets omitted by default")
	case "index":
		fs.Bool("status", false, "group records by status")
		fs.Bool("in-flight", false, "records not in a terminal status")
		fs.Bool("backlinks", false, "inbound predecessor/override edges")
		fs.String("cluster-of", "", "sibling records declaring the same cluster")
		fs.Bool("unresolved", false, "edges with no resolvable target")
	case "lint":
		fs.Bool("json", false, "emit findings as JSON")
	}
}
