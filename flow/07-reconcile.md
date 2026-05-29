# Stage 7 — Reconcile Spikes & Assumptions

**Goal**: close the gap the review rounds opened. Stage 4 verified the
assumptions *as the draft stood then*; Stages 5–6 mutated it — critique adds
load-bearing claims, fix-passes rewrite normative contracts. This gate runs
**once, after all rounds**, forcing every open spike and every round-disturbed
assumption to a terminal state **before** Finalize. Stage 4 guards the review
*entrance*; this is the *exit*.

## Paste this

Fill `{RDR_PATH}`, `{RDR_RESOURCES}` (default `_rdr/rdr-resources.md`), `{RDR_ENV}`
(default `_rdr/rdr-env.md`); paste the Stage 6 needs-verification list(s).

```text
For the RDR at {RDR_PATH}, reconcile every open spike and every assumption
the review rounds disturbed, before this RDR can lock. Read {RDR_RESOURCES} for
the corpora and design docs, and {RDR_ENV} for the spike location ({SPIKE_DIR}).
Stage 6 needs-verification list(s):
<paste the list(s)>


Build the open set from FOUR sources:
1. The Stage 6 needs-(re)verification list(s) — assumptions flipped back to
   Pending, and new A-N claims added during fix passes.
2. Every Critical Assumption currently Status: Pending or Unverified.
3. Every spike NAMED anywhere in the RDR (or its findings) with no captured
   run — check {SPIKE_DIR} for existing results before assuming one is unrun.
4. An exactness-word sweep: every all/every/first/nearest/byte-identical/
   lossless/canonical/deterministic/stable-order claim a review round touched
   or introduced. (Stage 4 already swept every exactness word against an
   Evidence Record; this is only the post-mutation delta — a claim with no
   record that the rounds did NOT touch is a Stage 8 mechanical-sweep
   regression catch, CHECK 4, not Stage 7's job.)

For heavy work — running a spike against the live target, searching corpora,
reading captured spike output — delegate to a sub-agent that returns verdict +
evidence pointer (command + output path, or file:line), not raw dumps.

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
done — Stage 8's Assumption Verification reads the RDR, not this report.

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

Be brief; ultrathink for any assumption whose refutation would force a design
change; come to the user for accept/defer tiebreakers.

Output:
  - A table: item | source(1–4) | disposition (VERIFIED/DOWNGRADED/ACCEPTED/
    BLOCKER) | evidence pointer or plan.
  - Verdict: RECONCILED (all items terminal, no BLOCKER) — ready for Finalize.
            NOT RECONCILED — list each BLOCKER and the stage to return to.
```

**Run when**: every pre-lock round in the RDR's profile has run and its
findings are resolved (Stage 6) — immediately before Finalize. **Produces**:
the RDR's Critical Assumptions edited in place to their terminal state
(Status/Method/Evidence); the reconcile report (table + verdict); any newly-run
spike's command + output under `{SPIKE_DIR}`.

## Review gate

- **Is the open set complete?** Cross-check the four sources yourself — a
  reconcile that only looks at source 2 (already-Pending) misses the dangerous
  ones (claims silently *introduced* by a fix pass).
- **Did any spike actually refute the design?** If so, the verdict must be NOT
  RECONCILED with a named return stage — not a quietly-edited assumption. Treat
  a "we adjusted the wording" disposition on a
  refutation with suspicion.
- **Are DOWNGRADED items genuinely survivable?** A downgrade promises
  implementation will verify it and that being wrong is recoverable. If being
  wrong breaks the MVV, it can't be downgraded — VERIFY now.
- **Were spikes actually run, with captured output** — not asserted? Check that
  `{SPIKE_DIR}` grew.
- **Did the dispositions land in the RDR**, not just the report? Open the
  Critical Assumptions section and confirm each item's Status/Method/Evidence
  matches its row in the table. A reconcile that produced only a report left the
  RDR unchanged — Stage 8 reads the RDR, so the work isn't done.

If NOT RECONCILED, return to the named stage; do not proceed to Finalize.

## Advance when

Every open spike and disturbed assumption is VERIFIED, DOWNGRADED (named
impl-time plan + survivable "If wrong"), or ACCEPTED (Design Decision), and each
disposition is written into the RDR's Critical Assumptions section (not only the
report); no assumption the RDR relies on stands refuted; no MVV-critical spike is
deferred. Verdict: RECONCILED.

→ Next: [08.0-finalize.md](08.0-finalize.md)

## Why the two HARD RULES are hard

Both encode a near-miss that reached lock day in practice — the cost of *not*
having this gate, not abstract caution:

- **Refutation routes backward.** A late critique-driven spike once refuted an
  assumption an earlier stage had marked `Verified` — the design assumed a
  weaker rule than the target system actually enforces. Caught on lock day, it
  forced a
  normative rewrite. Without a forced gate the temptation is to "adjust the
  wording" and lock anyway; that papers over a design that is now wrong. So a
  refutation is a BLOCKER that re-opens the RDR (Stage 2/3/4), never a checkbox.
- **An MVV-critical spike can't defer.** A spike that pins the very thing the
  Minimum Viable Validation proves once nearly slipped past lock as a "post-lock
  follow-up." If the MVV's fidelity bar rests on it, deferring it locks an
  unproven contract. It is a pre-lock prerequisite — run now or NOT READY.
