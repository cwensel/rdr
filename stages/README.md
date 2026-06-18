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
adapted from prompts actually run against RDRs 0011–0039.

## How to use this

**This flow is standalone and project-agnostic.** It carries no project
specifics — no domain, no paths, no corpora. A project plugs in through one
seam: an `.rdr/` folder at its root holding `resources.md` (the evidence
index) and `env.md` (the path map). The project itself stays
RDR-ignorant — `.rdr/` is gitignored, and nothing is written to the project's
`CLAUDE.md` or root `.gitignore`. **Run [Stage 0 — Bootstrap](00-bootstrap.md)
once** to create that folder; after that, the flow runs against any project.

**Quickstart.** From the project root: paste [`00-bootstrap.md`](00-bootstrap.md)
once (creates `.rdr/`), then for each RDR walk Stages 1→8 — each stage file
opens with a **Paste this** block you copy verbatim, filling only `{RDR_PATH}`
(and `{RDR_RESOURCES}`/`{RDR_ENV}`, which default to the two `.rdr/` files). You never
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
front-half resume point. Stage 8 (Implement) is the other resume point — it has
no "next stage" to advance to, terminates at `COMPLETE` or halts at
`INCOMPLETE`, is re-entrant across days from on-disk artifacts, and its only
backward edge is a contract-level spec defect that re-opens the RDR. The stage
file says how.

## The stages

Ordered. Skips are allowed per *Applicability* below — the order never changes.

| # | Stage | File | Run when |
| --- | --- | --- | --- |
| 0 | **Bootstrap** | [00-bootstrap.md](00-bootstrap.md) | Once per project — create the `.rdr/` seam (resources + env), non-invasively |
| 1 | **Seed** | [01-seed.md](01-seed.md) | Starting a brand-new RDR from a kata/idea |
| 2 | **Propose** | [02-propose.md](02-propose.md) | The seeded draft has a problem statement but no chosen approach |
| 3 | **Refine** | [03-refine.md](03-refine.md) | The draft is internally messy — redundant, contradictory, or bloated |
| 4 | **Resolve Assumptions** | [04-resolve-assumptions.md](04-resolve-assumptions.md) | Before any review round — verify every Critical Assumption (research + spikes) |
| 5+6 | **Pre-Lock Review & Resolve** | [05-prelock.md](05-prelock.md) | Draft complete + assumptions verified — per lens (by risk profile), one `/rdr-prelock <lens>` cycle reviews *and* resolves, grounding each finding and looping to convergence (cap 3 → flapping). Skip for a small RDR |
| 6 | **Reconcile Spikes & Assumptions** | [06-reconcile.md](06-reconcile.md) | After all lenses — close every open spike + disturbed assumption before lock |
| 7 | **Finalize** | [07.0-finalize.md](07.0-finalize.md) | Reconcile clean — run the gate and flip to Final |
| 7.1 | **Cluster Reconcile** | [07.1-cluster-reconcile.md](07.1-cluster-reconcile.md) | *Per cluster, not per RDR* — when ≥2 related RDRs are Final & none implemented, reconcile cross-RDR drift before any implements (most RDRs skip) |
| 8 | **Implement** | [08-implement.md](08-implement.md) | RDR is Final — drive it to working, spec-verified code (the flow's terminus) |

Three stages **dispatch into existing `../prompts/` files rather than duplicating
them**: Stage 5 into the [`../prompts/pre-lock/`](../prompts/pre-lock/) lenses;
Stage 7.1 into the [`../prompts/gate/`](../prompts/gate/) cross-RDR prompts
(`pairwise.md` + the whole-set `critique.md`); and Stage 8 into
[`../prompts/implementation/launch.md`](../prompts/implementation/launch.md).
Stage 7 is the Finalization Gate from
[`../README.md`](../README.md#finalization-gate). These map onto the parent
README's workflow steps 1–7: Stages 1–4 are steps 1–3 (Create/Research/Decide),
Stages 5–8 are steps 4–5 (Pre-Lock Review/Lock), Stage 8 is step 6 (Implement),
and the flow ends by handing to step 7 (Close — post-mortem).

```text
                 ┌──────── iterate (re-run same stage) ────────┐
                 ▼                                              │
  kata/idea → [1 Seed] → [2 Propose] → [3 Refine] → [4 Resolve] ┘
                                                        │
                                                        ▼
                       ┌──── per lens: review+resolve cycle ────┐
                       ▼   (loop to convergence; cap 3→flapping) │
              [5+6 Pre-Lock Review & Resolve]  (../prompts/pre-lock)
                       │                                         │
                       └──────────── next lens ──────────────────┘
                                          │ all lenses done
                                          ▼
                          [6 Reconcile Spikes & Assumptions]
                                          │ RECONCILED (else → 2/3/4)
                                          ▼
                                   [7 Finalize] → Final
                                          │
            (cluster only) ──────────────┤ ≥2 related RDRs Final, none impl.
                                          ▼
                          [7.1 Cluster Reconcile]  cross-RDR drift
                                          │ RECONCILED (else → 2/3/4 at scoped
                                          │ depth: re-lock-only/stage/full-flow)
                                          ▼  (hours/days later)
                     [8 Implement] (../prompts/implementation/launch.md)
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
slot can't capture, each finding grounded then edited in (or charted), each
disturbed assumption recorded. Reconcile → every disturbance forced terminal.
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
   rewrite normative contracts. Stage 4 guards the review *entrance*; Stage 6
   guards the *exit* — locking closes only when residual risk is explicitly
   judged acceptable, not silently (obstacle analysis, Letier 2025). Without
   it, a spike that refutes the design, or one quietly deferred past lock,
   slips through. See *Spike lifecycle* below.
3. **The mechanical sweep runs at the exit, in Stage 7 — not as a first
   filter.** It re-checks the same conformance Refine and Resolve already
   establish, so running it *before* the rounds only re-inspects their floor.
   Its value is as a *post-mutation regression sweep*: the rounds and Stage 6
   rewrite the draft, and a cheap mechanical pass confirms none of them
   hollowed a section or disturbed an evidence record before lock. Mining 16
   real runs found its self-reference check never fired once the structured
   Evidence Record and Resolve existed — so it is a regression guard, not a
   defect hunt.

## Spike lifecycle

Spikes are the one concern that **threads through three stages** rather than
living in one, so track them as a cycle, not a per-stage chore:

- **Created (4 Resolve)** — spikes run to verify load-bearing assumptions;
  command + output captured under `{SPIKE_DIR}`.
- **Multiplied & disturbed (5 Pre-Lock)** — a review round (critique especially)
  demands a *new* spike to test a claim it presses on (named, maybe unrun), and
  its fix half flag-as-you-go records any assumption the fix touched. Stage 5
  does **not** run these spikes.
- **Forced terminal (6 Reconcile)** — every named-but-unrun spike and every
  disturbed assumption is run, deferred, or accepted before lock.

The load-bearing invariant — a refutation re-opens the RDR rather than becoming
a wording fix, and an MVV-critical spike never defers past lock — lives at
Stage 6, which is why Reconcile is a forced gate and not a checkbox. Both rules
and the lock-day near-misses that justify them are stated where the driver acts
on them: [`06-reconcile.md`](06-reconcile.md#why-the-two-hard-rules-are-hard).

## Applicability — which stages a given RDR needs

> **Flow-lite (read this and stop, for a small RDR).** A small / single-file /
> non-user-facing RDR runs **1·2·3·4 → 6 → 7 → 8** and *no* analytical lens. If
> that is your RDR, you need only the stage files for those stages plus the Stage
> 7 mechanical sweep — you can skip the cross-RDR drift, lens-tree, cluster, and
> workspace-marker machinery below entirely; they are opt-in by risk profile, not
> required reading. Match the rows below to size the middle; the ends always run.

Seed/Propose/Refine/Reconcile/Finalize always run, and an RDR that locks is
there to be Implemented (8), so that stage runs too unless the RDR is
Abandoned. (A seed judged not RDR-shaped exits at Stage 1 as `Demoted` —
refiled as a plain issue — and runs none of these.) Resolve runs whenever the
RDR has any load-bearing assumption
(almost always). Pre-Lock's fix half runs once per lens that produced findings,
inside the same `/rdr-prelock <lens>` cycle.
The *analytical* rounds inside Stage 5 follow the **risk matrix** in
[`../README.md`](../README.md#applicability-matrix); the mechanical Tooling
sweep is not a round — it runs on every profile as Stage 7's Gate pre-step. The
profile sizes the *middle* (which Stage 5 rounds); the ends (1–4, 6, 7, 8) run
on every profile:

| RDR profile | Stages |
| --- | --- |
| Small / single-file / non-user-facing | 1·2·3·4 → *(skip 5)* → 6 → 7 → 8 |
| Mid / user-facing OR locks a contract | 1·2·3·4 → 5(grounding+3amigo) → 6 → 7 → 8 |
| Large / locks enum·hash·format·grammar·destructive op | 1·2·3·4 → 5(grounding+3amigo+critique) → 6 → 7 → 8 |
| Foundational / cross-RDR producer / spans modules | 1·2·3·4 → 5(cove+3amigo+critique+repeatability) → 6 → 7 → [7.1]† → 8 |

† **Stage 7.1 Cluster Reconcile** is *per cluster, not per RDR* — the `[7.1]†`
marker only flags that a foundational RDR's cross-RDR Pairwise happens there,
not in Stage 5. Single-RDR profiles skip it (7 → 8). See
[Cross-RDR drift](#cross-rdr-drift-stage-71).

**The profile is a recorded latch, not re-inferred each stage.** It lives in the
RDR's `Profile` Metadata field, sized by **blast radius, not word count** — the
max of two axes: the *contract* axis (the count of independent load-bearing
contracts the RDR is sole author of — the Normative Contracts section is the seam
detector; ≥2 across separate seams is a split signal, not a profile) and the
*accretion* axis (the **Seam Lineage** field's prior-point-fix count: ≥2 floors
the profile at `foundational`, a deterministic gate escapable only by a written
accretion disposition). The accretion floor is what stops a "one contract → mid"
sizing from under-gating an accreting seam. Its lifecycle is three-tier, with
provenance
carried by `Status` rather than any qualifier: **Seed** writes a provisional
estimate (an early budget signal — trust it only as a hint); **Resolve** (Stage
4) overwrites it from the now-verified contracts; the **Stage 7 Gate**
re-validates it and it becomes authoritative at the `Draft → Final` flip. A
Profile on a `Draft` is an estimate — the routing stages read the field instead
of re-deriving size, but **never skip the lenses off a Draft's Profile until
Resolve has run**.

A small RDR runs no analytical lens — it has no PM/UX or time-shifted-failure
surface — so it skips Stage 5 and goes Resolve → Reconcile → Finalize; the
Stage 7 mechanical sweep is its only pre-lock conformance check. Stage 6 runs
on every profile — usually a fast confirm that the lenses disturbed nothing;
cheap when the front half was disciplined, the only backstop when it wasn't.
Stage 8 runs whenever an RDR is locked rather than abandoned, and the launch
prompt sizes its own effort (sub-RDR fixes can run Phase 1 inline). If an RDR
is seeded already-coherent (e.g. a tight kata), Propose and Refine may each
collapse to a single pass. The order never changes.

## Cross-RDR drift (Stage 7.1)

The per-RDR stages check one RDR at a time; cross-RDR consistency is a
*set*-level concern no per-RDR gate covers. The flow tolerates cross-RDR
inconsistency during per-RDR work and reconciles it once, at
[Stage 7.1 — Cluster Reconcile](07.1-cluster-reconcile.md), when a cluster of
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
| `{RDR_RESOURCES}` | the per-project **evidence index** (corpora, doc contracts, anchors, domain priors) | exported by the seam marker (`$RDR_RESOURCES`); `.rdr/resources.md` only as the no-marker fallback — see *Where the seam lives* |
| `{RDR_ENV}` | the per-project **path map** (output staging + source-path roots) | exported by the seam marker (`$RDR_ENV`); `.rdr/env.md` only as the no-marker fallback |
| `{SPIKE_DIR}` · `{EVIDENCE_DIR}` · `{ARTIFACT_DIR}` | output staging — may point anywhere (in-project scratch, or a tracked sibling repo) | defined in `{RDR_ENV}` |
| `{IDEA}` | the seed idea — a kata id or one-line description (Stage 1) | — |
| `{RDR_RECORDS}` · `{RDR_A_PATH}` · `{RDR_B_PATH}` | local to [Stage 7.1 Cluster Reconcile](07.1-cluster-reconcile.md) (whole-set Critique + Pairwise), *not* Stage 5 — the cluster directory and the peer pair being compared; *not* the project RDR directory, which Seed derives from `{RDR_ENV}` | supplied per run |

**Arg-header convention (Stages 5–6).** Because the TUI collapses a pasted
prompt to one line, the Pre-Lock stages — the only ones that splice a lens body
and fill more than `{RDR_PATH}` — deploy as **two pastes**: an **arg header**
(`RDR:` / `LENS:` / optional `RUN:`) sent first, then a resolver + verbatim lens
body that reads the header and binds `{RDR_PATH}`, `{EVIDENCE_DIR}` (incl. re-entry
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
- **`{RDR_ENV}` = the path map** — `{SPIKE_DIR}`, `{EVIDENCE_DIR}`, `{ARTIFACT_DIR}`,
  and the source-path roots an audit starts from. A stage that only writes
  output loads `{RDR_ENV}` alone, not the corpus index.

**All project-specific context — corpus names and domain priors in
`{RDR_RESOURCES}`, paths in `{RDR_ENV}` — lives in those two files, not in any stage
prompt.** A different project runs this flow by writing its own seam data files
(`.rdr/resources.md` and `.rdr/env.md` by default).

**Where the seam lives — read this.** The seam location is resolved from a
single **workspace marker**, never guessed and never hardcoded in a stage
prompt. The marker is a small sourceable file — `.rdr-workspace` — at the
**workspace parent** (the directory holding the sibling repos), and it exports
the seam paths (`RDR_ENV`, `RDR_RESOURCES`, and the repo roots). The canonical
resolver, **worktree-invariant** because it keys off git topology rather than
cwd:

```sh
GC=$(git rev-parse --git-common-dir) && GC=$(cd "$GC" && pwd -P)   # split: empty capture would make cd a no-op
PROJECT=$(dirname "$GC"); WS=$(dirname "$PROJECT")
# nearest wins: a repo-local marker (inside .rdr/) overrides the shared workspace one
if   [ -f "$PROJECT/.rdr/workspace" ]; then . "$PROJECT/.rdr/workspace"   # repo-local scope (default)
elif [ -f "$WS/.rdr-workspace" ];      then . "$WS/.rdr-workspace"        # workspace scope (shared)
else echo "no marker in $PROJECT/.rdr or $WS — run /rdr-init" >&2; fi
```

`git rev-parse --git-common-dir` resolves to the **main** repo's `.git` even
from inside a worktree (a gitignored file at a repo root is *not* checked out
into its worktrees — verified — so the marker must live *above* the repos and be
reached by walking up from git topology, not by `./`). `dirname` twice gives the
workspace root `$WS`; sourcing `$WS/.rdr-workspace` binds every path. This is the
same topology the ship skills already use, now expressed as **data in one file**
instead of a derivation duplicated across stage prompts and skills.

Not an env var: env vars do not reliably reach spawned agents or worktree
processes (`direnv` only fires on interactive `cd`), so they can't be the source
of truth. The marker file + the topology derivation does reach them.

Consequences, all absorbed by the existing parameters (no per-stage path
literals):

- `{RDR_RESOURCES}` / `{RDR_ENV}` are whatever the marker exports — for a
  **pinned-seam** project a tracked external dir (e.g. a consumer that pins them
  into a sibling repo), for an unpinned project the gitignored `.rdr/`.
- `{SPIKE_DIR}` / `{EVIDENCE_DIR}` / `{ARTIFACT_DIR}` are whatever that project's
  `{RDR_ENV}` says — they need not sit under `.rdr/`, and may be tracked. The
  flow stays correct because every stage names these *via the parameter*, never
  by a literal path.

So `.rdr/` is only the **no-marker fallback** — the zero-footprint default for a
project that has not bootstrapped a workspace marker. With a marker present, no
stage ever probes `.rdr/`. Either way, do not rewrite stage prompts as `../`
hops back into wherever the flow docs sit.

**Two scopes, nearest wins.** The **default** is a repo-local marker at
`$PROJECT/.rdr/workspace` (inside the gitignored `.rdr/`, so no project-level
`.gitignore` edit) — each repo runs its own RDR process. §seam-bind prefers it over a
shared `$WS/.rdr-workspace`, which several sibling repos opt into with
`/rdr-init --workspace` (the `newcoinc/` setup: retrofit/process/flow share one seam).
A repo-local marker overrides the shared one for that repo without re-pointing it.

> **Worked example — a pinned-seam consumer.** A consumer may pin its seam this
> way and **retire `.rdr/` entirely** — no gitignored scratch seam at all.
> `{RDR_RESOURCES}`/`{RDR_ENV}` live in a tracked seam repo (e.g. a sibling
> `flow/context/`), evidence under that repo's `rdr/evidence/`, while the RDRs
> themselves stay in the consumer's own repo (e.g. `<consumer>/rdr/cli/`). Three
> distinct trees across two repos — see that consumer's `rdr-env.md`. The flow
> prompts below are unchanged by this; only the `{RDR_ENV}` values differ.
> (`.rdr/` remains the default for any project that has *not* set up a tracked
> seam repo.)

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
  these to a sub-agent that returns the *verdict + evidence pointer*
  (`path::Symbol`, or spike command + output path), not raw hits or whole files.
  Judgment stays
  in the main context; fetching is pushed out and discarded. Stage 8
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
- **Anchor by symbol, not by line.** Every source pointer — in an RDR, a spike
  record, or a sub-agent's evidence return — is a greppable `path::Symbol`, never
  a bare `file:line`. Line numbers rot the moment code above them changes; the
  symbol stays findable and is how the reader re-locates anyway. Use a commit-SHA
  permalink only for audit/traceability. Write fewer, durable anchors rather than
  many volatile ones — a stale anchor is an active distractor, not a harmless one.
- **Single-source each contract.** State a load-bearing contract in exactly one
  authoritative place. Where a rule lives both in a stage's prompt body and in
  prose, the **prompt body** (`../prompts/`) is the source of truth and the stage
  doc points to it; a stage doc's Review gate may *name* a check but never restate
  its mechanics. Two copies of one contract drift into self-contradiction — the
  most common internal-defect class — so the cure is deletion, not a lint to keep
  them in sync.

The RDR process is deliberately token-frugal — code is generated last, after
the design is settled, so tokens aren't spent implementing a design research
will overturn. This doctrine extends that frugality to the authoring loop.

## Resources vs. memory

`{RDR_RESOURCES}` and `{RDR_ENV}` resolve (via the workspace marker — see *Where
the seam lives*) to **files on disk**, not the per-session memory feature. The corpus shows why: resource-loaded fix-pass sessions spent their
first 3–4 tool calls *always* re-reading `docs/principles.md` + `ls
docs/internal/` — boilerplate a manifest removes. Beyond saving those calls, a
file beats memory on counts that matter for a spec process:

- **Explicit + inspectable** — you can open it and see exactly what context a
  stage draws on; memory is opaque and recalled non-deterministically.
- **Project-local + editable in place** — it sits at a known path in the
  project root the flow is run from; add a row when a stage keeps reaching for a
  resource it lacks, remove one when it stops earning its place.

Both are **per-machine working config, not versioned docs** — for an unpinned
project the gitignored `.rdr/` files, for a pinned one a tracked location its
`{RDR_ENV}` names; either way resolved through the marker, not guessed.

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

Every stage writes via the `{SPIKE_DIR}` / `{EVIDENCE_DIR}` / `{ARTIFACT_DIR}`
names defined under *Output staging* in `{RDR_ENV}` (marker-resolved). For an
unpinned project the pre-lock evidence (`{SPIKE_DIR}`, `{EVIDENCE_DIR}`) is
gitignored scratch under `.rdr/`; a pinned project may put it in a tracked tree.
Implementation artifacts (`{ARTIFACT_DIR}`) stay tracked beside
the RDR. The flow stays correct because it names the locations, never hardcodes
them.

Evidence is **per-RDR-first** (symmetric with `{ARTIFACT_DIR}`): each RDR owns
`<RDR_EVIDENCE>/<rdr-slug>/evidence/`, with one folder per lens inside it.
`{EVIDENCE_DIR}` is the fully-bound per-lens dir
`<RDR_EVIDENCE>/<rdr-slug>/evidence/<lens>/`, and a lens writes its *element* files
there rather than one flat file — e.g. `…/0038-policies/evidence/3amigo/` holds
`persona-1-pm.md`, `persona-2-implementer.md`, `persona-3-qa.md`,
`consolidation.md`. Stage 6 reads that folder's consolidation/findings/diff file.
(Stage 7.1 is the exception — it is per-cluster, not per-RDR, so it writes
`<RDR_EVIDENCE>/<rdr-slug>/evidence/cluster-reconcile/<cluster>/`.)

## The skills (live)

Each stage is wrapped by a **`/rdr-*` skill** that supplies its params and runs
its prompt — so driving an RDR is `/rdr-<stage> NNNN`, not hand-pasting a block.
The skills are thin: each runs §seam-bind + §rdr-resolve (shared bones in
[`skills/rdr-common.md`](skills/rdr-common.md)), runs its stage's prompt file under
[`../prompts/stages/`](../prompts/stages/), then prints the gate + the next command.

**One skill per stage. No orchestrator.** A `/rdr-flow` walker was considered and
rejected: it would either accumulate every stage's context (bloat) or hand off
between sub-agents (losing the iterative, human-in-the-loop context a driver needs
to drop in and correct at any stage). Granularity is the point — each invocation
holds only its own stage's state.

| Skill | Param | Stage |
| --- | --- | --- |
| `/rdr-init` | `[--interactive\|--defaults\|--reconfigure]` (run at project root) | 0 — **one-time seam bootstrap**; writes `rdr-env`/`rdr-resources` + the workspace marker. Bare run infers locations + discloses them; flags force prompting / defaults / re-decide. The only skill that does *not* run §seam-bind (it creates the seam) |
| `/rdr-doctor` | none (run anywhere) | **health check** — read-only; verifies the five-var contract binds, the engine layout resolves, the evidence root is reachable, and no symlink is broken. Run after `/rdr-init` or an engine upgrade |
| `/rdr-status` | `NNNN` (or none) | **navigator** — read-only, derives position from disk |
| `/rdr-seed` | `"<idea>"` / kata id | 1 (allocates the next number) |
| `/rdr-propose` | `NNNN` | 2 |
| `/rdr-refine` | `NNNN` | 3 |
| `/rdr-resolve` | `NNNN` | 4 |
| `/rdr-prelock` | `NNNN <lens>` | 5+6 — reviews **and** resolves one lens, looping to convergence (`lens ∈ grounding·3amigo·critique·repeatability·cove`; repeatability also resolves its `diff.md`) |
| `/rdr-reconcile` | `NNNN` | 6 |
| `/rdr-finalize` | `NNNN` | 7 (one gated prompt — READY locks, NOT READY flips nothing) |
| `/rdr-cluster-reconcile` | cluster / RDR list | 7.1 (per cluster, not per RDR) |
| `/rdr-implement` | `NNNN` | 8 |

**Contract.** Skills take the **RDR number** (`0046`), not a path — they resolve
number→file against the RDR dir `{RDR_ENV}` names. **Re-entry is self-detected**,
not a flag: the stage prompts already read the RDR `Status:` line
(`Draft [revised from Final …; re-verify <IDs>]`) or `status.md` and scope
themselves. **`/rdr-status` tracks position from disk** — the evidence folder, the
`Status` line, the artifact dir — with **no ledger** (the resources-vs-memory
doctrine: a derived position can't drift from the artifacts).

**Homing.** The skill sources live here in this RDR-process repo at
[`skills/`](skills/), next to these stage docs and the prompts they run — the RDR
flow is this repo's deliverable, so its skills version with it. A consumer makes
them invocable through a **discovery farm** under its own `.claude/skills/`, via
relative symlinks `<consumer>/.claude/skills/rdr-* -> <…>/rdr/skills/rdr-*`
(the same farm may also hold a consumer's other skill symlinks). Drive RDR work
from a consumer cwd; the workspace-marker resolver is location-invariant, so the
skills reach the RDR directory and the evidence folder by path regardless of cwd.

## Provenance

The pre-lock prompts adapt the SpecDrivenDev battery (see
[`../prompts/README.md`](../prompts/README.md)); the implementation orchestrator
(Stage 8) is `../prompts/implementation/launch.md`, which predates this mining.
The front-half stages are mined from this project's own Claude session corpus —
the recurring shape of how *this author* actually drove RDRs from idea to Final
and on into implementation. Literature anchors for the front-half moves are
noted per stage; back-half and implementation anchors live in the existing
prompt files.
