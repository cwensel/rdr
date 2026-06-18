# Grounding Step-0 — Codebase Claim Sweep (S-00)

**Use when**: any RDR that **locks a contract** (`mid` and up). The cheap oracle
half of [cove](4-cove.md), run standalone at `mid` where the full cove persona
pass is overhead but an ungrounded contract frame is still expensive. At
`foundational` it is Step 0 *inside* cove (which runs first), not run separately.

**What it uniquely catches**: a contract frame that is internally coherent but
**false against the codebase** — a new discriminator/heuristic invented when a
sibling path already holds the signal, or a cited symbol that doesn't resolve.
One deterministic source-reading sweep, not a review loop.

**Cost**: 5–10 min. Single-model is fine (a check, not a judgment).

## Prompt

```text
Sweep the RDR at {RDR_PATH} for every claim that asserts a codebase fact —
each cited `path::Symbol` (especially Critical Assumptions with Method: Source
Search, and Normative Contract symbols), each "no existing X does Y", each "a
sibling/adjacent path already does Z", each "the only place this happens is …".
For EACH, read the actual source on `main` and record CONFIRMED (paste the
greppable `path::Symbol` or line), REFUTED (paste what you found instead), or
NOT-FOUND (symbol doesn't resolve). Do not take the RDR's word for a codebase
fact.

Then the inverse the RDR did not check: if the approach adds a NEW
discriminator, heuristic, switch case, or identity rule, grep whether a
sibling/adjacent path already makes that decision — name it (`path::Symbol`) or
record "searched, none exists".

Write each REFUTED / NOT-FOUND, and each new-rule-with-an-existing-sibling, as a
finding to {FLOW_DIR}/findings.md (the dispatcher binds {FLOW_DIR} — do not
re-derive it). A CONFIRMED claim needs no finding. Report nothing else.
```

## Expected signal

- **Healthy** — every codebase claim resolves CONFIRMED (cited), or a
  REFUTED/NOT-FOUND is surfaced. A false assumption found here is the
  highest-value output — cheaper to reopen the frame now than post-lock.
- **Unhealthy** — answers cite the RDR instead of source. Re-run insisting on a
  greppable cite per claim; escalate to a human grep if it won't read source.

## Source

The oracle half of [cove](4-cove.md): a deterministic codebase check, run wherever
a contract is locked because that is where a wrong frame is most expensive. Evidence
base in [`RESEARCH.md`](../../RESEARCH.md) §2 (Grounding).
