# Recommendation 0024: Cache Persistence

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-07-25
- **Status**: Deferred [revisit when the restart budget is measured]
- **Type**: Architecture
- **Profile**: mid
- **Priority**: Low

## Problem Statement

A restart discards the cache. Whether that costs anything is unmeasured.

## Critical Assumptions

- **A1 [Restarts are rare enough that the cost is not worth paying]**
  - **Status**: Pending
  - **Method**: Source Search
  - **Evidence**: Pending — needs the restart budget.
  - **If wrong**: Persistence is owed.

## Proposed Solution

Parked: no acceptable mechanism until the budget exists.

## Decision Rationale

Parked pending the restart budget.
