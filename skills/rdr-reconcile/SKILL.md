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

1. Read [`rdr-common.md`](rdr-common.md) **whole, with the Read tool** (it exceeds the 30KB
   Bash cap — `cat` truncates and costs a retry; never `sed`/`grep` §-slices); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `$RDR_ENV`, `RDR_PATH`. Bind the report dir and the spikes
   tree in two calls (§evidence — ask for the dir, never compose one):
   `eval "$("$RDR_HOME/bin/rdr" paths --lens reconcile --next-iter <NNNN>)"` →
   write the report to `$ITER_DIR` (at `ITER=1` it *is* the base; re-runs land in
   `iter-N/`; `ITER_FOUND`/`ITER_NOTE` say what was on disk), and
   `eval "$("$RDR_HOME/bin/rdr" paths --tree spikes <NNNN>)"` → `$EVIDENCE_DIR` is
   `{SPIKE_DIR}`. Have the Pre-Lock needs-verification list(s) ready to paste.
   A `[routed back from … @reconcile]` Status qualifier names this stage as owing
   the work: close what it names and clear the qualifier (rdr-common §rdr-write
   *Receiving a route-back*) — nothing else will.
   - **Preflight Stage 5 completeness before reconciling** — run **§lens-row**'s
     call; don't re-read the row.

     Anything but `/rdr-reconcile` means Stage 5 is not done: stop with
     `NOT RECONCILED — return to Stage 5` and `emit.next` as the exact command
     (`stopped:no-profile` is the same stop — the latch was never written).
     Then ask **`--outcome critique`** and **`--outcome repeatability`** for the two
     lenses whose completion a folder cannot show: each answers `none` when finished,
     names the lens again when not (the same stop), and carries the caveat to copy
     into Caveats — a single-model fallback, an unstamped pass, or a variant mismatch
     (full ran where lite was owed). The `mid`/`large` **Determinacy** add-on is
     `--outcome repeatability`'s answer too (chain `resolve:determinacy` once):
     `stopped:determinacy-trigger-unjudged` means the `Determinacy:` line is unwritten
     in Normative Contracts, and a route to run 1 is the same stop.
2. **Run the prompt** [`06-reconcile.prompt.md`](06-reconcile.prompt.md);
   paste the Pre-Lock list(s) where its `<paste the list(s)>` marker is. It builds the
   open set from four sources (Pre-Lock list, still-Pending assumptions, named-but-
   unrun spikes, the exactness-word delta), forces ONE disposition each, and **writes
   it into the RDR's Critical Assumptions section** (not just a report). Delegate
   spike runs / corpus reads to a spawned sub-agent returning verdict + evidence
   pointer (§return-packet) (rdr-common §delegation). While it runs, the parent
   re-does nothing it delegated and waits by ending the turn (§delegation).
   - **Absorption-audit delegation (mined — recurs verbatim).** To build source 3 +
     confirm the rounds were folded in, spawn one sub-agent over the lens
     output dirs that exist for this slug — `rdr paths --lens <lens> <NNNN>` for
     `3amigo`, `critique`, `repeatability`, `cove` (rdr-common §evidence; `ls`
     the answer, skip a lens whose dir is absent): "report, per
     round, whether every finding was absorbed into the current RDR or survives as
     residue; list each unabsorbed finding + the spike/assumption it implies." It
     returns the residue list, not the round files. The main agent writes the
     dispositions into the RDR.
3. **Two hard rules** the prompt enforces: a spike that **refutes** an assumption the
   RDR relies on is a BLOCKER that re-opens the RDR (never a wording fix — and a
   refutation with an in-hand repair still passes through the consult and the
   user's accept); an MVV-critical spike or assumption **cannot** defer past lock.

## Review gate (Stage `06-reconcile.md`)

- The open set is complete (all four sources, especially claims silently introduced
  by a fix pass).
- Any refutation → verdict NOT RECONCILED with a named return stage, not a quietly
  edited assumption.
- DOWNGRADED items genuinely survivable; spikes actually run with captured output.
- Dispositions landed **in the RDR**, not only the report (Stage 7 reads the RDR).

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `reconcile` first (+ a separate reconcile-evidence commit; row in rdr-commit-map.md).
- Verdict **RECONCILED** → `Next: /rdr-finalize NNNN`.
- Verdict **NOT RECONCILED** → back to the named stage: `/rdr-propose NNNN` (approach),
  `/rdr-refine NNNN`, or `/rdr-resolve NNNN` (re-resolve).
- `/rdr-status NNNN` to re-orient.
