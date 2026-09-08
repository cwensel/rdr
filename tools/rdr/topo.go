package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// topoRow is one record in build order, with the set members it follows.
type topoRow struct {
	Record   string   `json:"record"`
	Title    string   `json:"title"`
	Status   string   `json:"status,omitempty"`
	Priority string   `json:"priority,omitempty"`
	After    []string `json:"after"`
}

// topoExternal is a predecessor edge leaving the set: the record still
// depends on it, but the order here cannot place it.
type topoExternal struct {
	Record      string `json:"record"`
	Predecessor string `json:"predecessor"`
}

// priorityRank orders the Priority vocabulary for the tiebreak: High
// before Medium before Low before unset. The value is read by its leading
// word, as every other qualified field is.
func priorityRank(p string) int {
	switch strings.ToLower(leadingWord(p)) {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	}
	return 3
}

// topoFacet is the build order joint-propose used to derive by hand: a
// topological sort of the set over `predecessor` edges (Kahn), ties
// broken by Priority and then by number. The set is `--topo=NNNN,…`, or
// every in-flight record when bare. Records in a predecessor cycle are
// reported under `cycles`, unordered, and the rest are ordered without
// them; a predecessor outside the set is `external`. A name the corpus
// does not hold is a `skipped` row, never a silent omission.
func topoFacet(f *flags, stdout, stderr io.Writer) int {
	docs, _, code := records(f, stderr)
	if code != 0 {
		return code
	}
	byRecord := map[string]*scan.Document{}
	for _, d := range docs {
		byRecord[d.Record] = d
	}
	set := map[string]bool{}
	skipped := []string{}
	if f.topo.value == "" {
		for _, d := range docs {
			if scan.Summarize(d).InFlight {
				set[d.Record] = true
			}
		}
	} else {
		for _, part := range strings.Split(f.topo.value, ",") {
			part = strings.TrimSpace(part)
			if n := shortRecordNumber(part); n != "" {
				part = n
			}
			num := ident.RecordOf(part)
			if num == "" || byRecord[num] == nil {
				skipped = append(skipped, part)
				continue
			}
			set[num] = true
		}
	}
	kinds := map[edge.Kind]bool{}
	for _, k := range strings.Split(*f.edges, ",") {
		switch strings.TrimSpace(k) {
		case "predecessors", "predecessor":
			kinds[edge.Predecessor] = true
		case "overrides":
			// An Overrides entry re-cuts a contract the other record
			// shipped, so the overridden record builds first — the same
			// direction as a predecessor edge, read from a different field.
			kinds[edge.Overrides] = true
		default:
			fmt.Fprintf(stderr, "stopped:unknown-edge-kind — --edges takes predecessors and/or overrides, got %q\n", k)
			return 2
		}
	}
	after := map[string][]string{} // record -> records it builds after, in the set
	external := []topoExternal{}
	for rec := range set {
		seen := map[string]bool{}
		for _, e := range byRecord[rec].Edges {
			if !kinds[e.Kind] {
				continue
			}
			t := refRecord(e.To)
			if t == "" || t == rec || seen[t] {
				continue
			}
			seen[t] = true
			if set[t] {
				after[rec] = append(after[rec], t)
			} else {
				external = append(external, topoExternal{rec, t})
			}
		}
		sort.Strings(after[rec])
	}
	sort.Slice(external, func(i, j int) bool {
		if external[i].Record != external[j].Record {
			return external[i].Record < external[j].Record
		}
		return external[i].Predecessor < external[j].Predecessor
	})
	cycles := scc(func(v string) []string { return after[v] }, keys(set))
	inCycle := map[string]bool{}
	for _, comp := range cycles {
		for _, r := range comp {
			inCycle[r] = true
		}
	}
	// Kahn over the acyclic remainder. A ready set ordered by Priority
	// then number is the tiebreak, applied at every pop rather than once.
	indeg := map[string]int{}
	dependents := map[string][]string{}
	for rec := range set {
		if inCycle[rec] {
			continue
		}
		for _, p := range after[rec] {
			if inCycle[p] {
				continue
			}
			indeg[rec]++
			dependents[p] = append(dependents[p], rec)
		}
	}
	less := func(a, b string) bool {
		ra, rb := priorityRank(scan.Summarize(byRecord[a]).Priority), priorityRank(scan.Summarize(byRecord[b]).Priority)
		if ra != rb {
			return ra < rb
		}
		return a < b
	}
	var ready []string
	for rec := range set {
		if !inCycle[rec] && indeg[rec] == 0 {
			ready = append(ready, rec)
		}
	}
	order := []topoRow{}
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return less(ready[i], ready[j]) })
		rec := ready[0]
		ready = ready[1:]
		s := scan.Summarize(byRecord[rec])
		row := topoRow{Record: rec, Title: s.Title, Priority: leadingWord(s.Priority), After: after[rec]}
		if row.After == nil {
			row.After = []string{}
		}
		if s.Status != nil {
			row.Status = s.Status.Value
		}
		order = append(order, row)
		for _, d := range dependents[rec] {
			indeg[d]--
			if indeg[d] == 0 {
				ready = append(ready, d)
			}
		}
	}
	if cycles == nil {
		cycles = [][]string{}
	}
	if *f.json {
		return emit(map[string]any{"schema": schemaVersion, "order": order, "cycles": cycles,
			"external": external, "skipped": skipped}, stdout, stderr)
	}
	for i, r := range order {
		fmt.Fprintf(stdout, "%2d %s %-8s %-7s after %-20s %s\n", i+1, r.Record, r.Status, r.Priority, strings.Join(r.After, ","), r.Title)
	}
	for _, c := range cycles {
		fmt.Fprintf(stdout, "cycle %s\n", strings.Join(c, " "))
	}
	for _, e := range external {
		fmt.Fprintf(stdout, "external %s -> %s\n", e.Record, e.Predecessor)
	}
	for _, s := range skipped {
		fmt.Fprintf(stdout, "skipped %s (not in the records dir)\n", s)
	}
	fmt.Fprintf(stdout, "total %d ordered, %d in cycles, %d external predecessors\n", len(order), len(inCycle), len(external))
	return 0
}
