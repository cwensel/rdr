# Recommendation 0028: Cache Flush Hook

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-12
- **Status**: Final
- **Type**: Architecture
- **Profile**: small
- **Priority**: Low

## Problem Statement

Nothing tells a cache that its backing table was rewritten.

## Critical Assumptions

- **A1 [The rewrite path has one exit]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: Every rewrite returns through one function.
  - **If wrong**: The hook misses a path and serves stale entries.

## Proposed Solution

Fire a flush hook from the rewrite path's single exit.

## Decision Rationale

Premortem: no fatal failure mode found.
