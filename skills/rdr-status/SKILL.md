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

## What it reads (the disk signals)

**`rdr status` evaluates all of these — you do not walk the tree.** The shapes are
here to make a fact's name legible and the table reviewable, not to run by hand.

- **Lenses:** `<RDR_EVIDENCE>/<slug>/evidence/<lens>/` — slug, literal `evidence/`,
  then the lens. (A check at `<slug>/<lens>/` finds nothing and falsely reports the
  lens un-run, which is why probes spell whole paths and never guess.)
- **Spikes:** `…/<slug>/evidence/spikes/` (`{SPIKE_DIR}`, rdr-common §evidence).
- `propose-premortem/` is Stage 2's critic output, a non-lens sibling — never count
  it toward Stage-5 lens convergence.
- Loose top-level lens folders are an older layout; `legacy_evidence_shape` names
  them, so they can never read as "lens un-run".

| Stage | Done-signal on disk |
| --- | --- |
| 1 Seed | RDR file exists; `Status: Draft`; Problem Statement filled (not placeholder) |
| 2 Propose | Proposed Solution / Alternatives / Decision Rationale filled; Critical Assumptions list present (even if Pending); `Premortem:`, `Ground-sweep:`, and `Joint-check:` verdict lines in Decision Rationale (legacy RDRs predate them — absence alone doesn't reopen propose when the sections are filled, but surface it as a Caveat naming the unrun check, since a skipped gate item otherwise reads as a passed one; a *paused* joint-decision fire or bridge choice is propose not done) |
| 3 Refine | *human-judged* — certified only by **Stage 4's product**: an assumption at `Status: Verified`, or `{SPIKE_DIR}`. A `Method:`/`Evidence:` line is **not** a signal (TEMPLATE.md ships both as skeleton labels; Stage 2 lists CAs `Pending` by design) — an all-`Pending` list means Refine is un-run. Never certify it from Stage-2 output (CA count, `Premortem:`/`Joint-check:` verdicts, propose evidence) |
| 4 Resolve | Critical Assumptions all `Verified` or `Pending`-with-plan (the **primary** signal, from the CA tallies above); `{SPIKE_DIR}` present when spikes were named. A pure source-search resolve names no spikes and writes **no** evidence folder — verdicts are inline; an absent `<slug>/` dir is then expected, not a sign Resolve is unrun. An MVV-critical assumption left `Pending` (the MVV, or a normative fixture it consumes, rests on it) resolves at Stage 4 — surface it as a Caveat, don't mark Resolve unrun. |
| 5+6 Pre-Lock (review+resolve) | which `<RDR_EVIDENCE>/<slug>/evidence/<lens>/` folders exist — per lens (`grounding`, `3amigo`, `critique`, `repeatability`, `cove`), incl. `iter-N`. Review + resolve are one cycle; *resolution is human-judged* — infer a lens converged from the next lens's folder existing, or from `evidence/reconcile/`. **`critique` on a `foundational` RDR needs the dual-model diff** (`critique-modelB.md`/diff), not just `critique.md` — a lone single-model file is in-progress, not done (rdr-common §model-stamp). |
| 6 Reconcile | `<RDR_EVIDENCE>/<slug>/evidence/reconcile/` report exists; assumptions all terminal (no Pending without impl-plan) |
| 7 Finalize | `Status: Final`; `{ARTIFACT_DIR}/gate.md` present, with `### Cross-Cutting Concerns` retained in the RDR (legacy RDRs: all five responses inline — either satisfies); README index row updated |
| 7.1 Cluster | `<RDR_EVIDENCE>/cluster-reconcile/<key>/` — keyed by the CLUSTER (`0117-0118`), not by slug, so it is not under `<slug>/`. In the current shape the key is the members' numbers joined, so the key IS the membership and `cluster_reconciled` answers it exactly. An earlier topical epoch (`dml-purpose`, `final-cluster-2026-05-28`) is keyed by subject instead; those are out of scope and read `false` — all their records are terminal (only when the RDR is in a cluster) |
| 8 Implement | `{ARTIFACT_DIR}/status.md` capsule header read first (phase/next/blocker/state in one pass); state reads `COMPLETE`, `INCOMPLETE`, or `IN-PROGRESS`. Only open req-list/coverage/verification.md if the header is missing, stale, or contradicts the tree |

### Both halves — one call, no listing

```sh
"$RDR_HOME/bin/rdr" status --json <NNNN>     # ~250 lines / 5KB — read it whole
```

`models/rdr-facts.toml` declares every signal in the table above — projected
fields AND exact-path probes — and this evaluates them all, small enough that
**paging it with `head`/`tail`/`sed` just re-runs the command.**

**Do not `ls` the evidence tree, re-read the record, or poll the cluster peers**:
the probes already looked, by exact path, and a hand-built path is how a lens that
ran reads as un-run. A record's own `clustered` + `cluster_reconciled` settle
Stage 7.1 — a peer's facts change no answer. `impl_state` is the Stage-8 capsule's
own state word: open implementation artifacts only if it is absent or contradicts
the tree.

In `--json` an **absent** key means nothing looked (unbound root); it is not
`false`, and never read one as the other. (`--tags` substitutes declared sentinels
for the routing dimensions, since argv cannot spell absence.)

Only two signals need a second call, both by design — the facts say a verdict line
*is written*, not what it said, and Status qualifier prose is deliberately not a
fact (`status_form` is):

```sh
"$RDR_HOME/bin/rdr" inspect --select <NNNN>:§decision-rationale <NNNN>  # Premortem:/Ground-sweep: text
"$RDR_HOME/bin/rdr" inspect --json --filter metadata <NNNN>             # qualifier, where Output says "verbatim"
```

## How it decides "next"

These branches are also **data** — `$RDR_HOME/models/rdr-status.toml`, one linted
decision table. When `intrastate` resolves, one call answers each half; it is an
accelerator, never a dependency, so if it does not, read on.

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"   # marker var, else PATH, else skip
M="$RDR_HOME/models/rdr-status.toml"; R="$RDR_HOME/bin/rdr"
if [ -x "$IS" ]; then
  "$IS" flow resolve --model "$M" --outcome locate $("$R" status --tags NNNN)
  "$IS" flow resolve --model "$M" --outcome lens   $("$R" status --tags NNNN)
else echo "note: routing model not consulted (intrastate unresolved)"; fi
```

If that note fires, the branches came from prose — **say so in Caveats**. The
model is linted and the prose is not, so an unannounced skip reads as the checked
answer when it is the unchecked one.

Unquoted `$(…)` is safe — every fact is one shell word, prose facts aren't
rendered — but keep it **inline**: zsh does not word-split an unquoted variable,
so `T=$(…)` then `$T` sends the whole vector as one argument (`unknown flag:
--tag`). Take `emit.next` (a command, `none`, or `stopped:…`), `emit.why`, and
`emit.surface` (print verbatim); append `NNNN` yourself — emit interpolates nothing.

**Edit the model with any change here.** `intrastate lint --model` proves every
status×qualifier×ca×cluster and profile×lens cell is claimed exactly once — the
guarantee this prose cannot give, and where a gap becomes a test failure.

1. **`Status` first — it is the coarse position.**
   - `Demoted` → the RDR exited at Seed; next is none (refiled as an issue).
   - `Final` → next is `/rdr-implement` (unless a cluster of ≥2 Final-unimplemented
     peers exists and `cluster_reconciled` is `false` → `/rdr-cluster-reconcile`
     first; `true` means 7.1 already ran over a set containing this RDR). A
     `Final [joint decision →
     <home §-anchor>: <question>]` qualifier is still `Final` for routing —
     surface the home AND the open question so the human sees what is unanswered.
     But check the home first: if it has ANSWERED that question, the scoped
     answer-vs-fences check is owed **before** implement (Stage 7.1) — that check
     is the next step, not `/rdr-implement`.
   - A bare `Draft` that declares `Cluster` is barred from refine until every
     member has completed propose (the tandem barrier) — if a sibling hasn't
     proposed, next is that sibling's `/rdr-propose`, not this RDR's `/rdr-refine`.
   - `Implemented` / `Reverted` / `Abandoned` / `Superseded` → terminal; report the
     post-mortem state, no next command.
   - `Deferred [revisit when <condition>]` → **parked, not terminal**. No post-mortem
     is owed and no next command is due *while the condition holds* — report the
     condition verbatim so the human can judge whether it has fired. If it plainly
     has, next is the stage the RDR stopped at (usually `/rdr-propose`, which is
     what returned no acceptable mechanism).
   - A re-entry qualifier `Draft [revised from Final <date>; re-verify <IDs>]` →
     this is a **scoped backward-edge**; next is `/rdr-resolve NNNN` (it self-scopes
     to the listed IDs). Surface the qualifier so the human knows the run is a delta.
   - bare `Draft` → front-half; use the evidence signals to find the furthest stage.
     **Read the in-record signals, not just folders.** Some stages prove "done" in
     the RDR body, not on disk: Stage 4 Resolve is done when the Critical
     Assumptions are all `Verified`/`Pending`-with-plan **even if no evidence folder
     exists** — a pure source-search resolve (no spikes, no lenses) writes its
     verdicts inline and creates no `<RDR_EVIDENCE>/<slug>/` dir at all. A missing
     evidence folder therefore does **not** mean Resolve hasn't run; check the CA
     verdicts first. Never recommend re-running a stage whose done-signal (folder
     **or** record) is already satisfied — if CAs are all `Verified`, Resolve is
     behind you and next is the first Pre-Lock lens, not `/rdr-resolve`.
     The converse binds equally: an **open `~` is the next command**, not a
     footnote to step past. Refine is the front half's only `~` — when CAs are all
     `Pending`, next is `/rdr-refine`, *then* `/rdr-resolve`.
2. **Bind the `Profile` field first — it is the routing latch, not a Caveats
   footnote.** It maps to an exact lens row: **rdr-common §lens-row** is the
   authority (row, first-lens fork, Determinacy add-on, completion rules) —
   never reconstruct it from memory. The Pre-Lock row you print lists **only
   this profile's lenses**, plus a Determinacy-owed `repeatability` when it
   fires (an owed obligation is never hidden); other off-profile lenses are
   absent, not `–`. A `Draft`
   Profile is provisional (Resolve earns it, Stage 7 latches it) — a hint, never
   a basis for certifying a lens-skip; flag the basis when unearned. If the field
   is absent, infer from the row and flag it (Caveats).
3. **Per-lens for Stage 5**: if some profile lenses ran and others haven't, next is
   the first un-run lens (`/rdr-prelock NNNN <lens>`) — that one command runs the
   lens *and* resolves its findings (review + fix are one cycle now). For
   `repeatability`, the variant follows `Profile`, not the files present:
   `mid`/`large` = lite (only `run-1` then a focused RDR-vs-run diff);
   `foundational`/escalation = full (`run-1/2/3` then `diff`). Point at the next
   missing piece for that variant; the diff session also resolves `diff.md`.
   Escalation is additive and the row never shrinks — §lens-row.

## Output

Be brief. Print:

1. **Header** — `RDR NNNN-<slug> — <Status line verbatim>`.
2. **Stage checklist** — one line per row of the signal table above, verbatim in
   name and order; never invent, split, or rename a row. **Pre-Lock is the single
   `5+6` row** (review+resolve are one cycle) — no separate "Resolve-findings"
   stage; a lens shows `✓` when its folder exists, its resolution certified by the
   next lens's folder or `evidence/reconcile/` in that same row. Mark `✓` (present),
   `–` (not started), or `~` (no durable artifact by design — certified downstream,
   not forgotten), naming the evidence keyed on
   (e.g. `5+6 Pre-Lock  ✓ grounding  ✓ 3amigo` for a mid RDR). For `~`, name the
   downstream signal, not just the verdict
   (`3 Refine  ~ judged done — A3/A5 Verified`); if absent the `~` is open — say
   so and let it own **Next** (`3 Refine  ~ open — CAs all Pending`). Never write
   `~` as done while naming no Stage-4 product.
3. **Next** — the exact command to run, e.g. `Next: /rdr-prelock 0046 critique`.
   If terminal, say so and name the disposition.
4. **Caveats** — only genuinely-open items: a `~` gate whose downstream signal is
   **absent** (a `~` already certified downstream stays in the checklist, never
   here — don't nudge a re-run of a done stage); a re-entry qualifier; and an
   unearned Profile basis (absent → "no Profile; inferred mid"; `Draft` →
   "Profile mid is Seed's estimate — Resolve to confirm"). A `Final` Profile is
   earned → no caveat. And an **unconsulted routing model**, when `intrastate` did
   not resolve.

No writes. Confirm `git status` would be unchanged (you ran only reads).

## No-arg mode

One command, no glob and no per-file read:

```sh
"$RDR_HOME/bin/rdr" status
```

It returns the `Draft`/`Final`-not-yet-`Implemented` set with each Status and
qualifier already split, and each row's facts under it (the signal table above,
evaluated) — so a row needs no follow-up read.
Report each as `NNNN-slug · <Status> · next: /rdr-<stage> NNNN` — each row's `next`
comes from the same branches below (resolve per row only where a row is the one
being acted on; the worklist itself needs no per-row resolve call).
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

If the disk signals here drift from how the stages actually write (a lens renames
its output, a new evidence dir appears), update the signal table in this file and
`rdr-common.md` §evidence. Stage-mechanics changes live in the stage `.md` +
its prompt file, not here.
