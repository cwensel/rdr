# Phase 3b — adversarial reviewer
Fields: RDR_PATH NNNN RDR_HOME ART WORKTREE BRANCH TEST_DIR TEST_FRAMEWORK
(Fields arrive as `NAME=value` lines; `{NAME}` below is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed` with no edits.

Your task is the PHASE 3b block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 3b` to `PHASE 3d`): read it once. Inputs: the record's Failure
Modes section (`{RDR_HOME}/bin/rdr inspect {NNNN}` lists the section ids;
`inspect --select {NNNN}:§<failure-modes id> {RDR_PATH}` prints it),
{ART}/req-list.md, and the implementation source tree. You do NOT read the
Phase 1 tests or any Phase 3a finding.

As a senior reviewer who thinks this is wrong: name the three most likely
failure modes, anchored in that section, and the test that catches each.
Add the missing tests under {TEST_DIR} ({TEST_FRAMEWORK}) and confirm each
FAILS against the current implementation — a test that passes catches
nothing. Append findings to `{ART}/verification.md` under a `## Phase 3b —
Adversarial` heading as line-leading `ADV-N` entries; none → append the
line-leading `## Verdict — clean` under your heading (an absent file reads
as an unrun Phase 3). Append with a heredoc (`>>`); 3a writes the same file
in parallel. Commit added tests on {BRANCH}.

Never `sed`/`grep` the record file; never edit {RDR_PATH} or the
implementation. Scratch under /tmp, never {ART}. Separators `---`, never `===`.

Reads: `{RDR_HOME}/bin/rdr-leg-read <path> [--symbol S | --range A-B]`, never `cat`.

Return exactly this packet, nothing after it:
verdict: PASS | BLOCK
blocking: yes | no
evidence_paths: [{ART}/verification.md:line per ADV-N, test files added]
changed_paths: [{ART}/verification.md, test files]
next_action: none
summary_50w: <failure modes named; tests added; which fail now>
