# Repeatability — Under-Specified Signature Probe (G-08)

**Use when**: single-RDR pre-lock, for RDRs that lock a public API, signature,
or data model an implementer must reproduce exactly — or when the Stage 5
**Determinacy trigger** fires on an algorithmic contract (step-ordering,
parse/deparse, import/export, compose/decompose, hashing, identity, migration,
multi-step MVV fidelity). Apply after the draft is complete **and its assumptions
are verified** (Stage 4). One of the single-RDR lenses; peers with
[3amigo](1-3amigo.md), [critique](2-critique.md), and [cove](4-cove.md).

**The variant is fixed by profile, not by which `run-*.md` already exist.** At
`mid`/`large` the determinacy default is the **lite variant** below (one
alternate-model run, `run-1` only); the full ×3 lens is for Foundational or on a
recorded escalation. At `mid`/`large`, do not write `run-2`/`run-3` — that silently
promotes the RDR to full; escalate deliberately (below) or diff `run-1`.

**What it uniquely catches**: under-specified signatures, types, and APIs.
Where multiple generation runs disagree, the RDR is silent — that silence is
the finding.

**Cost**: 15 min × 3 runs (one session each) + 10 min for a 4th `diff` pass in a
fresh session.

## Repeatability-lite (one alternate-model reconstruction)

The default when the Stage 5 Determinacy trigger fires on a `mid`/`large` RDR —
the full ×3 lens stays gated to Foundational or escalation.

**Cost**: 15 min × 1 alternate-model run + 10 min focused diff (vs. 55 min full).

**Run**: one reconstruction on a **different base model** (the cross-model draw is
the whole point — same generation prompt below, a single `run-1.md`), then a
focused diff against the RDR in a fresh session. Lite never runs `run-2` or
`run-3` unless it escalates to the full lens.

**The lite diff must name concrete RDR silences, not style differences.** A finding
is admissible only if it points at a specific algorithmic contract the RDR left
under-determined — a step whose order the RDR doesn't fix, a field whose owner it
doesn't name, a transform whose tie-break it doesn't state. Naming, wording, or
helper-decomposition taste is **not** a finding (the full lens's three-way
disagreement count isn't available at lite, so each silence must be concrete on its
own).

**Escalate to the full ×3 lens when**: the single alt-model run *agrees* with the
RDR but you can't tell determinacy from shared-model overfit (suspiciously clean, no
GUESS markers); **or** the lite diff surfaces a determinacy silence on a
**load-bearing or cross-RDR** contract (a one-run sample under-powers a contract
peers depend on); **or** the profile is already Foundational (full is the floor
there). **To escalate, rewrite `run-1.md`'s `variant:` line to `full (escalated:
<reason>)`** — that recorded line is what makes the later `run-2`/`run-3` legitimate
rather than a mismatch. Otherwise the lite diff is sufficient and feeds Resolve like
any lens folder.

## Generation prompt

**Run 3 times, one run per fresh session** — at least one run on a **different
base model**. The cross-model run is the point: it surfaces where the RDR reads
differently to a different model, not just a different temperature draw.

**One session writes exactly one run.** A context window that authored a run
must not write a second `run-*.md` — the next run goes in a fresh session so
each draw is independent. A sub-agent **cannot** stand in (self-consistency
check). After writing `run-<N>.md`, stop — the next invocation in a fresh
session writes the next one. Relaunch on the alt model between runs (e.g.
`ollama launch claude --model kimi-k2.6:cloud`); if `{RDR_RESOURCES}` lists
an alt-model roster, use one of its commands for run N.

`{RDR_PATH}`, `{EVIDENCE_DIR}` (this lens's `<rdr-slug>/evidence/repeatability/`
folder under `$RDR_EVIDENCE` — `{RDR_ENV}` defines), and `<N>` (this run's
number, 1–3) are bound by the Stage 5 arg header — paste verbatim. Running
standalone (outside the flow)? Read `{RDR_ENV}` for the base and fill them
yourself first.

```text
Based only on the RDR at {RDR_PATH}, produce:

  1. The module's public API (function signatures, types, error modes).
  2. The three most important internal helper functions and their
     responsibilities.
  3. The data model for anything persisted or passed across the boundary.
  4. The top-level pseudo-code of the main operation, 20–40 lines.

Do not ask clarifying questions. Do not say "it depends." Make your best
guess where the RDR is silent, and *mark each guess explicitly as a GUESS*.

Begin run-<N>.md with two header lines: `model: <base model you are running on>`
then `variant: <lite|full> (profile: <RDR profile>)` — the variant resolved from
the profile before this run (lite for mid/large, full for foundational/escalation).
Write your output to {EVIDENCE_DIR}/run-<N>.md. Report nothing else.
```

## Diff prompt

Run the diff as its own pass (`/rdr-prelock NNNN repeatability diff`) **in a
fresh session that authored none of the runs**. Full repeatability diffs three
runs; repeatability-lite diffs the single alternate-model `run-1.md` against the
RDR and admits only concrete contract silences. The analysis is pure, but an
authoring session isn't neutral — it diffs toward the interpretation it already
wrote — so the full diff is a fourth pass, and the lite diff is a separate neutral
pass (not a tail on the generation). A sub-agent of an authoring session can't
stand in either. Once the required runs exist and this session wrote none, paste
the matching prompt once. `{RDR_PATH}`/`{EVIDENCE_DIR}` bind from the Stage 5 arg
header — paste verbatim, or fill them if standalone.

For repeatability-lite, read `{EVIDENCE_DIR}/run-1.md`; compare it against the RDR
at `{RDR_PATH}` and write `{EVIDENCE_DIR}/diff.md` with only concrete contract
silences: missing step order, field owner, transform tie-break, error mode, or
similar determinacy gap. This is not a same-model consistency sample and not a
three-run disagreement count. Do not report naming, wording, or
helper-decomposition differences.

For full repeatability:

```text
Read the three repeatability runs at {EVIDENCE_DIR}/run-1.md, run-2.md, run-3.md
(each starts with its `model:` + `variant:` header lines; `run-1`'s `variant:`
records the intended variant — if it reads `lite`, extra runs are a mismatch to
flag, not full coverage). Compare them against the RDR at
{RDR_PATH} and write {EVIDENCE_DIR}/diff.md with:

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

`diff.md` is the file the cycle's fix half
([`$RDR_HOME/stages/05-prelock.md`]($RDR_HOME/stages/05-prelock.md), *Resolve (the fix half)*) reads.

## Expected signal

- **Healthy** — the diff localizes to 3–5 specific interfaces; `GUESS` markers
  cluster on the same under-specified contracts across runs.
- **Unhealthy** — the three runs are identical-but-confidently-wrong (overfit to
  a single interpretation, no `GUESS` markers). Switch model and rerun.

## Source

Fowler, *Understanding Spec-Driven Development* (the non-determinism /
regenerate-and-compare experiment) —
<https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html>.
