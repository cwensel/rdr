package main

// `rdr env` publishes the bound seam, so a skill stops carrying a shell
// resolver to read the three vars this tool never opens.
//
// The verb is deliberately NOT `envOrSeam`. Every other seam read in this
// binary lets an exported variable win, because `--records` and `--repo`
// name one directory for one invocation and a caller who exports one means
// it. Publishing the whole seam is a different act: an inherited RDR_* is
// far more often a leak from the previous call — in a workspace where a
// repo-local marker sits beside a shared one, the two seams are disjoint,
// and the calls that cross them are the ones nobody notices. So this
// answers from the MARKER, and an ambient value is overwritten rather than
// echoed back.
//
// That is the property §seam-bind's shell block had for free: it sourced
// the marker, and sourcing overwrites. A verb that preferred the
// environment would re-emit a stale value, the caller would `eval` it, and
// the leak would become load-bearing instead of dying with the turn.
//
// The failure that motivates the care is silent. Read a record from one
// project's RDR_RECORDS while RDR_EVIDENCE still resolves from another and
// every lens fact reads false — a record with four finished lenses reports
// as never lensed, and the router sends the flow back to re-run them over
// the evidence that is already there.
//
//	0  a marker bound; its vars on stdout
//	1  no marker, or not in a git project — the caller stops exactly as
//	   the shell block did, with the same reason
//
// It reads no record and opens no template, so it is schema-bind exempt
// (main.go): a skill runs this BEFORE it can know where TEMPLATE.md is.

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// envMarkerVars name where the seam came from. They are not marker
// exports — the marker cannot state its own path — but a caller needs
// them: $RDR_MARKER is what §seam-bind records, and $RDR_PROJECT is what
// lets a caller assert the seam it just bound belongs to the repo it is
// standing in. Without that comparison a foreign seam binds silently,
// which is the whole hazard this verb has to avoid re-introducing.
const (
	envMarkerVar  = "RDR_MARKER"
	envProjectVar = "RDR_PROJECT"
)

// envCmd emits the seam. Text is `k='v'` per line, sorted, single-quoted
// so `eval` survives a path with a space and cannot execute what a marker
// happened to contain.
func envCmd(f *flags, stdout, stderr io.Writer) int {
	marker, project, ws, _ := findMarker()
	if project == "" {
		fmt.Fprintln(stderr, "stopped:not-in-a-project (run /rdr-* from inside the consumer repo)")
		return 1
	}
	if marker == "" {
		// The wording and both directories are §seam-bind's, unchanged: a
		// caller who has seen this message before must not have to learn
		// it again because the mechanism moved into the binary.
		fmt.Fprintf(stderr, "stopped:no-marker — run /rdr-init in this repo (looked in %s/.rdr and %s)\n", project, ws)
		return 1
	}

	// A marker that REFUSED is not an unconfigured repo, and publishing an
	// empty seam would make it look like one: the caller's own
	// `RDR_PROJECT` assertion still passes (this binary computes that var,
	// the marker does not), so it would proceed with every contract var
	// unset. That is the silent foreign bind the anchor guard exists to
	// stop, arriving one layer later. The marker's message is carried
	// verbatim — it names the project it describes and the one it was
	// handed, which is what a caller needs and this binary cannot restate.
	if why := markerRefusal(); why != "" {
		fmt.Fprintf(stderr, "%s\n", why)
		fmt.Fprintf(stderr, "stopped:marker-refused %s\n", marker)
		return 1
	}

	bound := map[string]string{}
	for _, v := range seamVars {
		// seamValue, never envOrSeam: the marker is the authority here.
		if val := strings.TrimSpace(seamValue(v)); val != "" {
			bound[v] = val
		}
	}
	bound[envMarkerVar] = marker
	bound[envProjectVar] = project

	if f != nil && f.json != nil && *f.json {
		out, err := json.MarshalIndent(bound, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "stopped:unprojectable (%v)\n", err)
			return 2
		}
		fmt.Fprintln(stdout, string(out))
		return 0
	}

	keys := make([]string, 0, len(bound))
	for k := range bound {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(stdout, "%s=%s\n", k, shellQuote(bound[k]))
	}
	return 0
}

// shellQuote renders a value as one single-quoted shell word. A marker
// holds paths someone typed, so a space is ordinary and a quote is not
// impossible; `eval` is only safe if neither can end the word. The
// embedded-quote escape is the POSIX one: close, escape, reopen.
func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}
