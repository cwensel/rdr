# Recommendation 0038: Cache Warm Lead

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-23
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Low
- **Cluster**: 0039-cache-warm-follow

## Problem Statement

A warm-cycle lead value is computed twice, once here and once by a sibling
that reads the same source counter.

## Critical Assumptions

- **A1 [The source counter is stable across a cycle]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/cycle.go::Counter` is written once per cycle.
  - **If wrong**: The lead value would drift mid-cycle.

## Proposed Solution

Compute the lead value once and expose it for the sibling to read.

## Decision Rationale

Premortem: no fatal failure mode found; the counter is already stable.
