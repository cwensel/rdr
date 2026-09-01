# Recommendation 0027: Cache Tier Labels

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-07-22
- **Status**: Implemented
- **Type**: Architecture
- **Profile**: small
- **Priority**: Low

## Problem Statement

Tier names are spelled three ways across the metrics and the config.

## Critical Assumptions

- **A1 [One label vocabulary already exists in the config loader]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The loader owns the enum; the metrics restate it.
  - **If wrong**: The label is minted twice and drifts.

## Proposed Solution

Read the tier label from the config loader everywhere it is printed.

## Decision Rationale

Premortem: no fatal failure mode found.
