# Recommendation 0026: Cache Hash Identity

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-20
- **Status**: Draft
- **Type**: Architecture
- **Profile**: foundational — locks the key-hash pre-image across every
  reader, so the whole corpus rehashes if it moves
- **Priority**: High

## Problem Statement

The cache key hash has no written pre-image, so two readers that agree
today may disagree after either is refactored.

## Critical Assumptions

- **A1 [Load-bearing]**: The pre-image is stable across readers.
  - **Evidence**: `internal/cache/key.go::Preimage` writes the fields in
    declaration order, and the round-trip test asserts it.
  - **Method**: Source Search
  - **Status**: Verified

## Normative Contracts

The key pre-image is the tuple `(namespace, shard, key)` encoded with a
single NUL separator, and the hash is taken over those bytes with no
trailing newline.

## Decision Rationale

A written pre-image is the only thing that makes two readers comparable.
