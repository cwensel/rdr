# Phase 3b — adversarial reviewer
Fields: RDR_PATH NNNN RDR_HOME ART WORKTREE BRANCH TEST_DIR TEST_FRAMEWORK
(Fields arrive as `NAME=value` lines; `{NAME}` is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed`, no edits; else
`{RDR_HOME}/bin/rdr-leg-mark --role verifier {WORKTREE}`; last call, the
same with `--clear`.

Task: the PHASE 3b block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 3b` to `PHASE 3d`), read once. Inputs: the record's Failure
Modes section (`{RDR_HOME}/bin/rdr inspect {NNNN}` lists the section ids;
`inspect --select {NNNN}:§<id> {RDR_PATH}` prints it), {ART}/req-list.md, the
source tree. You do NOT read the Phase 1 tests or any Phase 3a finding.

As a senior reviewer who thinks this is wrong: name the three most likely
failure modes, anchored in that section, and the test that catches each.
Add the missing tests under {TEST_DIR} ({TEST_FRAMEWORK}); confirm each
FAILS now — a test that passes catches nothing. Append findings to
`{ART}/verification.md` under a `## Phase 3b — Adversarial` heading as
line-leading `ADV-N` entries; none → the line-leading `## Verdict — clean`
under it (an absent file reads as an unrun Phase 3). Append by heredoc
(`>>`); 3a writes the same file.

ONLY read, test and commit commands, none budget-cut, each `-C {WORKTREE}`
and under `{RDR_HOME}/bin/` (never `cat`, `go test`, `git commit`):
`rdr-leg-read <path> [--symbol S|--range A-B]`, `rdr-leg-test -- <test
command>`, and for the added tests on {BRANCH} `rdr-leg-commit -m "<subject>"`.

Foreground only; never background, wait, or use Monitor — an armed wait dies
with you and the harness reports you COMPLETE. Never `sed`/`grep` the record;
never edit {RDR_PATH} or the implementation. Scratch in /tmp, never {ART}.
Separators `---`, never `===`.

Return exactly this packet, nothing after it:
verdict: PASS | BLOCK
blocking: yes | no
evidence_paths: [{ART}/verification.md:line per ADV-N, test files added]
changed_paths: [{ART}/verification.md, test files]
next_action: none
summary_50w: <failure modes named; tests added; which fail now>
