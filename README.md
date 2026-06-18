# Recommendation Decisioning Records (RDRs)

RDRs are specification prompts built through iterative research and refinement.

Unlike [ADRs](https://adr.github.io) which document decisions already made, RDRs evolve during planning as the problem
statement becomes more refined through research and exploration and alternatives emerge with one as the best option.

**Core Purpose**: Capture both the final recommendation and supporting evidence to prevent purpose drift during
implementation.

## Install

This repo is a Claude Code plugin (and a self-hosted marketplace). Two ways to
get the `/rdr-*` skills into a project:

- **Marketplace plugin** — add the marketplace, install, then bind your project:

  ```
  /plugin marketplace add cwensel/rdr
  /plugin install rdr@rdr
  /rdr:rdr-init        # run once at the project root to bootstrap the seam
  ```

  The engine ships inside the plugin; `/rdr-init` records its path
  (`$CLAUDE_PLUGIN_ROOT`) in the workspace marker, so re-run it after a plugin
  upgrade to refresh that path.

- **Sibling checkout** — clone this repo beside the consumer repos and link the
  skills into each consumer's `.claude/skills/`; `/rdr-init` then resolves the
  engine as the sibling `$WS/rdr`.

Either way, `/rdr-init` writes the seam files + workspace marker, scaffolds the
RDR home and its index README, and offers the optional SessionStart seam hook.
Then `/rdr:rdr-seed <idea>` starts the first record.

## Indexes

RDRs are indexed per project. Each consumer keeps its RDR instances in its own
repo (the engine never holds them), and that consumer's README carries the
authoritative Status/Priority table, co-located with the RDR files it lists —
e.g. a `cli/README.md` listing a `cli/` RDR set.

This README is the cross-project **process manual** for authoring RDRs: the
workflow, finalization gate, pre-lock review rounds, and post-mortem process
below apply to every project's RDRs regardless of which index lists them.

## Workflow

1. **Create** (Draft) — Document problem, initial constraints, technical environment
2. **Research** (Draft) — Investigate, add findings, refine problem statement, explore alternatives. Label all findings
   as Verified/Documented/Assumed
3. **Decide** (Draft) — Select approach, document rationale
4. **Pre-Lock Review** (Draft) — Apply the review rounds matching this RDR's risk profile, fix the draft, then run the
   Finalization Gate. See [Pre-Lock Review](#pre-lock-review)
5. **Lock** (Final) — All gate items answered; RDR is the specification prompt for implementation
6. **Implement** (Final) — Use locked RDR as spec; do not edit during implementation
7. **Close** (Implemented | Reverted | Abandoned | Superseded) — once the RDR is implemented and in use, update status
   and write the post-mortem. See [Post-Mortem Process](#post-mortem-process)

**If implementation reveals RDR is wrong**: Abandon implementation, iterate on RDR with lessons learned, start fresh.

**Driving the workflow**: [`stages/`](stages/README.md) is a runnable recipe for steps 1–6 — one parameterized prompt
per stage (seed → propose → refine → resolve-assumptions → pre-lock → reconcile → finalize → implement), each with a
review gate and an advance-when condition (the implement stage terminates at COMPLETE/INCOMPLETE instead), so an RDR can
be driven from idea to implemented code without reconstructing the prompts from memory, then handed to step 7 (Close).
The stages are mined from this project's own session history; Stage 5 dispatches into the [`prompts/`](prompts/) pre-lock
rounds and Stage 8 into [`prompts/implementation/launch.md`](prompts/implementation/launch.md) rather than duplicating
them.

## Status Definitions

- **Draft** — During planning/research phase
- **Final** — Locked, ready for or during implementation
- **Implemented** — Implementation complete
- **Reverted** — Implemented then undone (document why)
- **Abandoned** — RDR not implemented
- **Superseded** — Replaced by another RDR

## When to Create an RDR

- Complex multi-step feature implementation
- Significant technical debt or architectural decisions with trade-offs
- Framework/library workarounds requiring substantial research
- Any work benefiting from documented alternatives before committing to writing code

When scoping work, account for the full surface — not just the happy path. Include Day 2 operations (list, info, delete,
verify, backup for each resource created), error handling, and cross-cutting integration points.

## Research Guidance

Consult during planning:

- Requirements and standards documentation (cite specific sections)
- Dependency source code — see [Verifying load-bearing claims](#verifying-load-bearing-claims) below
- Existing codebase modules that overlap with the proposed design (audit for reuse before designing new components)
- Existing codebase patterns or idioms
- Run spikes/POCs for opaque services where source code is unavailable

Include in RDR: Citations, code snippets demonstrating capabilities/limitations, hard-to-earn discoveries.

### Verifying load-bearing claims

API signatures, constraints, and defaults assumed from documentation are frequently wrong at implementation time. Clone
the dependency repo and search its source to verify method signatures, parameter constraints, default values, error
conditions, and version availability (5-10 min per dependency).

Every load-bearing claim in an RDR — API behavior, peer-RDR contract, byte layout, scoping decision — is recorded as a
**Critical Assumption Evidence Record** with one of the eight Methods (Source Search | Spike | Prior Art | Derivation
| Design Decision | Peer RDR | MVV Test | Docs Only) and a concrete Evidence pointer. The full vocabulary and the
self-reference rule live in [TEMPLATE.md](TEMPLATE.md) under *Critical Assumptions*. Two rules of thumb:

- **`Docs Only` is insufficient for anything load-bearing.** It is allowed only when paired with a Spike or Source
  Search plan in the Evidence line.
- **`Source Search` whose Evidence cites this same RDR — or any path under the RDR's artifact directory — is
  self-reference, not verification.** The X4 triage on RDRs 0001–0010 found this failure mode in 24 of 54 assumptions;
  the gate now blocks it explicitly. The proof must also support **the specific claim**, not an adjacent one
  ("verified something adjacent and called it the thing" is the same failure in a subtler form).
- **Anchor by symbol, not by line.** Cite source as a greppable `path::Symbol`, never a bare `file:line`; a commit-SHA
  permalink only for audit/traceability. The only mechanical anchor check worth running is *does the cited symbol still
  resolve on `main`?* Write fewer, durable anchors rather than many volatile ones. (Rationale: flow `README` *Doctrine*.)
- **Exactness words are claims, not emphasis.** If the RDR says all/every, first/nearest, byte-identical, lossless,
  canonical, deterministic, or stable order, back it with an Evidence Record or with the Minimum Viable Validation.
- **Examples are contracts only when labeled Normative.** Fixtures, sample inputs/outputs, numeric counts, platform
  paths, and worked examples are either Normative (tests may assert them; cite the artifact or derivation) or
  Illustrative (intent only; tests must not assert them literally).

**On code examples**: code in RDRs is either *Normative* (load-bearing — signatures, type definitions, wire formats,
error envelopes, I/O contracts that the implementer must match exactly) or *Illustrative* (pseudocode, examples,
sample invocations — shape only). The TEMPLATE's *Technical Design* section separates the two. Every external API call
inside a Normative block must have a corresponding Evidence Record above. Illustrative code is not annotated; the
surrounding prose makes assumptions explicit. Do not include full class implementations, config/schema definitions, or
code for deferred features.

**On capability dependencies**: do not specify a future subsystem as if it exists today. For each load-bearing behavior,
the RDR states whether the enabling capability exists now, is introduced by this RDR, is provided by a predecessor, or
is deferred to a future RDR. Use `Overrides` metadata when a new RDR intentionally replaces or narrows a prior contract.

## Finalization Gate

The Finalization Gate (in the template) is the mechanism that ensures no contradictions, inconsistencies, or
redundancies survive into the locked RDR.

**Why written responses, not checkboxes**: Checkboxes are easily rubber-stamped. Each gate item requires a written
statement that becomes part of the permanent RDR record. This forces the author to actively verify rather than passively
confirm.

**The gate covers five concerns derived from recurring RDR authoring patterns:**

1. **Contradiction Check** — Do research findings conflict with the proposed solution? Do planned features contradict
   stated design principles?
2. **Assumption Verification** — Is every Critical Assumption Evidence Record internally consistent (Status, Method,
   Evidence agree; "If wrong" non-empty)? Are any load-bearing records on `Docs Only` without a Spike or Source Search
   plan? Any whose `Source Search` evidence self-references this same RDR?
3. **Scope Verification** — Is the Minimum Viable Validation in scope? Is the downstream use case that proves the
   approach included, not deferred?
4. **Cross-Cutting Concerns** — For every concern that applies, has this RDR addressed it directly or pointed at the
   peer RDR that owns the project-wide policy? Concerns to consider: versioning, licensing, build/deployment, secrets,
   memory, concurrency, character encoding, canonical-form/determinism. Omit (don't N/A-bullet) what doesn't apply.
5. **Proportionality** — Is the document right-sized? Are alternatives, code examples, and future considerations trimmed
   to what adds value? The split test is **contract count, not word count**: an RDR that is the sole author of more than
   one independent load-bearing contract (a type design *and* a hash *and* a taxonomy…) spans more than one seam and
   should be split along those seams, not locked together. The Normative Contracts section is the seam detector.

**When to run the gate**: After the Proposed Solution and Alternatives are complete, before marking Draft → Final.

**Who runs it**: The author, a reviewer, or an AI assistant reading the complete RDR. For AI-assisted workflows, the
gate can be run as a verification prompt against the finished document.

## Pre-Lock Review

The TEMPLATE frontloads the questions a structured slot can capture. Some classes of issue cannot be absorbed by a
template — heterogeneous PM/UX gaps, time-shifted failures invisible at lock time, cross-RDR contract leaks. These need
a review *activity*, not a *slot*. Run the appropriate rounds between Decide and Lock, then re-run the Finalization
Gate on the fixed draft.

The three analytical rounds, in order of cost. Each links to its prompt file under [`prompts/`](prompts/):

1. **[3amigo](prompts/pre-lock/1-3amigo.md)** (~30 min) — Read the draft three times as PM / Implementer / QA personas;
   consolidate items flagged by ≥2 personas. Uniquely catches heterogeneous PM/UX gaps, first-hour implementer questions,
   "I cannot write a pass/fail test for this" QA gaps.

2. **[Critique](prompts/pre-lock/2-critique.md)** (~20–30 min, dual-model) — Adversarial future projection: three ways this
   fails in 6 weeks; the section rewritten first; the assumption that won't survive first user contact. Uniquely catches
   frozen-at-lock invariants without version markers and destructive operations whose policy was implicit.

3. **Repeatability** ([prompts/pre-lock/3-repeatability.md](prompts/pre-lock/3-repeatability.md), codegen × 3 + diff)
   and **CoVe** ([prompts/pre-lock/4-cove.md](prompts/pre-lock/4-cove.md)) — the single-RDR implementation-prompt
   lenses: under-specified signatures the implementer can't grip on, and internal silences/contradictions. CoVe leads
   with a [grounding sweep](prompts/pre-lock/0-grounding.md) (Step 0) that verifies the RDR's codebase claims against
   source — also run standalone at Mid/Large where a contract is locked. Cross-RDR
   contradiction is a *separate* concern — [pairwise.md](prompts/gate/pairwise.md) — that needs two settled
   (Final) RDRs, so the [`stages/`](stages/README.md#cross-rdr-drift-stage-71) recipe runs it post-Final at Stage 7.1
   (per cluster), not pre-lock, since a per-RDR pass can't catch drift a later lock introduces into an earlier peer.

The **[Tooling pass](prompts/gate/tooling-pass.md)** (~5 min, seconds when scripted) is *not* a fourth round — it is
the mechanical pre-step of the Finalization Gate, run on every RDR after the rounds (and after spike/assumption
reconcile) as a post-mutation regression sweep: TEMPLATE section coverage, Method-label vocabulary, `Source Search`
self-reference, `Docs Only` on load-bearing claims. The rounds rewrite the draft; this sweep confirms none of them
hollowed a section or disturbed an evidence record before the Gate's written responses. It is also the conformance
backstop for RDRs whose earlier passes were skimped. (Placed last per `_spec_validation/RDR-PROCESS-IMPROVEMENT.md`
§D.2; the assumption-evidence checks it runs are the mechanical share of what Resolve and the Repeatability/CoVe lenses
verify analytically.)

### Applicability matrix

Not every RDR needs every analytical round. Match the rounds to the RDR's risk surface. The Tooling-pass sweep and the
Finalization Gate run on every profile, so they are folded into "Gate" below:

| RDR profile | Rounds to apply |
| --- | --- |
| Small / single-file / non-user-facing | Gate (sweep + written responses; no analytical rounds) |
| Mid / has user-facing surface OR locks a contract | Grounding + 3amigo + Gate |
| Large / locks an enum, hash, format, grammar, or destructive operation | Grounding + 3amigo + Critique + Gate |
| Foundational / cross-RDR producer / spans modules | CoVe + 3amigo + Critique + Repeatability + Gate (+ Stage 7.1 cross-RDR pairwise, post-Final, per cluster) |

Grounding (a cheap codebase-claim sweep) runs first wherever a contract is locked — standalone at Mid/Large, embedded
as CoVe's Step 0 at Foundational (so CoVe leads there); 3amigo runs on every non-trivial RDR; Critique on RDRs that lock
a surface; Repeatability on foundational work. The Tooling-pass sweep runs on every RDR as the Gate's mechanical
pre-step.

The profile is the RDR's `Profile` Metadata field, sized by **blast radius** (the max of the contract axis and the
accretion axis) rather than contract count alone: a `Seam Lineage` carrying ≥2 closed prior point-fixes floors the
profile at Foundational (a deterministic count, escapable only by a written accretion disposition in that field). Seed
estimates it, Resolve overwrites it from the verified count, the Gate re-validates it. See the
[flow README](stages/README.md#applicability--which-stages-a-given-rdr-needs) for the field's lifecycle.

After each round, the author makes a fix pass on the draft. The Finalization Gate is the *last* check, not a
substitute for these rounds — its written-response items are sharper when the rounds have already absorbed the gaps a
template slot can capture, and its mechanical pre-sweep catches anything the rounds disturbed.

## Post-Mortem Process

The final lifecycle phase, reached only after the RDR has been implemented and lived with — once code is the source of
truth and the plan has met reality. It is the deliberate counterpart to the up-front Pre-Lock Review: that gate looks
forward at a draft, this one looks back at an implementation in use.

After an RDR is implemented, reverted, or abandoned, create a post-mortem as a sibling file next to the original RDR
using the [post-mortem template](post-mortem/TEMPLATE.md). Name the file `NNNN-slug-postmortem.md` (e.g.
`cli/0003-project-config-and-init-postmortem.md`). Cross-RDR synthesis artifacts (e.g. `SYNTHESIS.md`) live under
`post-mortem/`.

The purpose is **not** to catalog gaps for their own sake, but to identify recurring patterns of plan-vs-reality drift
that improve the RDR authoring process itself.

**When to create a post-mortem:**

- After any implemented RDR — compare plan vs. reality
- After a reverted RDR — understand why the plan failed
- Periodically across multiple post-mortems — synthesize cross-cutting findings into a SYNTHESIS.md

**Key sections:**

- **Implementation vs. Plan** — what matched, what diverged, what was added, what was skipped
- **Drift Classification** — categorize divergences to enable pattern analysis across RDRs
- **Key Takeaways** — actionable, generalizable, evidence-based improvements to the RDR process

Implementation artifacts should classify deviations before post-mortem synthesis. This is the same taxonomy the
implementation prompt ([`prompts/implementation/launch.md`](prompts/implementation/launch.md), Phase 2) records into
`deviations.md` during implementation — keep the two definitions in sync:

- **SPEC-DEFECT** — the locked RDR was impossible, internally wrong, or contradicted verified facts.
- **SPEC-UNDER** — the RDR left a load-bearing choice open.
- **DEPENDENCY-LIMIT** — an existing or predecessor capability could not satisfy the contract as written.
- **TEST-FIXTURE** — an example, fixture, generated test, count, or platform path was wrong.
- **IMPL-DECISION** — valid implementation latitude; record only when it affects future interpretation.
