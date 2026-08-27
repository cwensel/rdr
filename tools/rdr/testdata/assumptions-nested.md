# Recommendation 0002: Cache Eviction Policy

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-01-19
- **Status**: Implemented
- **Type**: Feature
- **Profile**: mid — one contract plus the operator-facing metrics surface
- **Priority**: Medium
- **Related**: tracker#204
- **Predecessors**: 0001-frame-length-prefix-width
- **Overrides**: none
- **Seam Lineage**: `cache::Evict` — 1st point-fix; trail: a1b2c3d +
  tracker#204. No prior closed point-fixes at this locus beyond the one
  named.

## Problem Statement

The cache grows without bound. Nothing decides what leaves when memory
runs short, so the process is killed instead.

## Proposed Solution

### Approach

Evict least-recently-used entries until the cache is back under its
configured byte ceiling.

### Technical Design

Entries carry a monotonic access stamp. Eviction walks ascending by stamp
and stops the moment the ceiling is satisfied.

### Critical Assumptions

- **A1 [The access stamp is monotonic under concurrent reads]**
  - **Status**: Verified
  - **Method**: Source Search + Spike
  - **Evidence**: `clock::Monotonic` is documented and implemented as
    non-decreasing; a two-thread spike recorded no inversion over ten
    million reads.
  - **If wrong**: Eviction picks a recently-used entry, and the hit rate
    collapses without any error surfacing.
- **A2 [Byte accounting matches the allocator's view within 5%]**
  - **Status**: Verified
  - **Method**: Spike (repro)
  - **Evidence**: Measured resident size against the accounted total
    across a full eviction cycle; the gap never exceeded 3%.
  - **If wrong**: The ceiling is enforced against a number that does not
    describe memory, so the process is killed anyway.

#### Normative Contracts

- Eviction is least-recently-used, by access stamp ascending.
- Eviction stops at the first entry that brings the total under the
  ceiling; it does not evict past it.
- `cache::Evict` returns the number of entries removed and the bytes
  reclaimed.

#### Load-Bearing Decisions

- **Identity** — two entries are the same when their keys compare equal
  byte-for-byte; the access stamp is not part of identity.
- **Selection / predicate** — when several entries share an access stamp,
  the one with the larger byte size is evicted first, on the grounds that
  it buys more headroom per eviction.

#### Illustrative Code

    evict(ceiling):
      for entry in entries_by_stamp_ascending():
        if total <= ceiling: break
        remove(entry)

### Decision Rationale

LRU is the policy operators already expect from a cache, and the access
stamp needed for it is already recorded. Premortem: the likeliest failure
is stamp inversion under concurrency, which A1 verifies directly.
Joint-check: no peer owns the eviction seam.

## Alternatives Considered

### Alternative 1: Least-frequently-used

**Description**: Evict by access count rather than recency.

**Pros**:

- Resists a single large scan flushing the whole cache.

**Cons**:

- Requires a decay policy, which is a second contract this record would
  then own.

**Reason for rejection**: Two contracts in one record; the split test is
contract count.

### Briefly Rejected

- **Random eviction**: Cheap, but its hit rate is not something an
  operator can reason about.

## Context

### Background

The cache was introduced without a ceiling on the assumption that the
working set was small. It is not.

### Technical Environment

The cache module, the monotonic clock, and the metrics surface.

## Research Findings

### Investigation

Read the allocator's accounting path and the clock's monotonicity
guarantee.

### Key Discoveries

- **Verified** — the access stamp does not invert under concurrent reads.
- **Verified** — byte accounting tracks resident size within 3%.

## Trade-offs

### Consequences

- A large scan can evict the useful working set.
- Memory is bounded, so the process is no longer killed.

### Risks and Mitigations

- **Risk**: A scan flushes the cache.
  **Mitigation**: Scans use the no-cache read path.

### Failure Modes

If the stamp inverts, eviction quietly removes hot entries. It surfaces as
a hit-rate drop in the metrics, not as an error.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Fill the cache past its ceiling.
2. Read one early entry to refresh its stamp.
3. Trigger eviction.
4. The refreshed entry survives; the untouched older entries do not.

### Phase 1: Code Implementation

#### Step 1: Record the access stamp

#### Step 2: Implement the eviction walk

#### Step 3: Export the eviction metrics

## Validation

### Testing Strategy

1. **Scenario**: Cache filled past the ceiling with one refreshed entry.
   **Expected**: The refreshed entry survives eviction.

## Finalization Gate

### Contradiction Check

No contradictions found between research findings, design principles, and
proposed solution.

### Assumption Verification

A1 and A2 are both Verified with concrete evidence. Neither is Docs Only.
Neither cites this record.

### Scope Verification

The Minimum Viable Validation runs in Phase 1 and is not deferred.

### Cross-Cutting Concerns

Memory management: the ceiling is the whole point. Concurrency model: A1
covers the stamp under concurrent reads.

### Proportionality

One independent contract — the eviction policy. Profile `mid` matches: one
contract plus a user-facing metrics surface.

## References

- Clock package, monotonicity guarantee.
- Allocator accounting source.
