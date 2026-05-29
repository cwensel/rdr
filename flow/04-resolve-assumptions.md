# Stage 4 — Resolve Assumptions (Research + Spikes)

**Goal**: verify every Critical Assumption against reality — source, live
spikes, prior art, standards — so the review rounds critique a *true*
document. The load-bearing stage. **Running a pre-lock round before this one
wastes the round**: it critiques claims that may not hold.

## Paste this

Fill `{RDR_PATH}`, `{RDR_RESOURCES}` (default `_rdr/rdr-resources.md`), and `{RDR_ENV}`
(default `_rdr/rdr-env.md`). Have a live spike target reachable.

```text
For the RDR at {RDR_PATH}, resolve all critical assumptions.

Read {RDR_RESOURCES} for the corpora, design docs, and anchors; read {RDR_ENV} for the
spike-output location ({SPIKE_DIR}) and the reuse-audit source paths. Two source
domains: {RDR_ENV}'s paths point at THIS project's own code; {RDR_RESOURCES}'s
corpora point at EXTERNAL source (dependency/peer/standard). A claim is verified
against whichever domain owns the behavior — own-code claims against project
source, external-behavior claims against the corpora. Use the corpora to
research against the literature, prior art, competition, and standards; confirm
all references and citations from that local research, not from memory.

FIRST, the reuse audit: for each behavior the approach introduces, check the
{RDR_ENV} reuse-audit paths for code that already does it. If the codebase already
provides it, that is a finding — fold reuse into the design or return to Stage
2/3 rather than verifying an assumption about building anew.

This stage is read-heavy: for each corpus search, reuse audit, or dependency
spelunk, delegate to a sub-agent that returns the verdict + evidence pointer
(file:line, or spike command + output path) — NOT raw hits or whole files.
Hold only the verdicts here.

SCOPED RE-ENTRY: if the Status reads `Draft [revised from Final …; re-verify
<IDs>]`, this RDR was lock-audited and demoted by the 08.1 cluster gate for a
named defect. Re-verify ONLY the listed assumption IDs (plus any whose Evidence
anchor the demotion's edit touched); carry the rest forward as already Verified
— do NOT re-derive them. A bare `Draft` (no qualifier) is the cold path: verify
every assumption from scratch as below.

For each Critical Assumption in scope:
- Pick exactly one Method: Source Search | Spike | Prior Art | Derivation |
  Design Decision | Peer RDR | MVV Test | Docs Only.
- Produce concrete Evidence for it:
    Source Search → file:line in the actual source that owns the behavior:
      for a claim about THIS project's own code, the project source (the
      {RDR_ENV} reuse-audit paths are a fine starting point); for a claim about
      EXTERNAL behavior (a dependency, peer tool, or standard), the
      {RDR_RESOURCES} corpora. NOT this RDR or its artifact dir — citing the
      spec to verify the spec is self-reference, not verification. (The reuse
      audit above asks a different question of the same own-code source: does a
      capability already EXIST? Source Search asks: does the code BEHAVE as the
      claim says?)
    Spike → the command run against a live service/fixture + where output is
      captured under {SPIKE_DIR}. Actually run it; paste the output.
    Prior Art → named external system + section/page.
    Derivation → the math, inline.
    Peer RDR → the RDR id + section that owns the property.
    Docs Only → INSUFFICIENT for load-bearing claims; allowed only paired
      with a Spike or Source Search plan stated in the Evidence line.
- Set Status: Verified only when Method + Evidence actually support it.
  Otherwise leave Pending with a named verification plan.
- Confirm "If wrong" is non-empty and names how it surfaces to a user/test.

Verify EXACTNESS words too (all/every, first/nearest, byte-identical,
lossless, canonical, deterministic, stable order) — each needs an Evidence
Record or coverage by the Minimum Viable Validation. For byte-stable output,
run the determinism checklist (hash fn+lib, pre-image byte layout, encodings,
map order, whitespace, case folding, empty/null/absent, version marker).

Be brief in results; ultrathink for complex design or any load-bearing
assumption; never trade brevity for a weaker verification. Report per
assumption: Status + Method + one-line Evidence, and flag any you could NOT
verify.
```

**Run when**: the draft has a chosen, tightened approach with a Critical
Assumptions list (almost always — any load-bearing assumption). **Produces**:
assumptions flipped to `Verified` with Method + Evidence; spike commands +
output under `{SPIKE_DIR}`.

## Review gate

- **Is every load-bearing assumption actually Verified** — with Evidence that
  isn't self-reference and isn't bare `Docs Only`? The Stage 8 mechanical
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

- **On a `revised from Final` re-entry, was scope honored?** Only the
  qualifier's listed IDs (and anchors the edit touched) get re-verified; a full
  re-verify of lock-audited assumptions is the wasted pass this scoping exists to
  prevent.

Re-run for the *subset* of assumptions that didn't verify — narrow to the
unresolved ones, don't re-run the whole stage.

## Advance when

Every Critical Assumption is `Verified` with a non-self-referential Evidence
Record (or explicitly `Pending` with a plan that will run during
implementation), no load-bearing claim rests on `Docs Only` alone, and no
research finding contradicts the approach.

→ Next: [05-prelock.md](05-prelock.md)
