# Recommendation 0021: Cache Warm-up Order

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-02
- **Status**: Final [joint decision → 0022-cache-metrics-surface §
  A1: whether warm-up counts toward the hit-rate metric]
- **Type**: Architecture
- **Profile**: foundational — locks the warm-up contract across both
  readers and the metrics surface
- **Priority**: High
- **Seam Lineage**: `cache::Warm` — 3rd point-fix; trail: a1b2c3d +
  tracker#204, tracker#331, tracker#402.
- **Cluster**: 0020-cache-eviction-policy, 0022-cache-metrics-surface

## Problem Statement

Warm-up populates the cache in insertion order, which makes the first
requests after a restart slower than the steady state.

## Critical Assumptions

- **A1 [Warm-up order is derivable from the manifest alone]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The manifest carries a declared priority per entry;
    shown in Research Findings.
  - **If wrong**: Warm-up needs a second input nothing produces.

#### Normative Contracts

**C1**

```normative
// Warm-up populates entries in declared-priority order.
func WarmUp(m Manifest) error
```

## Proposed Solution

Read the manifest's declared priority and populate in that order.

## Decision Rationale

Premortem: no fatal failure mode found; the manifest is already read at
start-up, so the order costs nothing.
Ground-sweep: clean — no prior art in the tree contradicts the ordering.
Joint-check: OPEN → 0022-cache-metrics-surface §A1: whether warm-up
counts toward the hit-rate metric.
