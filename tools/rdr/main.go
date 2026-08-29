// Command rdr is the deterministic, read-only reader for RDR markdown records.
//
// It never writes to a record. Markdown remains the source of truth; this
// binary only projects it.
//
// Usage:
//
//	rdr inspect <NNNN|slug|path> [--json] [--filter k1,k2] [--select outline|elements|warnings|<element-id>] [--project P] [--records DIR]
//	rdr index [--json] [--status|--backlinks[=ID]|--cluster-of N|--anchor-intersect|--literal-intersect|--unresolved|--derived|--coverage|--readme[=PATH]] [--records DIR]
//	rdr lint [<NNNN|path>] [--locking] [--json] [--records DIR]
//	rdr status [<NNNN|slug|path>…] [--json|--tags] [--filter f1,f2] [--facts PATH] [--records DIR]
//	rdr env [--json]
//	rdr version
//
// Exit codes:
//
//	0  success, findings or not
//	1  lint only: a finding blocks a lock
//	2  unparseable input, an unresolvable selector, or a subcommand whose
//	   scanner has not landed yet
//
// Findings never change inspect's exit code; lint owns PASS/BLOCK exits.
//
// Every subcommand and facet is live; an unknown subcommand degrades to a
// clear stopped:<reason> rather than a stack trace.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/lint"
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
  rdr inspect <NNNN|slug|path> [--json] [--filter k1,k2] [--select <facet>|<id>] [--all] [--project P] [--records DIR] [--repo DIR]
  rdr index [--json] [<facet>] [--filter k1,k2] [--records DIR] [--repo DIR]
  rdr lint [<NNNN|path>] [--locking] [--json] [--records DIR]
  rdr receipt <NNNN|path> [--since RFC3339] [--records DIR]
  rdr status [<NNNN|slug|path>…] [--json|--tags] [--filter f1,f2] [--facts PATH] [--records DIR]
  rdr paths <NNNN|slug|path> [--lens L|--cluster KEY|--tree N[=OP]] [--next-iter] [--json]
  rdr env [--json]
  rdr version

index with no facet is the corpus graph: every record, element and edge,
plus the derived backlinks (README §Queries over the graph). Facets:
  --status                    every record grouped by status
  --backlinks[=NNNN[:elem]]   who points at each target / at one target
  --cluster-of NNNN           7.1's membership rule as a query
  --anchor-intersect [--all]  in-flight pairs sharing code anchors, uncited first
  --literal-intersect [--all] in-flight pairs whose contracts share a literal, uncited first
  --open-joint [--all]        open joint decisions: Joint-check (home: OPEN) lines + joint-decision Status forms
  --cycles                    ownership cycles, Joint-check home cycles, homes ahead of a lock, OPEN checks on a Final
  --unresolved                typed edges with no target — record data errors
  --derived                   the unlabelled-element backlog per record
  --coverage                  the drift alarm: unclassified-line rate, unknowns
  --readme[=PATH]             the README index table checked against the records

--filter keeps only the named top-level keys — of inspect's envelope
(metadata,counts,…) or of the index graph (records,elements,edges,backlinks) —
identity keys always included — one call where --select would need several.
Only edges[] carries "resolved", and deciding it scans the records dir and
walks --repo: the whole envelope, --select edges, --filter …edges and lint
pay that (~1.5s on a large corpus); every other facet answers from the
record alone (~20ms).
inspect with no flag is the summary: one line per element, id and line range —
the cheap id list (~50 lines); --select elements is every element as JSON,
~25× larger. --select <id> then names a section or element to read.
A record is named by number (3, 03, 0003), slug, or path; --records defaults
to $RDR_RECORDS and a relative one resolves against it.

element ids (README §identifiers):
  NNNN:A3 assumption · NNNN:C4 contract · NNNN:D-identity decision · NNNN:RT1
  invariant · NNNN:ALT2 alternative · NNNN:BR3 briefly rejected · NNNN:S5
  scenario (list item or T-5 table row) · NNNN:MVV · NNNN:F2 failure mode ·
  NNNN:JC2 Joint-check line (joint.open says whether it is ruled) · NNNN:G-scope gate response ·
  NNNN:§approach section · cli/NNNN:C4 across records dirs

lint is the conformance authority, one pass over three severities (README
§lint): parse warnings on every record, conformance ADVICE on live records
only, and resolution findings — dangling edges, Peer-RDR Evidence naming no
element, unlabelled contracts on a post-rule record — that block a lock.
With no argument it lints the whole records dir.

status evaluates models/rdr-facts.toml over one record — the navigator's
whole read in one call (README §Facts, §status). Text is one fact per
line; --json is the neutral vector; --tags renders "--tag k=v" argv for a
resolver. A fact the table declares prose is not rendered as a tag: an
unquoted $(rdr status --tags NNNN) splits on whitespace, so a sentence
would arrive truncated at the first space. Name SEVERAL records for the
set question (Stage 8's predecessors, 7.1's cluster): each is resolved by
name, so the corpus is never scanned, and one that does not resolve is a
"skipped" row with its reason — absent, not "looked and not COMPLETE".
With no argument it is the Draft+Final worklist, each row carrying its
facts; --tags needs one record. --filter keeps only the named facts,
which is what makes a set affordable to read (48 facts ≈ 7KB per record);
a name the table does not declare is refused. It never writes.

receipt asks the usage log whether the record was linted at or after its
last write (README §receipt): exit 0 and the lint's log line; 1 and
stopped:no-lint-receipt; 2 when no log is bound. §commit refuses a record
commit on 1 — the check that catches a gate closed without lint.

paths binds the evidence directory and the iteration number the flow's
skills used to build by hand, from the same models/rdr-facts.toml roots
and [iteration] block the facts read — so a path and a probe cannot
disagree about where evidence lives. With no tree flag it prints the
bound roots; --lens/--cluster/--tree name a tree and print its dir.
--next-iter LISTS that dir and reports the iteration the next pass owes:
1 + the highest segment found, never the lowest absent, with the segments
it found and a note when they are not contiguous. Loose files are
iteration 1 (rdr-common §evidence), so a first pass writes the base
itself. An unbound root is a stated absence and exit 1, never a
fabricated path. It never writes, and creates no directory.

env prints the seam this cwd binds — every marker var, plus
RDR_MARKER and RDR_PROJECT, one quoted k=v per line for eval, or --json.
It answers from the MARKER, not the environment: every other seam read
here lets an exported var win, because --records names one dir for one
call, but re-publishing an inherited RDR_* would let a leak outlive the
turn that made it. Compare RDR_PROJECT against the repo you stand in
before trusting the rest (rdr-common §seam-bind). No marker: exit 1.

exit codes:
  0  success, findings or not
  1  lint: a finding blocks a lock; receipt: no lint since the last write
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

	case "inspect", "index", "lint", "receipt", "status", "env", "paths":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(stderr)
		f := declareFlags(args[0], fs)
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		// The schema is TEMPLATE.md, read now rather than restated in Go.
		// `receipt` is exempt: it reads only the usage log, and §commit
		// refuses a commit without it, so a missing template must not
		// break the commit path over a file receipt never opens.
		// `env` is exempt for the stricter reason: it is what a skill runs
		// to LEARN where the engine is, so requiring the template first
		// would be a cycle — and it opens no record to read against one.
		// `paths` is exempt on env's terms: a record's SLUG is its whole
		// input and that comes off the filename, so it reads no body and
		// has nothing to read against a schema.
		if args[0] != "receipt" && args[0] != "env" && args[0] != "paths" {
			if err := bindSchema(f); err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
		}
		// Measure what this invocation cost to answer. The counter wraps
		// stdout so the size is the real emitted size, not an estimate of
		// it; the log is written on the way out, whatever the exit.
		counted := &countingWriter{w: stdout}
		stdout = counted
		started := stamp()
		cmd, target := args[0], strings.Join(fs.Args(), " ")
		f.argc = len(fs.Args())
		code := 2
		defer func() {
			logUsage(usageRecord{
				TS:        started.Format(time.RFC3339),
				Cmd:       cmd,
				Facet:     usageFacet(cmd, f, target),
				Target:    target,
				BytesOut:  counted.n,
				ElapsedMS: stamp().Sub(started).Milliseconds(),
				Exit:      code,
			})
		}()
		code = dispatch(args[0], fs, f, stdout, stderr)
		return code

	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0

	default:
		fmt.Fprintf(stderr, "stopped:unknown-subcommand (%s)\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}

// dispatch routes a parsed subcommand to its facet. It is split from run
// so that every exit path passes through one place that can be measured
// — a facet that returned directly from the switch would escape the log.
func dispatch(cmd string, fs *flag.FlagSet, f *flags, stdout, stderr io.Writer) int {
	switch cmd {
	case "inspect":
		return inspect(fs.Args(), f, stdout, stderr)
	case "index":
		if *f.derived {
			return indexDerived(f, stdout, stderr)
		}
		if *f.coverage {
			return indexCoverage(f, stdout, stderr)
		}
		if f.backlinks.value != "" {
			docs, _, code := corpus(f, stderr)
			if code != 0 {
				return code
			}
			return backlinksTo(docs, f.backlinks.value, f, stdout, stderr)
		}
		if f.backlinks.set || *f.unresolved || *f.clusterOf != "" {
			return indexEdges(f, stdout, stderr)
		}
		if *f.status {
			return statusFacet(f, stdout, stderr)
		}
		if *f.cycles {
			return cyclesFacet(f, stdout, stderr)
		}
		if *f.openJoint {
			return openJointFacet(f, *f.all, stdout, stderr)
		}
		if *f.anchors {
			return anchorFacet(f, stdout, stderr)
		}
		if *f.literals {
			return literalFacet(f, stdout, stderr)
		}
		if f.readme.set {
			return readmeFacet(f, f.readme.value, stdout, stderr)
		}
		return indexGraph(f, stdout, stderr)
	case "lint":
		return lintCmd(fs.Args(), f, stdout, stderr)
	case "receipt":
		return receipt(fs.Args(), f, stdout, stderr)
	case "status":
		return statusCmd(fs.Args(), f, stdout, stderr)
	case "env":
		return envCmd(f, stdout, stderr)
	case "paths":
		return pathsCmd(fs.Args(), f, stdout, stderr)
	}
	fmt.Fprintf(stderr, "stopped:not-implemented (%s)\n", cmd)
	return 2
}

// flags holds the values of every declared flag; a subcommand reads only
// its own.
type flags struct {
	json, all, derived    *bool
	coverage              *bool
	sel, project, records *string
	filter                *string
	// argc is how many positional arguments the invocation carried. The
	// usage log reads it to tell `status NNNN` from `status NNNN NNNN`,
	// which are the same verb at two very different costs.
	argc                int
	repo                *string
	status              *bool
	backlinks, readme   optString
	clusterOf           *string
	unresolved, anchors *bool
	literals            *bool
	openJoint, cycles   *bool
	locking             *bool
	since               *string // receipt: the instant a lint must postdate
	tags                *bool   // status: render the facts as a resolver's argv
	facts               *string // status/paths: the fact table to evaluate
	lens                *string // paths: the per-lens iteration tree
	cluster             *string // paths: Stage 7.1's cluster-keyed tree
	tree                *string // paths: any declared tree, as <name>[=<operand>]
	nextIter            *bool   // paths: list the base, report the next iteration
	template            *string // the schema: TEMPLATE.md (default $RDR_HOME, else beside the binary)
}

// declareFlags registers each subcommand's flags. They are declared here —
// not where a facet reads them — so that `rdr <cmd> --help` states the
// invocation contract before every facet exists, and so an unknown flag
// fails loudly today rather than silently once it does.
func declareFlags(cmd string, fs *flag.FlagSet) *flags {
	f := &flags{}
	records := fs.String("records", envOrSeam("RDR_RECORDS"), "records dir for NNNN lookup (default $RDR_RECORDS, else the marker's, else .)")
	project := fs.String("project", "", "project prefix for ids (cli/NNNN:C4); omitted inside one records dir")
	repo := fs.String("repo", envOrSeam("RDR_SOURCE_REPO"), "repo root for source-anchor symbol resolution (default $RDR_SOURCE_REPO, else the marker's); unset leaves those edges unchecked")
	f.records, f.project, f.repo = records, project, repo
	f.template = fs.String("template", "", "the schema to read records against (default $RDR_HOME/TEMPLATE.md, else beside the binary)")
	switch cmd {
	case "inspect":
		f.json = fs.Bool("json", false, "emit the JSON envelope")
		f.sel = fs.String("select", "", "project one facet: outline|elements|edges|warnings|metadata|fields|anchors|<element-id>")
		f.filter = fs.String("filter", "", "comma-separated envelope keys to keep (metadata,counts,…); identity keys are always included")
		f.all = fs.Bool("all", false, "include facets omitted by default")
	case "index":
		f.json = fs.Bool("json", false, "emit the index as JSON")
		f.derived = fs.Bool("derived", false, "count derived (unlabelled) element ids per record — the labelling backlog")
		f.coverage = fs.Bool("coverage", false, "unclassified-line rate over the records dir, warnings by code, recurring unknown headings and labels — the drift alarm")
		f.all = fs.Bool("all", false, "anchor-intersect: every record, not only those in flight")
		f.status = fs.Bool("status", false, "group records by status")
		fs.Var(&f.backlinks, "backlinks", "the reverse edge table; =NNNN[:elem] answers who cites one target, mentions included")
		f.clusterOf = fs.String("cluster-of", "", "the record's cluster by 7.1's membership rule")
		f.unresolved = fs.Bool("unresolved", false, "typed edges whose target was looked for and not found")
		f.cycles = fs.Bool("cycles", false, "dependency shapes the flow cannot progress through: ownership cycles (predecessor/overrides/moved-to), Joint-check home cycles, and Final records whose home is Draft or whose check is OPEN")
		f.openJoint = fs.Bool("open-joint", false, "open joint decisions across in-flight records: Joint-check lines whose home is OPEN, and Status qualifiers in joint-decision form; --all: every record")
		f.anchors = fs.Bool("anchor-intersect", false, "pairs of in-flight records citing the same code anchors, uncited pairs first")
		f.literals = fs.Bool("literal-intersect", false, "pairs of in-flight records whose contracts share a backticked literal, uncited pairs first")
		f.filter = fs.String("filter", "", "comma-separated graph keys to keep (records,elements,edges,backlinks); identity keys are always included")
		fs.Var(&f.readme, "readme", "drift between the README index table and the records; =PATH names the README")
	case "lint":
		f.json = fs.Bool("json", false, "emit findings as JSON")
		f.locking = fs.Bool("locking", false, "the record is at a lock gate: resolution findings block, exit 1")
	case "receipt":
		f.since = fs.String("since", "", "RFC3339 instant the lint must postdate (default: the record's mtime)")
	case "env":
		f.json = fs.Bool("json", false, "emit the bound seam as a JSON map")
	case "paths":
		f.json = fs.Bool("json", false, "emit the bound paths as JSON")
		f.facts = fs.String("facts", "", "the fact table to read roots and the iteration convention from (default $RDR_HOME/models/rdr-facts.toml, else beside the binary)")
		f.lens = fs.String("lens", "", "the lens whose evidence dir to bind (grounding|3amigo|critique|repeatability|cove)")
		f.cluster = fs.String("cluster", "", "the Stage 7.1 cluster key whose dir to bind (the members' numbers joined)")
		f.tree = fs.String("tree", "", "any tree the table declares, as <name>[=<operand>]")
		f.nextIter = fs.Bool("next-iter", false, "list the bound dir and report the iteration the next pass should write")
	case "status":
		f.json = fs.Bool("json", false, "emit the fact vector as JSON")
		f.tags = fs.Bool("tags", false, "render the facts as `--tag k=v` argv for a resolver (one record only)")
		f.facts = fs.String("facts", "", "the fact table to evaluate (default $RDR_HOME/models/rdr-facts.toml, else beside the binary)")
		f.filter = fs.String("filter", "", "comma-separated fact names to keep (impl_state,status); a name the table does not declare is refused")
	}
	return f
}

// resolve turns a NNNN or a path into a record path. A bare number is
// looked up as NNNN-*.md in the records dir.
func resolve(arg, records string) (string, error) {
	// The corpus cites its own records as `<dir>/NNNN` — `cli/0112` under
	// a records dir at …/rdr/cli — and lint's fix hints print that
	// spelling back. Read as a path it named `./cli/0112`, a file nobody
	// asked for, and cost the caller a retry turn per citation.
	arg = localCitation(arg, records)
	// A caller who types `3` means record 0003. Only the flow's own shell
	// helpers zero-pad today, so a direct call — which is how a stage
	// prompt reaches this binary — used to fall through to the path
	// branch and fail with `open 3: no such file`, an error naming a file
	// nobody asked for. The number is the flow's vocabulary; read it.
	if n := shortRecordNumber(arg); n != "" {
		arg = n
	}
	// A bare slug (`0142-data-cli-surfaces-label-opt-in`) names a record
	// as surely as its number does, and the flow writes slugs constantly
	// — `$RDR_SLUG` is bound beside `$RDR_PATH`. Treated as a cwd-relative
	// path it failed with `no such file`, the same misdirection as a bare
	// number. It is a record name when it leads with NNNN- and carries no
	// separator of its own.
	if ident.RecordOf(arg) != arg && ident.RecordOf(arg) != "" &&
		!strings.ContainsRune(arg, filepath.Separator) && !strings.HasSuffix(arg, ".md") {
		if dir := recordsDir(records); dir != "" {
			if p := filepath.Join(dir, arg+".md"); fileExists(p) {
				return p, nil
			}
		}
	}
	if ident.RecordOf(arg) == arg {
		dir, tried := resolveRecordsDir(records)
		matches, _ := filepath.Glob(filepath.Join(dir, arg+"-*.md"))
		if len(matches) == 0 {
			matches, _ = filepath.Glob(filepath.Join(dir, arg+".md"))
		}
		matches = recordFiles(matches)
		switch len(matches) {
		case 0:
			return "", fmt.Errorf("stopped:no-such-record (%s in %s%s)", arg, absOrSelf(dir), whereItLooked(tried))
		case 1:
			return matches[0], nil
		default:
			sort.Strings(matches)
			return "", fmt.Errorf("stopped:ambiguous-record (%s: %s)", arg, strings.Join(matches, ", "))
		}
	}
	return arg, nil
}

// localCitation strips the records dir's own name off a citation: with
// records at …/rdr/cli, `cli/0112` is `0112` and `cli/0112:A3` is
// `0112:A3`. The prefix is the DIRECTORY'S basename, read off the bound
// dir rather than spelled here, because the corpus writes its citations
// with exactly that word and nothing else makes it a project name. Any
// other prefix is left alone: it names another records dir, and a
// lookup here would find the wrong record or none.
func localCitation(arg, records string) string {
	prefix, rest, ok := strings.Cut(arg, "/")
	if !ok || prefix == "" || strings.Contains(rest, "/") {
		return arg
	}
	if !ident.IsRecord(rest) && !ident.IsID(rest) {
		return arg
	}
	if prefix != filepath.Base(absOrSelf(recordsDir(records))) {
		return arg
	}
	return rest
}

// splitCitation reads an element citation used as a positional: `0112:A3`
// or `cli/0112:A3` is the record 0112 with A3 selected. It is the form the
// corpus and lint print, and pasting it back is how a caller reaches the
// bytes a finding names without re-spelling it as two arguments.
func splitCitation(arg, records string) (record, sel string) {
	local := localCitation(arg, records)
	id, err := ident.Parse(local)
	if err != nil {
		return arg, ""
	}
	return id.Record, local
}

// identityKeys are carried by every filtered envelope, unasked. They cost
// ~120 bytes together and answer "which record is this, and is the
// projection I am reading the one I asked for" — a question whose absence
// costs a whole turn to re-establish.
var identityKeys = []string{"schema", "record", "path"}

// filterEnvelope projects only the named top-level keys.
//
// `--select` answers "give me exactly one facet" and is the right tool
// when one facet is what you need. This answers the other question: a
// caller who needs metadata AND counts had to spend two invocations —
// and in an agent loop an invocation is a TURN, which re-sends the whole
// conversation. Two turns to read 5 KB out of a 130 KB envelope is the
// expensive shape this closes.
//
// The saving is real because the envelope is lopsided: on a large record
// `elements` and `edges` are ~80% of it, so a caller wanting status and
// counts pays 130 KB for 5 KB of answer, and `inspect --json` is bigger
// than the record it read on a quarter of the corpus.
//
// Keys are the envelope's own JSON names, read off the marshalled
// document rather than a hand-kept list, so a filter can never name a
// key the envelope does not have and no second list can drift from the
// struct. An unknown key is a stop, never a silent empty result: a
// consumer that asked for `elments` must be told, not handed `{}` and
// left to conclude the record has none.
func filterEnvelope(doc *scan.Document, filter string) (map[string]json.RawMessage, error) {
	return filterKeys(doc, filter, identityKeys)
}

// filterKeys is the projection both --filter flags share: marshal, keep
// the named top-level keys plus the identity ones, and stop on a key the
// document does not have.
//
// It is ONE function because the two callers must not drift. `inspect`
// filters a record envelope and `index` filters the corpus graph — the
// question is the same ("give me these keys and not the rest") and so is
// the failure that matters: a filter naming a key that does not exist
// must be told, not handed an empty object it will read as an answer.
func filterKeys(v any, filter string, identity []string) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("stopped:unprojectable (%v)", err)
	}
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("stopped:unprojectable (%v)", err)
	}

	out := map[string]json.RawMessage{}
	for _, k := range identity {
		if v, ok := full[k]; ok {
			out[k] = v
		}
	}
	for _, name := range strings.Split(filter, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		v, ok := full[name]
		if !ok {
			return nil, fmt.Errorf("stopped:no-such-facet (%s; have %s)",
				name, strings.Join(envelopeKeys(full), " "))
		}
		out[name] = v
	}
	return out, nil
}

// envelopeKeys lists what a filter may name, sorted, for the error that
// tells a caller what it could have asked for instead.
func envelopeKeys(full map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(full))
	for k := range full {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// shortRecordNumber zero-pads a bare record number, or returns "" when
// the argument is not one. `3`, `03` and `003` all name `0003`.
//
// Deliberately narrow: only an all-digit string of fewer than four
// digits. A four-digit string is already a record number, and anything
// carrying a separator, a slash or a suffix is a path or an ID and is
// left exactly as written. Leading zeros are stripped decimally, never
// as octal — the bug that once resolved `0106` to `0070` and read the
// wrong record without erroring.
func shortRecordNumber(arg string) string {
	if arg == "" || len(arg) >= 4 {
		return ""
	}
	for _, r := range arg {
		if r < '0' || r > '9' {
			return ""
		}
	}
	n, err := strconv.Atoi(arg)
	if err != nil || n <= 0 {
		return ""
	}
	return fmt.Sprintf("%04d", n)
}

// recordsDir decides which directory a lookup reads.
//
// A relative `--records` is resolved against `$RDR_RECORDS` when that
// names a real directory, not against the process's cwd. A stage prompt
// runs from whatever directory the harness happened to be in, so a
// relative path that is correct in one cwd silently names nothing in
// another — and the tool reported `no-such-record`, blaming the record
// for a directory that was never read. The marker knows where records
// live; ask it before giving up.
//
// An absolute `--records` is always obeyed as written, and a relative
// one that does resolve from the cwd wins too: this only rescues the
// case that would otherwise have found nothing.
func recordsDir(records string) string {
	dir, _ := resolveRecordsDir(records)
	return dir
}

// resolveRecordsDir returns the directory to read and a one-line account
// of how it was chosen, so a failure can say where it looked instead of
// echoing back the relative word the caller typed.
//
// That echo is the whole reason this exists. `docs/rdr holds no NNNN-*.md`
// names no directory a user can check: it is true of a hundred places,
// and it does not say whether the tool read the cwd, the marker, or some
// join of the two. Every candidate below is recorded, and the caller
// prints the trail on failure.
//
// Candidates, in order, first hit wins:
//
//	absolute            obeyed exactly, always — a spelled-out path means itself
//	the cwd             a relative path that resolves from here is what was meant
//	$RDR_RECORDS        when it already ENDS with the relative path (the common
//	                    shape: --records rdr/cli under a marker at …/x/rdr/cli)
//	$RDR_RECORDS/<rel>  only when that really exists, never invented
//	$RDR_RECORDS        the marker alone, when nothing else resolved
func resolveRecordsDir(records string) (string, []string) {
	env := envOrSeam("RDR_RECORDS")
	var tried []string
	note := func(what, path string) { tried = append(tried, what+" "+path) }

	if records == "" {
		if env != "" {
			note("$RDR_RECORDS", env)
			return env, tried
		}
		note("cwd", ".")
		return ".", tried
	}
	if filepath.IsAbs(records) {
		note("--records", records)
		return records, tried
	}
	if abs, err := filepath.Abs(records); err == nil && dirExists(abs) {
		note("--records under the cwd", abs)
		return records, tried
	} else if err == nil {
		note("--records under the cwd", abs)
	}
	if env == "" {
		return records, tried
	}
	// $RDR_RECORDS may already BE the dir the caller named relatively.
	if strings.HasSuffix(filepath.Clean(env), string(filepath.Separator)+filepath.Clean(records)) {
		note("$RDR_RECORDS (already ends with it)", env)
		return env, tried
	}
	joined := filepath.Join(env, records)
	if dirExists(joined) {
		note("$RDR_RECORDS + --records", joined)
		return joined, tried
	}
	note("$RDR_RECORDS + --records", joined)
	// Deliberately NOT falling back to $RDR_RECORDS alone. A caller who
	// passed `--records nope/here` asked for a directory; answering out
	// of the marker instead would return a real, plausible corpus for a
	// path that names nothing — the tool guessing, and the one failure
	// mode worse than an error, because the answer looks right. The
	// relative path stands, and the trail says every place it was sought.
	return records, tried
}

// absOrSelf prints a directory as an absolute path when it can, because
// a relative one names no place a reader can go and check.
func absOrSelf(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// whereItLooked renders the resolution trail, but only when more than
// one candidate was considered — on the ordinary absolute path there is
// nothing to explain and the noise would be pure cost.
func whereItLooked(tried []string) string {
	if len(tried) < 2 {
		return ""
	}
	return "; looked in: " + strings.Join(tried, ", ")
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
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
	arg := args[0]
	if record, sel := splitCitation(arg, *f.records); sel != "" {
		if *f.sel != "" && *f.sel != sel {
			fmt.Fprintf(stderr, "stopped:usage (%s names an element and --select names %s; pass one)\n", arg, *f.sel)
			return 2
		}
		arg, *f.sel = record, sel
	}
	*f.sel = localCitation(*f.sel, *f.records)
	path, err := resolve(arg, *f.records)
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
	if showsEdges(f) {
		resolveEdges(doc, f, stderr)
	}

	if f.filter != nil && *f.filter != "" {
		out, err := filterEnvelope(doc, *f.filter)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		return emit(out, stdout, stderr)
	}

	var out any = doc
	switch sel := *f.sel; sel {
	case "":
	case "outline":
		out = doc.Outline
	case "elements":
		out = doc.Elements
	case "edges":
		out = doc.Edges
	case "warnings":
		out = doc.Warnings
	case "metadata":
		out = doc.Metadata
	case "fields":
		out = doc.Fields
	case "anchors":
		out = doc.Anchors
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
	fmt.Fprintf(w, "%s  %s  %d lines\n", doc.Record, doc.Title, doc.Lines)
	// Sections first, nested by level: the read plan for a record too big
	// for one call, without the 700-line JSON outline that used to be the
	// only way to get these ranges.
	for _, n := range doc.Outline {
		indent := strings.Repeat("  ", n.Level-1)
		fmt.Fprintf(w, "§ %-36s %5d-%-5d %s%s\n", n.ID, n.LineStart, n.LineEnd, indent, n.Heading)
	}
	for _, e := range doc.Elements {
		// `~` marks an id a label would pin. A kind the template does not
		// label carries no mark: its ordinal is the identity, so there is
		// nothing for a reader to act on.
		mark := " "
		if e.Backlog {
			mark = "~"
		}
		fmt.Fprintf(w, "%s %-24s %5d-%-5d %s\n", mark, e.ID, e.LineStart, e.LineEnd, e.Label)
	}
	for _, wn := range doc.Warnings {
		fmt.Fprintf(w, "! %-24s %5d-%-5d %s\n", wn.Code, wn.LineStart, wn.LineEnd, wn.Message)
	}
	fmt.Fprintf(w, "derived: %s\n", derivedLine(doc.Counts))
	// The read instruction lands where the ranges are read: a model that
	// has just seen `118-1128` reaches for sed -n; the id beside it is the
	// call that returns the same bytes and survives the next edit.
	fmt.Fprintf(w, "read: rdr inspect --select <id> %s   (one section or element; never sed -n on these ranges)\n", doc.Record)
	return 0
}

// derivedLine reports the labelling backlog per kind, and the ids that
// are not a backlog in a `structural:` tail. A kind whose every minted id
// is structural — BR, F and MVV, which the template keys nowhere — never
// shows a `0/540` column, because that reads like work already done
// rather than work that was never there.
func derivedLine(c scan.Counts) string {
	var parts []string
	var structural []string
	for _, k := range ident.Kinds {
		if c.Elements[k] == 0 {
			continue
		}
		if c.Derived[k] > 0 || c.Structural[k] == 0 {
			parts = append(parts, fmt.Sprintf("%s %d/%d", k, c.Derived[k], c.Elements[k]))
		}
		if c.Structural[k] > 0 {
			structural = append(structural, fmt.Sprintf("%s %d", k, c.Structural[k]))
		}
	}
	line := strings.Join(parts, "  ")
	if len(structural) > 0 {
		line += "  | structural: " + strings.Join(structural, " ")
	}
	return line
}

// derivedRow is one record's labelling backlog.
type derivedRow struct {
	Record     string             `json:"record"`
	Path       string             `json:"path"`
	Elements   map[ident.Kind]int `json:"elements"`
	Derived    map[ident.Kind]int `json:"derived"`
	Structural map[ident.Kind]int `json:"structural"`
	Warnings   int                `json:"warnings"`
}

// indexDerived walks the records dir and reports, per record and in
// total, how many elements carry derived ids — the backlog the labelling
// rule burns down.
func indexDerived(f *flags, stdout, stderr io.Writer) int {
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return code
	}
	total := scan.Counts{Elements: map[ident.Kind]int{}, Derived: map[ident.Kind]int{}, Structural: map[ident.Kind]int{}}
	var rows []derivedRow
	for _, doc := range docs {
		rows = append(rows, derivedRow{doc.Record, doc.Path, doc.Counts.Elements, doc.Counts.Derived, doc.Counts.Structural, len(doc.Warnings)})
		for k, n := range doc.Counts.Elements {
			total.Elements[k] += n
		}
		for k, n := range doc.Counts.Derived {
			total.Derived[k] += n
		}
		for k, n := range doc.Counts.Structural {
			total.Structural[k] += n
		}
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "records": rows, "total": total, "skipped": skipped}, stdout, stderr)
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%s %3dw  %s\n", r.Record, r.Warnings, derivedLine(scan.Counts{Elements: r.Elements, Derived: r.Derived, Structural: r.Structural}))
	}
	fmt.Fprintf(stdout, "total %d records  %s\n", len(rows), derivedLine(total))
	for _, p := range skipped {
		fmt.Fprintf(stdout, "skipped %s (not an RDR: no Metadata Status and no Critical Assumptions)\n", p)
	}
	return 0
}

// records walks the records dir, scanning every NNNN-*.md that is a
// record. Files that are not records are returned as skipped, never
// silently dropped.
func records(f *flags, stderr io.Writer) (docs []*scan.Document, skipped []string, code int) {
	dir, tried := resolveRecordsDir(*f.records)
	paths, _ := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9][0-9]-*.md"))
	paths = recordFiles(paths)
	sort.Strings(paths)
	if len(paths) == 0 {
		fmt.Fprintf(stderr, "stopped:no-records (%s holds no NNNN-*.md%s)\n",
			absOrSelf(dir), whereItLooked(tried))
		return nil, nil, 2
	}
	for _, p := range paths {
		doc, err := scan.File(p, scan.Options{Project: *f.project})
		if err != nil {
			fmt.Fprintf(stderr, "stopped:unreadable (%s: %v)\n", p, err)
			return nil, nil, 2
		}
		if !doc.IsRecord() {
			skipped = append(skipped, p)
			continue
		}
		docs = append(docs, doc)
	}
	return docs, skipped, 0
}

// coverageRow is one record's unclassified-line count.
type coverageRow struct {
	Record       string  `json:"record"`
	Path         string  `json:"path"`
	Lines        int     `json:"lines"`
	Unclassified int     `json:"unclassified"`
	Rate         float64 `json:"rate"`
	Warnings     int     `json:"warnings"`
}

// recurring is a heading or label the model does not know, seen in more
// than one record. One record's invention is the author's; the same text
// across records is a convention — or a TEMPLATE.md addition whose
// entry was never written, which is what the drift alarm points at.
type recurring struct {
	Kind string `json:"kind"` // heading | label
	Text string `json:"text"`
	// Level is the heading level; Section is the label's template section.
	Level   int    `json:"level,omitempty"`
	Section string `json:"section,omitempty"`
	Records int    `json:"records"`
}

// recurThreshold is how many records must share an unknown heading or
// author label before index --coverage lists it.
const recurThreshold = 3

// indexCoverage is the drift alarm: the unclassified-line rate over the
// corpus, warnings by code, and every unknown heading or author label
// that recurs across records.
func indexCoverage(f *flags, stdout, stderr io.Writer) int {
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return code
	}
	var rows []coverageRow
	total := scan.Coverage{}
	byCode := map[string]int{}
	type key struct {
		kind, text, section string
		level               int
	}
	seen := map[key]map[string]bool{}
	note := func(k key, record string) {
		if seen[k] == nil {
			seen[k] = map[string]bool{}
		}
		seen[k][record] = true
	}
	for _, d := range docs {
		c := d.Coverage
		rows = append(rows, coverageRow{d.Record, d.Path, c.Lines, c.Unclassified, c.Rate, len(d.Warnings)})
		total.Lines += c.Lines
		total.Unclassified += c.Unclassified
		for _, w := range d.Warnings {
			byCode[w.Code]++
		}
		// A heading or label inside a foreign section is that section's
		// warning, not a recurrence of its own: only the foreign root and
		// the author's structure inside recognised sections are counted.
		byID := map[string]scan.Node{}
		for _, n := range d.Outline {
			byID[n.ID] = n
		}
		insideForeign := func(id string) bool {
			for n, ok := byID[id]; ok && n.Parent != ""; n, ok = byID[n.Parent] {
				if byID[n.Parent].Match == "unknown" {
					return true
				}
			}
			return false
		}
		canon := map[string]string{}
		for _, n := range d.Outline {
			canon[n.ID] = n.Canonical
			if (n.Match == "unknown" || n.Match == "author-subsection") && !insideForeign(n.ID) {
				note(key{"heading", n.Heading, "", n.Level}, d.Record)
			}
		}
		fields := append([]scan.Field{}, d.Metadata...)
		fields = append(fields, d.Fields...)
		for _, e := range d.Elements {
			fields = append(fields, e.Fields...)
		}
		for _, fl := range fields {
			if fl.Match == "author" && !insideForeign(fl.Section) && byID[fl.Section].Match != "unknown" {
				note(key{"label", fl.Label, canon[fl.Section], 0}, d.Record)
			}
		}
	}
	if total.Lines > 0 {
		total.Rate = float64(total.Unclassified) / float64(total.Lines)
	}
	var recur []recurring
	for k, recs := range seen {
		if len(recs) >= recurThreshold {
			recur = append(recur, recurring{k.kind, k.text, k.level, k.section, len(recs)})
		}
	}
	sort.Slice(recur, func(i, j int) bool {
		a, b := recur[i], recur[j]
		if a.Records != b.Records {
			return a.Records > b.Records
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Text < b.Text
	})
	if recur == nil {
		recur = []recurring{}
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "records": rows, "total": total,
			"warnings": byCode, "recurring": recur, "skipped": skipped}, stdout, stderr)
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%s %6d lines %4d unclassified %2dw\n", r.Record, r.Lines, r.Unclassified, r.Warnings)
	}
	fmt.Fprintf(stdout, "total %d records  %d lines  %d unclassified  rate %.4f\n", len(rows), total.Lines, total.Unclassified, total.Rate)
	codes := make([]string, 0, len(byCode))
	for c := range byCode {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		fmt.Fprintf(stdout, "warning %-28s %d\n", c, byCode[c])
	}
	for _, r := range recur {
		where := fmt.Sprintf("level %d", r.Level)
		if r.Kind == "label" {
			where = "§" + r.Section
			if r.Section == "" {
				where = "outside any template section"
			}
		}
		fmt.Fprintf(stdout, "recurring %-7s %-48q %-28s %d records\n", r.Kind, r.Text, where, r.Records)
	}
	for _, p := range skipped {
		fmt.Fprintf(stdout, "skipped %s (not an RDR: no Metadata Status and no Critical Assumptions)\n", p)
	}
	return 0
}

// showsEdges reports whether the projection asked for can carry a
// `resolved` verdict. Only edges[] does: the whole envelope, `--select
// edges`, or a --filter naming edges. Every other facet is answered from
// the record alone, so the corpus scan and repo walk resolution costs —
// ~1.5s at 157 records over a 4k-file repo, ~30ms without — are not paid
// for a `--filter counts` or a `--select 0055:A3` that could not show it.
func showsEdges(f *flags) bool {
	if f.filter != nil && *f.filter != "" {
		for _, k := range strings.Split(*f.filter, ",") {
			if strings.TrimSpace(k) == "edges" {
				return true
			}
		}
		return false
	}
	sel := ""
	if f.sel != nil {
		sel = *f.sel
	}
	return sel == "edges" || (sel == "" && f.json != nil && *f.json)
}

// resolveEdges decides one record's edges against the records dir it
// lives in. `inspect` is a single-record command, so it has no corpus of
// its own; it builds one from the dir the record was resolved out of,
// when there is one. With no dir — a loose path, no --records and no
// $RDR_RECORDS — nothing is checked, and every edge says `resolved`
// absent rather than claiming a verdict it did not reach.
//
// The corpus it builds is the EDGE TARGETS, not the directory. A verdict
// about an edge is decided by the target's own projection, so every
// record the document does not name was parsed and thrown away: 144
// records read to answer a question about 15 of them, which was 1.85s of
// a 2.10s call and the single largest cost in the tool.
//
// It resolves through ResolveAll rather than Resolve even though the set
// is one document, because ResolveAll is what PRIMES THE SYMBOL CACHE.
// Unprimed, every distinct symbol the record cites greps the repo on its
// own, so a record citing 42 symbols walks the source tree 42 times: 1.45s
// against retrofit, where one primed walk is 0.56s, for identical
// verdicts. Priming is not an optimisation of the whole-corpus path that
// happens to be reusable here — it is the only reason the walk is stated
// as a single pass, and the single-record path was reading the corpus
// walk's contract without taking it.
func resolveEdges(doc *scan.Document, f *flags, stderr io.Writer) []*scan.Document {
	dir := *f.records
	if dir == "" && doc.Path != "" {
		// A record inspected by path resolves against its own directory:
		// that is where its peers are, and it is what the author means by
		// `cli/0055` in a record already sitting in `cli/`.
		if d := filepath.Dir(doc.Path); d != "" && d != "." {
			dir = d
		}
	}
	if dir == "" {
		// No corpus to check element targets against — but a source root
		// is a separate authority. Symbol resolution reads `--repo` alone,
		// so it still runs; only the element half goes absent.
		scan.NewResolver(nil, *f.repo).ResolveAll([]*scan.Document{doc})
		return nil
	}
	docs, err := scanTargets(dir, *f.project, edgeTargets(doc))
	if err != nil {
		// Resolution is best-effort here: a records dir that cannot be
		// walked leaves the *element* edges unchecked, which is the honest
		// state. It is never a reason to fail a projection of the record
		// asked for — nor to withhold the source-anchor verdicts `--repo`
		// can still reach on its own.
		fmt.Fprintf(stderr, "note:unresolved-edges (%v)\n", err)
		scan.NewResolver(nil, *f.repo).ResolveAll([]*scan.Document{doc})
		return nil
	}
	// The dir WAS read, so a target it does not hold is missing from the
	// corpus rather than unchecked — stated, because a document whose
	// edges all dangle yields an empty set that must still resolve false.
	scan.NewResolverOver(docs, *f.repo, true).ResolveAll([]*scan.Document{doc})
	return docs
}

// edgeTargets is the set of record numbers a document's edges name — the
// only records that can change any verdict about it.
//
// It reads the target the same way resolveElement does, because it is
// answering the same question one step earlier: strip a `project/`
// prefix, drop an `:element` suffix, keep what is four digits. A kind
// whose target is not an element (a symbol, an artifact path, an issue)
// names no record and contributes nothing.
func edgeTargets(doc *scan.Document) map[string]bool {
	want := map[string]bool{}
	for _, e := range doc.Edges {
		if e.Kind.Class() != edge.TargetElement {
			continue
		}
		num := e.To
		if before, _, ok := strings.Cut(num, ":"); ok {
			num = before
		}
		if _, after, ok := strings.Cut(num, "/"); ok {
			num = after
		}
		if ident.RecordOf(num) == num && num != "" {
			want[num] = true
		}
	}
	return want
}

// scanTargets scans only the records a document's edges name.
//
// Resolution asks one question of the corpus — does this target exist,
// and does the element inside it exist — and that question is answered by
// the TARGET's own projection. Every other record in the dir is read,
// parsed and discarded. On the reference corpus one record's edges name
// 15 peers out of 144 records: 2.3MB of 14.5MB, and the whole-dir scan
// was 1.85s of a 2.10s `inspect --json`.
//
// It reports whether the DIR held records, not whether the wanted set
// did. A dir with no NNNN-*.md is the unwalkable case the caller
// announces; a dir full of records none of which the document names is a
// successful scan that yields nothing, and its targets resolve false.
func scanTargets(dir, project string, want map[string]bool) (docs []*scan.Document, err error) {
	dir, tried := resolveRecordsDir(dir)
	all, _ := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9][0-9]-*.md"))
	all = recordFiles(all)
	if len(all) == 0 {
		return nil, fmt.Errorf("%s holds no NNNN-*.md%s", absOrSelf(dir), whereItLooked(tried))
	}
	var paths []string
	for _, p := range all {
		if want[ident.RecordOf(filepath.Base(p))] {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		doc, e := scan.File(p, scan.Options{Project: project})
		if e != nil {
			return nil, fmt.Errorf("%s: %w", p, e)
		}
		if !doc.IsRecord() {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// scanDir scans every record in a directory. It is the corpus builder
// both the index facets and inspect's resolver use.
func scanDir(dir, project string) (docs []*scan.Document, skipped []string, err error) {
	// Same rescue as a single-record lookup: `index` and `lint` read a
	// relative --records the way `inspect` does, so one wrong cwd does
	// not report an empty corpus.
	dir, tried := resolveRecordsDir(dir)
	paths, _ := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9][0-9]-*.md"))
	paths = recordFiles(paths)
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("%s holds no NNNN-*.md%s", absOrSelf(dir), whereItLooked(tried))
	}
	for _, p := range paths {
		doc, e := scan.File(p, scan.Options{Project: project})
		if e != nil {
			return nil, nil, fmt.Errorf("%s: %w", p, e)
		}
		if !doc.IsRecord() {
			skipped = append(skipped, p)
			continue
		}
		docs = append(docs, doc)
	}
	return docs, skipped, nil
}

// edgeRow is one edge as the index reports it, with the record it leaves.
type edgeRow struct {
	Record   string    `json:"record"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Kind     edge.Kind `json:"kind"`
	Resolved *bool     `json:"resolved,omitempty"`
	Line     int       `json:"line"`
	LineEnd  int       `json:"line_end"`
	Field    string    `json:"field,omitempty"`
	Evidence string    `json:"evidence,omitempty"`
}

// indexEdges serves the three corpus-level edge facets.
//
//	--unresolved  every typed edge whose target was looked for and not found
//	--backlinks   the reverse edge set: who points at each target
//	--cluster-of  the cluster of a record, derived from the edge graph
//
// Only two of the three can SHOW a `resolved` verdict, and only those two
// pay for one. `--unresolved` is a query over the verdict itself and
// `--backlinks` carries it on every row; `--cluster-of` is a walk of the
// edge GRAPH — three edge kinds and a direction test — and reads no
// verdict at all. Resolving for it walked the source tree for every symbol
// the corpus cites to answer a question about record relations: 14.1s
// where the traversal alone is 0.89s, for byte-identical output.
//
// This is the same shape as the worklist that ran the full resolver it
// never used, and the same rule as inspect's: resolve when a facet can
// show it, never because the corpus happened to be in hand.
func indexEdges(f *flags, stdout, stderr io.Writer) int {
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return code
	}

	switch {
	case *f.unresolved:
		scan.NewResolver(docs, *f.repo).ResolveAll(docs)
		return unresolvedFacet(docs, skipped, f, stdout, stderr)
	case f.backlinks.set:
		scan.NewResolver(docs, *f.repo).ResolveAll(docs)
		return backlinksFacet(docs, f, stdout, stderr)
	default:
		return clusterFacet(docs, *f.clusterOf, f, stdout, stderr)
	}
}

// unresolvedFacet is the lint finding class this issue names: a typed
// edge whose target was looked for and is not there — a Peer-RDR record
// citing `0055 A9` when 0055 has A1 through A7.
//
// Mentions are excluded. A bare prose reference states no relation, so a
// missing target is not a broken promise; including them would bury the
// typed findings under thousands of prose references to records that
// live in another dir or were never written.
func unresolvedFacet(docs []*scan.Document, skipped []string, f *flags, stdout, stderr io.Writer) int {
	var rows []edgeRow
	for _, d := range docs {
		for _, e := range d.Edges {
			if !e.Kind.Typed() || e.Resolved == nil || *e.Resolved {
				continue
			}
			rows = append(rows, edgeRow{d.Record, e.From, e.To, e.Kind, e.Resolved, e.Line, e.LineEnd, e.Field, e.Evidence})
		}
	}
	if *f.json {
		if rows == nil {
			rows = []edgeRow{}
		}
		return emit(map[string]any{"schema": schemaVersion, "unresolved": rows, "skipped": skipped}, stdout, stderr)
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%s:%d-%d %-22s %s -> %s  %s\n", r.Record, r.Line, r.LineEnd, r.Kind, r.From, r.To, r.Field)
	}
	fmt.Fprintf(stdout, "total %d unresolved typed edges over %d records\n", len(rows), len(docs))
	if len(rows) > 0 {
		fmt.Fprintln(stdout, "each is a record data error: correct the reference text in the named range, nothing else")
	}
	return 0
}

// backlinksFacet transposes the forward edges. Reverse edges are derived,
// never re-parsed: an inbound query is a lookup in this table.
func backlinksFacet(docs []*scan.Document, f *flags, stdout, stderr io.Writer) int {
	back := scan.Reverse(docs)
	targets := make([]string, 0, len(back))
	for t := range back {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	if *f.json {
		out := map[string][]edgeRow{}
		for _, t := range targets {
			for _, e := range back[t] {
				out[t] = append(out[t], edgeRow{recordOfID(e.From), e.From, e.To, e.Kind, e.Resolved, e.Line, e.LineEnd, e.Field, ""})
			}
		}
		return emit(map[string]any{"schema": schemaVersion, "backlinks": out}, stdout, stderr)
	}
	for _, t := range targets {
		var typed []string
		for _, e := range back[t] {
			if e.Kind.Typed() {
				typed = append(typed, fmt.Sprintf("%s(%s)", e.From, e.Kind))
			}
		}
		if len(typed) == 0 {
			continue
		}
		sort.Strings(typed)
		fmt.Fprintf(stdout, "%-28s <- %s\n", t, strings.Join(typed, " "))
	}
	return 0
}

// clusterFacet derives a record's cluster from the edge graph. It is the
// 7.1 prompt's own membership rule, expressed as a query rather than as
// an LLM reading every candidate: related = mutual Predecessors, Peer-RDR
// citations, or a shared Cross-Cutting Concern owner.
func clusterFacet(docs []*scan.Document, of string, f *flags, stdout, stderr io.Writer) int {
	// `--cluster-of 113` means record 0113, exactly as `inspect 113` does:
	// resolve() already zero-pads its positional argument, and refusing the
	// same vocabulary here cost the caller a retry turn.
	if n := shortRecordNumber(of); n != "" {
		of = n
	}
	seed := ident.RecordOf(of)
	if seed == "" {
		fmt.Fprintf(stderr, "stopped:usage (--cluster-of takes a record number, got %q)\n", of)
		return 2
	}
	members := scan.ClusterOf(docs, seed)
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "seed": seed, "cluster": members}, stdout, stderr)
	}
	for _, m := range members {
		fmt.Fprintf(stdout, "%s %-18s %-12s %s\n", m.Record, m.Relation, m.Status, m.Title)
	}
	fmt.Fprintf(stdout, "cluster of %s: %d members\n", seed, len(members))
	return 0
}

// recordOfID reads the record number out of an element or document ID.
func recordOfID(id string) string {
	if _, after, ok := strings.Cut(id, "/"); ok {
		id = after
	}
	before, _, _ := strings.Cut(id, ":")
	return before
}

// lintCmd runs the conformance authority over one record, or over the
// whole records dir when given no argument.
//
// Exit codes carry the verdict, because the callers are gates: 0 is
// PASS, 1 is BLOCK. That is why lint and not inspect owns a non-zero
// exit — a projection is never a verdict, and a gate needs one. A
// findings-but-no-block run still exits 0: advice that stopped a stage
// would be a block wearing another name.
func lintCmd(args []string, f *flags, stdout, stderr io.Writer) int {
	opts := lint.Options{Locking: *f.locking}

	var docs []*scan.Document
	switch len(args) {
	case 0:
		var code int
		docs, _, code = records(f, stderr)
		if code != 0 {
			return code
		}
		scan.NewResolver(docs, *f.repo).ResolveAll(docs)
		opts.Corpus = docs
	case 1:
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
		opts.Corpus = resolveEdges(doc, f, stderr)
		docs = []*scan.Document{doc}
	default:
		fmt.Fprintln(stderr, "stopped:usage (lint takes at most one NNNN or path)")
		return 2
	}

	reports := make([]lint.Report, 0, len(docs))
	block := false
	for _, d := range docs {
		r := lint.Run(d, opts)
		reports = append(reports, r)
		if r.Verdict == "BLOCK" {
			block = true
		}
	}

	if *f.json {
		var out any = reports
		if len(reports) == 1 {
			out = reports[0]
		}
		if code := emit(out, stdout, stderr); code != 0 {
			return code
		}
	} else {
		lintText(reports, stdout)
	}
	if block {
		return 1
	}
	return 0
}

// lintText is the human form: findings grouped under their record, tier
// first so the reader can see at a glance which of the three authorities
// spoke, and a verdict line per record.
func lintText(reports []lint.Report, w io.Writer) {
	for _, r := range reports {
		state := "live"
		if r.Terminal {
			state = "terminal"
		}
		fmt.Fprintf(w, "%s  %s  %s  %s\n", r.Record, r.Status, state, r.Verdict)
		for _, fd := range r.Findings {
			mark := " "
			if fd.Blocking {
				mark = "!"
			}
			fmt.Fprintf(w, "%s %-10s %-28s %5d-%-5d %s\n", mark, fd.Tier, fd.Code, fd.LineStart, fd.LineEnd, fd.Message)
			if fd.Fix != "" {
				fmt.Fprintf(w, "  %-10s %-28s %11s fix: %s\n", "", "", "", fd.Fix)
			}
			// The patch is printed as the lines it would write, so a
			// reader can judge the repair without running the applier.
			// A finding with advice but no patch is the normal case: it
			// says the repair is a decision, not a computation.
			if fd.Patch != nil {
				fmt.Fprintf(w, "  %-10s %-28s %11s patch: %s %d-%d\n", "", "", "",
					fd.Patch.Op, fd.Patch.LineStart, fd.Patch.LineEnd)
				for _, l := range strings.Split(fd.Patch.Text, "\n") {
					fmt.Fprintf(w, "  %-10s %-28s %11s   + %s\n", "", "", "", l)
				}
			}
		}
	}
	if len(reports) > 1 {
		blocked := 0
		findings := 0
		for _, r := range reports {
			findings += len(r.Findings)
			if r.Verdict == "BLOCK" {
				blocked++
			}
		}
		fmt.Fprintf(w, "total %d records  %d findings  %d blocking\n", len(reports), findings, blocked)
	}
}
