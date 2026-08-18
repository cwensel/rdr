---
name: rdr-cluster-reconcile
metadata:
  argument-hint: <cluster-name | NNNN NNNN [NNNN…]> [--commit | --no-commit]
description: 'Use to reconcile drift across related Final, unimplemented RDRs before any implement. Runs Stage 7.1 whole-set critique and pairwise contradiction scan. Trigger for cluster reconcile, $rdr-cluster-reconcile, or /rdr-cluster-reconcile.'
---

# rdr-cluster-reconcile — Stage 7.1 (Cluster Reconcile)

Per-RDR stages check one RDR at a time; cross-RDR consistency is a *set*-level
concern no per-RDR gate covers. This stage reconciles a cluster of related RDRs —
all Final-and-unimplemented — **once**, before any of them implements. **Per
cluster, not per RDR.** Most RDRs skip it (it applies to foundational / cross-RDR
producers).

## Usage

```
Codex: $rdr-cluster-reconcile <cluster-name | NNNN NNNN [NNNN ...]>
Claude: /rdr-cluster-reconcile <cluster-name | NNNN NNNN [NNNN ...]>
```

Takes a cluster (a name you give the set, or the list of RDR numbers in it) — **not
a single RDR**. If only one RDR is Final-and-unimplemented, there is no cluster;
say so and point at `/rdr-implement NNNN`.

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** to bind `$RDR_ENV`,
   `$RDR_RESOURCES`, and `$RDR_RECORDS` (the consumer's RDR directory, exported by the
   marker) plus the output base `<EVIDENCE_DIR>/cluster-reconcile/<cluster>/`.
2. **Build the cluster**: list the Final-and-unimplemented RDRs under `{RDR_RECORDS}`;
   form the peer pairs to compare.
3. **Detect the iteration**: N = 1 + the highest existing `reconcile-report.md`
   under the output base (loose = iter-1, `iter-N/` subdirs after); write this
   run's outputs to `iter-N/` when N>1, and record the current revision of each
   member AND each joint-decision home for the next iteration. At N>1 the run is
   anchored to its own history:
   - **Origin ledger** — iter-1's findings table is the ledger; each finding
     traces to an open entry. One tracing to none is **net-new scope**: recorded
     with evidence, demoting no member on its own.
   - **Delta-scope** — re-scan only pairs with ≥1 member changed since the prior
     report's revisions; unchanged pairs **carry** the prior verdict, naming its
     iteration. The whole-set critique re-runs only if ≥1 member changed. The
     report's "Inputs run" table marks every check RE-SCANNED or CARRIED, so a
     skip never reads as a pass.
   - **Cap = 2 re-runs** — N=3 is the last that may demote. At N>3 with findings
     open, demote nothing and run no further gate pass: emit
     `stopped:cluster-flapping:<cluster>:<the open entries>` (§stop-packet)
     carrying those entries and the decomposition question ("is the cluster cut
     right at all?") with the gate's accumulated evidence, then §strong-consult,
     then the author. An answered tolerance's scoped answer-vs-fences check
     still runs.
4. **Dispatch into the gate prompts** for whatever step 3 left in scope (Stage
   `07.1-cluster-reconcile.md` owns the mechanics — read it for the
   cluster-membership and re-entry-scope calls):
   - **Whole-set critique** — [`2-critique.md`](2-critique.md)
     run against the set; writes `critique-set.md`.
   - **Pairwise contradiction scan** — [`pairwise.md`](pairwise.md)
     run per pair; writes `pairwise-<A>-<B>.md`.
   Delegate the heavy reads (several RDRs, the pairwise runs) to a sub-agent that
   returns verdict + evidence pointer (§return-packet).

## Review gate (Stage `07.1-cluster-reconcile.md`)

- Cluster membership is right — over-broad wastes pairwise runs; a missed peer is
  the drift this stage exists to catch.
- A cross-RDR defect routes a peer **Final → Draft** at the right re-entry scope
  (re-lock-only / stage-scoped / full-flow), sized to the defect — the stage doc owns
  this call.
- At N>1: every check is a RE-SCANNED or CARRIED row, the re-scans are exactly the
  pairs whose members moved, and each demotion traces to an open ledger entry —
  a demotion on an unledgered finding means the gate is grading its own last repair.
  Past the cap (N>3, findings open) the run stops rather than demoting again.
- A **JOINT-DECISION** finding (a shared decision neither RDR solely owns) demotes
  no one: hoist it to a named home, record the tolerance, siblings stay Final —
  and it must be genuinely joint, not a dodged single-RDR defect. The tolerance
  names the open QUESTION, not just the home: it is an open obligation, not a
  coherence claim. Diff each home's recorded revision: if one moved it has
  answered something, and every sibling holding a tolerance against that
  question owes its scoped answer-vs-fences check here — even one whose own text
  never moved. The home is one paragraph per decision (decision + user-visible
  stake; derivation goes to the evidence tree) — an oversized home is just
  another peer — and a sibling restating its
  mechanism prose is itself a finding, repaired to a citation.
- A shared census/figure/enumeration has **one artifact of record**, named at
  the home and cited by every peer. A peer figure disagreeing with it is a
  CITATION REPAIR: repair the citation, no demotion, no tolerance, no deviation
  entry. It outranks DEFER-TO-IMPLEMENTATION, which covers a shared figure only
  while no artifact of record exists.
- A **DEFER-TO-IMPLEMENTATION** finding (real, but too cheap to demote for)
  demotes no one either. It qualifies only if all three hold: the text is
  outside every ```normative fence (or is a figure no contract's pass condition
  reads); a mechanical check at implementation decides it (grep, compile, or a
  test the RDRs already specify); and no clause's meaning changes. Write it into
  that RDR's `<art>/deviations.md` under an existing Type (usually TEST-FIXTURE)
  with the named check, left open for Stage 8 to discharge by running it. Mark
  the report row `DEFERRED` and leave the RDR Final; it does not block
  RECONCILED. Fenced normative text never defers,
  nor does a clash between two clauses' meaning — both are SPEC-DEFECT. The
  deferral closes its ledger entry, so it cannot drive the cap; the report's
  DEFERRED rows carry forward, and re-finding one is a SPEC-DEFECT.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for the cluster-reconcile *evidence* (`chore(rdr): cli/NNNN cluster-reconcile <cluster>`); the doc/lock commit lands at finalize, not here.
- Reconciled, no demotion → `Next: /rdr-implement NNNN` for each cluster member;
  list any deferred items and their checks in the close packet's `Deviations:`
  field (the entries are already in each RDR's `<art>/deviations.md`).
- Reconciled with tolerances → same `Next: /rdr-implement NNNN` (no demotion, no
  re-walk) *while the tolerance is unanswered*; list each standing tolerance
  (pair, joint decision, home, open question) and whether the home has answered
  it in the close packet's `Deviations:` field. An ANSWERED tolerance runs its
  scoped answer-vs-fences check before that sibling implements.
- A peer demoted to Draft → re-enter it at the scoped stage (`/rdr-propose` /
  `/rdr-refine` / `/rdr-resolve` per the scope), re-lock via `/rdr-finalize NNNN`,
  then implement.
- `stopped:cluster-flapping` → no `Next:`; the cluster waits on §strong-consult
  over the open entries, then the author's decomposition call.
- `/rdr-status NNNN` to re-orient any member.
