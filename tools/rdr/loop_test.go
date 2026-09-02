package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const loopModelName = "rdr-loop.toml"

// TestLoopModelResolvesTheCapCases records, as rows, the cases the prose
// ladders decided before they were deleted: three lens passes still
// raising net-new anchors stop as verdict-flapping; a small fix converges
// whatever the diff says; a cluster gate past its cap with entries open
// stops as cluster-flapping; and a first cluster pass runs its checks in
// full. `iter` is ITER_BUCKET — the pass the NEXT run would write — so
// `over` is the cell where a rerun is the fourth pass. Every tag is
// caller-bound, so the argv is built here as the skill builds it.
func TestLoopModelResolvesTheCapCases(t *testing.T) {
	bin := intrastateBinary(t)
	model := repoFile(t, filepath.Join("models", loopModelName))
	resolve := func(outcome string, tags ...string) (rule, next string) {
		t.Helper()
		args := []string{"flow", "resolve", "--model", model, "--outcome", outcome, "--plan-only", "--as", "json"}
		for i := 0; i+1 < len(tags); i += 2 {
			args = append(args, "--tag", tags[i]+"="+tags[i+1])
		}
		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("intrastate refused the %s resolve %v: %v\n%s", outcome, tags, err, out)
		}
		var env struct {
			Data struct {
				Rule string            `json:"rule"`
				Emit map[string]string `json:"emit"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &env); err != nil {
			t.Fatalf("%s: unreadable plan: %v\n%s", outcome, err, out)
		}
		return env.Data.Rule, env.Data.Emit["next"]
	}
	for _, c := range []struct {
		name, outcome string
		tags          []string
		rule, next    string
	}{
		{"cap 3, net-new: the plank problem", "lens-loop",
			[]string{"iter", "over", "found", "some", "net_new", "some", "fix", "substantial"},
			"loop-flapping-net-new", "stopped:verdict-flapping"},
		{"cap 3, substantial fix, nothing net-new: no fourth pass", "lens-loop",
			[]string{"iter", "over", "found", "some", "net_new", "none", "fix", "substantial"},
			"loop-flapping-substantial", "stopped:verdict-flapping"},
		{"small fix converges", "lens-loop",
			[]string{"iter", "2", "found", "some", "net_new", "some", "fix", "small"},
			"loop-small-fix", "converged"},
		{"substantial fix inside the cap reruns", "lens-loop",
			[]string{"iter", "3", "found", "some", "net_new", "none", "fix", "substantial"},
			"loop-rerun", "rerun"},
		{"nothing found converges", "lens-loop",
			[]string{"iter", "over", "found", "none", "net_new", "none", "fix", "substantial"},
			"loop-converged", "converged"},
		{"net-new with nothing found is a bad diff", "lens-loop",
			[]string{"iter", "2", "found", "none", "net_new", "some", "fix", "small"},
			"loop-diff-inconsistent", "stopped:ledger-diff-inconsistent"},
		{"N>3 with entries open", "cluster-cap",
			[]string{"iter", "over", "open", "some"},
			"cap-flapping", "stopped:cluster-flapping"},
		{"N=1 runs in full", "cluster-cap",
			[]string{"iter", "1", "open", "none"},
			"cap-first", "run"},
		{"N=2 delta-scopes", "cluster-cap",
			[]string{"iter", "2", "open", "some"},
			"cap-delta", "run"},
	} {
		if rule, next := resolve(c.outcome, c.tags...); rule != c.rule || next != c.next {
			t.Errorf("%s: rule %q next %q, want %s/%s", c.name, rule, next, c.rule, c.next)
		}
	}
}

// TestLoopEmitsAreDeclaredDispositions pins the loop table's answer
// surface the way the launch test pins its own: every row's `next` is a
// member of a declared partition, every declared member is emitted by
// some row, and every row carries the why the stop packet prints.
func TestLoopEmitsAreDeclaredDispositions(t *testing.T) {
	m := loadRoutingModelNamed(t, loopModelName)
	domain := emitNextDomain(t, loopModelName)
	member := map[string]string{}
	for part, vals := range domain {
		for _, v := range vals {
			member[v] = part
		}
	}
	emitted := map[string]bool{}
	for _, id := range m.RuleIDs {
		next := m.Emits[id]["next"]
		if _, ok := member[next]; !ok {
			t.Errorf("rule %q emits next %q, which no [emit.next.domain] partition declares", id, next)
		}
		emitted[next] = true
		if strings.TrimSpace(m.Emits[id]["why"]) == "" {
			t.Errorf("rule %q emits no why; the stop packet and the rerun line print it", id)
		}
	}
	for v := range member {
		if !emitted[v] {
			t.Errorf("[emit.next.domain] declares %q and no row emits it", v)
		}
	}
}
