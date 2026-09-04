---
name: rdr-draft-to-lock
argument-hint: "<NNNN> [--to <stage>] [--ask-each] [--commit | --no-commit]"
metadata:
  argument-hint: "<NNNN> [--to <stage>] [--ask-each] [--commit | --no-commit]"
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

**At a stop, name the one next command and stop.** Report what the run settled
and what it parked; don't reason past the edge of the flow into advice the
stages own. Where the flow is silent this skill is silent too — a plausible
suggestion invented in a gap is the failure mode, and it reads as authority.

## Usage

```
Codex:  $rdr-draft-to-lock <NNNN> [--to reconcile] [--ask-each]
Claude: /rdr-draft-to-lock <NNNN> [--to reconcile] [--ask-each]
```

`--to <stage>` stops after that stage (`refine|resolve|prelock|reconcile|finalize`;
default `finalize`). `--ask-each` confirms before every stage — the training-wheels
mode; use it the first few runs (the `posture` row's `confirm_each`).
`--commit`/`--no-commit` passes through to every spawn (§commit).

**No model flag.** Stages spawn at the session model. §model-ceiling is a *bump*
for authoring sub-agents and forbids spawning below the session model, so it
could never make this run cheaper — and the cross-model work that pre-lock does
want is `§auto-fanout`'s, selected on model *identity*, not tier. To run the
cascade on a different model, start the session on it.

**Precondition — one projection, no body read.**

```sh
"$RDR_HOME/bin/rdr" inspect --json --filter metadata,outline,counts <NNNN>
```

Read literally: `metadata[]` where `label=="Status"` → `.status.{value,qualifier,form}`;
`outline[]` for section presence (`canonical=="Technical Design"` and
`"Normative Contracts"`); `counts.elements.C`.

- `.status.value == "Final"` → `stopped:already-final` (the qualifier is not part
  of this test — that is why `value` is read, not the line).
- Proposed = both sections present in `outline[]`, **or** `propose-premortem/` on
  disk (still a disk check). Neither → `stopped:not-proposed:<NNNN>` →
  `/rdr-propose NNNN`. `counts.elements.C` corroborates only: prose contracts are
  not addressable, so `C == 0` on an older record means unlabelled, **not**
  absent — never route `stopped:not-proposed` off the count.

A **first** run enters at Stage 3 — refine always runs, so the cascade starts at
its head rather than mid-way on an assumption; a re-invocation enters where the
router says (Re-entry below). Stage 8 is out of scope (`launch.md` owns it).

**A demoted Draft carries its own re-entry scope — ask the router, never
re-derive it.** `.status.form == "revised-from"` means 7.1 (or a Stage-8 spec
defect) sent it back with a scope the report already sized
(`$RDR_HOME/stages/07.1-cluster-reconcile.md`) and wrote as the qualifier's
`@<stage>`. The `reentry` group of `models/rdr-status.toml` owns the mapping
from that target to a scope; `intrastate lint` proves every target has a row.
One chained call answers it — the same facts, no second projection:

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"; M="$RDR_HOME/models/rdr-status.toml"; R="$RDR_HOME/bin/rdr"
"$R" status --tags NNNN >/dev/null || exit 2   # rdr-common §intrastate; the substitution stays inline (zsh)
"$IS" flow resolve --model "$M" --outcome locate  --plan-only $("$R" status --tags NNNN)   # emit.next == resolve:reentry on a demoted Draft
"$IS" flow resolve --model "$M" --outcome reentry --plan-only $("$R" status --tags NNNN)
```

Read `emit.next` and `rule` as values. Only one scope is this skill's:

| `reentry` answer | Scope | Here |
| --- | --- | --- |
| `/rdr-finalize` | RE-LOCK-ONLY | `stopped:scope-relock-only:<NNNN>` → `/rdr-finalize NNNN` |
| `/rdr-refine` or `/rdr-resolve`, rule ≠ `reentry-untargeted` | STAGE-SCOPED | **run it** — enter at that stage; the delta is `edges[]` where `kind=="reverify"` (add `edges` to the precondition's `--filter`); `resolved:false` is a reportable finding, `resolved` absent means nothing looked |
| `/rdr-propose` | FULL-FLOW | `stopped:scope-full-flow:<NNNN>` → `/rdr-propose NNNN` |
| rule `reentry-untargeted` | unstated | `stopped:scope-unstated:<NNNN>` — the row's fallback is "read the note's target line"; this orchestrator reads no body, so it stops rather than guessing from the `reverify` set |

Scope picks *which stages* re-walk; it never edits the lens row. A STAGE-SCOPED
re-entry keeps the profile's whole row — what shrinks is each pass (delta-scoped
to the `reverify` targets), not the sequence; RE-LOCK-ONLY is the cheap path, and
7.1 already chose. The answer is applied as handed: re-deriving the scope from
the qualifier's words, or checking the row's answer against the `reverify` set,
makes the guarantee prose again.

## Posture — delegate everything, hold only the ledger

Like `/rdr-joint-propose` and `launch.md`, this orchestrator **never reads the
RDR text, evidence bodies, or source** — Phase 0's routing reads are the only
exception, as they are for launch.md. Each stage is one sub-agent invoking
the real stage skill; the orchestrator holds only the seam vars, the plan, the
packets, and the parked forks. Two things that buys, neither tradable: stage
skills still author in *their* main context (§delegation is satisfied — the
orchestrator's context is not the stage's), and each stage's Review gate still
runs inside it.

Each stage returns a **§return-packet**. Reject a malformed one and ask only for
a corrected packet; still malformed → one re-spawn, then surface.

## Phase 0 — bind, then plan once

§seam-bind + §rdr-resolve (they bind `{ARTIFACT_DIR}` from `$RDR_ENV`). Then read
**only** the RDR's `Status:` line, `Profile` field, `Seam Lineage`, and — for
`mid`/`large` — the fenced ` ```normative ` block under `#### Normative
Contracts` (an h4 inside Proposed Solution, per TEMPLATE.md; reading it is not
reading the body). Take the row from **§lens-row**'s call — the same one every
stage makes — never a row computed here.

`emit.next` is the first lens and `emit.row` the span — write it as handed.
Write the plan to `{ARTIFACT_DIR}/run-plan.md` (`mkdir -p` it — Stage 7 is
otherwise its first writer). It holds what this run scheduled and decided, and
opens by saying so — never where the RDR stands, which is derived (Re-entry below):

```
rdr: <RDR_SLUG>           profile: <value>   (as read; Draft = provisional)
posture: upfront=<v> each=<v> finalize=<v>   (the posture row's answer; re-asked with the lens row)
lenses: <emit.row verbatim; "none" at small or complete>   [re-entry: delta-scoped to <IDs>]
                                             (critique/repeatability run --auto)
stages: refine -> resolve -> [lenses] -> reconcile -> finalize   stop-after: <--to>
```

On a demoted Draft the bracket is **required**: the row alone overstates the run.
Each pass is a fresh iteration delta-scoped to `re-verify <IDs>` against an
already-resolved draft (`stages/05-prelock.md`), not a full re-review, and a plan
showing only the row asks a human to approve a cost the run will not spend.
Don't compute an iteration number here — `iter-N` is **per-lens** (a row's lenses
can sit at different N) and the stage skill assigns it. Name the delta; let each
lens number itself.

Then a **Ledger** — one row per planned stage (`verdict`, `blocking`, one-line
note), appended as each packet lands. It carries what only this run knows —
verdicts, parks, the `--to` bound — and what a human reads to see where the run
got to. **Never a position**: that is derived (Re-entry below). **Never a
ruling**: what the user decides at a fork is appended to `rulings.md`
(rdr-common §run-prompt — the file every stage reads; nothing reads this plan),
and the Ledger row points at it.

The plan file is the durable state — **re-read it each hop, never carry it in
context** (§no-heartbeat). Profile can change under you: Stage 4 rewrites it
(count, then `floor`), so **re-ask the model after resolve
returns** — the same call as above, which reads the rewritten field itself.
Any stage can move the field, so re-ask every hop and **rewrite the plan when
the answer diverges** — a durable state you don't update is a stale one you
will trust, and this is the one plan that outlives the session that wrote it.
The plan records intent; the model decides.

**Re-entry: re-invoke, never `--resume`.** There is no resume flag (§run-prompt —
re-entry is a property of on-disk state). A run that stopped at a fork resumes by
running the same command again. **The resume point is `emit.next`** — the same
call §lens-row already makes — never the Ledger: a stage's evidence is what the
router reads, so there is no skip guard and no tiebreak, and `/rdr-status NNNN`
alone names the next command with this skill deleted. Read the plan for the
`--to` bound and the forks this run settled; the re-run stage reads `rulings.md`
itself. No plan file → Phase 0 from scratch: that loses the run's bookkeeping,
never its position.

## The loop — one stage per sub-agent

Per stage, spawn one sub-agent whose brief is: the bound seam vars, `{RDR_PATH}`,
and *"run `/rdr-<stage> NNNN [lens]` in full, including its Review gate and
§commit; return a §return-packet."* Nothing else — no orchestrator summary of
the RDR, which would anchor the stage on a reading it did not do. On a
STAGE-SCOPED re-entry the brief adds nothing either: stages 4/5 self-detect the
`revised from Final` qualifier and delta-scope to `re-verify <IDs>` themselves
(§run-prompt). Route to the right stage; let it scope itself.

**Pass `--auto` on `critique` and `repeatability`** — and only there, since
`/rdr-prelock` rejects it elsewhere (`stopped:auto-not-applicable:<lens>`).
Their cross-model passes are otherwise hand CLI relaunches, which no delegated
run can perform: without the flag those lenses park (an owed critique second
pass, or `stopped:repeatability-needs-fresh-session:run-<N+1>` after every run)
and the cascade stops for a human to type one command. With `--auto`,
`repeatability` takes **no `run` arg** — it spawns the variant's whole set, so
the brief names the lens alone. §auto-fanout owns the rest, degradation
included: a harness that can't pin models per spawn emits the relaunch command
and stamps `auto: unavailable (harness) — manual relaunch`, which parks exactly
as before. Passing the flag never makes the run worse; withholding it
guarantees the park.

Then resolve each packet — one call, the answer applied as a value. The
`packet` group of `$RDR_HOME/models/rdr-cascade.toml` owns the ladder;
`intrastate lint` proves every verdict × blocking × retry × action cell.

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"; M="$RDR_HOME/models/rdr-cascade.toml"
# verdict, blocking: as the packet says. retry: INCOMPLETE packets this stage already
# returned this run (Ledger; cap 2). action: next_action classified — `none`; begins
# `stopped:` → stop; names an /rdr- command → stage; else imperative.
"$IS" flow resolve --model "$M" --outcome packet --plan-only \
  --tag verdict=<v> --tag blocking=<yes|no> --tag retry=<0|1|2> --tag action=<none|imperative|stop|stage>
```

Read `emit.next` and `emit.stage` as values:

| `emit.next` | Do |
| --- | --- |
| `advance` | spawn the router's next stage (`emit.stage` = `router`; after `resolve`, the Phase 0 re-ask runs first) |
| `rerun` | spawn the same stage again, the packet's `next_action` appended to its brief |
| `park` | Ledger the verdict and stop advancing this RDR. `Next:` is this stage (`same`) or the packet's named command (`named` — a reconcile `NOT RECONCILED`, a finalize `NOT READY`, a prelock refutation naming an earlier stage) — never run it: re-opening a settled stage is the human's decision. On `named`, the route-back brief tells the stage sub-agent to append the §punt-ledger row (before refine collapses the history) and `changed_paths` must show it |
| `stopped:stage-stop` | relay the stage's own `stopped:*` line verbatim — the codes are the stages' and are never translated |

`park` and `stopped:stage-stop` are forks: §fork-disposition decides ask-now or
park, and ask-now wins whenever the stop blocks the next stage.

## Forks — schedule them, never answer them

**rdr-common §fork-disposition** owns the rule — schedule, never answer — and the
ask-now/park split. This skill's ask-now triggers: an unresolved BLOCK, an
assumption refuted, `stopped:verdict-flapping`, and Stage 4's author's round.

**Stage 4's author's round is a hard stop and is never batched.** "An unapproved
fixture is not Evidence" (`04-resolve.prompt.md`) — it is the flow's one mandated
user interaction in this span, and a run that auto-approves it has forged
evidence, not saved a turn.

Relaying it takes the **one read carve-out** here: a §return-packet can't carry
the round's items and their grounding, so resolve **writes the round to a file**
and returns its path in `evidence_paths` with `verdict: NEEDS_DECISION`. Read
**that file only** and put it to the user unedited — fixtures and questions alike. Without the carve-out the round degrades to a
summary, which is the same forgery by a slower route. Their answers go to
`rulings.md` (§run-prompt), never into a brief.

Before escalating a *judgment* fork (not the author's round, not a mechanical stop),
run **§strong-consult** once — a fresh strongest-tier look may collapse it. Its
`NEEDS_DECISION` goes to the user.

**A consult never closes `stopped:verdict-flapping`.** Its cure is a human look
or a model switch (`$RDR_HOME/stages/05-prelock.md`); a consult that returns PASS
and resumes the lens is a fourth pass in a different hat. Hard stop, like the
author's round — the user picks. Having read no evidence, this context cannot judge that a
consult legitimately collapsed a fork: advisory here, never dispositive.

## Autonomy by profile — the `posture` row decides it

Risk is already sized: `Profile` is the blast-radius latch and `--ask-each`
only raises it. The `posture` group of `$RDR_HOME/models/rdr-cascade.toml`
owns the table (profile × status × ask_each; `intrastate lint` proves every
cell — a provisional `small` on a Draft reads as `mid` by cell, and nothing
reads lower than its field). Ask it where §lens-row is asked — Phase 0, and
every re-ask after a stage returns:

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"
"$IS" flow resolve --model "$RDR_HOME/models/rdr-cascade.toml" --outcome posture --plan-only \
  $("$RDR_HOME/bin/rdr" status --tags --filter profile,status NNNN) --tag ask_each=<true|false>
```

Apply the three emits as values, each at its one site: `confirm_upfront` —
put the plan to the user before Stage 3 spawns; `confirm_each` — before every
stage spawn; `confirm_finalize` — before `/rdr-finalize` (the lock is the
irreversible step). Write them on the plan's `posture:` line and rewrite it
when a re-ask diverges.

The up-front confirm asks about **cost and posture** — the lens row's span and
`stop-after` — never "is the row right?", which §lens-row already decided from
the Profile and the human cannot answer better. State `emit.row`, its
`iter-N`/delta bracket when demoted, and that `critique`/`repeatability` fan out
under `--auto`: that span and that fan-out are where the cost lands.

## Review gate

- Every stage ran its **own** skill and gate — this skill added no gate and
  skipped none. A stage's Review gate is not restated here and never re-judged.
- The orchestrator authored no RDR content and read no RDR body.
- The close named **one** next command and invented no advice the flow doesn't
  own — no alternative orderings, no suggestion a stage would have to refuse.
- Stage 4's author's round reached the user (or the round had nothing to put to
  them, stated as such in the packet).
- Every parked fork is in the close packet — a fork dropped to reach Final is
  the failure mode this skill must not have.
- `Profile` was re-read after resolve; the lens row matches the *current* field.
- Posture came from the `posture` row's current answer, never from the Profile
  read directly; the plan's `posture:` line matches the last re-ask.
- On a re-entry the resume point came from the router, never from the plan file.
- A demoted Draft ran at its report's scope — never a scope this skill chose.
- `critique`/`repeatability` were spawned with `--auto`; a park on either names
  a harness degradation or a real finding, never a missing flag.

## Next step (rdr-common §next-step)

- **Stopped at the pre-finalize confirm** (`large`/`foundational`) → `Next:
  /rdr-finalize NNNN`. Report what reconcile settled and parked; offer no
  alternative to locking. Cross-RDR work is not one — it runs at propose
  (`/rdr-joint-propose`) or after the lock (7.1 needs **Final** members, so a
  Draft can't be in a cluster). Final-and-unimplemented peers are a `Continue
  check:` note, never a reason to defer.
- Ran to `Final` → the finalize packet's `after-lock` answer is `Next:` (name any
  parked non-blocking forks in `Deviations:`).
- Parked at a fork → `Next:` is the stage that owns it (the named return stage
  for a route-back), with the fork stated as the reason.
- `Continue check:` names what judgment remains — this skill's close packet
  carries the whole run, so a human who read nothing else can decide.
- Stage skills self-commit (§commit); this skill commits **only**
  `{ARTIFACT_DIR}/run-plan.md`, subject `chore(rdr): cli/NNNN run-plan`.
