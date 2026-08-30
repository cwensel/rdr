Review the RDR at {RDR_PATH}. Verify it against the rdr README and TEMPLATE
and identify exactly what's required to flip it to Final.

Read {RDR_ENV} for the pre-lock output location ({EVIDENCE_DIR}) and the gate
record's home ({ARTIFACT_DIR}). If reading those outputs is heavy, delegate to
a sub-agent that returns a §return-packet (rdr-common) — findings-closed in
summary_50w, residue in next_action. Re-run none of what you delegated; wait by
ending the turn (rdr-common §delegation).

FIRST run the mechanical sweep — the Tooling pass: open
$RDR_HOME/prompts/gate/tooling-pass.md and run its checks verbatim. It runs on
every RDR as a post-mutation regression check, since the rounds and the Stage 6
reconcile just rewrote this draft. Bind the report dir in one call
(rdr-common §evidence — ask for the dir, never compose one):
`eval "$("$RDR_HOME/bin/rdr" paths --lens tooling-pass --next-iter <NNNN>)"` →
write `tooling-pass.md` to `$ITER_DIR` (at `ITER=1` it *is* the base; re-runs
land in `iter-N/`). On BLOCK, split the findings:

- MECHANICAL — fix in this pass, re-run the sweep: a hollow/bracketed section
  fillable from material already in the RDR (`## References` from citations the
  body already carries), surviving template instructional text — including
  blocks an older TEMPLATE.md shipped and the current one doesn't — and C5
  anchor rewrites. Never route these to Refine: conformance is outside its
  contract, so the finding returns unchanged.
- SUBSTANTIVE — NOT READY: needs new evidence or a design call (disturbed/
  refuted assumption, section unfillable from what the RDR carries). Name the
  stage whose prompt owns the fix.

LOOP-BREAKER: before any NOT-READY return pointer, read the prior reports
under the same eval's `$EVIDENCE_DIR` (`$ITER_FOUND` names the iterations on
disk). A finding re-reported unchanged after its named stage
ran is a ROUTING failure, not an author failure — stop per §stop-packet
(`stopped:finalize-routing-loop:<finding + stage that failed to clear it>`).

ALSO confirm no cluster re-entry note survives: a `## Refinement Context
(cluster re-entry — delete on re-lock)` block left by the 07.1 gate MUST be
gone by re-lock — its defect resolved and folded into live text. Stage 7 is the
one chokepoint every re-entry path (2/3/4) shares, so a surviving note is NOT
READY: the cross-RDR defect that demoted this RDR was never closed.

ALSO scan `Profile` + `Normative Contracts`: for a `mid`/`large` RDR whose
contract names step-ordering, parse/deparse, import/export, compose/decompose,
hashing, identity, migration, ambiguous field ownership, or multi-step MVV
fidelity, require `evidence/repeatability/run-1.md` + `diff.md` or a written
`determinacy: n/a - <reason>` disposition. If absent, NOT READY; return to
Stage 5 repeatability-lite.

Then run the Finalization Gate from the template as written responses (not
checkboxes), written to {ARTIFACT_DIR}/gate.md — a short header (RDR id/slug,
date, verdict), then one H2 section per item — not into the RDR:
1. Contradiction Check — conflicts between Research Findings and Proposed
   Solution; planned features vs stated principles.
2. Assumption Verification — every Critical Assumption record internally
   consistent (Status/Method/Evidence agree, "If wrong" non-empty); no
   load-bearing Docs-Only without a Spike/Source-Search plan; no Source-Search
   self-reference.
3. Scope Verification — the Minimum Viable Validation is in scope, not
   deferred; name the specific test/proof.
4. Cross-Cutting Concerns — for each that applies, how this RDR addresses it
   or which peer RDR owns the policy. Omit (don't N/A-bullet) what doesn't.
5. Proportionality — right-sized; flag anything to trim before locking.

Verdict the gate, then act on it in this same pass — the verdict is the gate,
not a human pause:

- NOT READY (any blocker) — report the named blockers and the stage each
  returns to. Flip NOTHING. Stop here.
- READY — lock immediately. First write responses 1, 2, 3 and 5 to
  {ARTIFACT_DIR}/gate.md (on a re-lock, overwrite it — gate.md is the current
  lock's record). Then run rdr-common §rdr-write twice: `--outcome lock` moves
  those four sub-sections out behind the pointer line and flips Status to Final,
  and `--outcome readme` brings the index row with it (adding one if this is a
  pre-seed RDR that has none). Apply each `emit.edit` as handed. A
  `stopped:gate-stale` probe before gate.md exists is not the lock — re-run
  after gate.md is written. **`### Cross-Cutting Concerns` STAYS in the
  record**, below the pointer: it is the one gate item written to be cited BY
  OTHER RDRs (this prompt's own item 4 says "which peer RDR owns the policy"),
  so it stays projected and addressable as `cli/NNNN:G-cross-cutting`. The
  other four judge this record at this lock and no peer cites them.
  If autocommit is on (rdr-common §commit),
  commit it as a **standalone** `docs(rdr): finalize cli/NNNN <slug> (Gate PASS)`
  over the RDR + README + {ARTIFACT_DIR}/gate.md — **never** a `fixup!` (RDR
  commits ARE the design history; we record the lock as its own real subject,
  not a deferred-squash).
- A single gate item genuinely in doubt (not a clear pass or fail) — stop and
  surface it per §stop-packet rather than guessing the lock.

Ultrathink only on a gate item genuinely in doubt. Be brief but not lossy in the
gate responses — terse verdicts, no over-explaining a clear pass; spend tokens
only where they change the result.
