# Recommendation 0020: Cache Eviction Policy

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-01
- **Status**: Draft
- **Type**: Architecture
- **Profile**: mid — one contract plus the eviction metrics surface
- **Priority**: High
- **Seam Lineage**: `cache::Evict` — 2nd point-fix; trail: a1b2c3d +
  tracker#204, tracker#331.
  - Accretion disposition: the 2 point-fixes at `cache::Evict` are NOT one
    missing design decision — each fixed a distinct call site's assumption;
    cite: 0023-cache-shard-count.

## Problem Statement

The cache evicts on insertion pressure, but nothing states which entry
goes. Two call sites assume opposite answers.

## Critical Assumptions

- **A1 [The working set fits in the configured bound at steady state]**
  - **Status**: Pending
  - **Method**: Source Search
  - **Evidence**: Pending — measure the resident set across a full
    replay before the policy is chosen.
  - **If wrong**: Any policy thrashes and the bound is the wrong knob.
- **A2 [Eviction order is observable to no caller]**
  - **Status**: Pending
  - **Method**: Source Search
  - **Evidence**: Pending.
  - **If wrong**: The policy becomes a compatibility surface.

## Proposed Solution

Not yet chosen.

## Decision Rationale

Pending the assumptions above.

Joint-check: clear (3 peers; no shared anchor or literal).

## Implementation Plan

### Phase 1: Code Implementation

#### Step 1: Measure the resident set

Replay a full day of traffic against the bounded cache and record the
resident set at each eviction, before any policy is chosen.
