# Grounding Step-0 — Codebase Claim Sweep (S-00)

**Use when**: the `lens` outcome names `grounding`. The cheap oracle half of
[cove](4-cove.md), which embeds it as Step 0 (never run standalone there).

**What it uniquely catches**: a contract frame that is internally coherent but
**false against the codebase** — a new discriminator/heuristic invented when a
sibling path already holds the signal, or a cited symbol that doesn't resolve.
One deterministic source-reading sweep, not a review loop.

**Cost**: 5–10 min. Single-model is fine (a check, not a judgment).

## Prompt

```text
Scope the sweep first. Run once:

  "$RDR_HOME/bin/rdr" inspect --json --filter edges,elements {RDR_PATH}

(`--repo` defaults to `$RDR_SOURCE_REPO`, rdr-common §source-root; without a repo
root every `resolved` comes back ABSENT.) Take two lists — this is your
STARTING SET and your primary read:
  - `edges[]` where `kind=="source-anchor"` — every cited `path::Symbol`, each
    with `to` (the symbol), `line`/`line_end`, `field`, and `from` (the element
    that claims it). `resolved` is the projector's own verdict, THREE-valued:
    true = the symbol exists (CONFIRMED — cite it and move on), false =
    NOT-FOUND (a finding), ABSENT = nothing looked (no `--repo`; sweep it
    yourself, never read absent as either).
  - `elements[]`, each element's own `fields[]` where `label=="Evidence"` (not
    the top-level `fields[]`, which carries no Evidence), with `element` and
    `line_start`/`line_end`. Read those LINE spans — the one read here with no
    id, since the span is the Evidence field alone and `--select <element>`
    would return the whole assumption. The assumptions that owe source are the
    ones whose `Method` field has `Source Search` in `method.members`.

An EMPTY starting set is not a clean record. Older records state their evidence
as prose and label nothing, so they project no `Evidence` field and no
`source-anchor` edge at all. If both lists come back empty, the scoping told you
nothing — fall through and read the record whole, finding these claims by eye.
Never report a sweep as complete off an empty projection.

Then sweep. `resolved:false` and `resolved:ABSENT` anchors are the work; so are
the claims that name no symbol and so mint no edge — each "no existing X does
Y", each "a sibling/adjacent path already does Z", each "the only place this
happens is …", and each claim that a call runs before/after another, that a site
is (un)reachable, or that output does not change. Refute those at the call flow,
the producing/population site, or the output, not by opening the named symbol;
such a claim anchored by no assumption is itself a finding.

WIDEN when the ranges are not enough — an unanchored prose claim lives between
them by definition. Read the fuller record whenever the scoped spans leave a
codebase claim you cannot judge, and say in the report which spans you widened
past and why. A scope is a starting set, never a licence to stop early.

For EACH claim, read the actual source on `main` and record CONFIRMED (paste the
greppable `path::Symbol` or line), REFUTED (paste what you found instead), or
NOT-FOUND (symbol doesn't resolve). Do not take the RDR's word for a codebase
fact. If Decision Rationale carries a `Ground-sweep:` verdict (the propose-time
micro-sweep), scope to claims added or edited after propose — diff the RDR
against its propose commit; no verdict line → sweep everything.

Then the inverse the RDR did not check: if the approach adds a NEW
discriminator, heuristic, switch case, or identity rule, grep whether a
sibling/adjacent path already makes that decision — name it (`path::Symbol`) or
record "searched, none exists".

Write each REFUTED / NOT-FOUND, and each new-rule-with-an-existing-sibling, as a
finding to {EVIDENCE_DIR}/findings.md (the dispatcher binds {EVIDENCE_DIR} — do not
re-derive it). A CONFIRMED claim needs no finding. Report nothing else.
```

## Expected signal

- **Healthy** — every codebase claim resolves CONFIRMED (cited), or a
  REFUTED/NOT-FOUND is surfaced. A false assumption found here is the
  highest-value output — cheaper to reopen the frame now than post-lock.
- **Unhealthy** — answers cite the RDR instead of source. Re-run insisting on a
  greppable cite per claim; escalate to a human grep if it won't read source.
  A sweep that reports only `source-anchor` verdicts and never widened has read
  the anchors and skipped the prose claims — the ones with no symbol to anchor
  are exactly where a false frame hides.

## Source

The oracle half of [cove](4-cove.md): a deterministic codebase check, run wherever
a contract is locked because that is where a wrong frame is most expensive. Evidence
base in [`RESEARCH.md`](../../RESEARCH.md) §2 (Grounding).
