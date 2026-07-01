---
name: rdr-reconcile
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: 'Use to close open spikes and round-disturbed assumptions before lock. Runs Stage 6 and routes back if evidence refutes the draft. Trigger for reconcile RDR, $rdr-reconcile, or /rdr-reconcile.'
---

# rdr-reconcile — Stage 6 (Reconcile Spikes & Assumptions)

Close the gap the review rounds opened. Stage 4 verified the assumptions as the
draft stood then; Stages 5–6 mutated it. This gate runs **once, after all rounds**,
forcing every open spike and every round-disturbed assumption to a terminal state
**before** Finalize. It is a forced gate, not a checkbox.

## Usage

```
Codex: $rdr-reconcile <NNNN>
Claude: /rdr-reconcile <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `$RDR_ENV`, `RDR_PATH`, `{SPIKE_DIR}`. Have the Pre-Lock
   needs-verification list(s) ready to paste.
   - **Preflight Stage 5 completeness before reconciling.** Re-read `Profile` and
     **Normative Contracts**. If a required profile lens is missing, or `mid`/`large`
     Determinacy fired without `evidence/repeatability/run-1.md` + `diff.md` or
     `determinacy: n/a - <reason>`, stop: `NOT RECONCILED — return to Stage 5`,
     with the exact next `/rdr-prelock NNNN ...` command. `run-2`/`run-3` beside a
     `run-1.md` whose `variant:` header reads `lite` (not `full (escalated: …)`) is a
     variant mismatch (full ran where lite was owed) — flag it (Caveats), don't
     silently pass it as done.
2. **Run the prompt** [`06-reconcile.prompt.md`](06-reconcile.prompt.md);
   paste the Pre-Lock list(s) where its `<paste the list(s)>` marker is. It builds the
   open set from four sources (Pre-Lock list, still-Pending assumptions, named-but-
   unrun spikes, the exactness-word delta), forces ONE disposition each, and **writes
   it into the RDR's Critical Assumptions section** (not just a report). Delegate
   spike runs / corpus reads to a `Task` sub-agent returning verdict + evidence
   pointer (§return-packet) (rdr-common §delegation).
   - **Absorption-audit delegation (mined — recurs verbatim).** To build source 3 +
     confirm the rounds were folded in, spawn one `Task` sub-agent over the lens
     output dirs that exist for this slug
     (`{EVIDENCE_DIR}/{3amigo,critique,repeatability,cove}/<RDR_SLUG>/`): "report, per
     round, whether every finding was absorbed into the current RDR or survives as
     residue; list each unabsorbed finding + the spike/assumption it implies." It
     returns the residue list, not the round files. The main agent writes the
     dispositions into the RDR.
3. **Two hard rules** the prompt enforces: a spike that **refutes** an assumption the
   RDR relies on is a BLOCKER that re-opens the RDR (never a wording fix); an
   MVV-critical spike **cannot** defer past lock.

## Review gate (Stage `06-reconcile.md`)

- The open set is complete (all four sources, especially claims silently introduced
  by a fix pass).
- Any refutation → verdict NOT RECONCILED with a named return stage, not a quietly
  edited assumption.
- DOWNGRADED items genuinely survivable; spikes actually run with captured output.
- Dispositions landed **in the RDR**, not only the report (Stage 7 reads the RDR).

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `reconcile` first (+ a separate reconcile-evidence commit; see the rdr-common table).
- Verdict **RECONCILED** → `Next: /rdr-finalize NNNN`.
- Verdict **NOT RECONCILED** → back to the named stage: `/rdr-propose NNNN` (approach),
  `/rdr-refine NNNN`, or `/rdr-resolve NNNN` (re-resolve).
- `/rdr-status NNNN` to re-orient.
