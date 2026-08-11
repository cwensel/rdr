---
name: rdr-propose
metadata:
  argument-hint: <NNNN> [--model-ceiling <model>] [--commit | --no-commit]
description: 'Use to move a seeded RDR from problem statement to chosen approach with alternatives weighed. Runs Stage 2 and re-validates stale seeds. Trigger for propose approach, $rdr-propose, or /rdr-propose.'
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
   to bind `$RDR_RESOURCES`, `RDR_PATH`. Bind `{EVIDENCE_DIR}` =
   `<RDR_EVIDENCE>/<RDR_SLUG>/evidence/` (§evidence) — the prompt's
   `research/` and `propose-premortem/` outputs land under it.
   **Model-adequacy fork (§model-ceiling).** On a heavy RDR, run the fork
   *now, before the prompt* — nothing is written yet, so cancel is free.
   Main-context propose cannot bump its own model; the fork's pause is where
   the user cancels and restarts stronger (or routes to `/rdr-joint-propose
   --model-ceiling`). A "continue" answer lands on `Deviations:`.
2. **Run the prompt** [`02-propose.prompt.md`](02-propose.prompt.md).
   Propose owns **selection** (which approach wins, read from prior art), not
   **verification** (deep spikes / full corpora) — that stays at Resolve, by
   design. The split is depth, not avoidance. It runs the freshness check (step 0),
   then reads prior art, enumerates 2–4 grounded approaches, recommends one,
   premortems it (`small`/`mid`: one paragraph; `large`/`foundational`: a
   draft-free critic sub-agent briefed without the RDR's justifying prose,
   writing `{EVIDENCE_DIR}propose-premortem/critic.md`), records the
   `Premortem:` verdict line in Decision Rationale, and writes Proposed
   Solution / Alternatives / Decision Rationale +
   a `Pending` Critical Assumptions list into the draft. **Retrieval-first:** prior
   art is read *before* approaches are named (an LLM that enumerates first anchors
   on its training prior), with a **cite-check** — every prior-art claim the choice
   rests on is quoted/anchored from source, not paraphrased (the guard against the
   inverted-citation defect that overturned an approach three stages late). No prior
   art found → an explicit `⚠ no prior-art coverage` line. **`large`/`foundational`
   also build a scored Questions-Options-Criteria matrix** so the choice falls out
   of an explicit comparison, not prose.
3. **Run the joint-decision check** (the prompt's final step — it needs the
   freshly written draft): grep every open peer under `$RDR_RECORDS` for this
   RDR's modify-anchors / contract literals; a shared whole-token hit with zero
   cross-citation FIRES (mechanics live in the prompt step). Either way write
   the `Joint-check:` line into Decision Rationale — unwritten reads as *never
   ran*. A fire is a **human-judgment fork**: emit §stop-packet
   `stopped:joint-decision:…`, put both answers and the peer RDRs to the user,
   and **wait** — don't close the stage as done.

## Review gate (Stage `02-propose.md`)

- The seed was still current (step 0 ran; stale refs/scope fixed; any prior "retained
  as history" note acted on and deleted).
- Real alternatives weighed (not one option or strawmen).
- Prior art read *before* the approaches (grounds the set; `⚠ no prior-art coverage`
  if none), and the choice's prior-art claims are quoted/anchored, not paraphrased.
- `large`/`foundational`: the choice falls out of a scored Q-O-C matrix, not prose.
- The chosen approach solves the *user's* problem; the premortem ran in its
  profile's form (paragraph, or the draft-free critic at `large`/`foundational`)
  and the `Premortem:` verdict line is recorded in Decision Rationale.
- The joint-decision check ran against all open peers and its `Joint-check:`
  verdict line is recorded in Decision Rationale (absent line = did not run —
  re-run it; a skipped gate item must not read as a passed one). Any fire was
  paused on and answered by the user, not advanced over. In a Cluster, the
  bridge sub-check's (a) skip-to-end-state / (b) `Transient`-marker choice was
  surfaced and answered when its cues were present — an unanswered choice does
  not advance.
- Load-bearing assumptions named (Pending is fine) — Stage 4's worklist.
- Technical Design proportionate — enough to commit, not a full implementation.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `propose` first (see the rdr-common table).
- `Next: /rdr-refine NNNN` — remove contradiction, redundancy, bloat.
- Sibling seeds still unproposed? Recommend *their* `/rdr-propose` before this
  RDR refines — each successive propose greps peers still in Draft, keeping any
  joint-decision fire in the free-switching window (stages/02-propose.md,
  batch ordering). `/rdr-joint-propose <NNNN …>` orchestrates the whole cohort
  dependency-ordered.
- A surfaced contradiction is a real design hole → iterate this stage (re-run
  `/rdr-propose NNNN`) before refining.
- A joint-decision fire stops the stage (`Gate: stopped:joint-decision`): Next
  is the user's answer, not a command — hoist to the consumer's
  umbrella-decision record (e.g. an RFD), cite-don't-restate, or declare the
  shared interface in both RDRs. Never `/rdr-refine` with the fire open.
- A bridge sub-check answer routes the handoff: choice (a) skip to end-state →
  iterate this stage (re-run `/rdr-propose NNNN` — the plan changed); choice (b)
  → the `Transient` marker is recorded in the bridge's contract block, proceed
  to `/rdr-refine NNNN`.
- `/rdr-status NNNN` to re-orient.
