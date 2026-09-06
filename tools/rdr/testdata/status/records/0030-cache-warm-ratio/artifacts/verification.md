# Verification — cache warm ratio

## Phase 3a — CoVe

Every REQ-N in req-list.md was re-read against the coverage row that
claims it; each test asserts the behaviour the REQ names.

## Verdict — clean

## Phase 3b — Adversarial

### ADV-1 — a cold tier reports a warm ratio of zero, not NaN

The divisor is guarded, so an unwarmed tier reads 0.0 rather than
dividing by an empty sample count.
