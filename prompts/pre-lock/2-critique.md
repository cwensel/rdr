# Critique — Devil's Advocate Premortem

**Use when**: an RDR locks an enum, hash, format, grammar, or destructive operation. Skip for purely-additive RDRs.

**What it uniquely catches**: time-shifted failures invisible at lock-time
review — frozen-at-lock invariants without version markers,
enums/grammars/formats that get reopened in 6 weeks.

**Cost**: ~20–30 min. **Dual-model, profile-graduated** — run the two passes on
different base models and diff the outputs. Disagreement is the signal. A
sub-agent *can* carry a second model where the harness pins models per spawn
(`/rdr-prelock … critique --auto` runs both in parallel — rdr-common
§auto-fanout); what it can't reach is a **non-Anthropic/open-weight** model, so a
cross-vendor draw is still a CLI relaunch you do yourself. If `{RDR_RESOURCES}`
lists an alt-model roster, use it and its launch command for the second model
(e.g. `ollama launch claude --model kimi-k2.6:cloud`); otherwise any second base
model works.

- **`foundational`** — dual-model is **required to converge** (or the recorded
  single-model fallback below if no alt model is reachable). A lone single-model
  `critique.md` with no diff does **not** complete the lens.
- **`large`** — dual-model is **strongly recommended**. Single-model fallback is
  acceptable, but it must be **recorded as a fallback** (note "single-model only,
  no alt model reachable" in `critique.md`), not silently passed off as a full
  dual-model pass.

**Single-model fallback** (when a second model isn't available): run the
critique twice in *fresh* contexts and diff. Fresh context with no prior
answers recovers most of the dual-model anti-sycophancy benefit — the
independence discipline of Chain-of-Verification (Dhuliawala 2024), applied to
defeat the LLM's tendency to agree with a plausible spec (Mitani 2025). Cheaper
than a second model; do not let the absence of one skip the anti-sycophancy
step entirely.

## Re-entry — new model, or repeat?

Never conclude "already complete" from an existing `critique.md` alone. **Ask
`--outcome critique`** (rdr-common §lens-row's call, that outcome): it reads both
passes' `Model:` stamps (§model-stamp) and answers `none` when the lens is
finished, or `/rdr-prelock critique` with the reason — no second pass, or two
passes never diffed. Its `surface` line is the caveat to carry: a single-model
fallback, or a pass with no stamp, where the cross-model claim is unproven.

What stays this prompt's: when a second pass IS owed, write `critique-modelB.md`
stamped with this session's model and diff the two — the disagreement is the
signal to resolve. Where the prior run was the same base model, say so ("prior
run was the same base model `<id>`") and point at the alt-model roster.

**This lens reads the record whole, by design.** The other pre-lock lenses take
line ranges from the projector and read spans; critique cannot. "The one section
rewritten within 6 weeks" and "the assumption that will not survive first
contact" are judgements about the whole frame — the Alternatives, the Trade-offs,
the Decision Rationale and the Context are the evidence, and a scoped read would
delete exactly the prose the premortem argues from. Only the ledger's anchors are
projected (below).

## Prompt — single RDR

```text
Fresh context. You are a senior engineer who has seen many projects like
this one fail. You have been asked to review the RDR at {RDR_PATH}, and
you believe it will fail.

Write the strongest possible critique. Do not hedge. Do not balance.

Structure:

1. The three most likely ways implementation goes wrong. For each: the
   root cause in the RDR, the specific passage that enabled it, and the
   symptom the user will see.

2. The one section that will be rewritten within 6 weeks of shipping,
   and why.

3. The one assumption in the RDR that will not survive first contact
   with a real user.

4. The premortem, written as if the failure already happened. One page.
   Names specific functions and specific user journeys. (This is a
   premortem authored at draft time — distinct from the real post-mortem
   the RDR README describes for after Close.)

5. Working backward from the premortem, the acceptance tests (in
   Gherkin or plain steps) that would have caught each failure at
   RDR-review time.

6. The findings ledger — one row per distinct defect raised in 1–5. The
   fix half resolves against this, so an unrowed defect is unreported.

   | ID | RDR passage | Failure mode | Symptom user sees | Origin |
   |----|-------------|--------------|-------------------|--------|

   - **ID** — `C-1`, `C-2`, … stable within this file.
   - **RDR passage** — MUST be the element id the projector addresses
     (`NNNN:C4`, `NNNN:A3`; `NNNN:§slug` for prose) — it survives a reword and
     it is the key a re-run's ledger diff (`rdr anchors`) compares by, so a
     quoted phrase is a row the diff cannot see; `"$RDR_HOME/bin/rdr" inspect
     {RDR_PATH}` lists them, one per line. "The RDR generally" is not one.
   - **Origin** — the section that raised it: `§1`/`§2`/`§3`/`premortem`/`AT-N`
     (may be several). Premortem and AT origins rank equal to the rest.

Hostile critique is the assignment. If you find yourself softening,
restart.

Write the critique to {EVIDENCE_DIR}/critique.md ({EVIDENCE_DIR} is already bound
by the dispatcher to this lens's `critique/<rdr-slug>/` folder under the
base {RDR_ENV} defines — do not re-derive it). Ledger first, under the `Model:`
stamp (rdr-common §model-stamp); 1–5 follow at full length — never compressed to
make room, since a row pointing into a truncated section resolves to nothing. On
a dual-model run, the second model writes critique-modelB.md alongside it (its
own stamp); diff the ledgers by passage anchor (IDs are per-file), then the prose.
```

## Prompt — whole RDR set (higher-leverage variant)

Running Critique against the *whole set* of RDRs surfaces cross-cutting failures
earlier than per-RDR critique. Use this when several locked-surface RDRs are
batched for review:

```text
Fresh context. You are a senior engineer who has seen many projects like
this one fail. You have been asked to review the RDR set under
{RDR_RECORDS}, and you believe the project will fail.

[Same six-step structure as above, but Step 1 names the most likely
inter-RDR failure mode; Step 2: if any RDR in the set will be rewritten,
name it and say why — if none will, say so and state what would have to
be true for one to be; Step 3 names the cross-cutting assumption that
will not survive. The Step 6 ledger gains an RDR column — each row names
the RDR it lands on, so the fix half can route rows to the right draft.]
```

## Expected signal

- **Healthy** — the critique names concrete passages, concrete functions, concrete
  user journeys; the Gherkin tests map cleanly to specific lines. Ledger rows carry a
  real anchor and span several origins — an all-`§1` ledger means the premortem and
  the ATs were written as ceremony, not analysis.
- **Unhealthy** — generic advice ("consider adding more tests"); no named
  passages; abstract user journeys; rows anchored to "the RDR", or a ledger missing
  defects the prose raises. Switch model and rerun.
- **Whole-set variant** — Step 2 naming no rewrite candidate is a reportable
  result, not a softened critique; both signals above apply to it unchanged.

## Source

Klein 2007, *Performing a Project Premortem* —
<https://hbr.org/2007/09/performing-a-project-premortem>. Combined with the
DEBATE Devil's Advocate pattern and the Mitani 2025 LLM-sycophancy caveat (run
dual-model and diff). Adapted from the corresponding spec-fitness battery prompt
(placeholder change only).
