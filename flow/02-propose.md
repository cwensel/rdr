# Stage 2 — Propose

**Goal**: move from a problem statement to a *chosen approach* with the
seriously-considered alternatives recorded and rejected — so the draft has one
coherent solution for Refine to tighten.

## Paste this

Fill `{RDR_PATH}` (the RDR file) and `{RDR_RESOURCES}` (the project's evidence
index — default `_rdr/rdr-resources.md`). Both resolve from the project root.

```text
Make a proposal for the RDR at {RDR_PATH}. See the rdr README for what an RDR
is and the process. Read {RDR_RESOURCES} — its *Domain priors* section
(principles, feature surface, user mental model, prior-art/competitor bias) —
to ground WHICH approaches are on the table; do NOT load the full verification
corpora or source paths, Resolve does that.

Read the Problem Statement and Context, then:

0. **Freshness check.** A seed may have sat idle and gone stale. Re-validate
   everything it names against the current world — references, paths, identifiers,
   peer-RDR statuses, and the scope itself — and fix what drifted in place. If a
   prior pass left a "Refinement Context" /
   "retained as history" / "folded into the body" note, act on any live "when
   you revise…" instruction it still carries, then delete the note (Draft =
   rewrite in place). If this invalidates the Problem Statement itself, stop and
   flag it — don't propose onto a false premise.

1. Enumerate the 2–4 genuinely distinct approaches that solve the stated USER
   OUTCOME (not just the mechanism). For each: one-paragraph description,
   Pros, Cons. Where prior art exists (the competitors {RDR_RESOURCES} names),
   weigh against how they treat it — align or deliberately diverge, and say
   which in the rationale.
2. Recommend ONE. State the decision rationale — the key factors and why each
   rejected alternative was rejected (one sentence for trivial ones).
3. **Premortem the chosen approach (one paragraph).** Assume this approach
   shipped and failed in production; write the one-paragraph post-mortem, then
   confirm the recommendation survives it. If the post-mortem exposes a failure
   the chosen approach can't answer, revise the choice now — it is far cheaper
   to switch approaches here than after Resolve has spent its spike budget. This
   is the approach-level gut-check, distinct from the lock-time hostile
   premortem in Stage 5 Critique; it runs on every profile because Propose
   always does.
4. Write the recommended approach into Proposed Solution and the alternatives
   into Alternatives Considered. Keep Technical Design at the level the
   problem needs — do NOT over-specify signatures yet; that sharpens during
   Resolve and Pre-Lock.

Surface, but do not resolve, the load-bearing assumptions the chosen approach
depends on — list them in Critical Assumptions as Status: Pending with a
one-line "If wrong". Stage 4 verifies them.

Be brief but not lossy; drop into ultrathink if the design gets complex.
```

**Run when**: the seeded draft has a problem statement but no chosen approach
(or the approach is asserted with no alternatives weighed) — including
**re-entering a seed that sat idle** (the freshness check in step 0 is the
resume point). **Produces**: the RDR's *Proposed Solution* + *Alternatives
Considered* + *Decision Rationale* + a `Pending` Critical Assumptions list,
written into the draft.

## Review gate

- **Was the seed still current?** Confirm step 0 ran — stale references/scope
  fixed, and any "retained as history"/"Refinement Context" note from a prior
  pass acted on and deleted. A proposal on a stale premise is wrong even if
  internally sound.
- **Were real alternatives weighed?** One option, or strawmen, means the
  choice isn't earned — re-run asking for distinct, defensible options.
- **Does the chosen approach solve the *user's* problem**, not an adjacent
  one? (The PM lens, surfaced early.)
- **Did the premortem run, and did the choice survive it?** A premortem that
  found nothing on a non-trivial approach is a tell it was skipped — the
  cheapest high-yield check in the literature (Klein 2007) shouldn't come back
  empty. If it forced a switch, confirm the new choice is the one written in.
- **Are the load-bearing assumptions named** (even if unverified)? Stage 4
  needs this worklist.
- **Is Technical Design proportionate** — enough to commit, not a full
  implementation? Over-specifying here wastes effort Refine/Resolve reworks.

Re-run if the approach changes. If you only disagree with rationale *wording*,
fix inline — don't re-run.

## Advance when

One approach is chosen, alternatives are recorded with rejection reasons, and
the Critical Assumptions list exists (Pending is fine).

→ Next: [03-refine.md](03-refine.md)
