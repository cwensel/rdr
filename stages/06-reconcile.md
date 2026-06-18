# Stage 6 — Reconcile Spikes & Assumptions

**Goal**: close the gap the review rounds opened. Stage 4 verified the
assumptions *as the draft stood then*; Stages 5–6 mutated it — critique adds
load-bearing claims, fix-passes rewrite normative contracts. This gate runs
**once, after all rounds**, forcing every open spike and every round-disturbed
assumption to a terminal state **before** Finalize. Stage 4 guards the review
*entrance*; this is the *exit*.

## Paste this

Fill `{RDR_PATH}`. Resolve `{RDR_RESOURCES}` / `{RDR_ENV}` via the workspace
marker (see README *Where the seam lives* — `. "$WS/.rdr-workspace"` exports
`$RDR_RESOURCES` / `$RDR_ENV`); paste the Pre-Lock needs-verification list(s).

**Prompt**: [`../prompts/stages/06-reconcile.prompt.md`](../prompts/stages/06-reconcile.prompt.md)
— the `/rdr-reconcile` skill runs it (paste the Pre-Lock needs-verification list
where the prompt's `<paste the list(s)>` marker is). Paste its body by hand if
driving without the skill.

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
  RDR unchanged — Stage 6 reads the RDR, so the work isn't done.
- **Is the body whole?** No `_Draft placeholder._` or seed-skeleton header
  survives — a hollow body is NOT RECONCILED, back to Propose/Resolve. The grep
  that catches at Stage 6 what CHECK 1 would otherwise catch at the lock door.

If NOT RECONCILED, return to the named stage; do not proceed to Finalize.

## Advance when

Every open spike and disturbed assumption is VERIFIED, DOWNGRADED (named
impl-time plan + survivable "If wrong"), or ACCEPTED (Design Decision), and each
disposition is written into the RDR's Critical Assumptions section (not only the
report); no assumption the RDR relies on stands refuted; no MVV-critical spike is
deferred. Verdict: RECONCILED.

→ Next: [07.0-finalize.md](07.0-finalize.md)

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
