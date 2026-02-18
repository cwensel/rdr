# Recommendation Decisioning Records (RDRs)

RDRs are specification prompts built through iterative
research and refinement.

Unlike [ADRs](https://adr.github.io) which document
decisions already made, RDRs evolve during planning as
the problem statement becomes more refined through
research and exploration and alternatives emerge with
one as the best option.

**Core Purpose**: Capture both the final solution and
supporting evidence to prevent purpose drift during
implementation.

## Index

| ID | Title | Status | Priority |
| --- | --- | --- | --- |

## Workflow

1. **Create** (Draft) — Document problem, initial
   constraints, technical environment
2. **Research** (Draft) — Investigate, add findings,
   refine problem statement, explore alternatives.
   Label all findings as Verified/Documented/Assumed
3. **Decide** (Draft) — Select approach, document
   rationale, complete Finalization Gate
4. **Lock** (Final) — All gate items answered; RDR is
   the specification prompt for implementation
5. **Implement** (Final) — Use locked RDR as spec;
   do not edit during implementation
6. **Close** (Implemented | Reverted | Abandoned) —
   Update status; create post-mortem

**If implementation reveals RDR is wrong**: Abandon
implementation, iterate on RDR with lessons learned,
start fresh.

## Status Definitions

- **Draft** — During planning/research phase
- **Final** — Locked, ready for or during implementation
- **Implemented** — Implementation complete
- **Reverted** — Implemented then undone (document why)
- **Abandoned** — RDR not implemented
- **Superseded** — Replaced by another RDR

## When to Create an RDR

- Complex multi-step feature implementation
- Significant technical debt or architectural decisions
  with trade-offs
- Framework/library workarounds requiring substantial
  research
- Any work benefiting from documented alternatives
  before committing to writing code

When scoping work, account for the full surface — not
just the happy path. Include Day 2 operations (list,
info, delete, verify, backup for each resource
created), error handling, and cross-cutting integration
points.

## Research Guidance

Consult during planning:

- Requirements and standards documentation (cite
  specific sections)
- Dependency source code — see
  [Dependency Source Verification](#dependency-source-verification)
  below
- Existing codebase modules that overlap with the
  proposed design (audit for reuse before designing
  new components)
- Existing codebase patterns or idioms
- Run spikes/POCs for opaque services where source
  code is unavailable

Include in RDR: Citations, code snippets demonstrating
capabilities/limitations, hard-to-earn discoveries.

### Dependency Source Verification

API signatures, constraints, and defaults assumed from
documentation are frequently wrong at implementation
time. Clone the dependency repo and search its source
to verify method signatures, parameter constraints,
default values, error conditions, and version
availability (5-10 min per dependency).

Mark every external API reference in an RDR:

- **Source Search** — verified against dependency
  source code. Standard method for libraries.
- **Spike** — verified by running code against a
  live service. For opaque services where source
  is unavailable.
- **Docs Only** — documentation reading alone.
  Insufficient for load-bearing assumptions.

**On code examples**: All code in RDRs is illustrative
by default. Mark framework API calls as Verified
(source search) or Assumed (needs validation). Do not
include full class implementations, config/schema
definitions, or code for deferred features. Specify
behavior and interfaces instead — code for the current
scope should demonstrate architectural patterns, not
production-ready implementations.

## Finalization Decisioning

The Finalization Gate (in the template) is the
mechanism that ensures no contradictions,
inconsistencies, or redundancies survive into the
locked RDR.

**Why written responses, not checkboxes**: Checkboxes
are easily rubber-stamped. Each gate item requires a
written statement that becomes part of the permanent
RDR record. This forces the author to actively verify
rather than passively confirm.

**The gate covers five concerns derived from
recurring RDR authoring patterns:**

1. **Contradiction Check** — Do research findings
   conflict with the proposed solution? Do planned
   features contradict stated design principles?
2. **Assumption Verification** — Are all load-bearing
   assumptions verified by source search, not just
   documentation review?
3. **Scope Verification** — Is the minimum viable
   validation in scope? Is the downstream use case
   that proves the approach included, not deferred?
4. **Cross-Cutting Concerns** — Are versioning,
   licensing, build tool compatibility, deployment
   model, IDE compatibility, and incremental adoption
   addressed where applicable?
5. **Proportionality** — Is the document right-sized?
   Are alternatives, code examples, and future
   considerations trimmed to what adds value?

**When to run the gate**: After the Proposed Solution
and Alternatives are complete, before marking
Draft → Final.

**Who runs it**: The author, a reviewer, or an AI
assistant reading the complete RDR. For AI-assisted
workflows, the gate can be run as a verification prompt
against the finished document.

## Post-Mortem Process

After an RDR is implemented, reverted, or abandoned,
create a post-mortem in `post-mortem/` using the
[post-mortem template](post-mortem/TEMPLATE.md). Name
the file to match the original RDR
(e.g. `post-mortem/003-feature-name.md`).

The purpose is **not** to catalog gaps for their own
sake, but to identify recurring patterns of
plan-vs-reality drift that improve the RDR authoring
process itself.

**When to create a post-mortem:**

- After any implemented RDR — compare plan vs. reality
- After a reverted RDR — understand why the plan failed
- Periodically across multiple post-mortems — synthesize
  cross-cutting findings into a SYNTHESIS.md

**Key sections:**

- **Implementation vs. Plan** — what matched, what
  diverged, what was added, what was skipped
- **Drift Classification** — categorize divergences to
  enable pattern analysis across RDRs
- **Key Takeaways** — actionable, generalizable,
  evidence-based improvements to the RDR process
