# Chain-of-Verification — Silence & Contradiction Probe (S-04)

**Use when**: single-RDR pre-lock, on any non-trivial RDR. Apply after the
draft is complete **and its assumptions are verified** (Stage 4). One of the
single-RDR lenses; peers with [3amigo](1-3amigo.md), [critique](2-critique.md),
and [repeatability](3-repeatability.md).

**What it uniquely catches**: silences ("the RDR is silent on this") and
internal contradictions — the gaps a reviewer only finds by answering concrete
questions independently rather than re-reading agreeably.

**Cost**: 15–20 min. **Dual-model recommended**.

## Prompt

```text
Step 1: Draft 12 verification questions about the RDR at {RDR_PATH} —
questions a skeptical senior reviewer would ask to catch errors,
contradictions, or under-specified behavior. Aim for questions that would
have concrete yes/no or concrete-value answers.

Step 2: Answer each of your 12 questions *independently*, without referring
back to your earlier answers. For each answer, cite the exact RDR passage
you relied on, or say "RDR is silent on this."

Step 3: Review your own answers for internal contradictions and for places
where a "RDR is silent" answer reveals a missing requirement. List each
contradiction or silence as a separate finding.

Do not skip Step 2's independence discipline — that is the whole point.

Write the Step 3 findings list to {FLOW_DIR}/findings.md ({FLOW_DIR} is
already bound by the dispatcher to this lens's `cove/<rdr-slug>/` folder
under the base {RDR_ENV} defines — do not re-derive it) — that is the file
the fix pass reads. Report nothing else.
```

## Expected signal

- **Healthy** — surfaces 2–4 concrete silences or contradictions, each anchored
  to a passage (or an explicit "silent on this").
- **Unhealthy** — finds zero silences (the model hallucinated coverage rather
  than answering independently). Switch model and rerun.

## Source

Dhuliawala et al., *Chain-of-Verification*, ACL Findings 2024 —
<https://arxiv.org/abs/2309.11495>. The independence discipline (Step 2)
defeats the LLM's tendency to agree with a plausible spec (Mitani 2025).
