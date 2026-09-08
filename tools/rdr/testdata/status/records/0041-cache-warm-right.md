# Recommendation 0041: Cache Warm Right

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-25
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Medium
- **Cluster**: 0040-cache-warm-left

## Problem Statement

A warm-cycle right value is computed independently of a sibling that
reads the same source counter, with neither naming the other.

## Critical Assumptions

- **A1 [The source counter is stable across a cycle]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/cycle.go::Counter` is written once per cycle.
  - **If wrong**: The right value would drift mid-cycle.

## Proposed Solution

Compute the right value from the source counter.

## Decision Rationale

Premortem: no fatal failure mode found; the counter is already stable.
