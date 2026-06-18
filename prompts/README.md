# RDR Prompts

Operational prompts for the RDR lifecycle.

The directory is organized by **where a prompt fires in the flow**, not by
filename: `flow/` holds the front-half stage prompts (Seed → Reconcile →
Finalize); `pre-lock/` holds the Stage 5 analytical lenses; `gate/` holds the
Stage 7 / 7.1 lock-gate prompts; `implementation/` drives a locked RDR. Every
[`../flow/`](../flow/) stage runs one of these (or, for the front half, its
`flow/NN-*.prompt.md`); the stage `.md` is the human operating manual, the prompt
file is the pasteable body the `/rdr-*` skill runs.

## Flow — the front-half stage prompts (`flow/`)

The pasteable bodies for the linear front-half stages, **extracted verbatim** from
each [`../flow/`](../flow/) stage doc's `Paste this` block so a skill (or a hand
driver) runs the prompt without re-parsing the doc. Each binds `{RDR_PATH}` /
`{RDR_RESOURCES}` / `{RDR_ENV}` / `{FLOW_DIR}` from the workspace marker.

| Stage | File | Run by |
| --- | --- | --- |
| 1 Seed | [flow/01-seed.prompt.md](flow/01-seed.prompt.md) | `/rdr-seed` |
| 2 Propose | [flow/02-propose.prompt.md](flow/02-propose.prompt.md) | `/rdr-propose` |
| 3 Refine | [flow/03-refine.prompt.md](flow/03-refine.prompt.md) | `/rdr-refine` |
| 4 Resolve | [flow/04-resolve.prompt.md](flow/04-resolve.prompt.md) | `/rdr-resolve` |
| 5+6 Pre-Lock Resolve | [flow/05-prelock-resolve.prompt.md](flow/05-prelock-resolve.prompt.md) | `/rdr-prelock` (the fix half of the lens cycle) |
| 7 Reconcile | [flow/06-reconcile.prompt.md](flow/06-reconcile.prompt.md) | `/rdr-reconcile` |
| 7 Finalize | [flow/07-finalize.prompt.md](flow/07-finalize.prompt.md) | `/rdr-finalize` (one gated prompt — READY locks) |

Stage 5 (Pre-Lock) has no `flow/` prompt — it dispatches into the `pre-lock/`
lenses below; Stages 8.1 and 9 dispatch into `gate/` and `implementation/`. The
skills live beside the stage docs at [`../flow/skills/`](../flow/skills/).

## Pre-Lock — validating a draft (`pre-lock/`)

The analytical lenses of the [Pre-Lock Review](../README.md#pre-lock-review)
process. Run between Decide and Lock; the Finalization Gate is the final check.
The **single-RDR** lenses are numbered for run order; each is one file, one
paste block, reading only `{RDR_PATH}` (grounding also reads the source the RDR
cites). Grounding runs first wherever a contract is locked — standalone at
Mid/Large, embedded as CoVe's Step 0 at Foundational.

| # | Lens | File | Cost | Skip when |
| --- | --- | --- | --- | --- |
| 0 | Grounding | [pre-lock/0-grounding.md](pre-lock/0-grounding.md) | ~5–10 min | RDR locks no contract (small) — or it is run inside CoVe (foundational) |
| 1 | 3amigo | [pre-lock/1-3amigo.md](pre-lock/1-3amigo.md) | ~30 min | trivial single-file RDRs |
| 2 | Critique | [pre-lock/2-critique.md](pre-lock/2-critique.md) | ~20–30 min, dual-model | RDR is purely additive — locks no enum/hash/format/grammar |
| 3 | Repeatability | [pre-lock/3-repeatability.md](pre-lock/3-repeatability.md) | 3 runs + diff | RDR locks no public API/signature/data model |
| 4 | CoVe | [pre-lock/4-cove.md](pre-lock/4-cove.md) | ~20–30 min | trivial single-file RDRs |

See the parent README's *Pre-Lock Review* section for the applicability matrix
that says which lenses to run on which RDR profile.

After each lens, the author makes a fix pass on the RDR draft. Re-run any lens whose findings caused a substantial rewrite.

## Gate — locking an RDR (`gate/`)

These two are **not** pre-lock lenses — they fire at the lock gates, not Stage
5, and are dispatched by file (the flow stage points at them, it does not inline
them). Each carries content that earns its standalone file: a runnable rubric or
a reusable output contract.

- [**tooling-pass.md**](gate/tooling-pass.md) — the *mechanical pre-step of the
  Finalization Gate*, run on **every** RDR at [Stage
  8](../flow/07.0-finalize.md), just before the Gate's written responses, as a
  post-mutation regression sweep (the four CHECK blocks live here — Stage 7
  runs them verbatim). ~5 min via AI, seconds when scripted; never skipped. It
  is also the spec a future `tooling-pass` script implements against.
- [**pairwise.md**](gate/pairwise.md) — the *cross-RDR* contradiction scan, run
  per plausibly-interacting pair at [Stage 7.1
  Cluster-reconcile](../flow/07.1-cluster-reconcile.md) (post-Final, per
  cluster), never as a per-RDR pre-lock lens — it needs two settled (Final)
  RDRs to compare. Carries the `TYPE/A-QUOTE/B-QUOTE/SEVERITY` output contract
  the run must emit verbatim.

## Implementation — driving a locked RDR

| Phase | File | Notes |
| --- | --- | --- |
| Implementation launch | [implementation/launch.md](implementation/launch.md) | Drives a fresh AI session through Phases 0–3 of a locked RDR; produces file-backed traceability artifacts next to the RDR. |

The implementation prompt treats the RDR as immutable. If implementation reveals
the RDR is wrong, abandon implementation and iterate on the RDR — do not edit it
during implementation.

In the [`../flow/`](../flow/README.md) recipe this prompt is **Stage 8
(Implement)**, the flow's terminus — [`flow/08-implement.md`](../flow/08-implement.md)
dispatches into it (it does not duplicate it) the same way Stage 5 dispatches
into the pre-lock rounds above.

## Adapting these prompts

The pre-lock prompts adapt earlier general-purpose spec-fitness prompts;
adaptations are placeholder-only (`{SPEC_A}` → `{RDR_PATH}`; inline `<spec>`
block → file reference). Each prompt file cites its source.

If a prompt produces unhealthy signal repeatedly (over-agreement, generic
findings, no concrete passages), it likely needs adversarial reframing before
re-running — not endless reruns of the same prompt.
