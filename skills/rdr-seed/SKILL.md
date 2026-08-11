---
name: rdr-seed
metadata:
  argument-hint: <kata-id… | "one-line idea"> [--label <name>] [--commit | --no-commit]
description: 'Use to start new RDRs from an idea, a kata, or a triaged batch (kata-id list or --label cohort). Runs Stage 1 per seed, allocates numbers, writes Draft skeletons. Trigger for seed/create RDR, $rdr-seed, or /rdr-seed.'
---

# rdr-seed — Stage 1 (Seed)

Turn an idea into a template-conformant Draft so every later stage has structured
slots to fill. A skeleton, not a finished document — no solution, no assumptions,
no research at seed time.

## Usage

```
Codex: $rdr-seed <kata-id | "one-line idea">
Claude: /rdr-seed <kata-id | "one-line idea">
        /rdr-seed <kata-id> <kata-id> …   # batch: explicit ids
        /rdr-seed --label <name>          # batch: open kind:rdr-seed ∩ <name>
```

**Batch selectors.** Multiple kata ids, or `--label <name>` → `kata list --status
open --label kind:rdr-seed --label <name> --json` (typically `rdr-seed-triage`'s
`batch:rdr-YYYY-MM-DD` cohort). Union + dedup, ascending id order; empty set →
refuse; a one-line idea is single-form only. Run steps 1–3 **once per seed,
sequentially** (§rdr-claim allocates in order; parallel claims just churn the
collision loop), §commit per seed. A member tripping the duplicate-seed guard is
**skipped and reported** (with its `tracks:` target), not a stop. Step 3 strips
`batch:*` at re-route, so re-invoking an interrupted `--label` run resumes the
remaining members.

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** to bind `$RDR_ENV`.
   **Duplicate-seed guard (before claiming):** a kata-id `{IDEA}` already labeled
   `kind:rdr-tracked` means a prior seed was consumed — report its `tracks:`
   target and stop; never mint a duplicate RDR. (`kind:rdr-seed` proceeds normally.)
   This skill does **not** resolve a number — it **allocates** one. Run
   **§rdr-claim**: atomically reserve the next number as `${NNNN}-RESERVED.md`
   *before authoring* (retry-on-collision loop), so concurrent sessions can't both
   take it. If `$RDR_ENV` names no RDR directory yet, ask the user before claiming.
2. **Run the prompt** [`01-seed.prompt.md`](01-seed.prompt.md)
   with `{IDEA}` = the arg and `{RDR_ENV}` bound. The §rdr-claim step already
   materialized the reserved file **as a copy of `TEMPLATE.md`** — fill that copy in
   place (H1 number/title, Metadata, Problem Statement, Context), leaving every other
   section as the template ships it. **Do not copy a neighbor RDR for style** — that
   drifts the template and leaks its solution. Then `git mv` to `<rdr-dir>/NNNN-slug.md`
   once the slug is known (the artifact dir `<rdr-dir>/NNNN-slug/` is its sibling).
   Finally **add the new RDR's row to the README index** (`$RDR_RECORDS/README.md`,
   the authoritative Status table) at `Draft` — the prompt spells out the format;
   finalize later flips this same row to `Final`.
3. **Kata-sourced seed? Re-route the kata** once the Draft exists: remove
   `kind:rdr-seed`, strip every `batch:*`, add `kind:rdr-tracked`, and
   `kata comment <id>` with **first line `tracks: cli/NNNN`** (the RDR number
   rides the comment — the `reviewed-at:` pattern; Stage 8's close step and the
   `kata-flow-ops` reaper read it back). The kata stays **OPEN** as the defect
   tracker — do NOT close it (dedup anchor, `blocks`-edge holder, impact record);
   it closes at Stage 8 when the RDR implements and the defect resolves. The
   label skip-filters it out of scope-review and flight waves (`label-vocabulary.md`).

## Review gate (Stage `01-seed.md`)

- Number + slug free (no collision), slug descriptive.
- Problem Statement leads with a **user outcome**, not a mechanism.
- No premature solution — Proposed Solution / Critical Assumptions / Research
  Findings are placeholders.
- **Is it RDR-shaped at all?** No real design fork → not an RDR. Refile as a plain
  issue and set `Status: Demoted [→ <issue link>]`; it runs no further stages.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `seed` first over `$RDR_PATH` + `$RDR_RECORDS/README.md` (the new index row).
- `Next: /rdr-propose NNNN` — enumerate approaches, choose one, premortem it.
- A batch run closes with one roster — `NNNN ← <kata-id>` per seed, plus any
  duplicate-guard skips — and the paste-ready `Next: /rdr-joint-propose
  <NNNN NNNN …>`: propose **all** siblings before any refines (the
  joint-decision check compares written proposals; bare seeds are invisible to
  it — stages/02-propose.md, batch ordering).
- Demoted → done (refiled as an issue).
- `/rdr-status NNNN` to re-orient.
