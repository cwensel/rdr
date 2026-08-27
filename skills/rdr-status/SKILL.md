---
name: rdr-status
metadata:
  argument-hint: "[NNNN]   # omit to list in-flight RDRs"
description: 'Use to find the current stage and next command for an RDR. Read-only navigator over status, evidence, and artifacts. Trigger for where is RDR, next RDR step, $rdr-status, or /rdr-status.'
---

# rdr-status

The **navigator** for the RDR flow. Given an RDR number, it reports which stages
have produced evidence on disk, the RDR's current `Status`, and the recommended
next command. It is the answer to "I lost my place — what do I run next?"

**Read-only. It writes nothing.** State is *derived*, never stored — there is no
ledger. This is deliberate (the flow's resources-vs-memory doctrine,
`$RDR_HOME/stages/README.md`): a position computed from the artifacts can't drift from them,
and it stays correct even when a stage was driven by hand or in another session.

## Usage

```
Codex: $rdr-status <NNNN>
Claude: /rdr-status <NNNN>
Codex: $rdr-status            # no arg: list in-flight RDRs (Draft/not-yet-Final) + their next step
Claude: /rdr-status           # no arg: list in-flight RDRs (Draft/not-yet-Final) + their next step
```

Read the shared bones in [`rdr-common.md`](rdr-common.md): run **§seam-bind**
then **§rdr-resolve** to bind `$RDR_ENV`, `RDR_PATH`, `RDR_SLUG`, and the evidence
roots. Then derive position as below. Do **not** edit any file.

## One call

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"   # marker var, else PATH, else skip
M="$RDR_HOME/models/rdr-status.toml"; R="$RDR_HOME/bin/rdr"
if [ -x "$IS" ]; then "$IS" flow resolve --model "$M" --outcome locate $("$R" status --tags NNNN)
else echo "note: routing model not consulted (intrastate unresolved)"; fi
```

Test the binary, never the exit code: a refusal exits 2, and `&& … ||` would
report a model that WAS consulted and refused as one that never ran — the
opposite of what happened, and the one thing this note must never get wrong.

That one call answers both halves. `observed.*` in its output **is** the fact
vector — every signal the checklist renders, echoed back — and `emit` is the
answer. There is nothing else to fetch: **do not** also run `status --json`, `ls`
the evidence tree, re-read the record, or loop the cluster peers' facts. The probes
already looked, by exact path; a hand-built path is how a lens that ran reads as
un-run, and a record's own `clustered` + `cluster_reconciled` settle **which
stage** is next without a peer's facts changing it.

**Run `--outcome lens` only when `emit.next` is `resolve:lens`.** That token is
the model's own chain instruction and one row emits it (a `Draft` whose
assumptions are all terminal, where Resolve is behind you and §lens-row owns the
question). Every other row answers completely, so a second call adds a duplicate
48-fact echo and nothing else.

```sh
"$IS" flow resolve --model "$M" --outcome lens $("$R" status --tags NNNN)
```

Keep `$(…)` **inline**. Unquoted is safe — every fact is one shell word and prose
facts are not rendered — but zsh does not word-split an unquoted *variable*, so
`T=$(…)` then `$T` sends the whole vector as one argument (`unknown flag:
--tag`). The repeated ~25ms call is the cost of the correct shape.

Take `emit.next` (a command, `none`, or `stopped:…`), `emit.why`, and
`emit.surface` (print verbatim); append `NNNN` yourself — emit interpolates
nothing. A `flow-guard-unevaluable` refusal means a fact went **absent**, not
false: a root is unbound or names no directory. Say so; never fill the gap.

Only one signal needs a second `rdr` call, and only when `status_form` is not
`none` — the Status qualifier's prose is deliberately not a fact:

```sh
"$RDR_HOME/bin/rdr" inspect --json --filter metadata NNNN   # the qualifier text
```

### When the model is not consulted

If the note fired, the facts and the routing both have to come from elsewhere:

```sh
"$RDR_HOME/bin/rdr" status --json NNNN     # ~250 lines / 5KB — read it whole, do not page it
```

and read the rows of `$RDR_HOME/models/rdr-status.toml` for the routing. **They
are the prose of record — do not re-derive the branches from memory.** Then
**say so in Caveats**: the model is linted and this path is not, so an
unannounced skip reads as the checked answer when it is the unchecked one.

In `--json` an **absent** key means nothing looked (unbound root); it is not
`false`, and never read one as the other. (`--tags` substitutes declared
sentinels for the routing dimensions, since argv cannot spell absence.)

**Edit the model with any routing change.** `intrastate lint --model` proves
every status×qualifier×ca×cluster and profile×lens cell is claimed exactly once —
the guarantee prose cannot give, and where a gap becomes a test failure.

## What the facts mean

`models/rdr-facts.toml` declares every signal; these are the judgements it
**cannot** make, which is the whole of what this skill still decides.

| Stage | Facts | Judgement left to you |
| --- | --- | --- |
| 1 Seed | `status` | — |
| 2 Propose | `premortem_line`, `ground_sweep_line`, `joint_checks` | Legacy records predate the verdict lines: absence alone does not reopen propose when the sections are filled, but **surface the unrun check as a Caveat** — a skipped gate item otherwise reads as a passed one. A *paused* joint-decision fire is propose-not-done. |
| 3 Refine | `ca_verified`, `ca_pending`, `spikes` | Human-judged, certified only by **Stage 4's product** — a Verified assumption, or spikes. A `Method:`/`Evidence:` line is not a signal (TEMPLATE ships both as skeleton labels). All-`Pending` means Refine is un-run. |
| 4 Resolve | `ca` (rollup), `spikes` | `Pending`-with-plan counts as terminal; the plan is prose, so you read it. A pure source-search resolve names no spikes and writes **no** folder — an absent dir is expected, not a sign Resolve is unrun. An MVV-critical assumption left `Pending` resolves here — Caveat it, don't mark Resolve unrun. |
| 5+6 Pre-Lock (review+resolve) | `profile`, `contracts`, `lens_grounding`, `lens_3amigo`, `lens_critique`, `lens_cove`, `lens_repeatability` (ran); `lens_grounding_findings`, `lens_cove_findings`, `lens_3amigo_consolidation`, `lens_critique_single`, `lens_critique_modelb`, `lens_critique_diff`, `lens_repeatability_run1`/`run2`/`run3`, `lens_repeatability_diff` (finished); `reconcile`, `iter_2` | Review + resolve are one cycle. Resolution is human-judged: a lens converged if the next lens's folder exists, or `reconcile`. `critique` on a `foundational` RDR owes the dual-model diff — a lone `lens_critique_single` is in-progress, not done. |
| 6 Reconcile | `reconcile`, `reconcile_report{,_alt,_alt2}` | A bare folder with no report is a real state: the stage started and left nothing. |
| 7 Finalize | `status`, `gate_written` | Legacy records carry all five gate responses inline — either satisfies. The README index row is not a fact; do not claim it. |
| 7.1 Cluster | `cluster` (declared), `clustered`, `cluster_reconciled` | Read both booleans, or a solo Final routes to a stage with nothing to reconcile. `cluster` is a Propose-time claim, **not the membership** — print it as what the record declares, never as the cluster; Stage 7.1 builds its own set. The topical epoch (`dml-purpose`) is keyed by subject, reads `false`, and is out of scope — all its records are terminal. |
| 8 Implement | `impl_capsule`, `impl_state` | `impl_state` is the capsule header's own state word. Open req-list/coverage/verification.md **only** if it is absent or contradicts the tree. |

`propose_premortem` is Stage 2's critic output, a non-lens sibling — never count it
toward Stage-5 convergence. `legacy_evidence_shape` names the older top-level
layout, so those can never read as "lens un-run".

## Judgement the facts cannot make

Four questions route nothing, because each is about **another record** or about
prose. The model declines them on purpose; they are yours.

- **The tandem barrier.** A bare `Draft` declaring a `Cluster` is barred from
  refine until every member has completed propose. `cluster` is set-valued and no
  row guards it — if a sibling has not proposed, next is that sibling's
  `/rdr-propose`, not this RDR's `/rdr-refine`.
- **Has the home answered?** `stopped:check-the-joint-decision-home` means a
  `Final` owes a joint decision. If the home has ANSWERED, the scoped
  answer-vs-fences check (Stage 7.1) is owed **before** implement. Surface the home
  and the open question.
- **The Determinacy trigger.** On a `mid`/`large` RDR an algorithmic contract
  appends `repeatability` (lite) to the row. `contracts` is a count and **zero
  means unlabelled, not absent** — read the Normative Contracts section and judge.
  Variant follows `Profile`, never the files present: `mid`/`large` = lite
  (`run-1` then a focused diff); `foundational` = full (`run-1/2/3` then `diff`).
- **Is a `Profile` earned?** A `Draft` Profile is provisional (Resolve earns it,
  Stage 7 latches it) — a hint, never a basis for certifying a lens-skip. Absent
  is `profile=none`, a §stop-packet rather than a default.

## Output

Be brief. Everything below comes from `observed.*` and `emit` — nothing is
fetched for it.

1. **Header** — `RDR NNNN-<slug> — <Status line verbatim>`. The bare status is
   `status`; the qualifier text needs the `--filter metadata` call, and only when
   `status_form` is not `none`.
2. **Stage checklist** — one line per row of **What the facts mean**, verbatim in
   name and order; never invent, split, or rename a row. **Pre-Lock is the single
   `5+6` row** — no separate "Resolve-findings" stage. Mark each:
   - `✓` — the row's facts are `true`.
   - `–` — they are `false`.
   - `~` — no durable artifact by design (Stage 3, and Stage 4 without spikes):
     certified downstream, not forgotten. Name the downstream signal, not just the
     verdict (`3 Refine  ~ judged done — A3/A5 Verified`). If that signal is
     absent the `~` is **open** — say so and let it own **Next**
     (`3 Refine  ~ open — CAs all Pending`). Never write `~` as done while naming
     no Stage-4 product.
   - **absent** (the fact is missing, not `false`) — write `?` and name the
     unbound root. Never render an absent fact as `–`.

   Print only **this profile's** lenses on the 5+6 row, plus a Determinacy-owed
   `repeatability` when it fires — an owed obligation is never hidden. Off-profile
   lenses are absent from the row, not `–`
   (`5+6 Pre-Lock  ✓ grounding  ✓ 3amigo` for a mid RDR).
3. **Next** — `emit.next` with `NNNN` appended, e.g.
   `Next: /rdr-prelock 0046 critique`. `none` → say terminal and name the
   disposition. `stopped:…` → print `emit.why` and stop there; a stop is an answer,
   never a stage to guess past.
4. **Caveats** — `emit.surface` verbatim, plus only genuinely-open items: a `~`
   gate whose downstream signal is **absent** (a `~` already certified downstream
   stays in the checklist, never here — don't nudge a re-run of a done stage); an
   unearned Profile basis (`profile=none` → "no Profile; inferred mid"; a `Draft`
   Profile → "Profile mid is Seed's estimate — Resolve to confirm"; a `Final`
   Profile is earned, no caveat); a Stage-2 verdict line the record does not carry;
   and an **unconsulted routing model**, when `intrastate` did not resolve.

No writes. Confirm `git status` would be unchanged (you ran only reads).

## No-arg mode

One command, no glob and no per-file read:

```sh
"$RDR_HOME/bin/rdr" status
```

It returns the `Draft`/`Final`-not-yet-`Implemented` set with each Status and
qualifier already split, and every fact under it — so a row needs no follow-up
read. Report each as `NNNN-slug · <Status> · next: /rdr-<stage> NNNN`, routing
from the same model; resolve per row only for the row being acted on, since the
worklist itself needs no per-row call.
Parked RDRs are not in flight, so add `--status --json` and take the records with
`terminal:false, in_flight:false` (`Deferred`) — list each on a separate **parked**
line with its `status.qualifier` revisit condition verbatim: not in flight, not
closed either, and a trigger nobody re-reads is how a park becomes an abandon by
default. `--status` also groups the rest, so `Implemented`/`Demoted`/`Abandoned`/
`Superseded` need no separate skip rule.

When several listed Drafts are
pre-propose siblings, recommend proposing **all** of them before any refines —
breadth-first keeps joint-decision fires against still-fluid drafts
(stages/02-propose.md, batch ordering).

## Self-update

When the stages change how they write, the signal is data: add or amend the fact
in `models/rdr-facts.toml`, and the routing that reads it in
`models/rdr-status.toml` (the two are coupled by fact NAME and nothing else, so a
rename is a breaking change across both). Update this file only for the
judgement columns above and `rdr-common.md` §evidence. Stage-mechanics changes
live in the stage `.md` + its prompt file, not here.
