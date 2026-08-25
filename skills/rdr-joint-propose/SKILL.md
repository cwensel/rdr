---
name: rdr-joint-propose
metadata:
  argument-hint: <NNNN NNNN …> [--model-ceiling <model>] [--commit | --no-commit]
description: 'Use to propose a batch of seeded sibling RDRs in dependency order so the joint-decision check runs against written proposals. Sequential per-RDR proposers, re-ordered on emergent dependencies; true joint forks go to the user. Trigger for batch/joint propose, $rdr-joint-propose, or /rdr-joint-propose.'
---

# rdr-joint-propose — Stage 2 across a cohort

Run Stage 2 over seeded siblings in one orchestrated pass — the breadth-first
order stages/02-propose.md mandates. The joint-decision check compares *written
proposals*, and dependencies emerge from it: a propose sometimes cannot complete
until a peer is proposed first (observed: an RDR could not bind a flag
disposition until its sibling's Peer Bindings existed). So this is a dependency-driven
**sequential scheduler**, not a parallel fan-out: pre-order the roster, propose
one at a time, re-order when a dependency emerges, surface only the true joint
forks.

## Usage

```
Codex: $rdr-joint-propose <NNNN> <NNNN> …
Claude: /rdr-joint-propose <NNNN> <NNNN> …
```

List-only — RDR numbers, never labels or kata ids (`/rdr-seed`'s batch close
packet prints this command ready to paste). Per member: §rdr-resolve; must be
`Status: Draft` with Proposed Solution still the template placeholder
(seeded-unproposed). Already-proposed → **skip and report** ("re-open singly:
`/rdr-propose NNNN`" — re-proposal is a deliberate single act with the
freshness check in play). Empty roster → refuse. A **single member is valid** —
the elevation path: one heavy propose delegated at `--model-ceiling` (the
model-adequacy fork routes here); Phase 0 degenerates to a trivial order and
the loop runs once. Interrupted run → re-invoke the same list; the skip guard
resumes from disk state (no `--resume`, §no-heartbeat).

## Posture — the Stage-8 delegation exception

Single `/rdr-propose` authors in the main context (§delegation). This
orchestrator cannot — N full proposals in one context is the token blowup batch
mode exists to avoid. Like Stage 8's `launch.md`, it delegates everything: each
proposer sub-agent authors its own RDR + evidence and runs §commit itself
(propose rows of the table; the brief carries the orchestrator's resolved
autocommit on/off), returning only a §return-packet. The orchestrator holds the
roster, the overlap graph, packets, and fork dispositions; it never authors
proposal content.

## Phase 0 — bind & pre-order

§seam-bind, then §rdr-resolve each member. Build the overlap graph from the
projection — one pass, no seed body read:

```sh
[ -x "$RDR_HOME/bin/rdr" ] && {
  "$RDR_HOME/bin/rdr" index --anchor-intersect --json --all --records "$RDR_RECORDS" --repo "$RDR_SOURCE_REPO"
  "$RDR_HOME/bin/rdr" index --json --records "$RDR_RECORDS"   # elements[] kind=="C"
  "$RDR_HOME/bin/rdr" index --in-flight --records "$RDR_RECORDS"
}
```

`overlaps[]` `{records[], anchors[], cited}` **is** the overlap graph — seeds
are pre-proposal, so `--all` widens past in-flight. `$RDR_SOURCE_REPO` is the source
root (rdr-common §source-root); `--repo` is required or source-anchor edges carry
no `resolved` key (absent ≠ false, never "no overlap"). Contract literals: `elements[]`
`kind=="C"`, equal `hash` across two records = same contract text. `overlaps[]`
already ignores the template's own `path::Symbol`, but **`C` hashes are not** —
a template-shipped contract hashes identically in every seed that kept it
(observed: one hash across 13 records), so **drop any hash also carried by
`TEMPLATE.md`** before linking a pair, as the token filter did. `Predecessors`
edges (`edges[]` `kind=="predecessor"`) are hard edges — topo-sort those first.
Within an overlap group the likely **contract owner** proposes first: locus *is*
the shared anchor > lower-level seam > Priority > number order. The order is a
prior, not a promise — the loop corrects it; this pass makes corrections rare.

Absent binary: collect modify-anchors per member by hand (the `Seam Lineage`
locus + backticked `path::Symbol` from Problem/Context, since Proposed Solution
is placeholder) plus contract literals, **dropping any token also backticked in
`TEMPLATE.md`**; grep-cheap — orchestrator-local, or one small sub-agent.

## The loop — one proposer at a time

Spawn one proposer per member, sequentially in order (§delegation `Task` tool;
model per §model-ceiling). Brief = the bound seam vars + `{RDR_PATH}`, the peer
roster (NNNN + title — awareness, never solutions; `index --in-flight` prints
exactly that, no body — do not widen it), and: run the sibling
[`02-propose.prompt.md`](02-propose.prompt.md) **in full — including step 8**,
which now compares real peer proposals — plus two batch-only additions:

- **Fail-fast order sniff.** After prompt step 1 (prior art), before drafting:
  if a still-unproposed roster peer owns a decision this RDR must bind against
  (the consumer side of a shared anchor), stop — return `NEEDS_DECISION`,
  `next_action: propose NNNN first` (code `stopped:propose-order:NNNN-first`).
  Who shares which anchor is Phase 0's `overlaps[]` — read it, don't re-derive;
  which side *owns* the contract stays judgment. Never author a proposal doomed
  to re-run.
- **Self-commit** via §commit before returning.

Orchestrator, per packet:

- `PASS` → next member.
- `stopped:propose-order` → suspend this member behind NNNN, promote NNNN,
  continue. The re-run is discounted by design — `{EVIDENCE_DIR}research/` is
  the prompt's citation cache, so a re-ordered propose repays drafting, not
  research. An order **cycle** (A needs B, B needs A) is a true fork — the
  mutual fire *is* the joint decision (the 0128/0129 contract split).
- `stopped:joint-decision` (a fire that survives ordering, or names a
  non-roster peer) → fork dispositions below.
- Malformed packet → request a corrected packet only (§return-packet); still
  malformed → one re-spawn at the ceiling (§model-ceiling), then surface.

## Fork dispositions

A fire is a human-judgment fork (prompt step 8) — schedule it, never answer it
(rdr-common §fork-disposition owns the rule and the ask-now/park split):

- **Ask now** when any *remaining* member's anchors overlap the fired pair —
  proposing past an unresolved fork that feeds a later member recreates the
  waste this skill exists to kill. Otherwise park the fired member and finish
  the roster.
- **Non-roster fire.** Never silently expand the roster: the question carries
  the standard dispositions (hoist / cite-don't-restate / declare in both)
  plus "pull NNNN into this run" when that peer is itself seeded-unproposed.
- **Apply.** Delegate each answered disposition as a scoped edit sub-agent per
  affected RDR (update the `Joint-check:` line + cross-citations, re-commit);
  re-run the check for just that pair.

## Model ceiling

rdr-common §model-ceiling: per-spawn, profile-gated — `small`/`mid` at session
model, `large`/`foundational` at the ceiling — plus the one escalation retry
above. Unset ceiling = every spawn at session model — so if the roster holds
heavy members and no ceiling is set, run the **model-adequacy fork** once,
before the loop (there is nothing to bump to; the user may want to cancel and
restart with `--model-ceiling`).

## Review gate

Per member, the single-propose gate (stages/02-propose.md) applies — each RDR
carries its own `Premortem:`/`Ground-sweep:`/`Joint-check:` lines. The batch adds:

- Every member reached a terminal outcome: proposed | skipped | parked-fork.
- Every `Joint-check:` line written; every fire disposed or surfaced.
- No member proposed past an unresolved fork that feeds it (ask-now rule held).

## Next step (rdr-common §next-step)

Emit the standard close packet **once for the batch**, with the roster inside
it: `Outcome:` = one line per member (`NNNN  outcome  joint-check  model`);
`RDR delta:` = the propose row per member, plus the fires resolved/parked —
the batch's own product; `Next:` = `/rdr-refine NNNN` per clear member (the
fork question per parked one, `/rdr-propose NNNN` per skip), both command
spellings.
