# Stage 3 — Refine

**Goal**: remove internal contradiction, redundancy, and bloat so the draft is
*internally consistent* before its claims are verified. Refine fixes the
document; it does not verify the world (that's Stage 4).

## Paste this

Fill `{RDR_PATH}` and `{RDR_RESOURCES}` (default `_rdr/rdr-resources.md`).

```text
Review the RDR at {RDR_PATH} for inconsistencies and redundancies. This RDR is
a DRAFT — edit it in place. Rewrite to reflect the current world; do not
preserve superseded content. (The "never amend" rule is for Final/frozen RDRs,
not Drafts — see the flow README Doctrine.) Keep the rationale for decisions;
cut the history that produced them. See the rdr README for guidance, and the
*Domain priors* section of {RDR_RESOURCES} when deciding which side of a
contradiction is right (a clause that violates a product principle is the side
to fix).

1. Find every place two sections disagree (the approach says X, a later
   example assumes not-X). List each as a contradiction with both quotes,
   then resolve it in the draft.
2. Find redundancy — the same claim in three sections, Problem-Statement
   recaps inside Alternatives. Collapse to one authoritative statement and
   cross-reference.
3. **Collapse change-history into live text.** Delete every "Rescope Note",
   "Refinement Context", "retained as history" section, "re-proposed/​
   re-promoted" Metadata narration, and per-section "this is now wrong / read
   against §X" annotation. Fold whatever is still TRUE into the live
   Problem Statement / Solution / Alternatives; delete the rest outright. A
   reader should see only the current design, never the path to it. EXCEPTION:
   a `Status: Draft [revised from Final …]` qualifier is live state (the 08.1
   re-entry signal Stage 4 needs), not history — leave it; the Stage 8 re-lock
   flip clears it.
4. Find bloat — rejected-alternative narratives over ~30 lines, code blocks
   longer than the prose explaining them, deferred-feature detail. Trim.
5. Preserve decision rationale. Never trim WHY a choice was made; only trim
   restatement and history.

Report the contradictions/redundancies found, then apply the fixes. Drop into
ultrathink only for a contradiction whose resolution is a real design call.
```

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
- If a surfaced contradiction is a *real design hole* (not sloppy wording),
  that's a signal to **return to Stage 2** — the approach is underspecified.

Re-run if the draft is still self-inconsistent after one pass.

## Advance when

No two sections contradict; no claim is stated in more than one authoritative
place; the document is proportionate; all decision rationale survives.

→ Next: [04-resolve-assumptions.md](04-resolve-assumptions.md)
