package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// The corpus facets of `rdr index`: the whole records dir projected once,
// with the flow's standing questions answered as queries over it instead
// of by opening every file. See scan/corpus.go for the queries; this file
// is their rendering.

// optString is a flag that is either present (`--backlinks`, value
// "true"), present with a value (`--backlinks=0055:C4`) or absent. Go's
// flag package has no optional-argument form; a bool-flag Value that
// accepts text is the closest, and it keeps the bare form working.
type optString struct {
	set   bool
	value string
}

func (o *optString) String() string   { return o.value }
func (o *optString) IsBoolFlag() bool { return true }
func (o *optString) Set(v string) error {
	o.set = true
	if v != "true" {
		o.value = v
	}
	return nil
}

// corpus scans and resolves the records dir once for every facet below.
func corpus(f *flags, stderr io.Writer) ([]*scan.Document, []string, int) {
	return corpusResolving(f, stderr, true)
}

// corpusResolving is corpus with the resolution pass made optional.
//
// Resolution is the only thing a corpus read does that reaches outside
// the records dir, and it is most of the wall time: it walks the source
// repo to decide `resolved` on every source anchor. `inspect` already
// refuses to pay for it when the projection asked for cannot SHOW a
// verdict (showsEdges); this is that rule at corpus scale, and it applies
// to exactly one caller — a `--filter` on the graph that keeps no edge
// key.
//
// It is opt-IN rather than inferred everywhere, because the cost of being
// wrong is asymmetric. Skipping resolution for a facet that does show a
// verdict would emit `resolved` absent where it used to be true or false,
// and this file's own doctrine is that an unchecked edge must never read
// as a checked one. So the default stays "resolve", and only a caller
// that has proved the verdict is unreachable may say otherwise.
func corpusResolving(f *flags, stderr io.Writer, resolve bool) ([]*scan.Document, []string, int) {
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return nil, nil, code
	}
	if resolve {
		scan.NewResolver(docs, *f.repo).ResolveAll(docs)
	}
	return docs, skipped, 0
}

// graphShowsEdges reports whether a graph projection can carry a
// `resolved` verdict, and so whether resolution must run.
//
// Only two of the graph's keys carry one: `edges`, which holds the
// verdict itself, and `backlinks`, whose rows are transposed from the
// same edges. Everything else — `records`, `elements`, `skipped` — is
// read out of one record at a time and cannot show it. With no filter the
// whole graph is emitted, which includes both, so the answer is yes.
func graphShowsEdges(f *flags) bool {
	if f.filter == nil || *f.filter == "" {
		return true
	}
	for _, k := range strings.Split(*f.filter, ",") {
		switch strings.TrimSpace(k) {
		case "edges", "backlinks":
			return true
		}
	}
	return false
}

// graphIdentityKeys is what a filtered graph carries unasked. Only the
// schema: the graph is the whole corpus, so there is no record or path to
// identify it by, and every other key is large enough that carrying one
// uninvited would defeat the filter.
var graphIdentityKeys = []string{"schema"}

// indexGraph is the bare `rdr index`: the one graph document, or as text
// the record table an index README carries.
func indexGraph(f *flags, stdout, stderr io.Writer) int {
	// Text output is the record table, which shows no verdict; JSON
	// resolves unless the filter has ruled the verdict out.
	docs, skipped, code := corpusResolving(f, stderr, *f.json && graphShowsEdges(f))
	if code != 0 {
		return code
	}
	g := scan.BuildGraph(docs, skipped)
	if *f.json {
		// The graph is the tool's largest single emission — 5.6 MB on the
		// reference corpus, of which `elements` is 18% and `edges` 33% —
		// and a caller that wants one of those keys has had to take all
		// of them. The joint-decision check reads `elements[] kind=="C"`
		// and paid 5.6 MB for 52 KB of answer; the same read filtered is
		// ~106x smaller. Same flag, same semantics and the same shared
		// projection as inspect's, so a caller learns the idiom once.
		if f.filter != nil && *f.filter != "" {
			out, err := filterKeys(g, *f.filter, graphIdentityKeys, nil)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
			return emit(out, stdout, stderr)
		}
		return emit(g, stdout, stderr)
	}
	for _, r := range g.Records {
		fmt.Fprintf(stdout, "%s %-12s %-8s %s\n", r.Record, statusWord(r), r.Priority, r.ShortTitle)
	}
	fmt.Fprintf(stdout, "total %d records  %d elements  %d edges  %d targets with backlinks\n",
		len(g.Records), len(g.Elements), len(g.Edges), len(g.Backlinks))
	for _, p := range skipped {
		fmt.Fprintf(stdout, "skipped %s (not an RDR: no Metadata Status and no Critical Assumptions)\n", p)
	}
	return 0
}

func statusWord(r scan.Summary) string {
	if r.Status == nil {
		return "-"
	}
	return r.Status.Value
}

// statusFacet groups every record by status — the corpus question.
//
// It once also answered the WORKLIST (`--in-flight`: Draft, and Final
// not yet Implemented), which is a different question with a different
// caller: the navigator, which then had to go and derive each record's
// signals itself. That form moved to `rdr status` with no argument,
// where the facts come with it. Grouping the whole corpus stayed here,
// because it is a question about the corpus and not about what to do
// next.
func statusFacet(f *flags, stdout, stderr io.Writer) int {
	docs, skipped, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	groups := map[string][]scan.Summary{}
	var rows []scan.Summary
	for _, d := range docs {
		s := scan.Summarize(d)
		rows = append(rows, s)
		groups[statusWord(s)] = append(groups[statusWord(s)], s)
	}
	if *f.json {
		if rows == nil {
			rows = []scan.Summary{}
		}
		return emit(map[string]any{"schema": schemaVersion, "records": rows, "skipped": skipped}, stdout, stderr)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ids := make([]string, 0, len(groups[k]))
		for _, s := range groups[k] {
			ids = append(ids, s.Record)
		}
		fmt.Fprintf(stdout, "%-12s %3d  %s\n", k, len(ids), strings.Join(ids, " "))
	}
	fmt.Fprintf(stdout, "total %d records\n", len(docs))
	return 0
}

func qualifier(s scan.Summary) string {
	if s.Status == nil || s.Status.Qualifier == "" {
		return ""
	}
	return "[" + s.Status.Qualifier + "]"
}

// padRecordHalf zero-pads a bare record number inside a --backlinks
// target, so `26`, `26:C4` and `cli/26:C4` reach the same row as `0026`,
// `0026:C4` and `cli/0026:C4`. Only the record half — before any `:`,
// after any `qualifier/` prefix — is a candidate, and only when
// shortRecordNumber accepts it as a bare number; anything else (already
// four digits, or not a number at all) passes through unchanged so the
// usage check below still refuses it.
func padRecordHalf(target string) string {
	prefix := ""
	rest := target
	if before, after, ok := strings.Cut(target, "/"); ok {
		prefix, rest = before+"/", after
	}
	record, suffix, hasColon := strings.Cut(rest, ":")
	padded := shortRecordNumber(record)
	if padded == "" {
		return target
	}
	if hasColon {
		return prefix + padded + ":" + suffix
	}
	return prefix + padded
}

// backlinksTo answers the impact question for one target — "who cites
// 0055:C4?" — or, given a bare record, for the record and every element
// in it. Mentions are included here and marked, because an impact
// analysis wants every reader, typed or not; the whole-table form stays
// typed-only so it remains readable.
func backlinksTo(docs []*scan.Document, target string, f *flags, stdout, stderr io.Writer) int {
	target = padRecordHalf(target)
	if !ident.RecordNumber.MatchString(recordOfID(target)) {
		fmt.Fprintf(stderr, "stopped:usage (--backlinks takes NNNN or NNNN:<element>, got %q)\n", target)
		return 2
	}
	// A bare `0055` matches `cli/0055` too: inside one records dir they
	// are the same record, and authors write both. A qualified target
	// matches exactly.
	qualified := strings.Contains(target, "/")
	whole := !strings.Contains(target, ":")
	matches := func(t string) bool {
		if !qualified {
			if _, after, ok := strings.Cut(t, "/"); ok {
				t = after
			}
		}
		return t == target || (whole && strings.HasPrefix(t, target+":"))
	}
	back := scan.Reverse(docs)
	var rows []edgeRow
	for t, edges := range back {
		if !matches(t) {
			continue
		}
		for _, e := range edges {
			rows = append(rows, edgeRow{recordOfID(e.From), e.From, e.To, e.Kind, e.Resolved, e.Line, e.LineEnd, e.Field, ""})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Kind.Typed() != b.Kind.Typed() {
			return a.Kind.Typed()
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Record != b.Record {
			return a.Record < b.Record
		}
		return a.Line < b.Line
	})
	if *f.json {
		if rows == nil {
			rows = []edgeRow{}
		}
		return emit(map[string]any{"schema": schemaVersion, "target": target, "backlinks": rows}, stdout, stderr)
	}
	typed := 0
	for _, r := range rows {
		if r.Kind.Typed() {
			typed++
		}
		fmt.Fprintf(stdout, "%-28s <- %-22s %-20s %s:%d\n", r.To, r.From, r.Kind, r.Record, r.Line)
	}
	fmt.Fprintf(stdout, "%s: %d typed, %d mentions\n", target, typed, len(rows)-typed)
	return 0
}

// scopeRecord reads `--record`: the one record the pair facets keep rows
// for, in any spelling resolve() accepts (113, 0113, the slug, a path,
// `cli/0113`). Empty means unscoped. A record that does not resolve is
// a stop, never an empty scope: no rows for a misspelt number would read
// as "nothing intersects", the joint-decision arm's false clear.
func scopeRecord(f *flags) (string, error) {
	if f.record == nil || *f.record == "" {
		return "", nil
	}
	path, err := resolve(*f.record, *f.records)
	if err != nil {
		return "", err
	}
	num := ident.RecordOf(filepath.Base(path))
	if num == "" {
		return "", fmt.Errorf("stopped:no-record-number (%s carries no NNNN)", *f.record)
	}
	return num, nil
}

// scopedOverlaps keeps the pairs touching the scoped record. The propose
// stage asks "does THIS record appear in a pair", and answered it by
// filtering every pair in the corpus with inline python, twice in one
// session; the corpus is still scanned once — pairs need both sides —
// but the answer is the record's rows alone.
func scopedOverlaps(overlaps []scan.Overlap, record string) []scan.Overlap {
	if record == "" {
		return overlaps
	}
	kept := []scan.Overlap{}
	for _, o := range overlaps {
		if o.Records[0] == record || o.Records[1] == record {
			kept = append(kept, o)
		}
	}
	return kept
}

// scopeNote is the text form's account of the scope, beside the count.
func scopeNote(record string) string {
	if record == "" {
		return ""
	}
	return " touching " + record
}

// anchorFacet is the after-propose scan: pairs of in-flight records that
// cite the same code anchors, the uncited pairs first. `--all` widens it
// to every record; `--record` keeps one record's pairs.
func anchorFacet(f *flags, stdout, stderr io.Writer) int {
	record, err := scopeRecord(f)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	docs, _, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	overlaps := scopedOverlaps(scan.AnchorIntersect(docs, !*f.all), record)
	if *f.json {
		if overlaps == nil {
			overlaps = []scan.Overlap{}
		}
		return emit(map[string]any{"schema": schemaVersion, "open_only": !*f.all, "record": record, "overlaps": overlaps}, stdout, stderr)
	}
	fires := 0
	for _, o := range overlaps {
		mark := "cited"
		if !o.Cited {
			mark, fires = "UNCITED", fires+1
		}
		fmt.Fprintf(stdout, "%s %s %-7s %d shared: %s\n", o.A, o.B, mark, len(o.Anchors), strings.Join(o.Anchors, " "))
	}
	scope := "in-flight"
	if *f.all {
		scope = "all"
	}
	fmt.Fprintf(stdout, "total %d overlapping pairs over %s records%s, %d with no cross-citation\n", len(overlaps), scope, scopeNote(record), fires)
	if fires > 0 {
		fmt.Fprintln(stdout, "an uncited pair shares code neither record acknowledges: ask the joint-decision question before either locks")
	}
	return 0
}

// readmeFacet checks a records dir's README index table against the
// records. It reports; it never edits.
func readmeFacet(f *flags, path string, stdout, stderr io.Writer) int {
	docs, _, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	if path == "" {
		dir := *f.records
		if dir == "" {
			dir = "."
		}
		path = filepath.Join(dir, "README.md")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "stopped:unreadable (%s: %v)\n", path, err)
		return 2
	}
	rows := scan.ParseReadmeIndex(strings.Split(string(body), "\n"))
	if len(rows) == 0 {
		fmt.Fprintf(stderr, "stopped:no-index-table (%s has no `| NNNN |` index rows)\n", path)
		return 2
	}
	drift := scan.ReadmeDrift(docs, rows)
	if *f.json {
		if drift == nil {
			drift = []scan.Drift{}
		}
		return emit(map[string]any{"schema": schemaVersion, "readme": path, "rows": len(rows), "records": len(docs), "drift": drift}, stdout, stderr)
	}
	for _, d := range drift {
		switch d.Kind {
		case "missing-row":
			fmt.Fprintf(stdout, "%s missing-row     record has no row: %s\n", d.Record, d.Actual)
		case "extra-row", "duplicate-row":
			fmt.Fprintf(stdout, "%s %-15s %s:%d\n", d.Record, d.Kind, path, d.Line)
		default:
			fmt.Fprintf(stdout, "%s %-15s readme %q  record %q  %s:%d\n", d.Record, d.Kind, d.Readme, d.Actual, path, d.Line)
		}
	}
	fmt.Fprintf(stdout, "total %d drift over %d rows, %d records\n", len(drift), len(rows), len(docs))
	return 0
}

// rowFacet answers what the index table says about ONE record, addressed
// by its number. It reports; it never edits.
//
// It exists because a write to the README row needs a reader for its
// read-back, and the row lives in a sibling document keyed by number —
// the same read `readmeRow` in facts.go makes for the write side, and the
// same rows `--readme` checks for drift, so none of the three can
// disagree about what a row says.
//
// Unlike every other facet here it does NOT walk the corpus: a row read
// needs no records, and making it pay for 157 of them would put a corpus
// scan inside every write's read-back. That is the whole reason it is
// separate from `--readme`, which compares the table AGAINST the records
// and so must scan them.
//
// The three answers are distinguished, because a writer acts differently
// on each: a row (emit it), no row (`readme --add` is the op — exit 0
// with a null row, since "looked and found none" is an answer), or no
// table at all (nothing looked — exit 2, the same `stopped:no-index-table`
// the write table's own row emits).
func rowFacet(f *flags, record, readmePath string, stdout, stderr io.Writer) int {
	// `RecordOf` takes the number off a bare `0055` or a filename/path
	// base, so the caller may pass whatever it already holds.
	num := ident.RecordOf(filepath.Base(record))
	if num == "" {
		fmt.Fprintf(stderr, "stopped:not-a-record-number (%q does not begin with NNNN)\n", record)
		return 2
	}
	// The table is named three ways, most specific first.
	//
	// A trailing PATH argument is the one a declared command reader can
	// use: `{artifact}` substitutes whole-element only (intrastate
	// 0025:C2), so the path cannot ride inside `--readme=<path>`, and
	// `--readme <path>` (space-separated) silently reads as the bare flag
	// because that flag takes an optional argument. A positional leaves
	// no way to get it wrong.
	path := readmePath
	if path == "" {
		path = f.readme.value
	}
	if path == "" {
		dir := *f.records
		if dir == "" {
			dir = "."
		}
		path = filepath.Join(dir, "README.md")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "stopped:unreadable (%s: %v)\n", path, err)
		return 2
	}
	rows := scan.ParseReadmeIndex(strings.Split(string(body), "\n"))
	if len(rows) == 0 {
		fmt.Fprintf(stderr, "stopped:no-index-table (%s has no `| NNNN |` index rows)\n", path)
		return 2
	}
	// Matched on the number, never the title, which drifts. A duplicate
	// row is reported rather than silently resolved: two rows for one
	// record is the `duplicate-row` drift `--readme` names, and a writer
	// told to edit "the" row would pick one arbitrarily.
	var found []scan.ReadmeRow
	for _, r := range rows {
		if r.Record == num {
			found = append(found, r)
		}
	}
	if len(found) > 1 {
		fmt.Fprintf(stderr, "stopped:duplicate-row (%s carries %d rows for %s)\n", path, len(found), num)
		return 2
	}
	// `--flat` is the same rendering `status --flat` gives, for the same
	// reason: a declared command reader takes a flat JSON object of
	// strings (RDR 0025), and the nested form below is unreadable to one.
	// A model binds this as the `readme` role's reader so a write to the
	// index row has a read-back.
	//
	// The key is `readme_status`, not `status`, because that is what the
	// fact is called everywhere else — `rdr-facts.toml` declares it, the
	// write table owns it, and a reader that answered `status` would put
	// the RECORD's key on the README role.
	if f.flat != nil && *f.flat {
		flat := map[string]string{}
		if len(found) == 1 {
			flat["readme_status"] = strings.TrimSpace(found[0].Status)
		} else {
			// Looked, and the table has no row for this record: `none` is
			// the declared member for exactly that, and it is what the
			// `readme --add` rows guard on. An omitted key would be
			// UNREADABLE to the accessor instead, which is a different
			// answer and a refusal.
			flat["readme_status"] = "none"
		}
		return emit(flat, stdout, stderr)
	}
	out := map[string]any{"schema": schemaVersion, "readme": path, "record": num, "row": nil}
	if len(found) == 1 {
		out["row"] = found[0]
	}
	return emit(out, stdout, stderr)
}

// openJoint is one open joint decision, from either of the two places a
// record states one: a `Joint-check: … (home: OPEN)` line in its body
// (Signal "joint-check", with the element id and line), or a Status line
// in joint-decision form (Signal "status", with the qualifier as written).
type openJoint struct {
	Record    string   `json:"record"`
	Signal    string   `json:"signal"`
	ID        string   `json:"id,omitempty"`
	Line      int      `json:"line,omitempty"`
	Targets   []string `json:"targets,omitempty"`
	Home      string   `json:"home,omitempty"`
	Qualifier string   `json:"qualifier,omitempty"`
	Decisions []string `json:"open_joint_decisions,omitempty"`
}

// openJointFacet answers 7.1's first question over the whole corpus —
// which members hold an open joint decision, and against what — in one
// call. Before this it was a per-member inspect plus a grep of the body
// for `home: OPEN`, and the grep once lost the only open line to `| head`.
// In flight by default; --all includes terminal records, whose open
// checks are a data error worth seeing, not a worklist item.
func openJointFacet(f *flags, all bool, stdout, stderr io.Writer) int {
	record, err := scopeRecord(f)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	docs, skipped, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	rows := []openJoint{}
	considered := 0
	for _, d := range docs {
		s := scan.Summarize(d)
		if !all && !s.InFlight {
			continue
		}
		considered++
		// `--record` keeps the record's own open decisions. `considered`
		// still counts the scan, which is what was paid.
		if record != "" && d.Record != record {
			continue
		}
		if s.Status != nil && (s.Status.Form == "joint-decision" || len(s.Status.OpenJointDecisions) > 0) {
			rows = append(rows, openJoint{Record: d.Record, Signal: "status",
				Qualifier: s.Status.Qualifier, Decisions: s.Status.OpenJointDecisions})
		}
		for _, e := range d.Elements {
			if e.Joint != nil && e.Joint.Open {
				rows = append(rows, openJoint{Record: d.Record, Signal: "joint-check", ID: e.ID,
					Line: e.LineStart, Targets: e.Joint.Targets, Home: e.Joint.Home})
			}
		}
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "record": record, "open": rows,
			"records_considered": considered, "skipped": skipped}, stdout, stderr)
	}
	for _, r := range rows {
		switch r.Signal {
		case "status":
			fmt.Fprintf(stdout, "%s status      [%s]\n", r.Record, r.Qualifier)
		default:
			fmt.Fprintf(stdout, "%s %-11s %5d  → %s (home: %s)\n", r.Record, r.ID[len(r.Record)+1:], r.Line,
				strings.Join(r.Targets, ", "), r.Home)
		}
	}
	fmt.Fprintf(stdout, "total %d open joint decisions over %d records%s\n", len(rows), considered, scopeNote(record))
	return 0
}

// literalFacet is the contract half of the joint-decision check: pairs of
// in-flight records whose contracts name the same backticked literal, the
// uncited pairs first. `--all` widens it to every record.
//
// It is a separate facet from `--anchor-intersect` rather than a widening
// of it because the two answer different questions and a reader acts on
// them differently. A shared ANCHOR is two records proposing to edit one
// symbol; a shared LITERAL is two records naming one error code, flag or
// sentinel in contract text that is otherwise unalike. Merging them would
// report a pair once with no way to say which coupling fired, and the
// stage's own vocabulary names the arms separately.
func literalFacet(f *flags, stdout, stderr io.Writer) int {
	record, err := scopeRecord(f)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	docs, _, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	overlaps := scopedOverlaps(scan.LiteralIntersect(docs, !*f.all), record)
	if *f.json {
		if overlaps == nil {
			overlaps = []scan.Overlap{}
		}
		return emit(map[string]any{"schema": schemaVersion, "open_only": !*f.all, "record": record, "overlaps": overlaps}, stdout, stderr)
	}
	fires := 0
	for _, o := range overlaps {
		mark := "cited"
		if !o.Cited {
			mark, fires = "UNCITED", fires+1
		}
		fmt.Fprintf(stdout, "%s %s %-7s %d shared: %s\n", o.A, o.B, mark, len(o.Anchors), strings.Join(o.Anchors, " "))
	}
	scope := "in-flight"
	if *f.all {
		scope = "all"
	}
	fmt.Fprintf(stdout, "total %d pairs sharing contract literals over %s records%s, %d with no cross-citation\n", len(overlaps), scope, scopeNote(record), fires)
	if fires > 0 {
		fmt.Fprintln(stdout, "an uncited pair names the same contract literal in neither record's citation: ask the joint-decision question before either locks")
	}
	return 0
}
