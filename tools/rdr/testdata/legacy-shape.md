# Recommendation 0001: Frame Length Prefix Width

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2025-11-04
- **Status**: Implemented (`main` a1b2c3d)
- **Type**: Architecture
- **Priority**: High
- **Related Issues**: tracker#88
- **Overrides**: none

## Problem Statement

The frame writer emits a length prefix before each payload, but the width
of that prefix was never fixed. Two readers disagree about whether it is
two bytes or four, and the disagreement only shows up on payloads larger
than 64 KiB.

## Proposed Solution

### Approach

Fix the prefix at four bytes, unsigned, big-endian.

### Technical Design

The writer emits the width unconditionally. The reader consumes exactly
four bytes before interpreting anything that follows.

#### Dependency Source Verification

The upstream buffer library exposes `buffer::PutUint32` and
`buffer::Uint32`, both big-endian, so no byte-order shim is needed.
`buffer::PutUint32` writes most-significant byte first; confirmed by
reading the package source rather than its documentation.

#### Normative Contracts

- The length prefix is exactly four bytes.
- The prefix is unsigned big-endian.
- A frame whose declared length exceeds the remaining input is a read
  error, not a truncated read.

#### Illustrative Code

    write_frame(payload):
      emit_u32_be(len(payload))
      emit(payload)

### Decision Rationale

Four bytes costs two bytes per frame over the two-byte alternative and
removes an entire class of size-dependent bug. Premortem: the likeliest
failure is a reader built before this lock; the version handshake catches
it.

## Alternatives Considered

### Alternative 1: Two-byte prefix

**Description**: Keep the narrower prefix and cap payloads at 64 KiB.

**Pros**:

- Two bytes smaller per frame.

**Cons**:

- Caps payload size at a value the format has no reason to impose.

**Reason for rejection**: The cap is a product decision leaking into a
wire format.

### Briefly Rejected

- **Varint prefix**: Saves bytes on small frames but makes the reader's
  bounds check depend on a decode that can itself fail.

## Context

### Background

Discovered when a caller sent a 70 KiB payload and the reader silently
truncated it.

### Technical Environment

The frame codec, its two readers, and the buffer library beneath them.

## Research Findings

### Investigation

Read both readers and the writer; compared their prefix handling.

### Key Discoveries

- **Verified** — the writer emits four bytes; one reader consumes two.
- **Documented** — the buffer library's integer helpers are big-endian.

## Trade-offs

### Consequences

- Two extra bytes per frame.
- Payload size is no longer capped by the prefix.

### Risks and Mitigations

- **Risk**: A reader built before the lock still consumes two bytes.
  **Mitigation**: The version handshake refuses the older reader.

### Failure Modes

A mismatched reader misreads the length and then misframes everything
after it. It surfaces as a decode error on the following frame, not on the
frame that actually broke.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Write a 70 KiB frame.
2. Read it back with the fixed reader.
3. The payload compares byte-for-byte equal to the input.

### Phase 1: Code Implementation

#### Step 1: Widen the reader

Consume four bytes where it consumed two.

#### Step 2: Add the bounds check

Refuse a declared length larger than the remaining input.

## Validation

### Testing Strategy

1. **Scenario**: A payload larger than 64 KiB.
   **Expected**: Round-trips byte-for-byte.

## Finalization Gate

### Contradiction Check

No contradictions found between research findings, design principles, and
proposed solution.

### Assumption Verification

Both records are Verified against the buffer library's source. Neither is
self-referential.

### Scope Verification

The Minimum Viable Validation is in scope and runs in Phase 1.

### Cross-Cutting Concerns

Versioning: the handshake already refuses mismatched readers.

### Proportionality

One contract, right-sized.

## References

- Buffer library source, integer helpers.
