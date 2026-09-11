Make a proposal for the RDR at {RDR_PATH}. See the rdr README for what an RDR
is and the process. Read {RDR_RESOURCES} — its *Domain priors* section
(principles, feature surface, user mental model, prior-art/competitor bias) —
to ground WHICH approaches are on the table.

This stage owns **selection** (which approach wins, read from prior art), not
**verification** (live spikes, exactness, full corpora — Resolve's, deliberately:
batch the expensive spikes once). The split is *depth, not avoidance*: read prior
art deeply enough to pick right, but stop short of the spike budget. A chosen
approach overturned three stages late on misread (inverted) citations is the
documented failure this guards — so read prior art *correctly and first*, here.

Read the Problem Statement and Context, then:

0. **Route-back check, first.** A `[routed back from <origin> <date>; re-verify
   <IDs> @propose — <reason>]` Status qualifier means a later stage refuted this
   proposal: run a **delta** per rdr-common §rdr-write *Receiving a route-back* —
   rework what it names, carry the rest, clear the qualifier, note it under
   `Deviations:`. Skip step 0 here: the record is refuted, not stale.

0. **Freshness check.** A seed may have sat idle and gone stale. Re-validate
   everything it names against the current world — references, paths, identifiers,
   peer-RDR statuses, and the scope itself — and fix what drifted in place. If a
   prior pass left a "Refinement Context" /
   "retained as history" / "folded into the body" note, act on any live "when
   you revise…" instruction it still carries, then delete the note (Draft =
   rewrite in place). If this invalidates the Problem Statement itself, stop and
   flag it — don't propose onto a false premise.

0.5 **Accretion gate.** Run rdr-common §lens-row's call, `--outcome floor`
   (it reads the `Seam Lineage` field's facts — do not re-read or count it).
   `raise` → this is a *missing design decision, not a missing patch*:
   before proposing any mechanism, write one sentence naming the **undecided
   contract** the prior point-fixes all danced around (Resolve sizes Profile
   one tier above it). Enumerate approaches as answers to that contract
   question, not as the next patch. `none` → skip this step; a `stopped:*` → surface it (§stop-packet).

1. **Read prior art FIRST — before naming any approach.** An LLM that enumerates
   first anchors on its training prior and won't reliably self-correct, so read
   before the candidate set exists, not as a bias applied after. Read the Domain
   priors' competitors/prior systems/standards for THIS problem class — and,
   when the problem names a specific operator/surface, ALSO ask the instance
   question (what does each named peer do for *this* operator), never the
   class question alone: the class read picks the frame, the instance read
   decides the disposition. On a
   `large`/`foundational` RDR, also do a bounded external pass (corpora or
   web/literature) — a read, not a spike. **Coverage:** found nothing? Write one
   Investigation line (`⚠ no prior-art coverage for <problem class>; approaches
   below are from the model prior`) — never leave the gap silent. **Cite-check:**
   every prior-art claim the choice will *rest on* must be quoted/anchored from
   source (named system + section, or `path::Symbol`), not paraphrased — a misread
   or inverted citation is the exact defect that overturned an approach three
   stages late. A claim you can't open and quote isn't load-bearing: demote it to a
   Resolve assumption rather than leaning the choice on it.
   Order/reachability/emission claims can never be quote-confirmed — always
   demote them (Method: Spike or MVV Test), including a rejection reason that
   rests on one.
   - **Prior-art search budget.** Stay bounded: ≤3 corpus queries per candidate
     problem-class claim, ≤5 opened hits total per claim; stop the moment a hit
     confirms or refutes the claim — do not sweep for completeness. Search via
     `arc search semantic --corpus <C> --limit N --json "<q>"` (current flag is
     `--corpus`, not the obsolete `--collection`); a flag-retry on the same query
     counts against the budget. Found nothing? Write a `⚠ no prior-art coverage`
     line — a negative result is a result, not a reason to widen. Record accepted
     citations + the queries that found them, and rejected branches, to
     {EVIDENCE_DIR}research/ (see Resolve); a re-run reads that file instead of
     re-searching.
2. **Now enumerate — grounded in step 1.** Name the 2–4 genuinely distinct
   approaches solving the stated USER OUTCOME (not the mechanism); for each: a
   one-paragraph description, Pros, Cons, and how it aligns with or diverges from
   the prior art. A missing approach is the costliest error here — Resolve only
   verifies approaches that made the list.
   - **`large`/`foundational` → scored matrix, not prose.** Build a
     Questions-Options-Criteria matrix (approaches × deciding criteria —
     correctness fit, prior-art alignment, reversibility, blast radius, cost; one
     clause per cell); the choice falls out of it. This is the multi-axis re-score
     the corpus only reached *at rework* — do it here, where switching is cheap.
   - **`small`/`mid` → prose Pros/Cons** suffices; the premortem is the backstop.
   (Profile is the Seed estimate here — use it; Resolve recounts later.)
3. Recommend ONE. State the decision rationale — the key factors and why each
   rejected alternative was rejected (one sentence for trivial ones); for a scored
   matrix, the rationale references the deciding rows.
4. **Premortem the chosen approach.** Assume it shipped and failed in
   production; confirm the recommendation survives, or revise the choice now —
   far cheaper to switch here than after Resolve has spent its spike budget.
   This is the approach-level check, distinct from the lock-time hostile
   premortem in Stage 5 Critique: it runs pre-body, on the approach alone (the
   hardened form on a brief), where Critique reads the full draft at lock.
   Either way, close by writing one verdict line into Decision Rationale —
   `Premortem: survived | hardened | switched (paragraph | hardened)` — closed
   vocabulary, variant named (the repeatability `variant:` precedent); that
   one greppable line is what makes per-form miss rates computable later.
   (Profile is the *current* value here — step 0.5 may have set it to
   `foundational` in-flight.)
   - **`small`/`mid` → one paragraph.** Write the one-paragraph post-mortem of
     the shipped-and-failed approach, then confirm the recommendation survives
     it. A failure the chosen approach can't answer → switch. Variant
     `(paragraph)`.
   - **`large`/`foundational` → hardened critic: ONE spawned sub-agent
     (rdr-common §delegation), queried ONCE** — never more critics or rounds (homogeneous
     debate converges, it doesn't dissent; one hardened critic queried once
     beats it at fewer tokens). The extension past "delegate reads" is honest:
     this delegates *judgment*, because a fresh context that never saw the
     justifying prose is the independence that defeats self-confirmation — the
     same CoVe discipline Critique already uses. It is NOT a dual-model pass —
     spawn it at the §model-ceiling resolution when the harness has per-spawn
     model control (these profiles are where the ceiling applies); without
     that control it inherits the session model (see `2-critique.md`).
     - **Seeds first (main agent, before dispatch).** Glob
       `$RDR_RECORDS/*-postmortem.md` at depth 1 (never recurse); from their
       Escaped-Defect Ledgers take 2–3 rows with `Expected-catching stage:
       2-propose` (then `3-refine`), preferring high Escape distance; fall
       back to the consumer's post-mortem `SYNTHESIS.md` if present. Nothing
       found → write `⚠ no escaped-defect ledger entries yet` into the brief
       and proceed — never a silent skip, never a blocker.
     - **The brief — independence enforced structurally, not by honour.** Pass
       ONLY: (a) the Problem Statement; (b) the chosen approach stated
       neutrally in 3–6 lines; (c) its load-bearing claims as an enumerated
       negation-target list, PLUS each rejected alternative's one-line
       rejection *reason* (claims to attack — an inverted rejection is a
       documented escape); (d) a one-line technical-environment summary;
       (e) the seeds. NEVER the RDR path, the QOC matrix, the Decision
       Rationale, or any justifying prose. The critic works from the brief
       alone — no repo reads, no RDR read.
     - **The critic's task** (four elements): (1) *prospective hindsight* —
       the approach shipped and failed; write the failure narrative as
       accomplished fact; (2) *obstacle negation* — negate each load-bearing
       claim; for each, a domain-consistent failure scenario the approach must
       answer; (3) *consumer artifacts* — for each failure, the concrete
       artifact that would have caught it at review time: a named test case or
       user journey, not an opinion; (4) *refutation targets* — the seeds are
       specs that survived their premortem and failed anyway; find that class
       of failure here before accepting anything.
     - **Output.** The critic writes
       `{EVIDENCE_DIR}propose-premortem/critic.md`: the rdr-common
       §model-stamp header, then a findings ledger in the critique lens's
       column form (`| ID | passage/claim | Failure mode | Symptom user sees |
       Origin |`, ids `P-N`, Origin = which of the four elements raised it),
       then the narrative/negations/artifacts at full length. It returns a
       rdr-common §return-packet exactly: survived → `PASS`; mitigations to fold →
       `PASS` with `next_action` listing them; forced switch →
       `NEEDS_DECISION`; `changed_paths` = the critic.md; `summary_50w` = the
       verdict's reason.
     - **Consequence (main agent, in-context).** Fold mitigations into the
       draft — Pending assumptions and Failure Modes; each unanswered scenario
       either forces the switch or becomes one. On `NEEDS_DECISION`, loop back
       to step 3 with the failure as a new criterion — in-stage, no user
       pause; the close packet reports it. Variant `(hardened)`.
5. **Sibling-path check.** If the chosen approach adds a new discriminator,
   heuristic, switch case, or identity rule, grep whether an adjacent/sibling
   path already makes that decision, and *exhibit* the result in the RDR (a
   pasted `path::Symbol`, or "searched, none exists") — reusing an existing
   signal beats inventing a parallel one.
6. Write the recommended approach into Proposed Solution and the alternatives
   into Alternatives Considered. Keep Technical Design at the level the
   problem needs — do NOT over-specify signatures yet; that sharpens during
   Resolve and Pre-Lock. Every load-bearing citation in the body — prior-art
   or `path::Symbol` — carries a one-line `⇒ <what this forces here>`; and a
   cited state-*read* (a filter, flag test, marker check) names its *writer*
   too (what sets that state, when) — a true fact cited without its
   consequence or its writer is the documented uncomposed-fact escape.
7. Author the **design-body** (replace its `_Draft placeholder._`s):
   - **Investigation**: the prior art, code paths, and constraints that
     shaped the choice — one paragraph; deep evidence lands at Resolve.
   - **Implementation Plan**: Prerequisites, the MVV, and Phase 1…N as
     **named phases with a one-line intent each** — the spine, not the
     steps. Stay at the "what": a phase that reads like code is
     over-specified; that detail fills at Resolve/Pre-Lock.
   Leave Testing Strategy and Performance Expectations as placeholders —
   Resolve owns those, after the assumptions they lean on are verified.
7.5 **Grounding micro-sweep — on the written proposal.** Delegate ONE
   fresh-context sub-agent (rdr-common §delegation): the brief carries ONLY
   the proposal's cited anchors — every backticked `path::Symbol` and every
   quoted peer-RDR passage — never the justifying prose (factored
   verification: the checker must not see the argument). It reads each
   anchor's source on `main` (or the peer RDR) and returns per anchor
   CONFIRMED | REFUTED (what it found instead) | NOT-FOUND, as one
   §return-packet. A load-bearing REFUTED / NOT-FOUND reopens the choice —
   loop to step 3 with the correction as a new criterion; cosmetic misses fix
   inline. Record the verdict beside `Premortem:`, closed vocabulary:
   `Ground-sweep: clean (N anchors) | reopened → <claim>` — unwritten = did
   not run. Stage 5's grounding lens then delta-scopes to claims added after
   this sweep.
8. **Joint-decision check — runs last, on the written proposal.** Catch two
   open RDRs coupling on one decision while switching is still free. It has
   **three arms**; run all three and say so, because a check that fired one arm
   and is reported as "run" is two arms skipped reading as a pass.
   **Arm 1 — modify-anchors** (mechanical): two open records proposing to change
   one symbol. **Arm 2 — contract literals** (mechanical): two open records
   naming one error/rule code, flag, field, exit code or sentinel inside their
   ```normative fences. Both are one query each, over the whole corpus, with no
   peer body opened, scoped to this RDR's rows (`<slug>` = its filename stem):

   ```sh
   "$RDR_HOME/bin/rdr" index --anchor-intersect --json --record <slug>   # arm 1
   "$RDR_HOME/bin/rdr" index --literal-intersect --json --record <slug>  # arm 2
   ```

   Each emits `overlaps[]` `{records[], anchors[], cited}`, in-flight by default
   and uncited pairs first. **FIRE** = `overlaps[]` is non-empty. Cross-citation
   is *context reported beside the fire*, never suppression — writing a citation
   is prose about the coupling, not a decision about it. Arm 1 needs `--repo`
   (defaults to `$RDR_SOURCE_REPO`); without a repo root its edges carry no
   `resolved` key and an unchecked scan must not read as "no intersection".
   Do not read `index --json` for either: the whole graph is ~5.6 MB against
   ~8 KB, and the element `hash` answers neither arm (it is exact-text identity;
   0 of 316 contracts share one).
   **Arm 3 — absence** (MANUAL; there is no query and there cannot be one).
   When the proposal converts a refusal into an acceptance (fills a
   previously-empty cell, removes a guard, classifies the
   previously-unclassified), grep `Final` peers for the refusal token itself
   (the error / `unclassified` / refused literal): a Final peer *relying on the
   refusal* is a FIRE. Open peers = `"$RDR_HOME/bin/rdr" status --json --filter status`
   `records[].path` (this record included is harmless: the token is absent
   from it by definition). This arm is not convertible: it searches for a token
   *because it is absent from the new proposal*, so it needs the proposal rather
   than the corpus — the intersection facets can only report what two records
   both say, never what one of them stopped saying. A clear on arms 1 and 2 is
   not a clear on this one.
   **Record the verdict in Decision Rationale — fire or not** — one greppable
   line beside `Premortem:`, closed vocabulary:
   `Joint-check: clear (N peers) | fired → NNNN[, NNNN] (home: <home> | OPEN)`.
   `<home>` is a resolvable `RFD NNNN`/`cli/NNNN` §-anchor where the decision is
   normatively settled (07.1's bar), else the literal `OPEN` — prose naming no
   authority is not a home, and unhomed is `OPEN`, not `clear`.
   **Fires are symmetric**: write the same fire (same home) into each named
   `Draft` peer's line — a peer already fired on may not later record `clear`.
   A `Final` peer is never edited (no-amend); its coupling rides to 7.1.
   Unwritten = *did not run*: absence reopens propose, since a later session
   can't tell a skipped gate item from a passed one.
   Both PAUSEs below walk rdr-common §ground-before-ask first.
   On fire, PAUSE — emit §stop-packet `stopped:joint-decision:<the joint
   question>` and WAIT. This is a human-judgment fork, not a gate failure to
   report and move past: name the peer RDR(s) and both answers — hoist to the
   consumer's umbrella-decision record (e.g. an RFD) as the single normative
   home, cite-don't-restate; or declare the shared interface in both RDRs.
   Don't choose for the user, and don't close as if the stage finished. A fire
   is detection, not demotion: siblings proceed under recorded tolerance once
   the joint decision has a home.
   **Bridge sub-check** — Cluster members only, behind the tandem barrier
   (`cluster_members_proposed=all`, so a scheduled deletion is decidable). When this
   RDR's Implementation Plan introduces a surface a sibling's plan schedules
   for deletion/replacement — cues `bridge`, `retire`, `replace` (stem match,
   in either plan; NOT `supersede`, which overwhelmingly claims lineage over
   prior closed RDRs and would false-fire), cross-referenced with Cluster
   membership and the joint decision's named home if one exists — PAUSE
   (§stop-packet `stopped:bridge-choice:<the choice>`) and WAIT on the user
   for: **(a) skip to end-state** — implement the
   sibling's replacement directly, or reorder shipping; **(b) build the
   bridge under the `Transient` marker** (TEMPLATE.md Normative Contracts) —
   spec treatment matched to its lifespan. Record the choice where it is
   durable: (a) reshapes this plan; (b) writes the marker line into the
   bridge's contract block.

Surface, but do not resolve, the load-bearing assumptions the chosen approach
depends on — list them in Critical Assumptions as Status: Pending with a
one-line "If wrong". Stage 4 verifies them.

Be brief but not lossy; drop into ultrathink if the design gets complex.
Close with rdr-common §mechanical-gate.
