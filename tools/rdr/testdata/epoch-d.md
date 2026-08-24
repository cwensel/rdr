# Recommendation 0004: Frame Checksum Algorithm

> Revise during planning; lock at implementation.
> If wrong, abandon code and iterate RDR.

## Metadata

- **Date**: 2026-07-30
- **Status**: Final [joint decision → 0005-frame-version-handshake §
  A2: whether the version byte is covered by the checksum]
- **Type**: Architecture
- **Profile**: foundational — locks the checksum grammar across the writer
  and both readers
- **Priority**: High
- **Related Issues**: tracker#402
- **Predecessors**: 0001-frame-length-prefix-width,
  0003-frame-checksum-placement
- **Overrides**: 0003-frame-checksum-placement retires the compatibility
  shim its Transient marker scheduled; the placement itself stands.
- **Seam Lineage**: `frame::Encode` — 3rd point-fix; trail: a1b2c3d +
  tracker#204, tracker#331, tracker#402.
  Accretion disposition: the 3 point-fixes at `frame::Encode` are NOT one
  missing design decision because each locks a distinct field of the
  header — width, position, algorithm; cite: 0003-frame-checksum-placement.
- **Cluster**: 0005-frame-version-handshake

## Problem Statement

The checksum algorithm was never named. The writer and the readers happen
to agree today because they share a helper, not because anything requires
them to.

## Critical Assumptions

- **A1 [The chosen checksum detects single-bit errors in frames up to the
  declared maximum]**
  - **Status**: Verified
  - **Method**: Derivation
  - **Evidence**: The polynomial's Hamming distance is 4 for messages
    below its stated length bound, which exceeds the declared maximum
    frame size; shown inline in Research Findings.
  - **If wrong**: A corrupted frame verifies clean and the corruption
    reaches the caller as valid data.
- **A2 [The checksum helper is the only implementation in the tree]**
  - **Status**: Verified
  - **Method**: Source Search
  - **Evidence**: `hash::Sum32` is the single definition; every call site
    resolves to it.
  - **If wrong**: A second implementation drifts and the two sides
    disagree exactly as they do today.
- **A3 [The version byte's coverage is settled by the sibling record]**
  - **Status**: Pending
  - **Method**: Peer RDR + Source Search
  - **Evidence**: 0005-frame-version-handshake § A2 owns the question; its
    answer determines whether the checksum's input starts at byte 0 or
    byte 1.
  - **If wrong**: The two records lock incompatible checksum inputs and
    every cross-version frame fails.

## Proposed Solution

### Approach

Name the polynomial in the contract and delete the shared-helper
coincidence as the source of agreement.

### Technical Design

The checksum is a 32-bit CRC over the payload, with the polynomial stated
in the Normative Contracts rather than inherited from a helper.

#### Normative Contracts

```normative
const ChecksumPolynomial uint32 = 0x1EDC6F41
func Sum32(payload []byte) uint32
```

- The checksum is a 32-bit CRC using the polynomial above.
- The checksum covers the payload only. Whether it also covers the version
  byte is the open joint decision named on this record's Status.

#### Load-Bearing Decisions

- **Wire / byte format** — the checksum is four bytes, unsigned,
  big-endian, at offset 4.
- **Naming** — `ChecksumPolynomial`, rejected: `CRCPoly`, which names the
  algorithm family rather than this format's choice.

### Decision Rationale

Naming the polynomial in the contract is the difference between two sides
that agree and two sides that are required to agree. Premortem: the
likeliest failure is the sibling record answering the version-byte
question the other way, which the joint-decision qualifier tracks rather
than hides. Joint-check: 0005-frame-version-handshake owns § A2.

## Alternatives Considered

### Alternative 1: Keep inheriting from the helper

**Description**: Leave the algorithm implicit and rely on the shared
helper.

**Pros**:

- No change at all.

**Cons**:

- The agreement is a coincidence, and coincidences end.

**Reason for rejection**: The record exists precisely to remove the
coincidence.

### Alternative 2: A cryptographic digest

**Description**: Use a truncated cryptographic hash instead of a CRC.

**Pros**:

- Detects adversarial modification, not only corruption.

**Cons**:

- Costs an order of magnitude more per frame for a threat this format does
  not defend against.

**Reason for rejection**: Wrong threat model; integrity here means
corruption, not tampering.

### Briefly Rejected

- **A 16-bit CRC**: Halves the header cost and quarters the useful Hamming
  distance.

## Context

### Background

Found while reading the writer and a reader side by side and noticing that
neither states what it computes.

### Technical Environment

The frame codec, both readers, and the hash package.

## Research Findings

### Investigation

Derived the Hamming distance bound for the candidate polynomial and
checked every call site of the existing helper.

### Key Discoveries

- **Verified** — one helper definition, no second implementation.
- **Documented** — the polynomial's distance bound, derived inline.

## Trade-offs

### Consequences

- The algorithm is a stated contract rather than an accident.
- The version-byte question is now explicit and open rather than
  unasked.

### Risks and Mitigations

- **Risk**: The sibling answers the version-byte question the other way.
  **Mitigation**: The joint-decision qualifier keeps the obligation open
  until the home answers; this record re-checks before it implements.

### Failure Modes

If the two records lock different checksum inputs, every cross-version
frame fails verification. It surfaces as a total decode failure between
versions, which is loud rather than silent.

## Implementation Plan

### Prerequisites

- [ ] All Critical Assumptions verified
- [ ] The joint decision on the version byte is answered

### Minimum Viable Validation

1. Encode a frame, flip one bit of the payload.
2. Decode it.
3. Decoding reports a checksum failure rather than returning the corrupted
   payload.

### Phase 1: Code Implementation

#### Step 1: State the polynomial as a constant

#### Step 2: Retire the compatibility shim

### Day 2 Operations

| Resource | List | Info | Delete | Verify | Backup |
| --- | --- | --- | --- | --- | --- |
| None created | N/A | N/A | N/A | N/A | N/A |

## Validation

### Testing Strategy

1. **Scenario**: A single-bit corruption in the payload.
   **Expected**: Decode reports a checksum failure.

## Finalization Gate

See `gate.md` for the written responses to each item below.

## References

- Hash package, `Sum32`.
- 0005-frame-version-handshake § A2.
