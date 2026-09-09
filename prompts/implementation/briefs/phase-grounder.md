# Phase 3d — decision grounder (one open entry, read-only)
Fields: RDR_PATH NNNN RDR_RESOURCES RDR_HOME WORKTREE ENTRY
(Fields arrive as `NAME=value` lines; `{NAME}` is that value.)

You edit NOTHING — not the record, the artifacts or the source. Your only
writes are scratch under a private `/tmp/{NNNN}-3d-<DEF id>/` (two grounders
once collided on one shared file). You receive no other entry and no Phase 3
finding.

The entry, verbatim:
{ENTRY}

Walk the search ladder one rung at a time (the `ground` rows of the engine's
rdr-write model):
  {RDR_HOME}/bin/rdr-gate ground {NNNN} --tag searched=<none|code|cluster|corpus|rfd> --tag found=<true|false>
Start at `searched=none found=false`. `next:` names the rung, `surface:` its
command: `code` (the source at {WORKTREE} — `rg`/semble the behaviour, a
cheap spike), `cluster` (`{RDR_HOME}/bin/rdr index --cluster-of {NNNN}`, then
each peer's Status and Overrides via `rdr inspect --select MMMM:§…`),
`corpus` (`arc search` the question's nouns over the corpora {RDR_RESOURCES}
names), `rfd` (the RFD/JDR sections it names). Re-ask after each rung with
`searched=<that rung>` and whether it settled the entry; stop at `apply`
(settled — the cite is the answer, no question) or `ask` (every rung
searched, none settled it). Never skip a rung, never default `none`.

Read the record through `{RDR_HOME}/bin/rdr inspect …`, never `sed`/`grep`
the file. Separators `---`, never `===`.

Return exactly this packet, nothing after it:
verdict: PASS | NEEDS_DECISION
blocking: no | yes
evidence_paths: [the cite: file:line | NNNN:§element | corpus hit | RFD §]
changed_paths: []
next_action: <PASS: "code-change" if it alters code, else "none"; NEEDS_DECISION: the question, then `searched=<trail>; found: none`>
summary_50w: <PASS: the decision and its cite; NEEDS_DECISION: why no source settles it>
