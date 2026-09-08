# Phase 3c — fixup
Fields: RDR_PATH NNNN RDR_RESOURCES RDR_HOME ART WORKTREE BRANCH TEST_FRAMEWORK START_SHA START_EPOCH
(Fields arrive as `NAME=value` lines with this path; `{NAME}` below is that value.)

First call: `cd {WORKTREE}`; if `git branch --show-current` is not {BRANCH},
return `stopped:worktree-isolation-failed` with no edits.

Your task is the PHASE 3c block of {RDR_HOME}/prompts/implementation/launch.md
(from `PHASE 3c` to `COMPLETION GATE`), with Phase 2's deviation rules
(`PHASE 2 —` to `PHASE 3 —`) for anything new you classify: read both once.
Inputs: {ART}/verification.md, {ART}/deviations.md (apply each
`→ RESOLVED` decision that alters code),
{ART}/triage.md (each IN-SCOPE row not yet `fixed:`/`held:`),
{ART}/req-list.md, the source tree, {TEST_FRAMEWORK}, {RDR_RESOURCES} to
ground a gap before it becomes an author decision.

Minimum change per defect and open row; a regression test where none
exists; rewrite each row's Outcome. The ONLY commit command,
after every increment:
  {RDR_HOME}/bin/rdr-leg-commit --start {START_SHA} --since {START_EPOCH} --suite-green <true|false> -m "<subject>"
`continue` → next; `return-green` (the ONE full run, last) → PASS;
`return-partial` → the {ART}/status.md capsule (`phase: 3c — fixup`,
`next: respawn 3c`), then INCOMPLETE. Never `git commit` yourself. A new
needs-author-decision entry: Phase 2's form in {ART}/deviations.md, named
in the packet.

Never edit {RDR_PATH}; read it via `{RDR_HOME}/bin/rdr inspect …`, not
`sed`/`grep`. Scratch under /tmp, never {ART}. Separators
`---`, never `===`.

Return exactly this packet, nothing after:
verdict: PASS | BLOCK | INCOMPLETE | NEEDS_DECISION
blocking: yes | no
evidence_paths: [file:line per defect fixed, row rewritten, new deviation]
changed_paths: [source, tests, the {ART} ledgers touched]
next_action: <INCOMPLETE: "respawn 3c"; "run 3d on DEF-N" per new entry; else "none">
summary_50w: <defects fixed; rows fixed|held; tests added; full suite green yes|no; new open decisions>
