# Recommendation 0104: Frame Version Negotiation

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-04
- **Status**: Deferred — **no solution decided; must be revisited** (2026-08-04).
  Propose did not select an approach because every path requires a
  fork of the reader the project rules out. **Revisit trigger:** re-open
  if the upstream reader exposes a version hook.
  <!-- Draft | Final | Implemented | Reverted | Abandoned | Superseded | Demoted -->
- **Type**: Architecture
- **Profile**: foundational — would lock the negotiation
  grammar across the writer and both readers
- **Priority**: Medium
- **Related**: tracker#440, tracker#441
- **Predecessors**: 0001-frame-length-prefix-width,
  0003-frame-checksum-placement,
  0004-frame-checksum-algorithm
- **Overrides**: 0004-frame-checksum-algorithm's assumption that the
  version byte is fixed; this record would make it negotiated.
- **Seam Lineage**: `frame::Encode` — 4th point-fix; trail: a1b2c3d +
  tracker#204, tracker#331, tracker#402, tracker#440.
  Accretion disposition: the 4 point-fixes at `frame::Encode` ARE one
  missing design decision — a version grammar — which this record
  proposes; cite: 0004-frame-checksum-algorithm.
  - **Disposition owner**: this record.
- **Release scope**: not in the next minor
- **Referenced by**: 0105-frame-reader-fork

## Problem Statement

The writer cannot learn which frame versions a reader accepts.

## Critical Assumptions

- **A1 [The readers expose no version hook today]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: Neither reader has a call that returns accepted
    versions.
  - **If wrong**: Negotiation is a call away and no fork is needed.

## Proposed Solution

### Approach

No approach was selected; see Status.

### Technical Design

#### Normative Contracts

No contract is proposed while the record is parked.

### Decision Rationale

Every path forks a reader.

## Alternatives Considered

### Briefly Rejected

- **Fork the reader**: The project rules out carrying a fork.

## Context

### Background

A writer upgrade broke an older reader in the field.

### Technical Environment

The frame codec and both readers.

## Research Findings

### Investigation

Looked for a version hook in both readers.

### Key Discoveries

- **Documented** — no hook exists.

## Trade-offs

### Consequences

- Nothing changes until upstream moves.

### Risks and Mitigations

- **Risk**: Another field break before upstream moves.
  **Mitigation**: The writer pins the oldest version until then.

### Failure Modes

An older reader rejects a newer frame, visible as a decode error.

## Implementation Plan

### Prerequisites

- [ ] Upstream exposes a version hook

### Minimum Viable Validation

1. Re-open this record when the trigger fires.

### Phase 1: Code Implementation

#### Step 1: None while parked

## Validation

### Testing Strategy

1. **Scenario**: None while parked.
   **Expected**: None.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- Both readers' public surface.
