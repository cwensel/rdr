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
4. **Premortem the chosen approach (one paragraph).** Assume this approach
   shipped and failed in production; write the one-paragraph post-mortem, then
   confirm the recommendation survives it. If the post-mortem exposes a failure
   the chosen approach can't answer, revise the choice now — it is far cheaper
   to switch approaches here than after Resolve has spent its spike budget. This
   is the approach-level gut-check, distinct from the lock-time hostile
   premortem in Stage 5 Critique; it runs on every profile because Propose
   always does. **Sibling-path check:** if the chosen approach adds a new
   discriminator, heuristic, switch case, or identity rule, grep whether an
   adjacent/sibling path already makes that decision, and *exhibit* the result
   in the RDR (a pasted `path::Symbol`, or "searched, none exists") — reusing an
   existing signal beats inventing a parallel one.
5. Write the recommended approach into Proposed Solution and the alternatives
   into Alternatives Considered. Keep Technical Design at the level the
   problem needs — do NOT over-specify signatures yet; that sharpens during
   Resolve and Pre-Lock.
6. Author the **design-body** (replace its `_Draft placeholder._`s):
   - **Investigation**: the prior art, code paths, and constraints that
     shaped the choice — one paragraph; deep evidence lands at Resolve.
   - **Implementation Plan**: Prerequisites, the MVV, and Phase 1…N as
     **named phases with a one-line intent each** — the spine, not the
     steps. Stay at the "what": a phase that reads like code is
     over-specified; that detail fills at Resolve/Pre-Lock.
   Leave Testing Strategy and Performance Expectations as placeholders —
   Resolve owns those, after the assumptions they lean on are verified.

Surface, but do not resolve, the load-bearing assumptions the chosen approach
depends on — list them in Critical Assumptions as Status: Pending with a
one-line "If wrong". Stage 4 verifies them.

Be brief but not lossy; drop into ultrathink if the design gets complex.
