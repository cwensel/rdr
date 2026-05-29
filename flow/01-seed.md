# Stage 1 — Seed

**Goal**: turn an idea (a kata, a matrix gap, a sentence) into a
template-conformant Draft so every later stage has structured slots to fill.

## Paste this

Fill `{IDEA}` (a kata id or a one-line description) and `{RDR_ENV}` (default
`_rdr/rdr-env.md`). The RDR process README + TEMPLATE are the flow's own,
beside this file.

```text
Seed a new RDR from: {IDEA}. See the rdr README and TEMPLATE (beside the flow
docs) for what an RDR is and the template.

Read {RDR_ENV} and place the new RDR file in the project's RDR directory — the
parent of {ARTIFACT_DIR} (the RDR file and its artifact dir are siblings:
file `<rdr-dir>/NNNN-slug.md`, artifacts `<rdr-dir>/NNNN-slug/`). If {RDR_ENV}
names no RDR directory yet, ask me before writing.

Allocate the next free NNNN (max across existing RDR files there). Fill the
template's Metadata, Problem Statement, and Context from the kata. For every
other section, keep the template's structure and mark it a Draft
placeholder — do NOT invent a solution, assumptions, or research findings;
those are later stages.

State the Problem Statement as a USER OUTCOME first (who runs this, what they
want to accomplish, how they discover they need it), then the system-internal
requirement. Do not solve it here.

If {IDEA} is a plain description rather than a kata id, synthesize a
one-paragraph problem statement from it, flagged for my review.

This is a skeleton, not a finished document. No ultrathink at seed time.
```

**Run when**: starting a brand-new RDR. **Produces**: a new RDR file in the
project's RDR directory (the `{ARTIFACT_DIR}` parent from `{RDR_ENV}`), Status: Draft.

## Review gate

- **Number + slug** are free (no collision) and the slug is descriptive.
- **Problem Statement leads with a user outcome**, not a mechanism. "The
  system will parse…" is a mechanism — send it back.
- **No premature solution.** Proposed Solution, Critical Assumptions, and
  Research Findings are placeholders. Seeding that already picks an approach
  skips Propose's option analysis — reject it.
- **Metadata is honest**: Status Draft; Type/Priority set; Predecessors listed
  if the kata names dependencies.

## Advance when

The Draft exists, conforms to the template skeleton, and its Problem Statement
is a reviewable user-outcome statement with no solution baked in.

→ Next: [02-propose.md](02-propose.md)
