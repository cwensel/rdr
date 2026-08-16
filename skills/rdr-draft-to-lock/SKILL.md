---
name: rdr-draft-to-lock
metadata:
  argument-hint: "<NNNN> [--to <stage>] [--ask-each] [--model-ceiling <model>] [--commit | --no-commit]   # EXPERIMENTAL: drives Refine→Finalize"
description: 'Use to drive one proposed RDR from Refine through Finalize as delegated stages, stopping at genuine human forks. Experimental. Trigger for run the flow, drive to final, $rdr-draft-to-lock, or /rdr-draft-to-lock.'
---

# rdr-draft-to-lock — Stages 3→7 as one delegated pass (EXPERIMENTAL)

Drive **one already-proposed** RDR from Refine to Final. Design judgment is
settled at Seed and Propose; 3→7 is a filter cascade whose stages already own
their gates. This skill schedules them and holds the forks — never re-decides
them, and never spans Seed/Propose or Implement (`$RDR_HOME/stages/README.md`
*One skill per stage* records why that bound is the trial's constraint).

**Experimental.** It changes *who types the next command*, never what a stage
does: each stage runs its own skill, unmodified, in its own sub-agent. The
sub-agent hand-off is what costs the driver their seat, so **every stop rule
below is what buys it back — none is optional.** Two invariants: the RDR sits at
a clean stage boundary throughout (delete this skill mid-run and `/rdr-status
NNNN` names the next command), and no fork is ever answered here.

## Usage

```
Codex:  $rdr-draft-to-lock <NNNN> [--to reconcile] [--ask-each]
Claude: /rdr-draft-to-lock <NNNN> [--to reconcile] [--ask-each]
```

`--to <stage>` stops after that stage (`refine|resolve|prelock|reconcile|finalize`;
default `finalize`). `--ask-each` confirms before every stage — the training-wheels
mode; use it the first few runs. `--model-ceiling` and `--commit`/`--no-commit`
pass through to every spawn (§model-ceiling, §commit). Resolve the ceiling in
Phase 0 and **record it in the plan** — `large`/`foundational` spawn at it, and an
unset ceiling silently runs the judgment-dense profiles at session model, which
is a choice worth seeing rather than inheriting.

**Precondition.** `Status: Draft`, proposed — tested without reading the body:
`### Technical Design` + `#### Normative Contracts` present, or `propose-premortem/`
on disk. Neither → `stopped:not-proposed:<NNNN>` → `/rdr-propose
NNNN`. Refuse `Final` (`stopped:already-final`). Entry is Stage 3 because refine
always runs — the cascade is re-entered at its head, never mid-way on an
assumption. Stage 8 is out of scope (`launch.md` owns it).

## Posture — delegate everything, hold only the ledger

Like `/rdr-joint-propose` and `launch.md`, this orchestrator **never reads the
RDR text, evidence bodies, or source** — the cheap metadata reads in Phase 0 are
the exception, as they are for launch.md. Each stage is one sub-agent invoking
the real stage skill; the orchestrator holds only the seam vars, the plan, the
packets, and the parked forks. Two things that buys, neither tradable: stage
skills still author in *their* main context (§delegation is satisfied — the
orchestrator's context is not the stage's), and each stage's Review gate still
runs inside it.

Each stage returns a **§return-packet**. Reject a malformed one and ask only for
a corrected packet; still malformed → one re-spawn at the ceiling, then surface.

## Phase 0 — bind, then plan once

§seam-bind + §rdr-resolve (they bind `{ARTIFACT_DIR}` from `$RDR_ENV`). Then read
**only** the RDR's `Status:` line, `Profile` field, `Seam Lineage`, and — for
`mid`/`large` — the fenced ` ```normative ` block under `#### Normative
Contracts` (an h4 inside Proposed Solution, per TEMPLATE.md; reading it is not
reading the body). Those are the cheap routing reads. Compute the lens row via **§lens-row** and write the
plan to `{ARTIFACT_DIR}/run-plan.md` (`mkdir -p` it — Stage 7 is otherwise its
first writer):

```
rdr: <RDR_SLUG>           profile: <value>   (as read; Draft = provisional)
ceiling: <model | unset (session model)>
lenses: <row, in order, or "none (small)">
stages: refine -> resolve -> [lenses] -> reconcile -> finalize   stop-after: <--to>
```

The plan file is the durable state — **re-read it each hop, never carry it in
context** (§no-heartbeat). Profile can change under you: Stage 4 rewrites it
(count, then the accretion floor), so **recompute §lens-row from the field after
resolve returns**. Any stage can move the field, so re-read `Profile` every hop
and **rewrite the plan when they diverge** — a durable state you don't update is
a stale one you will trust. The plan records intent; the field decides.

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
- A stage's own `stopped:*` → carry the code **verbatim** (the codes are the
  stages', and it must not translate them). Verbatim is about wording, not
  timing: whether it asks now or parks is the fork rule below, and ask-now wins
  whenever the stop blocks the next stage.

**Route-backs are the orchestrator's hard boundary.** A reconcile `NOT
RECONCILED`, a finalize `NOT READY`, or a prelock refutation names an *earlier*
stage. This skill **never drives backward** — park the verdict with its named
return stage and stop; re-opening a settled stage is the human's decision. The
§punt-ledger row is **not** theirs to remember: the route-back brief tells the
stage sub-agent to append it (before refine collapses the history) and
`changed_paths` must show it. Report `Next: /rdr-<named-stage> NNNN`.

## Forks — schedule them, never answer them

**rdr-common §fork-disposition** owns the rule — schedule, never answer — and the
ask-now/park split. This skill's ask-now triggers: an unresolved BLOCK, an
assumption refuted, `stopped:verdict-flapping`, and Stage 4's I/O round.

**Stage 4's I/O round is a hard stop and is never batched.** "An unapproved I/O
pair is not Evidence" (`04-resolve.prompt.md`) — it is the flow's one mandated
user interaction in this span, and a run that auto-approves it has forged
evidence, not saved a turn.

Relaying it takes the **one read carve-out** here: a §return-packet can't carry
I/O pairs, so resolve **writes the round to a file** and returns its path in
`evidence_paths` with `verdict: NEEDS_DECISION`. Read **that file only** and put
the pairs to the user unedited. Without the carve-out the round degrades to a
summary, which is the same forgery by a slower route.

Before escalating a *judgment* fork (not an I/O round, not a mechanical stop),
run **§strong-consult** once — a fresh strongest-tier look may collapse it. Its
`NEEDS_DECISION` goes to the user.

**A consult never closes `stopped:verdict-flapping`.** Its cure is a human look
or a model switch (`$RDR_HOME/stages/05-prelock.md`); a consult that returns PASS
and resumes the lens is a fourth pass in a different hat. Hard stop, like the I/O
round — the user picks. Having read no evidence, this context cannot judge that a
consult legitimately collapsed a fork: advisory here, never dispositive.

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
provisional** (Resolve earns it), so posture never *drops* on an unearned field:
read an unearned `small` as `mid`, and keep an unearned `large`/`foundational` at
its own row until Stage 4 writes the field.

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
