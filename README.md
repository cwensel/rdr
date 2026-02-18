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

| ID  | Title | Status | Priority |
|-----|-------|--------|----------|

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

## Research Guidance

Consult during planning:

- Requirements and standards documentation (cite
  specific sections)
- Dependency source code (not just public docs —
  documentation-only research frequently produces
  wrong assumptions)
- Existing codebase patterns or idioms
- Run spikes/POCs for any load-bearing assumption

Include in RDR: Citations, code snippets demonstrating
capabilities/limitations, hard-to-earn discoveries.

**On code examples**: Include illustrative snippets for
novel patterns. Mark framework API calls as "verify
during implementation" — method signatures from
documentation reading are frequently wrong in detail.
Do not include full implementations; specify behavior
and architecture instead.

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
post-mortem analysis of implemented RDRs:**

1. **Contradiction Check** — Do research findings
   conflict with the proposed solution? Do planned
   features contradict stated design principles?
   *(Observed in RDR-004, 011)*
2. **Assumption Verification** — Are all load-bearing
   assumptions verified by spike/POC, not just
   documentation review?
   *(Observed in RDR-003, 005, 006, 009, 010)*
3. **Scope Verification** — Is the minimum viable
   validation in scope? Is the downstream use case
   that proves the approach included, not deferred?
   *(Observed in RDR-003, 006)*
4. **Cross-Cutting Concerns** — Are versioning,
   licensing, build tool compatibility, deployment
   model, IDE compatibility, and incremental adoption
   addressed where applicable?
   *(Observed in RDR-004, 007, 010, 011)*
5. **Proportionality** — Is the document right-sized?
   Are alternatives, code examples, and future
   considerations trimmed to what adds value?
   *(Observed in RDR-005, 006, 008, 010)*

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
