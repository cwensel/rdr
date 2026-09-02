# Recommendation 0031: Cache Warm Report

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-14
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Medium
- **Predecessors**: 0030-cache-warm-ratio, 0021-cache-warmup-order

## Problem Statement

The warm ratio is computed but never reported; operators read the raw
counter and infer it by hand.

## Critical Assumptions

- **A1 [The ratio is already exported]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/warm.go::Ratio` is called from the exporter.
  - **If wrong**: The report has nothing to read.

## Proposed Solution

Print the ratio beside the counter in the existing report line.

## Decision Rationale

Premortem: no fatal failure mode found; the ratio is one more column.
