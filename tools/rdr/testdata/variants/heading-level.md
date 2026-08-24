# Recommendation 0101: Frame Padding Byte

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-01
- **Status**: Final [joint decision → 0004-frame-checksum-algorithm §
  A3: whether padding is covered by the checksum]
- **Type**: Feature
- **Profile**: standard — one contract, the padding rule
- **Priority**: Medium
- **Related Issues**: tracker#410
- **Predecessors**: 0004-frame-checksum-algorithm

## Problem Statement

Frames whose payload is an odd length are padded by the writer and not by
one of the readers.

### Critical Assumptions

- **A1 [Both readers tolerate a trailing pad byte today]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `frame::Decode` discards bytes past the declared length
    in both readers.
  - **If wrong**: One reader rejects every odd-length frame.
- **A2 [The pad byte is never interpreted as payload]**
  - **Status**: Pending
  - **Method**: Spike
  - **Evidence**: A spike encoding an odd-length payload and decoding it on
    both readers is owed before lock.
  - **If wrong**: A reader hands the caller one byte of garbage.

## Proposed Solution

### Approach

State the pad rule in the contract: a single zero byte, present iff the
payload length is odd, excluded from the declared length.

### Technical Design

#### Normative Contracts

```normative
func Pad(payload []byte) []byte
```

#### Load-Bearing Decisions

- **Wire / byte format** — the pad is one zero byte after the payload.

### Decision Rationale

A stated rule replaces two readers' differing tolerance.

## Alternatives Considered

### Briefly Rejected

- **Pad to four bytes**: Costs up to three bytes per frame for alignment
  no reader needs.

## Context

### Background

Found while comparing the readers' handling of odd-length payloads.

### Technical Environment

The frame codec and both readers.

## Research Findings

### Investigation

Read both decoders' length handling.

### Key Discoveries

- **Verified** — both discard trailing bytes.

## Trade-offs

### Consequences

- Odd-length frames are one byte longer.

### Risks and Mitigations

- **Risk**: A third reader appears that does not discard trailing bytes.
  **Mitigation**: The rule is in the contract, so a new reader reads it.

### Failure Modes

A reader that does not discard the pad hands a caller one extra byte,
visible as a length mismatch in every odd-length test.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Encode a one-byte payload.
2. Decode it on both readers.
3. Both return exactly one byte.

### Phase 1: Code Implementation

#### Step 1: Add the pad rule to the writer

## Validation

### Testing Strategy

1. **Scenario**: Odd-length payload round-trips on both readers.
   **Expected**: Payload is returned unchanged.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- `frame::Decode` in both readers.
