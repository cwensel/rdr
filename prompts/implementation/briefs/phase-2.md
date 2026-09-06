# Phase 2 — implementer (one leg of the fixed worklist)
Fields: RDR_PATH NNNN RDR_RESOURCES RDR_HOME ART WORKTREE BRANCH TEST_FRAMEWORK TEST_FILES PREDECESSOR_TESTS RESUME START_SHA START_EPOCH
(Fields arrive as `NAME=value` lines with this path; `{NAME}` below is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed` with no edits.

Your task is the PHASE 2 block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 2 —` to `PHASE 3 —`): read it once — the task, the suite policy,
the deviation Types, ADDITIVE IS NOT EXEMPT, the return-partial contract. Inputs:
{ART}/req-list.md, {ART}/coverage.md, {ART}/impact.md (work a family by its
`## <Family>` block), the Phase 1 tests {TEST_FILES}, predecessor tests
{PREDECESSOR_TESTS}, {RDR_PATH} for context only, {RDR_RESOURCES} for
grounding. NO Phase 3 findings reach you. Write `{ART}/deviations.md`
(even empty). Position: {RESUME}; read its `reads:` first.

The ONLY commit command, after every increment and run (a clean tree
still asks):
  {RDR_HOME}/bin/rdr-leg-commit --start {START_SHA} --since {START_EPOCH} --suite-green <true|false> -m "<subject>"
It commits, reads git and the clock, prints `next:`:
`continue` → next item; `return-green` → REQ-MVV, then PASS;
`return-partial` → the status.md capsule, then INCOMPLETE. Never `git
commit` yourself. `--suite-green` is `true` only from the ONE full run,
the last item; a refused hook prints, nothing moves.

Never edit {RDR_PATH}; read it via `{RDR_HOME}/bin/rdr inspect …`, not
`sed`/`grep`. Scratch under /tmp, never {ART}. Separators `---`, never `===`.

Return exactly this packet, nothing after:
verdict: PASS | BLOCK | INCOMPLETE | NEEDS_DECISION
blocking: yes | no
evidence_paths: [{ART}/deviations.md:line per open deviation]
changed_paths: [source, tests, {ART}/deviations.md]
next_action: <INCOMPLETE: "respawn Phase 2 from the capsule"; else "none">
summary_50w: <green yes|no; commits; items done; open DEF ids>
