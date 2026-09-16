# Phase 3a — CoVe verifier
Fields: RDR_PATH NNNN RDR_HOME ART WORKTREE BRANCH
(Fields arrive as `NAME=value` lines; `{NAME}` is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed`, no edits; else
`{RDR_HOME}/bin/rdr-leg-mark --role verifier {WORKTREE}`; last call, the
same with `--clear`.

Your task is the PHASE 3a block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 3a` to `PHASE 3b`): read it once. Inputs: {RDR_PATH} and
{ART}/req-list.md ONLY, plus the implementation source tree to run your
inputs against. You do NOT read the Phase 1 tests, {ART}/coverage.md or
{ART}/deviations.md — independence is the point of this pass.

For each REQ-N name an input that would make a correct implementation
visibly violate it; run it against the actual code. Append each real
violation to `{ART}/verification.md` under a `## Phase 3a — CoVe` heading
as a line-leading `FAIL-N` entry (failing input, observed behaviour). No
violation: append the line-leading `## Verdict — clean` under your heading
— an absent file reads as an unrun Phase 3 and stops the gate. Append with
a heredoc (`>>`); 3b writes the same file, after you.

Read the record through `{RDR_HOME}/bin/rdr inspect …`, never `sed`/`grep`
the file. Never edit {RDR_PATH} or the source. Scratch in /tmp, never
{ART}. Separators `---`, never `===`.

ONLY read and test commands, neither budget-cut, each `-C {WORKTREE}` and
under `{RDR_HOME}/bin/` (never `cat`, `go test`): `rdr-leg-read <path>
[--symbol S|--range A-B]`, `rdr-leg-test -- <test command>`. You commit
nothing.

Return exactly this packet, nothing after it:
verdict: PASS | BLOCK
blocking: yes | no
evidence_paths: [{ART}/verification.md:line per FAIL-N]
changed_paths: [{ART}/verification.md]
next_action: none
summary_50w: <one line per FAIL-N, or "clean">
