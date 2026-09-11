# rdr-common — shared bones for the `/rdr-*` skills

Not a skill. Shared preamble the per-stage RDR skills cite by `§<anchor>` so the
seam-bind, number→path resolve, and evidence-glob logic live in one place. Every
`/rdr-*` SKILL.md runs §seam-bind and (except `/rdr-seed`) §rdr-resolve first,
then runs its stage prompt, then (when autocommit is on) runs §commit on its own
files, then prints §next-step.

These skills **stay on the current branch** — never `git branch`/`switch -c`/
`checkout -b`, never a worktree (the "branch before feature work" reflex is wrong
here); §commit owns why. They also **do not edit the consumer project's source** —
they author/inspect RDR documents and write flow evidence. The only writes are to
the RDR file, the evidence dirs, and (Stage 8) the artifact dir, all named via the
seam. `/rdr-status` writes **nothing**.

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
seam block below; **§rdr-resolve still runs** (it is the record lookup, not the seam). The block carries `RDR_AUTOCOMMIT` precisely so the fast path
doesn't blind §commit's gate — if the block omits it, treat `RDR_AUTOCOMMIT` as unset
(autocommit off). Harnesses and sessions without that block (other agents,
headless runs) run the resolver as written — same bindings either way.

**Run the block below verbatim — do not abbreviate or paraphrase, and never
source a marker file directly.** `rdr env` is the resolver, keying off git
topology, so it binds the same seam from a consumer cwd, a worktree, or the flow
repo. **Nearest marker wins**: a repo-local `$PROJECT/.rdr/workspace` (this
repo's own RDR env, inside its gitignored `.rdr/` — the default) beats the shared
`$WS/.rdr-workspace` (a workspace seam siblings opt into via `--workspace`) —
like `.git` or `.editorconfig`, the closest governs. `$PROJECT` is `dirname` of
the git-common-dir, so a worktree resolves its main repo's local marker; a
repo-local marker's records bind to the worktree toplevel.
`$RDR_MARKER` records which one won.

```sh
# §seam-bind — copy/run verbatim. Binds every seam var the marker exports.
GC=$(git rev-parse --git-common-dir 2>/dev/null) || { echo "stopped:not-in-a-project (run /rdr-* from inside the consumer repo)" >&2; exit 1; }
PROJECT=$(dirname "$(cd "$GC" && pwd -P)"); WS=$(dirname "$PROJECT"); export PROJECT WS
eval "$("$RDR_HOME/bin/rdr" env)" || exit 1        # stops itself: no-marker / not-in-a-project
[ "$RDR_PROJECT" = "$PROJECT" ] || { echo "stopped:foreign-seam:$RDR_MARKER (binds $RDR_PROJECT, not $PROJECT)" >&2; exit 1; }
```

**Two properties of that block are load-bearing — never trade them away.**
`rdr env` answers from the **marker**, ignoring an inherited `RDR_*`, so `eval`
*overwrites* a stale var instead of re-exporting it (sourcing had this for
free). A `--records`/`--repo` flag keeps the opposite rule — one dir named for
one call is a decision. And the guard catches a **foreign seam**: siblings under
one workspace can bind *disjoint* seams, and records read from one project while
`$RDR_EVIDENCE` resolves from another make every lens read as never-run — a
finished record looks unlensed and the flow re-runs lenses over the evidence
already on disk. Silent without the guard.

**Shell state dies between Bash tool calls**, so anything you still need from the
marker must be bound in the *same* call that uses it. Bind once per call, at the
top — never carry an `export RDR_…=…` prefix from one call to the next, and never
re-run this block just to reach `rdr`.

**`rdr` itself never needs the block** — it finds the marker on its own, so a
bare `"$RDR_HOME/bin/rdr" inspect …` works with no seam bound. Run §seam-bind
when the *shell* needs the vars: `$RDR_ENV`, `$RDR_RESOURCES`, `$RDR_AUTOCOMMIT`
(the three the projector never opens), or a path for a later command.

**What bootstraps what.** `$RDR_HOME` must be bound *before* this block — it is
how the block reaches the binary — by the harness (plugin root, or the fast-path
block above) or an earlier resolve. Everything else comes from the marker, which
the binary finds itself: no cycle, one precondition. `/rdr-init` *creates* the
marker so it cannot call this (its inverse-seam-bind step), and `/rdr-doctor`
resolves independently on purpose, so a broken seam cannot hide from the check
looking for it.

The marker exports the **five-var engine contract** every skill may read — all
five required, and `/rdr-doctor` check 3 fails without them:
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

Three more are optional and bound the same way: `$RDR_SOURCE_REPO` (below),
`$RDR_AUTOCOMMIT` (§commit's gate) and `$RDR_USAGE_LOG`. `rdr env` prints
whichever of the eight the marker set, plus `$RDR_MARKER` and `$RDR_PROJECT`.

`--repo` defaults to it, so no call spells it out. Unset, or an anchor naming a
foreign codebase (a third-party library cited for contrast), leaves those edges
unresolved — say so.
**A wrong root is worse than none**: grepping a tree the symbol was never in
returns `resolved:false`, a false finding a consumer chases, where omitting
`--repo` returns the key **absent**, which honestly says nothing looked.

## §intrastate — the routing binary, required

`intrastate` resolves the routing models under `$RDR_HOME/models/` (§lens-row,
`/rdr-status`). It is a **dependency, not an accelerator**: those models are the
authority for which stage and which lens come next, and `intrastate lint` is
what *proves* every routing cell is claimed exactly once. A skill that cannot
reach it has no routing answer — so it **stops** rather than inferring one.

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"   # marker var, else PATH
[ -x "$IS" ] || { echo "stopped:no-intrastate — run /rdr-init to install it" >&2; exit 1; }
"$RDR_HOME/bin/rdr" status --tags NNNN >/dev/null || exit 2   # test the read before substituting it
```

Resolution order is `$RDR_INTRASTATE` (marker var, for a binary off PATH), else
PATH. `/rdr-init` installs it and `/rdr-doctor` check 12 FAILs without it.

The third line guards every `$("$RDR_HOME/bin/rdr" status --tags …)` below: a
refused read lands its `stopped:…` line in intrastate's argv, which answers
`unknown command "stopped:…"` and the real refusal is gone. The substitution
stays inline (zsh does not word-split a captured `$T`), so the read is tested
first, once per run.

## §rdr-resolve — RDR number → file path

Every stage skill except `/rdr-seed` takes a 4-digit number `NNNN`. Resolve it
against the **canonical RDR directory** — `$RDR_RECORDS`, bound by §seam-bind (this
consumer's RDR-instances dir; the parent of `{ARTIFACT_DIR}`). Look in **only** that
dir so the many decoy `NNNN-*` entries under `$RDR_EVIDENCE` (per-lens, tooling-pass,
spikes — e.g. a real `evidence/tooling-pass/0039-*.md` is **not** an RDR) can never
be picked:

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
with line ranges, then every element, labels capped at 100 runes — the whole
read plan in one call (~100-200 lines, under 20KB on the largest records; never
`| head`, which drops element rows silently; not `--filter outline`, ~800 lines,
nor `--select elements`, 25× larger). Then read by id:
`--select NNNN:§critical-assumptions NNNN` / `NNNN:A3` / `NNNN:ALT5` / a contract
clause `NNNN:S-1`, `NNNN:L-3` (the label as written inside the fence; never pull
the whole `§normative-contracts`, tens of KB, to reach one clause) — each `§` row
is a selector, and ids survive edits where line numbers shift. In JSON
(`--json --filter elements`, or `--select` with `--json`) an element's Status /
Method / Evidence are `fields[]{label,value}` rows, not top-level keys. WHICH
assumption is a fact, not a walk: `status --tags NNNN` carries `ca_pending_ids`,
`ca_off_vocabulary_ids` and `reverify` (the qualifier's re-verify ids; absent off a
re-entry); the anchor tallies are on demand (§mechanical-gate); and `--json --filter assumptions` is the per-assumption `status.value` /
`method.{members,off_vocabulary}` / `evidence.anchors` view, a tenth of `elements`.
`--json --filter metadata,counts` for the Status/qualifier text — its `status`
object; that is not a fact name (`reverify` alone carries the re-verify ids).
Those cost ~20ms; `edges`,
bare `--json` and `lint` (text-only — it has no `--json`) pay a corpus scan +
repo walk (~1.5s). A big section or corpus-scanning facet is **saved once to a
scratch file and sliced there — in the main context exactly as in a spawn**;
the ban is on the record file, not the scratch copy. A stage that must
rewrite the file reads it whole in one sanctioned call —
`--select NNNN:§title NNNN > /tmp/<slug>.md` (the H1 span), sliced from the
scratch — never `sed -n` windows on the record. Body edits replace exact bytes
read from the file **in the same turn** (an exact-replace with a
one-occurrence check); retyping from an earlier projection is where edits
fail. Locating an edit anchor with `grep -n` on the file for a phrase just
projected is sanctioned; window-reading content that way is not. And never
silence an `rdr` call's stderr — a `stopped:` line is the answer, not noise. A corpus question
("who cites this", "what is Draft", "what blocks") goes through
`index --status` / `--backlinks=NNNN[:elem]` / `--cycles` — a few lines each —
never bare `index --json` (the whole graph, MB, and nothing to grep it for).

Pass `$arg` through as the user typed it: a number with or without leading zeros,
a slug, or a full path all resolve. `--filter path` keeps the answer to a few
hundred bytes instead of the whole envelope.

`/rdr-seed` does **not** resolve — it *allocates* the next number (see §rdr-claim).

## §rdr-write — the structural edits, resolved not described

The structural edits (claim a number, add/flip the README row, lock, route
back, size the Profile) and a blocker's return stage are **resolved from a
linted decision table**, not restated per site.
`intrastate lint --model` proves every cell of `models/rdr-write.toml` is claimed
by exactly one row, so an unhandled case is a lint failure, not a wrong answer.

`lock` and `readme` are **applied for you**; the other six emit an edit you
apply. The split is permanent — see the end of this section.

```sh
# §rdr-write. $RDR_HOME/$RDR_PATH/$RDR_RECORDS from §seam-bind.
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"
M="$RDR_HOME/models/rdr-write.toml"
PATH="$RDR_HOME/bin:$PATH"; export PATH   # the model's accessors exec `rdr` by name
BIND=(--artifact record="$RDR_PATH" --artifact readme="$RDR_RECORDS/README.md"
      --tag nnnn=NNNN --allow-commands)
TAGS=$("$RDR_HOME/bin/rdr" status --tags NNNN --except status,readme_status) || exit 2

# lock | readme — resolved AND applied, nothing retyped:
"$IS" flow resolve --model "$M" "${BIND[@]}" --as json --outcome <lock|readme> \
  [--tag k=v …] $TAGS | "$IS" flow set-state --model "$M" "${BIND[@]}" --as json --plan -

# claim | demote | profile | return | fence | ground — you apply the emit:
"$IS" flow resolve --model "$M" "${BIND[@]}" --plan-only \
  --outcome <claim|demote|profile|return|fence|ground> [--tag k=v …] $TAGS
```

`--except status,readme_status` is required: the table OWNS those two, and
intrastate reads owned state through the model's own accessors, refusing it as
argv (`flow-tag-owned`). Those accessors are `rdr` itself — hence
`--allow-commands` and the binds — so the tool that renders the facts reads them
back after a write.

**Hence the `PATH` line**, which is not decoration. A declared `command`
accessor is `exec`d, not run through a shell: argv[0] resolves against `PATH`
with no expansion (`$RDR_HOME/bin/rdr` execs that literal, relative to the
model's own dir) and no placeholder to reach the seam (`{…}` is a closed
vocabulary — `{artifact}`, `{tag.*}`). A portable model can therefore only
spell argv[0] bare, so the *caller* supplies the path. Without the line the
accessor dies `flow-accessor-failed: exec: "rdr": executable file not found`,
mid-write and after the input checks passed.

`emit` is the answer for the second form (`--plan-only` drops the fact echo): `op`,
`target`, `edit` (the exact expression), `why`, `surface` (show verbatim).
**Apply `edit` as handed** — it is data, not a description; retyping it makes
the guarantee prose again.

**Branch on the verb that answered.** On `resolve`, read `dispositions.op`,
never the `op` string: RDR 0024 partitions the domain into `edit` (apply it),
`none` (the state already holds — every op is idempotent) and `stop` (surface
per §stop-packet). Matching a `stopped:` prefix re-derives in prose what the
loader proved, and a token added later would not reach it. On `set-state`, read
`writers`/`writes` **emptiness** — it carries no `dispositions`, deliberately,
since that block is joined from the selected row and `set-state` selects none.
Applied names its writer and keys; a no-op is exit 0 with both empty. A refusal
never arrives as a write: refusing rows declare `advance = false`, so their plan
carries none.

Six tags are the caller's, not facts, and only the rows that read them demand
them: `profile` takes `floor` (§lens-row's `emit.floor`, passed through as a
value), `user_facing=<yes|no|unknown>` and `locks=<none|contract|format|cross-rdr|unknown>`
— Resolve's two judgements; `demote` and `return` take
`blocker_class=<approach|contradiction|contract|proportionality|assumption-gap|assumption-disturbed|spike|determinacy|wording|none>`;
`ground` takes `searched=<none|code|cluster|corpus|rfd>` and `found=<true|false>`.
`unknown` and `none` are declared members that stop by name — never default them.

The table routes structure and status, never judgement: the gate verdict, the
demotion call and the Profile's two dispositions arrive as your `--outcome` and
tags. `rdr` stays read-only — it renders the facts and reads them back; it never
writes. A write re-arms the lint receipt (§commit).

**Why only those two.** An `edit` writer substitutes a planned VALUE into an
anchored line. `profile`, `demote` and `readme --add` interpolate author prose
no fact supplies (`<one clause naming the contract>`, `<one-line reason>`, a
title); `claim` allocates with no planned value; `fence`/`return`/`ground` never
edit. Converting them would mean typing that prose as a tag for the table to
interpolate — the transcription relocated, not removed.

`lock` has one author step BEFORE the pipe: `sections` moves the four judged
gate responses out to `gate.md`. Do it first — the status flip is the declared
write, so a failure mid-move leaves a legible Draft, not a Final whose responses
never moved.

## §rdr-claim — atomically reserve a number before authoring (`/rdr-seed` only)

`max(NNNN)+1` alone races: two concurrent sessions both read the same max, both
spend a minute authoring, and both write distinct `NNNN-*slug*.md` files — two
RDRs silently sharing one number (no write error, because the slugs differ). The
cure is to **claim the number atomically as the first step**, before any
authoring, and **fail loudly on collision** so the loser just bumps and retries.

Run §rdr-write with `--outcome claim` and apply its `edit`. It reserves
`NNNN-RESERVED.md` under `set -C` (noclobber) as a **verbatim copy of
TEMPLATE.md**, retrying on collision, and echoes the claimed `NNNN`. The
filename is keyed on the number **only** — no slug, no session suffix — so the
number is the lock and a concurrent session's `max()` scan counts the
reservation and picks NNNN+1.

The reserved file **is** the canonical skeleton, so there is never a reason to
author structure from scratch or copy a neighbour RDR for "house style" (that
drifts the template and leaks the neighbour's solution into a Draft that must
have none). Fill it IN PLACE — only the H1 `[NUMBER]`/`[TITLE]`, Metadata,
Problem Statement and Context; every other section stays exactly as the template
ships it, because those placeholders ARE the Draft placeholders. Then rename to
the final slug: `git mv "$RDR_RECORDS/NNNN-RESERVED.md" "$RDR_RECORDS/NNNN-<slug>.md"`.

If authoring is abandoned before the rename, the stray `NNNN-RESERVED.md` is the
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
hardened-critic output — a sibling, not a Stage-5 lens signal) and the file
`rulings.md` (§run-prompt). Cluster reconcile
output is the one thing NOT under the record: it is keyed by the cluster, at
`<RDR_EVIDENCE>/cluster-reconcile/<key>/`, because the report is about the set,
not about any one member. The key is the members' record numbers joined and
sorted (`0117-0118`, `0122-0123-0130-0131-0132`) — so the key IS the membership,
which is what lets `rdr status` answer 7.1 exactly instead of searching. Name a
new run's directory that way; a topical key (`dml-purpose`) is a pre-2026-06-29
shape that no longer reads.
**Ask for the dir; never compose one** — `rdr paths` answers from the same
declarations `rdr status`'s probes read, so what you write to and what the
navigator checks cannot drift apart:

```sh
eval "$("$RDR_HOME/bin/rdr" paths --lens <lens> --next-iter <NNNN>)"
mkdir -p "$ITER_DIR"   # the projector names, it never creates
# EVIDENCE_DIR the lens dir · ITER/ITER_DIR where this pass writes
# ITER_FOUND/ITER_NOTE what was on disk (a gap is named, not hidden)
```

`--cluster <key>` is 7.1's tree, `--tree spikes` is `{SPIKE_DIR}`; `--json` for
structure, omit `--next-iter` for the base alone. `--next-iter` LISTS the dir:
loose files are iteration 1, so a first pass gets the base and a re-entry
`iter-N/`. Unbound `$RDR_EVIDENCE` → exit 1 and a stated absence, never a
fabricated path; it creates nothing — hence the `mkdir`. **Eval once per pass**
and reuse `$ITER_DIR`: after the `mkdir`, a second `--next-iter` names the
NEXT iteration, splitting one pass across two dirs.

The existence of the lens dir is the disk signal that that lens ran — this is
what `/rdr-status` reads. **Not every stage leaves a disk signal:** Stage 4
Resolve writes an evidence folder only when it names spikes; a pure
source-search resolve records its verdicts inline in the RDR (CAs flip to
`Verified`) and creates no `<RDR_SLUG>/` dir. So a missing per-RDR folder means
"no lens/spike artifact yet," **not** "Resolve hasn't run" — the CA verdicts in
the RDR body are the authority for Resolve-done. A consumer either points
`$RDR_EVIDENCE` at its own dir/repo or leaves it defaulting to `$RDR_RECORDS`
(evidence beside artifacts under one `<RDR_SLUG>/`); `rdr paths` answers under
both without being told which.

## §model-stamp — every lens evidence file records its producing model

Folder existence proves a lens *ran*, not *which model* ran it — and for the
cross-model lenses (`critique`, `repeatability`) that identity is the whole signal.
So every lens element file (`critique.md`, `persona-*.md`, `run-<N>.md`,
`findings.md`, grounding/cove outputs) opens with a one-line stamp of the **actual
base model** (a sub-agent stamps the session's model it inherits):

```
Model: <base-model-id>   (e.g. claude-opus-4-8, kimi-k2.6:cloud)
```

A **different** model is a legitimate second pass (never "already complete"); the
**same** model is the recorded single-model fallback; **no stamp** = model-unknown,
so a match is never assumed. §lens-row's completion outcomes read these stamps and
apply that rule, so nothing compares them by hand; the convergence rule lives in
[`05-prelock.md`](../stages/05-prelock.md) and the critique lens.

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

**Author rulings live in `rulings.md`** at the record's evidence base
(`eval "$("$RDR_HOME/bin/rdr" paths NNNN)"` → `$RDR_ROOT_EVIDENCE/rulings.md`), one
`- **<Qn|fixture|fork>** — RULED: <verbatim>` line each under a `## <date> — <asking
stage>` heading. A stage reads it before the record, applies every line with no
`absorbed @` tail, and appends ` — absorbed @<stage> <date>` to each it applied.
`rulings_open` (rdr-facts.toml) counts the rest; the lock fence refuses on it. A
ruling written anywhere else is read by nothing (cli/0113 locked over one in
run-plan.md).

## §delegation — who reads, who writes, what spawns

- **Spawn sub-agents with the harness's built-in spawn tool** (`Agent` in Claude
  Code, a.k.a. `Task`). **Not** a task-list/todo tracker — that tracks items, it
  spawns nothing. Naming it here saves the tool-lookup round-trip every delegating stage otherwise burns.
- **Delegate heavy *reads*, author *writes* in the main context.** The read-heavy
  stages (4, 6, 7) and `/rdr-status` push corpus searches, source spelunks, and
  reading several round-output files to a spawned sub-agent that returns *verdict +
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
  `$RDR_HOME/bin/rdr`, `$RDR_PATH`, and the three reads — `inspect NNNN` (id list,
  under 20KB, never `| head`), `inspect --select NNNN:A7 NNNN` / `NNNN:§section`,
  `--json --filter metadata` — with "never `sed`/`grep` the record". The spawn
  prompt also carries the shell rule: output separators are `---`, never `===` — zsh aborts an unquoted
  `=`-leading word (`=== not found`), poisoning the turn and skipping every
  chained call. Paste these read rules too, each a turn saved: **batch
  `--select`s by summed line range** — the ranges in `inspect NNNN` add up, ~250
  lines is ~25 KB, one call; past that split, since an overflow past the ~30 KB
  cap costs a file + read-back round-trip. **Read an element once and keep it**
  — never re-select the same id with a different grep. **A big section or a
  corpus-scanning facet (`edges`, `index --backlinks`) is saved once** to a
  scratch file (`> /tmp/…`) and sliced there — `sed`/`grep` on the scratch
  copy is fine; the ban is on the record file. **A grounding or cove spawn's
  FIRST call** is `inspect --json --filter edges,elements NNNN`, kept to the
  edges whose `resolved` is false or absent — that list is its worklist, not a
  hand sweep of every anchor. **An outsized single element is never pulled
  whole** (a contracts section runs tens of KB) — the summary row's size
  column says what a select weighs; check it first and split. A spawn working lint findings gets the captured
  `$ITER_DIR/lint.txt` path, not a fresh lint; text it drafts for the record
  cites colon ids (`NNNN:ID`), never `NNNN ID`. Delegate the *other*-repo
  reads — the host record's own clauses stay with the authoring parent.
  And a sub-agent that owes an evidence file (a
  lens contract's `findings.md`, a persona file) writes it with a Bash heredoc,
  never the Write tool — some harnesses refuse sub-agent Writes and return
  only text.
- **After spawning, the parent waits by ending its turn.** It does only items its
  own list still owes and nothing the brief names — a delegated check is never
  re-run in the parent, and a useful-looking side read on the delegated question
  is a re-run too: the round ledgers, the peer-clause reads, and any question
  handed to a consult leave the parent's list the moment the spawn starts.
  When nothing is owed, end the turn with no tool call: the
  completion notification is the wait (no sleep, no poll, no agent-list check,
  no filler echo, no dir-watch). Act on the packet; doubting one is a second
  consult, never a parent re-read. Open the report file only for a finding the
  packet names as needing the parent's judgment. **A long-running command is
  waited on the same way** — background it and let the notification arrive;
  polling a log is the same wasted turn as polling an agent.
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
appendix: <optional, last; verbatim quotes / query log the parent files as evidence>
```

On `NEEDS_DECISION`, `next_action` is the **question to put to the human**,
never an action — a runnable repair beside that verdict is a contradiction the
parent rejects like any malformed packet (the decision is the human's; §strong-consult).

`appendix:` is the one optional field, only when the brief asks for it, and it is
never the parent's read — the six fields above decide; the appendix is copied into
the evidence file. **A packet is ≤1 K chars**; grounding, quotes and diffs go to
a file the packet cites (five grounder packets once carried 96 K chars into a
parent that holds "paths and one-line summaries"). The parent **rejects a malformed packet** (missing/extra field,
unlisted verdict value) and asks ONLY for a corrected packet — never a fresh
analysis pass. Do
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
not to remove the human. A design-call packet carries `searched=` (§ground-before-ask).

## §ground-before-ask — search before the human

Before any open author question or design-call §stop-packet (propose's
joint-decision/bridge-choice stops, resolve's author's round, prelock-resolve's
tiebreakers, reconcile's accept/defer), resolve `--outcome ground` (§rdr-write's
one call) with `--tag searched=<none|code|cluster|corpus|rfd> --tag found=<true|false>`.
`ground` names the next source to search and `surface` its command; run it,
re-call with that rung, until `apply` (recommend with the cite; no question) or
`ask` (the packet carries `searched=…; found: none`). Why: 74 of 75 open author
questions since 2026-05 were already settled in one of these sources.

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
`/rdr-prelock <lens>` cycle reviews *and* resolves, looping to convergence; the
`lens` outcome orders them, `--outcome repeatability` orders its
runs)→reconcile→finalize→[cluster-reconcile]→implement.
The branch after `resolve` reads the RDR's **`Profile` field** (the latch Stage 4
writes), not a fresh size inference — resolve it through **§lens-row** below. The
finalize Gate re-validates the field before lock, so a wrong value cannot
silently route past the lenses.

## §lens-row — Profile → the Stage-5 lens row (one authority)

Every "which lens next?" answer — resolve's first pointer, prelock's next
pointer, status's derived position — **is resolved by the routing model, not by
reading this section**. `$RDR_HOME/models/rdr-status.toml` is the authority:
`intrastate lint` *proves* every profile × lens-flag cell is claimed by exactly
one rule, which is a guarantee prose cannot give.

```sh
IS="${RDR_INTRASTATE:-$(command -v intrastate)}"   # marker var, else PATH
[ -x "$IS" ] || { echo "stopped:no-intrastate — run /rdr-init to install it" >&2; exit 1; }
"$RDR_HOME/bin/rdr" status --tags "$NNNN" >/dev/null || exit 2   # §intrastate
"$IS" flow resolve --model "$RDR_HOME/models/rdr-status.toml" --plan-only \
  --outcome lens $("$RDR_HOME/bin/rdr" status --tags "$NNNN")
"$IS" flow resolve --model "$RDR_HOME/models/rdr-status.toml" --plan-only \
  --outcome floor $("$RDR_HOME/bin/rdr" status --tags "$NNNN")   # the accretion floor on Profile
```

(`--plan-only` drops the ~50-line `observed.*` echo of the tags — the one
idiom for that, here as in §rdr-write; no grep filter.)

**`--outcome floor` is the accretion floor**, resolved from the Seam Lineage
facts (`seam_lineage`, `accretion_disposition`) by the model's `floor` group —
the rule lives there, not here. `emit.floor` is `raise` (Profile sits one
tier above its contract axis; rdr-write's `profile` rows apply it) or `none`.
`stopped:profile-below-floor` means the recorded field is below every raised
tier and outranks the lens answer; `stopped:seam-lineage-count-unread` names
a field whose count must be written in TEMPLATE.md's form before the floor
can be resolved.

Substitute `--tags` inline as written — never capture it into a variable
first: zsh does not word-split an unquoted `$TAGS`, so intrastate receives one
newline-joined argument and refuses it.

The guard is not optional. Unresolved, `$IS` is EMPTY, and an unguarded
`"$IS" …` exits **0** with a bare `permission denied:` — a stage would read a
no-answer as a clean run, which is precisely the failure the paragraph below
names. Test the binary, never the exit code.

It answers with the next command and its reason, and returns
`stopped:no-profile` when the `Profile` field is absent — a §stop-packet, never
a default. On a re-entered Draft it also names a row lens whose evidence
predates the qualifier's demote date (`lens_stale`), since a folder its Final
earned is not a lens run over the rework. The row, the first-missing rule, the
additive-on-escalation rule and the remaining span are encoded there: `emit.next`
is the next lens, `emit.row` the row still owed **headed by it**, in order
(`none` = complete) — as handed, never with the head stripped. Do not restate or
walk them here.

`intrastate` is **required** (§intrastate). Unresolved, this is
`stopped:no-intrastate` — never a hand-walked row: an inferred lens that reads
like a resolved one is the failure this flow guards against everywhere else.

**`--outcome lens` says a lens RAN; two more say whether it FINISHED** — a folder
cannot, and for the cross-model lenses the model that wrote each pass is the whole
signal (§model-stamp). Same call, different outcome: **`critique`** compares the two
passes' stamps, **`repeatability`** reads `run-1.md`'s `variant:` header (the durable
record, never the file count). Each answers `none` when finished or names the lens
again, with the caveat on `surface` — single-model fallback, unstamped pass, variant
mismatch. Ask at a lens close-out and at Stage 5's preflight; their answer outranks
the row's folder-level one.

**The Determinacy trigger is a written line, not a re-read.** Stage 5 judges it
once (`$RDR_HOME/prompts/pre-lock/3-repeatability.md` §Determinacy trigger) and
writes `Determinacy: fired — …` or `Determinacy: n/a — …` in Normative Contracts;
`--outcome repeatability` chains to the `determinacy` group on that line
(`resolve:determinacy`, followed once), which routes run 1, `none`, or
`stopped:determinacy-trigger-unjudged` while the line is unwritten. A zero
`contracts` count says nothing here — the line does.

## §mechanical-gate — 30-second template/anchor grep at stage exit

Cited by the 02/03/04 stage prompts, not by any SKILL.md — a SKILL-only reference
count reads this as dead; it is not.

Propose, refine, and resolve close by running `rdr lint`, then judging its
`placeholder:survived` findings against what this stage's `Advance when`
requires authored. Any hit in a section this stage owed → fix now, or close
`Gate: NOT READY (mechanical: <item>)`. This is the Stage-7 tooling-pass
CHECK 1/5 subset run where the regression is created instead of four stages
later; the Stage-7 sweep stays the backstop. (Mined: a Required contract
section once survived propose, refine, resolve, and four lenses as verbatim
template text.)

```sh
"$RDR_HOME/bin/rdr" lint "$NNNN"                            # placeholder:survived + the structural share; header: blocking=N resolution=N placeholder=N advisory=N
"$RDR_HOME/bin/rdr" status --tags --filter anchors_total,anchors_unresolved,anchors_unlooked,peer_evidence_unresolved "$NNNN"   # the anchor counts (on demand: only when named); --filter edges only to read WHICH
```

**The tool reports the text; the stage judges the obligation.** A finding says
TEMPLATE.md's own words survived at that range — it never says the section is
unauthored, because a record can carry a real contract with the guidance block
still above it. Which sections this stage owed authored is the stage's call,
and the two messages separate the cases: a block over an *unfilled skeleton*
is always a fix, a block over authored content is a deletion.

`rdr lint` also does the structural share: unlabelled contracts, Peer-RDR
Evidence naming a record not an element, unresolvable typed references — and
leaves the receipt §commit demands (`rdr receipt <NNNN>`: a lint at/after the
record's last write, else the commit is refused). `conformance` findings are
advice the rewriting stage applies in-pass (label contracts `C1..Cn`);
`resolution` findings are the fix-now class, and the only ones that block
(`--locking`, live record, exit 1). A dangling reference into a *terminal* peer
comes back as a fix pointer with a line range — correcting that reference text
is the one sanctioned amendment to a locked RDR.

For anchors, `resolved` is **three-valued** (§source-root): `true`, `false`, or
**absent** — nothing looked. Absent is neither pass nor fail; the gate says the
check did not run rather than closing over it. `rdr index --unresolved` is the
corpus-wide form.

**Cadence: twice per stage, not per edit.** A baseline lint before the first
edit (it surfaces the findings that pre-date this stage — no stash-and-compare
to tell old from new), and one after the last write (that is the receipt). The
only reason to lint mid-pass is a `resolution` finding this stage just created.
Each run is a corpus scan + repo walk, so the run IS the capture:
`lint <NNNN> | tee "${ITER_DIR:-/tmp}/lint.txt"` — every follow-up question
(tier counts, per-code greps, one finding's text) reads the saved copy, never
a re-invocation.

## §amendment-sweep — propagate clause changes at disposition

Cited by the 04/05/06 stage prompts, not by any SKILL.md (as §mechanical-gate).

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

**Model-adequacy fork (heavy RDRs, before any authoring).** Heavy = `profile`
`large`/`foundational`, or `seam_lineage=2+` in `rdr status --tags` (the `floor`
row Stage 2 applies). The authoring model is a design input; decide it *before* tokens are
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
evidence won't collapse, verdict-flapping at the cap, or a `return` row that
emits `consult: strong` (a route-back to Stage 2 or 3 over the approach or a
contradiction) — consult ONE fresh-context sub-agent at the strongest
reasoning tier available (the §model-ceiling resolution; none set → the strongest model this harness
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
— run `rdr lint NNNN` after the last write, then retry), and signed per the TARGET
repo's `commit.gpgsign` (a signing failure is `stopped:commit-sign-failed`: nothing moves,
the paths stay written — fix the signer, never retry unsigned). This is the whole point:
**no reconnaissance, no round-trip, and no confusion about what this session owns** —
the owned set is a property of the stage, not a discovery. It also makes parallel
`/rdr-*` runs safe **without a worktree**: each run commits through its *own* private
index and advances **the current branch** with a compare-and-swap, so concurrent runs
never fight over `index.lock` and never clobber each other's commits. `rdr_commit` prints
`committed <sha>  <subject>` — that line is the confirmation: no `git log`/`git status` after
it, and no reading the helper or the map before it. **Never branch
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
The var is read from a fresh `eval "$("$RDR_HOME/bin/rdr" env)"` in the shell
that commits — never restated from session memory (a resumed session's recall
can force-commit a project that has it off).

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
complex design; terse everywhere else. This file is read whole on every
`/rdr-*` run, so **its size is a per-run cost paid by every skill** — an added
paragraph is charged fourteen times over. Defer to CLAUDE.md and the README
Doctrine on any conflict.

**Do not split this file into a hot core + cold appendix.** Measured 2026-08-27:
counting each skill *plus the stage prompt it runs in the same context*, all 9
skills that read it cite ≥1 "cold" section, so a split buys zero avoided reads
and costs an extra Read turn each — and the turn is the cost. Prune duplication
instead. (Re-measure before re-opening; section reference counts taken from
SKILL.md alone will mislead you — prompts cite anchors too.)
