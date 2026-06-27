For the RDR at {RDR_PATH}, reconcile every open spike and every assumption
the review rounds disturbed, before this RDR can lock. Read {RDR_RESOURCES} for
the corpora and design docs, and {RDR_ENV} for the spike location ({SPIKE_DIR}).
Pre-Lock needs-verification list(s):
<paste the list(s)>


Build the open set from FOUR sources:
1. The Pre-Lock needs-(re)verification list(s) — assumptions flipped back to
   Pending, and new A-N claims added during fix passes.
2. Every Critical Assumption currently Status: Pending or Unverified.
3. Every spike NAMED anywhere in the RDR (or its findings) with no captured
   run — check {SPIKE_DIR} for existing results before assuming one is unrun.
4. An exactness-word sweep: every all/every/first/nearest/byte-identical/
   lossless/canonical/deterministic/stable-order claim a review round touched
   or introduced. (Stage 4 already swept every exactness word against an
   Evidence Record; this is only the post-mutation delta — a claim with no
   record that the rounds did NOT touch is a Stage 7 mechanical-sweep
   regression catch, CHECK 4, not Stage 6's job.)

For heavy work — running a spike against the live target, searching corpora,
reading captured spike output — delegate to a sub-agent that returns verdict +
evidence pointer (command + output path, or file:line), returned as a §return-packet (rdr-common), not raw dumps.

For each item, force ONE terminal disposition AND write it into the RDR's
Critical Assumptions section (this is an in-place edit to the draft, not just a
report — the report is the audit trail, the RDR is the artifact):
  - VERIFIED — run the spike / source-search now; capture command + output
    under {SPIKE_DIR}; set Status: Verified with the concrete Evidence pointer.
  - DOWNGRADED — cannot verify before lock, but not load-bearing for the MVV:
    set Status: Pending with a NAMED plan that WILL run during implementation
    (cite the test/spike). Allowed only if "If wrong" is survivable and stated.
  - ACCEPTED — a Design Decision, not a fact to verify: Method: Design
    Decision, with the rejected alternative named.

After dispositioning, the RDR's Critical Assumptions records and the report
table MUST agree: every item's Status/Method/Evidence in the RDR matches its
row here. A disposition that lives only in the report and not in the RDR is not
done — Stage 7's Assumption Verification reads the RDR, not this report.

HARD RULE — refutation is not a checkbox. If a spike or source-search REFUTES
an assumption the RDR currently relies on (the design says X, the target system
or the source does not-X), STOP. Do not paper over it. Report it as a BLOCKER
and send the
RDR back to Stage 2 (approach), 3 (refine), or 4 (re-resolve) — name which.
Locking over a refuted assumption is the failure this gate exists to prevent.

HARD RULE — no spike deferred past lock if the MVV depends on it. If a spike
pins the very thing the Minimum Viable Validation proves (byte-parity
reference, the load-bearing external behavior), it is a pre-lock prerequisite,
not a post-lock follow-up — run it now or the RDR is NOT READY.

Before the verdict, a completeness check: no `_Draft placeholder._` may survive
in any body section and no `this is a seed skeleton` header may remain. A hollow
body is NOT RECONCILED — return it to Propose (Investigation / Implementation
Plan) or Resolve (Testing Strategy / Performance Expectations), whichever owns
the empty section. This is a grep, not a judgment; it does not reopen assumptions.

Be brief; ultrathink for any assumption whose refutation would force a design
change; come to the user for accept/defer tiebreakers.

Output:
  - A table: item | source(1–4) | disposition (VERIFIED/DOWNGRADED/ACCEPTED/
    BLOCKER) | evidence pointer or plan.
  - Verdict: RECONCILED (all items terminal, no BLOCKER) — ready for Finalize.
            NOT RECONCILED — list each BLOCKER and the stage to return to.
