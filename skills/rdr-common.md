# rdr-common — shared bones for the `/rdr-*` skills

Not a skill. Shared preamble the per-stage RDR skills cite by `§<anchor>` so the
seam-bind, number→path resolve, and evidence-glob logic live in one place. Every
`/rdr-*` SKILL.md runs §seam-bind and (except `/rdr-seed`) §rdr-resolve first,
then runs its stage prompt, then (when autocommit is on) runs §commit on its own
files, then prints §next-step.

These skills **stay on the current branch** — never `git branch`/`switch -c`/
`checkout -b`, never a worktree (the "branch before feature work" reflex is wrong
here). Many sessions share one branch on the instance folder; the §commit
compare-and-swap *is* the isolation, so branching hides work from siblings and a
worktree swaps the branch out from under them. They also **do not edit the consumer
project's source** — they author/inspect RDR documents and write flow evidence. The
only writes are to the RDR file, the evidence dirs, and (Stage 8) the artifact dir,
all named via the seam. `/rdr-status` writes **nothing**.

## Invocation syntax

These skills are invoked as `$rdr-*` in Codex and `/rdr-*` in Claude. Treat them
as the same operation with different UI prefixes. When printing a next command or
a one-command fix, prefer the active harness prefix; if unknown, print both forms:
`Next: $rdr-status NNNN in Codex, or /rdr-status NNNN in Claude`.

Consumer repos mount these skills through **per-directory symlink farms**
(e.g. `<consumer>/.claude/skills/<name>` → here), so a `../` reference inside a
skill resolves against the farm, not this tree, and breaks. Therefore **every
support file a skill loads is symlinked beside its SKILL.md** (sibling name:
`rdr-common.md`, `<NN>.prompt.md`, `pre-lock/`, …) — read siblings, never reach
through `..`. Canonical homes stay where they were, in the RDR engine repo
(`$RDR_HOME`, exported by §seam-bind): stage docs at
`$RDR_HOME/stages/<NN-stage>.md` (human operating manual) and prompt bodies
under `$RDR_HOME/prompts/`. A skill runs the **prompt file**, never
re-parses the stage doc.

## §seam-bind — resolve the workspace marker (worktree-invariant)

**Fast path — pre-resolved seam.** If the session context already carries an
`RDR seam pre-resolved` block (a consumer's `SessionStart` hook emits one, if it
installs one), take its paths verbatim: substitute the **literal values** for the
contract vars (`$RDR_HOME` / `$RDR_RECORDS` / `$RDR_EVIDENCE` / `$RDR_ENV` / `$RDR_RESOURCES`,
plus the `$RDR_AUTOCOMMIT` gate var §commit reads) in the snippets below and skip the
resolver block entirely. The block carries `RDR_AUTOCOMMIT` precisely so the fast path
doesn't blind §commit's gate — if the block omits it, treat `RDR_AUTOCOMMIT` as unset
(autocommit off). Harnesses and sessions without that block (other agents,
headless runs) run the resolver as written — same bindings either way.

**Run the block below verbatim — do not abbreviate, paraphrase, or source the
marker file directly.** The marker guards against direct sourcing: it exits 1 if
`$WS` is unset, and `$WS` is only produced by the `git rev-parse` lines that come
before the source call. Skipping those lines is the failure mode.

Run this first. It keys off git topology, so it resolves the same from a consumer
cwd, a consumer worktree, or the flow repo. **Nearest marker wins**: a repo-local
`$PROJECT/.rdr/workspace` (this repo's own RDR env, inside its gitignored `.rdr/` — the
default) takes precedence over the shared `$WS/.rdr-workspace` (a workspace seam siblings
opt into via `--workspace`) — like `.git` or `.editorconfig`, the closest one governs. `$PROJECT`
is `dirname` of the git-common-dir, so a worktree still resolves its main repo's local marker. Shell
state dies between Bash tool calls — run §seam-bind and §rdr-resolve **in one call**
(or re-run this block first in any later call), else `$RDR_RECORDS`/`$RDR_ENV` are
empty when the glob runs (the classic empty-`RDR_PATH` miss). Source the marker
**only** via this block; `$RDR_MARKER` records which one resolved.

```sh
# §seam-bind — copy/run verbatim; do NOT source the marker file directly (exits 1 without $WS).
# $WS must be derived from git topology first — the three lines below do that.
GC=$(git rev-parse --git-common-dir 2>/dev/null) || { echo "stopped:not-in-a-project (run /rdr-* from inside the consumer repo)" >&2; exit 1; }
GIT_COMMON=$(cd "$GC" && pwd -P)
PROJECT=$(dirname "$GIT_COMMON")                  # this repo's root (worktree-invariant: git-common-dir is the main .git)
WS=$(dirname "$PROJECT")                            # workspace root — required by the marker; set here, not by sourcing it
export PROJECT WS                                   # both exported so a marker can anchor on either ($PROJECT repo-local, $WS workspace)
# Nearest marker wins: a repo-local marker overrides / replaces the shared workspace one.
# Repo-local lives INSIDE .rdr/ (already gitignored — no project-level .gitignore edit).
if   [ -f "$PROJECT/.rdr/workspace" ]; then RDR_MARKER="$PROJECT/.rdr/workspace"  # repo-local scope (default)
elif [ -f "$WS/.rdr-workspace" ];      then RDR_MARKER="$WS/.rdr-workspace"       # workspace scope (shared, --workspace)
else echo "stopped:no-marker — run /rdr-init in this repo (looked in $PROJECT/.rdr and $WS)" >&2; exit 1; fi
. "$RDR_MARKER"                                     # source ONLY via this path (never directly)
[ -n "$RDR_ENV" ] && [ -f "$RDR_ENV" ] || { echo "stopped:no-rdr-env:$RDR_ENV" >&2; exit 1; }
[ -n "$RDR_HOME" ] || { echo "stopped:no-rdr-home — run /rdr-init to write the marker" >&2; exit 1; }
# $RDR_RECORDS is required for every stage except /rdr-seed-into-a-fresh-dir; resolve/claim assert it themselves.
```

The marker exports the **five-var engine contract** every skill may read:
- `$RDR_HOME` — the RDR engine repo (`stages/`, `prompts/`, `skills/`, `TEMPLATE.md`).
- `$RDR_RECORDS` — **this consumer's RDR-instances directory** (absolute path; the
  parent of `{ARTIFACT_DIR}`). The one place the "where do my RDRs live" decision
  is recorded. `/rdr-init` writes it; resolve/claim read it.
- `$RDR_EVIDENCE` — **this consumer's evidence root** (absolute path; the base
  `{EVIDENCE_DIR}` / `{SPIKE_DIR}` hang under, per-RDR-first: `<slug>/evidence/<lens>/`).
  Decouples lens/spike output from the records tree so a consumer can stage it
  anywhere (its own dir, a sibling repo, gitignored scratch). **Defaults to
  `$RDR_RECORDS`** when `/rdr-init` is not told otherwise — then each RDR's
  `evidence/` and `artifacts/` are siblings under one `<slug>/` folder.
- `$RDR_ENV` — the path-map data file (`{ARTIFACT_DIR}`, `{SPIKE_DIR}`, `{EVIDENCE_DIR}`
  resolved against `$RDR_RECORDS` / `$RDR_EVIDENCE`, plus source-path roots for the
  reuse audit).
- `$RDR_RESOURCES` — the evidence index data file.

These resolve per-consumer — e.g. a pinned-seam project records
`RDR_RECORDS=<its-repo>/rdr/cli`, a default `.rdr/` project records `RDR_RECORDS` under
its own root. Read `$RDR_ENV` for the staging paths; read `$RDR_RECORDS` / `$RDR_EVIDENCE`
directly for the records and evidence roots. **Never** hardcode a repo root or a
`/rdr/cli` path shape, and never parse `$RDR_ENV`'s cwd-relative strings — the marker
already gives you absolute values.

## §rdr-resolve — RDR number → file path

Every stage skill except `/rdr-seed` takes a 4-digit number `NNNN`. Resolve it
against the **canonical RDR directory** — `$RDR_RECORDS`, the absolute path the marker
exports (this consumer's RDR-instances dir; the parent of `{ARTIFACT_DIR}`).
`$RDR_RECORDS` is bound by §seam-bind — read it, **never** recompute it from a repo
root or parse it out of `$RDR_ENV`'s cwd-relative strings (brittle). Glob **only**
that dir so the many decoy `NNNN-*` entries
under `$FLOW_ROOT/rdr/evidence/` (per-lens, tooling-pass, spikes — e.g. a real
`evidence/tooling-pass/0039-*.md` is **not** an RDR) can never be picked:

```sh
printf -v NNNN '%04d' "$arg"            # when $arg is all-digits; else treat as slug/path
# $RDR_RECORDS is exported by §seam-bind (the marker). Do NOT recompute it here.
[ -n "$RDR_RECORDS" ] && [ -d "$RDR_RECORDS" ] || { echo "stopped:no-rdr-dir:$RDR_RECORDS" >&2; exit 1; }
# A no-match glob is a hard error under zsh (nomatch on by default) and aborts the
# line before the guard runs; under bash it stays literal. Enable nullglob per-shell
# so a no-match yields an EMPTY array in both, and the count guard below decides.
if [ -n "$ZSH_VERSION" ]; then setopt null_glob ksh_arrays; else shopt -s nullglob; fi
hits=( "$RDR_RECORDS"/${NNNN}-*.md )
[ "${#hits[@]}" -ge 1 ] || { echo "stopped:rdr-not-found:$NNNN in $RDR_RECORDS" >&2; exit 1; }
[ "${#hits[@]}" -eq 1 ] || { echo "stopped:rdr-ambiguous:$NNNN -> ${hits[*]}" >&2; exit 1; }
RDR_PATH="${hits[0]}"   # ksh_arrays (set above) makes [0] the first element in zsh too
RDR_SLUG=$(basename "$RDR_PATH" .md)   # e.g. 0046-auto-named-constraint-identity
```

Accept the number with or without leading zeros. If the user passes a full path or
a slug, accept it directly (still assert it lives under `$RDR_RECORDS`).

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
# $RDR_RECORDS is exported by §seam-bind (the marker); do NOT recompute it.
[ -n "$RDR_RECORDS" ] && [ -d "$RDR_RECORDS" ] || { echo "stopped:no-rdr-dir:$RDR_RECORDS" >&2; exit 1; }
TEMPLATE="$RDR_HOME/TEMPLATE.md"
while :; do
  max=$(ls "$RDR_RECORDS" | grep -oE '^[0-9]{4}' | sort -n | tail -1)
  printf -v NNNN '%04d' "$((10#${max:-0} + 1))"
  reserved="$RDR_RECORDS/${NNNN}-RESERVED.md"
  if ( set -C; cat "$TEMPLATE" > "$reserved" ) 2>/dev/null; then break; fi  # atomic; loser loops
done
# NNNN is now ours and "$reserved" holds the verbatim template skeleton. A
# concurrent session's max() scan counts ${NNNN}-RESERVED.md, so it picks NNNN+1.
# 2. FILL IN PLACE: edit ONLY the H1 [NUMBER]/[TITLE], Metadata, Problem Statement,
#    Context. Leave every other section exactly as the template ships it — those
#    placeholders ARE the Draft placeholders. No solution/assumptions/research.
# 3. RENAME to the final slug once known:  git mv "$reserved" "$RDR_RECORDS/${NNNN}-<slug>.md"
```

If authoring is abandoned before the rename, the stray `${NNNN}-RESERVED.md` is the
trace — delete it (or rename it) so the number frees up; until then it correctly
holds the slot. The reservation is intentionally visible to other sessions.

`/rdr-cluster-reconcile` takes a cluster name or an RDR pair, not a single number;
see its SKILL.md.

## §evidence — where the lens/spike/reconcile output lives

The evidence tree is **per-RDR-first**, rooted at `$RDR_EVIDENCE` (the contract var)
and symmetric with the per-RDR `{ARTIFACT_DIR}` under `$RDR_RECORDS`: every RDR owns
`<RDR_EVIDENCE>/<RDR_SLUG>/evidence/`, holding one folder per lens
(`grounding`, `3amigo`, `critique`, `repeatability`, `cove`) plus siblings `reconcile/`,
`spikes/`, `tooling-pass/`, `action-items/`, and the per-cluster
`cluster-reconcile/<cluster>/`. `{EVIDENCE_DIR}` is the **fully-bound per-lens dir** —
`<RDR_EVIDENCE>/<RDR_SLUG>/evidence/<lens>/`; `{SPIKE_DIR}` is
`<RDR_EVIDENCE>/<RDR_SLUG>/evidence/spikes/`. A re-entry pass appends `iter-N/`
(`…/evidence/<lens>/iter-2/`); loose files directly under `…/evidence/<lens>/` are
iteration 1. The existence of `…/evidence/<lens>/` is the disk signal that that
lens ran — this is what `/rdr-status` reads. **Not every stage leaves a disk
signal:** Stage 4 Resolve writes an evidence folder only when it names spikes;
a pure source-search resolve records its verdicts inline in the RDR (CAs flip to
`Verified`) and creates no `<RDR_SLUG>/` dir. So a missing per-RDR folder means
"no lens/spike artifact yet," **not** "Resolve hasn't run" — the CA verdicts in
the RDR body are the authority for Resolve-done. Bind the concrete `{EVIDENCE_DIR}` /
`{SPIKE_DIR}` from `$RDR_ENV`, which resolves them against `$RDR_EVIDENCE`. When
`$RDR_EVIDENCE` defaults to `$RDR_RECORDS`, an RDR's `evidence/` and `artifacts/`
sit as siblings under one `<RDR_SLUG>/` folder; when a consumer points
`$RDR_EVIDENCE` at its own dir/repo, the same per-RDR shape lives there instead,
keeping evidence isolated and self-identifying for commits.

## §run-prompt — run the stage's prompt file

Load the stage prompt **symlinked beside the skill's SKILL.md** — `<NN>.prompt.md`,
or for the dispatch stages 5/7.1/8 the sibling `pre-lock/` dir / gate file /
`launch.md` (canonical homes under `$RDR_HOME/prompts/`). The prompt body
uses `{RDR_PATH}` / `{RDR_RESOURCES}` / `{RDR_ENV}` / `{EVIDENCE_DIR}` / `{SPIKE_DIR}` /
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

## §no-heartbeat — resume from state, not a timer

Stages resume from durable on-disk state — `status.md`, evidence folders, the RDR
`Status:` line — plus `$rdr-status` / `/rdr-status`, or a subagent/task completion
notification. **Not** from routine scheduled/time-based wakeups (no heartbeat polling
that just re-reads and restates status). A bounded fallback wakeup is permitted *only*
when an external tool emits no completion signal **and** the next action is genuinely
blocked: it must name the uncertainty it guards, self-clear after one firing, and do no
already-completed work on wake (read state first, continue from the next incomplete step).
Echo §stop-packet's "surface, don't fake" — prefer explicit user re-entry over a timer.

## §next-step — close with a decision packet

End every stage skill (not `/rdr-status`) by printing a compact **close packet** — one
line per field — so a human or next session can decide whether to continue *without
re-reading the RDR*:

```
Outcome: <stage verdict in user terms>
Gate: PASS / NOT READY / <stopped-code>  (name the failed gate item only if it is the reason not to continue)
RDR delta: <1-3 bullets naming the changed recommendation/assumptions/contracts/tests/risks — or "none owed because <reason>">
Deviations: <none | accepted/dismissed/deferred findings, downgraded assumptions, scope changes>
Continue check: <why the next stage is justified, or what judgment still remains>
Next: $rdr-<next-stage> NNNN [lens] (Codex) / /rdr-<next-stage> NNNN [lens] (Claude) — or $rdr-status NNNN / /rdr-status NNNN to re-orient
```

`RDR delta` is **mandatory**: name at least one concrete RDR content change, or state
`none owed because <reason>` for a no-edit verification stage. The old gate-only footer
(Review gate + Next) is no longer acceptable. The Review gate is **compressed** to the
`Gate:` line unless a failed gate item is the reason not to continue (then name it). Keep
*both* command spellings in `Next` (the dual-surface convention). Do not enumerate
per-stage delta contents here.

Per-stage next pointers: seed→propose→refine→resolve→prelock(per lens — one
`/rdr-prelock <lens>` cycle reviews *and* resolves, looping to convergence;
`repeatability` first loops run-1/2/3→diff in fresh sessions, then the diff
session resolves its `diff.md`)→reconcile→finalize→[cluster-reconcile]→implement.
The branch after `resolve` reads the RDR's **`Profile` field** (the latch Stage 4
writes), not a fresh size inference: `small` skips prelock (resolve→reconcile);
`mid`+ runs the profile's lenses (`$RDR_HOME/stages/README.md` matrix). The
finalize Gate re-validates the field before lock, so a wrong value cannot
silently route past the lenses.

For any `prelock` close-out, compute the next pointer from the **current**
`Profile` field and the full matrix row, not from the lens just run or a stale
profile remembered earlier in the session. Existing evidence folders satisfy
their matching lenses, but they do not shrink the row: when a reset/escalation
makes the RDR `foundational`, `cove 3amigo critique repeatability` is the required
set until all four are present/resolved.

## §commit — optionally commit this run's *own* files, fast, no exploration

A writing stage already knows the exact files it wrote — `$RDR_PATH`,
`$RDR_RECORDS/README.md`, and its `$RDR_EVIDENCE/$RDR_SLUG/evidence/<subdir>` — bound by
§seam-bind + §rdr-resolve. So it can commit *those paths and nothing else* without ever
running `git status` / `git add -A` / inspecting "what's dirty". This is the whole point:
**no reconnaissance, no round-trip, and no confusion about what this session owns** —
the owned set is a property of the stage, not a discovery. It also makes parallel
`/rdr-*` runs safe **without a worktree**: each run commits through its *own* private
index and advances **the current branch** with a compare-and-swap, so concurrent runs
never fight over `index.lock` and never clobber each other's commits. **Never branch
first** (`git branch`/`switch -c`/`checkout -b`) — that lands commits where siblings
can't see them and defeats the CAS; stay on the branch as found.

**Records and evidence may live in DIFFERENT repos.** `$RDR_RECORDS` and `$RDR_EVIDENCE`
are independent absolute paths the marker exports — a consumer can stage records in one
repo (e.g. `process/rdr/cli`) and evidence in a sibling (e.g. `flow/rdr/evidence`). So
`rdr_commit` is **repo-aware**: it derives the owning repo from the paths themselves and
commits there via `git -C` — it does **not** assume cwd's repo, and the doc commit and the
evidence commit may land in two different repos. Pass paths that all live in ONE repo per
call (records paths together, evidence paths together); the helper asserts this.

**When it runs (autocommit gate).** The gate var is `RDR_AUTOCOMMIT`, exported by the
marker (§seam-bind sources it) — add `RDR_AUTOCOMMIT=true` beside the other `RDR_*`
exports in the workspace/`.rdr` marker to default-on a project; unset/false = off. A
`--commit` arg forces on for this run, `--no-commit` forces off. Precedence:
`--no-commit` > `--commit` > `RDR_AUTOCOMMIT` > off. Resolve it with `rdr_autocommit_on`
below and skip §commit entirely when it returns false (the human commits manually).

**Run the functions below verbatim** (same doctrine as §seam-bind — do not paraphrase or
abbreviate; weaker models must run them literally). Call `rdr_commit` once per logical
commit, passing the subject then the owned absolute paths:

```sh
# §commit — commit an EXACT owned path-set to its OWN repo. No staging churn, no
# git-status read, repo-aware, parallel-safe (private index + compare-and-swap ref update).
# Usage:  rdr_autocommit_on "$@" && rdr_commit "docs(rdr): … cli/$NNNN — <summary>" "$RDR_PATH" "$RDR_RECORDS/README.md"
# Preconditions: §seam-bind ran (RDR_AUTOCOMMIT + paths bound). Paths are ABSOLUTE; all in one repo per call.

rdr_autocommit_on() {                                  # reads the gate; "$@" = the skill's own args
  case " $* " in *" --no-commit "*) return 1;; *" --commit "*) return 0;; esac
  [ "$RDR_AUTOCOMMIT" = "true" ]                        # marker var; unset/anything-else = off
}

rdr_commit() {
  SUBJECT="$1"; shift                                  # remaining args = the owned ABSOLUTE paths
  [ "$#" -ge 1 ] || return 0
  # Derive the owning repo from the FIRST path; assert every path lives in that same repo.
  REPO=$(cd "$(dirname "$1")" 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null) || {
    echo "stopped:commit-no-repo:$1" >&2; return 1; }
  for p in "$@"; do
    r=$(cd "$(dirname "$p")" 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null)
    [ "$r" = "$REPO" ] || { echo "stopped:commit-cross-repo:$p not in $REPO" >&2; return 1; }
  done
  TMPIDX="$REPO/.git/rdr-skillidx-$$-${NNNN:-x}"        # per-run PRIVATE index in the TARGET repo
  n=0
  while [ "$n" -lt 50 ]; do                            # CAS retry cap — generous; the retry is cheap
    PARENT=$(git -C "$REPO" rev-parse HEAD)
    GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" read-tree "$PARENT"        # seed full tree from HEAD
    GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" add -- "$@"                # stage ONLY my paths, in MY index
    TREE=$(GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" write-tree)
    if [ "$TREE" = "$(git -C "$REPO" rev-parse "$PARENT^{tree}")" ]; then  # no-op guard: my paths unchanged
      rm -f "$TMPIDX"; return 0                                            # → no empty commit, silent
    fi
    COMMIT=$(GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" commit-tree "$TREE" -p "$PARENT" -m "$SUBJECT")
    if git -C "$REPO" update-ref HEAD "$COMMIT" "$PARENT" 2>/dev/null; then   # CAS: only if HEAD unmoved
      # Reconcile ONLY my paths in the REAL index so `git status` is clean afterward,
      # without disturbing the user's own staged work. `git reset -- <pathspec>` handles
      # both file and DIRECTORY args (evidence is passed as a dir); retry if index.lock is busy.
      r=0; while [ "$r" -lt 20 ]; do git -C "$REPO" reset -q HEAD -- "$@" 2>/dev/null && break; r=$((r+1)); done
      rm -f "$TMPIDX"
      echo "committed $(git -C "$REPO" rev-parse --short HEAD)  $SUBJECT"
      return 0
    fi
    n=$((n+1))                                          # CAS lost (a parallel run advanced HEAD): retry
    sleep "0.0$((n % 9))"                               # brief jittered backoff so racers don't re-collide
  done
  rm -f "$TMPIDX"
  echo "stopped:commit-contended — HEAD moved 50×; the paths are written, commit manually" >&2
  return 1
}
```

**What each stage commits** (owned path-set + subject — never `git add -A`):

| Stage | Doc commit (`$RDR_PATH` [+ `$RDR_RECORDS/README.md`]) | Evidence commit (separate) |
|-------|---|---|
| seed | `docs(rdr): seed cli/NNNN <slug>` (`$RDR_PATH` + `$RDR_RECORDS/README.md` — the new index row) | — |
| propose | `docs(rdr): propose cli/NNNN — <summary>` | — |
| refine | `docs(rdr): refine cli/NNNN — <summary>` | — |
| resolve | `docs(rdr): resolve cli/NNNN — <summary>` | `chore(rdr): cli/NNNN spike evidence` (if a spike wrote) |
| prelock (non-repeatability lens) | `docs(rdr): prelock cli/NNNN — <lens> pass` | `chore(rdr): cli/NNNN <lens> evidence` |
| reconcile | `docs(rdr): reconcile cli/NNNN — <summary>` | `chore(rdr): cli/NNNN reconcile evidence` |
| finalize | `docs(rdr): finalize cli/NNNN <slug> (Gate PASS)` | — |
| cluster-reconcile | (commits at finalize) | `chore(rdr): cli/NNNN cluster-reconcile <cluster>` |
| implement | (code-repo `feat(...)` commit — its own contract) | artifact files only, if gated on |

Two commits, never one: the doc/README commit is the **design history** (`docs(rdr):`,
real subject — *never* `fixup!`; per the no-fixup doctrine, RDR commits ARE the history);
the evidence subtree is a separate `chore(rdr):` commit so the `docs(rdr)` log stays
readable. Match the subject grammar already in the consumer's log
(`docs(rdr): <stage> cli/NNNN — <one-line>`).

**The `repeatability` lens is the one exception to "commit at the lens."** Its `run-1/2/3`
each run in a *fresh* session that writes only `evidence/repeatability/run-N.md` and does
**not** touch the RDR doc. So each run session commits just its own run file
(`chore(rdr): cli/NNNN repeatability run-N`) — that keeps the loop tight (a fresh session's
owned set is one file, unambiguous). The **doc commit is deferred to the diff session**,
the first point the RDR doc actually changes (`docs(rdr): prelock cli/NNNN — repeatability`).
**The diff session's evidence commit covers the whole `repeatability/` dir**, not just
`diff.md` (`chore(rdr): cli/NNNN repeatability evidence`) — so it is **self-healing**: any
run whose own session did not commit it (autocommit was off then, or the runs predate
enabling it) is swept in here, alongside `diff.md`. The no-op guard makes this idempotent —
already-committed runs add nothing. So however the per-run commits went, the lens lands
fully committed at the diff, never leaving a straggler for the human to chase.

## Brevity & doctrine

Be brief without being lossy (the flow's standing doctrine —
`$RDR_HOME/stages/README.md` *Doctrine*). Spend tokens on load-bearing or
complex design; terse everywhere else. Delegate heavy reads (corpus, source,
several round-output files) to a sub-agent that returns verdict + evidence
pointer — the read-heavy stages (4, 6, 7) and `/rdr-status` rely on this. Defer
to CLAUDE.md and the README Doctrine on any conflict.
