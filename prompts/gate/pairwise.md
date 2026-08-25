# Pairwise — Cross-RDR Contradiction Scan (S-07)

**Use when**: the **cluster** gate, not single-RDR pre-lock. In the flow it runs
at [Stage 7.1 Cluster-reconcile]($RDR_HOME/stages/07.1-cluster-reconcile.md), over a
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

SELECT THE PAIRS by query, not by reading the cluster. If
`[ -x "$RDR_HOME/bin/rdr" ]`:

```sh
"$RDR_HOME/bin/rdr" index --json --anchor-intersect --records "$RDR_RECORDS" --repo "$RDR_REPO"
"$RDR_HOME/bin/rdr" index --json --backlinks=<NNNN> --records "$RDR_RECORDS"
```

- `overlaps[]` `{records[], anchors[], cited}` — two in-flight records citing
  the same `path::Symbol`. **`cited: false` ranks first**: they propose to change
  one symbol with no edge of any kind between them, the pair this scan exists
  for. `cited: true` is still a pair, just already related.
- `backlinks[]` from `--backlinks=<NNNN>` — who cites this record or anything in
  it, typed edges and mentions kept apart. `kind=="peer-evidence"` is the pair
  already related by cited evidence (the *Critical Assumptions* Method: Peer RDR
  case); `cross-cutting-owner` is the shared-concern case. Each carries `record`
  (the citing peer), `from`, `to` and `line`/`line_end`.

Absent binary → the prose rule: pairs explicitly listed in a *Critical
Assumptions* (Method: Peer RDR) or *Cross-Cutting Concerns* entry.

SCOPE THE READ. A pair already related by `peer-evidence` need not be handed
both whole records: pass those backlinks' `line`/`line_end` ranges, plus each
record's contract and scenario ranges (`inspect --json`, `elements[]`
`kind=="C"` / `kind=="S"`, `line_start`/`line_end`) — read them with `sed -n`.
An uncited `--anchor-intersect` pair has no such spine: read both records.

```text
Here are two RDRs, {RDR_A_PATH} and {RDR_B_PATH}. Report every place they
contradict, duplicate, or leave a gap (an interaction one mentions that the
other does not) that you can anchor to two direct quotes — one per RDR, or
one plus an explicit 'silent'. No minimum, no maximum: zero anchored
findings is a valid report.

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
  OWNERSHIP: [single-RDR | joint] — single-RDR when the wrongness is internal
    to one RDR; joint when the finding is a shared decision neither RDR
    solely owns
  BLOCKS: <the decision this blocks or the test it prevents>

Do not paraphrase quotes — use exact wording. If you cannot find a direct
quote, say "NO DIRECT QUOTE" and explain why you still believe the conflict
exists. When the quote is an addressable element, `rdr inspect --select <id>
<NNNN>` prints exactly that element's bytes — quote from it, not from memory.
```

**Dual-model recommended** for a foundational cluster — disagreement between
models on "do these two contradict?" is itself a signal.

## Expected signal

- **Healthy** — every finding anchored to direct quotes; `blocks-impl`
  findings name the exact clauses. Zero findings on a pair whose members have
  not changed since the last scan is the expected result, not a malfunction.
- **Unhealthy** — findings with no quote, paraphrase in place of quotation, or
  "NO DIRECT QUOTE" used as a routine escape.

## What a finding does

A `blocks-impl` or `risks-impl` finding with OWNERSHIP single-RDR is a
**SPEC-DEFECT** against the *less foundational* RDR of the pair — it does not
get edited in place. The cluster gate drops that RDR from Final back to Draft
and re-enters the flow. A `joint` finding instead takes the gate's
**JOINT-DECISION** disposition — mark, hoist the shared decision to a single
normative home, siblings proceed under recorded tolerance, no demotion. See
[Stage 7.1]($RDR_HOME/stages/07.1-cluster-reconcile.md) for the disposition
rules and the literature behind tolerating cross-RDR inconsistency: this gate
is the chosen checkpoint where drift is marked and homed, not necessarily
eliminated.

## Source

Liu et al., *Cross-Spec Inconsistency Detection*, QRS 2025 —
<https://doi.org/10.1109/QRS65678.2025.00014>; Finkelstein/Nuseibeh viewpoints
tradition (cross-view consistency checked at chosen stages, not enforced as a
precondition).
