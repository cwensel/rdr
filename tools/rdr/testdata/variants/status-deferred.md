# Recommendation 0111: Shared Decoder Instance per Process

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-11
- **Status**: Deferred [revisit when the upstream decoder exposes a public
  pool knob, or the project's stance on carrying a dependency fork
  changes]
- **Type**: Framework Workaround
- **Profile**: small — one contract, the decoder constructor
- **Priority**: High — the cost is paid on every parallel test run, so the
  High measures what the fix is worth, not what is scheduled. Moot while
  Deferred: no in-our-control path exists to realize it.
- **Related Issues**: tracker#517
- **Predecessors**: 0109-parallel-test-rollout

## Problem Statement

A contributor running the parallel test suite waits while every worker
rebuilds the same decoder from scratch, and the wait grows with each
worker added.

## Critical Assumptions

- **A1 [The rebuild is per-worker, not per-process]**
  - **Status**: **Verified**
  - **Method**: Spike
  - **Evidence**: A counter in the constructor increments once per worker.
  - **If wrong**: The cost is already amortized and this record is
    unnecessary.
- **A2 [The compile-once lever is unexported upstream]**
  - **Status**: **Verified**
  - **Method**: Source Search
  - **Evidence**: The runtime cache lives behind an unexported
    constructor with no exported knob.
  - **If wrong**: A supported path exists and this record un-parks
    immediately.
- **A3 [Every in-our-control path requires a dependency fork]**
  - **Status**: **Verified**
  - **Method**: Derivation + Prior Art
  - **Evidence**: Shown in Alternatives Considered — external fork,
    in-tree replace-target, and vendor-and-patch each carry a copy of
    someone else's decoder.
  - **If wrong**: A non-fork path exists and the deferral was wrong.

## Proposed Solution

### Approach

No approach is selected. Propose ran to completion and returned no
acceptable mechanism: the lever is unreachable without carrying a fork,
which the project rules out, and the one non-fork path is an upstream
change that is outside our control and delivers nothing near-term.

The record is parked, not abandoned. It re-enters at Propose when the
revisit condition on the Status line fires.

### Technical Design

#### Normative Contracts

```normative
Deferred: no contract is locked. This record locks nothing while parked.
```

#### Illustrative Code

None — no approach was selected.

### Decision Rationale

Deferring states the shape of the blockage so the next author does not
re-derive it. Abandoning would close the record, owe a post-mortem for an
implementation that never happened, and lose the trigger.

## Alternatives Considered

### Alternative 1: Vendor and patch the decoder

Works, and is a dependency fork by another name — the project carries the
patch forever.

### Briefly Rejected

- **Upstream patch**: The only non-fork path, but outside our control,
  human-brokered, and zero near-term speedup.
- **Per-worker pool**: Caches the wrong layer; the rebuild is inside the
  constructor.

## Context

### Background

The parallel rollout removed the serialization ceiling that had hidden
this cost.

### Technical Environment

The decoder package and its upstream dependency.

## Research Findings

### Investigation

Profiled a parallel run and read the upstream constructor.

### Key Discoveries

- **Verified** — the rebuild dominates worker startup.
- **Verified** — no exported seam reaches the cache.

## Trade-offs

### Consequences

- The cost is accepted as a known test-suite tax until upstream supplies
  a seam.

### Risks and Mitigations

- **Risk**: The trigger fires and nobody notices.
  **Mitigation**: The condition names an observable upstream event.

### Failure Modes

A parked record read as terminal would be closed and post-mortemed for an
implementation that never happened.

## Implementation Plan

### Prerequisites

- [ ] The revisit condition has fired

### Minimum Viable Validation

1. Run the parallel suite with a worker counter in the constructor.
2. The counter reads one per process, not one per worker.

### Phase 1: Code Implementation

#### Step 1: Deferred — no steps while parked

## Validation

### Testing Strategy

1. **Scenario**: Parallel run, counter instrumented.
   **Expected**: One construction per process.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- The upstream decoder constructor.
