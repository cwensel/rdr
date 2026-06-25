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
3. **Dispatch into the gate prompts** (Stage `07.1-cluster-reconcile.md` owns the
   mechanics — read it for the cluster-membership and re-entry-scope calls):
   - **Whole-set critique** — [`2-critique.md`](2-critique.md)
     run against the set; writes `critique-set.md`.
   - **Pairwise contradiction scan** — [`pairwise.md`](pairwise.md)
     run per pair; writes `pairwise-<A>-<B>.md`.
   Delegate the heavy reads (several RDRs, the pairwise runs) to a sub-agent that
   returns verdict + evidence pointer.

## Review gate (Stage `07.1-cluster-reconcile.md`)

- Cluster membership is right — over-broad wastes pairwise runs; a missed peer is
  the drift this stage exists to catch.
- A cross-RDR defect routes a peer **Final → Draft** at the right re-entry scope
  (re-lock-only / stage-scoped / full-flow), sized to the defect — the stage doc owns
  this call.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for the cluster-reconcile *evidence* (`chore(rdr): cli/NNNN cluster-reconcile <cluster>`); the doc/lock commit lands at finalize, not here.
- Reconciled, no demotion → `Next: /rdr-implement NNNN` for each cluster member.
- A peer demoted to Draft → re-enter it at the scoped stage (`/rdr-propose` /
  `/rdr-refine` / `/rdr-resolve` per the scope), re-lock via `/rdr-finalize NNNN`,
  then implement.
- `/rdr-status NNNN` to re-orient any member.
