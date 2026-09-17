# Phase 1 — test author (red before green)
Fields: RDR_PATH NNNN RDR_HOME ART WORKTREE BRANCH TEST_FRAMEWORK TEST_DIR PREDECESSOR_ARTIFACTS

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed` with no edits; else
`{RDR_HOME}/bin/rdr-leg-mark --role test-author {WORKTREE}`; last call, the
same with `--clear`.

Task: the PHASE 1 block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 1 —` to `PHASE 2 —`), read once. Inputs: {ART}/req-list.md,
{PREDECESSOR_ARTIFACTS}, {TEST_FRAMEWORK}, tests under {TEST_DIR}. NO
implementation hints or design notes: do not go looking. Write `{ART}/coverage.md` as the block says.

Red must be committable: where the pre-commit hook vets or compiles
(`go vet ./...`, say), a suite naming a symbol that does not exist yet
cannot be committed. Reach the unwritten surface through a probe that
compiles today and fails at run time (a reflective adapter over
candidate spellings; a lookup reporting the REQ owed): every red is an
assertion failure, never a compile break. A stub you add is non-satisfying
(outside the contract's domain), never the happy value. Commit the red
suite on {BRANCH}.

ONLY read, test and commit commands (never `cat`, `go test`, `git commit`): `{RDR_HOME}/bin/rdr-leg-read -C {WORKTREE} <path> [--symbol S|--range A-B]`,
`rdr-leg-test -C {WORKTREE} -- <test command>` (no budget cut) and
`rdr-leg-commit -C {WORKTREE} -m "<subject>"`. The record:
`{RDR_HOME}/bin/recs inspect …` (`--select {NNNN}:<id>` for exact quotes), never
`sed`/`grep` it; never edit {RDR_PATH}. Scratch in /tmp, never {ART}.
Separators `---`, never `===`.

Return exactly this packet, nothing after:
verdict: PASS | BLOCK | INCOMPLETE | NEEDS_DECISION
blocking: yes | no
evidence_paths: [test files, the REQ-MVV runner]
changed_paths: [{ART}/coverage.md, …]
next_action: <`demote: REQ-N — <reason>` per unassertable REQ, else "none">
summary_50w: <tests written; red-confirmed yes|no (no → INCOMPLETE)>
