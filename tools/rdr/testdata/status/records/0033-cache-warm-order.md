# Recommendation 0033: Cache Warm Order

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-20
- **Status**: Draft
- **Type**: Architecture
- **Profile**: mid — one warm path, measured by a spike
- **Priority**: Medium

## Problem Statement

The warm path fills the cache in whichever order the loader yields, and
two readers assume opposite orders.

## Critical Assumptions

- **A1 [Insertion order is the order the loader yields]**
  - **Status**: Verified
  - **Method**: Spike
  - **Evidence**: spikes/warm-order.md — measured both orders across a
    full replay; insertion order held.
  - **If wrong**: The warm path needs its own ordering key.
- **A2 [No reader observes the fill before it completes]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The fill holds the write lock for its whole span.
  - **If wrong**: A partial fill is a compatibility surface.

## Proposed Solution

Warm in loader order and document it as the contract.

## Normative Contracts

The warm order is a policy, not a locked format; nothing here is parsed twice.

Determinacy: n/a — no algorithmic contract is locked.

## Decision Rationale

- Premortem: no failure mode survived the spike.
- Ground-sweep: clean — the two readers were the only assumptions.
- Joint-check: fired → 0026 (home: OPEN) — which record owns the key pre-image.
