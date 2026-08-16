---
name: rdr-run
metadata:
  argument-hint: "<NNNN> [--to <stage>] [--ask-each] [--model-ceiling <model>] [--commit | --no-commit]   # EXPERIMENTAL: drives Refine→Finalize"
description: 'Use to drive one proposed RDR from Refine through Finalize as delegated stages, stopping at genuine human forks. Experimental. Trigger for run the flow, drive to final, $rdr-run, or /rdr-run.'
---

# rdr-run — Stages 3→7 as one delegated pass (EXPERIMENTAL)

Drive **one already-proposed** RDR from Refine to Final — the span propose hands
off to. The premise: the design judgment lives at Seed and Propose; 3→7 is a
filter cascade whose stages already own their gates and already know when to
stop. This skill schedules them and holds the forks — it does not re-decide them.

**Experimental.** It changes *who types the next command*, never what a stage
does: each stage runs its own skill, unmodified, in its own sub-agent. If this
skill were deleted mid-run, the RDR is on disk at a clean stage boundary and
`/rdr-status NNNN` names the next command. That is the safety property; keep it.

## Usage

```
Codex:  $rdr-run <NNNN> [--to reconcile] [--ask-each]
Claude: /rdr-run <NNNN> [--to reconcile] [--ask-each]
```

`--to <stage>` stops after that stage (`refine|resolve|prelock|reconcile|finalize`;
default `finalize`). `--ask-each` confirms before every stage — the training-wheels
mode; use it the first few runs. `--model-ceiling` and `--commit`/`--no-commit`
pass through to every spawn (§model-ceiling, §commit).

**Precondition.** `Status: Draft` with Proposed Solution authored (post-Stage-2).
A seeded-unproposed RDR stops with `stopped:not-proposed:<NNNN>` → `/rdr-propose
NNNN`. Refuse `Final` (`stopped:already-final`). Stage 8 is **out of scope** —
implement is a separate act with its own orchestrator (`launch.md`).

## Posture — delegate everything, hold only the ledger

Like `/rdr-joint-propose` and Stage 8's `launch.md`, this orchestrator **never
reads the RDR body, evidence files, or source** (§delegation, and the stricter
orchestrator rule in `$RDR_HOME/stages/README.md` *Doctrine*). Each stage runs
as one sub-agent invoking the real stage skill; the orchestrator holds only:
the bound seam vars, the lens plan, the packets, and the parked forks. Two
consequences it must not trade away: stage skills keep authoring in *their* main
context (§delegation is satisfied — the orchestrator's context is not the
stage's), and a stage's own Review gate still runs inside that stage.

Each stage returns a **§return-packet**. Reject a malformed one and ask only for
a corrected packet; still malformed → one re-spawn at the ceiling, then surface.

## Phase 0 — bind, then plan once

§seam-bind + §rdr-resolve. Then read **only** the RDR's `Status:` line, `Profile`
field, `Seam Lineage`, and (for `mid`/`large`) Normative Contracts — the cheap
routing reads, not the body. Compute the lens row via **§lens-row** and write the
plan to `{ARTIFACT_DIR}/run-plan.md`:

```
rdr: NNNN-<slug>          profile: <value>   (as read; Draft = provisional)
lenses: <row, in order, or "none (small)">
stages: refine -> resolve -> [lenses] -> reconcile -> finalize   stop-after: <--to>
```

The plan file is the durable state — **re-read it each hop, never carry it in
context** (§no-heartbeat). Profile can change under you: Stage 4 rewrites it
(count, then the accretion floor), so **recompute §lens-row from the field after
resolve returns**, not from this plan. The plan records intent; the field decides.

## The loop — one stage per sub-agent

Per stage, spawn one sub-agent whose brief is: the bound seam vars, `{RDR_PATH}`,
and *"run `/rdr-<stage> NNNN [lens]` in full, including its Review gate and
§commit; return a §return-packet."* Nothing else — no orchestrator summary of
the RDR, which would anchor the stage on a reading it did not do.

Then, per packet:

- `PASS` **with `blocking: no`** → next stage. After `resolve`, re-derive the
  lens row first (above). `blocking:` is a separate field from `verdict:` — a
  `PASS` that sets `blocking: yes` is a passed gate carrying an open item, and
  it **parks**; never advance on the verdict alone.
- `INCOMPLETE` → re-run the **same** stage once with the packet's `next_action`
  appended. Twice incomplete → park as a fork; never a third silent re-run.
- `BLOCK` / `NEEDS_DECISION` → **park and stop advancing this RDR** (below).
- A stage's own `stopped:*` → park it verbatim; the codes are the stages', not
  this skill's, and it must not translate them.

**Route-backs are the orchestrator's hard boundary.** A reconcile `NOT
RECONCILED`, a finalize `NOT READY`, or a prelock refutation names an *earlier*
stage. This skill **never drives backward** — it parks the verdict with the named
return stage and stops. Re-opening a settled stage is a design decision, and
§punt-ledger records it at the route-back; that ledger row is the human's to
own. Report `Next: /rdr-<named-stage> NNNN`.

## Forks — schedule them, never answer them

The doctrine is `/rdr-joint-propose`'s: **a human-judgment fork is scheduled, not
answered.** Two rules decide *when* to interrupt:

- **Ask now** when the fork blocks the next stage — an unresolved BLOCK, an
  assumption refuted, a `verdict-flapping` at cap, or Stage 4's I/O round.
  Advancing past it wastes the stages after it.
- **Park** anything that does not block, and batch every parked item into **one**
  `AskUserQuestion` when the run stops. One interruption per run is the target.

**Stage 4's I/O round is a hard stop and is never batched.** "An unapproved I/O
pair is not Evidence" (`04-resolve.prompt.md`) — it is the flow's one mandated
user interaction in this span, and a run that auto-approves it has forged
evidence, not saved a turn. The resolve sub-agent surfaces the consolidated
round; the orchestrator relays it to the user verbatim and waits.

Before escalating a *judgment* fork (not an I/O round, not a mechanical stop),
run **§strong-consult** once — a fresh strongest-tier look may collapse it. Its
`NEEDS_DECISION`, or a repeat flap, goes to the user.

## Autonomy by profile — bias to hands-off where the blast radius is small

Risk is already sized: `Profile` is the blast-radius latch. Use it, don't invent
a second scale.

| Profile | Posture |
| --- | --- |
| `small` | Hands-off `3 → 4 → 6 → 7`. No lens. Report at Final. |
| `mid` | Hands-off through the lens row; forks batched at the end. |
| `large` | Hands-off, but confirm once before `/rdr-finalize` (lock is the irreversible step). |
| `foundational` | Confirm the lens plan up front; confirm before finalize. Cross-RDR blast radius earns two interruptions. |

`--ask-each` overrides the table upward (confirm everywhere); nothing overrides
it downward — a `foundational` run cannot be made silent. A **`Draft` Profile is
provisional** (Resolve earns it): treat an unearned `small` as `mid` for posture
until Stage 4 has written the field.

## Review gate

- Every stage ran its **own** skill and gate — this skill added no gate and
  skipped none. A stage's Review gate is not restated here and never re-judged.
- The orchestrator authored no RDR content and read no RDR body.
- Stage 4's I/O round reached the user (or the RDR had no I/O-expressible
  assumption, stated as such in the packet).
- Every parked fork is in the close packet — a fork dropped to reach Final is
  the failure mode this skill must not have.
- `Profile` was re-read after resolve; the lens row matches the *current* field.

## Next step (rdr-common §next-step)

- Ran to `Final` → `Next: /rdr-implement NNNN` (name any parked non-blocking
  forks in `Deviations:`).
- Parked at a fork → `Next:` is the stage that owns it (the named return stage
  for a route-back), with the fork stated as the reason.
- `Continue check:` names what judgment remains — this skill's close packet
  carries the whole run, so a human who read nothing else can decide.
- Stage skills self-commit (§commit); this skill commits **only**
  `{ARTIFACT_DIR}/run-plan.md`, subject `chore(rdr): cli/NNNN run-plan`.
