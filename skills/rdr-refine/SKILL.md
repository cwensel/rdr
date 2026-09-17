---
name: rdr-refine
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: 'Use to remove contradiction, redundancy, change-history narration, and bloat from an RDR draft. Runs Stage 3 before assumption verification. Trigger for refine RDR, $rdr-refine, or /rdr-refine.'
---

# rdr-refine — Stage 3 (Refine)

Remove internal contradiction, redundancy, and bloat so the draft is *internally
consistent* before its claims are verified. Refine fixes the document; it does not
verify the world (that's Stage 4).

## Usage

```
Codex: $rdr-refine <NNNN>
Claude: /rdr-refine <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md) **whole, with the Read tool** (it exceeds the 30KB
   Bash cap — `cat` truncates and costs a retry; never `sed`/`grep` §-slices); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `RDR_PATH`.
2. **Run the prompt** [`03-refine.prompt.md`](03-refine.prompt.md).
   The RDR is a **Draft** — edit it in place; rewrite to reflect the current world,
   don't preserve superseded content. Keep decision *rationale*; cut the *history*
   that produced it. It uses the Domain-priors section of `$RDR_RESOURCES` to decide
   which side of a contradiction is right (a clause that violates a product principle
   is the side to fix). EXCEPTION: a `Status: Draft [revised from Final …; re-verify
   <IDs> @<stage>]` qualifier is parsed routing data (`reverify` edges,
   `reentry_target`), not history — keep it byte-for-byte; a route-back refine
   rewrites only the id list to `re-verify <IDs> + <ids it amended or added>`,
   keeping `@<stage>`, never `none`. A `[routed back from … @refine]` inverts that —
   rework and clear it whole (rdr-common §rdr-write *Receiving a route-back*). The 07.1 `## Refinement Context` note is
   read-only here — no disposition appended; Stage 7 deletes it. Read the record by `inspect NNNN` then
   `--select NNNN:§section` (rdr-common §rdr-resolve), never `sed -n` line windows.

## Review gate (Stage `03-refine.md`)

- All change-history gone — `recs lint`'s `prose:change-history` is the check;
  only the current design remains.
- Decision rationale NOT cut by mistake (rationale = *why this design*; history =
  *what it used to be* — keep the first, cut the second).
- Contradictions actually resolved, not deleted on one (wrong) side.
- Failure mode triaged, not just patched (the prompt routes the cure).
- Proportionate to the change's risk (contract count, not word count).

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `refine` first (row in rdr-commit-map.md).
- `Next: /rdr-resolve NNNN` — verify the Critical Assumptions against reality.
- A contradiction that's a real design hole → `/rdr-propose NNNN` (the approach is
  underspecified).
- `/rdr-status NNNN` to re-orient.
