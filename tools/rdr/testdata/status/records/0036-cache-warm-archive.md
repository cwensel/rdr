# Recommendation 0036: Cache Warm Archive

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-21
- **Status**: Implemented
- **Type**: Feature
- **Profile**: small
- **Priority**: Low

## Problem Statement

Warm-cycle counters roll off with no archive, so a long-window trend has
nothing to read.

## Critical Assumptions

- **A1 [An archive table already exists]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `cache/archive.go::Table` is the existing sink.
  - **If wrong**: The archive needs a store nothing provides.

## Proposed Solution

Write each cycle's counters into the existing archive table.

## Decision Rationale

Premortem: no fatal failure mode found; the table already rotates itself.
