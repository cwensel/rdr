---
name: rdr-resolve
argument-hint: <NNNN>
description: Use to verify an RDR's critical assumptions against reality before any pre-lock review (e.g. "resolve assumptions for RDR 46", "/rdr-resolve 0046"). Runs Stage 4 of the RDR flow — research + spikes + reuse audit, flipping assumptions to Verified with Method + Evidence. Self-detects scoped re-entry from the RDR Status line. Pairs with /rdr-prelock (next) and /rdr-propose (back, if research refutes the approach).
---

# rdr-resolve — Stage 4 (Resolve Assumptions)

Verify every Critical Assumption against source, live spikes, prior art, and
standards so the pre-lock rounds critique a *true* document. The load-bearing
front-half stage. Running a pre-lock round before this one wastes the round.

## Usage

```
/rdr-resolve <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_ENV`, `$RDR_RESOURCES`, `RDR_PATH`, `RDR_SLUG`, `{SPIKE_DIR}`.
2. **Run the prompt** [`04-resolve.prompt.md`](04-resolve.prompt.md)
   with those bound. Have a live spike target reachable. The prompt is read-heavy —
   delegate corpus searches / reuse audits / dependency spelunks to a sub-agent that
   returns *verdict + evidence pointer*, not raw hits (rdr-common §delegation; spawn
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
- Spikes actually ran (command + captured output under `{SPIKE_DIR}/<slug>/`).
- The reuse audit ran against the `$RDR_ENV` reuse-audit paths.
- No research finding contradicts the approach — if one does, that's **not** a
  citation fix.

## Next step (rdr-common §next-step)

- Research refuted the approach or surfaced existing capability → **back** to
  `/rdr-propose NNNN` (or `/rdr-refine NNNN`); rework, then re-run this.
- Assumptions verified → forward. This stage **sets the `Profile` Metadata
  field** (the routing latch) from the contract count; the next pointer reads
  that field — it does not re-derive size (`$RDR_FLOW_HOME/flow/README.md` matrix):
  - `Profile: small` → skip Pre-Lock: `Next: /rdr-reconcile NNNN`.
  - `Profile: mid`+ → `Next: /rdr-prelock NNNN 3amigo` (then more lenses per profile).
- `/rdr-status NNNN` to re-orient.
