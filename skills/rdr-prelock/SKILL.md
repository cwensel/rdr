---
name: rdr-prelock
metadata:
  argument-hint: "<NNNN> <lens> [run] [--auto] [--commit | --no-commit]   # lens ∈ {grounding, 3amigo, critique, repeatability, cove}; run ∈ {1,2,3,diff}; --auto: critique/repeatability only"
description: 'Use to run one pre-lock lens and resolve its findings in the same pass. Lenses: grounding, 3amigo, critique, repeatability, cove — the `lens` outcome names the next one. Trigger for pre-lock review, $rdr-prelock, or /rdr-prelock.'
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
one run. **Variant follows the profile, not the `run` arg** (§repeatability-variant):
`mid`/`large` = lite (`run-1` only, then `diff`); `foundational`/escalation = full
(`run-1/2/3` then `diff`). Reject an unknown lens with `stopped:bad-lens:<value>`.

`--auto` (`critique` | `repeatability` only — reject elsewhere with
`stopped:auto-not-applicable:<lens>`) runs the lens's cross-model passes as
parallel sub-agents instead of hand CLI relaunches, then the diff behind the
barrier: **rdr-common §auto-fanout** owns the mechanics (distinct models per
spawn, barrier before diff, harness degradation) — don't restate them. With
`--auto`, `repeatability` needs no `run` arg; it spawns the variant's whole set.

**With no lens argument, run §lens-row's call** to get one — it owns the row,
the first-lens fork, the Determinacy add-on and the additive-on-escalation
rule, all encoded in the table, so don't restate or re-derive them.

`emit.next` names the lens to run; `emit.row` is what remains after it.

**Precondition.** Stage 4 must have verified the assumptions — running a lens on
unverified claims wastes it. If Critical Assumptions are still `Pending` without a
plan, stop with `stopped:assumptions-unverified` and point at `/rdr-resolve NNNN`.

## The cycle (3amigo · critique · cove)

One invocation runs the full loop for one lens:

1. Read [`rdr-common.md`](rdr-common.md) **whole, with the Read tool** (it exceeds the 30KB
   Bash cap — `cat` truncates and costs a retry; never `sed`/`grep` §-slices); run **§seam-bind** + **§rdr-resolve**.
   Bind the lens dir and this pass's iteration in one call (§evidence):
   `eval "$("$RDR_HOME/bin/rdr" paths --lens <lens> --next-iter <NNNN>)"` →
   `$EVIDENCE_DIR`, `$ITER`, `$ITER_DIR` (write to `$ITER_DIR`; at `ITER=1` it
   *is* the base). **Re-entry** is self-detected: a `Status: Draft [revised from
   Final …; re-verify <IDs>]` qualifier names the scope's floor, not its ceiling —
   after a rework, real defects land outside the listed ids. Whenever `lens_stale`
   names this lens (`status --tags`), scope = `<IDs>` ∪ `.touched[].id` of
   `base=$(git -C "$RDR_RECORDS" log -1 --format=%h --grep='^docs(rdr): finalize' -- "$RDR_PATH")`
   `"$RDR_HOME/bin/rdr" inspect --touched-since "$base" --json <NNNN>`;
   `re-verify none` → the touched ids alone. Compute it once, here, saving the
   JSON to `$ITER_DIR/touched.json`; a spawn brief carries the id list — never
   a diff.
2. **Run the lens prompt** (`pre-lock/0-grounding.md` · `1-3amigo.md` ·
   `2-critique.md` · `4-cove.md`)
   → it writes element files to `{EVIDENCE_DIR}`. `3amigo` is not one prompt run:
   fan out its three persona passes as isolated sub-agents (no cross-persona
   visibility), then consolidate their files mechanically per the prompt.
   Heavy/dual-model → sub-agent returns
   the findings list, not a re-dump. Anything still on the parent's own list
   (baseline lint, the mini-check cue read) runs **before** the spawn; after it,
   end the turn — a read taken while a spawn runs is a re-run of its brief
   (rdr-common §delegation). **This first run's findings are the origin
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

**The RDR's first lens pass owes the mini-check cue read** before its resolve
closes (cues, tables, desk trace: `$RDR_HOME/stages/05-prelock.md`, single-source);
each fired cue writes its compact table into the RDR. Its close packet reports
`Mini-checks: <fired table names | none>`. A later pass re-runs the read only if
its own fixes add a cue — tables persist in the draft, so a route-back/re-entry
inherits them.

## Repeatability (the multi-session exception)

Not an in-skill loop — independence comes from each run being written by a context
that authored no other run, not from the invoking session being RDR-naive (the
generation prompt reads the RDR anyway). Separate sessions (below) and `--auto`'s
parallel spawns (§auto-fanout) both satisfy that. **Commit cadence is the §commit
exception**: each run session commits only its own `run-<N>.md` (`chore(rdr):
cli/NNNN repeatability run-N`); the doc commit defers to the diff session — see
rdr-common §commit. Under `--auto` the same deferral holds: the diff session's
whole-dir sweep (rdr-commit-map) is the run-set commit — the orchestrator adds
none of its own at the barrier.

### §repeatability-variant — resolve the variant from profile before writing any run

**The intended variant lives on disk, not in session memory** — pre-lock clears
context and switches models between runs, so nothing you hold survives. `run-1.md`'s
`variant:` header line (written at generation, beside `model:`) is the durable record.
**Ask the model for the variant and the next run** — `--outcome repeatability`
(§lens-row's call, that outcome) reads the header and answers with the run to write
or the diff; it never infers the variant from how many `run-*.md` files exist, which
is what silently promotes a `large` RDR to full ×3. When no `run-1.md` exists yet the
row names the variant to stamp into that line.

Two things the row hands back rather than decides. A `run-2`/`run-3` request against a
`variant: lite` header stops with `stopped:repeatability-lite-no-run-2:<NNNN>` —
escalate to full only by rewriting that line to `full (escalated: <reason>)` first
(`3-repeatability.md` *Escalate*), else `diff` on `run-1`. And on `mid`/`large` the
lens is owed only if the `Determinacy:` line in Normative Contracts reads fired: the
row chains to `resolve:determinacy`, which routes run 1, `none`, or stops with
`stopped:determinacy-trigger-unjudged` until the line is written
(`3-repeatability.md` §Determinacy trigger).

- **Run the generation prompt directly** — bind `{RDR_PATH}`, `{EVIDENCE_DIR}`,
  `<N>` and execute `3-repeatability.md`. Write `run-<N>.md` and stop. One session
  writes exactly one run; never write a second run in the same session. (Under
  `--auto` the orchestrator writes none itself — each parallel sub-agent is that
  one-run context; §auto-fanout.)
- **Next run needs a fresh context (full only)** — serial relaunch is the manual
  path, and the fallback when `--auto` is unavailable. After writing, stop with
  `stopped:repeatability-needs-fresh-session:run-<N+1>`; the next
  `/rdr-prelock NNNN repeatability <N+1>` invocation in a fresh session writes
  the next run, relaunched on the alt model for the cross-model draw. **Lite stops
  after `run-1`** with next `Next: /rdr-prelock NNNN repeatability diff` (never a
  `run-2` pointer); do not continue to reconcile.
- **Diff only when complete + clean** — the `repeatability` row answers completeness
  from the header, never the file count. The diffing context wrote none — this session
  manually, or a fresh post-barrier sub-agent under `--auto` (§auto-fanout); else
  `stopped:repeatability-incomplete:<missing>`. ≥1 run on a different model.
- **Resolve once `diff.md` lands** — this same skill runs the resolve prompt on
  `diff.md`; its REPEATABILITY DIFF clause governs each divergence. If autocommit is on,
  the diff session does the doc commit (`docs(rdr): prelock cli/NNNN — repeatability`) and
  **one evidence commit over the whole `repeatability/` dir** (`chore(rdr): … repeatability
  evidence`) — self-healing: it sweeps any run files an earlier session left uncommitted
  (no-op guard skips ones already in). See rdr-common §commit.

## Review gate (Stage `05-prelock.md`)

- Read findings against the lens's own **Expected signal**. Healthy = concrete,
  named passages. Generic advice / unanchored findings → switch model and re-run;
  don't proceed on a bad pass. For `3amigo`, isolated personas converging on the
  same passage is real agreement (a hotspot), not a bad pass — the failure to
  catch is a persona file citing another persona's output (isolation leaked).
- After resolving: edits stayed brief (no change-history narration); fixes were
  grounded (a fix to satisfy a finding whose cited code doesn't exist on `main` is
  wrong); the needs-verification list is honest; net-new scope was charted, not
  folded in; tiebreakers surfaced only when evidence was indeterminate.

## Next step (rdr-common §next-step)

- If autocommit is on: **non-repeatability lenses** run **§commit** for `prelock <lens>`
  here (doc + a separate `<lens>` evidence commit). **repeatability** follows its own
  cadence — run files commit per session, the doc commit + a whole-`repeatability/`-dir
  (self-healing) evidence commit land at the diff (see above / §commit).
- **Before printing `Next:` re-run §lens-row's call** (after this lens's
  evidence has landed) — it re-reads the current
  `Profile` and subtracts the completed evidence itself, so nothing here
  recomputes a row by hand; mandatory after a reset, demotion, or escalation,
  which are exactly the cases a remembered row gets wrong. **Completion is the
  model's too**, and is a second call rather than a judgement here: for the
  lens just run, `--outcome critique` or `--outcome repeatability` reads the
  `Model:` stamps and the `variant:` header (§model-stamp) and answers `none`
  when the lens is finished, or names it again when it is not. Where it names
  the lens again, `Next:` is this lens — on a lens the row does not read
  as stale, that answer outranks the row's folder-level one, which is what
  it is for; a stale lens stays owed whatever the completion files say.
- **3amigo | critique | cove** converged → `emit.next` names the next lens:
  `Next: /rdr-prelock NNNN <next-lens>`. An owed critique
  second pass keeps `critique` the first missing item — `Next:` is
  `/rdr-prelock NNNN critique --auto` where the harness spawns per-model
  (§auto-fanout), else "relaunch the CLI on a second base model, then
  `/rdr-prelock NNNN critique`"; never a later lens (§next-step: one action;
  other open obligations go in `Continue check:`).
- **repeatability** — `--outcome repeatability` names the next run or the diff from
  run-1's own header (§repeatability-variant), never from the files present. Diff runs
  in a fresh session; it also resolves `diff.md`.
- **All profile lenses done** → the model answers `/rdr-reconcile` (row
  complete): `Next: /rdr-reconcile NNNN` (carry the needs-verification list).
  Ask `--outcome repeatability` first — it decides the Determinacy add-on from the
  `Determinacy:` line.
- **`stopped:verdict-flapping`** → resume after a human look / model switch, or chart the churning entry.
- A finding refuted an *assumption* → it's on the needs-verification list (Stage 6); if it forces
  a redesign, `/rdr-resolve NNNN` / `/rdr-propose NNNN` now.
- `/rdr-status NNNN` to re-orient.
