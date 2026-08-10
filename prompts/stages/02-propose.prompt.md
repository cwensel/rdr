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

0. **Freshness check.** A seed may have sat idle and gone stale. Re-validate
   everything it names against the current world — references, paths, identifiers,
   peer-RDR statuses, and the scope itself — and fix what drifted in place. If a
   prior pass left a "Refinement Context" /
   "retained as history" / "folded into the body" note, act on any live "when
   you revise…" instruction it still carries, then delete the note (Draft =
   rewrite in place). If this invalidates the Problem Statement itself, stop and
   flag it — don't propose onto a false premise.

0.5 **Accretion gate.** Read the `Seam Lineage` field (filled at Seed from
   `kata-scope-review` — do not re-derive). If it carries ≥2 closed prior
   point-fixes at this locus, this is a *missing design decision, not a missing
   patch*: before proposing any mechanism, write one sentence naming the
   **undecided contract** the prior point-fixes all danced around, and set
   Profile to `foundational` (the cove lens then runs — it subsumes grounding as
   its Step 0). Enumerate approaches
   as answers to that contract question, not as the next patch. (Escape only via
   a written accretion disposition in `Seam Lineage`; count <2 → skip this step.)

1. **Read prior art FIRST — before naming any approach.** An LLM that enumerates
   first anchors on its training prior and won't reliably self-correct, so read
   before the candidate set exists, not as a bias applied after. Read the Domain
   priors' competitors/prior systems/standards for THIS problem class; on a
   `large`/`foundational` RDR, also do a bounded external pass (corpora or
   web/literature) — a read, not a spike. **Coverage:** found nothing? Write one
   Investigation line (`⚠ no prior-art coverage for <problem class>; approaches
   below are from the model prior`) — never leave the gap silent. **Cite-check:**
   every prior-art claim the choice will *rest on* must be quoted/anchored from
   source (named system + section, or `path::Symbol`), not paraphrased — a misread
   or inverted citation is the exact defect that overturned an approach three
   stages late. A claim you can't open and quote isn't load-bearing: demote it to a
   Resolve assumption rather than leaning the choice on it.
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
   (Profile is the *current* value here — the accretion gate above may have
   floored it to `foundational` in-flight.)
   - **`small`/`mid` → one paragraph.** Write the one-paragraph post-mortem of
     the shipped-and-failed approach, then confirm the recommendation survives
     it. A failure the chosen approach can't answer → switch. Variant
     `(paragraph)`.
   - **`large`/`foundational` → hardened critic: ONE `Task` sub-agent
     (rdr-common §delegation), queried ONCE** — never more critics or rounds (homogeneous
     debate converges, it doesn't dissent; one hardened critic queried once
     beats it at fewer tokens). The extension past "delegate reads" is honest:
     this delegates *judgment*, because a fresh context that never saw the
     justifying prose is the independence that defeats self-confirmation — the
     same CoVe discipline Critique already uses. It is NOT a dual-model pass
     (a sub-agent inherits this session's model — see `2-critique.md`).
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
       negation-target list; (d) a one-line technical-environment summary;
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
   Resolve and Pre-Lock.
7. Author the **design-body** (replace its `_Draft placeholder._`s):
   - **Investigation**: the prior art, code paths, and constraints that
     shaped the choice — one paragraph; deep evidence lands at Resolve.
   - **Implementation Plan**: Prerequisites, the MVV, and Phase 1…N as
     **named phases with a one-line intent each** — the spine, not the
     steps. Stay at the "what": a phase that reads like code is
     over-specified; that detail fills at Resolve/Pre-Lock.
   Leave Testing Strategy and Performance Expectations as placeholders —
   Resolve owns those, after the assumptions they lean on are verified.
8. **Joint-decision check — runs last, on the written proposal.** Catch two
   open RDRs coupling on one decision while switching is still free. From THIS
   RDR collect *modify-anchors* — the `Seam Lineage` locus plus backticked
   `path::Symbol` tokens in Proposed Solution / Implementation Plan — and
   *contract literals* — backticked error/rule codes, flag/field names, exit
   codes, sentinels in or near its ```normative fences. Open peers = `*.md` at
   depth 1 of `$RDR_RECORDS` (never recurse) whose FIRST `- **Status**:` value
   starts with `Draft` or `Final` (prefix match; the template's Status comment
   carries decoy statuses), excluding this RDR. Grep each peer for each
   anchor/literal, whole-token match. **FIRE** = a peer shares a modify-anchor
   or a contract literal AND neither RDR cites the other (`NNNN` or slug,
   either direction). On fire, PAUSE — do not hand off to refine: put the joint
   question to the user (hoist the joint decision to the consumer's
   umbrella-decision record — e.g. an RFD — as its single normative home,
   cite-don't-restate; or declare the shared interface in both RDRs) before
   first lock. A fire is detection, not demotion: the siblings proceed under
   recorded tolerance once the joint decision has a home. Report the verdict
   either way (fired on NNNN / N peers checked, none).
   **Bridge sub-check** — Cluster members only, behind the tandem barrier
   (all members' plans exist, so a scheduled deletion is decidable). When this
   RDR's Implementation Plan introduces a surface a sibling's plan schedules
   for deletion/replacement — cues `bridge`, `retire`, `replace` (stem match,
   in either plan; NOT `supersede`, which overwhelmingly claims lineage over
   prior closed RDRs and would false-fire), cross-referenced with Cluster
   membership and the joint decision's named home if one exists — PAUSE and
   put the choice to the user: **(a) skip to end-state** — implement the
   sibling's replacement directly, or reorder shipping; **(b) build the
   bridge under the `Transient` marker** (TEMPLATE.md Normative Contracts) —
   spec treatment matched to its lifespan. Record the choice where it is
   durable: (a) reshapes this plan; (b) writes the marker line into the
   bridge's contract block.

Surface, but do not resolve, the load-bearing assumptions the chosen approach
depends on — list them in Critical Assumptions as Status: Pending with a
one-line "If wrong". Stage 4 verifies them.

Be brief but not lossy; drop into ultrathink if the design gets complex.
