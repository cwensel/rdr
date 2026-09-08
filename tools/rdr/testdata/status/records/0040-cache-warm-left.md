# Recommendation 0040: Cache Warm Left

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-25
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Medium
- **Cluster**: 0041-cache-warm-right

## Problem Statement

A warm-cycle left value is computed independently of a sibling that
reads the same source counter, with neither naming the other.

## Critical Assumptions

- **A1 [The source counter is stable across a cycle]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/cycle.go::Counter` is written once per cycle.
  - **If wrong**: The left value would drift mid-cycle.

## Proposed Solution

Compute the left value from the source counter.

## Decision Rationale

Premortem: no fatal failure mode found; the counter is already stable.
