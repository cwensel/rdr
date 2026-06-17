# Stage 3 — Refine

**Goal**: remove internal contradiction, redundancy, and bloat so the draft is
*internally consistent* before its claims are verified. Refine fixes the
document; it does not verify the world (that's Stage 4).

## Paste this

Fill `{RDR_PATH}`. Resolve `{RDR_RESOURCES}` via the workspace marker (see
README *Where the seam lives* — `. "$WS/.rdr-workspace"` exports `$RDR_RESOURCES`).

**Prompt**: [`../prompts/flow/03-refine.prompt.md`](../prompts/flow/03-refine.prompt.md)
— the `/rdr-refine` skill runs it. Paste its body by hand if driving without
the skill.

**Run when**: the draft has a chosen approach but is messy — repeats itself,
carries change-history narration, contradicts its own sections, or is bloated.
**Produces**: edits to the RDR draft.

## Review gate

- **Is all change-history gone?** No "Rescope Note", "Refinement Context",
  "retained as history", "re-proposed" Metadata narration, or "this is now
  wrong" annotation should survive — only the current design. A draft that
  still reads as a layered edit log goes back.
- **Did it cut rationale by mistake?** The one thing this stage must not lose
  is *why* decisions were made. Spot-check that every non-obvious choice still
  has its reason. (Rationale = *why this design*; history = *what the design
  used to be* — keep the first, cut the second.)
- **Are contradictions actually resolved**, not just deleted on one side?
  Resolving by picking the wrong side is worse than leaving it.
- **Is it proportionate now?** Right-sized for the change's risk (Gate
  Proportionality).
- **Was the failure mode triaged, not just patched?** The prompt routes
  too-large / under-specified / wrong-entangled to their (opposite) cures;
  confirm the right cure was applied, not that the mechanics were re-derived.
- If a surfaced contradiction is a *real design hole* (not sloppy wording),
  that's a signal to **return to Stage 2** — the approach is underspecified.

Re-run if the draft is still self-inconsistent after one pass.

## Advance when

No two sections contradict; no claim is stated in more than one authoritative
place; the document is proportionate; all decision rationale survives.

→ Next: [04-resolve-assumptions.md](04-resolve-assumptions.md)
