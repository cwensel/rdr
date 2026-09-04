# Recommendation 0040: Impact Fixture

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-09-04
- **Status**: Final
- **Type**: Feature
- **Profile**: small
- **Priority**: Medium
- **Predecessors**: 0103-sidecar-import (load-bearing), 0040-impact-fixture
- **Overrides**: 0120-sidecar-export, 0103-sidecar-import

## Problem Statement

A synthetic record whose Metadata names two records it succeeds or
overrides, one of them twice, and itself once — the set the projection
reads is {0103, 0120}.

## Proposed Solution

Retire the sidecar.
