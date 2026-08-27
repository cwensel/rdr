# Recommendation 0003: Frame Checksum Placement

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-04-02
- **Status**: Superseded [→ 0004-frame-checksum-algorithm]
- **Type**: Technical Debt
- **Profile**: large — locks the on-disk frame grammar
- **Priority**: High
- **Related Issues**: tracker#331
- **Predecessors**: 0001-frame-length-prefix-width
- **Overrides**: 0001-frame-length-prefix-width narrows the trailing-byte
  rule stated there; the prefix width itself is unchanged.
- **Seam Lineage**: `frame::Encode` — 2nd point-fix; trail: a1b2c3d +
  tracker#204, tracker#331.
  Accretion disposition: the 2 point-fixes at `frame::Encode` are NOT one
  missing design decision because the first fixed the prefix width and
  this one fixes the checksum position, which are independent fields;
  cite: 0001-frame-length-prefix-width.

## Problem Statement

The checksum is written after the payload, so a reader cannot verify a
frame until it has already buffered the whole thing.

### Critical Assumptions

- **A1 [The checksum can be computed before the payload is emitted]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `hash::Sum32` accepts the complete payload slice, which
    the writer already holds before it emits anything.
  - **If wrong**: The writer would have to buffer twice, and the change
    would cost more than the streaming read it buys.
- **A2 [No shipped reader depends on the trailing position]**
  - **Status**: Verified
  - **Method**: Source Search + Peer RDR
  - **Evidence**: Both readers call `frame::Decode`, which locates the
    checksum by offset constant, not by scanning to the end.
  - **If wrong**: A shipped reader misreads every frame, and the format
    change needs a version gate it does not currently have.

## Proposed Solution

### Approach

Move the checksum into the header, immediately after the length prefix.

### Technical Design

The header becomes length prefix, then checksum, then payload. A reader
can verify incrementally as the payload arrives.

#### Normative Contracts

State each Normative item in a clearly labeled block, e.g.:

```normative
func Encode(payload []byte) []byte
func Decode(frame []byte) (payload []byte, err error)
```

- The checksum occupies bytes 4 through 7 of every frame.
- The checksum covers the payload only, not the length prefix.

> Transient — scheduled deletion by 0004-frame-checksum-algorithm, Phase 1;
> the compatibility shim that accepts a trailing checksum goes with it.

#### Round-Trip / Inverse Invariants

`Decode ∘ Encode = identity` on the class of payloads shorter than the
declared maximum. Equality is byte-for-byte on the reconstructed payload,
not merely the absence of an error.

### Premortem

The likeliest failure is a reader that locates the checksum by scanning
rather than by offset. A2 checks exactly that, and finds none.

### Decision Rationale

Header placement is what every comparable format does, and it is what
makes streaming verification possible at all.

## Alternatives Considered

### Briefly Rejected

- **Keep the trailing checksum and buffer**: Preserves the format but
  gives up the streaming read that motivated the change.

## Context

### Background

Noticed when a streaming consumer had to buffer 40 MiB frames purely to
reach the checksum.

### Technical Environment

The frame codec and its two readers.

## Research Findings

### Investigation

Read `frame::Decode` in both readers to see how each locates the checksum.

### Key Discoveries

- **Verified** — both readers use an offset constant.

## Trade-offs

### Consequences

- Streaming verification becomes possible.
- The frame grammar changes, so the format version moves.

### Risks and Mitigations

- **Risk**: An unshipped reader scans for the checksum.
  **Mitigation**: The compatibility shim accepts both placements until the
  successor record removes it.

### Failure Modes

A reader using the old placement reads payload bytes as a checksum and
rejects every frame. It surfaces immediately and loudly, which is the good
case.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Encode a 1 MiB payload.
2. Verify the checksum after reading only the first 8 bytes plus the
   payload stream.
3. The verification succeeds without the reader buffering the frame.

### Phase 1: Code Implementation

#### Step 1: Move the checksum write

#### Step 2: Add the compatibility shim

## Validation

### Testing Strategy

1. **Scenario**: A frame encoded with the new placement.
   **Expected**: Verified incrementally, no full buffering.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- Hash package, `Sum32`.
