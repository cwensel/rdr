# Recommendation 0029: Cache Size Report

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-14
- **Status**: Final
- **Type**: Architecture
- **Profile**: small
- **Priority**: Low

## Problem Statement

Operators cannot see how much memory each cache tier holds.

## Critical Assumptions

- **A1 [Each tier already tracks its own byte count]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The counter is maintained on insert and evict.
  - **If wrong**: The report needs a walk, not a read.

## Proposed Solution

Expose each tier's byte count on the existing metrics endpoint.

## Decision Rationale

Premortem: no fatal failure mode found.
