Review the RDR at {RDR_PATH}. Verify it against the rdr README and TEMPLATE
and identify exactly what's required to flip it to Final.

Read {RDR_ENV} for the pre-lock output location ({FLOW_DIR}). If reading those
outputs is heavy, delegate to a sub-agent that returns per-round "findings
closed? y/n + residue", not the raw round files.

FIRST run the mechanical sweep — the Tooling pass: open
$PROCESS_ROOT/rdr/prompts/gate/tooling-pass.md and run its checks verbatim. It runs on
every RDR as a post-mutation regression check, since the rounds and the Stage 6
reconcile just rewrote this draft. If the sweep returns BLOCK, the RDR is NOT
READY — fix the regressions before the written responses below.

ALSO confirm no cluster re-entry note survives: a `## Refinement Context
(cluster re-entry — delete on re-lock)` block left by the 07.1 gate MUST be
gone by re-lock — its defect resolved and folded into live text. Stage 7 is the
one chokepoint every re-entry path (2/3/4) shares, so a surviving note is NOT
READY: the cross-RDR defect that demoted this RDR was never closed.

Then run the Finalization Gate from the template as written responses (not
checkboxes), each becoming part of the permanent record:
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
- READY — lock immediately: set Status to Final, write the gate's five
  responses into the Finalization Gate section, update the status/title row for
  this RDR in the RDR README index, and commit as a fixup to the RDR's existing
  commit (we never amend RDRs as documentation — this is a status flip with the
  gate record).
- A single gate item genuinely in doubt (not a clear pass or fail) — stop and
  surface it per §stop-packet rather than guessing the lock.

Ultrathink only on a gate item genuinely in doubt.
