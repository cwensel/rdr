# Recommendation 0039: Cache Warm Follow

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-24
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Low
- **Cluster**: 0038-cache-warm-lead
- **Predecessors**: 0038-cache-warm-lead

## Problem Statement

A follow-on value needs the lead value computed before it can be derived.

## Critical Assumptions

- **A1 [The lead value is exposed for a reader]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The lead computation writes to a field a sibling can read.
  - **If wrong**: The follow value has nothing to build on.

## Proposed Solution

Derive the follow value from the exposed lead value.

## Decision Rationale

Premortem: no fatal failure mode found; the lead value is already exposed.
