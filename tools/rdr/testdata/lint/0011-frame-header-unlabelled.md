# Recommendation 0011: Frame Header Field Widths

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-09-02
- **Status**: Draft
- **Type**: Architecture
- **Profile**: mid — locks the header field widths
- **Priority**: High

## Problem Statement

The header's field widths are inherited from the encoder's types.

## Critical Assumptions

- **A1 [Ten bytes is enough for the header.]**
  - **Status**: Verified
  - **Method**: Peer RDR
  - **Evidence**: `0010` fixes the field order, and the four fields it
    names sum to ten bytes.
  - **If wrong**: the header overruns into the payload.

## Proposed Solution

### Approach

State each width in the contract.

### Technical Design

Four fields, ten bytes.

#### Normative Contracts

```normative
const HeaderLen = 10
```

- The header is exactly ten bytes.

```normative
func HeaderWidth(field string) int
```

- An unknown field name is a zero width, never a panic.

#### Illustrative Code

```go
n := HeaderWidth("Length")
```

### Decision Rationale

Widths belong beside the order they are read in.

## Alternatives Considered

### Briefly Rejected

- Variable widths: they defeat the fixed-offset decode.

## Context

### Background

The widths were never written down.

### Technical Environment

One writer, two readers.

## Research Findings

### Investigation

Read the encoder's struct.

### Key Discoveries

Every width is a Go type's natural size.

## Trade-offs

### Consequences

A width change becomes a record.

### Risks and Mitigations

None beyond the order record's.

### Failure Modes

- A widened field overruns the payload. Visible: decode error.

## Implementation Plan

### Prerequisites

None.

### Minimum Viable Validation

Assert `HeaderLen` against the encoded size.

### Phase 1: Code Implementation

#### Step 1: State the widths

Write the contract.

## Validation

### Testing Strategy

One size assertion.

## Finalization Gate

See `gate.md`.

## References

- 0010-frame-header-labelled
