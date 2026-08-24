# Recommendation 0102: Cache Entry Expiry Clock

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-02
- **Status**: Draft
- **Type**: Bug Fix
- **Profile**: standard — one contract, the clock source
- **Priority**: High
- **Related Issues**: tracker#420
- **Seam Lineage**: `cache::Expire` — 1st point-fix; trail: tracker#420.

## Problem Statement

Entries expire against wall-clock time on one host and monotonic time on
another, so a clock step evicts or retains the wrong entries.

## Critical Assumptions

- **A1 [Every expiry comparison goes through one clock accessor]**
  - **Status:** Verified
  - **Method**: Source Search (the three call sites + the test double)
  - **Evidence — the two accessor paths**: `clock::Now` is the only reader
    of time in the cache package; the test double implements the same
    interface.
  - **If wrong (a refusal is owed)**: A second time source survives the
    change and the bug persists on one path.
  - **Note**: The test double was found by its interface, not by name.
- **A2 [A monotonic clock is available on every supported host]**
  - **Status**: Pending
  - **Method**: Docs Only + Spike
  - **Evidence (plan)**: Read the runtime's clock documentation for each
    host; spike the one that documents no guarantee.
  - **If wrong**: The fallback is wall-clock with a step guard, which is
    the second alternative.

## Proposed Solution

### Approach

Route every expiry comparison through a monotonic clock.

### Technical Design

#### Normative Contracts

```normative
func Now() Instant
```

### Decision Rationale

One clock, one answer.

## Alternatives Considered

### Briefly Rejected

- **Step detection on the wall clock**: Detects only steps larger than
  the polling interval.

## Context

### Background

Two hosts disagreed on which entries were live after a time step.

### Technical Environment

The cache package and the runtime clock.

## Research Findings

### Investigation

Traced every read of time in the package.

### Key Discoveries

- **Verified** — one accessor, three call sites.

## Trade-offs

### Consequences

- Expiry no longer moves with the wall clock.

### Risks and Mitigations

- **Risk**: A host without a monotonic clock.
  **Mitigation**: A2's spike decides the fallback before lock.

### Failure Modes

An entry outlives its expiry by the size of a clock step; visible only as
a stale read.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Insert an entry with a one-second expiry.
2. Step the wall clock back an hour.
3. The entry expires one second later.

### Phase 1: Code Implementation

#### Step 1: Swap the accessor

## Validation

### Testing Strategy

1. **Scenario**: A wall-clock step during an entry's lifetime.
   **Expected**: Expiry is unaffected.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- `clock::Now`.
