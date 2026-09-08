# Recommendation 0043: Cache Warm Urgent

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-26
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: High
- **Cluster**: 0042-cache-warm-late

## Problem Statement

A warm-cycle urgent value is computed independently of a lower-priority
sibling that reads the same source counter, with neither naming the
other.

## Critical Assumptions

- **A1 [The source counter is stable across a cycle]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/cycle.go::Counter` is written once per cycle.
  - **If wrong**: The urgent value would drift mid-cycle.

## Proposed Solution

Compute the urgent value from the source counter.

## Decision Rationale

Premortem: no fatal failure mode found; the counter is already stable.
