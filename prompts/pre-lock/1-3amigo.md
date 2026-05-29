# 3amigo — Persona Review

**Use when**: every non-trivial RDR, as the first review round. Apply after the
draft is complete **and its Critical Assumptions are verified** — reviewing
against still-`Pending` assumptions critiques claims that may not hold (in the
flow, Stage 4 Resolve precedes this). Runs before Critique.

**What it uniquely catches**: heterogeneous PM/UX gaps, "first-hour implementer"
questions, "I cannot write a pass/fail test for this" QA gaps.

**Cost**: ~30 min per RDR.

## Prompt

```text
Review the RDR at {RDR_PATH} three times, each as a different persona, in
order. Do NOT reuse findings between personas.

Persona 1 — Product Manager
  Question: does this RDR actually deliver the user outcome?
  Deliverable: list every passage where the user outcome is unclear or the
    RDR is solving a different problem than the user has.

Persona 2 — Implementer
  Question: if I started coding this Monday, what would I ask in the first
    hour?
  Deliverable: list every concrete clarification-request the implementer
    would raise, with the RDR passage that triggered it.

Persona 3 — QA / Tester
  Question: how do I test this? What are the pass/fail criteria?
  Deliverable: list every test you cannot write because the RDR does not
    give enough to decide pass/fail.

Write each persona's list to {FLOW_DIR} as its own file —
persona-1-pm.md, persona-2-implementer.md, persona-3-qa.md
(where {FLOW_DIR} = _rdr/3amigo/<rdr-slug>/).

At the end, consolidate: which passages appeared under two or more personas?
Those are the highest-priority rewrites. Write that to
{FLOW_DIR}/consolidation.md — the file Stage 6 reads.
```

## Expected signal

- **Healthy** — three distinct lists; consolidated overlap highlights 2–4 hotspot passages.
- **Unhealthy** — each persona produced the same critique; no consolidation. (Restart with explicit instruction not to reuse.)

## Source

Basili et al. 1996, *Perspective-Based Reading* —
<https://doi.org/10.1007/BF00368702>. Combined with the Three Amigos practice
from BDD. Adapted from the corresponding spec-fitness battery prompt
(placeholder change only).
