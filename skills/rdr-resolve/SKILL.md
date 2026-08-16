---
name: rdr-resolve
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: 'Use to verify critical RDR assumptions against source, spikes, prior art, and reuse audits. Runs Stage 4 before pre-lock review. Trigger for resolve assumptions, $rdr-resolve, or /rdr-resolve.'
---

# rdr-resolve — Stage 4 (Resolve Assumptions)

Verify every Critical Assumption against source, live spikes, prior art, and
standards so the pre-lock rounds critique a *true* document. The load-bearing
front-half stage. Running a pre-lock round before this one wastes the round.

## Usage

```
Codex: $rdr-resolve <NNNN>
Claude: /rdr-resolve <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_ENV`, `$RDR_RESOURCES`, `RDR_PATH`, `RDR_SLUG`, `{SPIKE_DIR}`.
2. **Run the prompt** [`04-resolve.prompt.md`](04-resolve.prompt.md)
   with those bound. Have a live spike target reachable. The prompt is read-heavy —
   delegate corpus searches / reuse audits / dependency spelunks to a sub-agent that
   returns *verdict + evidence pointer* (§return-packet), not raw hits (rdr-common §delegation; spawn
   with the `Task` tool).
   - **Fan-out heuristic (mined):** group the **Source-Search** assumptions into one
     sub-agent (multi-pass, one verdict each) and give each **Spike** assumption its
     own sub-agent (it needs the live target). Don't spawn one agent per source-search
     claim — they share the same corpus/source reads. Hold only the verdicts in the
     main context; author the `Status: Verified` edits here, not in the sub-agent.
3. **Self-detected re-entry (no flag).** The prompt reads the RDR `Status:` line: a
   `Draft [revised from Final …; re-verify <IDs>]` qualifier scopes the run to the
   listed assumption IDs (+ anchors the demotion touched), carrying the rest forward
   as Verified; a bare `Draft` verifies every assumption cold. Do not pass a resume
   flag — the RDR's state drives it.

## Review gate (what the human checks — Stage `04-resolve-assumptions.md`)

- Every load-bearing assumption actually `Verified`, Evidence not self-reference and
  not bare `Docs Only`.
- Spikes actually ran (command + captured output under `{SPIKE_DIR}`).
- I/O-expressible assumptions rendered as I/O pairs and put to the user in one
  consolidated approve/reject round; approved I/O pairs named as normative fixtures
  in the RDR body — an unapproved I/O pair is not Evidence.
- The reuse audit ran against the `$RDR_ENV` reuse-audit paths.
- No research finding contradicts the approach — if one does, that's **not** a
  citation fix.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `resolve` first (+ a separate spike-evidence commit if a spike wrote; row in rdr-commit-map.md).
- Research refuted the approach or surfaced existing capability → **back** to
  `/rdr-propose NNNN` (or `/rdr-refine NNNN`); rework, then re-run this.
- Assumptions verified → forward. This stage **sets the `Profile` Metadata
  field** (the routing latch) from the contract count, then the Seam Lineage
  accretion floor, which outranks it (the prompt owns both). Take the next
  pointer from the field it just wrote via **rdr-common §lens-row** — the first
  lens is profile-dependent (`foundational` leads with `cove`, not `grounding`),
  so read the row, never default.
- `/rdr-status NNNN` to re-orient.
