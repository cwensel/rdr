# Recommendation 0014: Frame Header Debug Flag

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-09-03
- **Status**: Demoted [→ 0013-frame-header-order-v2]
- **Type**: Feature
- **Profile**: small — one flag byte
- **Priority**: Low
- **Predecessors**: 0013-frame-header-order-v2

## Problem Statement

A debug flag was proposed for the flags byte; it turned out not to be a design fork at all.

## Critical Assumptions

- **A1 [The reserved bits are ignored on read.]**
  - **Status**: Verified
  - **Method**: Peer RDR
  - **Evidence**: `0013-frame-header-order-v2:C1` fixes the layout the
    debug flag would have used.
  - **If wrong**: a future flag breaks every existing reader.

## Proposed Solution

### Approach

Declare the six bits reserved and require they be written zero.

### Technical Design

The flags byte carries two live bits and six reserved.

#### Normative Contracts

**C1**

```normative
const FlagsReservedMask uint8 = 0b1111_1100
```

- Reserved bits are written zero and ignored on read.

#### Illustrative Code

```go
flags &^= FlagsReservedMask
```

### Decision Rationale

Reserving is cheaper than versioning.

## Alternatives Considered

### Briefly Rejected

- A second flags byte: nothing needs it yet.

## Context

### Background

The flags byte was added with one flag.

### Technical Environment

One writer, two readers.

## Research Findings

### Investigation

Read both readers' flag handling.

### Key Discoveries

Neither masks; both compare the whole byte.

## Trade-offs

### Consequences

Both readers change.

### Risks and Mitigations

The mask is the mitigation.

### Failure Modes

- A reader comparing the whole byte rejects a valid frame.

## Implementation Plan

### Prerequisites

None.

### Minimum Viable Validation

Set a reserved bit; assert the frame still decodes.

### Phase 1: Code Implementation

#### Step 1: Mask on read

Apply the mask.

## Validation

### Testing Strategy

One masking test.

## Finalization Gate

See `gate.md`.

## References

- 0010-frame-header-labelled
