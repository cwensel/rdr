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
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return nil, nil, code
	}
	scan.NewResolver(docs, *f.repo).ResolveAll(docs)
	return docs, skipped, 0
}

// indexGraph is the bare `rdr index`: the one graph document, or as text
// the record table an index README carries.
func indexGraph(f *flags, stdout, stderr io.Writer) int {
	docs, skipped, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	g := scan.BuildGraph(docs, skipped)
	if *f.json {
		return emit(g, stdout, stderr)
	}
	for _, r := range g.Records {
		fmt.Fprintf(stdout, "%s %-12s %-8s %s\n", r.Record, statusWord(r), r.Priority, r.ShortTitle)
	}
	fmt.Fprintf(stdout, "total %d records  %d elements  %d edges  %d targets with backlinks\n",
		len(g.Records), len(g.Elements), len(g.Edges), len(g.Backlinks))
	for _, p := range skipped {
		fmt.Fprintf(stdout, "skipped %s (not an RDR: no epoch fingerprint)\n", p)
	}
	return 0
}

func statusWord(r scan.Summary) string {
	if r.Status == nil {
		return "-"
	}
	return r.Status.Value
}

// statusFacet groups records by status; --in-flight keeps the worklist:
// Draft, and Final not yet Implemented. Terminal records are skipped,
// and so is Deferred, which is parked, not in flight.
func statusFacet(f *flags, inFlight bool, stdout, stderr io.Writer) int {
	docs, skipped, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	groups := map[string][]scan.Summary{}
	var rows []scan.Summary
	for _, d := range docs {
		s := scan.Summarize(d)
		if inFlight && !s.InFlight {
			continue
		}
		rows = append(rows, s)
		groups[statusWord(s)] = append(groups[statusWord(s)], s)
	}
	if *f.json {
		if rows == nil {
			rows = []scan.Summary{}
		}
		return emit(map[string]any{"schema": schemaVersion, "records": rows, "skipped": skipped}, stdout, stderr)
	}
	if inFlight {
		for _, s := range rows {
			fmt.Fprintf(stdout, "%s %-8s %s\n", filepath.Base(strings.TrimSuffix(s.Path, ".md")), s.Status.Value, qualifier(s))
		}
		fmt.Fprintf(stdout, "total %d in flight over %d records\n", len(rows), len(docs))
		return 0
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

// backlinksTo answers the impact question for one target — "who cites
// 0055:C4?" — or, given a bare record, for the record and every element
// in it. Mentions are included here and marked, because an impact
// analysis wants every reader, typed or not; the whole-table form stays
// typed-only so it remains readable.
func backlinksTo(docs []*scan.Document, target string, f *flags, stdout, stderr io.Writer) int {
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

// anchorFacet is the after-propose scan: pairs of in-flight records that
// cite the same code anchors, the uncited pairs first. `--all` widens it
// to every record.
func anchorFacet(f *flags, stdout, stderr io.Writer) int {
	docs, _, code := corpus(f, stderr)
	if code != 0 {
		return code
	}
	overlaps := scan.AnchorIntersect(docs, !*f.all)
	if *f.json {
		if overlaps == nil {
			overlaps = []scan.Overlap{}
		}
		return emit(map[string]any{"schema": schemaVersion, "open_only": !*f.all, "overlaps": overlaps}, stdout, stderr)
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
	fmt.Fprintf(stdout, "total %d overlapping pairs over %s records, %d with no cross-citation\n", len(overlaps), scope, fires)
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
		fmt.Fprintf(stderr, "stopped:no-index-table (%s has no `| [NNNN](…) |` rows)\n", path)
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
		return emit(map[string]any{"schema": schemaVersion, "open": rows,
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
	fmt.Fprintf(stdout, "total %d open joint decisions over %d records\n", len(rows), considered)
	return 0
}
