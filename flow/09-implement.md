# Stage 9 — Implement

**Goal**: turn the locked RDR into working, spec-verified code. This is the
flow's true terminus — the RDR exists to be implemented, and implementation
is the last filter: tests built from the spec prove the design holds, or
surface the contract defect that sends it back.

This stage **dispatches into the orchestrator prompt in
[`../prompts/implementation/launch.md`](../prompts/implementation/launch.md)**
— it doesn't redefine it. So the paste is the launch body (see *Paste this*
below); this file adds only the *driving discipline*: the entry contract it
consumes, the gate it terminates on, and how it resumes across long gaps. The
launch prompt is standalone — pasted with `{RDR_PATH}` and `{RDR_RESOURCES}`
(default `_rdr/rdr-resources.md`, so a defect can be reasoned against the same
evidence base the flow used) filled, it needs nothing else loaded;
`{ARTIFACT_DIR}` (from `_rdr/rdr-env.md`) is where it writes.

**Run when**: the RDR is **Final** — Stage 8 left it locked. Implementation
usually starts hours or days after lock, often in a fresh session; this stage
is re-entrant by design (see *resuming* below). **Produces**: working code +
tests in the project source tree, and the artifacts under `{ARTIFACT_DIR}`
(`req-list.md`, `coverage.md`, `verification.md`, `deviations.md`,
`status.md`), ending at exactly `COMPLETE` or `INCOMPLETE`.

## Entry contract (what Stage 8 must have left true)

launch.md's PRECHECKS assume a locked RDR. Stage 8 guarantees exactly what
they consume — state it once, here:

- **Status: Final**, with the Finalization Gate's five written responses in the
  RDR. launch.md treats the spec as an immutable contract.
- **MVV in scope** (Stage 8 Scope Verification) — Phase 1 turns it into
  `REQ-MVV`, the runnable end-to-end test.
- **Predecessors COMPLETE** — every `**Predecessors**:` entry's
  `{ARTIFACT_DIR}/status.md` reads `COMPLETE`. This is the same gate the
  TEMPLATE's Predecessors field names; launch.md's PRECHECKS enforce it and
  halt otherwise.

If any of these is not true, the RDR is not ready to implement — return to
Stage 8 (lock) or Stage 7 (an unreconciled assumption), not into this stage.

## Paste this

Fill `{RDR_PATH}` (the locked RDR), then paste the launch body where shown:

```text
Implement the locked RDR at {RDR_PATH} as its ORCHESTRATOR.
<paste the body of ../prompts/implementation/launch.md, {RDR_PATH} and {RDR_RESOURCES} filled in>
```

launch.md is the doctrine's "delegate heavy reads to a sub-agent" taken to its
strongest form: the orchestrator never reads the RDR, source, or test files —
phase sub-agents do, returning ≤200–300-word summaries + artifact paths. Do
not loosen those orchestrator rules to match a looser stage; they are stricter
on purpose, and that is what keeps the implementation context small enough to
resume.

## Review gate (terminate or halt — this stage does not "advance")

Implementation has no hand-off to a next stage. It terminates at `COMPLETE`
or halts at `INCOMPLETE`; the gate is launch.md's COMPLETION GATE, computed
from artifact headers, not a re-read.

- **`status.md` == COMPLETE?** Then every `REQ-N` has ≥1 green test,
  `coverage.md` has no orphans either direction, `REQ-MVV` output is recorded,
  and `deviations.md` has no open needs-author-decision entries. This is the
  terminus — proceed to Close (workflow step 7): write the post-mortem.
- **`status.md` == INCOMPLETE?** It names the blocker. A precondition failure
  (predecessor not COMPLETE, artifacts inconsistent, Phase 1 red-before-green
  gate failed) is a halt, not a question — fix the named condition and
  re-enter the stage; the resume logic picks up at the next phase.
- **A contract-level deviation** (`SPEC-DEFECT`, or a `SPEC-UNDER` no reading
  resolves) means the locked RDR is wrong. **Do not edit the RDR.** Abandon
  implementation and iterate the RDR — re-enter the flow at Stage 2/3/4 as the
  defect dictates, re-lock (Stage 8), then re-run this stage. This is the
  flow's only backward edge out of Final, and it is deliberate: the spec is the
  source of truth, so a spec defect is fixed in the spec, never in the code.
  (Deviation Types — SPEC-DEFECT / SPEC-UNDER / DEPENDENCY-LIMIT /
  TEST-FIXTURE / IMPL-DECISION — are launch.md's Phase 2 taxonomy; it is the
  same taxonomy the post-mortem uses, in [`../README.md`](../README.md#post-mortem-process).)

## Resuming across long gaps

The flow's earlier stages advance linearly in one sitting; implementation does
not. launch.md is re-entrant: its PRECHECKS read `{ARTIFACT_DIR}/status.md`,
and if it names a phase, resume at the next one. A reader returning days later
pastes the same prompt against the same RDR — the on-disk artifacts, not the
session, carry state. This is why the artifacts are the authoritative record
and why `COMPLETE`/`INCOMPLETE` must be computable from disk. The linear
per-stage model holds up to Stage 8; Stage 9 trades it for a disk-backed,
resumable gate.

## Terminus

`status.md` is `COMPLETE`, every `REQ-N` is covered by a green test, and
`REQ-MVV` ran end-to-end with its output recorded. The flow is done.

→ Close (workflow step 7 in [`../README.md`](../README.md#workflow)): set the
RDR's status (Implemented | Reverted | Abandoned | Superseded) and write the
post-mortem ([`../prompts/post-mortem/`](../prompts/post-mortem/)).
