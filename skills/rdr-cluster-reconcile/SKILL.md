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

1. Read [`rdr-common.md`](rdr-common.md) **whole, with the Read tool** (it exceeds the 30KB
   Bash cap — `cat` truncates and costs a retry; never `sed`/`grep` §-slices); run **§seam-bind** to bind `$RDR_ENV`,
   `$RDR_RESOURCES`, and `$RDR_RECORDS` (the consumer's RDR directory, exported by the
   marker). The output base comes from
   `eval "$("$RDR_HOME/bin/rdr" paths --cluster <key> --next-iter <NNNN>)"` →
   `$EVIDENCE_DIR` and `$ITER_DIR` (§evidence). `<key>` is the resolved members'
   record numbers joined in ascending order — `0117-0118`,
   `0122-0123-0130-0131-0132`. The key is the membership, and `rdr status` reads
   it to answer 7.1; a topical name (`dml-purpose`) is a pre-2026-06-29 shape and
   reads as never reconciled. Use the members the stage RESOLVED, not the ones
   proposed: a dropped candidate is not in the key.
2. **Build the cluster**: `"$RDR_HOME/bin/rdr" index --json --cluster-of NNNN --closure --final-unimplemented`
   is the membership — `cluster[]` is the set (relation and `via` per member),
   `out_of_scope[]` the record of what was dropped and why; a `candidate` is
   confirmed (re-run with it as a second seed) or dismissed, never expanded
   (Stage `07.1-cluster-reconcile.md` step 1). Then form the peer pairs — **not all
   C(n,2)**: trim to the plausibly-interacting ones (that stage's step 3 owns the
   rule). The trim is the cost control; report scanned/possible.
3. **Run the stage prompt** — [`07.1-cluster-reconcile.prompt.md`](07.1-cluster-reconcile.prompt.md);
   it owns the iteration contract and the four dispositions, so read it rather
   than re-deriving either. Bind the contract first — it scopes step 4 — and
   write this run's outputs to step 1's `$ITER_DIR`.
4. **Dispatch into the gate prompts** for whatever the iteration contract left in
   scope (Stage `07.1-cluster-reconcile.md` owns the cluster-membership and
   re-entry-scope calls — read it for those):
   - **Whole-set critique** — [`2-critique.md`](2-critique.md)
     run against the set; writes `critique-set.md`.
   - **Pairwise contradiction scan** — [`pairwise.md`](pairwise.md)
     run per pair; writes `pairwise-<A>-<B>.md`.
   Delegate the heavy reads (several RDRs, the pairwise runs) to a sub-agent that
   returns verdict + evidence pointer (§return-packet). Pairs are independent —
   spawn them concurrently where the harness allows, each with the pairwise
   prompt's **SCOPE THE READ** ranges: a spawn handed two whole records reads
   thousands of lines for a spine of a few hundred.
5. **Disposition every finding** by running the stage prompt's disposition half
   (step 3's prompt) — NO CONFLICT / JOINT-DECISION / SPEC-DEFECT /
   DEFER-TO-IMPLEMENTATION, one per finding.

## Review gate (Stage `07.1-cluster-reconcile.md`)

- Cluster membership is right — over-broad wastes pairwise runs; a missed peer is
  the drift this stage exists to catch — and the scanned pairs are the trimmed
  set, reported as scanned/possible: an untrimmed C(n,2) sweep is the same waste
  the over-broad cluster is.
- A cross-RDR defect routes a peer **Final → Draft** at the right re-entry scope
  (re-lock-only / stage-scoped / full-flow), sized to the defect — the stage doc owns
  this call.
- At N>1: every check is a RE-SCANNED or CARRIED row, the re-scans are exactly the
  pairs the `revisions.txt` diff moved, and each demotion traces to an open ledger
  entry — a demotion on an unledgered finding means the gate is grading its own
  last repair. The cap is the `cluster-cap` row resolved at entry
  (`models/rdr-loop.toml`): `stopped:cluster-flapping` stops rather than demoting.
- Each **JOINT-DECISION** is genuinely joint (not a dodged single-RDR defect),
  and its tolerance names the open QUESTION as well as the home. Every home's
  recorded revision was diffed; each answer that surfaced has its siblings'
  scoped answer-vs-fences checks shown here — including a sibling whose own text
  never moved.
- Each home is a paragraph per decision, and no sibling restates its mechanism
  prose; each shared census/figure/enumeration has one artifact of record, with
  a disagreeing peer figure typed CITATION REPAIR rather than joint decision,
  demotion, or deferral.
- Each **DEFER-TO-IMPLEMENTATION** clears all three conditions, its entry is
  actually written to that RDR's `<art>/deviations.md` with its named check, and
  its report row reads `DEFERRED` with the RDR left Final. Nothing fenced or
  meaning-changing was deferred, and no prior iteration's `DEFERRED` row was
  deferred a second time.

The prompt (step 3) defines each disposition and its conditions; these are the
checks, not a second definition.

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
