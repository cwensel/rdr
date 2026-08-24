# Recommendation 0013: Frame Header Field Order, Revised

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-09-05
- **Status**: Draft
- **Type**: Architecture
- **Profile**: mid — re-homes the header field order
- **Priority**: High
- **Predecessors**: 0010-frame-header-labelled
- **Overrides**: `0010-frame-header-labelled:C1` — the field order moves
  here; the offsets it stated are superseded by the reserved-bit layout.
  0010:C2 stands and is not touched.
- **Cluster**: 0010-frame-header-labelled

## Problem Statement

The reserved bits took a byte the original field order did not budget,
so the offsets 0010 stated no longer describe the header.

## Critical Assumptions

- **A1 [Only the offsets move; the field set is unchanged.]**
  - **Status**: Verified
  - **Method**: Peer RDR
  - **Evidence**: `0010-frame-header-labelled:C1` names four fields; this
    record keeps all four and changes only where they sit.
  - **If wrong**: a reader written against either record decodes the
    other's frames as garbage.

## Proposed Solution

### Approach

Restate the field order with the corrected offsets, and mark the
predecessor's contract overridden rather than editing it.

### Technical Design

The field set is 0010's; the offsets shift by one byte after Flags.

#### Normative Contracts

**C1**

```normative
type Header struct {
	Length  uint32 // offset 0
	Version uint8  // offset 4
	Flags   uint8  // offset 5
	Sum     uint32 // offset 7
}
```

- Fields appear in the order above, at the offsets given. This contract
  supersedes `0010-frame-header-labelled:C1`.

#### Illustrative Code

```go
h, err := DecodeHeader(buf)
```

### Decision Rationale

Overriding by ID keeps the old record readable and the citation intact.

## Alternatives Considered

### Briefly Rejected

- Editing 0010 in place: a locked record is never amended.

## Context

### Background

0010 predates the reserved-bit record.

### Technical Environment

One writer, two readers.

## Research Findings

### Investigation

Compared both offset tables.

### Key Discoveries

Only the trailing offset moves.

## Trade-offs

### Consequences

Readers pinned to 0010's offsets must move.

### Risks and Mitigations

The override edge is the worklist.

### Failure Modes

- A reader on the old offsets misreads Sum. Visible: checksum failure.

## Implementation Plan

### Prerequisites

None.

### Minimum Viable Validation

Decode a frame at the new offsets.

### Phase 1: Code Implementation

#### Step 1: Shift the offsets

Apply the new table.

## Validation

### Testing Strategy

One offset assertion per field.

## Finalization Gate

See `gate.md`.

## References

- 0010-frame-header-labelled
