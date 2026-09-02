# Recommendation 0025: Cache Key Encoding

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-07-28
- **Status**: Draft
- **Type**: Architecture
- **Priority**: Medium

## Problem Statement

Keys are encoded ad hoc at each call site, so two callers can name the
same entry differently.

## Critical Assumptions

- **A1 [Collisions are the binding risk, not encoding cost]**
  - **Status**: Verified | Pending | Unverified
  - **Method**: TBD
  - **Evidence**: TBD — `internal/cache/key.go::Preimage` is the only
    encoder in the tree, so a collision test there settles it.
  - **If wrong**: The encoding optimises the wrong axis.

## Proposed Solution

TBD.

## Decision Rationale

TBD.
