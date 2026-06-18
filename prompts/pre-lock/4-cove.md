# Chain-of-Verification — Codebase-Grounded Silence & Contradiction Probe (S-04)

**Use when**: single-RDR pre-lock, on any non-trivial RDR. Apply after the
draft is complete **and its assumptions are verified** (Stage 4). One of the
single-RDR lenses; peers with [3amigo](1-3amigo.md), [critique](2-critique.md),
and [repeatability](3-repeatability.md). At Foundational, cove runs **first** —
ground the frame before the personas debate it (see the Stage 5 matrix).

**What it uniquely catches**: silences ("the RDR is silent on this"), internal
contradictions, **and claims internally coherent but false against the codebase**
— the only lens that checks the RDR against the *code*, not just itself. The
[grounding Step 0](0-grounding.md) is its oracle half (cove embeds it); the 12
questions are the silence/contradiction half.

**Cost**: 20–30 min. **Dual-model recommended.**

## Prompt

```text
Step 0 (GROUNDING — do this first): run the codebase claim sweep from
0-grounding.md against {RDR_PATH} — verify every cited `path::Symbol` and
every "no existing X" / "a sibling path already does Z" against actual source
(CONFIRMED / REFUTED / NOT-FOUND), and check the inverse: if this RDR adds a new
discriminator/heuristic/switch/identity rule, grep whether a sibling path
already makes that decision. Do not take the RDR's word for a codebase fact.

Step 1: Draft 12 verification questions a skeptical senior reviewer would ask to
catch errors, contradictions, or under-specified behavior — concrete yes/no or
concrete-value answers. AT LEAST 6 must be answerable only by reading source
(e.g. "does F already handle case C?", "is the switch in W exhaustive over the
node enum?", "does sibling path P already compute this?").

Step 2: Answer each question *independently*, without referring back to earlier
answers. A CODEBASE question MUST cite source you actually read (a greppable
`path::Symbol` or pasted line) — "the RDR says so" is not valid. An
RDR-internal question cites the exact RDR passage or says "RDR is silent."

Step 3: From Step 0 + Step 2, list as separate findings: (a) a codebase claim
REFUTED or NOT-FOUND; (b) a new rule with an existing sibling (cite it); (c) an
internal contradiction; (d) a "RDR is silent" revealing a missing requirement.
A false-against-codebase assumption is the highest-value finding — surface it
even if it reopens the chosen approach.

Write the findings to {EVIDENCE_DIR}/findings.md (the dispatcher binds {EVIDENCE_DIR} —
do not re-derive it). Report nothing else.
```

## Expected signal

- **Healthy** — Step 0 grounds every codebase claim (CONFIRMED with a cite, or
  REFUTED/NOT-FOUND surfaced); Step 3 surfaces 2–4 concrete findings, some
  source-grounded.
- **Unhealthy** — Step 0 answers codebase questions from the RDR not source, or
  finds zero silences (hallucinated coverage). Switch model and rerun; if it
  still won't read source, escalate to a human-run grep.

## Source

Dhuliawala et al., *Chain-of-Verification*, ACL Findings 2024 —
<https://arxiv.org/abs/2309.11495>. The independence discipline (Step 2)
defeats the LLM's tendency to agree with a plausible spec. The Step 0 grounding
rationale and its evidence base are in [`RESEARCH.md`](../../RESEARCH.md) §2
(Grounding).
