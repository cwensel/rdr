# Recommendation 0022: Cache Metrics Surface

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-03
- **Status**: Draft [revised from Final 2026-08-10; re-verify A1,A2 — the
  dashboard panel reads the raw counter after all]
- **Type**: Architecture
- **Profile**: small
- **Priority**: Medium

## Problem Statement

The hit-rate metric counts warm-up reads, which makes the number
unreadable for the first minute after a restart.

## Critical Assumptions

- **A1 [No dashboard reads the raw counter directly]**
  - **Status**: Refuted
  - **Method**: Source Search
  - **Evidence**: One dashboard panel reads the counter without the
    ratio; it has to move first.
  - **If wrong**: The counter cannot be re-scoped in place.
- **A2 [The ratio is computed at read time, not accumulated]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The ratio is derived on export; nothing persists it.
  - **If wrong**: The change needs a migration.

## Proposed Solution

Scope the counter to steady-state reads and move the panel first.

## Decision Rationale

Premortem: the dashboard panel is the only blocker and it is one edit.
