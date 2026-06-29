# Critique — Devil's Advocate Premortem

**Use when**: an RDR locks an enum, hash, format, grammar, or destructive operation. Skip for purely-additive RDRs.

**What it uniquely catches**: time-shifted failures invisible at lock-time
review — frozen-at-lock invariants without version markers,
enums/grammars/formats that get reopened in 6 weeks.

**Cost**: ~20–30 min. **Dual-model, profile-graduated** — run in two sessions on
different base models and diff the outputs. Disagreement is the signal. A
sub-agent can't substitute: it inherits this session's model, so the second
run must be a CLI relaunch you do yourself. If `{RDR_RESOURCES}` lists an
alt-model roster, use it and its launch command for the second model (e.g.
`ollama launch claude --model kimi-k2.6:cloud`); otherwise any second base
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

Never conclude "already complete" from an existing `critique.md` alone. Read its
model stamp (rdr-common §model-stamp) and compare to this session's base model:

- **Different model / no stamp** → the dual-model second pass. Write
  `critique-modelB.md` (stamped this session) and diff the two; the disagreement
  is the signal to resolve. (No stamp = unknown, so don't assume a match.)
- **Same model** → redundant repeat; short-circuit is fine, but say so ("prior run
  was the same base model `<id>`") and point at the alt-model roster.

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

Hostile critique is the assignment. If you find yourself softening,
restart.

Write the critique to {EVIDENCE_DIR}/critique.md ({EVIDENCE_DIR} is already bound
by the dispatcher to this lens's `critique/<rdr-slug>/` folder under the
base {RDR_ENV} defines — do not re-derive it). Open the file with the `Model:`
stamp (rdr-common §model-stamp). On a dual-model run, the second model writes
critique-modelB.md alongside it (its own stamp), and you diff the two.
```

## Prompt — whole RDR set (higher-leverage variant)

Running Critique against the *whole set* of RDRs surfaces cross-cutting failures
earlier than per-RDR critique. Use this when several locked-surface RDRs are
batched for review:

```text
Fresh context. You are a senior engineer who has seen many projects like
this one fail. You have been asked to review the RDR set under
{RDR_RECORDS}, and you believe the project will fail.

[Same five-step structure as above, but Step 1 names the most likely
inter-RDR failure mode; Step 2 names the one RDR that will be rewritten;
Step 3 names the cross-cutting assumption that will not survive.]
```

## Expected signal

- **Healthy** — the critique names concrete passages, concrete functions, concrete
  user journeys; the Gherkin tests map cleanly to specific lines.
- **Unhealthy** — generic advice ("consider adding more tests"); no named
  passages; abstract user journeys. Switch model and rerun.

## Source

Klein 2007, *Performing a Project Premortem* —
<https://hbr.org/2007/09/performing-a-project-premortem>. Combined with the
DEBATE Devil's Advocate pattern and the Mitani 2025 LLM-sycophancy caveat (run
dual-model and diff). Adapted from the corresponding spec-fitness battery prompt
(placeholder change only).
