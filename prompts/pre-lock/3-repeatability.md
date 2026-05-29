# Repeatability — Under-Specified Signature Probe (G-08)

**Use when**: single-RDR pre-lock, for RDRs that lock a public API, signature,
or data model an implementer must reproduce exactly. Apply after the draft is
complete **and its assumptions are verified** (Stage 4). One of the single-RDR
lenses; peers with [3amigo](1-3amigo.md), [critique](2-critique.md), and
[cove](4-cove.md).

**What it uniquely catches**: under-specified signatures, types, and APIs.
Where multiple generation runs disagree, the RDR is silent — that silence is
the finding.

**Cost**: 15 min × 3 runs + 10 min to diff.

## Generation prompt

**Run 3 times in fresh sessions** — at least one run on a **different base
model**. The cross-model run is the point: it surfaces where the RDR reads
differently to a different model, not just a different temperature draw. A
sub-agent **cannot** stand in for this — it inherits this session's model, so
it would weaken the probe to a self-consistency check. Relaunch the CLI on the
alt model yourself between runs (e.g.
`ollama launch claude --model kimi-k2.6:cloud`); if `{RDR_RESOURCES}` lists an
alt-model roster, use it and one of its commands for run N.

`{RDR_PATH}`, `{FLOW_DIR}` (`_rdr/repeatability/<rdr-slug>/`), and `<N>` (this
run's number, 1–3) are bound by the Stage 5 arg header — paste verbatim. Running
standalone (outside the flow)? Fill them yourself first.

```text
Based only on the RDR at {RDR_PATH}, produce:

  1. The module's public API (function signatures, types, error modes).
  2. The three most important internal helper functions and their
     responsibilities.
  3. The data model for anything persisted or passed across the boundary.
  4. The top-level pseudo-code of the main operation, 20–40 lines.

Do not ask clarifying questions. Do not say "it depends." Make your best
guess where the RDR is silent, and *mark each guess explicitly as a GUESS*.

Begin run-<N>.md with one line: `model: <base model you are running on>`.
Write your output to {FLOW_DIR}/run-<N>.md. Report nothing else.
```

## Diff prompt

After all three runs exist, paste this once (any session — pure analysis, no
independence requirement, so a sub-agent is fine here). `{RDR_PATH}`/`{FLOW_DIR}`
bind from the Stage 5 arg header — paste verbatim, or fill them if standalone.

```text
Read the three repeatability runs at {FLOW_DIR}/run-1.md, run-2.md, run-3.md
(each starts with its `model:` line). Compare them against the RDR at
{RDR_PATH} and write {FLOW_DIR}/diff.md with:

  1. Disagreements — every interface, type, error mode, data-model field, or
     pseudo-code step where the three runs differ. For each: the contract in
     question, how each run rendered it, and the RDR passage (or silence) that
     let them diverge. A disagreement that splits along the model boundary
     (the alt-model run vs. the others) is a stronger signal — flag it.
  2. GUESS clusters — contracts two or more runs marked GUESS. These are the
     candidate RDR rewrites.
  3. Agreement — one line noting interfaces all three rendered identically
     (not findings; they confirm the RDR is determinate there).

Order findings by how many runs disagree (3-way first). Each finding names the
RDR section it lands in — the **Normative Contracts** subsection of the
Technical Design is where most rewrites go. Report nothing else.
```

`diff.md` is the file Stage 6
([`../../flow/06-prelock-resolve.md`](../../flow/06-prelock-resolve.md)) reads.

## Expected signal

- **Healthy** — the diff localizes to 3–5 specific interfaces; `GUESS` markers
  cluster on the same under-specified contracts across runs.
- **Unhealthy** — the three runs are identical-but-confidently-wrong (overfit to
  a single interpretation, no `GUESS` markers). Switch model and rerun.

## Source

Böckeler, *Understanding Spec-Driven-Development* (on martinfowler.com; the
non-determinism / regenerate-and-compare experiment) —
<https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html>.
