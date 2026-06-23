---
name: rdr-refine
argument-hint: <NNNN> [--commit | --no-commit]
description: Use to remove internal contradiction, redundancy, change-history narration, and bloat from an RDR draft (e.g. "refine RDR 46", "/rdr-refine 0046"). Runs Stage 3 of the RDR flow — fixes the document so it is internally consistent before its claims are verified. Pairs with /rdr-resolve (next) and /rdr-propose (back, if a contradiction is a real design hole).
---

# rdr-refine — Stage 3 (Refine)

Remove internal contradiction, redundancy, and bloat so the draft is *internally
consistent* before its claims are verified. Refine fixes the document; it does not
verify the world (that's Stage 4).

## Usage

```
/rdr-refine <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_RESOURCES`, `RDR_PATH`.
2. **Run the prompt** [`03-refine.prompt.md`](03-refine.prompt.md).
   The RDR is a **Draft** — edit it in place; rewrite to reflect the current world,
   don't preserve superseded content. Keep decision *rationale*; cut the *history*
   that produced it. It uses the Domain-priors section of `$RDR_RESOURCES` to decide
   which side of a contradiction is right (a clause that violates a product principle
   is the side to fix). EXCEPTION: a `Status: Draft [revised from Final …]` qualifier
   is live state, not history — leave it.

## Review gate (Stage `03-refine.md`)

- All change-history gone (no Rescope Note / Refinement Context / "retained as
  history" / "this is now wrong" annotation) — only the current design remains.
- Decision rationale NOT cut by mistake (rationale = *why this design*; history =
  *what it used to be* — keep the first, cut the second).
- Contradictions actually resolved, not deleted on one (wrong) side.
- Failure mode triaged, not just patched (the prompt routes the cure).
- Proportionate to the change's risk (contract count, not word count).

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `refine` first (see the rdr-common table).
- `Next: /rdr-resolve NNNN` — verify the Critical Assumptions against reality.
- A contradiction that's a real design hole → `/rdr-propose NNNN` (the approach is
  underspecified).
- `/rdr-status NNNN` to re-orient.
