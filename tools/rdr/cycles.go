package main

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// ownershipKinds are the relations that must form a DAG: each says one
// record stands in for, follows, or has moved to another. A cycle in
// them — two records each claiming to override the other — names no
// authority at all.
var ownershipKinds = map[edge.Kind]bool{edge.Predecessor: true, edge.Overrides: true, edge.MovedTo: true}

// recordRef reads the record number out of an edge target or a home
// (`cli/0112`, `0110`, `cli/0143 §Normative Contracts R-2`).
var recordRef = regexp.MustCompile(`\b(\d{4})\b`)

func refRecord(s string) string {
	if m := recordRef.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// cycleFinding is one row of `index --cycles`.
type cycleFinding struct {
	Kind    string   `json:"kind"`
	Records []string `json:"records"`
	Detail  string   `json:"detail"`
	// Element and Line locate a per-record finding; empty on a cycle.
	Record  string `json:"record,omitempty"`
	Element string `json:"element,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// cyclesFacet reports the dependency shapes the flow cannot make progress
// through, over the whole corpus, and nothing from the relations that
// are symmetric by design (cluster, peer-evidence, mentions — a cycle
// there is the corpus working):
//
//	ownership-cycle     predecessor/overrides/moved-to edges that come back around
//	home-cycle          Joint-check homes on Draft records that defer to each other — a deadlock
//	home-ahead-of-lock  a Final record whose Joint-check home is a Draft record
//	open-at-lock        a Final record carrying an OPEN Joint-check
//
// The last two are advice about a lock that already happened; the first
// two are the blockers 7.1 has to clear before anything else.
func cyclesFacet(f *flags, stdout, stderr io.Writer) int {
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return code
	}
	status := map[string]string{}
	own := map[string]map[string]string{} // from -> to -> kind
	home := map[string]map[string]bool{}  // from -> home record
	rows := []cycleFinding{}
	for _, d := range docs {
		s := scan.Summarize(d)
		if s.Status != nil {
			status[d.Record] = s.Status.Value
		}
		for _, e := range d.Edges {
			if !ownershipKinds[e.Kind] {
				continue
			}
			if t := refRecord(e.To); t != "" && t != d.Record {
				if own[d.Record] == nil {
					own[d.Record] = map[string]string{}
				}
				own[d.Record][t] = string(e.Kind)
			}
		}
		for _, el := range d.Elements {
			j := el.Joint
			if j == nil || j.Verdict != "fired" {
				continue
			}
			// A home written `OPEN | cli/0143 §R-2` proposes 0143 as the
			// home; the record after the last bar is what a ruling would bind.
			parts := strings.Split(j.Home, "|")
			if t := refRecord(parts[len(parts)-1]); t != "" && t != d.Record {
				if home[d.Record] == nil {
					home[d.Record] = map[string]bool{}
				}
				home[d.Record][t] = true
			}
		}
	}
	// Per-record advice, before the cycles: the status map is complete now.
	for _, d := range docs {
		if status[d.Record] != "Final" {
			continue
		}
		for _, el := range d.Elements {
			j := el.Joint
			if j == nil || j.Verdict != "fired" {
				continue
			}
			if j.Open {
				rows = append(rows, cycleFinding{Kind: "open-at-lock", Records: []string{d.Record}, Record: d.Record,
					Element: el.ID, Line: el.LineStart, Detail: "Final record carries an OPEN Joint-check: " + strings.Join(j.Targets, ", ")})
			}
			parts := strings.Split(j.Home, "|")
			if t := refRecord(parts[len(parts)-1]); t != "" && t != d.Record && status[t] == "Draft" {
				rows = append(rows, cycleFinding{Kind: "home-ahead-of-lock", Records: []string{d.Record, t}, Record: d.Record,
					Element: el.ID, Line: el.LineStart, Detail: "Final record's Joint-check home is " + t + ", which is Draft: a ruling change there strands this lock"})
			}
		}
	}
	for _, comp := range scc(func(v string) []string {
		out := make([]string, 0, len(own[v]))
		for t := range own[v] {
			out = append(out, t)
		}
		return out
	}, keys(own)) {
		var arcs []string
		for _, a := range comp {
			for _, b := range comp {
				if k, ok := own[a][b]; ok {
					arcs = append(arcs, a+" "+k+" "+b)
				}
			}
		}
		sort.Strings(arcs)
		rows = append(rows, cycleFinding{Kind: "ownership-cycle", Records: comp, Detail: strings.Join(arcs, "; ")})
	}
	// A home on a Final record is a ruling that exists; only a home on a
	// Draft is a deferral, and only deferrals can wait on each other.
	deferral := func(v string) []string {
		var out []string
		for t := range home[v] {
			if status[t] == "Draft" {
				out = append(out, t)
			}
		}
		sort.Strings(out)
		return out
	}
	for _, comp := range scc(deferral, keys(home)) {
		var arcs []string
		for _, a := range comp {
			for _, b := range comp {
				if home[a][b] && status[b] == "Draft" {
					arcs = append(arcs, a+" → "+b)
				}
			}
		}
		sort.Strings(arcs)
		rows = append(rows, cycleFinding{Kind: "home-cycle", Records: comp, Detail: "Joint-check homes defer to each other: " + strings.Join(arcs, "; ")})
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "findings": rows, "records": len(docs), "skipped": skipped}, stdout, stderr)
	}
	for _, r := range rows {
		if r.Record != "" {
			fmt.Fprintf(stdout, "%-19s %s %-8s %5d  %s\n", r.Kind, r.Record, r.Element[len(r.Record)+1:], r.Line, r.Detail)
		} else {
			fmt.Fprintf(stdout, "%-19s %s  %s\n", r.Kind, strings.Join(r.Records, " "), r.Detail)
		}
	}
	fmt.Fprintf(stdout, "total %d findings over %d records\n", len(rows), len(docs))
	return 0
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// scc is Tarjan's algorithm, returning only the components of size > 1
// (a self-edge is excluded at the call sites), each sorted, in a stable
// order — the same corpus must print the same report.
func scc(next func(string) []string, nodes []string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	n := 0
	var visit func(string)
	visit = func(v string) {
		index[v], low[v] = n, n
		n++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range next(v) {
			if _, seen := index[w]; !seen {
				visit(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && index[w] < low[v] {
				low[v] = index[w]
			}
		}
		if low[v] == index[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			if len(comp) > 1 {
				sort.Strings(comp)
				out = append(out, comp)
			}
		}
	}
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			visit(v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}
