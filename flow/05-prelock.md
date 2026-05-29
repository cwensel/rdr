# Stage 5 — Pre-Lock Review

**Goal**: catch the defect classes a template slot can't absorb —
heterogeneous PM/UX gaps, time-shifted failures, cross-RDR contract leaks.

This stage **dispatches into the lens prompts in
[`../prompts/pre-lock/`](../prompts/pre-lock/)** — it doesn't redefine them.
Each lens is one file, one paste block. Pick the lens set by risk profile
(below), paste each lens's body in order, and run
[Stage 6](06-prelock-resolve.md) after each.

**Run when**: the draft is complete and **Stage 4 has verified its
assumptions** — running these lenses against unverified claims wastes them.

**Re-entry pass (demotion second pass).** A qualified Status line —
`Draft [revised from Final <date>; re-verify <IDs> — <reason>]` (08.1's demotion
signal) — means a prior iteration's lens outputs already sit under `{FLOW_DIR}`.
Don't overwrite them or stall asking; run as a **new iteration**, **delta-scoped to
the `re-verify <IDs>`**, not a full re-run (which regenerates equivalent findings
against a mostly-unchanged draft). The paste block detects this and sets the folder.

## Which lenses (the risk matrix)

From [`../README.md`](../README.md#applicability-matrix). These are the
*analytical* lenses, run in cost-order — a filter cascade. The mechanical
Tooling sweep is **not** here: it moved to the Gate's pre-step in
[Stage 8](08.0-finalize.md), because the lenses and Stage 7 reconcile rewrite the
draft and a mechanical sweep is only meaningful *after* the last mutation.

| RDR profile | Lenses (in order) |
| --- | --- |
| Small / single-file / non-user-facing | *(none — straight to Stage 7, then the Stage 8 sweep + Gate)* |
| Mid / user-facing OR locks a contract | 3amigo |
| Large / locks enum·hash·format·grammar·destructive | 3amigo → critique |
| Foundational / cross-RDR / spans modules | 3amigo → critique → repeatability → cove |

| Lens | Prompt file | What it uniquely catches |
| --- | --- | --- |
| 3amigo | [`../prompts/pre-lock/1-3amigo.md`](../prompts/pre-lock/1-3amigo.md) | PM/UX gaps, first-hour implementer questions, untestable clauses |
| critique | [`../prompts/pre-lock/2-critique.md`](../prompts/pre-lock/2-critique.md) | Time-shifted failures, frozen-at-lock invariants. **Dual-model.** |
| repeatability | [`../prompts/pre-lock/3-repeatability.md`](../prompts/pre-lock/3-repeatability.md) | Under-specified signatures/APIs (generate ×3, diff) |
| cove | [`../prompts/pre-lock/4-cove.md`](../prompts/pre-lock/4-cove.md) | Internal silences + contradictions (verify-independently) |

All four are **single-RDR** lenses — each reads only `{RDR_PATH}`, independent
of the others, so they are peer files (numbered 1–4 for run order), not a
composite round. Cross-RDR contradiction is *not* a pre-lock lens — it needs
two settled (Final) RDRs, so it lives at
[Stage 08.1 Cluster-reconcile](08.1-cluster-reconcile.md). A small/single-file
RDR runs no lens (no PM/UX or time-shifted surface); the Stage 8 sweep is its
only pre-lock check.

**Output convention.** `{FLOW_DIR}` is **lens-first, then per-RDR**
(`_rdr/<lens>/<rdr-slug>/`); each lens writes its *element* files there, shape
lens-specific. On a **re-entry pass** (qualified Status, above) `{FLOW_DIR}` gains
an iteration segment — `_rdr/<lens>/<rdr-slug>/iter-N/` — so a second pass never
overwrites the first. A first pass may write the element files directly under
`_rdr/<lens>/<rdr-slug>/` (that loose set *is* iteration 1).

- `3amigo/<rdr-slug>/` → `persona-1-pm.md`, `persona-2-implementer.md`,
  `persona-3-qa.md`, `consolidation.md`
- `critique/<rdr-slug>/` → `critique.md` (+ `critique-modelB.md` on a dual-model run)
- `repeatability/<rdr-slug>/` → `run-1.md`, `run-2.md`, `run-3.md`, `diff.md`
- `cove/<rdr-slug>/` → `findings.md`

Stage 6 then *reads* the lens's `{FLOW_DIR}` folder (the consolidation /
findings / diff file) instead of you pasting a findings list — that folder is
the hand-off from 5 to 6.

## Paste this (once per lens in your profile's set)

Two pastes, neither edited. The TUI collapses a pasted block to one line, so
set the values in an **arg header** (Paste 1, its own message), then paste the
resolver + the chosen lens body verbatim (Paste 2). The lens body comes
straight from `../prompts/pre-lock/` — its `{RDR_PATH}`/`{FLOW_DIR}` are left as
written; the resolver binds them.

**Paste 1 — arg header** (edit only the values):

```text
RDR: {RDR_PATH}
LENS: <3amigo | critique | repeatability | cove>
RUN: <1–3, repeatability only — omit otherwise>
```

**Paste 2 — resolver, then verbatim lens body:**

```text
From the arg header above, bind for this session and the lens body below:
  - {RDR_PATH} = RDR:; <rdr-slug> = its filename stem; <lens> = LENS:; <N> = RUN:.
  - {RDR_ENV}/{RDR_RESOURCES} default to _rdr/rdr-env.md / _rdr/rdr-resources.md.
  - {FLOW_DIR} from the RDR's `**Status**:` line: a re-entry qualifier
    (`Draft [revised from Final <date>; re-verify <IDs> — <reason>]`) →
    _rdr/<lens>/<rdr-slug>/iter-N/, N = 1 + the highest existing iter-* (loose
    files = iteration 1; never overwrite one), and scope to the `re-verify <IDs>`
    delta per any `## Refinement Context` note, not the whole draft. Otherwise
    _rdr/<lens>/<rdr-slug>/ (first pass).
Read {RDR_ENV}, then run the lens body against {RDR_PATH}, writing its element
files to {FLOW_DIR}. The body's {RDR_PATH}/{FLOW_DIR}/<N> are already bound.

--- paste the chosen lens's prompt block below, verbatim (text only, no fence) ---
```

A lens with multiple prompt blocks (critique, repeatability): paste the one it
names, re-send Paste 2 per model run (`RUN:` sets the run number). If lens
output is heavy or you spawn a dual-model pass, have the reviewer sub-agent
return its findings list, not a re-dump of the RDR.

## Review gate (after EACH lens)

1. **Read findings against the lens's "Expected signal"** in its prompt file.
   Healthy = concrete, named passages. Unhealthy (generic advice,
   over-agreement, identical persona lists) → switch model and re-run that
   lens; don't proceed on a bad pass.
2. **Resolve the findings — run [Stage 6](06-prelock-resolve.md).** It reads
   the lens's `{FLOW_DIR}` folder, edits the *draft* (legitimate — the no-edit rule
   binds only after Final), and records any assumption a fix disturbs.
3. **Re-run the lens if the fix was substantial** (a rewrite can introduce
   new gaps); a small fix doesn't need it.
4. **Then advance to the next lens** in the set.

If critique or cove surfaces that an *assumption* was wrong (not just
under-documented), Stage 6's flag-as-you-go captures it and **Stage 7
reconciles it** before lock. If the refutation forces a redesign, **return to
Stage 4** now rather than waiting.

## Advance when

Every analytical lens in the RDR's profile has run and its findings are
resolved via Stage 6, with the disturbed-assumption list carried forward to
Stage 7. (Small profile: no lens runs — advance immediately to Stage 7.)

→ Per lens: [06-prelock-resolve.md](06-prelock-resolve.md). All lenses
done: [07-reconcile.md](07-reconcile.md) → [08.0-finalize.md](08.0-finalize.md).
