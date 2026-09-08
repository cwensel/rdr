# Recommendation 0037: Cache Warm Digest

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-22
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Low
- **Predecessors**: 0036-cache-warm-archive

## Problem Statement

The archived counters have no rollup, so a reviewer must sum the raw rows
by hand.

## Critical Assumptions

- **A1 [The archive table is queryable by range]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/archive.go::Range` already scans by date.
  - **If wrong**: The digest needs a scan nothing provides.

## Proposed Solution

Sum the archived counters over a range into one digest row.

## Decision Rationale

Premortem: no fatal failure mode found; the range scan is already shared.
