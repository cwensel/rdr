# Recommendation 0103: Cache Eviction Order

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-03
- **Status**: Implemented (`main` a1b2c3d)
- **Type**: Technical Debt
- **Profile**: standard — one contract, the eviction comparator
- **Priority**: Low
- **Related Issues**: tracker#431
- **Predecessors**: 0102-cache-entry-expiry-clock

## Problem Statement

Eviction order depends on map iteration order, which is unspecified.

## Critical Assumptions

- **A1 [Eviction reads entries through the map]**
  - **Status**: **Verified**
  - **Method**: Source Search
  - **Evidence**: `cache::evict` ranges over the entry map directly.
  - **If wrong**: The order is already fixed elsewhere and this record is
    unnecessary.
- **A2 [No caller depends on the current order]**
  - **Status**: REFUTED (one test pinned the old order)
  - **Method**: Source Search + Spike
  - **Evidence**: `TestEvictOrder` asserted the accidental order; it is
    rewritten against the comparator.
  - **If wrong**: A caller's behaviour changes silently.
- **A3 [The comparator is total over live entries]**
  - **Status**: Verified as narrowed — total over entries with an expiry;
    entries without one sort last, by insertion.
  - **Method**: Derivation
  - **Evidence**: Shown in Research Findings.
  - **If wrong**: Two entries compare equal and the order is again
    unspecified.
- **A4 [The insertion counter never wraps in practice]**
  - **Status**: Verified | Pending | Unverified
  - **Method**: Derivation
  - **Evidence**: At one insertion per nanosecond the counter wraps after
    five centuries.
  - **If wrong**: The tie-break inverts once.
- **A5 [The tie-break is a design choice, not a fact]**
  - **Status**: Resolved — settled by the Naming decision below
  - **Method**: Design Decision
  - **Evidence**: Load-Bearing Decisions, Selection / predicate.
  - **If wrong**: Nothing; it is a choice.

## Proposed Solution

### Approach

Sort by expiry, then by insertion counter.

### Technical Design

#### Normative Contracts

```normative
func Less(a, b Entry) bool
```

#### Load-Bearing Decisions

- **Selection / predicate** — expiry ascending, then insertion ascending.

### Decision Rationale

A stated order is testable; an accidental one is not.

## Alternatives Considered

### Briefly Rejected

- **Random eviction**: Cheaper, but the tests want determinism more than
  the cache wants speed.

## Context

### Background

A test passed on one runtime and failed on another.

### Technical Environment

The cache package.

## Research Findings

### Investigation

Read the eviction loop and the one test that pinned its order.

### Key Discoveries

- **Verified** — map iteration was the only order.

## Trade-offs

### Consequences

- Eviction is deterministic.

### Risks and Mitigations

- **Risk**: The comparator is slower than map iteration.
  **Mitigation**: Eviction is off the hot path.

### Failure Modes

A non-total comparator leaves the order unspecified again, visible as a
flaky ordering test.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Insert three entries with distinct expiries in reverse order.
2. Evict one.
3. The earliest expiry is gone.

### Phase 1: Code Implementation

#### Step 1: Add the comparator

## Validation

### Testing Strategy

1. **Scenario**: Three entries, reverse insertion.
   **Expected**: Earliest expiry evicts first.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- `cache::evict`.
