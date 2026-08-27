# Recommendation 0023: Cache Shard Count

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-07-20
- **Status**: Implemented
- **Type**: Architecture
- **Profile**: small
- **Priority**: Low

## Problem Statement

The shard count was a constant with no stated basis.

## Critical Assumptions

- **A1 [Contention is the binding cost, not memory]**
  - **Status**: Verified
  - **Method**: Derivation
  - **Evidence**: Shown in Research Findings.
  - **If wrong**: The count trades the wrong resource.

## Proposed Solution

Derive the count from the configured parallelism.

## Decision Rationale

Premortem: no fatal failure mode found.
