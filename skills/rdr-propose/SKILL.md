---
name: rdr-propose
argument-hint: <NNNN>
description: Use to move a seeded RDR from problem statement to a chosen approach with alternatives weighed and premortemed (e.g. "propose an approach for RDR 46", "/rdr-propose 0046"). Runs Stage 2 of the RDR flow. Also the front-half resume point — its freshness check re-validates a seed that sat idle. Pairs with /rdr-refine (next) and /rdr-seed (back).
---

# rdr-propose — Stage 2 (Propose)

Move from a problem statement to a *chosen approach* with the seriously-considered
alternatives recorded and rejected, so the draft has one coherent solution for
Refine to tighten. Doubles as the **front-half resume point**: its freshness check
re-validates a seed that sat idle and went stale.

## Usage

```
/rdr-propose <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `RDR_PATH`.
2. **Run the prompt** [`02-propose.prompt.md`](02-propose.prompt.md).
   It reads only the **Domain priors** section of `$RDR_RESOURCES` (which approaches
   are on the table) — not the full verification corpora; Resolve does that. It runs
   the freshness check (step 0), enumerates 2–4 distinct approaches, recommends one,
   premortems it, and writes Proposed Solution / Alternatives / Decision Rationale +
   a `Pending` Critical Assumptions list into the draft. **Breadth bias:** approaches
   are grounded in the priors' prior art, not the model's training prior — if the
   priors name no competitor for the problem class, a `small`/`mid` RDR carries a
   `⚠ no prior-art coverage` line into Investigation and a `large`/`foundational`
   one does a forced bounded prior-art search before fixing the option set.

## Review gate (Stage `02-propose.md`)

- The seed was still current (step 0 ran; stale refs/scope fixed; any prior "retained
  as history" note acted on and deleted).
- Real alternatives weighed (not one option or strawmen).
- Breadth bias honored — approaches grounded in prior art, or the thin-priors gap
  marked (`⚠` line on `small`/`mid`, forced search cited on `large`/`foundational`).
- The chosen approach solves the *user's* problem; the premortem ran and the choice
  survived it.
- Load-bearing assumptions named (Pending is fine) — Stage 4's worklist.
- Technical Design proportionate — enough to commit, not a full implementation.

## Next step (rdr-common §next-step)

- `Next: /rdr-refine NNNN` — remove contradiction, redundancy, bloat.
- A surfaced contradiction is a real design hole → iterate this stage (re-run
  `/rdr-propose NNNN`) before refining.
- `/rdr-status NNNN` to re-orient.
