---
name: rdr-status
metadata:
  argument-hint: "[NNNN]   # omit to list in-flight RDRs"
description: Use to find out where an RDR is in the flow and what to run next (e.g. "where is RDR 46", "$rdr-status 0046" in Codex, "/rdr-status 0046" in Claude, "what's the next step for the import RDR"). Read-only navigator — derives flow position purely from disk (the evidence folder, the RDR Status line, the artifact dir); never writes. Pairs with the per-stage $rdr-* skills in Codex or /rdr-* skills in Claude that it points you at.
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

The **evidence folder is the primary signal.** For `RDR_SLUG`, check existence and
combine with the RDR's own header. All reads are cheap; delegate nothing unless an
evidence folder is large.

**Two artifact shapes — they nest differently; don't reuse one join for both:**
- **Lenses:** `<RDR_EVIDENCE>/<slug>/evidence/<lens>/` — slug, then a literal
  `evidence/`, then the lens. (The `evidence/` segment is the one most-missed; a
  check at `<slug>/<lens>/` finds nothing and falsely reports the lens un-run.)
- **Spikes:** `<RDR_EVIDENCE>/spikes/<slug>/` — `spikes/` is a top-level sibling,
  slug underneath, **no `evidence/` segment.**

There is no `<round>/<slug>` shape — don't invent one and flag the real tree as
deviating. Stale top-level lens folders may sit loose under `<RDR_EVIDENCE>/`
(e.g. `<RDR_EVIDENCE>/3amigo/`) from an older layout — ignore them; only the
per-slug paths above count.

| Stage | Done-signal on disk |
| --- | --- |
| 1 Seed | RDR file exists; `Status: Draft`; Problem Statement filled (not placeholder) |
| 2 Propose | Proposed Solution / Alternatives / Decision Rationale filled; Critical Assumptions list present (even if Pending) |
| 3 Refine | *human-judged* — infer done if Stage-4 evidence exists or assumptions carry Method/Evidence |
| 4 Resolve | Critical Assumptions all `Verified` or `Pending`-with-plan (the **primary** signal, read from the RDR body); `<RDR_EVIDENCE>/spikes/<slug>/` present when spikes were named (spike shape, not lens shape). A pure source-search resolve names no spikes and writes **no** evidence folder — verdicts are inline; an absent `<slug>/` dir is then expected, not a sign Resolve is unrun. |
| 5+6 Pre-Lock (review+resolve) | which `<RDR_EVIDENCE>/<slug>/evidence/<lens>/` folders exist — per lens (`3amigo`, `critique`, `repeatability`, `cove`), incl. `iter-N`. Review + resolve are one cycle; *resolution is human-judged* — infer a lens converged from the next lens's folder existing, or from `evidence/reconcile/` |
| 7 Reconcile | `<RDR_EVIDENCE>/<slug>/evidence/reconcile/` report exists; assumptions all terminal (no Pending without impl-plan) |
| 8 Finalize | `Status: Final`; the Finalization Gate's five responses present in the RDR; README index row updated |
| 8.1 Cluster | `<RDR_EVIDENCE>/<slug>/evidence/cluster-reconcile/<cluster>/` (only when the RDR is in a cluster) |
| 9 Implement | `{ARTIFACT_DIR}/<slug>/status.md` reads `COMPLETE` or `INCOMPLETE` |

Read in one pass: the RDR's `**Status**:` line (verbatim, including any qualifier),
its Critical Assumptions (count Verified vs Pending), then `ls` each folder at the
exact shape above — lenses + `reconcile` under `<slug>/evidence/`, spikes under
`spikes/<slug>/`, plus `cluster-reconcile/` and `{ARTIFACT_DIR}/<slug>/status.md`.

## How it decides "next"

1. **`Status` first — it is the coarse position.**
   - `Demoted` → the RDR exited at Seed; next is none (refiled as an issue).
   - `Final` → next is `/rdr-implement` (unless a cluster of ≥2 Final-unimplemented
     peers exists → `/rdr-cluster-reconcile` first).
   - `Implemented` / `Reverted` / `Abandoned` / `Superseded` → terminal; report the
     post-mortem state, no next command.
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
2. **Bind the `Profile` field first — it is the routing latch, not a Caveats
   footnote.** It maps to an exact lens-set (the `$RDR_HOME/stages/README.md`
   matrix is the sole authority — never reconstruct it from memory):

   | Profile | Stage-5 lens-set (in order) | After Resolve, next is |
   | --- | --- | --- |
   | `small` | *(none)* | `/rdr-reconcile` (skip Pre-Lock) |
   | `mid` | grounding → 3amigo | `/rdr-prelock NNNN grounding` |
   | `large` | grounding → 3amigo → critique | `/rdr-prelock NNNN grounding` |
   | `foundational` | cove → 3amigo → critique → repeatability | `/rdr-prelock NNNN cove` |

   The Pre-Lock row lists **only this profile's lenses**; off-profile lenses are
   absent, not `–`. Converged = every lens in *this* set ran; the first un-run one
   is the next command. A `Draft` Profile is provisional (Resolve earns it, Stage 8
   latches it) — a hint, never a basis for certifying a lens-skip; flag the basis
   when unearned. If the field is absent, infer from the matrix and flag it (Caveats).
3. **Per-lens for Stage 5**: if some profile lenses ran and others haven't, next is
   the first un-run lens (`/rdr-prelock NNNN <lens>`) — that one command runs the
   lens *and* resolves its findings (review + fix are one cycle now). For
   `repeatability`, point at the next missing piece: `/rdr-prelock NNNN
   repeatability <run>` until `run-1/2/3` exist, then `/rdr-prelock NNNN
   repeatability diff` (which also resolves the `diff.md`).

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
   (`3 Refine  ~ judged done — Stage-4 evidence present`); if that signal is absent
   the `~` is genuinely open — say so (`~ unverified — no Stage-4 evidence yet`).
3. **Next** — the exact command to run, e.g. `Next: /rdr-prelock 0046 critique`.
   If terminal, say so and name the disposition.
4. **Caveats** — only genuinely-open items: a `~` gate whose downstream signal is
   **absent** (a `~` already certified downstream stays in the checklist, never
   here — don't nudge a re-run of a done stage); a re-entry qualifier; and an
   unearned Profile basis (absent → "no Profile; inferred mid"; `Draft` →
   "Profile mid is Seed's estimate — Resolve to confirm"). A `Final` Profile is
   earned → no caveat.

No writes. Confirm `git status` would be unchanged (you ran only reads).

## No-arg mode

List in-flight RDRs: glob the RDR dir, read each `**Status**:` line, and report
every RDR whose Status is `Draft`/`Draft [revised…]` (and any `Final` not yet
`Implemented`) as one line — `NNNN-slug · <Status> · next: /rdr-<stage> NNNN`.
Skip `Implemented`/`Demoted`/`Abandoned`/`Superseded`. Keeps the standing
worklist visible without opening each RDR.

## Self-update

If the disk signals here drift from how the stages actually write (a lens renames
its output, a new evidence dir appears), update the signal table in this file and
`rdr-common.md` §evidence. Stage-mechanics changes live in the stage `.md` +
its prompt file, not here.
