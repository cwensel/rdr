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
Assumptions list, written into the draft — plus the **premortem verdict**
(`Premortem: survived | hardened | switched`, variant named), written into
Decision Rationale, the **ground-sweep verdict** (`Ground-sweep: clean |
reopened`, beside it), and the **joint-decision check's verdict** (fired on a
peer / no fire), all reported in the close packet.

The joint-decision check is propose-time *triage*, not reconciliation: a cheap
lexical pass at the one moment switching is free. Stage 7.1's
tolerate-then-reconcile doctrine stands — cross-RDR inconsistency is still
tolerated during per-RDR work and reconciled semantically once, per cluster
(see [07.1](07.1-cluster-reconcile.md) *Why a gate, not a per-RDR step*, and
its "Related" definition under *When it fires*). This check only stops two
open RDRs from silently coupling on the same decision before either locks.

**Order a seeded batch breadth-first: propose every sibling before any
refines.** The check collects anchors from *this* RDR's written proposal and
compares them against peers' — a bare seed has no anchors to collide with, so
depth-first (one RDR seed→Final at a time) blinds the early checks and lands
any late fire against locked text, where the fix is a recorded tolerance
instead of free switching. The tandem barrier enforces this order for declared
Clusters; extend the same discipline to any batch of sibling seeds. The
`/rdr-joint-propose` skill mechanizes this: dependency-ordered sequential
proposes, orchestrator-handled re-orders when a propose blocks on an unproposed
peer, and only the true joint forks surfaced to the user.

## Review gate

- **Was the seed still current?** Confirm step 0 ran — stale references/scope
  fixed, and any "retained as history"/"Refinement Context" note from a prior
  pass acted on and deleted. A proposal on a stale premise is wrong even if
  internally sound.
- **Did the accretion gate run?** When `--outcome floor` (§lens-row's call)
  answers `foundational`, the *undecided contract* is named **before** any
  mechanism is chosen and Profile reads `foundational`. Proposing an Nth mechanism
  without naming the missing decision is the failure this gate exists to stop — re-run.
- **Was the sibling-path check exhibited?** For any new discriminator, heuristic,
  switch case, or identity rule, confirm a grep for an existing sibling path was
  *shown* (a `path::Symbol`, or "searched, none exists") — not asserted.
- **Did the joint-decision check run, and was any fire paused on?** The prompt's
  final step has **three arms** — shared modify-anchors, shared contract
  literals, and the absence arm — and records a `Joint-check:` line in Decision
  Rationale either way. Name them separately: a check reported as "run" having
  fired only one arm is two arms skipped, reading as passed.

  **Arms 1 and 2 are queries** — one pass each, no peer body opened:

  ```sh
  "$RDR_HOME/bin/rdr" index --anchor-intersect --json   # arm 1: modify-anchors
  "$RDR_HOME/bin/rdr" index --literal-intersect --json  # arm 2: contract literals
  ```

  Both emit `overlaps[]` `{records[], anchors[], cited}`, in-flight and uncited
  first — the fire shape — so one reading serves both. Arm 1 needs a repo root:
  `--repo` defaults to `$RDR_SOURCE_REPO` (rdr-common §source-root), and
  **without one, source-anchor edges carry no `resolved` key at all** (absent ≠
  false); an unchecked scan must not read as "no intersection". Arm 2 reads the
  literals inside `C` elements and needs no repo. `cited: true` is context
  beside a fire, never suppression.

  Do **not** read `index --json` for this: the whole graph is ~5.6 MB where
  `--literal-intersect` answers in ~8 KB. Nor does the element `hash` answer arm
  2 — it is exact-text identity, and measured over the live corpus **no two
  contracts share one** (0 of 316). The arm asks about a shared *token* inside
  otherwise-different contract text, which is what the facet reports.

  **Arm 3, the absence arm, stays manual** and is not convertible. It greps
  `Final` peers for a refusal token *because that token is absent from the new
  proposal* — the query needs the proposal that was just written, which no
  corpus projection has. A tool can intersect what two records both say; it
  cannot see what one of them stopped saying. Run it by hand per the prompt, and
  do not read its absence from the two queries above as a clear.

  No `Joint-check:` line → it did not run, re-run it (a skipped gate item must
  not read as a passed one). A fire PAUSES propose before refine, as a user
  question — advanced over silently → re-run. A fire clears only onto a
  resolvable §-anchor `home:` or `OPEN` — prose naming no authority does not;
  check it as an edge, not a string: `edges[]` `kind=="joint-decision-home"`
  needs `resolved: true` (`false` or **absent** is not a pass).
- **Was the bridge question surfaced and answered when its cues were present?**
  Behind the tandem barrier, a plan-introduced surface a Cluster sibling
  schedules for deletion puts the (a) skip-to-end-state / (b) `Transient`-marker
  choice to the user (mechanics in the prompt's sub-check). An unanswered
  bridge choice → re-run.
- **Were real alternatives weighed?** One option, or strawmen, means the
  choice isn't earned — re-run asking for distinct, defensible options.
- **Was prior art read *before* the approaches?** Order matters — an LLM that
  enumerates first anchors on its training prior. Confirm the Investigation shows a
  prior-art read that *grounds* the set, with a `⚠ no prior-art coverage` line if
  none was found. Approaches grounded in nothing, unmarked, is the silent-invention
  failure this gate stops — re-run.
- **Did the cite-check hold?** Every prior-art claim the choice rests on is
  quoted/anchored from source (named system + section, or `path::Symbol`), not
  paraphrased — an inverted citation is what overturned a chosen approach three
  stages late. Unanchored load-bearing claim → re-run or demote to a Resolve
  assumption. Each load-bearing citation carries its `⇒` consequence line, and
  a cited state-read names its writer — a true fact left uncomposed is the
  same late-overturn class.
- **Did the grounding micro-sweep run?** A `Ground-sweep:` verdict line sits
  beside `Premortem:` (absence = it did not run — re-run). A load-bearing
  REFUTED/NOT-FOUND advanced over instead of re-choosing is the late-overturn
  failure the sweep exists to stop.
- **`large`/`foundational`: scored matrix, not prose?** The choice falls out of an
  explicit Q-O-C matrix (approaches × deciding criteria). Prose-only → re-run.
- **Does the chosen approach solve the *user's* problem**, not an adjacent
  one? (The PM lens, surfaced early.)
- **Did the premortem run in its profile's form, and is the verdict recorded?**
  `small`/`mid` write the one-paragraph post-mortem; `large`/`foundational`
  dispatch the draft-free critic (brief-only input, queried once) whose `P-N`
  ledger lands in `evidence/propose-premortem/critic.md`. Either form ends
  with the `Premortem:` verdict line in Decision Rationale. A premortem that
  found nothing on a non-trivial approach is a tell it was skipped — the
  cheapest high-yield check in the literature (Klein 2007) shouldn't come back
  empty; for the hardened form, an empty ledger on a foundational approach is
  the sycophancy tell — re-run with the seeds emphasized. If it forced a
  switch, confirm the new choice is the one written in.
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
Critical Assumptions list exists (Pending is fine), the `Premortem:` verdict
line is recorded in Decision Rationale (any value — an absent verdict does
not advance), and the design-body
(Investigation, Implementation Plan) is authored — no `_Draft placeholder._`
left in those two sections. If `floor` answered `foundational`, the undecided
contract is named and Profile reads `foundational`.
The `Joint-check:` and `Ground-sweep:` verdict lines are recorded in Decision
Rationale (either absent does not advance — that check never ran). If it fired, the joint
decision has a named home — the consumer's umbrella-decision record (e.g. an
RFD), or the shared interface declared in both RDRs — an open fire does not
advance. If the RDR declares a
`Cluster`, every member has completed propose before any member advances to
refine (the tandem barrier — see TEMPLATE.md `Cluster`), and an open bridge
choice — (a) skip to end-state / (b) `Transient` marker — does not advance.

**Or park it.** If Propose ran to completion and returned **no acceptable
mechanism** — every in-our-control path is ruled out on a stated ground, and
the one that would work is outside our control — do not force a choice and do
not abandon. Set `Status: Deferred [revisit when <condition>]`, name in the
same field what was ruled out and why, and leave Alternatives Considered
carrying the long form. The RDR is parked: it locks no contract, owes no
post-mortem, keeps its trackers, and re-enters **here** when the condition
fires. This is a high bar — "possible but unfunded" is a `Priority`, not a
`Deferred`; the test is that no path exists, not that none is scheduled.

→ Next: [03-refine.md](03-refine.md)
