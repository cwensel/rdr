# Phase 1 — test author (red before green)
Fields: RDR_PATH NNNN RDR_HOME ART WORKTREE BRANCH TEST_FRAMEWORK TEST_DIR PREDECESSOR_ARTIFACTS
(Fields arrive as `NAME=value` lines; `{NAME}` is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed` with no edits.

Your task is the PHASE 1 block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 1 —` to `PHASE 2 —`): read it once. Inputs: {ART}/req-list.md,
predecessor artifacts {PREDECESSOR_ARTIFACTS}, framework {TEST_FRAMEWORK},
tests under {TEST_DIR}. You receive NO implementation hints, sketches or
design notes; do not go looking for them. Write `{ART}/coverage.md` as the
block says (an uncovered REQ keeps its row with an EMPTY second cell).

Red must be committable. Where the checkout's pre-commit hook vets or
compiles the tree (`go vet ./...` is the common case), a suite naming a
symbol that does not exist yet cannot be committed. Reach the not-yet-written
surface through a probe that compiles today and fails at run time — a
reflective adapter over the candidate spellings, a lookup that reports the
REQ it is owed — so every red is an assertion failure, never a compile break.
A stub you must add is non-satisfying (a value outside the contract's
domain), never the happy value. Commit the red suite on {BRANCH}.

Read the record through `{RDR_HOME}/bin/rdr inspect …` (`--select
{NNNN}:<id>` for exact quotes), never `sed`/`grep` the file. Never edit
{RDR_PATH}. Scratch in /tmp, never {ART}. Separators `---`, never `===`.

Reads: `{RDR_HOME}/bin/rdr-leg-read -C {WORKTREE} <path> [--symbol S|--range A-B]`, never `cat`.

Return exactly this packet, nothing after it:
verdict: PASS | BLOCK | INCOMPLETE | NEEDS_DECISION
blocking: yes | no
evidence_paths: [test files, the REQ-MVV runner]
changed_paths: [{ART}/coverage.md, …]
next_action: <`demote: REQ-N — <reason>` per unassertable REQ, else "none">
summary_50w: <tests written; red-confirmed yes|no (no → INCOMPLETE)>
