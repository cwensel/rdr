# Implementation Launch Prompt for a Locked RDR

**Document:** launch.md

**Date:** 2026-04-27

Paste the prompt below above (or alongside) a locked RDR to drive its
implementation. Designed so tests verify the spec — not the LLM's own
code — and so each launch leaves file-backed artifacts the next launch
(or the next RDR) can rely on without re-deriving anything.

## Assumptions

- RDRs are run one at a time, manually, with each finishing `COMPLETE`
  before the next starts. Cross-RDR orchestration is out of scope.
- Each RDR is ≤2k lines × 120 chars (~30k tokens worst case). The full
  spec fits in any modern context; the prompt does not paginate it.
- RDRs follow the template at `../../TEMPLATE.md` (the rdr/ directory
  containing this prompt). Required sections, with their TEMPLATE
  paths:
  - `## Metadata` (top-level)
  - `## Research Findings > ### Critical Assumptions`
  - `## Proposed Solution > ### Technical Design > #### Normative Contracts`
  - `## Trade-offs > ### Failure Modes`
  - `## Validation > ### Testing Strategy`
- Tests and implementation code live in the project's normal source
  tree. The artifact directory holds traceability *about* the work, not
  the code itself. Post-mortems happen later, per the RDR workflow.

## Artifact Layout

For RDR `<rdr-dir>/NNNN-slug.md`, the prompt writes a sibling directory:

```text
<rdr-dir>/
├── NNNN-slug.md                 # locked RDR (input)
├── NNNN-slug/                   # implementation artifacts (output)
│   ├── req-list.md              # REQ-N quotes + ASSUMPTIONs
│   ├── coverage.md              # REQ-N × test-name + REQ-MVV output
│   ├── verification.md          # Phase 3 CoVe + adversarial findings
│   ├── deviations.md            # classified deviations
│   └── status.md                # COMPLETE | INCOMPLETE — current phase
└── NNNN-slug-postmortem.md      # post-mortem (added after close)
```

This `NNNN-slug/` directory is the RDR flow's `{ARTIFACT_DIR}` (defined under
*Output staging* in the flow's resource file, `_rdr/rdr-resources.md` at the
project root). The prompt derives the path itself — `<art>` = the directory
next to the RDR named after its basename — so it stays standalone-pasteable;
the flow simply gives that same location a name. These implementation
artifacts stay tracked beside the RDR; the flow's pre-lock evidence
(`{SPIKE_DIR}`/`{FLOW_DIR}`) is separate, non-permanent scratch under `_rdr/`.

## The Prompt

Replace `{RDR_PATH}` with the RDR file's path
(e.g. `<project>/0001-first-rdr.md`) and `{RDR_RESOURCES}` with the flow's
evidence index (default `_rdr/rdr-resources.md`) — the same corpora, design
docs, and anchors the flow grounded the spec against, so a sub-agent facing a
defect can verify and reason rather than ask:

```text
You are the ORCHESTRATOR for implementing the locked RDR at {RDR_PATH}.
The spec is a contract: correctness means "matches the spec," not
"passes tests you wrote."

Your role is to spawn phase sub-agents and read short summaries from
them — not to read the RDR, the implementation, or test files
yourself. Heavy reading and writing live in sub-agents whose context
is discarded on return; you hold only artifact paths and one-line
phase summaries. The on-disk artifacts (`<art>/*.md`) are the
authoritative record; if you and the artifacts disagree, the
artifacts are right.

`<art>` = directory next to the RDR named after its basename without `.md`.
Create if missing. The sub-agents write/update inside it: req-list.md,
coverage.md, verification.md, deviations.md, status.md.

ESCALATION RULE
Only stop to ask the user when a DESIGN DECISION is required:
genuine spec ambiguity that no reasonable reading resolves, or a
contract-level deviation (spec defect, deferred-scope decision,
accept-or-fix call). Never ask for permission to proceed between
phases, never ask whether to run the next phase, never ask about
mechanical translations the sub-agent can record and continue past.
If a precondition fails (predecessor not COMPLETE, on-disk artifacts
inconsistent, red-before-green gate fails), write
`<art>/status.md` as INCOMPLETE with the named blocker and halt —
that is a halt, not a question.

PRECHECKS (orchestrator runs these directly — cheap reads only)
- Resume: if `<art>/status.md` exists and names a phase, resume at
  the next phase. If artifacts are inconsistent (e.g., status says
  Phase 2 but no tests exist), write INCOMPLETE("artifact
  inconsistency: <detail>") and halt.
- Predecessors: read the RDR's `**Predecessors**:` field (one grep is
  fine). For each listed `MMMM-slug`: verify
  `<rdr-dir>/MMMM-slug/status.md` reads `COMPLETE` (else write
  INCOMPLETE("predecessor MMMM-slug not COMPLETE") and halt). Record
  the predecessor artifact paths to pass to Phase 0 and Phase 1.
  Skip if no Predecessors field.
- Test framework: infer from (in order) the project's existing test
  config (`go.mod`, `package.json`, `pyproject.toml`, `build.gradle`,
  `Cargo.toml`, …) and existing test files in the source tree. Pick
  the dominant one and record it. Only escalate as a user question
  if there is no detectable framework at all.

PHASE 0 — Spec audit [DELEGATE to sub-agent: "Phase 0 auditor"]
Brief the sub-agent with:
  - The RDR path {RDR_PATH} (the sub-agent reads it itself).
  - The `<art>` path. Create the directory if missing.
  - Predecessor `req-list.md` + `deviations.md` paths from prechecks.
Sub-agent's task:
  1. Write `<art>/req-list.md`: every testable clause as
     `[REQ-N] "<exact quote>" — (section)`. Include the Minimum
     Viable Validation as REQ-MVV.
  2. Append `ASSUMPTION:` lines for implicit choices it made when
     wording was imprecise but a single reading is defensible.
  3. If a clause is genuinely ambiguous (two readings would produce
     materially different behaviour, no precedent in predecessors),
     record it under a `QUESTIONS` section in `req-list.md` and
     surface it in the return summary.
Sub-agent returns ≤200 words: REQ count, REQ-MVV identifier, any
QUESTIONS list. If QUESTIONS is non-empty, the orchestrator asks the
user one consolidated question, records the answers as additional
ASSUMPTION lines in `req-list.md`, and re-briefs the auditor if the
answers change REQ wording; otherwise advance.

PHASE 1 — Tests first [DELEGATE to sub-agent: "Phase 1 test author"]
Brief the sub-agent with:
  - The RDR path, `<art>/req-list.md`, predecessor artifacts, the
    test framework, and the test directory.
  - NO implementation hints, sketches, or design notes.
Sub-agent's task: for each REQ-N, write tests that would fail if a
future change broke that clause. Each test:
  - Opens with `// REQ-N: "<quote>"` (or the language's comment syntax).
  - Labels one of: HAPPY PATH | INPUT EDGE | BOUNDARY | ADVERSARIAL | DOMAIN EDGE.
  - Asserts on spec-described behaviour only — no mocks of the unit
    under test, no private fields, no log strings, no call counts
    unless the spec names them.
  - REQ-MVV is a runnable end-to-end test.
After writing the tests, the sub-agent runs the suite and confirms
ALL new tests FAIL (red). Any test green against a missing/stub
implementation is tautological — the sub-agent rewrites it before
returning. The sub-agent writes `<art>/coverage.md` (REQ-N × test
name, orphans flagged both ways).
Sub-agent returns ≤200 words: test file paths, REQ-N coverage count,
red-confirmed yes/no, REQ-MVV runner command. If red-confirmed is
no, write INCOMPLETE("Phase 1 red gate failed") and halt — do not
advance to Phase 2.

PHASE 2 — Implementation [DELEGATE to sub-agent: "Phase 2 implementer"]
Brief the sub-agent with:
  - `<art>/req-list.md`, `<art>/coverage.md`, the test file paths
    from Phase 1, the source tree root, and the test framework.
  - The RDR path (sub-agent reads as needed for context, not as
    primary input).
  - `{RDR_RESOURCES}` — the evidence index to ground any apparent
    defect against (Source Search the corpora, check the design docs)
    before escalating.
  - Authority to write/update `<art>/deviations.md` for any
    classified deviation it encounters.
Sub-agent's task: write the minimum code to turn the tests green. No
features, validation, error handling, or abstractions no REQ-N
demands. Owns its own write/test/iterate loop until either all tests
are green OR a deviation blocks progress. When a gap appears that the
sub-agent can resolve mechanically (rename, path difference, type
shape mismatch, append-only enum extension), it records a classified
entry in `deviations.md` with "Status: mechanical translation" and
continues. When a gap looks contract-level (spec wording wrong,
behaviour under-specified in a load-bearing way, scope deferral
needed), it first grounds the gap against `{RDR_RESOURCES}` — Source
Search the corpora for the owning behaviour (the standard/spec corpus
for documented behaviour, the dependency/peer-source corpus for
implementation behaviour, per `{RDR_RESOURCES}`), check the design docs
and anchors — and ultrathinks a resolution the spec's own evidence
base supports, recording that evidence on the entry. Only a genuine
design decision the evidence cannot resolve gets "Status: needs author
decision" and returns without forcing green; a gap the evidence
resolves is recorded with its Type, the evidence, and "Status:
mechanical translation" (or the derived choice), then continue. Every entry uses exactly
one Type:
  - SPEC-DEFECT — the locked RDR is impossible, internally wrong, or
    contradicts verified facts.
  - SPEC-UNDER — the RDR left a load-bearing choice open.
  - DEPENDENCY-LIMIT — an existing or predecessor capability cannot
    satisfy the contract as written.
  - TEST-FIXTURE — an example, fixture, generated test, count, or
    platform path is wrong.
  - IMPL-DECISION — valid implementation latitude; record only when
    it affects future interpretation.
Do not record ordinary implementation choices unless they affect
contract, validation, or future interpretation. After the suite is
green, it runs REQ-MVV end-to-end and records the actual output in
`<art>/coverage.md`.
Sub-agent returns ≤300 words: green yes/no, count of mechanical
deviations (no action needed), list of needs-author-decision
deviations (with one-line each), REQ-MVV pass/fail.
If needs-author-decision deviations are non-empty, the orchestrator
asks the user one consolidated question listing each gap with the
sub-agent's recommendation, records the resolutions back to
`deviations.md`, and re-briefs the implementer with the decisions.
Otherwise advance.

PHASE 3 — Self-verification (two sub-agents in parallel, then fixup if needed)

PHASE 3a [DELEGATE to sub-agent: "CoVe verifier"]
Brief: RDR path and `<art>/req-list.md` only. NO implementation,
NO tests. Sub-agent's task: for each REQ-N, name an input that
would make a correct implementation visibly violate it. Then run
those inputs against the actual implementation (the sub-agent has
the source tree but is forbidden from reading the Phase 1 tests).
Any actual violation gets appended to `<art>/verification.md` as a
FAIL-N entry with the failing input and observed behaviour.
Sub-agent returns ≤200 words: defect count, FAIL-N list (one line each).

PHASE 3b [DELEGATE to sub-agent: "Adversarial reviewer"]
Brief: the RDR's Failure Modes section, `<art>/req-list.md`, and
the implementation source tree. NO Phase 1 tests, NO Phase 3a
findings. Sub-agent's task: as a senior reviewer who thinks this is
wrong, name the three most likely failure modes (anchored in the
RDR's Failure Modes section) and the test that catches each. Add
any missing tests to the test directory; confirm they fail against
the current implementation (else they don't actually catch the
failure mode). Append findings to `<art>/verification.md` as ADV-N
entries.
Sub-agent returns ≤200 words: failure modes named, tests added,
which (if any) currently fail against the implementation.

PHASE 3c — FIXUP [conditional, DELEGATE to sub-agent: "Phase 3 fixup"]
Run this only if Phase 3a returned FAIL-N entries OR Phase 3b added
tests that currently fail. Brief: `<art>/verification.md`,
`<art>/req-list.md`, source tree, test framework, `{RDR_RESOURCES}`.
Sub-agent's task:
fix each defect with the minimum change; add a regression test if
not already present. After fixing, run the full suite — must be
green. New deviations follow Phase 2's classification rules
(mechanical vs needs-author-decision).
Sub-agent returns ≤200 words: defects fixed, regression tests
added, green yes/no, any new deviations needing author decision.
Apply the same escalation rule as Phase 2 if needed.

COMPLETION GATE (orchestrator runs directly)
Read the headers of `<art>/coverage.md`, `<art>/verification.md`,
and `<art>/deviations.md` — do not re-read full files. Write
`<art>/status.md` as exactly one of:

  COMPLETE — every REQ-N has ≥1 green test, coverage.md has no orphans
  either direction, REQ-MVV output is recorded, and deviations.md has
  no open needs-author-decision entries (entries are closed,
  mechanical translation, or accepted by author).

  INCOMPLETE — <one-line named blocker, citing the failing condition>.

Do not declare success on INCOMPLETE.

GUARDRAILS
- Phase sub-agents own their own reading. The orchestrator never
  reads the RDR text, implementation files, or test files.
  Re-reading the same files in two contexts is the bug we are
  avoiding.
- If a test needs information not in the spec, the sub-agent first
  grounds the gap against `{RDR_RESOURCES}` (the spec's own evidence
  base) and ultrathinks whether that evidence resolves it; only if it
  cannot does it record a `SPEC-UNDER` "needs author decision" entry
  and return. Ground, don't invent — and don't escalate what the
  evidence already answers. If the spec is wrong, the sub-agent
  records a `SPEC-DEFECT` (citing the contradicting evidence); the
  orchestrator surfaces it as a user question, and
  only on the user's confirmation does it mark INCOMPLETE("spec
  defect") and halt without editing the spec.
- A reviewer reading only the test headers should see the spec
  re-stated.
- Sub-agent independence is enforced by brief, not honour:
  Phase 1 brief excludes implementation hints; Phase 2 brief
  excludes Phase 3 findings; Phase 3a brief excludes
  implementation reads of the Phase 1 tests; Phase 3b brief
  excludes Phase 3a findings.
```

## Predecessor Convention

When an RDR's tests or code will reference behaviour pinned by an
earlier RDR, list the load-bearing predecessors in the RDR's Metadata:

```markdown
- **Predecessors**: 0001-first-rdr, 0003-some-prior-rdr
```

The launch prompt's PRECHECKS step gates on each predecessor's
`status.md` being `COMPLETE` (hard stop otherwise) and surfaces their
`req-list.md` + `deviations.md` to the Phase 1 sub-agent. Predecessor
test files in the source tree are the executable ground truth — read
like any other code, not as a launch artifact. Predecessor
`verification.md` and `coverage.md` do not feed forward; they were
closed during that RDR's launch.

The integration layer is the spec author: when writing RDR N+1, cite
predecessor REQ-Ns where N+1 depends on them in the RDR text. The
prompt only needs to know which predecessors to gate on.

## Design Notes

- **REQ-N quote IDs commit the LLM to the spec text** before any drift
  can start. Every later phase cites them.
- **Red-before-green (Phase 1 gate) is the load-bearing rule.** A
  test green against a stub is tautological by definition; this gate
  is what distinguishes "tests verify the spec" from "tests verify
  what the LLM wrote."
- **Orchestrator-of-sub-agents architecture is what keeps the main
  context small.** The orchestrator holds only artifact paths and
  per-phase summaries (≤200–300 words each). Every phase that touches
  the RDR text, source tree, or test files runs in its own sub-agent
  whose context is discarded on return. This is the primary defence
  against the 1M-token blowup that happens when one agent accumulates
  every file it ever read across all phases. The on-disk artifacts
  (`<art>/*.md`) are the durable record between phases.
- **Sub-agent briefs enforce independence structurally.** Phase 1
  test author cannot see implementation; Phase 3a CoVe verifier
  cannot see Phase 1 tests; Phase 3b adversarial reviewer cannot see
  Phase 3a findings. Asking a single agent to "ignore what you just
  wrote" is aspirational — isolation is enforceable only via
  brief-scoping at spawn time.
- **Hard COMPLETE/INCOMPLETE gate** makes "done" computable from disk,
  which is what manual sequencing across long gaps requires. The
  orchestrator computes the gate from artifact headers only, not by
  re-reading content.
- **Escalation to the user is for design decisions, not continuation.**
  Spec ambiguity that no reading resolves, and contract-level
  deviations (spec defect, deferred-scope), surface as user questions.
  Phase transitions, framework inference, mechanical deviation
  resolution, and precondition failures (which become INCOMPLETE
  halts) never ask.
- **Sub-RDR fixes** (one-line bug fixes, mechanical extensions) can
  run Phase 1 inline without a sub-agent if the orchestrator judges
  the surface area trivial. The red-before-green gate still applies.
- **The Phase 2 deviation Types** (SPEC-DEFECT / SPEC-UNDER /
  DEPENDENCY-LIMIT / TEST-FIXTURE / IMPL-DECISION) are the same
  taxonomy the RDR process uses for post-mortem drift classification
  (`../../README.md`, *Post-Mortem Process*). They are defined inline
  in the prompt above so this file stays standalone-pasteable; keep the
  two definitions in sync. In the RDR flow ([`../../flow/`](../../flow/README.md)),
  this prompt is **Stage 9 (Implement)** — the flow's terminus.

## Citations

- TiCoder, TSE 2024 — tests-from-spec-first, ranking by test-consistency.
  <https://www.seas.upenn.edu/~asnaik/assets/papers/tse24_ticoder.pdf>
- Dhuliawala et al., *Chain-of-Verification*, ACL Findings 2024 —
  independence beats CoT/few-shot on hallucination.
  <https://arxiv.org/abs/2309.11495>
- Addy Osmani, *How to write a good spec for AI agents*, 2025 —
  conformance testing + per-test spec citation pattern.
  <https://addyosmani.com/blog/good-spec/>
