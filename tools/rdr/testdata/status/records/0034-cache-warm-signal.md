# Recommendation 0034: Cache Warm Signal

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-15
- **Status**: Superseded [→ 0035 — 2026-08-20: the warm signal and the alert
  are one seam; 0035 owns both]
- **Type**: Feature
- **Profile**: small
- **Priority**: Low

## Problem Statement

The warm ratio is computed but nothing publishes it.

## Critical Assumptions

- **A1 [The report already computes the ratio]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/report.go::WarmRatio` returns it today.
  - **If wrong**: The signal has nothing to publish.

## Proposed Solution

Publish the ratio on the existing metrics surface.

## Decision Rationale

Premortem: no fatal failure mode found; superseded before implementation.
