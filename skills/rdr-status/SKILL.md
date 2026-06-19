---
name: rdr-status
argument-hint: "[NNNN]   # omit to list in-flight RDRs"
description: Use to find out where an RDR is in the flow and what to run next (e.g. "where is RDR 46", "/rdr-status 0046", "what's the next step for the import RDR"). Read-only navigator — derives flow position purely from disk (the evidence folder, the RDR Status line, the artifact dir); never writes. Pairs with the per-stage /rdr-* skills it points you at.
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
/rdr-status <NNNN>
/rdr-status            # no arg: list in-flight RDRs (Draft/not-yet-Final) + their next step
```

Read the shared bones in [`rdr-common.md`](rdr-common.md): run **§seam-bind**
then **§rdr-resolve** to bind `$RDR_ENV`, `RDR_PATH`, `RDR_SLUG`, and the evidence
roots. Then derive position as below. Do **not** edit any file.

## What it reads (the disk signals)

The **evidence folder is the primary signal.** For `RDR_SLUG`, check existence and
combine with the RDR's own header. All reads are cheap; delegate nothing unless an
evidence folder is large.

| Stage | Done-signal on disk |
| --- | --- |
| 1 Seed | RDR file exists; `Status: Draft`; Problem Statement filled (not placeholder) |
| 2 Propose | Proposed Solution / Alternatives / Decision Rationale filled; Critical Assumptions list present (even if Pending) |
| 3 Refine | *human-judged* — infer done if Stage-4 evidence exists or assumptions carry Method/Evidence |
| 4 Resolve | Critical Assumptions all `Verified` or `Pending`-with-plan; `<RDR_EVIDENCE>/<slug>/evidence/spikes/` present when spikes were named |
| 5+6 Pre-Lock (review+resolve) | which `<RDR_EVIDENCE>/<slug>/evidence/<lens>/` folders exist — per lens (`3amigo`, `critique`, `repeatability`, `cove`), incl. `iter-N`. Review + resolve are one cycle; *resolution is human-judged* — infer a lens converged from the next lens's folder existing, or from `evidence/reconcile/` |
| 7 Reconcile | `<RDR_EVIDENCE>/<slug>/evidence/reconcile/` report exists; assumptions all terminal (no Pending without impl-plan) |
| 8 Finalize | `Status: Final`; the Finalization Gate's five responses present in the RDR; README index row updated |
| 8.1 Cluster | `<RDR_EVIDENCE>/<slug>/evidence/cluster-reconcile/<cluster>/` (only when the RDR is in a cluster) |
| 9 Implement | `{ARTIFACT_DIR}/<slug>/status.md` reads `COMPLETE` or `INCOMPLETE` |

Read in one pass: the RDR's `**Status**:` line (verbatim, including any qualifier),
its Critical Assumptions section (count Verified vs Pending), then `ls` the
per-lens evidence folders, `reconcile/<slug>/`, `spikes/<slug>/`,
`cluster-reconcile/`, and `{ARTIFACT_DIR}/<slug>/status.md`.

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
2. **Within the front half**, the furthest stage with its done-signal present sets
   the next stage. Read the RDR's **`Profile` field** for which Stage 5 lenses
   apply: `small`'s next after Resolve is **Reconcile**, not Pre-Lock. A Profile on
   a `Draft` is a provisional estimate (Seed's, or a back-fill) — Resolve has not
   yet earned it, so treat it as a hint and never certify a lens-skip off it. If
   absent, infer from the `$RDR_HOME/stages/README.md` matrix. Flag either
   case (Caveats).
3. **Per-lens for Stage 5**: if some profile lenses ran and others haven't, next is
   the first un-run lens (`/rdr-prelock NNNN <lens>`) — that one command runs the
   lens *and* resolves its findings (review + fix are one cycle now). For
   `repeatability`, point at the next missing piece: `/rdr-prelock NNNN
   repeatability <run>` until `run-1/2/3` exist, then `/rdr-prelock NNNN
   repeatability diff` (which also resolves the `diff.md`).

## Output

Be brief. Print:

1. **Header** — `RDR NNNN-<slug> — <Status line verbatim>`.
2. **Stage checklist** — one line per stage 1–9 with `✓` (evidence present),
   `–` (not started), or `~` (no durable artifact by design — certified from a
   downstream signal, not forgotten), and the one-token evidence it keyed on
   (e.g. `5 Pre-Lock  ✓ 3amigo  ✓ critique  – repeatability  – cove`). For a `~`
   stage, name the downstream signal that satisfies it, not just the verdict —
   `3 Refine  ~ judged done (Stage-4 evidence present; Refine edits the draft, no
   folder of its own)` — so the `~` reads as certified-by-downstream, never as an
   omission. If the downstream signal is **absent**, then the `~` is genuinely
   open: say so (`3 Refine  ~ unverified — no Stage-4 evidence yet`).
3. **Next** — the exact command to run, e.g. `Next: /rdr-prelock 0046 critique`.
   If terminal, say so and name the disposition.
4. **Caveats** — only for a `~` gate whose downstream signal is **absent** (the
   genuinely-open case — e.g. Refine with no Stage-4 evidence yet). A `~` that a
   downstream artifact already certifies is **not** a caveat — it belongs in the
   checklist line, not here, so the human isn't nudged to re-run a done stage.
   Also flag a re-entry qualifier, and flag the Profile basis when
   it is not yet earned: an **absent** field you inferred ("no Profile; inferred
   mid"), or a **Draft** Profile still provisional ("Profile mid is Seed's
   estimate — Resolve to confirm"). A `Final` Profile is earned → no caveat.

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
