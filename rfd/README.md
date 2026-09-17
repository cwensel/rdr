# Requests for Discussion (RFDs)

An **RFD** states a capability: the problem, the desired experience, and the
principles that capability commits to. It is the top tier of the three —
[RFD](README.md) for a capability, [JDR](../jdr/README.md) for what a seam's
records jointly decide, [RDR](../README.md) for one seam's contract.

This is the engine's RFD document class: the scope rules, the Principles
grammar, and [`TEMPLATE.md`](TEMPLATE.md). Instances live in the consumer repo
(`rfd/NNNN/README.md`), which also owns the authoring workflow — branch,
share, states, conversion. Convention is never restated in an instance.

## The rule this tier obeys

**Prose is mutable. Principle ids are not.**

An RFD can be rewritten end to end so long as no `P-n` changes meaning. That is
what makes it a living document while records cite it: a record citing a
principle cites a commitment, and one citing a section cites *context*.

## What an RFD holds

- Problem statement and background.
- Desired experience and journeys — **non-normative by default**, and marked
  so. Where the narrative needs an example, it is an example.
- A numbered **Principles** section. This is the citable surface.
- Non-goals, and open questions.
- `governs:` frontmatter — the JDRs and RDR clusters under it.

## What an RFD must not hold

Each of these belongs to a tier that can carry it, and each has been found
inside an RFD in practice:

| Not this | Because | Where it goes |
| --- | --- | --- |
| A locked decision table | A living document cannot carry append-only rows; the two rules contradict | a **JDR** entry |
| A closed-question ledger | Same, plus it turns a vision into a registry | a **JDR** |
| A push-down or follow-on work list | Stale the moment the first record re-locks, and nothing updates it | the RDR index and the tracker |
| A reconcile scratchpad ("Closed 2026-09-03", "re-locked") | Process history | git |
| A transcript presented as contract | It reads as normative and was never gated | a record, or nothing |

## Principles — the citable surface

```
- **P-3** Every refusal MUST name what was missing and who can fix it.
  *(Derived from §4's refusal journeys.)*
```

- **One line each.** A principle that needs a paragraph is an argument, and
  arguments belong in the section that makes them.
- **An RFC 2119 force word** — MUST, SHOULD, MAY. A principle with no force is
  a description.
- **A rationale**, and **derived, never invented**: traceable to the narrative
  above it. A principle with no anchor is the author legislating.
- **It earns its place by ruling options out.** If every option on a fork
  satisfies it, it is decoration.
- **Ids are append-only.** A principle that no longer holds is marked
  superseded in place, with a date and a pointer. **The id never moves** — a
  record that cited it still means what it said.

## Citation, and what freezes

```
RFD NNNN P-n     a principle — CONTRACT
RFD NNNN §x      a section — CONTEXT, tolerated for legacy records
```

Section numbers freeze once cited, and nothing enforces that separately:
`recs` resolves the anchor, so a renumber shows up as unresolved edges from
every record that cited it. Prose under a frozen section may still evolve —
that is the tier's whole point.

New material appends. A record cites a section for context, and git history at
the record's lock date gives the reading it saw.

## States and versioning

States are the consumer's (`prediscussion` … `committed`). Two rules the engine
does care about:

- **Versioning is git plus principle ids.** No document version number.
- **`committed` is the honest state for an RFD with shipped children.** An RFD
  still reading `prediscussion` while fifteen records have shipped against it
  is a defect in the frontmatter, not a description of reality.
