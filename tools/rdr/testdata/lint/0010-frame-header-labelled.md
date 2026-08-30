# Recommendation 0010: Frame Header Field Order

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-09-01
- **Status**: Draft
- **Type**: Architecture
- **Profile**: mid — locks the header field order for the writer and both
  readers
- **Priority**: High
- **Predecessors**: 0011-frame-header-unlabelled

## Problem Statement

The header's field order is implied by the encoder's statement order and
by nothing else, so a reordering that keeps every field is a silent wire
break.

## Critical Assumptions

- **A1 [The reader tolerates no reordering.]**
  - **Status**: Verified
  - **Method**: Peer RDR
  - **Evidence**: `0011-frame-header-unlabelled:C1` fixes the header at
    four fields and states their widths; the order is what this record
    adds.
  - **If wrong**: a reordered header decodes as garbage rather than
    failing, and the corruption surfaces at the payload.

## Proposed Solution

### Approach

State the field order in the contract, so the encoder's statement order
stops being the specification.

### Technical Design

The header is four fields in a fixed order; the contract names the order
and the offsets together.

#### Normative Contracts

**C1**

```normative
type Header struct {
	Length  uint32 // offset 0
	Version uint8  // offset 4
	Flags   uint8  // offset 5
	Sum     uint32 // offset 6
}
```

- Fields appear in the order above, at the offsets given.

**C2**

```normative
func DecodeHeader(b []byte) (Header, error)
```

- Decoding a buffer shorter than ten bytes is `ErrShortHeader`, never a
  partial header.

#### Illustrative Code

```go
h, err := DecodeHeader(buf)
```

### Decision Rationale

Naming the offsets beside the fields makes a reordering a contract edit
rather than a refactor.

## Alternatives Considered

### Briefly Rejected

- A self-describing header: the length cost is larger than the format's
  whole payload in the common case.

## Context

### Background

The header predates the format's first written spec.

### Technical Environment

One writer, two readers, one repository.

## Research Findings

### Investigation

Read both readers for their field-order assumptions.

### Key Discoveries

Both readers hard-code the order; neither states it.

## Trade-offs

### Consequences

A reordering now requires a new record.

### Risks and Mitigations

The risk is a reader that reads offsets rather than fields; both were
checked.

### Failure Modes

- A reordered header decodes as garbage. Visible: payload corruption.

## Implementation Plan

### Prerequisites

None.

### Minimum Viable Validation

Encode and decode a header; assert the byte offsets.

### Phase 1: Code Implementation

#### Step 1: State the order

Write the contract.

## Validation

### Testing Strategy

A round-trip test per field.

## Finalization Gate

See `gate.md`.

### Cross-Cutting Concerns

- **Character encoding**: header field names are ASCII-only.

## References

- 0011-frame-header-unlabelled
