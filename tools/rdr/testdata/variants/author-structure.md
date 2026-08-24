# Recommendation 0105: Frame Reader Fork Policy

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-08-05
- **Status**: Rejected (scope shipped as a one-line pin in the writer)
- **Type**: Architecture
- **Profile**: standard — one contract, the pin
- **Priority**: Low
- **Related Issues**: tracker#450
- **Predecessors**: 0104-frame-version-negotiation

## Problem Statement

Whether the project may carry a fork of a reader was never written down.

## Critical Assumptions

- **A1 [No reader is forked today]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: The module graph names both readers at their upstream
    paths.
  - **If wrong**: The policy is already broken and this record documents
    rather than decides.

## Proposed Solution

### Approach

Write the policy down: no forks; pin the writer instead.

### Why not a vendored copy

A vendored copy is a fork with a different directory name.

#### The maintenance argument

Every upstream fix must be re-applied by hand.

#### The trust argument

A copy drifts from what the readers' own tests cover.

### Why not a wrapper

A wrapper cannot change what the reader accepts.

### Technical Design

#### Normative Contracts

```normative
const OldestSupportedVersion = 1
```

### Decision Rationale

A pin costs one line; a fork costs a maintainer.

## Alternatives Considered

### Alternative 1: Carry a fork

**Description**: Fork the reader and add the hook.

**Pros**:

- Solves the problem today.

**Cons**:

- Someone owns the fork forever.

**Reason for rejection**: The maintenance argument above.

### Briefly Rejected

- **Ask upstream and wait**: Correct, but not a policy.

## Context

### Background

0104 was parked for want of this policy.

### Technical Environment

The writer, both readers, and the module graph.

## Research Findings

### Investigation

Counted the forks the project carries: zero.

### Key Discoveries

- **Verified** — no forks in the module graph.

## Trade-offs

### Consequences

- **Positive**: The policy is written down.
- **Negative**: 0104 stays parked.

### Risks and Mitigations

- **Risk**: The pin is forgotten when upstream moves.
  **Mitigation**: 0104's revisit trigger names it.

### Failure Modes

- **Visible**: A newer frame reaches an older reader and fails to decode.
- **Silent (guarded)**: None; every version mismatch is a decode error.
- **Recovery**: Lower the pin.
- **Diagnosis**: The decode error names the version byte.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified

### Minimum Viable Validation

1. Set the pin to 1.
2. Encode a frame.
3. Both readers decode it.

### Phase 1: Code Implementation

#### Step 1: Add the pin

- **Risk**: The pin is set too high.
  **Mitigation**: MVV decodes on both readers.

### Cross-Cutting Concerns

- **Versioning**: The pin is the version policy.
- **Determinism**: Not affected.
- **Forward-compat**: Owned by 0104 when it re-opens.

## Validation

### Testing Strategy

1. **Scenario**: A frame at the pinned version.
   **Expected**: Both readers decode it.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- The module graph.

## Appendix A — Reader Catalog

### Reader 1

- **Category**: upstream, unmodified.

### Reader 2

- **Category**: upstream, unmodified.
