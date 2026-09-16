# Phase 0 — spec auditor
Fields: RDR_PATH NNNN RDR_HOME ART WORKTREE PREDECESSOR_ARTIFACTS
(Fields arrive as `NAME=value` lines; `{NAME}` is that value.)

First call: `cd {WORKTREE}`; if `git rev-parse --show-toplevel` is not
{WORKTREE}, return `stopped:worktree-isolation-failed`, no edits; else
`{RDR_HOME}/bin/rdr-leg-mark --role verifier {WORKTREE}`; last call, the
same with `--clear`.

Your task is the PHASE 0 block of {RDR_HOME}/prompts/implementation/launch.md
(from the line `PHASE 0 —` to the line `PHASE 1 —`): read that block once,
then do its four steps — `{ART}/req-list.md` (REQ-N exact quotes with element
ids, EXCLUDED, ASSUMPTION, QUESTIONS) and `{ART}/impact.md` via
`{RDR_HOME}/bin/rdr impact {NNNN} --literal … > {ART}/impact.md`. Create {ART}
if missing. Predecessor req-list/deviations to read first: {PREDECESSOR_ARTIFACTS}.

Read the record through the projector, never `sed`/`grep` the record file:
`{RDR_HOME}/bin/rdr inspect {NNNN}` (the id list), `inspect --json --filter
elements,counts {RDR_PATH}`, `inspect --select {NNNN}:C4 {RDR_PATH}` (exact
bytes — batch selects by summed line range, ~250 lines a call, read each id
once). Source and artifacts through `{RDR_HOME}/bin/rdr-leg-read -C
{WORKTREE} <path> [--symbol S|--range A-B]`, never `cat`; you run no suite
and commit nothing. Never edit {RDR_PATH}. Scratch goes under /tmp, never
{ART}. Output separators are `---`, never `===`.

Return exactly this packet, nothing after it:
verdict: PASS | BLOCK | INCOMPLETE | NEEDS_DECISION
blocking: yes | no
evidence_paths: [file:line, …]
changed_paths: [{ART}/req-list.md, {ART}/impact.md]
next_action: <QUESTIONS verbatim if any, or a stopped:* from impact, else "none">
summary_50w: <REQ count, REQ-MVV id, impact.md `rows:` count>
