// Command recs is the deterministic, read-only reader for the numbered
// records this engine keeps. It never writes to one. Markdown remains the
// source of truth; this binary only projects it.
//
// It installs as $RDR_HOME/bin/recs, with $RDR_HOME/bin/rdr a symlink to
// it: frozen records and evidence spell the command that way, and their
// content is never amended.
//
// Usage:
//
//	recs inspect <NNNN|slug|path> [--json] [--filter k1,k2] [--select outline|elements|warnings|<element-id>] [--grep TEXT] [--touched-since REV] [--project P] [--records DIR]
//	recs index [--json] [--status|--backlinks[=ID]|--cluster-of N|--anchor-intersect|--literal-intersect|--unresolved|--derived|--coverage|--readme[=PATH]|--row-json NNNN] [--records DIR]
//	recs lint [<NNNN|path>] [--locking] [--json] [--records DIR]
//	recs status [<NNNN|slug|path>…] [--json|--tags|--flat|--checklist|--argv] [--filter f1,f2] [--except f3,f4] [--facts PATH] [--records DIR]
//	recs impact <NNNN|slug|path> [--literal TOKEN]... [--model PATH] [--repo DIR] [--json] [--records DIR]
//	recs env [--json]
//	recs version
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
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/lint"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
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

const usage = `recs — read-only projector for RDR markdown records

usage:
  recs inspect <NNNN|slug|path> [--json] [--filter k1,k2] [--select <facet>|<id>] [--grep TEXT] [--touched-since REV] [--all] [--project P] [--records DIR] [--repo DIR]
  recs index [--json] [<facet>] [--filter k1,k2] [--records DIR] [--repo DIR]
  recs lint [<NNNN|path>] [--locking] [--json] [--records DIR]
  recs receipt <NNNN|path> [--since RFC3339] [--records DIR]
  recs status [<NNNN|slug|path>…] [--json|--tags|--flat|--checklist|--argv] [--filter f1,f2] [--except f3,f4] [--facts PATH] [--records DIR]
  recs paths <NNNN|slug|path> [--lens L|--cluster KEY|--tree N[=OP]] [--next-iter] [--json]
  recs anchors --record <NNNN|slug|path> [--unresolved] FILE...
  recs impact <NNNN|slug|path> [--literal TOKEN]... [--model PATH] [--repo DIR] [--json] [--records DIR]
  recs env [--json]
  recs version

index with no facet is the corpus graph: every record, element and edge,
plus the derived backlinks (README §Queries over the graph). Facets:
  --status                    every record grouped by status
  --backlinks[=NNNN[:elem]]   who points at each target / at one target
  --cluster-of NNNN[,NNNN]    7.1's membership rule as a query; --closure runs it to a fixpoint,
                              --final-unimplemented drops not-Final and COMPLETE members to out_of_scope
  --topo[=NNNN,…]             build order over predecessor edges (Kahn; ties by Priority, then number); --edges predecessors,overrides adds the override edges
  --anchor-intersect [--all]  in-flight pairs sharing code anchors, uncited first
  --literal-intersect [--all] in-flight pairs whose contracts share a literal, uncited first
  --open-joint [--all]        open joint decisions: Joint-check (home: OPEN) lines + joint-decision Status forms
  --cycles                    ownership cycles, Joint-check home cycles, homes ahead of a lock, OPEN checks on a Final
  --unresolved                typed edges with no target — record data errors
  --derived                   the unlabelled-element backlog per record
  --coverage                  the drift alarm: unclassified-line rate, unknowns
  --readme[=PATH]             the README index table checked against the records
  --row-json NNNN             one record's index row as JSON (no corpus walk)

--filter keeps only the named top-level keys — of inspect's envelope
(metadata,counts,…) or of the index graph (records,elements,edges,backlinks) —
identity keys always included — one call where --select would need several.
Only edges[] carries "resolved", and deciding it scans the records dir and
walks --repo: the whole envelope, --select edges, --filter …edges and lint
pay that (~1.5s on a large corpus); every other facet answers from the
record alone (~20ms). --filter assumptions is a derived facet — one row per
Critical Assumption: id, status.{value,tier}, method.{members,
off_vocabulary}, evidence.{line_start,line_end,anchors[]} — the gate
questions in ~10KB where elements is ~140KB; an anchor's "resolved" is
present only when edges were resolved in the same call
(--filter assumptions,edges).
inspect with no flag is the summary: one line per section and element — id,
line range, byte size and a label capped at 100 runes — the cheap read plan
(~100-200 lines, under 21KB on the largest records); --select elements is
every element as JSON, uncapped and ~25× larger. --select <id> then names a
section or element to read;
repeat it for several, answered in order. --grep TEXT asks which elements
hold the literal (case-sensitive, fixed string): one row per containing
element — id, range, matching line numbers, first matching line — a bare
miss answers no-match at exit 0; it does not combine with --select/--filter.
--touched-since REV asks which ids the record's diff from REV to the
working tree overlaps (git diff -U0, post-image hunks; a pure deletion
touches the element ending at or beginning after it): elements then
sections, one row each — id, range; --json adds "kind" and carries the
hunks as evidence, so an empty answer is [] beside them. A record
renamed since REV diffs as a whole-file add, so every id is touched. A
rev git cannot diff is stopped:no-diff, never an empty set; it does not
combine with --grep/--select, and --filter may name only elements.
A record is named by number (3,
03, 0003), slug, path, or the corpus's own citation (cli/0003, and
cli/0003:A2 selects the element); --records defaults to $RDR_RECORDS and a
relative one resolves against it. Name SEVERAL records for the set: each
is resolved by name, one that does not is a "skipped" row, and --json
carries identity per row.

element ids (README §identifiers):
  NNNN:A3 assumption · NNNN:C4 contract · NNNN:D-identity decision · NNNN:RT1
  invariant · NNNN:ALT2 alternative · NNNN:BR3 briefly rejected · NNNN:S5
  scenario (list item or T-5 table row) · NNNN:S-1 / NNNN:L-3 contract
  clause (the label as written inside a normative fence) · NNNN:MVV ·
  NNNN:F2 failure mode ·
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
resolver; --flat renders the flat JSON object of strings a declared
command reader returns (intrastate RDR 0025), which is what lets an
accessor read a record's own state back after a write.
A fact the table declares prose is not rendered as a tag: an
unquoted $(recs status --tags NNNN) splits on whitespace, so a sentence
would arrive truncated at the first space. Name SEVERAL records for the
set question (Stage 8's predecessors, 7.1's cluster): each is resolved by
name, so the corpus is never scanned, and one that does not resolve is a
"skipped" row with its reason — absent, not "looked and not COMPLETE".
With no argument it is the Draft+Final worklist, each row carrying its
facts; --tags needs one record. --filter keeps only the named facts,
which is what makes a set affordable to read (48 facts ≈ 7KB per record);
a name the table does not declare is refused. --checklist renders one
record's stage checklist three-valued ("?" is a root nothing looked under,
never "–"), as text or as a "checklist" array beside the facts under --json.
--argv is one line per record — slug, Status, the "--tag k=v" argv, the
qualifier, tab-separated — and its worklist form carries Deferred rows,
which is what bin/rdr-next loops over. It never writes.

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
fabricated path. It never writes, and creates no directory. Under a tree
that declares a cap, PRIOR_DIR names where the last pass wrote and
ITER_BUCKET is ITER against the cap ("1".."cap" or "over") — the loop tag
models/rdr-loop.toml routes on.

anchors prints, sorted and unique, the element ids the given files cite
that the record's projection mints — the outline, element and anchor ids
inspect lists — one per line, so two findings ledgers are diffed with
comm rather than re-read side by side. A peer record's id is a citation,
not an anchor, and is skipped; --unresolved prints instead the tokens
shaped like this record's ids that name nothing it mints.

impact predicts which predecessor tests the record's contract changes
will turn red — the list Stage 8's implementer meets up front rather
than one red test at a time. It reads the override and predecessor
records off the record's own Metadata edges (never resolved), and
--literal names the tokens the change retires (repeatable, fixed string,
case-sensitive). Which files are tests and how a name pins a record is
a convention of the SOURCE repo, declared in models/rdr-impact.toml
(--model; default $RDR_HOME/models, else beside the binary) and chosen
by its detect file. Only files the convention's glob names are opened,
each once; a file is predicted when a test in it pins a set member or
its body carries a literal, and every test in a predicted file is a
row, grouped by family. Text is the body of <art>/impact.md; --json the
same with stable keys. An unbound --repo or a repo no convention detects
is a stop, never rows: 0 — a tree nothing looked at must not read as a
tree with no impact. It never writes.

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
		fmt.Fprintf(stdout, "recs %s (schema %s)\n", version, schemaVersion)
		return 0

	case "inspect", "index", "lint", "receipt", "status", "env", "paths", "anchors", "impact":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(stderr)
		f := declareFlags(args[0], fs)
		// The flag package stops parsing at the first positional, so a flag
		// typed after the target (`recs status 0020 --tags`) is the natural
		// argv shape, not a mistake — hoist it ahead of the positionals so
		// Parse sees it and it is accepted, rather than refused.
		if err := fs.Parse(hoistFlags(fs, args[1:])); err != nil {
			return 2
		}
		// Each invocation of this process (or, in a test binary, each call to
		// run()) resolves its own roots; a bind from a PRIOR invocation must
		// not answer for this one, since recordsDir below itself falls back
		// through envOrSeam and would otherwise see its own last answer
		// before this call has had a chance to overwrite it.
		bindFlag("RDR_RECORDS", "")
		bindFlag("RDR_SOURCE_REPO", "")
		// A resolved `--records`/`--repo` must outrank envOrSeam for every
		// OTHER reader of the same var — NewFactEnv's roots, the fact table's
		// path, resolveRecordsDir's own $RDR_RECORDS fallback — not only the
		// one flag that named it. Without this, `--records X` picked the
		// record from X while artifact/evidence roots kept reading the
		// env/marker tree, silently: two directories answering as one.
		// Binding the resolved default to itself is harmless, so this always
		// runs rather than only when the flag was actually typed.
		// Only when something NAMED a directory (a flag, env or marker):
		// an unconfigured repo's "." default must keep its roots unbound,
		// so its facts read absent, not false.
		if *f.records != "" {
			if dir := recordsDir(*f.records); dirExists(dir) {
				bindFlag("RDR_RECORDS", dir)
			}
		}
		if *f.repo != "" {
			bindFlag("RDR_SOURCE_REPO", *f.repo)
		}
		// The registry tree a `jdr:` citation resolves against. Bound here
		// for the same reason the two above are: one place, so every facet
		// that resolves edges reads the same tree. Unset leaves every
		// registry citation unchecked, which is the honest answer for a
		// consumer that has not adopted the class.
		scan.SetJDRRoot(envOrSeam("RDR_JDRS"))
		scan.SetRFDRoot(envOrSeam("RDR_RFDS"))
		// The same two roots, to the reader that asks which TEMPLATE.md a
		// file is judged against. They are the same fact — a tier's tree —
		// read by two packages, so they bind from one place: a lint that
		// judged a registry by the RDR template while the resolver read it
		// as a registry would be the two halves disagreeing about what the
		// file is.
		model.SetClassRoots(envOrSeam("RDR_JDRS"), envOrSeam("RDR_RFDS"))
		// Every stopped: line lands on BOTH streams. Sessions habitually
		// 2>/dev/null a read they expect to succeed, and a stated absence
		// that lives only on the suppressed stream reads as an empty
		// success. env is exempt: its stdout is eval'd, and a mirrored
		// stop would be executed rather than read.
		if args[0] != "env" {
			stderr = stopMirror{stderr: stderr, stdout: stdout}
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

// stopMirror is the stderr every subcommand but env writes: a chunk that
// leads with `stopped:` is copied to stdout before it goes to stderr.
// Each fmt.Fprint* is one Write, so a stop line arrives whole and
// nothing else is duplicated.
type stopMirror struct{ stderr, stdout io.Writer }

func (m stopMirror) Write(p []byte) (int, error) {
	if bytes.HasPrefix(p, []byte("stopped:")) {
		m.stdout.Write(p)
	}
	return m.stderr.Write(p)
}

// dispatch routes a parsed subcommand to its facet. It is split from run
// so that every exit path passes through one place that can be measured
// — a facet that returned directly from the switch would escape the log.
func dispatch(cmd string, fs *flag.FlagSet, f *flags, stdout, stderr io.Writer) int {
	switch cmd {
	case "inspect":
		return inspect(fs.Args(), f, stdout, stderr)
	case "index":
		// First: it is the one index facet that reads no records, and
		// dispatching it ahead of the rest keeps it that way.
		if *f.rowJSON != "" {
			// A trailing PATH names the README; a command reader needs it
			// positional (see rowFacet).
			var readmePath string
			if rest := fs.Args(); len(rest) > 0 {
				readmePath = rest[0]
			}
			return rowFacet(f, *f.rowJSON, readmePath, stdout, stderr)
		}
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
		if f.topo.set {
			return topoFacet(f, stdout, stderr)
		}
		if *f.status {
			return statusFacet(f, stdout, stderr)
		}
		if *f.cycles {
			return cyclesFacet(f, stdout, stderr)
		}
		if *f.jdrMembers != "" {
			return jdrMembersFacet(f, *f.jdrMembers, *f.all, stdout, stderr)
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
	case "anchors":
		return anchorsCmd(fs.Args(), f, stdout, stderr)
	case "impact":
		return impactCmd(fs.Args(), f, stdout, stderr)
	}
	fmt.Fprintf(stderr, "stopped:not-implemented (%s)\n", cmd)
	return 2
}

// hoistFlags reorders argv so every flag token precedes every positional,
// which is what fs.Parse requires to see a flag typed after the target
// (`recs status 0020 --tags`) rather than stopping at the target and
// leaving the flag as a stray positional. A flag that takes a value keeps
// its neighbour (`--select A9`, `--grep -seed-label`); an undefined flag
// moves alone, so Parse still refuses it with its own "flag provided but
// not defined" message, exactly as when it is typed first. A literal `--`
// ends the scan: it and everything after it stay in place as positionals.
func hoistFlags(fs *flag.FlagSet, args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positionals = append(positionals, args[i:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			positionals = append(positionals, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue
		}
		if fl := fs.Lookup(name); fl != nil && !isBoolFlag(fl) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positionals...)
}

// isBoolFlag asks the flag package's own question: a boolean flag never
// consumes the next argument, any other kind does.
func isBoolFlag(fl *flag.Flag) bool {
	b, ok := fl.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}

// flags holds the values of every declared flag; a subcommand reads only
// its own.
type flags struct {
	json, all, derived *bool
	coverage           *bool
	project, records   *string
	sel                multiString // inspect: every --select, in order
	grep               *string     // inspect: the literal whose containing elements to name
	touchedSince       *string     // inspect: the rev whose diff to the working tree scopes the ids
	filter             *string
	except             *string // status: fact names to drop from the vector
	// argc is how many positional arguments the invocation carried. The
	// usage log reads it to tell `status NNNN` from `status NNNN NNNN`,
	// which are the same verb at two very different costs.
	argc                int
	repo                *string
	status              *bool
	backlinks, readme   optString
	rowJSON             *string // index: one index-table row, by record number
	clusterOf           *string
	closure             *bool     // index: --cluster-of to a fixpoint
	finalUnimplemented  *bool     // index: --cluster-of scoped to Final-and-unimplemented
	topo                optString // index: build order over predecessor edges
	edges               *string   // index --topo: the edge kinds ordered over
	unresolved, anchors *bool
	literals            *bool
	openJoint, cycles   *bool
	jdrMembers          *string
	record              *string // index: scope the pair facets to one record
	locking             *bool
	since               *string     // receipt: the instant a lint must postdate
	tags                *bool       // status: render the facts as a resolver's argv
	flat                *bool       // status: render the facts as a command reader's flat object
	checklist           *bool       // status: render the stage checklist (one record)
	argv                *bool       // status: one tab-separated line per record with its argv
	facts               *string     // status/paths: the fact table to evaluate
	lens                *string     // paths: the per-lens iteration tree
	cluster             *string     // paths: Stage 7.1's cluster-keyed tree
	tree                *string     // paths: any declared tree, as <name>[=<operand>]
	nextIter            *bool       // paths: list the base, report the next iteration
	model               *string     // impact: the convention table to read
	literal             multiString // impact: every --literal, in order
	template            *string     // the schema: TEMPLATE.md (default $RDR_HOME, else beside the binary)
}

// multiString is a flag given several times. The flag package's String
// keeps only the last value, which is how `--select A20 --select A21`
// answered A21 alone without a word; this keeps every one, in order.
type multiString struct{ values []string }

func (m *multiString) String() string     { return strings.Join(m.values, ",") }
func (m *multiString) Set(v string) error { m.values = append(m.values, v); return nil }

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
		fs.Var(&f.sel, "select", "project a facet: "+strings.Join(selectFacets, "|")+"|<element-id>; repeat for several, answered in order")
		f.grep = fs.String("grep", "", "name the elements whose lines carry this literal (case-sensitive, fixed string); no-match answers exit 0; not with --select or --filter")
		f.touchedSince = fs.String("touched-since", "", "name the ids whose lines the diff from REV to the working tree touches (git diff -U0, post-image hunks); a rev git cannot diff stops; not with --grep or --select")
		f.filter = fs.String("filter", "", "comma-separated envelope keys to keep (metadata,counts,…); identity keys are always included")
		f.all = fs.Bool("all", false, "include facets omitted by default")
	case "index":
		f.json = fs.Bool("json", false, "emit the index as JSON")
		f.derived = fs.Bool("derived", false, "count derived (unlabelled) element ids per record — the labelling backlog")
		f.coverage = fs.Bool("coverage", false, "unclassified-line rate over the records dir, warnings by code, recurring unknown headings and labels — the drift alarm")
		f.all = fs.Bool("all", false, "anchor-intersect: every record, not only those in flight")
		f.status = fs.Bool("status", false, "group records by status")
		fs.Var(&f.backlinks, "backlinks", "the reverse edge table; =NNNN[:elem] answers who cites one target, mentions included")
		f.clusterOf = fs.String("cluster-of", "", "the records' cluster by 7.1's membership rule (NNNN, or a comma list of seeds)")
		f.closure = fs.Bool("closure", false, "cluster-of: repeat the hop from every asserted member to a fixpoint; candidates are never expanded")
		f.finalUnimplemented = fs.Bool("final-unimplemented", false, "cluster-of: keep Final members whose capsule is not COMPLETE; the rest go to out_of_scope with why")
		f.facts = fs.String("facts", "", "cluster-of --final-unimplemented: the fact table impl_state is read from (default $RDR_HOME/models/rdr-facts.toml, else beside the binary)")
		fs.Var(&f.topo, "topo", "build order over predecessor edges: Kahn, ties by Priority then number; =NNNN,… names the set (default: every in-flight record)")
		f.edges = fs.String("edges", "predecessors", "topo: the edge kinds ordered over — predecessors, or predecessors,overrides (an overridden record builds before its overrider)")
		f.unresolved = fs.Bool("unresolved", false, "typed edges whose target was looked for and not found")
		f.cycles = fs.Bool("cycles", false, "dependency shapes the flow cannot progress through: ownership cycles (predecessor/overrides/moved-to), Joint-check home cycles, and Final records whose home is Draft or whose check is OPEN")
		f.jdrMembers = fs.String("jdr-members", "", "records whose anchors touch a registry's declared seam: `cli/0001` or `0001`; membership is derived, never read off a cluster field")
		f.openJoint = fs.Bool("open-joint", false, "open joint decisions across in-flight records: Joint-check lines whose home is OPEN, and Status qualifiers in joint-decision form; --all: every record")
		f.anchors = fs.Bool("anchor-intersect", false, "pairs of in-flight records citing the same code anchors, uncited pairs first")
		f.literals = fs.Bool("literal-intersect", false, "pairs of in-flight records whose contracts share a backticked literal, uncited pairs first")
		f.record = fs.String("record", "", "anchor-intersect, literal-intersect, open-joint: only the rows touching this record (NNNN, slug, path or citation)")
		f.filter = fs.String("filter", "", "comma-separated graph keys to keep (records,elements,edges,backlinks); identity keys are always included")
		fs.Var(&f.readme, "readme", "drift between the README index table and the records; =PATH names the README")
		f.rowJSON = fs.String("row-json", "", "the index table's row for ONE record (NNNN), as JSON; no corpus walk")
		f.flat = fs.Bool("flat", false, "row-json: emit `{\"readme_status\":\"…\"}` — a declared command reader's flat object")
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
		// No list here: nothing validates the name — the lens tree's
		// `under` is `{lens}`, so any word binds a dir — and the five
		// this once enumerated had fallen behind the table's probes
		// (propose-premortem, reconcile, …). A list the flag does not
		// enforce is guidance that goes wrong on its own.
		f.lens = fs.String("lens", "", "the lens whose evidence dir to bind: a folder name under the table's lens tree; the probe facts (recs status) name the ones the flow reads")
		f.cluster = fs.String("cluster", "", "the Stage 7.1 cluster key whose dir to bind (the members' numbers joined)")
		f.tree = fs.String("tree", "", "any tree the table declares, as <name>[=<operand>]")
		f.nextIter = fs.Bool("next-iter", false, "list the bound dir and report the iteration the next pass should write")
	case "anchors":
		f.record = fs.String("record", "", "the record whose ids the files are read against (NNNN, slug or path)")
		f.unresolved = fs.Bool("unresolved", false, "print instead the tokens shaped like this record's ids that name no element it mints")
	case "impact":
		f.json = fs.Bool("json", false, "emit the prediction as JSON")
		f.model = fs.String("model", "", "the convention table to read (default $RDR_HOME/models/rdr-impact.toml, else beside the binary)")
		fs.Var(&f.literal, "literal", "a token the change retires (fixed string, case-sensitive); a test file carrying it is predicted; repeat for several")
	case "status":
		f.json = fs.Bool("json", false, "emit the fact vector as JSON")
		f.tags = fs.Bool("tags", false, "render the facts as `--tag k=v` argv for a resolver (one record only)")
		f.flat = fs.Bool("flat", false, "render the facts as a flat JSON object of strings — a declared command reader's wire shape (one record only)")
		f.checklist = fs.Bool("checklist", false, "render the stage checklist (one record only): text, or a `checklist` array beside the facts with --json")
		f.argv = fs.Bool("argv", false, "one line per record: slug, Status, its `--tag k=v` argv, qualifier — tab-separated; the worklist form includes Deferred")
		f.facts = fs.String("facts", "", "the fact table to evaluate (default $RDR_HOME/models/rdr-facts.toml, else beside the binary)")
		f.filter = fs.String("filter", "", "comma-separated fact names to keep (impl_state,status); a name the table does not declare is refused")
		f.except = fs.String("except", "", "comma-separated fact names to DROP from the vector (status,readme_status — a model that owns them refuses them as argv); a name the table does not declare is refused")
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
			// Nothing named a records dir and the marker refused to: the
			// refusal is the stop, not the cwd it fell back to.
			if records == "" && envOrSeam("RDR_RECORDS") == "" {
				if why := markerRefusal(); why != "" {
					return "", errors.New(why)
				}
			}
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
	return filterKeys(doc, filter, identityKeys, map[string]any{"assumptions": assumptionsFacet(doc)})
}

// assumptionRow is one Critical Assumption as the exit gates read it:
// the id, its Status as classified, its Method members and the ones in
// no vocabulary, and the Evidence field's span with the source anchors
// it cites. Nothing else — the whole list is ~3KB on a record whose
// `elements` facet is 140KB, and every gate check that walked elements
// for these four fields reads this instead. A field the element lacks is
// omitted, which is the finding (an Evidence Record with no Method).
type assumptionRow struct {
	ID       string         `json:"id"`
	Status   *statusBrief   `json:"status,omitempty"`
	Method   *methodBrief   `json:"method,omitempty"`
	Evidence *evidenceBrief `json:"evidence,omitempty"`
}

type statusBrief struct {
	Value       string `json:"value"`
	Tier        string `json:"tier"`
	Placeholder bool   `json:"placeholder,omitempty"`
}

type methodBrief struct {
	Members       []string `json:"members"`
	OffVocabulary []string `json:"off_vocabulary,omitempty"`
}

type evidenceBrief struct {
	LineStart int         `json:"line_start"`
	LineEnd   int         `json:"line_end"`
	Anchors   []anchorRef `json:"anchors,omitempty"`
}

// anchorRef is a source-anchor edge cut to its symbol and verdict. The
// verdict is three-valued exactly as on the edge: absent unless the
// call also resolved edges (`--filter assumptions,edges`).
type anchorRef struct {
	To       string `json:"to"`
	Resolved *bool  `json:"resolved,omitempty"`
}

func assumptionsFacet(doc *scan.Document) []assumptionRow {
	rows := []assumptionRow{}
	for _, el := range doc.Elements {
		if el.Kind != ident.Assumption {
			continue
		}
		row := assumptionRow{ID: el.ID}
		for i := range el.Fields {
			f := &el.Fields[i]
			switch f.Canonical {
			case "Status":
				if f.Status != nil {
					row.Status = &statusBrief{Value: f.Status.Value, Tier: f.Status.Tier, Placeholder: f.Status.Placeholder}
				}
			case "Method":
				if f.Method != nil {
					row.Method = &methodBrief{Members: f.Method.Members, OffVocabulary: f.Method.OffVocabulary}
				}
			case "Evidence":
				ev := &evidenceBrief{LineStart: f.LineStart, LineEnd: f.LineEnd}
				for _, e := range doc.Edges {
					if e.Kind == edge.SourceAnchor && e.From == el.ID &&
						e.Line >= f.LineStart && e.Line <= f.LineEnd {
						ev.Anchors = append(ev.Anchors, anchorRef{To: e.To, Resolved: e.Resolved})
					}
				}
				row.Evidence = ev
			}
		}
		rows = append(rows, row)
	}
	return rows
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
func filterKeys(v any, filter string, identity []string, derived map[string]any) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("stopped:unprojectable (%v)", err)
	}
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("stopped:unprojectable (%v)", err)
	}
	// A derived facet (`assumptions`) is a view over the envelope rather
	// than a key of it; it is offered under the same rules as the rest.
	for k, d := range derived {
		b, err := json.Marshal(d)
		if err != nil {
			return nil, fmt.Errorf("stopped:unprojectable (%v)", err)
		}
		full[k] = b
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

// inspect is `recs inspect [flags] NNNN…`: one record's projection, or a
// NAMED SET of records', each resolved by name so only those files are
// read. The set is the arity `status` already has, and it exists for the
// same reason: 7.1's critique agent held a list of members and tried
// `inspect 0097 0108 0110` twice before falling back to one call per
// record. A member that does not resolve is a skipped row, not a
// refusal — a set that answers nothing says nothing about any member.
func inspect(args []string, f *flags, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "stopped:usage (inspect takes one or more NNNN, citation or path)")
		return 2
	}
	// --grep is its own question — which elements hold the literal — and
	// crossing it with a projection flag has no one answer to give.
	if f.grep != nil && *f.grep != "" && (len(f.sel.values) > 0 || (f.filter != nil && *f.filter != "")) {
		fmt.Fprintln(stderr, "stopped:usage (--grep answers alone; drop --select/--filter)")
		return 2
	}
	// --touched-since is one question too — which ids the diff overlaps —
	// and its rows are already element-shaped, so --filter elements is
	// the one filter it accepts.
	if f.touchedSince != nil && *f.touchedSince != "" {
		if (f.grep != nil && *f.grep != "") || len(f.sel.values) > 0 {
			fmt.Fprintln(stderr, "stopped:usage (--touched-since answers alone; drop --grep/--select)")
			return 2
		}
		if f.filter != nil && *f.filter != "" && *f.filter != "elements" {
			fmt.Fprintln(stderr, "stopped:usage (--touched-since answers elements; --filter may name only elements)")
			return 2
		}
	}
	for i, sel := range f.sel.values {
		f.sel.values[i] = localCitation(sel, *f.records)
	}
	if len(args) == 1 {
		r, err := project(args[0], f, stderr)
		if err != nil {
			// A select miss is a partial answer, not a refusal: the
			// resolved selects emit first, then the stop names only the
			// ids that missed. One bad id used to zero the whole call.
			if r.value != nil || r.lines != nil {
				r.render(f, stdout, stderr)
			}
			fmt.Fprintln(stderr, err)
			return 2
		}
		return r.render(f, stdout, stderr)
	}
	return inspectSet(args, f, stdout, stderr)
}

// projection is what one record answers: the JSON value when the call is
// a JSON one, the element lines when it is a text `--select`, or neither
// when it is the text summary.
type projection struct {
	doc   *scan.Document
	value any
	lines []string
}

// render is the single-record output, unchanged from before the set
// arity: bytes for a text element select, JSON for everything --json or
// facet-shaped, the summary otherwise.
func (r projection) render(f *flags, stdout, stderr io.Writer) int {
	switch {
	case r.lines != nil && !*f.json:
		for _, l := range r.lines {
			fmt.Fprintln(stdout, l)
		}
		return 0
	case r.value != nil:
		return emit(r.value, stdout, stderr)
	}
	return summary(r.doc, stdout)
}

// inspectSet projects several records in the order named. Text is each
// record's own rendering in sequence — the summary heads itself with the
// record number, and element bytes are printed as the single form prints
// them — with a `skipped` line per member that did not resolve, as
// `status` writes them. `--json` carries identity on every row, which is
// the form to read when the selects are elements of several records.
func inspectSet(args []string, f *flags, stdout, stderr io.Writer) int {
	rows := []map[string]any{}
	skipped := []map[string]string{}
	var results []projection
	for _, arg := range args {
		r, err := project(arg, f, stderr)
		if err != nil {
			skipped = append(skipped, map[string]string{"target": arg, "why": strings.TrimSpace(err.Error())})
			continue
		}
		results = append(results, r)
		row := map[string]any{"record": r.doc.Record, "path": r.doc.Path, "value": r.value}
		if r.value == nil {
			row["value"] = r.doc
		}
		rows = append(rows, row)
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "records": rows, "skipped": skipped}, stdout, stderr)
	}
	for _, r := range results {
		if code := r.render(f, stdout, stderr); code != 0 {
			return code
		}
	}
	for _, s := range skipped {
		fmt.Fprintf(stdout, "%s skipped  %s\n", s["target"], s["why"])
	}
	return 0
}

// project answers one record. A citation used as the argument
// (`cli/0112:A3`) adds its element to the selects, since pasting a
// finding's id back is how a caller reaches its bytes.
//
// REPEATED `--select` ACCUMULATES. Each flag used to overwrite the last
// silently, so `--select A20 --select A21` answered A21 alone with no
// word said, and sessions fell back to one call per element — 64 selects
// in one refine pass. Several selects answer in the order given: text
// element selects print their bytes in sequence, exactly as each would
// alone; anything JSON-shaped is an array of the single forms, a named
// facet wrapped as `{"select": name, name: value}` so it keeps its name.
func project(arg string, f *flags, stderr io.Writer) (projection, error) {
	arg, own := splitCitation(arg, *f.records)
	sels := append([]string(nil), f.sel.values...)
	if own != "" && !hasString(sels, own) {
		sels = append(sels, own)
	}
	path, err := resolve(arg, *f.records)
	if err != nil {
		return projection{}, err
	}
	doc, err := scan.File(path, scan.Options{Project: *f.project})
	if err != nil {
		return projection{}, fmt.Errorf("stopped:unreadable (%v)", err)
	}
	if doc.Record == "" {
		return projection{}, fmt.Errorf("stopped:no-record-number (neither the title nor the filename carries NNNN)")
	}
	// --grep answers before edges resolve: a text search over one record
	// never needs the corpus scan the edge verdicts pay for.
	if f.grep != nil && *f.grep != "" {
		return grepRecord(doc, *f.grep, *f.json), nil
	}
	// --touched-since reads one git diff and the record's ranges; no
	// edge verdict is on the row, so none is paid for.
	if f.touchedSince != nil && *f.touchedSince != "" {
		return touchedSince(doc, *f.touchedSince, *f.json)
	}
	if showsEdges(f) {
		resolveEdges(doc, f, stderr)
	}
	r := projection{doc: doc}

	if f.filter != nil && *f.filter != "" {
		out, err := filterEnvelope(doc, *f.filter)
		if err != nil {
			return projection{}, err
		}
		r.value = out
		return r, nil
	}
	if len(sels) == 0 {
		if *f.json {
			r.value = doc
		}
		return r, nil
	}

	var items []any
	var lines []string
	var missing []string
	textual := !*f.json
	for _, sel := range sels {
		if v, ok := facetOf(doc, sel); ok {
			textual = false
			if len(sels) == 1 {
				items = append(items, v)
			} else {
				items = append(items, map[string]any{"select": sel, sel: v})
			}
			continue
		}
		start, end, ok := doc.Select(sel)
		if !ok {
			// A clause label defined twice in the record names two
			// answers; refusing with both is the honest stop, and it is
			// not the same stop as an id that names nothing.
			if why := doc.Ambiguity(sel); why != "" {
				return projection{}, fmt.Errorf("stopped:ambiguous-element (%s in %s: %s)", sel, doc.Record, why)
			}
			// An id that names nothing is collected, not returned: the
			// selects that DID resolve still answer, and the stop then
			// names only the missing.
			missing = append(missing, sel)
			continue
		}
		items = append(items, map[string]any{"id": sel, "line_start": start, "line_end": end,
			"lines": doc.Slice(start, end)})
		lines = append(lines, doc.Slice(start, end)...)
	}
	if len(sels) == 1 && len(items) == 1 {
		r.value = items[0]
	} else if len(items) > 0 {
		r.value = items
	}
	if textual {
		r.lines = lines
	}
	if len(missing) > 0 {
		return r, noSuchElement(doc, missing)
	}
	return r, nil
}

// noSuchElement is the stop for --select ids the record does not mint.
// One line however many ids missed — a set's skipped row and a stderr
// reader both take it whole — and each id carries a bounded hint. When
// the token stands verbatim in the body, the hint names the minted
// element whose lines hold it: an authored label like F-1 lives inside
// a contract that DOES have an id, and "it is right there" deserves the
// id that reaches it. Otherwise the near misses off the record's own
// roster: one edit away first, then the same kind prefix, capped as the
// lint peer hint caps its list.
func noSuchElement(doc *scan.Document, missing []string) error {
	var hints []string
	for _, sel := range missing {
		if h := missHint(doc, sel, len(missing) > 1); h != "" {
			hints = append(hints, h)
		}
	}
	msg := fmt.Sprintf("stopped:no-such-element (%s in %s", strings.Join(missing, ", "), doc.Record)
	if len(hints) > 0 {
		msg += "; " + strings.Join(hints, "; ")
	} else {
		msg += "; facets: " + strings.Join(selectFacets, " ")
	}
	return errors.New(msg + ")")
}

// missHintIDs bounds the near-miss list; missHintSpans bounds how many
// containing elements a verbatim hit names.
const (
	missHintIDs   = 8
	missHintSpans = 3
)

// missHint explains one missed select. The token is the id's element
// part as typed; with several ids missing each hint says whose it is.
func missHint(doc *scan.Document, sel string, several bool) string {
	token := sel
	if i := strings.LastIndex(token, ":"); i >= 0 {
		token = token[i+1:]
	}
	if token == "" {
		return ""
	}
	if spans := tokenSpans(doc, token); len(spans) > 0 {
		return token + " appears inside " + strings.Join(spans, ", ")
	}
	ids := nearMisses(doc, token)
	if len(ids) == 0 {
		return ""
	}
	label := "near misses"
	if several {
		label += " for " + token
	}
	return label + ": " + lint.Bounded(ids, missHintIDs)
}

// tokenSpans finds the token standing verbatim in the body and names the
// smallest minted element whose range holds each hit, with its lines.
func tokenSpans(doc *scan.Document, token string) []string {
	seen := map[string]bool{}
	var out []string
	for n := 1; n <= doc.Lines && len(out) < missHintSpans; n++ {
		if !carriesToken(doc.Line(n), token) {
			continue
		}
		e := tightestAt(doc, n)
		if e == nil || seen[e.ID] {
			continue
		}
		seen[e.ID] = true
		out = append(out, fmt.Sprintf("%s (%d-%d)", e.ID, e.LineStart, e.LineEnd))
	}
	return out
}

// tightestAt is the smallest element whose range holds the line — the
// clause over its contract, the contract over nothing.
func tightestAt(doc *scan.Document, line int) *scan.Element {
	var best *scan.Element
	for i := range doc.Elements {
		e := &doc.Elements[i]
		if line < e.LineStart || line > e.LineEnd {
			continue
		}
		if best == nil || e.LineEnd-e.LineStart < best.LineEnd-best.LineStart {
			best = e
		}
	}
	return best
}

// carriesToken reports the token on its own in the line, bounded by
// non-alphanumerics, so F-1 is found and F-12 is not.
func carriesToken(line, token string) bool {
	for from := 0; ; {
		j := strings.Index(line[from:], token)
		if j < 0 {
			return false
		}
		j += from
		k := j + len(token)
		if (j == 0 || !wordByte(line[j-1])) && (k == len(line) || !wordByte(line[k])) {
			return true
		}
		from = j + 1
	}
}

func wordByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// grepHit is one containing element the literal was found in.
type grepHit struct {
	ID        string `json:"id"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Lines     []int  `json:"lines"`
	First     string `json:"first"`
}

// grepHitLineNumbers bounds how many matching line numbers a text row
// prints; JSON carries them all.
const grepHitLineNumbers = 10

// grepRecord answers `inspect --grep`: which minted elements hold the
// literal, case-sensitive and fixed — the author-label question ("which
// element defines this term") that used to be a select-per-element hunt.
// One row per containing element: id, range, matching line numbers, the
// first matching line trimmed. A hit inside no element names the
// tightest section, so no line of the record is out of reach. A bare
// miss is a stated absence — `no-match`, exit 0 — not an error.
func grepRecord(doc *scan.Document, literal string, jsonOut bool) projection {
	var order []string
	byID := map[string]*grepHit{}
	for n := 1; n <= doc.Lines; n++ {
		line := doc.Line(n)
		if !strings.Contains(line, literal) {
			continue
		}
		id, start, end := containerOf(doc, n)
		if id == "" {
			continue
		}
		h := byID[id]
		if h == nil {
			h = &grepHit{ID: id, LineStart: start, LineEnd: end, First: clip(strings.TrimSpace(line))}
			byID[id] = h
			order = append(order, id)
		}
		h.Lines = append(h.Lines, n)
	}
	hits := []grepHit{}
	for _, id := range order {
		hits = append(hits, *byID[id])
	}
	r := projection{doc: doc}
	if jsonOut {
		r.value = map[string]any{"schema": schemaVersion, "record": doc.Record, "path": doc.Path,
			"grep": literal, "matches": hits}
		return r
	}
	if len(hits) == 0 {
		r.lines = []string{fmt.Sprintf("no-match  %q in %s", literal, doc.Record)}
		return r
	}
	for _, h := range hits {
		r.lines = append(r.lines, fmt.Sprintf("%-24s %5d-%-5d lines %s  %s",
			h.ID, h.LineStart, h.LineEnd, lineList(h.Lines), h.First))
	}
	return r
}

// lineList joins matching line numbers, bounded for the text row.
func lineList(ns []int) string {
	parts := make([]string, 0, len(ns))
	for i, n := range ns {
		if i == grepHitLineNumbers {
			parts = append(parts, fmt.Sprintf("+%d more", len(ns)-i))
			break
		}
		parts = append(parts, strconv.Itoa(n))
	}
	return strings.Join(parts, ",")
}

// containerOf is the minted id whose range holds the line: the tightest
// element, else the tightest section, each an id --select can read.
func containerOf(doc *scan.Document, line int) (id string, start, end int) {
	if e := tightestAt(doc, line); e != nil {
		return e.ID, e.LineStart, e.LineEnd
	}
	var best *scan.Node
	for i := range doc.Outline {
		n := &doc.Outline[i]
		if line < n.LineStart || line > n.LineEnd {
			continue
		}
		if best == nil || n.LineEnd-n.LineStart < best.LineEnd-best.LineStart {
			best = n
		}
	}
	if best == nil {
		return "", 0, 0
	}
	return best.ID, best.LineStart, best.LineEnd
}

// nearMisses filters the record's own id roster against the token: one
// edit apart first — the typo class — then ids sharing its kind prefix,
// in the order Select searches. Nothing is ever invented; every id
// offered resolves.
func nearMisses(doc *scan.Document, token string) []string {
	var dist1, prefixed []string
	seen := map[string]bool{}
	pfx := kindPrefix(token)
	for _, id := range selectableIDs(doc) {
		if seen[id] {
			continue
		}
		seen[id] = true
		local := id
		if i := strings.LastIndex(local, ":"); i >= 0 {
			local = local[i+1:]
		}
		switch {
		case local == token:
		case oneEditApart(local, token):
			dist1 = append(dist1, id)
		case pfx != "" && kindPrefix(local) == pfx:
			prefixed = append(prefixed, id)
		}
	}
	return append(dist1, prefixed...)
}

// selectableIDs is everything Select can resolve, in its search order:
// elements, then outline sections, then anchors.
func selectableIDs(doc *scan.Document) []string {
	var ids []string
	for _, e := range doc.Elements {
		ids = append(ids, e.ID)
	}
	for _, n := range doc.Outline {
		ids = append(ids, n.ID)
	}
	for _, a := range doc.Anchors {
		ids = append(ids, a.ID)
	}
	return ids
}

// kindPrefix is the token's leading non-digit run: L- for L-2, JC for
// JC5, the whole token when no digit follows. Prefixes are compared
// whole, so A12 neighbours A1 and not ALT2.
func kindPrefix(token string) string {
	for i, r := range token {
		if r >= '0' && r <= '9' {
			return token[:i]
		}
	}
	return token
}

// oneEditApart reports edit distance exactly one, byte-wise: the id
// grammar is ASCII but for §, where a deletion still lines up.
func oneEditApart(a, b string) bool {
	if a == b {
		return false
	}
	if len(a) == len(b) {
		diff := 0
		for i := 0; i < len(a); i++ {
			if a[i] != b[i] {
				diff++
			}
		}
		return diff == 1
	}
	long, short := a, b
	if len(long) < len(short) {
		long, short = short, long
	}
	if len(long) != len(short)+1 {
		return false
	}
	i := 0
	for i < len(short) && long[i] == short[i] {
		i++
	}
	return long[i+1:] == short[i:]
}

// selectFacets names every --select word besides an element id, in the
// order they are offered. It is the single source for both the --select
// help string and the no-such-element refusal's "facets:" hint, so the
// two cannot drift apart.
var selectFacets = []string{
	"outline", "elements", "edges", "warnings", "metadata", "fields", "anchors", "assumptions",
}

// facetOf is the named-facet half of --select: the projection's own
// top-level lists by name. An element id is anything else.
func facetOf(doc *scan.Document, sel string) (any, bool) {
	switch sel {
	case "outline":
		return doc.Outline, true
	case "elements":
		return doc.Elements, true
	case "edges":
		return doc.Edges, true
	case "warnings":
		return doc.Warnings, true
	case "metadata":
		return doc.Metadata, true
	case "fields":
		return doc.Fields, true
	case "anchors":
		return doc.Anchors, true
	case "assumptions":
		return assumptionsFacet(doc), true
	}
	return nil, false
}

func hasString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
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
	// Every range row carries its byte size: line counts do not predict
	// bytes — a dense table weighs pages of prose — and a reader budgeting
	// the next call pulled 65KB elements blind on the count alone.
	for _, n := range doc.Outline {
		indent := strings.Repeat("  ", n.Level-1)
		fmt.Fprintf(w, "§ %-36s %5d-%-5d %6s %s%s\n", n.ID, n.LineStart, n.LineEnd,
			humanBytes(doc.SpanBytes(n.LineStart, n.LineEnd)), indent, n.Heading)
	}
	for _, e := range doc.Elements {
		// `~` marks an id a label would pin. A kind the template does not
		// label carries no mark: its ordinal is the identity, so there is
		// nothing for a reader to act on.
		mark := " "
		if e.Backlog {
			mark = "~"
		}
		fmt.Fprintf(w, "%s %-24s %5d-%-5d %6s %s\n", mark, e.ID, e.LineStart, e.LineEnd,
			humanBytes(e.Bytes), clip(e.Label))
	}
	for _, wn := range doc.Warnings {
		fmt.Fprintf(w, "! %-24s %5d-%-5d %s\n", wn.Code, wn.LineStart, wn.LineEnd, clip(wn.Message))
	}
	fmt.Fprintf(w, "derived: %s\n", derivedLine(doc.Counts))
	// The read instruction lands where the ranges are read: a model that
	// has just seen `118-1128` reaches for sed -n; the id beside it is the
	// call that returns the same bytes and survives the next edit.
	fmt.Fprintf(w, "read: recs inspect --select <id> %s   (a section or element; repeat --select for several; never sed -n on these ranges)\n", doc.Record)
	return 0
}

// summaryLabelRunes bounds one summary row. A table-row scenario or a
// paragraph-long contract carries its whole text as its label, and on a
// large record those rows put the summary past the ~30KB a harness returns
// in one call — the read plan overflowed to a file and got sliced with
// sed. The id is the identity and the label only orients, so the row keeps
// its head and `--select <id>` returns the bytes. JSON is uncapped.
const summaryLabelRunes = 100

// humanBytes renders a size in at most six characters — `843B`, `12.4K`
// — so a summary row pays a fixed column, not a number that grows.
func humanBytes(n int) string {
	if n < 1000 {
		return strconv.Itoa(n) + "B"
	}
	k := float64(n) / 1024
	if k < 100 {
		return fmt.Sprintf("%.1fK", k)
	}
	return fmt.Sprintf("%.0fK", k)
}

// clip bounds a summary label to summaryLabelRunes, marking the cut.
func clip(s string) string {
	if utf8.RuneCountInString(s) <= summaryLabelRunes {
		return s
	}
	r := []rune(s)
	return string(r[:summaryLabelRunes-1]) + "…"
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
		// A clause is keyed by its label or not minted at all, so it
		// has no backlog column to show.
		if c.Elements[k] == 0 || k == ident.Clause {
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
	if hasString(f.sel.values, "edges") {
		return true
	}
	return len(f.sel.values) == 0 && f.json != nil && *f.json
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
//
// `--closure` is the same rule to a fixpoint from every seed (a comma
// list), which is what 7.1 step 1 used to assemble by hand from several
// one-hop calls; `--final-unimplemented` is that step's scope filter,
// evaluated here so the dropped members are listed with why rather than
// silently omitted. A candidate is dropped too: confirming one means
// re-running with it as a seed, never expanding it.
func clusterFacet(docs []*scan.Document, of string, f *flags, stdout, stderr io.Writer) int {
	var seeds []string
	for _, part := range strings.Split(of, ",") {
		part = strings.TrimSpace(part)
		// `--cluster-of 113` means record 0113, exactly as `inspect 113`
		// does: resolve() already zero-pads its positional argument, and
		// refusing the same vocabulary here cost the caller a retry turn.
		if n := shortRecordNumber(part); n != "" {
			part = n
		}
		seed := ident.RecordOf(part)
		if seed == "" {
			fmt.Fprintf(stderr, "stopped:usage (--cluster-of takes a record number or a comma list, got %q)\n", part)
			return 2
		}
		seeds = append(seeds, seed)
	}
	closure := f.closure != nil && *f.closure
	scoped := f.finalUnimplemented != nil && *f.finalUnimplemented
	var members []scan.Member
	if closure || len(seeds) > 1 {
		members = scan.ClusterClosure(docs, seeds)
	} else {
		members = scan.ClusterOf(docs, seeds[0])
	}
	if members == nil {
		members = []scan.Member{}
	}
	out := map[string]any{"schema": schemaVersion, "seed": seeds[0], "seeds": seeds, "closure": closure, "cluster": members}
	var dropped []clusterDrop
	if scoped {
		path, err := factTablePath(deref(f.facts))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		tbl, err := LoadFactTable(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		byRecord := map[string]*scan.Document{}
		for _, d := range docs {
			byRecord[d.Record] = d
		}
		kept := []clusterMember{}
		dropped = []clusterDrop{}
		for _, m := range members {
			state := ""
			if d := byRecord[m.Record]; d != nil {
				state = capsuleStateOf(tbl, NewFactEnv(tbl, d, recordSlug(d.Path)))
			}
			why := ""
			switch {
			case m.Status != "Final":
				why = "not-final"
			case state == "COMPLETE":
				why = "impl-complete"
			case m.Candidate:
				why = "candidate"
			}
			if why != "" {
				dropped = append(dropped, clusterDrop{m.Record, m.Status, state, why})
				continue
			}
			kept = append(kept, clusterMember{m, state})
		}
		out["cluster"], out["out_of_scope"] = kept, dropped
		members = members[:0]
		for _, k := range kept {
			members = append(members, k.Member)
		}
	}
	if *f.json {
		return emit(out, stdout, stderr)
	}
	for _, m := range members {
		via := ""
		if m.Via != "" {
			via = " via " + m.Via
		}
		fmt.Fprintf(stdout, "%s %-18s %-12s %s%s\n", m.Record, m.Relation, m.Status, m.Title, via)
	}
	for _, d := range dropped {
		fmt.Fprintf(stdout, "%s out-of-scope       %-12s %s\n", d.Record, d.Status, d.Why)
	}
	fmt.Fprintf(stdout, "cluster of %s: %d members\n", strings.Join(seeds, ","), len(members))
	return 0
}

// clusterMember is a scoped member with the capsule state that kept it.
type clusterMember struct {
	scan.Member
	ImplState string `json:"impl_state,omitempty"`
}

// clusterDrop is a member the scope filter removed, and why.
type clusterDrop struct {
	Record    string `json:"record"`
	Status    string `json:"status,omitempty"`
	ImplState string `json:"impl_state,omitempty"`
	Why       string `json:"why"`
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
		// The header carries the tier counts so a gate reads the verdict
		// AND its shape from one line, instead of re-running lint under
		// three greps. `advisory` is every finding that does not block.
		var blocking, resolution, placeholder int
		for _, fd := range r.Findings {
			if fd.Blocking {
				blocking++
			}
			if fd.Tier == lint.TierResolution {
				resolution++
			}
			if fd.Code == "placeholder:survived" {
				placeholder++
			}
		}
		fmt.Fprintf(w, "%s  %s  %s  %s  blocking=%d resolution=%d placeholder=%d advisory=%d\n",
			r.Record, r.Status, state, r.Verdict, blocking, resolution, placeholder, len(r.Findings)-blocking)
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
