# Stage 1 — Seed

**Goal**: turn an idea (a kata, a matrix gap, a sentence) into a
template-conformant Draft so every later stage has structured slots to fill.

## Paste this

Fill `{IDEA}` (a kata id or a one-line description); resolve `{RDR_ENV}` via the
workspace marker (see README *Where the seam lives* — `. "$WS/.rdr-workspace"`
exports `$RDR_ENV`). The RDR process README + TEMPLATE are the flow's own,
beside this file.

**Prompt**: [`../prompts/flow/01-seed.prompt.md`](../prompts/flow/01-seed.prompt.md)
— the `/rdr-seed` skill runs it (binds the params, allocates the number). Paste
its body by hand if driving without the skill.

**Run when**: starting a brand-new RDR. **Produces**: a new RDR file in the
project's RDR directory (the `{ARTIFACT_DIR}` parent from `{RDR_ENV}`), Status: Draft.

## Review gate

- **Number was claimed atomically** (§rdr-claim: reserved as `NNNN-RESERVED.md`
  *before* authoring, then `git mv`'d to the slug) — not a bare `max+1` that races
  a concurrent session. No stray `*-RESERVED.md` left behind; slug is descriptive.
- **Problem Statement leads with a user outcome**, not a mechanism. "The
  system will parse…" is a mechanism — send it back.
- **No premature solution.** Proposed Solution, Critical Assumptions, and
  Research Findings are placeholders. Seeding that already picks an approach
  skips Propose's option analysis — reject it.
- **Skeleton is TEMPLATE.md, filled in place** — not authored from scratch or
  copied from a neighbor RDR. The unfilled sections should read *verbatim* as the
  template ships them (`diff` against `TEMPLATE.md` shows changes only in the H1,
  Metadata, Problem Statement, Context). Neighbor-copied structure drifts the
  template and smuggles in the neighbor's solution — send it back.
- **Metadata is honest**: Status Draft; Type/Priority set; Predecessors listed
  if the kata names dependencies; **Seam Lineage** copied from
  `kata-scope-review` (count + trail, or "no prior accretion"), and the Profile
  floored to `foundational` if that count is ≥2.
- **Is it RDR-shaped at all?** If the {IDEA} carries no real design fork —
  there is one obvious implementation and nothing to weigh — it is not an RDR.
  Refile it as a plain issue and set `Status: Demoted [→ <issue link>]` (record
  the same link under Related Issues). A Demoted RDR runs no further stages.

## Two genesis pathways (both are first-class)

A seed arrives by one of two routes, and **neither is a process miss**:

- **Designed-ahead** — a well-bounded concept named top-down before any code.
  Foundational, separable concepts admit this.
- **Discovered-design** — a seed *forced out by building*: an implementation
  collision, a spike that refutes the design, or a Stage 8 spec-defect re-open
  surfaces a contract the up-front pass could not have anticipated. Deeply
  entangled concepts are *only* discoverable this way — expecting to design them
  whole ahead of time is the wrong frame, so a collision-born seed is the
  **expected** route for them, not a slowdown.

The economy is in converting *uncontrolled* discovered-design (N collisions each
absorbed as a fresh point-fix) into *controlled* discovered-design: **once a 2nd
collision lands on the same concept area, author the contract RDR proactively**
rather than absorbing a 3rd. (This is the RDR-level twin of the kata
seam-accretion tripwire in `kata-scope-review`.)

## Advance when

The Draft exists, conforms to the template skeleton, and its Problem Statement
is a reviewable user-outcome statement with no solution baked in.

→ Next: [02-propose.md](02-propose.md)
