# Stage 2 — Propose

**Goal**: move from a problem statement to a *chosen approach* with the
seriously-considered alternatives recorded and rejected — so the draft has one
coherent solution for Refine to tighten.

## Paste this

Fill `{RDR_PATH}` (the RDR file); resolve `{RDR_RESOURCES}` (the project's
evidence index) via the workspace marker (see README *Where the seam lives* —
`. "$WS/.rdr-workspace"` exports `$RDR_RESOURCES`).

**Prompt**: [`../prompts/stages/02-propose.prompt.md`](../prompts/stages/02-propose.prompt.md)
— the `/rdr-propose` skill runs it. Paste its body by hand if driving without
the skill.

**Run when**: the seeded draft has a problem statement but no chosen approach
(or the approach is asserted with no alternatives weighed) — including
**re-entering a seed that sat idle** (the freshness check in step 0 is the
resume point). **Produces**: the RDR's *Proposed Solution* + *Alternatives
Considered* + *Decision Rationale* + the design-body (*Investigation* +
*Implementation Plan* phases at design altitude) + a `Pending` Critical
Assumptions list, written into the draft.

## Review gate

- **Was the seed still current?** Confirm step 0 ran — stale references/scope
  fixed, and any "retained as history"/"Refinement Context" note from a prior
  pass acted on and deleted. A proposal on a stale premise is wrong even if
  internally sound.
- **Did the accretion gate run?** If `Seam Lineage` carries ≥2 prior point-fixes,
  the *undecided contract* must be named **before** any mechanism is chosen and the
  profile floored to `foundational` (unless a written accretion disposition escapes
  it). Proposing an Nth mechanism without naming the missing decision is the failure
  this gate exists to stop — re-run.
- **Was the sibling-path check exhibited?** For any new discriminator, heuristic,
  switch case, or identity rule, confirm a grep for an existing sibling path was
  *shown* (a `path::Symbol`, or "searched, none exists") — not asserted.
- **Were real alternatives weighed?** One option, or strawmen, means the
  choice isn't earned — re-run asking for distinct, defensible options.
- **Was the breadth bias honored?** Approaches must be grounded in the Domain
  priors' prior art, not invented from the model's training prior. If the priors
  name no competitor for this problem class, confirm the response either carried
  the `⚠ no prior-art coverage` line into Investigation (`small`/`mid`) or did the
  forced bounded prior-art search and cited it (`large`/`foundational`). A clean
  option set with no prior art *and* no warning is the silent-invention failure
  this gate stops — re-run.
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
- **Is the design-body authored, at altitude?** Investigation and the
  Implementation Plan phases are real, not placeholders — but phase intent,
  not code. (Testing Strategy / Performance Expectations stay for Resolve.)

Re-run if the approach changes. If you only disagree with rationale *wording*,
fix inline — don't re-run.

## Advance when

One approach is chosen, alternatives are recorded with rejection reasons, the
Critical Assumptions list exists (Pending is fine), and the design-body
(Investigation, Implementation Plan) is authored — no `_Draft placeholder._`
left in those two sections. If the accretion gate fired (≥2 prior point-fixes),
the undecided contract is named and the profile is floored to `foundational`.

→ Next: [03-refine.md](03-refine.md)
