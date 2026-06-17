# Stage 4 — Resolve Assumptions (Research + Spikes)

**Goal**: verify every Critical Assumption against reality — source, live
spikes, prior art, standards — so the review rounds critique a *true*
document. The load-bearing stage. **Running a pre-lock round before this one
wastes the round**: it critiques claims that may not hold.

## Paste this

Fill `{RDR_PATH}`. Resolve `{RDR_RESOURCES}` / `{RDR_ENV}` via the workspace
marker (see README *Where the seam lives* — `. "$WS/.rdr-workspace"` exports
`$RDR_RESOURCES` / `$RDR_ENV`). Have a live spike target reachable.

**Prompt**: [`../prompts/flow/04-resolve.prompt.md`](../prompts/flow/04-resolve.prompt.md)
— the `/rdr-resolve` skill runs it (it self-detects the scoped-re-entry path from
the RDR's Status line). Paste its body by hand if driving without the skill.

**Run when**: the draft has a chosen, tightened approach with a Critical
Assumptions list (almost always — any load-bearing assumption). **Produces**:
assumptions flipped to `Verified` with Method + Evidence; spike commands +
output under `{SPIKE_DIR}`; the evidence-body (*Testing Strategy* +
*Performance Expectations*) authored from the verified assumptions.

## Review gate

- **Is every load-bearing assumption actually Verified** — with Evidence that
  isn't self-reference and isn't bare `Docs Only`? The Stage 7 mechanical
  sweep re-checks this at lock; set the floor here so the sweep only ever
  catches a later regression, never an original defect.
- **Did spikes actually run?** A "Spike" Method with no command + captured
  output is a Docs-Only claim in a costume. Demand the output.
- **Did the reuse audit run?** Confirm the `{RDR_ENV}` reuse-audit paths were
  actually checked for code that already does what the approach introduces — a
  silent skip ships a design that rebuilds existing capability.
- **Did research change the design?** If a finding (or a reuse hit) contradicts
  the chosen approach, that's not a citation fix — **return to Stage 2 or 3**,
  rework the approach, then come back. (Ask: *do these findings force critical
  changes to the proposed solution?*)
- **Are citations from local corpora**, not hallucinated? Cross-check a sample
  against the corpus or cloned source.
- **Is the evidence-body authored from the verified assumptions?** Testing
  Strategy and Performance Expectations are real, not placeholders, and the
  seed-skeleton header is gone — the implementer reads these as the contract.

- **On a `revised from Final` re-entry, was scope honored?** Only the
  qualifier's listed IDs (and anchors the edit touched) get re-verified; a full
  re-verify of lock-audited assumptions is the wasted pass this scoping exists to
  prevent.

Re-run for the *subset* of assumptions that didn't verify — narrow to the
unresolved ones, don't re-run the whole stage.

## Advance when

Every Critical Assumption is `Verified` with a non-self-referential Evidence
Record (or explicitly `Pending` with a plan that will run during
implementation), no load-bearing claim rests on `Docs Only` alone, no
research finding contradicts the approach, and the evidence-body (Testing
Strategy, Performance Expectations) is authored — no `_Draft placeholder._`
nor seed-skeleton header left in the document.

→ Next: [05-prelock.md](05-prelock.md)
