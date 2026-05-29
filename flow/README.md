# RDR Flow — A Repeatable Recipe from Idea to Implementation

This directory is the **driver** for taking one RDR from an idea (a kata, a
gap, a sentence) all the way to **implemented code** — through **Final** (the
locked specification prompt) and into the implementation that locking exists to
serve. Final is the pivot, not the finish: the flow falls naturally into
[`prompts/implementation/launch.md`](../prompts/implementation/launch.md) as
its last stage.

`prompts/` already holds the *back half* (pre-lock review rounds + the
Finalization Gate) and the *implementation orchestrator* (launch.md). What was
missing — and was only ever reconstructed by copy/pasting prior chat history —
is the *front half*: how an RDR gets seeded, refined, and has its assumptions
verified before any review round is worth running, plus the seam where the
locked spec hands off to implementation. This directory captures **the whole
arc as one ordered recipe**, each stage a pasteable prompt + a review gate +
an advance-when condition.

The flow is **mined from practice, not invented** — every stage prompt is
adapted from prompts actually run against real RDRs across a multi-RDR project
(the mining methodology is described in [`../RESEARCH.md`](../RESEARCH.md)).

## How to use this

**This flow is standalone and project-agnostic.** It carries no project
specifics — no domain, no paths, no corpora. A project plugs in through one
seam: an `_rdr/` folder at its root holding `rdr-resources.md` (the evidence
index) and `rdr-env.md` (the path map). The project itself stays
RDR-ignorant — `_rdr/` is gitignored, and nothing is written to the project's
`CLAUDE.md` or root `.gitignore`. **Run [Stage 0 — Bootstrap](00-bootstrap.md)
once** to create that folder; after that, the flow runs against any project.

**Quickstart.** From the project root: paste [`00-bootstrap.md`](00-bootstrap.md)
once (creates `_rdr/`), then for each RDR walk Stages 1→9 — each stage file
opens with a **Paste this** block you copy verbatim, filling only `{RDR_PATH}`
(and `{RDR_RESOURCES}`/`{RDR_ENV}`, which default to the two `_rdr/` files). You never
reconstruct a prompt or decide what context to load — the block already names it.

You drive the loop; the model runs one stage at a time. For each stage:

1. **Run** — copy the stage's **Paste this** block, fill the blanks it names.
2. **Review** — read the output against the stage's **Review gate**. This is
   your checkpoint.
3. **Iterate or advance** — gate not met → re-run the *same* stage (often with
   a narrowing instruction); met → advance. Each stage file ends with a
   pointer to the next, so you never track order by hand.

Stages 1–8 advance linearly, usually in one sitting — but a **seed can sit
idle** between Seed (1) and Propose (2) long enough for its references and scope
to go stale, so Propose's step 0 re-validates the seed first and doubles as the
front-half resume point. Stage 9 (Implement) is the other resume point — it has
no "next stage" to advance to, terminates at `COMPLETE` or halts at
`INCOMPLETE`, is re-entrant across days from on-disk artifacts, and its only
backward edge is a contract-level spec defect that re-opens the RDR. The stage
file says how.

## The stages

Ordered. Skips are allowed per *Applicability* below — the order never changes.

| # | Stage | File | Run when |
| --- | --- | --- | --- |
| 0 | **Bootstrap** | [00-bootstrap.md](00-bootstrap.md) | Once per project — create the `_rdr/` seam (resources + env), non-invasively |
| 1 | **Seed** | [01-seed.md](01-seed.md) | Starting a brand-new RDR from a kata/idea |
| 2 | **Propose** | [02-propose.md](02-propose.md) | The seeded draft has a problem statement but no chosen approach |
| 3 | **Refine** | [03-refine.md](03-refine.md) | The draft is internally messy — redundant, contradictory, or bloated |
| 4 | **Resolve Assumptions** | [04-resolve-assumptions.md](04-resolve-assumptions.md) | Before any review round — verify every Critical Assumption (research + spikes) |
| 5 | **Pre-Lock Review** | [05-prelock.md](05-prelock.md) | Draft complete + assumptions verified — run the single-RDR lenses matching the risk profile (skip for a small RDR) |
| 6 | **Pre-Lock Resolve** | [06-prelock-resolve.md](06-prelock-resolve.md) | After each lens — fix findings one by one; flag any assumption a fix disturbs |
| 7 | **Reconcile Spikes & Assumptions** | [07-reconcile.md](07-reconcile.md) | After all lenses — close every open spike + disturbed assumption before lock |
| 8 | **Finalize** | [08.0-finalize.md](08.0-finalize.md) | Reconcile clean — run the gate and flip to Final |
| 8.1 | **Cluster Reconcile** | [08.1-cluster-reconcile.md](08.1-cluster-reconcile.md) | *Per cluster, not per RDR* — when ≥2 related RDRs are Final & none implemented, reconcile cross-RDR drift before any implements (most RDRs skip) |
| 9 | **Implement** | [09-implement.md](09-implement.md) | RDR is Final — drive it to working, spec-verified code (the flow's terminus) |

Three stages **dispatch into existing `../prompts/` files rather than duplicating
them**: Stage 5 into the [`../prompts/pre-lock/`](../prompts/pre-lock/) lenses;
Stage 8.1 into the [`../prompts/gate/`](../prompts/gate/) cross-RDR prompts
(`pairwise.md` + the whole-set `critique.md`); and Stage 9 into
[`../prompts/implementation/launch.md`](../prompts/implementation/launch.md).
Stage 8 is the Finalization Gate from
[`../README.md`](../README.md#finalization-gate). These map onto the parent
README's workflow steps 1–7: Stages 1–4 are steps 1–3 (Create/Research/Decide),
Stages 5–8 are steps 4–5 (Pre-Lock Review/Lock), Stage 9 is step 6 (Implement),
and the flow ends by handing to step 7 (Close — post-mortem).

```text
                 ┌──────── iterate (re-run same stage) ────────┐
                 ▼                                              │
  kata/idea → [1 Seed] → [2 Propose] → [3 Refine] → [4 Resolve] ┘
                                                        │
                                                        ▼
                              ┌──────── per lens ────────┐
                              ▼                          │
                     [5 Pre-Lock Review] → [6 Pre-Lock Resolve]
                              │  (../prompts/pre-lock)    │
                              └──── next lens ────────────┘
                                          │ all lenses done
                                          ▼
                          [7 Reconcile Spikes & Assumptions]
                                          │ RECONCILED (else → 2/3/4)
                                          ▼
                                   [8 Finalize] → Final
                                          │
            (cluster only) ──────────────┤ ≥2 related RDRs Final, none impl.
                                          ▼
                          [8.1 Cluster Reconcile]  cross-RDR drift
                                          │ RECONCILED (else → 2/3/4 at scoped
                                          │ depth: re-lock-only/stage/full-flow)
                                          ▼  (hours/days later)
                     [9 Implement] (../prompts/implementation/launch.md)
                          │  ◄── resume across gaps from status.md
        spec defect ──────┤
        → 2/3/4, re-lock  │ COMPLETE
                          ▼
                   Close (step 7): post-mortem
```

## The loop

The recipe is a **filter cascade with review gates**: each stage removes a
class of defect the next assumes gone. Seed → a template-conformant draft.
Propose → one committed approach, premortemed before it is written in.
Refine → no internal contradiction.
Resolve → every load-bearing claim verified. Pre-Lock → the gaps a template
slot can't capture. Resolve Findings → each finding edited in, each disturbed
assumption recorded. Reconcile → every disturbance forced terminal.
Finalize → a mechanical regression sweep over the now-settled draft, then the
written Gate, then lock. Implement → the locked spec turned into code whose
tests are built *from* the spec, so the design is finally tested against
reality; the only defect this last filter can't fix in place is a contract
defect, which re-opens the RDR rather than patching the code.

This shape — few phases, each producing an artifact that constrains the next,
with a human review at each checkpoint — is the spec-driven-development
default (Piskala 2026, *Spec-Driven Development*). The phase *count* is not
the lever: restructuring an inspection process changes cost, not defect yield;
the **review method at each gate** is what catches defects (Porter et al.
1998, via Lanubile 2000). So this flow keeps each stage that owns a distinct
artifact + a distinct gate method, and spends its economy on cutting prose,
not gates.

Two orderings carry the most weight; both are mined from the corpus and
backed by the literature:

1. **Resolve (4) precedes Pre-Lock (5).** Reviewing an RDR whose assumptions
   are still `Pending` wastes the round — it critiques claims that may not
   hold. Perspective-based reading builds and verifies a model *before*
   hunting defects (Lanubile 2000).
2. **Reconcile (7) sits between the rounds and the lock.** The review rounds
   are *assumption-disturbing*: critique adds load-bearing claims, fix-passes
   rewrite normative contracts. Stage 4 guards the review *entrance*; Stage 7
   guards the *exit* — locking closes only when residual risk is explicitly
   judged acceptable, not silently (obstacle analysis, Letier 2025). Without
   it, a spike that refutes the design, or one quietly deferred past lock,
   slips through. See *Spike lifecycle* below.
3. **The mechanical sweep runs at the exit, in Stage 8 — not as a first
   filter.** It re-checks the same conformance Refine and Resolve already
   establish, so running it *before* the rounds only re-inspects their floor.
   Its value is as a *post-mutation regression sweep*: the rounds and Stage 7
   rewrite the draft, and a cheap mechanical pass confirms none of them
   hollowed a section or disturbed an evidence record before lock. Mining 16
   real runs found its self-reference check never fired once the structured
   Evidence Record and Resolve existed — so it is a regression guard, not a
   defect hunt.

## Spike lifecycle

Spikes are the one concern that **threads through four stages** rather than
living in one, so track them as a cycle, not a per-stage chore:

- **Created (4 Resolve)** — spikes run to verify load-bearing assumptions;
  command + output captured under `{SPIKE_DIR}`.
- **Multiplied (5 Pre-Lock)** — a review round (critique especially) demands a
  *new* spike to test a claim it presses on. The spike gets named; it may not
  yet run.
- **Disturbed (6 Resolve Findings)** — a fix touches or introduces a
  load-bearing claim; flag-as-you-go records the assumption needing a
  (re-)spike. Stage 6 does **not** run it.
- **Forced terminal (7 Reconcile)** — every named-but-unrun spike and every
  disturbed assumption is run, deferred, or accepted before lock.

The load-bearing invariant — a refutation re-opens the RDR rather than becoming
a wording fix, and an MVV-critical spike never defers past lock — lives at
Stage 7, which is why Reconcile is a forced gate and not a checkbox. Both rules
and the lock-day near-misses that justify them are stated where the driver acts
on them: [`07-reconcile.md`](07-reconcile.md#why-the-two-hard-rules-are-hard).

## Applicability — which stages a given RDR needs

Seed/Propose/Refine/Reconcile/Finalize always run, and an RDR that locks is
there to be Implemented (9), so that stage runs too unless the RDR is
Abandoned. Resolve runs whenever the RDR has any load-bearing assumption
(almost always). Resolve Findings runs once per round that produced findings.
The *analytical* rounds inside Stage 5 follow the **risk matrix** in
[`../README.md`](../README.md#applicability-matrix); the mechanical Tooling
sweep is not a round — it runs on every profile as Stage 8's Gate pre-step. The
profile sizes the *middle* (which Stage 5 rounds); the ends (1–4, 7, 8, 9) run
on every profile:

| RDR profile | Stages |
| --- | --- |
| Small / single-file / non-user-facing | 1·2·3·4 → *(skip 5)* → 7 → 8 → 9 |
| Mid / user-facing OR locks a contract | 1·2·3·4 → 5(3amigo)·6 → 7 → 8 → 9 |
| Large / locks enum·hash·format·grammar·destructive op | 1·2·3·4 → 5(3amigo+Critique)·6 → 7 → 8 → 9 |
| Foundational / cross-RDR producer / spans modules | 1·2·3·4 → 5(3amigo+critique+repeatability+cove)·6 → 7 → 8 → [8.1]† → 9 |

† **Stage 8.1 Cluster Reconcile** is *per cluster, not per RDR* — the `[8.1]†`
marker only flags that a foundational RDR's cross-RDR Pairwise happens there,
not in Stage 5. Single-RDR profiles skip it (8 → 9). See
[Cross-RDR drift](#cross-rdr-drift-stage-81).

A small RDR runs no analytical lens — it has no PM/UX or time-shifted-failure
surface — so it skips Stage 5 and goes Resolve → Reconcile → Finalize; the
Stage 8 mechanical sweep is its only pre-lock conformance check. Stage 7 runs
on every profile — usually a fast confirm that the lenses disturbed nothing;
cheap when the front half was disciplined, the only backstop when it wasn't.
Stage 9 runs whenever an RDR is locked rather than abandoned, and the launch
prompt sizes its own effort (sub-RDR fixes can run Phase 1 inline). If an RDR
is seeded already-coherent (e.g. a tight kata), Propose and Refine may each
collapse to a single pass. The order never changes.

## Cross-RDR drift (Stage 8.1)

The per-RDR stages check one RDR at a time; cross-RDR consistency is a
*set*-level concern no per-RDR gate covers. The flow tolerates cross-RDR
inconsistency during per-RDR work and reconciles it once, at
[Stage 8.1 — Cluster Reconcile](08.1-cluster-reconcile.md), when a cluster of
related RDRs is all Final-and-unimplemented. That stage file owns the mechanics,
the rationale (Finkelstein/Easterbrook), and the Final→Draft routing — including
the *re-entry scope* call (re-lock-only / stage-scoped / full-flow), sized to the
defect so a simple change never drags a full flow behind it. It is *per cluster,
not per RDR*, and most RDRs skip it.

## Parameters

Every stage prompt is parameterized so it can be pasted with the blanks
filled (or wrapped by a skill that supplies them):

| Param | Meaning | Default |
| --- | --- | --- |
| `{RDR_PATH}` | the RDR file being driven | — (required) |
| `{RDR_RESOURCES}` | the per-project **evidence index** (corpora, doc contracts, anchors, domain priors) | `_rdr/rdr-resources.md`, resolved from the invoking session's project root |
| `{RDR_ENV}` | the per-project **path map** (output staging + source-path roots) | `_rdr/rdr-env.md`, same root |
| `{SPIKE_DIR}` · `{FLOW_DIR}` · `{ARTIFACT_DIR}` | output staging | defined in `{RDR_ENV}` |
| `{IDEA}` | the seed idea — a kata id or one-line description (Stage 1) | — |
| `{RDR_DIR}` · `{RDR_A_PATH}` · `{RDR_B_PATH}` | local to [Stage 8.1 Cluster Reconcile](08.1-cluster-reconcile.md) (whole-set Critique + Pairwise), *not* Stage 5 — the cluster directory and the peer pair being compared; *not* the project RDR directory, which Seed derives from `{RDR_ENV}` | supplied per run |

**Arg-header convention (Stages 5–6).** Because the TUI collapses a pasted
prompt to one line, the Pre-Lock stages — the only ones that splice a lens body
and fill more than `{RDR_PATH}` — deploy as **two pastes**: an **arg header**
(`RDR:` / `LENS:` / optional `RUN:`) sent first, then a resolver + verbatim lens
body that reads the header and binds `{RDR_PATH}`, `{FLOW_DIR}` (incl. re-entry
`iter-N`), and `<N>`. Only the header values are typed; lens files in
[`../prompts/pre-lock/`](../prompts/pre-lock/) paste verbatim. Other stages
still fill `{RDR_PATH}` inline.

The two project files split by *kind*, not by stage, so each stage loads only
what it needs:

- **`{RDR_RESOURCES}` = the evidence index** — corpora, doc contracts, external
  anchors, and a *Domain priors* section. The research-touching stages (4, 6,
  7) open with "read `{RDR_RESOURCES}`" instead of re-listing corpora inline;
  **Propose (2) and Refine (3) read only its Domain-priors section** to ground
  *which* approaches are on the table and *which* side of a contradiction is
  right — without the heavy verification manifest.
- **`{RDR_ENV}` = the path map** — `{SPIKE_DIR}`, `{FLOW_DIR}`, `{ARTIFACT_DIR}`,
  and the source-path roots an audit starts from. A stage that only writes
  output loads `{RDR_ENV}` alone, not the corpus index.

**All project-specific context — corpus names and domain priors in
`{RDR_RESOURCES}`, paths in `{RDR_ENV}` — lives in those two files, not in any stage
prompt.** A different project runs this flow by writing its own
`_rdr/rdr-resources.md` and `_rdr/rdr-env.md`.

**Where `_rdr/` lives — read this.** `_rdr/` is in the **project root of the
Claude session that runs the prompts** (the working checkout you drive the flow
from), *not* in the worktree where these flow documents are stored. The two are
usually different trees. So every reference here to `_rdr/rdr-resources.md`,
`_rdr/rdr-env.md`, `{SPIKE_DIR}`, or `{FLOW_DIR}` is a path **relative to the
invoking session's project root**, resolved at run time — never relative to
this file's location. (Running from the project root is the flow's standing
precondition; that is also what makes the project's hidden memory ambient — see
*Resources vs. memory*.)
Paste stage prompts from that project root and the paths resolve directly; do
not rewrite them as `../` hops back into wherever the flow docs happen to sit.

## Doctrine (applies to every stage)

Stated in full here. The prose around each stage (Goal, gate, advance-when)
assumes it; the pasteable blocks restate only the one-line operative form
(e.g. "be brief; ultrathink on complexity"), because a block you copy verbatim
must stand alone — the reader pastes the block, not this README.

- **Be brief; ultrathink on complexity.** Terse reasoning and tight edits are
  the norm; drop into ultrathink only for load-bearing or genuinely complex
  design. Brevity is never traded for a worse outcome — *spend tokens where
  they change the result, nowhere else.* Dropping into ultrathink on
  complexity is the primary quality lever; brevity everywhere else pays for it
  (the "minimum rigor that removes ambiguity" rule — Piskala 2026).
- **Delegate heavy reads to a sub-agent.** Corpus searches, dependency/source
  spelunking, and reading several round-output files crowd out the
  design reasoning. The read-heavy stages (4, 6, 7; the round runs in 5) push
  these to a sub-agent that returns the *verdict + evidence pointer* (file:line,
  or spike command + output path), not raw hits or whole files. Judgment stays
  in the main context; fetching is pushed out and discarded. Stage 9
  ([`../prompts/implementation/launch.md`](../prompts/implementation/launch.md))
  is this doctrine at its strongest — its orchestrator *never* reads the RDR,
  source, or test files; phase sub-agents do. Those orchestrator rules are
  stricter than the authoring stages on purpose; the flow does not loosen them.
- **A Draft is editable in place; that is what Draft means.** While Status is
  Draft, rewrite to reflect the current world: *replace* superseded content,
  delete what the world made wrong, fold any "rescope"/"refinement" note into
  the live text. Never keep a superseded decision as a "retained as history"
  section, a "what changed since" preamble, or a per-section "this is now wrong"
  annotation — that is change-history narration, and it is the most common way
  a Draft bloats. The "never amend RDRs" rule binds only **after Lock** (Final /
  Implemented): a *frozen* RDR is the immutable prompt, code is the source of
  truth. The two never conflict — they apply to different statuses. Check
  `Status:` before declining an edit on freeze grounds.
- **No change-history in the document.** Keep the rationale for decisions;
  never narrate the edits that produced them.

The RDR process is deliberately token-frugal — code is generated last, after
the design is settled, so tokens aren't spent implementing a design research
will overturn. This doctrine extends that frugality to the authoring loop.

## Resources vs. memory

`{RDR_RESOURCES}` and `{RDR_ENV}` resolve to **files in the project tree**
(`_rdr/rdr-resources.md`, `_rdr/rdr-env.md`), not the per-session memory
feature. The corpus shows why: resource-loaded fix-pass sessions spent their
first 3–4 tool calls *always* re-reading the same project principles doc and
listing the same internal docs directory — boilerplate a manifest removes.
Beyond saving those calls, a
file beats memory on counts that matter for a spec process:

- **Explicit + inspectable** — you can open it and see exactly what context a
  stage draws on; memory is opaque and recalled non-deterministically.
- **Project-local + editable in place** — it sits at a known path in the
  project root the flow is run from; add a row when a stage keeps reaching for a
  resource it lacks, remove one when it stops earning its place.

Both files live under `_rdr/` in the invoking project's root and are
**gitignored** (per-machine working config, not versioned docs) — so they do
not travel in review diffs or arrive with a fresh checkout; each project root
keeps its own `_rdr/`.

**The direction is one-way: memory → resources, by hand.** `rdr-resources.md`
is the *authoritative* evidence index; the memory feature is ambient
reinforcement, never the source of truth for what a stage consults. When a
memory pointer proves load-bearing for the flow — a corpus a stage keeps
reaching for, a competitor-comparison bias — **promote it into the file**, the
way the corpora table and Domain-priors competitor bias were themselves
promoted from memory during mining. Do *not* auto-generate the file from
memory: that would make the explicit artifact a cache of the hidden one and
re-introduce the non-determinism it exists to escape. Use memory for *durable
facts about you and the project*; use `{RDR_RESOURCES}`/`{RDR_ENV}` for *the
inspectable, standing record of where an RDR stage should look and write*. They
don't compete.

## Where stage outputs go

Every stage writes via the `{SPIKE_DIR}` / `{FLOW_DIR}` / `{ARTIFACT_DIR}`
names defined under *Output staging* in `{RDR_ENV}`
(`_rdr/rdr-env.md`). Pre-lock evidence (`{SPIKE_DIR}`, `{FLOW_DIR}`) is
gitignored scratch under `_rdr/`; implementation artifacts (`{ARTIFACT_DIR}`)
stay tracked beside the RDR. The flow stays correct because it names the
locations, never hardcodes them.

`{FLOW_DIR}` is **lens-first, then per-RDR**: `_rdr/<lens>/<rdr-slug>/`, and a
lens writes its *element* files there rather than one flat file — e.g.
`_rdr/3amigo/0001-example/` holds `persona-1-pm.md`, `persona-2-implementer.md`,
`persona-3-qa.md`, `consolidation.md`. Stage 6 reads that folder's
consolidation/findings/diff file. (Stage 8.1 is the exception — it is
per-cluster, not per-RDR, so it writes `_rdr/cluster-reconcile/<cluster>/`.)

## Converting stages to skills (later)

Each stage file is skill-shaped (a parameterized prompt + a self-contained
gate + an advance condition — except Stage 9, whose gate terminates at
`COMPLETE`/`INCOMPLETE` rather than advancing), so a `/rdr-seed`,
`/rdr-resolve`, `/rdr-prelock-resolve`, … skill can wrap one stage and supply
its params, and a `/rdr-flow` orchestrator can walk the stages pausing at each
gate. Stage 9 would wrap the already-standalone launch prompt (`/rdr-implement`)
rather than re-author it; Stage 8.1 (`/rdr-cluster-reconcile`) takes a cluster,
not one RDR. Highest-value first: `/rdr-resolve` and `/rdr-prelock-resolve`
(most-repeated, most resource-heavy). Deferred until the flow is vetted in use.

## Provenance

The pre-lock prompts adapt a spec-driven-development prompt battery (see
[`../prompts/README.md`](../prompts/README.md) and the citations in
[`../RESEARCH.md`](../RESEARCH.md)); the implementation orchestrator
(Stage 9) is `../prompts/implementation/launch.md`, which predates this mining.
The front-half stages are mined from a real project's own AI session corpus —
the recurring shape of how the author actually drove RDRs from idea to Final
and on into implementation. Literature anchors for the front-half moves are
noted per stage; back-half and implementation anchors live in the existing
prompt files. [`../RESEARCH.md`](../RESEARCH.md) consolidates all of them.
