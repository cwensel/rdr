# rdr-common — shared bones for the `/rdr-*` skills

Not a skill. Shared preamble the per-stage RDR skills cite by `§<anchor>` so the
seam-bind, number→path resolve, and evidence-glob logic live in one place. Every
`/rdr-*` SKILL.md runs §seam-bind and (except `/rdr-seed`) §rdr-resolve first,
then runs its stage prompt, then prints §next-step.

These skills **do not use a worktree** and **do not edit the consumer project's
source** — they author/inspect RDR documents and write flow evidence. The only writes are to the
RDR file, the evidence dirs, and (Stage 8) the artifact dir, all named via the
seam. `/rdr-status` writes **nothing**.

Consumer repos mount these skills through **per-directory symlink farms**
(e.g. `<consumer>/.claude/skills/<name>` → here), so a `../` reference inside a
skill resolves against the farm, not this tree, and breaks. Therefore **every
support file a skill loads is symlinked beside its SKILL.md** (sibling name:
`rdr-common.md`, `<NN>.prompt.md`, `pre-lock/`, …) — read siblings, never reach
through `..`. Canonical homes stay where they were, in the RDR-process repo:
stage docs at `<rdr-repo>/flow/<NN-stage>.md` (human operating manual) and prompt
bodies under `<rdr-repo>/prompts/`. A skill runs the **prompt file**,
never re-parses the stage doc.

## §seam-bind — resolve the workspace marker (worktree-invariant)

**Fast path — pre-resolved seam.** If the session context already carries an
`RDR seam pre-resolved` block (a consumer's `SessionStart` hook emits one, if it
installs one), take its paths verbatim: substitute the **literal values** for
`$PROCESS_ROOT` / `$RDR_ENV` / `$RDR_DIR` etc. in the snippets below and skip the
resolver block entirely. Harnesses and sessions without that block (other agents,
headless runs) run the resolver as written — same bindings either way.

Run this first. It keys off git topology, so it resolves the same from a consumer
cwd, a consumer worktree, or the flow repo. Shell state dies between Bash tool calls — run
§seam-bind and §rdr-resolve **in one call** (or re-run this block first in any later
call), else `$PROCESS_ROOT`/`$RDR_ENV` are empty when the glob runs (the classic
empty-`RDR_PATH` miss). Source the marker **only** via this block: a direct
`. .rdr-workspace` exits 1 (it needs `$WS` preset above).

```sh
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" && pwd -P)
WS=$(dirname "$(dirname "$GIT_COMMON")")          # workspace root (holds the sibling repos)
[ -f "$WS/.rdr-workspace" ] || { echo "stopped:no-workspace-marker:$WS" >&2; exit 1; }
. "$WS/.rdr-workspace"                             # ONLY sourcing path: $WS must be preset (above) or it exits 1
[ -n "$RDR_ENV" ] && [ -f "$RDR_ENV" ] || { echo "stopped:no-rdr-env:$RDR_ENV" >&2; exit 1; }
```

`$RDR_ENV` is the path map (`{ARTIFACT_DIR}`, `{SPIKE_DIR}`, `{FLOW_DIR}`, the RDR
directory, source-path roots); `$RDR_RESOURCES` is the evidence index. Read
`$RDR_ENV` to bind the concrete paths below. These resolve per-consumer — e.g. a
pinned-seam project keeps RDRs under `$PROCESS_ROOT/rdr/cli/`, evidence under
`$FLOW_ROOT/rdr/evidence/`, artifacts under `$PROCESS_ROOT/rdr/cli/<slug>/`; a
default `_rdr/` project keeps them under its project root. Always read the values
from `$RDR_ENV` — never hardcode these.

## §rdr-resolve — RDR number → file path

Every stage skill except `/rdr-seed` takes a 4-digit number `NNNN`. Resolve it
against the **canonical RDR directory** — `$PROCESS_ROOT/rdr/cli` (the parent of
`{ARTIFACT_DIR}`; `$PROCESS_ROOT` is exported by §seam-bind). Use the concrete
path, **not** a relative `../process/...` parse of `$RDR_ENV` — that string is
cwd-relative and brittle. Glob **only** that dir so the many decoy `NNNN-*` entries
under `$FLOW_ROOT/rdr/evidence/` (per-lens, tooling-pass, spikes — e.g. a real
`evidence/tooling-pass/0039-*.md` is **not** an RDR) can never be picked:

```sh
printf -v NNNN '%04d' "$arg"            # when $arg is all-digits; else treat as slug/path
RDR_DIR="$PROCESS_ROOT/rdr/cli"          # canonical; NOT evidence/ or tooling-pass/
# A no-match glob is a hard error under zsh (nomatch on by default) and aborts the
# line before the guard runs; under bash it stays literal. Enable nullglob per-shell
# so a no-match yields an EMPTY array in both, and the count guard below decides.
if [ -n "$ZSH_VERSION" ]; then setopt null_glob ksh_arrays; else shopt -s nullglob; fi
hits=( "$RDR_DIR"/${NNNN}-*.md )
[ "${#hits[@]}" -ge 1 ] || { echo "stopped:rdr-not-found:$NNNN in $RDR_DIR" >&2; exit 1; }
[ "${#hits[@]}" -eq 1 ] || { echo "stopped:rdr-ambiguous:$NNNN -> ${hits[*]}" >&2; exit 1; }
RDR_PATH="${hits[0]}"   # ksh_arrays (set above) makes [0] the first element in zsh too
RDR_SLUG=$(basename "$RDR_PATH" .md)   # e.g. 0046-auto-named-constraint-identity
```

Accept the number with or without leading zeros. If the user passes a full path or
a slug, accept it directly (still assert it lives under `$RDR_DIR`).

`/rdr-seed` does **not** resolve — it *allocates* the next number (see §rdr-claim).

## §rdr-claim — atomically reserve a number before authoring (`/rdr-seed` only)

`max(NNNN)+1` alone races: two concurrent sessions both read the same max, both
spend a minute authoring, and both write distinct `NNNN-*slug*.md` files — two
RDRs silently sharing one number (no write error, because the slugs differ). The
cure is to **claim the number atomically as the first step**, before any
authoring, and **fail loudly on collision** so the loser just bumps and retries.

Reserve-then-rename. The reservation filename is keyed on the number **only** — no
slug, no session suffix — so `set -C` (noclobber) makes a second claimant's write
genuinely fail rather than coexist under a different name. The number is the lock.
The claim **materializes a copy of TEMPLATE.md** in the same atomic step, so the
reserved file *is* the canonical skeleton — there is never a reason to author
structure from scratch or copy a neighbor RDR for "house style" (that drifts the
template and leaks the neighbor's solution into a Draft that must have none).

```sh
# 1. CLAIM — first thing, before authoring. Atomic copy of TEMPLATE.md; retry-on-collision.
RDR_DIR="$PROCESS_ROOT/rdr/cli"
TEMPLATE="$PROCESS_ROOT/rdr/TEMPLATE.md"
while :; do
  max=$(ls "$RDR_DIR" | grep -oE '^[0-9]{4}' | sort -n | tail -1)
  printf -v NNNN '%04d' "$((10#${max:-0} + 1))"
  reserved="$RDR_DIR/${NNNN}-RESERVED.md"
  if ( set -C; cat "$TEMPLATE" > "$reserved" ) 2>/dev/null; then break; fi  # atomic; loser loops
done
# NNNN is now ours and "$reserved" holds the verbatim template skeleton. A
# concurrent session's max() scan counts ${NNNN}-RESERVED.md, so it picks NNNN+1.
# 2. FILL IN PLACE: edit ONLY the H1 [NUMBER]/[TITLE], Metadata, Problem Statement,
#    Context. Leave every other section exactly as the template ships it — those
#    placeholders ARE the Draft placeholders. No solution/assumptions/research.
# 3. RENAME to the final slug once known:  git mv "$reserved" "$RDR_DIR/${NNNN}-<slug>.md"
```

If authoring is abandoned before the rename, the stray `${NNNN}-RESERVED.md` is the
trace — delete it (or rename it) so the number frees up; until then it correctly
holds the slot. The reservation is intentionally visible to other sessions.

`/rdr-cluster-reconcile` takes a cluster name or an RDR pair, not a single number;
see its SKILL.md.

## §evidence — where the lens/spike/reconcile output lives

`{FLOW_DIR}` is **lens-first, then per-RDR**: `<FLOW_DIR>/<lens>/<RDR_SLUG>/`
(`3amigo`, `critique`, `repeatability`, `cove`), plus siblings `reconcile/`,
`spikes/`, `tooling-pass/`, `action-items/`, and the per-cluster
`cluster-reconcile/<cluster>/`. A re-entry pass appends `iter-N/`
(`<lens>/<RDR_SLUG>/iter-2/`); loose files directly under `<lens>/<RDR_SLUG>/` are
iteration 1. The existence of `<lens>/<RDR_SLUG>/` is the disk signal that that
lens ran — this is what `/rdr-status` reads. Bind `{FLOW_DIR}` / `{SPIKE_DIR}` from
`$RDR_ENV`.

## §run-prompt — run the stage's prompt file

Load the stage prompt **symlinked beside the skill's SKILL.md** — `<NN>.prompt.md`,
or for the dispatch stages 5/8.1/9 the sibling `pre-lock/` dir / gate file /
`launch.md` (canonical homes under `$PROCESS_ROOT/rdr/prompts/`). The prompt body
uses `{RDR_PATH}` / `{RDR_RESOURCES}` / `{RDR_ENV}` / `{FLOW_DIR}` / `{SPIKE_DIR}` /
`{ARTIFACT_DIR}` — bind each from §seam-bind + §rdr-resolve before running. Do
**not** also read the stage `.md` doc; its Goal/gate/advance prose is for the human
driver, not the skill.

Honor the prompt's own **self-detected re-entry**: stages 4/5/9 read the RDR
`Status:` line (`Draft [revised from Final …; re-verify <IDs>]`) or `status.md`
and scope themselves. The skills carry **no `--resume` flag** — re-entry is a
property of the RDR's on-disk state, which the prompt already inspects.

## §delegation — who reads, who writes, what spawns

- **Spawn sub-agents with the built-in `Task` tool** (a.k.a. `Agent`). **Not
  `TaskCreate`** — that is task-list tracking, not a sub-agent spawner. Naming this
  here saves the ToolSearch round-trip every delegating stage otherwise burns.
- **Delegate heavy *reads*, author *writes* in the main context.** The read-heavy
  stages (4, 6, 7) and `/rdr-status` push corpus searches, source spelunks, and
  reading several round-output files to a `Task` sub-agent that returns *verdict +
  evidence pointer* (file:line, or spike command + output path), never raw hits.
  The **edit to the RDR / evidence file happens in the main agent**, not a
  sub-agent — keep authoring in one context (it also sidesteps any consumer-side
  sub-agent write guard on the primary checkout).
  Stage 8 (`launch.md`) is the exception that proves the rule: its orchestrator
  delegates *everything*, by its own stricter contract — do not loosen it.

## §stop-packet — surface, don't fake

When a stage hits a genuine human-judgment fork (a missing spike run, a scope
alarm, a real either/or design call, a gate item genuinely in doubt), **stop and
emit a one-line packet, then wait** — do not proceed degraded or invent a
disposition. Format: `stopped:<code>:<≤80-word question — what's ambiguous + what
answer unblocks>`. Reuse the stage's own `stopped:*` codes where it names them.
This boundary is inherent to the flow (mined: the recurring "I should stop and
surface this rather than fake it"); the skill's job is to make the stop *crisp*,
not to remove the human.

## §next-step — close with a pointer

End every stage skill (not `/rdr-status`) by printing, briefly:
1. the **Review gate** the human checks (one line, from the stage `.md` — quote it,
   don't re-derive), and
2. `Next: /rdr-<next-stage> NNNN [lens]`, plus `/rdr-status NNNN` to re-orient.

Per-stage next pointers: seed→propose→refine→resolve→prelock(per lens — one
`/rdr-prelock <lens>` cycle reviews *and* resolves, looping to convergence;
`repeatability` first loops run-1/2/3→diff in fresh sessions, then the diff
session resolves its `diff.md`)→reconcile→finalize→[cluster-reconcile]→implement.
The branch after `resolve` reads the RDR's **`Profile` field** (the latch Stage 4
writes), not a fresh size inference: `small` skips prelock (resolve→reconcile);
`mid`+ runs the profile's lenses (`$PROCESS_ROOT/rdr/flow/README.md` matrix). The
Stage 7 Gate re-validates the field before lock, so a wrong value cannot
silently route past the lenses.

## Brevity & doctrine

Be brief without being lossy (the flow's standing doctrine —
`$PROCESS_ROOT/rdr/flow/README.md` *Doctrine*). Spend tokens on load-bearing or
complex design; terse everywhere else. Delegate heavy reads (corpus, source,
several round-output files) to a sub-agent that returns verdict + evidence
pointer — the read-heavy stages (4, 6, 7) and `/rdr-status` rely on this. Defer
to CLAUDE.md and the README Doctrine on any conflict.
