# Recommendation 0102: Frame Padding Alignment

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-02
- **Status**: Final
- **Type**: Feature
- **Profile**: standard — one contract, the alignment rule
- **Priority**: Medium
- **Related Issues**: tracker#411
- **Predecessors**: 0101-frame-padding-byte

## Problem Statement

Readers disagree about where a padded frame's payload ends when the pad
run is longer than one byte.

### Critical Assumptions

- **A1 [Both readers accept a multi-byte pad run]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `frame::Decode` skips to the declared length in both.
  - **If wrong**: One reader mis-slices every aligned frame.

## Proposed Solution

### Approach

Align every payload to a four-byte boundary and declare the pad run in
the header, so a reader never infers it.

**The alignment values.** `align ∈ {1, 2, 4, 8}`, written as the header's
`align_log2` nibble. One is the identity, and eight is the ceiling the
header's four bits can express.

**Pad bytes are zero-filled and never inspected.**
A reader that reads a pad byte's value has read past the declared length,
which is the bug this alignment rule exists to make impossible.

The rest of the approach is ordinary prose, and a **bold run written
mid-paragraph** is emphasis: it opens nothing and names nothing, so it is
not addressable.

### Load-Bearing Decisions

- **D1** The pad run is declared, never inferred. A reader computes the
  payload end from the declared length alone.
- **D2** Alignment is per-frame, not per-stream. A stream may mix aligned
  and unaligned frames.
- **D6** The `align_log2` nibble is reserved-zero on version 1 frames, so
  an old reader rejects an aligned frame rather than mis-slicing it.

### Technical Design

The writer pads after the payload and before the checksum, so the
checksum covers the pad run.

#### Normative Contracts

**C1** the alignment contract

```normative
Encode(payload, align) writes ceil(len(payload)/align)*align payload
bytes, zero-filling the remainder, and sets align_log2 = log2(align).
```

### Decision Rationale

Declaring the pad run costs four header bits and removes a whole class of
reader disagreement. Premortem verdict: the failure would be a version-1
reader silently accepting an aligned frame, which D6 forecloses.

## Consequences

Version-1 readers reject aligned frames outright, which is the intended
migration pressure.

## Testing Strategy

1. An aligned frame round-trips through both readers.
2. A version-1 reader rejects a frame with a non-zero `align_log2`.

### Minimum Viable Validation

Encode at every alignment in the closed set and decode on both readers.

## Failure Modes

1. A writer sets `align_log2` without padding, and the reader over-reads.

## Finalization Gate

See `gate.md`.
