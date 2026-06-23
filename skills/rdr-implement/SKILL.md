---
name: rdr-implement
argument-hint: (<NNNN> [--commit | --no-commit]) | --launch-prompt-path
description: Use to implement a locked (Final) RDR into working, spec-verified code (e.g. "implement RDR 45", "/rdr-implement 0045"). Runs Stage 8 of the RDR flow — dispatches into prompts/implementation/launch.md as the orchestrator. Re-entrant across days from status.md. Terminates at COMPLETE/INCOMPLETE; a contract-level spec defect routes back to the RDR, never patches the code. Pairs with /rdr-finalize (must precede it). Also exposes `--launch-prompt-path` so wrappers get the orchestrator prompt's location from the engine instead of reconstructing its internal layout.
---

# rdr-implement — Stage 8 (Implement)

Turn the locked RDR into working, spec-verified code — the flow's true terminus.
Implementation is the last filter: tests built from the spec prove the design holds,
or surface the contract defect that sends it back. **Dispatches into the
orchestrator prompt `launch.md`** — it does not redefine
it. The orchestrator never reads the RDR, source, or test files; phase sub-agents
do. Do not loosen those orchestrator rules.

## Usage

```
/rdr-implement <NNNN>
/rdr-implement --launch-prompt-path    # print the orchestrator prompt path, then stop
```

## `--launch-prompt-path` — the prompt-location contract (query mode)

A consumer that orchestrates Stage 8 itself (e.g. `/rdr-implement-triage`) needs
`launch.md`'s location but must **not** hardcode the engine's internal layout.
This flag is that contract: the engine owns where its prompt lives and reports it.

Short-circuit — run **only** `rdr-common §seam-bind`, then print and stop. Take
**no** RDR argument; run **none** of the entry-contract gates below (no Final /
predecessor / resume checks — nothing is implemented):

```sh
# §seam-bind has bound $RDR_HOME (fast path or resolver — see rdr-common)
echo "$RDR_HOME/prompts/implementation/launch.md"
```

That canonical path is stable even though this skill references `launch.md` as a
sibling symlink. The prompt body still carries `{RDR_PATH}` / `{RDR_RESOURCES}`
placeholders — the caller fills them per run. If `$RDR_HOME` is unbound
(stale marker) → `stopped:no-rdr-flow-home` (`/rdr-init` to refresh).

## Entry contract (what Stage 7 must have left true)

Refuse with the named code if any is false (the RDR is not ready — return to
`/rdr-finalize` or `/rdr-reconcile`):

- **Status: Final** with the Finalization Gate's five responses in the RDR
  (`stopped:not-final`).
- **MVV in scope** — Phase 1 turns it into `REQ-MVV` (`stopped:mvv-deferred`).
- **Predecessors COMPLETE** — every `**Predecessors**:` entry's
  `{ARTIFACT_DIR}/status.md` reads `COMPLETE` (`stopped:predecessor-incomplete:<id>`).

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `RDR_PATH`, `{ARTIFACT_DIR}` (= `<RDR_RECORDS>/<RDR_SLUG>/artifacts/`).
2. **Self-detected resume (no flag).** launch.md's PRECHECKS read
   `{ARTIFACT_DIR}/status.md`; if it names a phase, resume at the next one. A reader
   returning days later runs the same skill against the same RDR — the on-disk
   artifacts, not the session, carry state. **If `status.md` already reads
   `COMPLETE`, do not re-run** — report it (the flow is done; point at Close /
   post-mortem) rather than re-implementing a finished RDR (`stopped:already-complete`).
3. **Run the orchestrator** [`launch.md`](launch.md)
   with `{RDR_PATH}` / `{RDR_RESOURCES}` filled; it writes the artifacts under
   `{ARTIFACT_DIR}` (`req-list.md`, `coverage.md`, `verification.md`,
   `deviations.md`, `status.md`).

## Gate — terminate or halt (this stage does not "advance")

- **`status.md` == COMPLETE** — every `REQ-N` has ≥1 green test, no coverage orphans,
  `REQ-MVV` output recorded, no open needs-author-decision deviations. → Close:
  set the RDR's terminal status and write the post-mortem
  (`post-mortem/`).
- **`status.md` == INCOMPLETE** — it names the blocker. A precondition failure is a
  halt: fix the named condition and re-run; resume picks up at the next phase.
- **A contract-level deviation** (`SPEC-DEFECT`, or an unresolvable `SPEC-UNDER`) —
  the locked RDR is wrong. **Do not edit the RDR or patch the code.** Abandon
  implementation and re-enter the flow at the stage the defect dictates
  (`/rdr-propose` / `/rdr-refine` / `/rdr-resolve`), re-lock via `/rdr-finalize`,
  then re-run this. The flow's only backward edge out of Final, and it is deliberate.

## Next step (rdr-common §next-step)

- §commit here is **artifact files only** (`<slug>/{req-list,coverage,verification,deviations,status}.md`),
  if autocommit is on. **Code** commits stay with the orchestrator's `feat(...)` contract — §commit never touches source.
- COMPLETE → Close (post-mortem); the flow is done.
- INCOMPLETE → fix the named blocker, re-run `/rdr-implement NNNN`.
- Spec defect → re-enter the flow at the named stage, then re-lock and re-run.
- `/rdr-status NNNN` to re-orient.
