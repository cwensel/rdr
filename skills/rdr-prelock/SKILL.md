---
name: rdr-prelock
metadata:
  argument-hint: "<NNNN> <lens> [run] [--commit | --no-commit]   # lens ∈ {grounding, 3amigo, critique, repeatability, cove}; run ∈ {1,2,3,diff}"
description: 'Use to run one pre-lock lens and resolve its findings in the same pass. Lenses: grounding, 3amigo, critique, repeatability, cove. Trigger for pre-lock review, $rdr-prelock, or /rdr-prelock.'
---

# rdr-prelock — Stage 5+6 (Pre-Lock Review **and** Resolve, per lens)

Catch the defect classes a template slot can't absorb — PM/UX gaps, time-shifted
failures, under-specified signatures, internal silences — **and fix them in the
same invocation**. One lens = one cycle: run the lens, ground each finding, edit
the draft, loop to convergence (or flapping). Dispatches into the lens prompts in
`pre-lock/` for the review half (doesn't redefine them) and runs
`05-prelock-resolve.prompt.md` for the fix half.

## Usage

```
Codex: $rdr-prelock <NNNN> <lens> [run]        # lens in {grounding, 3amigo, critique, repeatability, cove}
Claude: /rdr-prelock <NNNN> <lens> [run]       # lens in {grounding, 3amigo, critique, repeatability, cove}
```

`run` (`1` | `2` | `3` | `diff`) applies to `repeatability` only — 1–3 generates
that one run, `diff` compares them. Reject an unknown lens with
`stopped:bad-lens:<value>`.

**Pick the lens by profile** (`$RDR_HOME/stages/README.md` matrix): mid →
`grounding 3amigo`; large → `grounding 3amigo critique`; foundational →
`cove 3amigo critique repeatability` (cove embeds the grounding sweep as its
Step 0, so it leads and `grounding` is not run separately). A small RDR runs
**no** lens (skip to `/rdr-reconcile`). Run lenses in that order — grounding/cove
first (ground the frame before the personas debate it).

**Precondition.** Stage 4 must have verified the assumptions — running a lens on
unverified claims wastes it. If Critical Assumptions are still `Pending` without a
plan, stop with `stopped:assumptions-unverified` and point at `/rdr-resolve NNNN`.

## The cycle (3amigo · critique · cove)

One invocation runs the full loop for one lens:

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**.
   Bind `{EVIDENCE_DIR}` = `<RDR_EVIDENCE>/<RDR_SLUG>/evidence/<lens>/` (§evidence). **Re-entry** is
   self-detected: a `Status: Draft [revised from Final …; re-verify <IDs>]`
   qualifier → write to `iter-N/` (N = 1 + highest existing; loose files = iter-1)
   and delta-scope to `<IDs>`.
2. **Run the lens prompt** (`pre-lock/0-grounding.md` · `1-3amigo.md` ·
   `2-critique.md` · `4-cove.md`)
   → it writes element files to `{EVIDENCE_DIR}`. Heavy/dual-model → sub-agent returns
   the findings list, not a re-dump. **This first run's findings are the origin
   ledger** for the loop.
3. **Review gate** (below) — a bad pass is re-run on another model, not resolved.
4. **Resolve** — run the sibling [`05-prelock-resolve.prompt.md`](05-prelock-resolve.prompt.md);
   it owns the mechanics (grounding gate · origin anchor · durable disposition ·
   tiebreaker-reduction · flag-as-you-go). Don't restate them here.
5. **Loop or converge.** Substantial fix (a rewrite — can open gaps) → re-run the
   lens **delta-scoped to open ledger entries** under `iter-N/`, then resolve
   again; a small fix needn't. **Converged** = no open entries (all fixed /
   dismissed-with-cite / charted). **Cap = 3**: still surfacing net-new findings
   against a barely-changed draft after three iterations is the plank problem — stop
   with `stopped:verdict-flapping:<lens>:<NNNN>` (name the churning entries),
   surface once; the cure is a human look or model switch, not a fourth pass.

## Repeatability (the multi-session exception)

Not an in-skill loop — independence comes from separate sessions between runs,
not from the invoking session being RDR-naive (the generation prompt reads the
RDR anyway). **Commit cadence is the §commit exception**: each run session commits
only its own `run-<N>.md` (`chore(rdr): cli/NNNN repeatability run-N`); the doc commit
defers to the diff session — see rdr-common §commit.

- **Run the generation prompt directly** — bind `{RDR_PATH}`, `{EVIDENCE_DIR}`,
  `<N>` and execute `3-repeatability.md`. Write `run-<N>.md` and stop. One session
  writes exactly one run; never write a second run in the same session.
- **Next run needs a fresh session** — stop with
  `stopped:repeatability-needs-fresh-session:run-<N+1>` after writing. The next
  `/rdr-prelock NNNN repeatability <N+1>` invocation in a fresh session writes
  the next run. Relaunch on the alt model for the cross-model run.
- **Diff only when complete + clean** — once `run-1/2/3` exist and this session
  wrote none; else `stopped:repeatability-incomplete:<missing>`. ≥1 run on a
  different model.
- **Resolve once `diff.md` lands** — this same skill runs the resolve prompt on
  `diff.md`; its REPEATABILITY DIFF clause governs each divergence. If autocommit is on,
  the diff session does the doc commit (`docs(rdr): prelock cli/NNNN — repeatability`) and
  **one evidence commit over the whole `repeatability/` dir** (`chore(rdr): … repeatability
  evidence`) — self-healing: it sweeps any run files an earlier session left uncommitted
  (no-op guard skips ones already in). See rdr-common §commit.

## Review gate (Stage `05-prelock.md`)

- Read findings against the lens's own **Expected signal**. Healthy = concrete,
  named passages. Generic advice / over-agreement / identical persona lists → switch
  model and re-run; don't proceed on a bad pass.
- After resolving: edits stayed brief (no change-history narration); fixes were
  grounded (a fix to satisfy a finding whose cited code doesn't exist on `main` is
  wrong); the needs-verification list is honest; net-new scope was charted, not
  folded in; tiebreakers surfaced only when evidence was indeterminate.

## Next step (rdr-common §next-step)

- If autocommit is on: **non-repeatability lenses** run **§commit** for `prelock <lens>`
  here (doc + a separate `<lens>` evidence commit). **repeatability** follows its own
  cadence — run files commit per session, the doc commit + a whole-`repeatability/`-dir
  (self-healing) evidence commit land at the diff (see above / §commit).
- **3amigo | critique | cove** converged → next lens: `Next: /rdr-prelock NNNN <next-lens>`.
- **repeatability** — runs missing → `Next: /rdr-prelock NNNN repeatability <N+1>`
  (fresh session); third run exists → `Next: /rdr-prelock NNNN repeatability diff`
  (fresh session that authored no run; it also resolves `diff.md`).
- **All profile lenses done** → `Next: /rdr-reconcile NNNN` (carry the needs-verification list).
- **`stopped:verdict-flapping`** → resume after a human look / model switch, or chart the churning entry.
- A finding refuted an *assumption* → it's on the needs-verification list (Stage 6); if it forces
  a redesign, `/rdr-resolve NNNN` / `/rdr-propose NNNN` now.
- `/rdr-status NNNN` to re-orient.
