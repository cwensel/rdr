# Stage 5 — Pre-Lock Review (and Resolve)

**Goal**: catch the defect classes a template slot can't absorb —
heterogeneous PM/UX gaps, time-shifted failures, cross-RDR contract leaks — **and
resolve them in the same cycle**.

Review and resolve are **one cycle per lens**, not two hand-alternated steps: run
the lens, ground its findings, fix the draft, loop until that lens converges or
flapping is declared. The `/rdr-prelock NNNN <lens>` skill owns the whole cycle —
it dispatches into the lens prompts in
[`../prompts/pre-lock/`](../prompts/pre-lock/) for the review half (it doesn't
redefine them) and runs
[`../prompts/stages/05-prelock-resolve.prompt.md`](../prompts/stages/05-prelock-resolve.prompt.md)
for the fix half (mechanics in *Resolve (the fix half)* below). Each lens is one
file, one paste block. Pick the lens set by risk profile (below), and for each
lens run its cycle to convergence before advancing to the next.

**Run when**: the draft is complete and **Stage 4 has verified its
assumptions** — running these lenses against unverified claims wastes them.

**Re-entry pass (demotion second pass).** A qualified Status line —
`Draft [revised from Final <date>; re-verify <IDs> — <reason>]` (07.1's demotion
signal) — means a prior iteration's lens outputs already sit under `{EVIDENCE_DIR}`.
Don't overwrite them or stall asking; run as a **new iteration**, **delta-scoped to
the `re-verify <IDs>`**, not a full re-run (which regenerates equivalent findings
against a mostly-unchanged draft). The paste block detects this and sets the folder.

## Which lenses (the risk matrix)

From [`../README.md`](../README.md#applicability-matrix). These are the
*analytical* lenses, run in cost-order — a filter cascade. The mechanical
Tooling sweep is **not** here: it moved to the Gate's pre-step in
[Stage 7](07.0-finalize.md), because the lenses and Stage 6 reconcile rewrite the
draft and a mechanical sweep is only meaningful *after* the last mutation.

| RDR profile | Lenses (in order) |
| --- | --- |
| Small / single-file / non-user-facing | *(none — straight to Stage 6, then the Stage 7 sweep + Gate)* |
| Mid / user-facing OR locks a contract | grounding → 3amigo ‡ |
| Large / locks enum·hash·format·grammar·destructive | grounding → 3amigo → critique ‡ |
| Foundational / cross-RDR / spans modules | cove → 3amigo → critique → repeatability |

‡ *+ repeatability-lite if the Determinacy trigger fires (below).*

**Grounding runs first** (whenever present). It is the only lens that checks the
RDR against *source*; running it ahead of the personas keeps them from ratifying
a frame that's false against the codebase. At `mid`/`large` it is the standalone
[grounding Step-0](../prompts/pre-lock/0-grounding.md) (a cheap deterministic
sweep); at `foundational` it is Step 0 *inside* [cove](../prompts/pre-lock/4-cove.md)
(which subsumes it), so cove leads. The cost-order cascade still holds for the
remaining 3amigo → critique → repeatability.

**Accretion floor (deterministic).** Profile is blast-radius, not local diff
size. If the RDR's `Seam Lineage` (Metadata) carries **≥2 closed prior
point-fixes** at the locus, the profile is floored at **Foundational** — a seam
with prior point-fixes is the matrix's `cross-RDR` trigger. This is a mechanical
count read from the field (filled at Seed from `kata-scope-review`), not a
judgment; the only escape is the written accretion disposition in that field. It
is the lever that pulls an accreting `small`/`mid` RDR up to the grounding lens.

**Determinacy trigger (repeatability-lite).** A second mechanical gate, read from
the RDR's **Normative Contracts**, not its profile. If a locked contract is
*algorithmic* — its **output depends on step ordering**, or it defines
**parse/deparse, import/export, compose/decompose, hashing, identity, or migration**
behavior, or a **data-model field whose ownership/semantics could be inferred more
than one way**, or the **MVV rests on multi-step transformation fidelity** — add
**repeatability-lite** (one alternate-model reconstruction + a focused diff; see
[the lens](../prompts/pre-lock/3-repeatability.md#repeatability-lite-one-alternate-model-reconstruction))
to the profile's lenses, down to `mid`. This is a cue read from the contract's
*kind*, not a fresh judgment: a contract that names one of those verbs fires it; one
that does not, does not. The escape is a one-line written disposition in the lens
folder (`determinacy: n/a — <reason>`), mirroring the accretion escape. It does
**not** fire for CLI-flag/UX/surface changes, additive un-ordered config,
doc/wording, or pure plumbing with no transform — these have no step-ordering or
field-ownership ambiguity to diff. Foundational still runs the **full** ×3 lens
unchanged; the trigger only *adds* mid/large coverage at the lite tier, and the lite
diff escalates to the full lens on the criteria the lens names.

**Conditional mini-checks (structural triggers).** Four cheap checks that fire only
on a named cue in the draft — *not* lenses, *not* profile-tiered, *not* always-on.
Each fires when its cue is present and writes a **compact table into the RDR** (the
decision belongs in the spec); a draft with no cue gets no table and no note. They
surface, before lock, defect classes that otherwise escape to implementation/triage.

| Mini-check | Fires when the draft has… | Table the RDR must carry |
|---|---|---|
| source-authority census | a fallback path, derived/propagated output, or ≥2 source-of-truth candidates (`fallback`, `derived`, `propagate`, "source of truth", dispatcher/resolver/classifier, sibling/parallel arm) | `authority`: input/decision × writer · readers · call sites · sibling arms · which is canonical |
| test-discriminability | an MVV/oracle that passes by absence-of-error, no-op, or fixture-name match ("did not error", exit-0, substring/absence oracle) | `oracle`: each MVV row × "fails if X is wrong because Y" + the negative/failing control |
| round-trip / fidelity | import/export, parse/deparse, compose/decompose, serialize, snapshot/drift, migrate/rollback, inverse | `fidelity`: operation × byte/value-equality invariant OR the justified weaker invariant + named lossy-exemption sites |
| disposition | drop / ignore / filter / skip over input classes, or set exit/op outcomes | `disposition`: input class × exit code · event/error · artifact/op minted · silent-vs-loud |

Fire **only** on the named cue — do not add these tables to an RDR that lacks it.
Each is distinct from the Stage 8 impl-time gates: those catch the defect *during*
implementation (an unnamed surface, a green-against-stub test, a declared round-trip),
whereas these force the RDR to *specify* the class before lock so it never reaches
implementation. The grounding lens's "inverse sibling path" grep is a different check —
the authority census is the *named-arm census*, not the inverse-rule sweep.

| Lens | Prompt file | What it uniquely catches |
| --- | --- | --- |
| grounding | [`../prompts/pre-lock/0-grounding.md`](../prompts/pre-lock/0-grounding.md) | Codebase claims false against source; new rules a sibling path already makes |
| 3amigo | [`../prompts/pre-lock/1-3amigo.md`](../prompts/pre-lock/1-3amigo.md) | PM/UX gaps, first-hour implementer questions, untestable clauses |
| critique | [`../prompts/pre-lock/2-critique.md`](../prompts/pre-lock/2-critique.md) | Time-shifted failures, frozen-at-lock invariants. **Dual-model.** |
| repeatability | [`../prompts/pre-lock/3-repeatability.md`](../prompts/pre-lock/3-repeatability.md) | Under-specified signatures/APIs (generate ×3, diff) |
| cove | [`../prompts/pre-lock/4-cove.md`](../prompts/pre-lock/4-cove.md) | Grounding (Step 0) + internal silences/contradictions (verify-independently) |

All are **single-RDR** lenses — each reads only `{RDR_PATH}` (grounding also
reads the source the RDR cites), independent of the others, so they are peer
files (numbered 0–4 for run order), not a composite round. Cross-RDR
contradiction is *not* a pre-lock lens — it needs two settled (Final) RDRs, so
it lives at [Stage 7.1 Cluster-reconcile](07.1-cluster-reconcile.md). A
small/single-file RDR runs no lens (no contract to ground, no PM/UX or
time-shifted surface); the Stage 7 sweep is its only pre-lock check.

**Output convention.** Evidence is **per-RDR-first**: each RDR owns
`<RDR_EVIDENCE>/<rdr-slug>/evidence/`, with one folder per lens inside it.
`{EVIDENCE_DIR}` is the fully-bound per-lens dir
(`<RDR_EVIDENCE>/<rdr-slug>/evidence/<lens>/`); each lens writes its *element* files
there, shape lens-specific. On a **re-entry pass** (qualified Status, above)
`{EVIDENCE_DIR}` gains an iteration segment — `…/evidence/<lens>/iter-N/` — so a
second pass never overwrites the first. A first pass may write the element files
directly under `…/evidence/<lens>/` (that loose set *is* iteration 1).

- `…/evidence/3amigo/` → `persona-1-pm.md`, `persona-2-implementer.md`,
  `persona-3-qa.md`, `consolidation.md`
- `…/evidence/critique/` → `critique.md` (+ `critique-modelB.md` on a dual-model run)
- `…/evidence/repeatability/` → full: `run-1.md`, `run-2.md`, `run-3.md`, `diff.md`;
  lite: `run-1.md`, `diff.md`. `run-1.md`'s header carries a `variant: lite|full`
  line (beside `model:`) — the durable record of intent a cleared/model-switched
  session reads instead of re-deriving; extra runs beside a `lite` header are a mismatch.
- `…/evidence/cove/` → `findings.md`

The resolve half *reads* the lens's `{EVIDENCE_DIR}` folder (the consolidation /
findings / diff file) instead of you pasting a findings list — that folder is the
hand-off from the review half to the fix half, both inside the one cycle.

## Paste this (once per lens in your profile's set)

Two pastes, neither edited. The TUI collapses a pasted block to one line, so
set the values in an **arg header** (Paste 1, its own message), then paste the
resolver + the chosen lens body verbatim (Paste 2). The lens body comes
straight from `../prompts/pre-lock/` — its `{RDR_PATH}`/`{EVIDENCE_DIR}` are left as
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
  - {RDR_ENV} = the seam path map; it defines the lens-output base. Resolve its
    location via the workspace marker (see the flow README *Where the seam
    lives*): `WS=$(dirname "$(dirname "$(cd "$(git rev-parse --git-common-dir)"
    && pwd -P)")")` then `. "$WS/.rdr-workspace"` → `$RDR_ENV`. The base for this
    run is whatever {RDR_ENV}'s {EVIDENCE_DIR} row resolves to for this
    <rdr-slug>/<lens> (per-RDR-first: `<rdr-slug>/evidence/<lens>/`). Only if no
    marker and no {RDR_ENV} exist does the generic default apply.
  - {EVIDENCE_DIR} from the RDR's `**Status**:` line, under that resolved base: a
    re-entry qualifier (`Draft [revised from Final <date>; re-verify <IDs> —
    <reason>]`) → the `<rdr-slug>/evidence/<lens>/iter-N/` subfolder, N = 1 + the
    highest existing iter-* (loose files = iteration 1; never overwrite one), and
    scope to the `re-verify <IDs>` delta per any `## Refinement Context` note, not
    the whole draft. Otherwise the `<rdr-slug>/evidence/<lens>/` base itself (first pass).
Having read {RDR_ENV}, run the lens body against {RDR_PATH}, writing its element
files to {EVIDENCE_DIR}. The body's {RDR_PATH}/{EVIDENCE_DIR}/<N> are already bound.

--- paste the chosen lens's prompt block below, verbatim (text only, no fence) ---
```

A lens with multiple prompt blocks (critique, repeatability): paste the one it
names, re-send Paste 2 per model run (`RUN:` sets the run number). If lens
output is heavy or you spawn a dual-model pass, have the reviewer sub-agent
return its findings list, not a re-dump of the RDR.

## Resolve (the fix half)

The cycle's fix half turns a lens's findings into edits — grounded, scoped, and
dispositioned, without flapping. Mechanics (the prompt
[`../prompts/stages/05-prelock-resolve.prompt.md`](../prompts/stages/05-prelock-resolve.prompt.md)
enforces them; don't re-derive):

- **Grounding gate (before any edit).** Check each finding against `main` (does the
  cited symbol/behavior exist?), `{RDR_RESOURCES}` (principles + contract docs +
  named corpora), and the RDR's own decided text (a re-raise of a settled call?).
  A finding that fails any ground is **wrong** — dismiss it, don't edit to satisfy
  it. This is the anti-flapping core.
- **Origin anchor (anti-plank).** The lens's first-pass findings are the ledger;
  each fix traces to a ledger entry. A finding tracing to none is **net-new
  scope**. Re-runs delta-scope to open entries — never a fresh full critique of
  your own edits.
- **Durable disposition (no silent drops).** Every finding exits one way: **fixed**
  / **dismissed-with-cite** / **charted-to-successor** (a `Charted:` note in the
  lens folder, or flagged for `/rdr-seed` if substantial — never absorbed into this
  RDR; that is the scope-expansion wormhole).
- **Tiebreaker-reduction gate.** Ultrathink on load-bearing / cross-subsystem /
  structural / principle-touching / intent-conflicting findings and collapse the
  fork with reasoning + evidence. Escalate to the human only when the evidence is
  genuinely indeterminate.
- **Flag-as-you-go.** A fix that touches/adds a load-bearing claim flips a Verified
  assumption back to Pending, or adds a new `A-N` Pending — Stage 6 closes them.

## Review gate (the cycle, per lens)

1. **Findings vs the lens's "Expected signal"** (its prompt file). Healthy =
   concrete, named passages. Unhealthy (generic advice, over-agreement, identical
   persona lists) → switch model and re-run; don't resolve a bad pass.
2. **Resolve — grounded** (see *Resolve (the fix half)* above). Every finding is
   grounded against `main` + `{RDR_RESOURCES}` + the RDR's own decided text
   *before* it edits, dispositioned (fixed / dismissed-with-cite /
   charted-to-successor — no silent drops), anchored to the **origin ledger**, and
   tiebreakers collapsed with reasoning + evidence rather than handed to you.
   Editing the *draft* is legitimate — the no-edit rule binds only after Final.
3. **Loop or stop.** Substantial fix → re-run the lens **delta-scoped to open
   ledger entries** (under `iter-N/`), then resolve again; small fix doesn't need
   it. **Cap = 3**: three iterations still surfacing net-new findings against a
   barely-changed draft = the plank problem — the skill stops with
   `stopped:verdict-flapping` and surfaces it once; the cure is a human look or a
   model switch, not a fourth pass.
4. **Converged → next lens** in the set. **Critique is model-aware**: a lone
   single-model `critique.md` does **not** converge a `foundational` RDR (dual-model
   required, or recorded single-model fallback); on re-entry, compare the existing
   evidence's `Model:` stamp (§model-stamp) to this session's — a different model is
   the second pass to run, never "already complete."

If critique or cove surfaces that an *assumption* was wrong (not just
under-documented), flag-as-you-go captures it and **Stage 6 reconciles it** before
lock. If the refutation forces a redesign, **return to Stage 4** now.

## Advance when

Every analytical lens in the RDR's profile has converged (findings resolved or
charted), with the disturbed-assumption list carried forward to Stage 6. (Small
profile: no lens runs — advance immediately to Stage 6.)

→ Per lens: one `/rdr-prelock NNNN <lens>` cycle (review + resolve in the one
loop). All lenses done: [06-reconcile.md](06-reconcile.md) →
[07.0-finalize.md](07.0-finalize.md).
