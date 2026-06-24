---
name: rdr-propose
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: Use to move a seeded RDR from problem statement to a chosen approach with alternatives weighed and premortemed (e.g. "propose an approach for RDR 46", "$rdr-propose 0046" in Codex, "/rdr-propose 0046" in Claude). Runs Stage 2 of the RDR flow. Also the front-half resume point — its freshness check re-validates a seed that sat idle. Pairs with $rdr-refine in Codex or /rdr-refine in Claude (next) and $rdr-seed or /rdr-seed (back).
---

# rdr-propose — Stage 2 (Propose)

Move from a problem statement to a *chosen approach* with the seriously-considered
alternatives recorded and rejected, so the draft has one coherent solution for
Refine to tighten. Doubles as the **front-half resume point**: its freshness check
re-validates a seed that sat idle and went stale.

## Usage

```
Codex: $rdr-propose <NNNN>
Claude: /rdr-propose <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `RDR_PATH`.
2. **Run the prompt** [`02-propose.prompt.md`](02-propose.prompt.md).
   Propose owns **selection** (which approach wins, read from prior art), not
   **verification** (deep spikes / full corpora) — that stays at Resolve, by
   design. The split is depth, not avoidance. It runs the freshness check (step 0),
   then reads prior art, enumerates 2–4 grounded approaches, recommends one,
   premortems it, and writes Proposed Solution / Alternatives / Decision Rationale +
   a `Pending` Critical Assumptions list into the draft. **Retrieval-first:** prior
   art is read *before* approaches are named (an LLM that enumerates first anchors
   on its training prior), with a **cite-check** — every prior-art claim the choice
   rests on is quoted/anchored from source, not paraphrased (the guard against the
   inverted-citation defect that overturned an approach three stages late). No prior
   art found → an explicit `⚠ no prior-art coverage` line. **`large`/`foundational`
   also build a scored Questions-Options-Criteria matrix** so the choice falls out
   of an explicit comparison, not prose.

## Review gate (Stage `02-propose.md`)

- The seed was still current (step 0 ran; stale refs/scope fixed; any prior "retained
  as history" note acted on and deleted).
- Real alternatives weighed (not one option or strawmen).
- Prior art read *before* the approaches (grounds the set; `⚠ no prior-art coverage`
  if none), and the choice's prior-art claims are quoted/anchored, not paraphrased.
- `large`/`foundational`: the choice falls out of a scored Q-O-C matrix, not prose.
- The chosen approach solves the *user's* problem; the premortem ran and the choice
  survived it.
- Load-bearing assumptions named (Pending is fine) — Stage 4's worklist.
- Technical Design proportionate — enough to commit, not a full implementation.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `propose` first (see the rdr-common table).
- `Next: /rdr-refine NNNN` — remove contradiction, redundancy, bloat.
- A surfaced contradiction is a real design hole → iterate this stage (re-run
  `/rdr-propose NNNN`) before refining.
- `/rdr-status NNNN` to re-orient.
