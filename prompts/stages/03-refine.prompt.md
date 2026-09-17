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
3. **Collapse change-history into live text.** `recs lint`'s
   `prose:change-history` names the banned tokens where they survive; also
   delete "re-proposed/​re-promoted" Metadata narration and per-section
   "read against §X" annotation, which no rule decides. Fold whatever is still TRUE into the live
   Problem Statement / Solution / Alternatives; delete the rest outright. A
   reader should see only the current design, never the path to it. EXCEPTION:
   a `Status: Draft [revised from Final …; re-verify <IDs> @<stage>]` qualifier
   is parsed routing data (`inspect` emits its ids as `reverify` edges; `status`
   reads `reentry_target`), not history — keep it byte-for-byte. A route-back
   refine that amends or adds an assumption APPENDS that id to the list
   (`re-verify A2,A4` → `re-verify A2,A4,A9`), keeps `@<stage>`, and never
   writes `none`: `none` tells Stage 4 nothing is in scope.
   The sibling `[routed back from … @refine]` inverts that — it names THIS stage
   as owing the work, so rework and clear it whole (rdr-common §rdr-write
   *Receiving a route-back*). Preserve one, clear the other. The amended
   assumption itself keeps its prior terminal Status and stamp — the re-verify
   list, not the record's own line, says re-verification is owed; a
   "Verified (re-entry)" stamp written here asserts Stage-4 work this stage
   did not do. The 07.1
   `## Refinement Context` note is read-only for this stage — read it for the
   defect, append no disposition; Stage 7 deletes it. The re-lock flip clears
   the qualifier.
4. Find bloat — rejected-alternative narratives over ~30 lines, code blocks
   longer than the prose explaining them, deferred-feature detail. Trim.
5. Preserve decision rationale. Never trim WHY a choice was made; only trim
   restatement and history.

6. **Triage the failure mode before fixing it.** Refine is where size and
   under-specification first show. These are *opposite* diseases with *opposite*
   cures — applying the wrong cure backfires (splitting an under-spec'd RDR
   scatters the ambiguity; adding rigor to a too-large RDR makes it worse).
   Classify and route:
   - **Too large** — the RDR is the sole author of >1 independent load-bearing
     contract (forces simultaneous attention). → **Split** along the contract
     seams; do NOT add rigor. (Return toward Stage 2 to re-seed the split.)
   - **Under-specified** — an identity / format / naming / predicate decision is
     left open. → Add rigor to *that one decision* (the Load-Bearing Decisions
     slot), not prose everywhere.
   - **Wrong / entangled** — large AND on a structural/semantic concept (a
     drift, a trigger-emit, a policy). → Split AND pin the semantic decision
     before lock.
   A contradiction that is a real design hole (not sloppy wording) is the
   under-specified signal — route back to Stage 2.

Report the contradictions/redundancies found and the triage verdict, then apply
the fixes — each replacing exact bytes projected with `--select NNNN:<id>` **in
the same turn as the replace** (a projection several edits old has drifted),
never text recalled from squeezed/grepped output; portable tools only, no GNU-only
`sed` addresses (`N,+Mp`) or `cat -A`. Drop into ultrathink only for a contradiction whose resolution is a
real design call. Be brief but not lossy in the report — terse findings and
tight edits; spend tokens only where they change the result. Close with
rdr-common §mechanical-gate.
