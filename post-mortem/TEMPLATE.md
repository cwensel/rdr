# Post-Mortem: RDR-[NUMBER] [TITLE]

<!-- Engine-repo hygiene: this repo is public and generic.
This template is for RDR-INSTANCE post-mortems; instances
live in the consumer repo beside their RDR — never here.
PROCESS post-mortems (diagnoses of the flow itself) live in
the consumer's research tree, not this repo. Nothing stored
in this repo may carry project-specific metadata: no
consumer repo names, RDR ids/slugs, kata ids, source
paths/symbols, commit SHAs, or evidence paths. Aggregate
counts ("N of M") and engine-file references are fine;
scrub the rest before committing. -->


## RDR Summary

[2-3 sentence summary of what the RDR proposed to solve
and the approach it recommended.]

## Implementation Status

[Implemented | Partially Implemented | Not Implemented |
Reverted]

[1-2 sentences on the current state, including any notable
context like "reverted after 48 hours due to Maven
incompatibility" or "code complete but CI/CD activation
pending."]

---

## Implementation vs. Plan

### What Was Implemented as Planned

- [List items where the RDR accurately predicted the
  implementation]

### What Diverged from the Plan

For each divergence:

- **[Brief label]**: The RDR planned [X]. The
  implementation did [Y] because [reason].

### Existing Infrastructure Reused Instead of New Code

- [Component planned in RDR] was replaced by
  [existing module] because [reason]

### What Was Added Beyond the Plan

- [Features, infrastructure, or tooling that were
  implemented but not anticipated by the RDR]

### What Was Planned but Not Implemented

- [Items in the RDR that were skipped, with reason
  if known]

### Deviation Types

[Count the `Type:` entries in the launch's `deviations.md`
by the implementation taxonomy (README *Post-Mortem
Process*). Spec drift is SPEC-DEFECT + SPEC-UNDER;
implementation drift is IMPL-GAP; an entry typed outside
the list is listed by name — `recs status` flags it as
`impl_deviation_types_unknown`.]

| Type | Count |
| --- | --- |
| SPEC-DEFECT | |
| SPEC-UNDER | |
| DEPENDENCY-LIMIT | |
| TEST-FIXTURE | |
| IMPL-DECISION | |
| IMPL-GAP | |
| (outside the list) | |

---

## Drift Classification

Classify each divergence into one of these categories
to enable cross-RDR pattern analysis. Mark all that
apply with the count of instances.

| Category | Count | Examples | Preventable? |
| --- | --- | --- | --- |
| **Unvalidated assumption** | | | |
| **Framework API detail** | | | |
| **Missing failure mode** | | | |
| **Missing Day 2 operation** | | | |
| **Deferred critical constraint** | | | |
| **Over-specified code** | | | |
| **Under-specified architecture** | | | |
| **Scope underestimation** | | | |
| **Internal contradiction** | | | |
| **Missing cross-cutting concern** | | | |

**Preventable?** column: "Yes — source search" or
"Yes — spike" if verification would have prevented the
drift, "No" if inherently unpredictable.

### Pattern References

[For each drift category with count >= 2, reference
the applicable pattern from SYNTHESIS.md (if one
exists). This enables incremental synthesis updates.]

### Drift Category Definitions

- **Unvalidated assumption** — a claim presented as fact
  but never verified by source search or spike
- **Framework API detail** — method signatures, interface
  contracts, or config syntax wrong
- **Missing failure mode** — what breaks, what fails
  silently, recovery path not considered
- **Missing Day 2 operation** — bootstrap, CI/CD,
  removal, rollback, migration not planned
- **Deferred critical constraint** — downstream use case
  that validates the approach was out of scope
- **Over-specified code** — implementation code that was
  substantially rewritten
- **Under-specified architecture** — architectural
  decision that should have been made but wasn't
- **Scope underestimation** — sub-feature that grew into
  its own major effort
- **Internal contradiction** — research findings or stated
  principles conflicting with the proposal
- **Missing cross-cutting concern** — versioning,
  licensing, config cache, deployment model, etc.

---

## Escaped-Defect Ledger

One row per finding in the implementation's
`<art>/triage.md`, enriched with the two fields triage
cannot assign (at triage time the lens evidence is in
another repo, out of reach). Findings pre-filtered at
triage as `n/a-not-a-defect` (Phase-1 intentionally-RED
test findings) are excluded.

| Finding | odc-type | odc-trigger | Expected-catching stage | Precursor lens | Escape distance |
| --- | --- | --- | --- | --- | --- |
| | | | | | |

Route-back rows (rdr-common §punt-ledger) append here at the
moment a stage reopens a completed stage — odc columns n/a
until implementation triage.

**Lens miss note**: if a lens ran and returned clean but
the defect escaped anyway, write the cell as
`none (lens X ran clean)` — that separates "no lens
covered it" from "the right lens ran and missed it",
which is what makes per-lens false-positive rates mean
anything.

- **odc-type** / **odc-trigger** — copied verbatim from
  `<art>/triage.md`, which carries them per finding; not
  re-derived here.
- **Expected-catching stage** — which stage should have
  caught it. Closed vocabulary: `2-propose | 3-refine |
  4-resolve | 5-prelock | 6-reconcile | 7-finalize |
  7.1-cluster-reconcile | 8-implement |
  none-escaped-by-design | pre-existing-not-this-rdr`.
  The last two are not misses: `none-escaped-by-design`
  = the RDR specified the blind spot (cite the
  REQ/section); `pre-existing-not-this-rdr` = it
  reproduces at merge-base, so no stage of this RDR
  could have caught it.
- **Precursor lens** — which pre-lock lens raised a
  precursor, read from
  `<RDR_EVIDENCE>/<rdr-slug>/evidence/<lens>/`. Values:
  `grounding | 3amigo | critique | repeatability |
  repeatability-lite | cove | propose-premortem | none |
  n/a-no-lens-run`. `propose-premortem` is Stage 2's
  hardened critic (read from
  `evidence/propose-premortem/`), not a pre-lock lens —
  listed so the Lens miss note can say
  `none (propose-premortem ran clean)`.
  `repeatability` and `repeatability-lite` are distinct
  (different lens strength) — `run-1.md`'s header
  `variant: lite|full` line settles which ran. If more
  than one lens raised a precursor, list all that apply,
  comma-separated. On a foundational RDR grounding runs
  as Step 0 inside cove and its evidence lands in the
  cove folder — attribute `grounding` when the finding
  is a grounding-class claim, `cove` otherwise.
- **Escape distance** — stages between the
  expected-catching stage and stage 8 (expected at
  `2-propose` but reached implementation = 6; expected
  at `8-implement` = 0). `none-escaped-by-design` and
  `pre-existing-not-this-rdr` get `n/a`. It weights
  findings by how far they travelled, so cost is
  comparable across RDRs rather than counted flat.

---

## RDR Quality Assessment

### What the RDR Got Right

Focus on what was valuable and should be repeated in
future RDRs:

- [Architectural decisions, technology selections,
  trade-off analysis, research findings, etc.]

### What the RDR Missed

Focus on what should have been in the plan but wasn't:

- [Missing concerns, unconsidered constraints, wrong
  assumptions, etc.]

### What the RDR Over-specified

Focus on effort that did not contribute to a successful
implementation:

- **Code samples rewritten**: [list]
- **Deferred feature code unused**: [list]
- **Config/schema never implemented**: [list]
- **Performance targets unvalidated**: [list]
- **Alternative analysis disproportionate**: [list]

---

## Key Takeaways for RDR Process Improvement

List 3-5 actionable insights framed as improvements to
the RDR authoring process itself. Each takeaway should be:

- **Generalizable** — applicable to future RDRs beyond
  this specific topic
- **Actionable** — something an RDR author can do
  differently, not just "be more careful"
- **Evidence-based** — tied to a specific divergence or
  gap found in this post-mortem

1. **[Imperative verb phrase]**: [Explanation tied to
   evidence from this post-mortem]
2. **[Imperative verb phrase]**: [Explanation tied to
   evidence from this post-mortem]
3. **[Imperative verb phrase]**: [Explanation tied to
   evidence from this post-mortem]
