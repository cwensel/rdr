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
`eval "$("$RDR_HOME/bin/recs" paths --lens tooling-pass --next-iter <NNNN>)"`;
`mkdir -p "$ITER_DIR"` → write `tooling-pass.md` there (at `ITER=1` it *is*
the base; re-runs land in `iter-N/`). One eval per pass — after the `mkdir`,
a re-eval names the NEXT iteration and splits the pass across two dirs. Lint once to `$ITER_DIR/lint.txt` (the sweep's "Run first");
a sweep you spawn gets that path in its brief. On BLOCK, split the findings:

- MECHANICAL — fix in this pass, re-run the sweep: a hollow/bracketed section
  fillable from material already in the RDR (`## References` from citations the
  body already carries), surviving template instructional text — including
  blocks an older TEMPLATE.md shipped and the current one doesn't — and C5
  anchor rewrites. Never route these to Refine: conformance is outside its
  contract, so the finding returns unchanged.
- SUBSTANTIVE — NOT READY: needs new evidence or a design call (disturbed/
  refuted assumption, section unfillable from what the RDR carries). Class it
  (`blocker_class`, rdr-common §rdr-write) — `--outcome return` names the stage.

LOOP-BREAKER: before any NOT-READY return pointer, diff this pass against the
prior one from the same `paths` eval (`$PRIOR_DIR`; absent = first pass, skip):
`R="$RDR_HOME/bin/recs"; LC_ALL=C comm -12 <("$R" anchors --record <NNNN>
"$PRIOR_DIR"/tooling-pass.md) <("$R" anchors --record <NNNN> "$ITER_DIR"/tooling-pass.md)`.
A non-empty result is a finding re-reported after its named stage ran — a
ROUTING failure, not an author failure — stop per §stop-packet
(`stopped:finalize-routing-loop:<that list + the stage that failed to clear it>`).

ALSO confirm no cluster re-entry note survives: a `## Refinement Context
(cluster re-entry — delete on re-lock)` block left by the 07.1 gate MUST be
gone by re-lock — its defect resolved and folded into live text. Stage 7 is the
one chokepoint every re-entry path (2/3/4) shares, so a surviving note is NOT
READY: the cross-RDR defect that demoted this RDR was never closed. On a
`@finalize` re-entry (RE-LOCK-ONLY, `re-verify none`) the note's listed
defects ARE this stage's content fix: apply each in place first, by `--select`
id, then run the sweep and delete the note. On a stage-scoped re-entry whose
target pass has already run (the note names refine/resolve and that stage
closed), the listed defects should be closed in live text: verify each there
(delegate the read), then delete the note at lock — NOT READY only if one is
still open.

Repeatability-lite: run §lens-row's call with `--outcome repeatability` (follow a
`resolve:determinacy` chain once). `emit.next` must be `none`; anything else is
NOT READY, class `determinacy`, with `emit.surface` quoted.

Joint-decision fence: before the lock, re-run propose's arms 1 and 2 as one
rdr-common §rdr-write call,
`--outcome fence $("$RDR_HOME/bin/recs" status --tags --filter overlap_uncited,rulings_open,clustered,impact_families NNNN)`;
it must emit `op = none`. `stopped:overlap-uncited` is NOT READY (a shared
decision nobody fired on — fire it, do not sync the copies), and
`stopped:overlap-unchecked` means nothing looked. `stopped:rulings-open` is NOT
READY too: a `rulings.md` line (rdr-common §run-prompt) no stage marked
absorbed — apply it at the stage it names, mark it, re-run. The lock itself refuses
`joint_check_home` = `open` (`stopped:joint-decision-open`) — a fire must be
homed before Final; homing is a JDR/owner-clause edit, never a wording tweak
to a peer's Final. `unhomed` locks: the home names something the tool cannot
resolve (a register outside the records dir), and the gate's written response
is what vouches for it.

Then run the Finalization Gate from the template as written responses (not
checkboxes): items 1, 2, 3 and 5 go to {ARTIFACT_DIR}/gate.md — a short header
(RDR id/slug, date, verdict), then one H2 section per item — not into the RDR.
Item 4 (Cross-Cutting) is authored ONCE, in the record at the Gate's item,
where the lock keeps it citable; gate.md never carries a second copy to drift:
1. Contradiction Check — conflicts between Research Findings and Proposed
   Solution; planned features vs stated principles.
2. Assumption Verification — every Critical Assumption record internally
   consistent (Status/Method/Evidence agree, "If wrong" non-empty); no
   load-bearing Docs-Only without a Spike/Source-Search plan; no Source-Search
   self-reference.
3. Scope Verification — the Minimum Viable Validation is in scope, not
   deferred; name the specific test/proof. A record that declares Cluster
   siblings also names its blast radius here: name the retired literals
   (the exact tokens the CHANGE-tagged contracts retire or rename, ≤10,
   judgement) and run `"$RDR_HOME/bin/recs" impact <slug> --literal '<tok>'
   … > {ARTIFACT_DIR}/impact.md`; a predicted re-cut of a peer's shipped
   REQ is a lock condition — an Overrides entry, or NOT READY with the peer
   named. The fence refuses to lock a clustered record without the file
   (`stopped:impact-unwritten`); Phase 0 re-runs the projection from the
   file's `literals:` line, so the judgement is made once, here.
4. Cross-Cutting Concerns — for each that applies, how this RDR addresses it
   or which peer RDR owns the policy. Omit (don't N/A-bullet) what doesn't.
5. Proportionality — right-sized; flag anything to trim before locking.

Verdict the gate, then act on it in this same pass — the verdict is the gate,
not a human pause:

- NOT READY (any blocker) — report the named blockers, each with its class and
  the `next_action` that `--outcome return` emits, and apply that emit's Status
  qualifier so the named stage sees it is re-entering. Flip NOTHING — that bars
  the lock flip, not the route-back marker. Stop here.
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
