# Pairwise — Cross-RDR Contradiction Scan (S-07)

**Use when**: the **cluster** gate, not single-RDR pre-lock. In the flow it runs
at [Stage 7.1 Cluster-reconcile]($RDR_FLOW_HOME/flow/07.1-cluster-reconcile.md), over a
cluster of related RDRs that are all Final-and-unimplemented — *not* during a
single RDR's Stage 5. A per-RDR pass can only compare an RDR to peers that
already exist; running it at the cluster level is what catches drift a *later*
lock introduces into an *earlier* peer.

**What it uniquely catches**: cross-RDR contract leaks — places where two RDRs
disagree, duplicate, or leave a gap between them — and **cross-RDR round-trip
defects**: a pair of operations meant to compose to identity (A emits, B
re-pairs) that loses fidelity or fails idempotence in the seam, which no
single-RDR lens reads both sides of.

**Cost**: 10 min per pair.

## Prompt

Run for every plausibly-interacting pair in the cluster; trim to peer pairs
explicitly listed in a *Critical Assumptions* (Method: Peer RDR) or
*Cross-Cutting Concerns* entry.

```text
Here are two RDRs, {RDR_A_PATH} and {RDR_B_PATH}. Find every place they
could contradict each other, duplicate each other, or leave a gap between
them (an interaction one mentions that the other does not describe).

ALSO check round-trip / inverse invariants. If the two RDRs describe a pair of
operations expected to compose to identity (A emits, B consumes; encode/decode,
serialize/parse, import/export, snapshot/drift, migrate/rollback), assert the
invariant holds ACROSS THE SEAM: does B re-pair / reconstruct exactly what A
emits, byte- or value-for-byte, on every input class A produces? A defect here
hides between the two RDRs and no single-RDR lens can see it. Treat a
non-idempotent or fidelity-losing round-trip as a `contradiction` finding even
when neither RDR's prose is individually wrong.

For each finding, emit:

  TYPE: [contradiction | duplication | gap | round-trip]
  A-QUOTE: "<direct quote from RDR A>"
  B-QUOTE: "<direct quote from RDR B, or 'silent'>"
  EXPLANATION: one sentence (for round-trip: name the input class that breaks
    identity, and whether it is a fidelity loss or an outright failure)
  SEVERITY: [blocks-impl | risks-impl | cosmetic]

Do not paraphrase quotes — use exact wording. If you cannot find a direct
quote, say "NO DIRECT QUOTE" and explain why you still believe the conflict
exists.
```

**Dual-model recommended** for a foundational cluster — disagreement between
models on "do these two contradict?" is itself a signal.

## Expected signal

- **Healthy** — at least one gap per plausibly-interacting pair, anchored to
  direct quotes; `blocks-impl` findings name the exact clauses.
- **Unhealthy** — "no issues found" across every pair (over-agreement). Switch
  model and rerun.

## What a finding does

A `blocks-impl` or `risks-impl` cross-RDR contradiction is a **SPEC-DEFECT**
against the *less foundational* RDR of the pair — it does not get edited in
place. The cluster gate drops that RDR from Final back to Draft and re-enters
the flow; see [Stage 7.1]($RDR_FLOW_HOME/flow/07.1-cluster-reconcile.md) for the
disposition rules and the literature behind tolerating cross-RDR inconsistency
until this gate.

## Source

Liu et al., *Cross-Spec Inconsistency Detection*, QRS 2025 —
<https://doi.org/10.1109/QRS65678.2025.00014>; Finkelstein/Nuseibeh viewpoints
tradition (cross-view consistency checked at chosen stages, not enforced as a
precondition).
