# 3amigo — Persona Review

**Use when**: every non-trivial RDR, as the first review round. Apply after the
draft is complete **and its Critical Assumptions are verified** — reviewing
against still-`Pending` assumptions critiques claims that may not hold (in the
flow, Stage 4 Resolve precedes this). Runs before Critique.

**What it uniquely catches**: heterogeneous PM/UX gaps, "first-hour implementer"
questions, "I cannot write a pass/fail test for this" QA gaps.

**Cost**: ~30 min per RDR.

## Prompt

Dispatch **three isolated passes** — one per persona, each a separate sub-agent
(or fresh-context) run that sees the RDR and its own persona block only, never
another persona's findings. Each pass gets the prompt below with one persona
block. {EVIDENCE_DIR} is already bound by the dispatcher to this lens's
`3amigo/<rdr-slug>/` folder under the base {RDR_ENV} defines — do not re-derive it.

```text
Review the RDR at {RDR_PATH} as the persona below. Report every passage you
can anchor by name, severity-ranked, each naming the decision it blocks or
the test it prevents. No minimum, no maximum: zero anchored findings is a
valid report. Write your list to {EVIDENCE_DIR}/<persona-file>.

Where to start reading. Run once:

  "$RDR_HOME/bin/rdr" inspect --json --filter elements,outline {RDR_PATH}

Your persona names which `elements[]`/`outline[]` entries it owns; each carries
`id`, `line_start`, `line_end`. Read those spans with `sed -n` and anchor every
finding to the `id` (`NNNN:C4`, `NNNN:S2`) rather than a quoted phrase — an id
survives a reword, a quote does not.
If your persona's set comes back empty — an older record labels no contracts or
scenarios — read the record whole; an empty set is a record that named nothing,
never a record with nothing to find.

That set is a STARTING SET, not a boundary. WIDEN to the fuller record whenever
your persona's question cannot be answered inside it — an implementer's first-hour
question is often about what the contracts do NOT say, and a silence has no line
range. When you widen, say so in your report and name what sent you.

Persona 1 — Product Manager → persona-1-pm.md
  Question: does this RDR actually deliver the user outcome?
  Owns: `outline[]` §problem-statement, §approach, §decision-rationale, plus
    `elements[]` kind `MVV` — the outcome is stated in prose, so this persona
    widens most readily.
  Report: passages where the user outcome is unclear or the RDR solves a
    different problem; for each, the decision it blocks.

Persona 2 — Implementer → persona-2-implementer.md
  Question: if I started coding this Monday, what would I ask in the first
    hour?
  Owns: `elements[]` kind `C` (normative contracts) and kind `D` (load-bearing
    decisions); `edges[]` kind `source-anchor` names the code it lands on.
  Report: clarification-requests, each with the RDR passage that triggered
    it and the decision it blocks.

Persona 3 — QA / Tester → persona-3-qa.md
  Question: how do I test this? What are the pass/fail criteria?
  Owns: `elements[]` kind `S` (validation scenarios), kind `MVV`, and kind `C`
    — a scenario is testable only against the contract it exercises.
  Report: tests you cannot write for lack of pass/fail criteria, each with
    the passage at fault and the test it prevents.
```

After all three files land, the **dispatcher consolidates mechanically**:
compute which passages two or more personas named, rather than having a model
that has read all three re-judge them. Where the personas anchored to element
ids, the overlap is a set intersection on those ids — exact, not a string match
on quoted prose. Merge every finding into
{EVIDENCE_DIR}/consolidation.md — the file Stage 6 reads — tagging the
multi-persona passages as **hotspots**. Overlap marks a hotspot passage, not a
validated finding; a finding raised by exactly one persona is not thereby
weaker. (Because the personas never saw each other, their agreement is
evidence, not conformity.)

## Expected signal

- **Healthy** — every finding anchored to a named passage with the decision or
  test it blocks. A persona reporting zero findings on a clean draft is a valid
  result, not a malfunction.
- **Unhealthy** — unanchored or generic advice, findings that name no blocked
  decision or test, or a persona file that references another persona's output
  (isolation leaked — re-run that pass in a fresh context).

## Source

Basili et al. 1996, *Perspective-Based Reading* —
<https://doi.org/10.1007/BF00368702>. Combined with the Three Amigos practice
from BDD. Adapted from the corresponding spec-fitness battery prompt (isolated
per-persona execution; no finding quota).
