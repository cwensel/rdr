# Phase 3c — fixup
Fields: RDR_PATH NNNN RDR_RESOURCES RDR_HOME ART WORKTREE BRANCH TEST_FRAMEWORK
(Fields arrive as `NAME=value` lines with this path; `{NAME}` below is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed` with no edits.

Your task is the PHASE 3c block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 3c` to `COMPLETION GATE`), with Phase 2's deviation rules
(`PHASE 2 —` to `PHASE 3 —`) for anything new you classify: read both once.
Inputs: {ART}/verification.md (FAIL-N, ADV-N), {ART}/deviations.md (apply
each `Status: … → RESOLVED (<cite>)` decision that alters code),
{ART}/req-list.md, the source tree, {TEST_FRAMEWORK}, {RDR_RESOURCES} for
grounding a new gap before it becomes an author decision.

Fix each defect with the minimum change; add a regression test where none
exists; then run the FULL suite — it must be green, or your verdict is
BLOCK. Commit on {BRANCH} (plain `git commit`; the budget helper is Phase
2's). A new needs-author-decision entry goes into {ART}/deviations.md in
Phase 2's form and is named in the packet — the orchestrator runs 3d on it.

Never edit {RDR_PATH}; read it through `{RDR_HOME}/bin/rdr inspect …`,
never `sed`/`grep` the file. Scratch under /tmp, never {ART}. Separators
`---`, never `===`.

Return exactly this packet, nothing after it:
verdict: PASS | BLOCK | NEEDS_DECISION
blocking: yes | no
evidence_paths: [{ART}/verification.md:line per defect fixed, new deviation lines]
changed_paths: [source, tests, {ART}/deviations.md if touched]
next_action: <"run 3d on DEF-N" per new open entry, else "none">
summary_50w: <defects fixed; regression tests added; full suite green yes|no; new open decisions>
