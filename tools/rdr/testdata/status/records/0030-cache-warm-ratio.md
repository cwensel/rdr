# Recommendation 0030: Cache Warm Ratio

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-20
- **Status**: Final
- **Type**: Architecture
- **Profile**: small
- **Priority**: Low

## Problem Statement

Nothing reports how much of a cache was warmed before it served.

## Critical Assumptions

- **A1 [Warm-up runs to completion before the first read]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The warm-up loop returns before the listener opens.
  - **If wrong**: The ratio would read partial on a cache that is whole.

## Proposed Solution

Expose the warmed-entry ratio as one gauge on the existing endpoint.

## Decision Rationale

Premortem: no fatal failure mode found.
