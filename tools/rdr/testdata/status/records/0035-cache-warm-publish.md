# Recommendation 0035: Cache Warm Publish

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-20
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Low
- **Predecessors**: 0034-cache-warm-signal

## Problem Statement

The warm ratio and the alert that reads it were two records; they are one
seam and the split left neither able to state the publish contract.

## Critical Assumptions

- **A1 [The metrics surface accepts a gauge]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `metrics/surface.go::Gauge` is the registration point.
  - **If wrong**: The signal needs a transport nothing provides.

## Proposed Solution

Publish the ratio as a gauge and alert off the same value.

## Decision Rationale

Premortem: no fatal failure mode found; supersedes 0034, which was never
implemented and never will be.
