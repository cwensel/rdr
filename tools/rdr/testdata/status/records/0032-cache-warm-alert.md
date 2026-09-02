# Recommendation 0032: Cache Warm Alert

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-15
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Low
- **Predecessors**: 9999-cache-warm-threshold

## Problem Statement

A warm ratio below the threshold is visible only to whoever is reading
the report at the time.

## Critical Assumptions

- **A1 [An alert sink already exists]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `alert/sink.go::Emit` is the one sink every check uses.
  - **If wrong**: The alert needs a transport nothing provides.

## Proposed Solution

Emit through the existing sink when the ratio crosses the threshold.

## Decision Rationale

Premortem: no fatal failure mode found; the sink is shared and rate-limited.
