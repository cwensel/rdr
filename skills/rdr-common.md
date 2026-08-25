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
is `dirname` of the git-common-dir, so a worktree still resolves its main repo's local marker.
Source the marker **only** via this block; `$RDR_MARKER` records which one resolved.

**Shell state dies between Bash tool calls**, so anything you still need from the
marker must be bound in the *same* call that uses it. Bind once per call, at the
top — never carry an `export RDR_…=…` prefix from one call to the next, and never
re-run this block just to reach `rdr`.

**`rdr` never needs it.** The binary finds the marker itself, by the same
nearest-wins rule, and reads `$RDR_RECORDS` and `$RDR_SOURCE_REPO` from it — so a
bare `"$RDR_HOME/bin/rdr" inspect …` works from any directory in the project with
no seam bound at all. An exported var still wins over the marker, and a flag over
both. Run §seam-bind for what the tool does *not* read: `$RDR_EVIDENCE`,
`$RDR_ENV`, `$RDR_RESOURCES`, `$RDR_AUTOCOMMIT`.

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

**§source-root — the tree `--repo` greps.** `$RDR_SOURCE_REPO`, exported by the marker:
the checkout whose `path::Symbol` anchors an RDR cites. `/rdr-init` writes it;
never derive it. It is **not** `$PROJECT` — under a workspace-scope marker
`$PROJECT` is whichever sibling you invoked from (records and evidence repos are
both valid cwds), so it names the wrong tree. `$RDR_ENV` cannot supply it either:
that file lists *modules* (`internal/…/x.go`) anchored at this root, never the
root itself.

`--repo` defaults to it, so no call spells it out. Unset, or an anchor naming a
foreign codebase (a third-party library cited for contrast), leaves those edges
unresolved — say so.
**A wrong root is worse than none**: grepping a tree the symbol was never in
returns `resolved:false`, a false finding a consumer chases, where omitting
`--repo` returns the key **absent**, which honestly says nothing looked.

## §rdr-resolve — RDR number → file path

Every stage skill except `/rdr-seed` takes a 4-digit number `NNNN`. Resolve it
against the **canonical RDR directory** — `$RDR_RECORDS`, the absolute path the marker
exports (this consumer's RDR-instances dir; the parent of `{ARTIFACT_DIR}`).
`$RDR_RECORDS` is bound by §seam-bind — read it, **never** recompute it from a repo
root or parse it out of `$RDR_ENV`'s cwd-relative strings (brittle). Look in
**only** that dir so the many decoy `NNNN-*` entries
under `$FLOW_ROOT/rdr/evidence/` (per-lens, tooling-pass, spikes — e.g. a real
`evidence/tooling-pass/0039-*.md` is **not** an RDR) can never be picked:

```sh
# One call, no seam needed. `rdr` finds the marker itself for the records dir,
# pads the number (3 -> 0003, decimal, never octal), skips a
# NNNN-slug-postmortem.md sibling, and names both files on a real collision.
# On failure it exits 2 having already printed stopped:no-such-record /
# stopped:ambiguous-record with the directories it searched — let that stand as
# the stop reason; do not restate it as something else.
RDR_PATH=$("$RDR_HOME/bin/rdr" inspect --json --filter path "$arg" |
           sed -n 's/.*"path": "\([^"]*\)".*/\1/p' | head -1) || exit 1
[ -n "$RDR_PATH" ] || { echo "stopped:rdr-not-found:$arg" >&2; exit 1; }
RDR_SLUG=$(basename "$RDR_PATH" .md)   # e.g. 0046-auto-named-constraint-identity
```

**Reading a record** — this one or any peer — through the projector, never
`sed -n`/`grep` on the file. `inspect NNNN` first: its `§` rows are the sections
with line ranges, then every element — the whole read plan in ~150 lines (not
`--filter outline`, ~800, nor `--select elements`, 25× larger). Then read by id:
`--select NNNN:§critical-assumptions NNNN` / `NNNN:A3` / `NNNN:ALT5` — each `§`
row is a selector, and ids survive edits where line numbers shift.
`--json --filter metadata,counts` for the status line. Those cost ~20ms; `edges`,
bare `--json` and `lint` pay a corpus scan + repo walk (~1.5s). A stage that must
rewrite the file reads it whole, in one call; if that exceeds one call's output,
chunk by `--select NNNN:§section`, never `sed -n` windows.

Pass `$arg` through as the user typed it: a number with or without leading zeros,
a slug, or a full path all resolve. `--filter path` keeps the answer to a few
hundred bytes instead of the whole envelope.

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
`spikes/`, `tooling-pass/`, `action-items/`, `propose-premortem/` (Stage 2's
hardened-critic output — a sibling, not a Stage-5 lens signal), and the per-cluster
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

## §model-stamp — every lens evidence file records its producing model

Folder existence proves a lens *ran*, not *which model* ran it — and for the
cross-model lenses (`critique`, `repeatability`) that identity is the whole signal.
So every lens element file (`critique.md`, `persona-*.md`, `run-<N>.md`,
`findings.md`, grounding/cove outputs) opens with a one-line stamp of the **actual
base model** (a sub-agent stamps the session's model it inherits):

```
Model: <base-model-id>   (e.g. claude-opus-4-8, kimi-k2.6:cloud)
```

A re-entering session compares this stamp to its own base model: a **different**
model is a legitimate second pass (never "already complete"); the **same** model may
short-circuit; **no stamp** = model-unknown, so don't assume a match — offer the
second-model pass. `/rdr-status` and the Stage 5 gate read this stamp; the
convergence rule that depends on it lives in [`05-prelock.md`](../stages/05-prelock.md)
and the critique lens.

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
  Each delegated read returns a **§return-packet**, not prose.
  **The spawn prompt carries the projector.** A sub-agent loads no SKILL.md and
  no rdr-common, so it reads the record however it can — `sed -n`/`grep`/`awk`
  by line range, every call a turn. Paste into its prompt: the absolute
  `$RDR_HOME/bin/rdr`, `$RDR_PATH`, and the three reads — `inspect NNNN` (id list),
  `inspect --select NNNN:A7 NNNN` / `NNNN:§section`, `--json --filter metadata` —
  with "never `sed`/`grep` the record".
- **Anchor doctrine — ephemeral vs durable.** The sub-agent *return* pointer
  above (`file:line`) is ephemeral: it exists for the main agent to act on this
  turn, and is fine as-is. What gets **written into the RDR body** is durable
  evidence and must use a stable anchor (`path::Symbol`, section heading,
  REQ/assumption/test ID, grepable snippet, artifact path) — never a bare
  `file:line` or peer-RDR `~line N`, unless the line number *is* the behavior
  under test. Stale line numbers on a still-resolving symbol are non-findings,
  not re-anchor work.

## §return-packet — fixed subagent→parent contract

A delegated read/verify sub-agent returns **exactly** this packet — a
machine-checkable schema, not a word-capped summary (a word cap can still omit
the one field the parent needs). Distinct from §next-step, which is the parent's
*human-facing* close packet; this is the subagent→parent handoff.

```text
verdict: PASS | BLOCK | INCOMPLETE | NEEDS_DECISION
blocking: yes | no
evidence_paths: [file:line | spike-cmd→output-path, ...]
changed_paths: [path, ...]        # files the subagent wrote, or []
next_action: <imperative the parent runs, or "none">
summary_50w: <≤50 words; the verdict's reason, not a transcript>
```

The parent **rejects a malformed packet** (missing/extra field, unlisted verdict
value) and asks ONLY for a corrected packet — never a fresh analysis pass. Do
**not** fetch the full subagent transcript unless the packet's evidence_paths
cite a conflict the parent must adjudicate.

```text
verdict: PASS
blocking: no
evidence_paths: [src/codec.go:212, spike:run-mvv→art/mvv.out]
changed_paths: []
next_action: none
summary_50w: All 7 REQ-N have a green test; REQ-MVV round-trips byte-for-byte.
```

```text
verdict: BLOCK
blocking: yes
evidence_paths: [art/deviations.md:14]
changed_paths: [art/deviations.md]
next_action: ask author to resolve SPEC-UNDER on the new accessor surface.
summary_50w: New public accessor not in Normative Contracts; no REQ-N, ships untested.
```

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

Stages resume from durable on-disk state — `status.md` (for Stage 8, the `status.md`
**capsule header** is that cheap one-pass resume read — phase/next/blocker without
re-reading the implementation artifacts), evidence folders, the RDR
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
(Review gate + Next) is no longer acceptable — regression-guarded by
`skills/rdr-doctor/close-summary-check.sh` (golden fixtures, both surfaces; run
`--self-test`). The Review gate is **compressed** to the
`Gate:` line unless a failed gate item is the reason not to continue (then name it). Keep
*both* command spellings in `Next` (the dual-surface convention). `Next:` commits to
**one** action — the status pointer is the only permitted alternative; when two paths
compete, pick one and name the other open obligation in `Continue check:`. For what `RDR
delta` must name per stage, use the table below — it is the single source; SKILLs inherit
it via this anchor and do not restate it.

**What `RDR delta` must name, per stage** (≤1 line each; content/decisions, not engine mechanics):

| Stage | RDR delta must name |
|---|---|
| seed | user outcome; number/slug; why RDR-shaped or demoted |
| propose | chosen approach; rejected alternatives; decisive evidence/criteria; new Pending assumptions; premortem verdict |
| refine | contradictions removed; bloat/history cut; changed contract wording |
| resolve | assumptions verified/downgraded/blocked; evidence basis; recommendation pressure from verification |
| prelock | lens findings accepted/dismissed/deferred; RDR edits; net-new assumptions/failure modes |
| reconcile | terminal disposition of open assumptions/spikes (accepted/dismissed/deferred/downgraded — name them); survivable downgrades; blockers/return stage |
| finalize | READY/NOT READY tied to implementation readiness (not just gate mechanics); lock status; gate-response deltas; reasons impl must wait |
| cluster-reconcile | cross-RDR drift found/fixed; route-back records; impl-ordering consequences |
| implement | implemented contracts; RDR deviations; verification result; spec defects routed back |
| status (read-only) | current state; content reason for it; exact next command |

Per-stage next pointers: seed→propose→refine→resolve→prelock(per lens — one
`/rdr-prelock <lens>` cycle reviews *and* resolves, looping to convergence;
`repeatability` first loops fresh-session run(s)→diff, variant by `Profile` not
files present — `mid`/`large` lite (run-1), `foundational`/escalation full
(run-1/2/3) — then the diff session resolves its `diff.md`)→reconcile→finalize→[cluster-reconcile]→implement.
The branch after `resolve` reads the RDR's **`Profile` field** (the latch Stage 4
writes), not a fresh size inference — resolve it through **§lens-row** below. The
finalize Gate re-validates the field before lock, so a wrong value cannot
silently route past the lenses.

## §lens-row — Profile → the Stage-5 lens row (one authority)

Every "which lens next?" answer — resolve's first pointer, prelock's next
pointer, status's derived position — is this table plus what's on disk. Mirrors
the `$RDR_HOME/stages/README.md` matrix (still the human-facing authority);
inlined here so a skill never reconstructs it from memory. **Read the `Profile`
field, then the row — never default to a lens.** Read it typed —
`metadata[]` where `label=="Profile"`, then `.value`:

```sh
"$RDR_HOME/bin/rdr" inspect --json --filter metadata,counts,outline "$NNNN"
# metadata[] label=="Profile" -> .value ; counts.elements.C ; outline[] canonical=="Normative Contracts"
```

`.value` carries the keyword **plus its rationale tail** (`mid — one contract plus
the metrics surface`), so match the leading word, never the whole string. No
`Profile` entry = the field is absent: that is a §stop-packet, not a default.

| `Profile` | Lens row (in order) | First lens |
| --- | --- | --- |
| `small` | *(none — skip Stage 5)* | → `/rdr-reconcile NNNN` |
| `mid` | grounding → 3amigo | `grounding` |
| `large` | grounding → 3amigo → critique | `grounding` |
| `foundational` | cove → 3amigo → critique → repeatability | **`cove`** |

`foundational` leads with **cove**, never `grounding`: cove subsumes the grounding
sweep as its Step 0. Subsumption runs one way — a standalone `grounding/` from a
pre-escalation `mid`/`large` pass does **not** discharge cove's Step 0.

**Next lens = first row entry with no completed evidence.** Take the row as the
required checklist and subtract only *completed* lens evidence under
`<RDR_EVIDENCE>/<RDR_SLUG>/evidence/` — a bare folder is not completion
(`repeatability` owes its run/diff files; `critique` on `foundational` owes the
dual-model diff, §model-stamp). When the Stage 5 Determinacy trigger fires on a
`mid`/`large` RDR (algorithmic contract — `$RDR_HOME/stages/05-prelock.md`),
**append `repeatability` (lite variant) to the row** — a row entry, not a
footnote, so the first-missing rule sees it and `evidence/repeatability/` is its
completion signal. It is discharged only by those files or a written
`determinacy: n/a — <reason>` disposition. Whole row complete →
`/rdr-reconcile NNNN`.

Read the contracts the trigger judges from `counts.elements.C` plus the
`§normative-contracts` outline entry (`canonical=="Normative Contracts"`, whose
`line_start`/`line_end` bound them). **A zero `C` count does not skip the
trigger** — contracts written as prose are unaddressable and count zero, so on
an older record zero means *unlabelled*, not *absent*: read the section's lines
and judge the trigger on what they say.

**Profile changes are additive, and the row never shrinks.** Recompute from the
*current* field on every close-out, not from the lens just run or a profile
remembered earlier in the session. On a reset/escalation to `foundational`, keep
lower-profile lens folders as done, but the full `cove 3amigo critique
repeatability` row is still owed — and `grounding` is not a member of it, so a
complete `grounding/` subtracts nothing there. Never infer "all lenses done" from
the last lens of an older, smaller profile.

## §mechanical-gate — 30-second template/anchor grep at stage exit

Propose, refine, and resolve close with a grep of the RDR for: a surviving
verbatim template bracket (`[Required`, `[Conditional`, `[Resource]`,
`[Capability]`), `_Draft placeholder._` or a seed-skeleton header in any
section this stage's `Advance when` requires authored, and (propose/resolve)
each NEW `path::Symbol` resolving somewhere in the repo. Any hit → fix now,
or close `Gate: NOT READY (mechanical: <item>)`. This is the Stage-7
tooling-pass CHECK 1/5 subset run where the regression is created instead of
four stages later; the Stage-7 sweep stays the backstop. (Mined: a Required
contract section once survived propose, refine, resolve, and four lenses as
verbatim template text.)

`rdr lint <NNNN>` (`--locking` at lock; exit 1 = BLOCK)
does the structural share — and leaves the receipt §commit demands (`rdr receipt
<NNNN>`: a lint at/after the record's last write, else the commit is refused). It unlabelled contracts, Peer-RDR Evidence naming a
record not an element, unresolvable typed references. `conformance` findings
are advice the rewriting stage applies in-pass (label contracts `C1..Cn`);
`resolution` findings are the fix-now class above. A dangling reference into
a *terminal* peer comes back as a fix pointer with a line range — correcting
that reference text is the one sanctioned amendment to a locked RDR.

```sh
"$RDR_HOME/bin/rdr" lint "$NNNN"
# scope the bracket/placeholder grep: outline[] gives each section its canonical + line_start/line_end
"$RDR_HOME/bin/rdr" inspect --select outline "$NNNN"
# NEW path::Symbol: edges[] kind=="source-anchor" -> resolved
"$RDR_HOME/bin/rdr" inspect --json --filter edges "$NNNN"
```

**Neither lint nor `warnings[]` sees a surviving placeholder** — a record copied
straight from TEMPLATE.md lints PASS with no warnings, by design: the projector
judges structure and what a record emits, never whether a section was authored.
So the grep above stays, and is the only thing that catches this; `outline[]`
only narrows it to the `line_start`..`line_end` of the sections this stage owes.

For anchors, `resolved` is **three-valued** (§source-root): `true`, `false`, or
**absent** — nothing looked. Absent is neither pass nor fail; the gate says the
check did not run rather than closing over it. `rdr index --unresolved` is the
corpus-wide form.

## §amendment-sweep — propagate clause changes at disposition

Amending OR adding a normative predicate/contract clause: (1) grep the draft
for the pre-edit token(s) — every surviving stale site updates in the same
pass or is listed; (2) grep the subject token it defines or redefines;
re-read producers and consumers against the new wording — semantic
disagreement counts, not just staleness; (3) an ADDED clause amends its
contract block — re-read siblings for agreement (values, counts, predicates);
(4) a new order/reachability/emission claim grounds at its authority (call
flow, producing site, output) or registers a Pending assumption (Method:
Spike or MVV Test). (Mined: an amendment missed 1 of 13 sites — two more
reconcile iterations.)

## §auto-fanout — `--auto`: spawn a lens's cross-model passes in parallel

For the cross-model pre-lock lenses (`critique`, `repeatability`), whose passes
have no data dependency on each other and are serial only because a human
relaunches the CLI between them. `--auto` fans them out as concurrent sub-agents,
one per pass, each pinned to a **distinct** model via the spawn's model param
(assign the set up front — with parallel spawns there are no prior files to read
stamps from). Opt-in: without the flag nothing changes.

**Independence is preserved, not waived.** The one-pass-per-session rule exists to
stop a context that authored one pass from anchoring the next; concurrent spawns
satisfy that strictly better, since none exists when the others start. What stays
binding is the **barrier**: the diff/compare pass runs only after every pass lands,
in a further fresh sub-agent that authored none of them (an authoring context
diffs toward the interpretation it already wrote). Fan-out → barrier → diff.

**A spawned pass is a leaf** — fanned-out, barrier diff, or grounding alike. It
reads and grounds inline (its context is the disposable one, which is the point
of the spawn) and spawns nothing: a sub-agent cannot yield to await a nested
completion, so it `sleep`-polls (a turn per poll) and then redoes the work itself.
"Delegate heavy reads" (above) is the parent's rule, and the parent's prompt hands
the leaf the projector.

**Degrade, never fake.** A harness without per-spawn model control or concurrent
spawns runs the documented hand-relaunch path instead — emit the relaunch command
(the `{RDR_RESOURCES}` alt-model roster) and note `auto: unavailable (harness) —
manual relaunch` in the close packet. Never silently run the passes on one model:
same-`model:`-stamp files are not a cross-model pass however they were spawned
(§model-stamp is the check). The win lands only where the harness supports it;
that asymmetry is accepted — never serialize a capable harness for parity.

## §model-ceiling — per-spawn model bump for delegated authoring

For orchestrating skills that spawn *authoring* sub-agents (e.g.
`/rdr-joint-propose`) — a skill running in the main context cannot change its
own model. Resolve the ceiling: `--model-ceiling <model>` arg >
`RDR_MODEL_CEILING` marker var > unset (every spawn inherits the session model —
the default). Per-spawn policy, gated on the member's `Profile` field:
`small`/`mid` → session model; `large`/`foundational` → the ceiling (those
profiles already carry the heavier obligations — scored matrix, hardened
critic — so the stronger model lands where judgment is densest). One
**escalation retry**: a malformed packet or forced re-run may re-spawn once at
the ceiling. Never spawn authoring *below* the session model — judgment-dense
stages don't get cheaper models (mechanical extraction passes may). A stronger
model costs more per token, not more tokens — profile-gating is the efficient
shape; never run a whole cohort at the ceiling "to be safe." A harness without
per-spawn model control runs at session model and notes it in the report.

**Model-adequacy fork (heavy RDRs, before any authoring).** Heavy = `Profile`
`large`/`foundational`, or `Seam Lineage` ≥2 (the accretion floor Stage 2 will
apply). The authoring model is a design input; decide it *before* tokens are
spent — at stage start nothing is written, so cancel is free. Resolve:

- Ceiling set, session model is the ceiling → proceed silently.
- Ceiling set, session is not the ceiling → **pause** (§stop-packet
  `stopped:model-adequacy:…` / AskUserQuestion) and wait: *continue* at the
  session model (record on `Deviations:`), or *cancel* — restart in a ceiling
  session, or `/rdr-joint-propose … --model-ceiling` (it can bump spawns).
- No ceiling: session model confidently the strongest tier its harness offers
  → one note, proceed. Weaker **or unsure** → the same pause, adding "or set
  `RDR_MODEL_CEILING`". Uncertainty asks — never silently author a heavy RDR
  from a model that can't claim top tier. (No ranking table on purpose: names
  churn, harnesses differ; the session judges its own tier, conservatively.)

`small`/`mid` never trigger the fork.

## §strong-consult — a stronger fresh look before the human

At a *challenge* — a route-back reopening the approach, a tiebreaker the
evidence won't collapse, verdict-flapping at the cap — consult ONE
fresh-context sub-agent at the strongest reasoning tier available (the
§model-ceiling resolution; none set → the strongest model this harness
offers, judged conservatively) BEFORE escalating to the human. Factored
brief: the fork/claims in tension + the new evidence, never the justifying
prose or prior verdicts (precedent: a second-model critique pass refuted a
claim the first model had ratified). One consult, queried once — never a
panel. It returns a §return-packet; escalate to the human (§stop-packet)
only on its NEEDS_DECISION or a repeat flap. This re-orders the escalation
ladder, it does not remove the human: genuine either/or design calls still
stop.

## §fork-disposition — an orchestrator schedules a fork, never answers it

For skills that drive other stages (`/rdr-joint-propose`, `/rdr-draft-to-lock`). A
human-judgment fork surfaced by a delegated stage is **scheduled**, not resolved
by the orchestrator — which by construction has read none of the evidence behind
it. Two rules decide *when* to interrupt:

- **Ask now** when the fork blocks what comes next — advancing past it wastes
  the work after it.
- **Park** anything that does not block; batch every parked fork into **one**
  `AskUserQuestion` when the run stops.

Batching is for *economy of interruption*, never for reaching a terminal state:
a fork dropped to finish a run is the failure mode these skills must not have.
Each caller names its own ask-now triggers and dispositions; this anchor owns
only the schedule-don't-answer rule and the ask-now/park split.

## §punt-ledger — record the escape at the moment of the route-back

Any route-back that reopens a completed stage (a Stage-4/6 refutation
returning to propose/refine, a prelock refutation returning to Stage 4, a
7.1 demotion) appends ONE row to the consumer's escaped-defect ledger —
`$RDR_RECORDS/<slug>-postmortem.md` (start it from post-mortem/TEMPLATE.md
with just the ledger if absent): the defect one-line, the stage that caught
it, Expected-catching stage (the ledger's closed vocabulary), escape
distance; odc columns n/a until implementation triage. Append BEFORE any
refine collapses the history — refine strips change-history from the RDR
body by charter, so the ledger is the punt's only durable home, and the
propose premortem's seeds read it: an unrecorded punt is a premortem that
runs blind next time.

## §commit — optionally commit this run's *own* files, fast, no exploration

A writing stage already knows the exact files it wrote — `$RDR_PATH`,
`$RDR_RECORDS/README.md`, and its `$RDR_EVIDENCE/$RDR_SLUG/evidence/<subdir>` — bound by
§seam-bind + §rdr-resolve. So it can commit *those paths and nothing else* without ever
running `git status` / `git add -A` / inspecting "what's dirty". A record path is
committed only with a lint receipt (`rdr receipt`; refused as `stopped:commit-unlinted`
— run `rdr lint NNNN` after the last write, then retry). This is the whole point:
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

**Run the block below verbatim** (same doctrine as §seam-bind — do not paraphrase or
abbreviate; weaker models must run it literally). The gate is inline; `rdr_commit`
itself lives in **`$RDR_HOME/skills/rdr-commit.sh`** (symlinked beside each SKILL.md as
`rdr-commit.sh`) and is sourced **only when the gate passes** — off means the file is
never read. Call `rdr_commit` once per logical commit, subject then owned absolute paths:

```sh
# §commit — gate, then source the helper. Paths are ABSOLUTE; all in one repo per call.
# Preconditions: §seam-bind ran (RDR_AUTOCOMMIT + paths bound).

rdr_autocommit_on() {                                  # reads the gate; "$@" = the skill's own args
  case " $* " in *" --no-commit "*) return 1;; *" --commit "*) return 0;; esac
  [ "$RDR_AUTOCOMMIT" = "true" ]                        # marker var; unset/anything-else = off
}

# Source the helper only behind the gate, then commit. `$RDR_HOME` is exported by §seam-bind.
# Pick the file FIRST and test it — a failed `.` aborts the shell before any `||` runs,
# which would swallow the diagnostic below.
rdr_autocommit_on "$@" || exit 0
H="$RDR_HOME/skills/rdr-commit.sh"
[ -f "$H" ] || { echo "stopped:commit-helper-missing:$H (re-run /rdr-init or reinstall the engine)" >&2; exit 1; }
. "$H"
rdr_commit "docs(rdr): … cli/$NNNN — <summary>" "$RDR_PATH" "$RDR_RECORDS/README.md"
```

**Per-stage owned path-sets and subjects** — the row for each stage (doc commit,
evidence commit, and `repeatability`'s deferred-doc exception) lives in
**`$RDR_HOME/skills/rdr-commit-map.md`** (symlinked beside each SKILL.md as
`rdr-commit-map.md`), read behind the same gate as the helper. Two commits, never
one: the doc/README commit is the **design history** (`docs(rdr):`, real subject —
*never* `fixup!`); the evidence subtree is a separate `chore(rdr):` commit so the
`docs(rdr)` log stays readable.

## Brevity & doctrine

Be brief without being lossy (the flow's standing doctrine —
`$RDR_HOME/stages/README.md` *Doctrine*). Spend tokens on load-bearing or
complex design; terse everywhere else. Delegate heavy reads (corpus, source,
several round-output files) to a sub-agent that returns verdict + evidence
pointer — the read-heavy stages (4, 6, 7) and `/rdr-status` rely on this. Defer
to CLAUDE.md and the README Doctrine on any conflict.
