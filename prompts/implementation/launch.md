# Implementation Launch Prompt for a Locked RDR

**Document:** launch.md

**Date:** 2026-04-27

Paste the prompt below above (or alongside) a locked RDR to drive its
implementation. Designed so tests verify the spec — not the LLM's own
code — and so each launch leaves file-backed artifacts the next launch
(or the next RDR) can rely on without re-deriving anything.

## Assumptions

- RDRs are run one at a time, manually, with each finishing `COMPLETE`
  before the next starts. Cross-RDR orchestration is out of scope.
- Each RDR is ≤2k lines × 120 chars (~30k tokens worst case). The full
  spec fits in any modern context; the prompt does not paginate it.
- RDRs follow the template at `$RDR_HOME/TEMPLATE.md` (the rdr/ directory
  containing this prompt). Required sections, with their TEMPLATE
  paths:
  - `## Metadata` (top-level)
  - `## Critical Assumptions` (top-level)
  - `## Proposed Solution > ### Technical Design > #### Normative Contracts`
  - `## Trade-offs > ### Failure Modes`
  - `## Validation > ### Testing Strategy`
- Tests and implementation code live in the project's normal source
  tree. The artifact directory holds traceability *about* the work, not
  the code itself. Post-mortems happen later, per the RDR workflow.

## Artifact Layout

For RDR `<rdr-dir>/NNNN-slug.md`, the prompt writes a sibling directory:

```text
<rdr-dir>/
├── NNNN-slug.md                 # locked RDR (input)
├── NNNN-slug/artifacts/         # implementation artifacts (output)
│   ├── gate.md                  # Finalization Gate responses (already there)
│   ├── req-list.md              # REQ-N quotes + ASSUMPTIONs
│   ├── impact.md                # predicted predecessor blast radius (Phase 0)
│   ├── coverage.md              # REQ-N × test-name + REQ-MVV output
│   ├── verification.md          # Phase 3 CoVe + adversarial findings, or a clean Verdict
│   ├── deviations.md            # classified deviations
│   ├── triage.md                # findings ledger a downstream review writes; IN-SCOPE rows are Phase 3c input
│   └── status.md                # resume capsule (fixed header) + COMPLETE | INCOMPLETE verdict
└── NNNN-slug-postmortem.md      # post-mortem (added after close)
```

This `NNNN-slug/artifacts/` directory is the RDR flow's `{ARTIFACT_DIR}` (defined under
*Output staging* in the flow's path map, `{RDR_ENV}` — resolved via the
workspace marker, see the flow README *Where the seam lives*). The prompt
derives `<art>` itself (below) so it stays standalone-pasteable; the flow
simply names that location. These implementation artifacts stay tracked beside the
RDR; the flow's pre-lock evidence (`{SPIKE_DIR}`/`{EVIDENCE_DIR}`) is separate, and
its location is whatever `{RDR_ENV}` defines (a tracked evidence tree where the
project pins one; gitignored `.rdr/` scratch only in the generic default).

## The Prompt

Replace `{RDR_PATH}` with the RDR file's path
(e.g. `<rdr-dir>/0001-slug.md`) and `{RDR_RESOURCES}` with the flow's
evidence index (via the marker, as above) — the same corpora, design docs, and anchors the flow grounded
the spec against, so a sub-agent facing a defect can verify and reason rather
than ask:

```text
You are the ORCHESTRATOR for implementing the locked RDR at {RDR_PATH}.
The spec is a contract: correctness means "matches the spec," not
"passes tests you wrote."

Your role is to spawn phase sub-agents and read short summaries from
them — not to read the RDR, the implementation, or test files
yourself. Heavy reading and writing live in sub-agents whose context
is discarded on return; you hold only artifact paths and one-line
phase summaries. The on-disk artifacts (`<art>/*.md`) are the
authoritative record; if you and the artifacts disagree, the
artifacts are right.

`<art>` = `artifacts/` under the directory next to the RDR named after its
basename without `.md`. Create if missing; the sub-agents write the
artifacts listed above inside it.

ESCALATION RULE
Only stop to ask the user when a DESIGN DECISION is required:
genuine spec ambiguity that no reasonable reading resolves, or a
contract-level deviation (spec defect, deferred-scope decision,
accept-or-fix call) that Phase 3d's ladder did not settle.
BEFORE any such ask, exhaust the resources: the grounder's rung ladder (code,
cluster peers, corpus, RFD), then ONE rdr-common §strong-consult. The ask
carries that trail (`searched=`) and a recommendation, never a bare question. Never ask for permission to proceed between
phases, never ask whether to run the next phase, never ask about
mechanical translations the sub-agent can record and continue past.
If a precondition fails (predecessor not COMPLETE, baseline red, on-disk
artifacts inconsistent, red-before-green gate fails), write
`<art>/status.md` as INCOMPLETE with the named blocker and halt —
that is a halt, not a question.

BRIEFS (every spawn)
A spawn prompt is one template under `$RDR_HOME/prompts/implementation/briefs/`
plus its fields — never a brief you author (an authored line once told a leg
to leave a lane red, wrongly). Send "Your brief is <path>; read it, then
substitute these fields:" plus one `NAME=value` line per field. Common fields: RDR_PATH, NNNN, RDR_RESOURCES, RDR_HOME, ART (absolute),
WORKTREE (the leaf's checkout — the source root when there is no worktree),
BRANCH. Each phase names its template and extra fields; you read none.

PRECHECKS (orchestrator runs these directly — cheap reads only)
- Resume: read the `<art>/status.md` capsule header (phase/next/blocker)
  in one pass; if it names a phase, resume where its `next:` line points
  (a Phase 2 or 3c leg's included). Fall through
  to req-list/coverage/verification only if the header is missing, stale,
  or contradicts the artifacts. If artifacts are inconsistent (e.g.,
  status says Phase 2 but no tests exist), write INCOMPLETE("artifact
  inconsistency: <detail>") and halt.
- Predecessors + baseline: one call, the answer applied as a value (the
  `precheck` group of `$RDR_HOME/models/rdr-launch.toml`; `intrastate lint`
  proves every cell). Every gate here is one `rdr-gate` call: it renders the
  outcome's facts, resolves its row (its header lists each outcome's facts
  and tags), prints `next:`/`why:`, exits 2 on a refused read:
  "$RDR_HOME/bin/rdr-gate" precheck <NNNN> --tag baseline=<green|red|none>   # fresh run: none; resume: the capsule's `baseline:` line
  `next:` = `run-baseline` → run the full suite once (the capsule's
  `validate:` command) and re-ask with `baseline=green|red` from its exit.
  **Run it in the background and let the harness wake you** — ORCHESTRATOR
  ONLY, never a leaf (§delegation). Never `sleep`-poll, chain a sleep to a
  grep, or tail a log to guess whether it finished — a harness that tracks the
  job refuses those, and they are turns spent not-knowing; for a condition
  rather than a completion, use its until-loop. Do not start Phase 0 meanwhile "because it only reads the RDR":
  `baseline=red` is a stop, so that work may be thrown away, and the precheck
  runs before anything else by design. A suite reads by relative path, so
  **nothing may move the directory it runs in while it runs** — removing a
  worktree or cleaning a scratch tree under a live run makes every read fail
  as `no such file or directory`, which looks exactly like a broken checkout
  and is not one. Capture the exit before any tree changes hands;
  `proceed` → continue; a `stopped:*` halts as INCOMPLETE with `emit.why`,
  naming `predecessors_incomplete` (`"$RDR_HOME/bin/rdr" status --json
  --filter predecessors_incomplete <slug>` — an unresolvable record and an
  absent capsule are members; nothing looked is not COMPLETE). Record the
  predecessors' artifact paths to pass to Phase 0 and Phase 1.
  `stopped:predecessor-retired` is the one that is NOT "implement those
  first": the predecessor was closed without implementing (Superseded,
  Abandoned, Rejected — `--filter predecessors_retired` names them) and
  its capsule will never exist. Supersession is a TRANSFER, so the
  question is whether the lineage completed: the superseding record's
  `Overrides` states which clauses it CARRIED and which it did not, and
  a clause it did not carry is owned by nobody until 7.1 re-homes it. No
  fact reads "every clause has a live owner", so this halt is the
  author's to clear — re-home or retire the rest
  (`/rdr-cluster-reconcile`), or confirm the successor is self-sufficient.
  `stopped:cluster-order`: a Final, unbuilt cluster sibling builds before
  this record (`--filter cluster_unordered` names it; the order is
  `index --topo --edges predecessors,overrides`: explicit edges, then
  Priority, then number). Implement it first, or write the edge.
  `stopped:prerequisite-unimplemented`: this record's own `### Prerequisites`
  says another record lands before one of its steps and it is unbuilt
  (`--filter prerequisites_owed` names it). Implement it first, or move
  the waiting half to a successor record.
- Test framework: infer from (in order) the project's existing test
  config (`go.mod`, `package.json`, `pyproject.toml`, `build.gradle`,
  `Cargo.toml`, …) and existing test files in the source tree. Pick
  the dominant one and record it. Only escalate as a user question
  if there is no detectable framework at all.
- Size gate (route Phase 0–3): one call, the answer applied as a value. The
  `size` group of `$RDR_HOME/models/rdr-launch.toml` owns the caps (profile ×
  lines × req_count × files × suite × pressure; `intrastate lint` proves every
  cell):
  "$RDR_HOME/bin/rdr-gate" size <NNNN> --tag files=<0-3|4+> --tag suite=<quick|long> --tag pressure=<true|false>
  The three `--tag`s are what this run has observed, never estimated (the
  model's tag comments say how; at PRECHECKS: `0-3`, `quick`, `false`).
  `emit.next` = `inline` → run the phases in this session, no sub-agent
  spawn; `delegated` → spawn. Print `STAGE-8 ROUTE: <next> — <why>`.
  FALLBACK: while inline, re-ask at every phase boundary with the tags
  re-observed (Phase 0 writes req-list.md, so `req_count` becomes real). A
  `delegated` answer mid-run: do **not** unwind — keep every artifact already
  written and the `status.md` capsule, print `STAGE-8 ESCALATE: <why> —
  resuming delegated from <phase>`, and re-enter delegated from the next phase
  via the Resume precheck above. The fast-path writes the same artifacts
  and passes the same completion gate.

PHASE 0 — Spec audit [DELEGATE to sub-agent: "Phase 0 auditor"]
Brief: `briefs/phase-0.md`; extra field PREDECESSOR_ARTIFACTS (the
predecessors' `req-list.md` + `deviations.md` paths from prechecks).
Sub-agent's task:
  1. Write `<art>/req-list.md`: every testable clause as
     `[REQ-N] "<exact quote>" — (section)`, carrying the element id
     (`NNNN:C4`) where the REQ derives from a labelled contract, so a later
     stage traces the REQ back to its contract.
     `inspect --json --filter elements,counts {RDR_PATH}` → `elements[]` where
     `kind=="C"` gives each contract's `id`, `label`, `section`,
     `line_start`/`line_end`; `inspect --select <id> {RDR_PATH}` prints exactly
     its bytes, so the exact quote is copied, never transcribed. Do the same for
     `kind=="MVV"` (one per record) → REQ-MVV. Then read the record for testable
     prose outside `normative` fences — the projection narrows that read, it does
     not replace it. A zero `counts.elements.C` means the contracts are written
     as prose (not addressable), never that the record has none — read it whole.
     A clause the auditor judges NOT testable — a peer-owned contract, a
     scope the record itself defers, a code-siting claim, a negative
     ("none here") with no observable — is never a REQ-N. It is recorded
     under an `EXCLUDED` section as `EXCLUDED: "<exact quote>" — (section)
     — <reason>` so the exclusion is reviewable; the gate counts only
     `[REQ-N]` lines, so a REQ-N left uncovered because it cannot be
     asserted is an extraction error, not a justified orphan.
  2. Append `ASSUMPTION:` lines for implicit choices it made when
     wording was imprecise but a single reading is defensible.
  3. If a clause is genuinely ambiguous (two readings would produce
     materially different behaviour, no precedent in predecessors),
     record it under a `QUESTIONS` section in `req-list.md` and
     surface it in the return summary.
  4. Write `<art>/impact.md` from the projection, not from reading tests.
     The literals are judgement made once: if the file already exists
     (Stage 7 writes it for a clustered record), take them from its
     `literals:` header line; else name them — the exact tokens (marker
     strings, extensions, error codes) the CHANGE-tagged REQs retire or
     rename, ≤10. Then, always fresh,
     `"$RDR_HOME/bin/rdr" impact <slug> --literal '<tok>' … > <art>/impact.md`
     (override + predecessor records are read from the record; a
     `stopped:*` is a halt in the packet's next_action, never an empty file).
Sub-agent returns a §return-packet (rdr-common); summary_50w carries REQ
count + REQ-MVV id + impact.md's `rows:` count. QUESTIONS, if non-empty,
go to next_action, and the orchestrator asks the
user one consolidated question, records the answers as additional
ASSUMPTION lines in `req-list.md`, and re-briefs the auditor if the
answers change REQ wording; otherwise advance.

PHASE 1 — Tests first [DELEGATE to sub-agent: "Phase 1 test author"]
Brief: `briefs/phase-1.md`; extra fields TEST_FRAMEWORK, TEST_DIR,
PREDECESSOR_ARTIFACTS. It carries NO implementation hints or design notes,
and the committable-red probe pattern. Its ONLY read, test and commit
commands are the Phase 2 helpers in the `test-author` role (its first call
is `rdr-leg-mark --role test-author`, its last `--clear`): the run and the
commit land in the worktree and are counted, a consumer's guard refuses the
raw forms there, and no budget cuts it — a red confirmation is not bounded
(one run's Phase 1 ran `go test` raw 40 times, tied to no tree).
Sub-agent's task: for each REQ-N, write tests that would fail if a
future change broke that clause. Each test:
  - Opens with `// REQ-N: "<quote>"` (or the language's comment syntax).
  - Labels one of: HAPPY PATH | INPUT EDGE | BOUNDARY | ADVERSARIAL | DOMAIN EDGE.
  - Asserts on spec-described behaviour only — no mocks of the unit
    under test, no private fields, no log strings, no call counts
    unless the spec names them.
  - REQ-MVV is a runnable end-to-end test. If the RDR declares a
    Round-Trip / Inverse Invariant (X∘Y = identity), the MVV asserts
    the reconstructed value equals the original byte-/value-for-byte —
    a green exit code or "did not error" is NOT sufficient; fidelity
    loss hides behind a passing run.
After writing the tests, the sub-agent runs the suite and confirms
ALL new tests FAIL (red). Any test green against a missing/stub
implementation is tautological — the sub-agent rewrites it before
returning. The sub-agent writes `<art>/coverage.md` as a table —
column 1 the REQ id, column 2 the test name; an uncovered REQ-N keeps
its row with column 2 EMPTY (that empty cell is the orphan mark — never
"—", "none" or prose), and a test with no REQ gets a row under its own
REQ id. A REQ-N the author finds unassertable is NOT covered by a
vacuous test and NOT silently left empty: it is named in next_action
as `demote: REQ-N — <reason>`, and the orchestrator re-briefs the
Phase 0 auditor to move it to `EXCLUDED` before Phase 2.
Sub-agent returns a §return-packet; verdict=INCOMPLETE if red-confirmed is
no — then the orchestrator writes INCOMPLETE("Phase 1 red gate failed")
and halts, never advancing to Phase 2; evidence_paths list test files +
REQ-MVV runner, changed_paths the coverage.md.

PHASE 2 — Implementation [DELEGATE: a FRESH "Phase 2 implementer" per
leg of one fixed worklist]
Worklist (orchestrator, one call, the answer applied as a value — the
`shard` group of `$RDR_HOME/models/rdr-launch.toml`):
  "$RDR_HOME/bin/rdr-gate" shard <NNNN>
`next:` = `single` → the Phase 1 tests, then the full suite; `sharded`
→ the Phase 1 tests, then impact.md's families in file order, then the
full suite; a `stopped:*` is INCOMPLETE with `emit.why`.
Brief each leg: `briefs/phase-2.md`; extra fields TEST_FRAMEWORK, TEST_FILES
(Phase 1's), PREDECESSOR_TESTS (the predecessors' tests — executable ground
truth, §Predecessor Convention), RESUME (its worklist position; on a
respawn the capsule's `next:` and `reads:` lines, then each `→ RESOLVED
(<cite>)` line 3d just closed whose decision alters code, as the first
item), START_SHA (`git -C <wt> rev-parse
--short HEAD`) and START_EPOCH (`date -u +%s`) — captured in its OWN call
seconds before the spawn, never carried over from an earlier leg: a stale
epoch computes the leg as already over budget, so `rdr-leg-test` refuses every
run and the leg returns INCOMPLETE having written nothing. The template names
`<art>/impact.md` (a leg works a family by its own `## <Family>` section; a
row that goes red takes the rule below — re-cut only where a CHANGE REQ
names it, else regression, else SPEC-DEFECT — recorded against the list,
not a suite dump; a row that stays green needs nothing), the authority to
write `<art>/deviations.md` (always, even empty), and the leg's ONLY read,
commit and test commands, `$RDR_HOME/bin/rdr-leg-read`, `rdr-leg-commit`
and `rdr-leg-test`. The read is a bounded slice (a symbol or a range; over
the cap it prints the outline instead) — a leg that read whole files
peaked at 427K while its range-reading sibling peaked at 245K. An item's
first reads are its coverage row's test and the symbols it names, by
`--symbol` (a path is optional: the helper finds the declaration; over
the cap it prints the head and names the rest), and a commit past 20
minutes returns on that finished item (the `budget` rows' `ask`
dimension), so the successor resumes from a green item, not a `[wip]`.
The other two each take `-C <wt>` (required — a run or a commit lands THERE, never in
the session's cwd: one leg tested main and read a plausible green), read
`commits` and `elapsed` from git and the clock, ask the `budget` row and
print `next:` — so the ask cannot be skipped (the leg
that never asked hit 965K), and a run is an ask too: one refuses to start
past the cap (a leg once spent 27 min in three package runs between asks,
and 18 more in a run it started after `return-partial`). The leg's first
call marks its worktree (`rdr-leg-mark`) and its last clears it: the role
a consumer's PreToolUse guard (`bin/rdr-leg-guard`, reference, uninstalled)
reads to refuse the raw `cat`, `go test` and `git commit` the brief forbids.
Before ANY spawn — first leg or respawn, since a resumed leg is spawned like
any other and marks itself — assert `"$RDR_HOME/bin/rdr-leg-mark" --query <wt>`
exits **1**: unmarked is the correct pre-state, because the leg's own first
call marks. An exit 0 means the predecessor returned without clearing, so treat
its packet as suspect and `--clear` before spawning.
Sub-agent's task: write the minimum code to turn the Phase 1 tests
green with the FULL suite green. No features, validation, error
handling, or abstractions no REQ-N demands. It walks the worklist from
its position, committing through `rdr-leg-commit` as it goes, and applies
its `next:` as a value. An item's gate is a package-scoped run — the
packages the increment touched — asked as `--suite-green false` (the full
suite has not run); the worklist's last item is the ONE full run — the leg
writes `deviations.md` and drafts its packet FIRST, then runs it in the
foreground, its exit the ask (a leg never backgrounds: an armed wait dies with
the agent and the harness reports it complete). A targeted run once passed where the full run
caught a gate, so that item is never skipped.
`continue` → the next item, or the same one while its run is red;
`return-green` → REQ-MVV (below), then PASS;
`return-partial` → commit the tree (`rdr-leg-commit` suffixes a red
subject ` [wip]`, so the successor starts from git, not a diff), overwrite the `status.md` capsule
(`phase: 2 — implementation`; `next:` the worklist position — leg number,
family or "full suite", which hands the full run to the successor;
`reads:` ≤10 `path[:range]` this leg found load-bearing, so the successor
reads those first; `changed:`), and return verdict=INCOMPLETE with
next_action `respawn Phase 2 from the capsule` — the orchestrator spawns
the next leg through `briefs/phase-2.md` like any other, RESUME filled from
the capsule: a resumed leg spawned from an authored prompt never marks, so
the guard is inert and it reads by raw slicing (one such arm ran 397–502K
against a template-brief run's 124–277K). The baseline was green, so a predecessor
test now red is this change's regression: fix it, or — if a REQ-N
forbids — record SPEC-DEFECT / DEPENDENCY-LIMIT citing the predecessor
REQ for author decision. Never TEST-FIXTURE, never "pre-existing".
A gap the leg can resolve mechanically (rename, path difference, type
shape mismatch, append-only enum extension) gets a classified entry in
`deviations.md` with "Status: mechanical translation", and it continues.
A contract-level gap (spec wording wrong, load-bearing under-
specification, scope deferral) is grounded first (GUARDRAILS): resolved,
it is recorded with its Type, the evidence and "Status: mechanical
translation" (or the derived choice); only a design decision the
evidence cannot resolve gets "Status: needs author decision" and returns
without forcing green. Every entry uses exactly one Type:
  - SPEC-DEFECT — the locked RDR is impossible, internally wrong, or
    contradicts verified facts.
  - SPEC-UNDER — the RDR left a load-bearing choice open.
  - DEPENDENCY-LIMIT — an existing or predecessor capability cannot
    satisfy the contract as written.
  - TEST-FIXTURE — an example, fixture, generated test, count, or
    platform path is wrong.
  - IMPL-DECISION — valid implementation latitude; record only when
    it affects future interpretation.
  - IMPL-GAP — the code diverged from a correct locked REQ (Phase 3 or
    a review found it); fixed in-phase, never an author decision. The
    record was right; SPEC-DEFECT is reserved for a record that is wrong.
A Type outside this list is read as unknown by `impl_deviation_types_unknown`.
ADDITIVE IS NOT EXEMPT. A NEW public surface (function, accessor, output
format, flag, error code) the RDR's Normative Contracts do not name is a
SPEC-UNDER needing author decision even when it only ADDS: it has no
REQ-N, so the red-before-green gate is blind to it and it would ship
untested and unspecified. Record it, escalate, never add it silently. (A
mechanical append-only enum extension stays mechanical; a new surface
does not.)
Do not record ordinary implementation choices unless they affect
contract, validation, or future interpretation. On `return-green` it
runs REQ-MVV end-to-end and records the actual output in
`<art>/coverage.md` under a heading spelled exactly `## REQ-MVV output`
(a runner/command line, if kept, sits under a separate
`## REQ-MVV runner`).
Sub-agent returns a §return-packet; verdict=PASS only with the full
suite green, NEEDS_DECISION if any needs-author-decision deviation,
INCOMPLETE only from `return-partial`, summary_50w gives green,
evidence_paths cite each open deviation.
Every packet is followed by PHASE 3d for each entry it left as the plain
open `Status:` line — before the successor leg on `return-partial`, before
Phase 3 on PASS or NEEDS_DECISION — so no leg inherits a red a cite could
have closed (one run carried two such entries through nine legs); a
RESOLVED decision that alters code is the successor's first item, in
RESUME. PASS and NEEDS_DECISION both advance to Phase 3: it verifies code,
not decisions. The gate is never the next step after Phase 2.

PHASE 3 — Self-verification (3a and 3b, 3d for what the last packet left open, then fixup if needed)

SERIALIZE 3a AND 3b: run 3a to completion, then 3b. They are independent by
BRIEF, not by timing — both append `<art>/verification.md`, which sits next to
the RDR and is shared whatever the worktree arrangement, so concurrent appends
lose one side's entries with nothing recoverable from git. Serializing costs
nothing in rigor. (Two agents may run concurrently only when they share no
mutable state.)

PHASE 3a [DELEGATE to sub-agent: "CoVe verifier"]
Brief: `briefs/phase-3a.md` (RDR path and `<art>/req-list.md` only; NO
tests, NO other ledger). Sub-agent's task: for each REQ-N, name an input that
would make a correct implementation visibly violate it. Then run
those inputs against the actual implementation (source tree in hand,
Phase 1 tests unread). Any actual violation gets appended to `<art>/verification.md` as a
FAIL-N entry with the failing input and observed behaviour.
Sub-agent returns a §return-packet (verdict=BLOCK if FAIL-N; summary_50w lists the FAIL-N entries one line each).

PHASE 3b [DELEGATE to sub-agent: "Adversarial reviewer"]
Brief: `briefs/phase-3b.md` (the RDR's Failure Modes section,
`<art>/req-list.md`, the source tree; NO Phase 1 tests, NO Phase 3a
findings); extra fields TEST_DIR, TEST_FRAMEWORK. Sub-agent's task: as a senior reviewer who thinks this is
wrong, name the three most likely failure modes (anchored in the
RDR's Failure Modes section) and the test that catches each. Add
any missing tests to the test directory; confirm they fail against
the current implementation (else they don't actually catch the
failure mode). Append findings to `<art>/verification.md` as ADV-N
entries.
Sub-agent returns a §return-packet (verdict=BLOCK if any added test currently fails; summary_50w lists failure modes named, tests added, which currently fail against the implementation).
A 3a or 3b pass with no finding still appends a line-leading
`## Verdict — clean` under its heading: the gate reads
`verification.md`, and an absent file is an unrun Phase 3, a named stop.

PHASE 3d — DECISION GROUNDING [conditional, DELEGATE: one read-only
"decision grounder" per `Status: needs author decision` entry in
`<art>/deviations.md`]
Runs at every Phase 2 packet boundary and after 3c (PHASE 2's rule), on the
entries that packet left open; a RESOLVED line is closed, never re-grounded.
Brief: `briefs/phase-grounder.md`; extra field ENTRY (that one entry,
verbatim; NO other entry, NO Phase 3 finding). Sub-agent's task: walk
rdr-common §ground-before-ask rung by rung (`rdr-gate ground` asks the
`ground` outcome) — code, cluster peers' Status/Overrides, corpus, RFD —
until `apply` or `ask`. It edits nothing; scratch under /tmp.
Sub-agent returns a §return-packet: verdict=PASS with the decision and
its cite in summary_50w (next_action `code-change` if the decision
alters code), or NEEDS_DECISION with `searched=` in next_action.
The orchestrator records each PASS by REWRITING that entry's `Status:`
line in place — the open form is the plain line `Status: needs author
decision` (qualifiers only inside a trailing parenthesis); closed is
`Status: needs author decision → RESOLVED (<cite>)` — never a note
appended below it. Survivors go to ONE rdr-common §strong-consult, spawned at
the resolved §model-ceiling (`$RDR_MODEL_CEILING` when the marker sets one) —
brief: the entries, their `searched=` trails, and the resources it may search
(`{RDR_RESOURCES}`, the source root) — recorded the same
way. Only its survivors reach the gate as open decisions: ask the user
one consolidated question listing each with the recommendation
(ESCALATION RULE) and rewrite the answers in the same form.

PHASE 3c — FIXUP [conditional, DELEGATE to sub-agent: "Phase 3 fixup"]
Run this only if Phase 3a returned FAIL-N entries OR Phase 3b added
tests that currently fail OR a 3d resolution after the last leg alters
code (earlier ones were a successor leg's first item) OR `<art>/triage.md`
has an open IN-SCOPE row (the gate's `stopped:findings-open`). Brief:
`briefs/phase-3c.md` (`<art>/verification.md`, `<art>/deviations.md`,
`<art>/triage.md`, `<art>/req-list.md`, source tree, `{RDR_RESOURCES}`);
extra fields TEST_FRAMEWORK, START_SHA, START_EPOCH (as Phase 2's).
Sub-agent's task: fix each defect and each open row, and apply each
RESOLVED decision that alters code, with the minimum change; add a
regression test if not already present. It commits through
`rdr-leg-commit` like a Phase 2 leg: the full suite is its last commit's
`--suite-green true`, and `return-partial` writes the capsule (`phase:
3c`) for a fresh 3c to resume from the ledgers. THE LEDGER'S ROW is a
defect in a clause the record already decided, reproduced at HEAD (the
review that writes it holds that test); the leg rewrites its Outcome to
`fixed:<sha>`, or to `held:contract (<clause>)` when the fix would ADD a
normative choice (a new code, exemption, unit) — the record did not decide
it, so the run does not; the review files it downstream. New deviations
follow Phase 2's classification rules (mechanical vs needs-author-decision);
a defect in code against a correct REQ is IMPL-GAP, 3c's native type.
Sub-agent returns a §return-packet (verdict=BLOCK if it cannot reach green,
INCOMPLETE only from `return-partial` — spawn a fresh 3c; summary_50w lists
defects fixed, rows fixed/held, regression tests added, green yes/no, any
new deviations needing author decision).
A new needs-author-decision entry takes Phase 3d again.

COMPLETION GATE (orchestrator runs directly — one call, no artifact reads)
  "$RDR_HOME/bin/rdr-gate" complete <NNNN> --tag suite_green=<true|false>    # the last packet's verdict: PASS → true (full suite, predecessors included)
The `complete` group proves every cell — Phase 3 recorded, green tests, no
orphans either way, REQ-MVV output recorded, no open needs-author-decision
line, no open IN-SCOPE row in `<art>/triage.md` (a downstream review
writes that ledger after the first COMPLETE; the gate is re-asked once it
exists) — and an unread ledger file or an unwritten `verification.md` is a
named stop, never a pass. `stopped:findings-open` is the one stop the run
finishes itself: PHASE 3c on the open rows, then re-ask. Write
`<art>/status.md` state from
`next:` as a value: `COMPLETE`, or `INCOMPLETE — <stopped:token>: <why>`.

Do not declare success on INCOMPLETE.

RESUME CAPSULE (orchestrator writes status.md as the cheap one-pass
resume state). Each phase boundary, the completion gate, AND a Phase 2
leg's `return-partial` overwrite `<art>/status.md` with the fixed header
block below, then the verdict line. Overwrite, never append. This is the ONLY Stage-8 durable read on
re-entry — keep it ≤12 lines so a new session rehydrates in one read.

```text
RDR: NNNN-slug | phase: <N — name> | state: COMPLETE | INCOMPLETE | IN-PROGRESS
last: <last completed phase action, ≤1 line>
blocker: <named blocker | none>
changed: <comma-sep paths touched this run | none yet>
validate: <exact test/suite command>
baseline: <green | red> @<commit>       # PRECHECKS' full-suite run, before Phase 1
next: <exact next phase or $rdr-status NNNN re-entry command>
reads: <≤10 path[:range] the last leg found load-bearing | none>
session: <ISO8601 ts | session id>
artifacts: req-list.md impact.md coverage.md verification.md deviations.md
```

GUARDRAILS
- Phase sub-agents own their own reading (your role, above): re-reading
  the same files in two contexts is the bug we are avoiding.
- You never `cd`: a leaf's checkout is addressed from where you stand —
  `git -C <wt> …`, `go test -C <wt> …` (Go ≥ 1.20). A cwd left in a worktree
  once pinned a resumed session for 37 minutes.
- A multi-line script is written to a file, then run (`sh /tmp/x.sh`);
  inline heredocs and `python -` bodies are what a guard refuses.
- You never run the suite after PRECHECKS' baseline — `./...` least of all.
  The §return-packet's `suite_green` and command line are the evidence; a
  doubt is one package-scoped run (`go test -C <wt> ./<pkg>/`).
- Every phase return is a §return-packet; the orchestrator rejects a
  malformed packet and re-asks for the packet alone, not a re-run.
- If a test needs information not in the spec, the sub-agent first
  grounds the gap against `{RDR_RESOURCES}` (the spec's own evidence
  base) and ultrathinks whether that evidence resolves it; only if it
  cannot does it record a `SPEC-UNDER` "needs author decision" entry
  and return. Ground, don't invent — and don't escalate what the
  evidence already answers. If the spec is wrong, the sub-agent
  records a `SPEC-DEFECT` (citing the contradicting evidence); the
  orchestrator surfaces it as a user question, and
  only on the user's confirmation does it mark INCOMPLETE("spec
  defect") and halt without editing the spec.
- A reviewer reading only the test headers should see the spec
  re-stated.
- Sub-agent independence is enforced by brief, not honour:
  Phase 1 brief excludes implementation hints; Phase 2 brief
  excludes Phase 3 findings; Phase 3a brief excludes
  implementation reads of the Phase 1 tests; Phase 3b brief
  excludes Phase 3a findings; a Phase 3d brief carries one entry and
  no finding.
```

## Predecessor Convention

When an RDR's tests or code will reference behaviour pinned by an
earlier RDR, list the load-bearing predecessors in the RDR's Metadata:

```markdown
- **Predecessors**: 0001-core-schema, 0003-project-config
```

The launch prompt's PRECHECKS step gates on each predecessor's
`status.md` being `COMPLETE` (hard stop otherwise) and surfaces their
`req-list.md` + `deviations.md` to the Phase 1 sub-agent, and their
test files to Phase 2. Predecessor test files in the source tree are
the executable ground truth — read like any other code, not as a
launch artifact. Predecessor
`verification.md` and `coverage.md` do not feed forward; they were
closed during that RDR's launch.

The integration layer is the spec author: when writing RDR N+1, cite
predecessor REQ-Ns where N+1 depends on them in the RDR text. The
prompt only needs to know which predecessors to gate on.

## Design Notes

- **REQ-N quote IDs commit the LLM to the spec text** before any drift
  can start. Every later phase cites them.
- **Red-before-green (Phase 1 gate) is the load-bearing rule.** A
  test green against a stub is tautological by definition; this gate
  is what distinguishes "tests verify the spec" from "tests verify
  what the LLM wrote."
- **Orchestrator-of-sub-agents architecture is what keeps the main
  context small.** Every phase that touches the RDR text, source tree,
  or test files runs in its own sub-agent whose context is discarded on
  return, and the `budget` rows bound each Phase 2 leg the same way — a
  fresh leg per cut, resuming from the on-disk artifacts (`<art>/*.md`),
  the durable record between phases.
- **Sub-agent briefs enforce independence structurally** (GUARDRAILS
  lists the exclusions; `briefs/` templates carry them, filled, never
  authored). Asking a single agent to "ignore what you just
  wrote" is aspirational — isolation is enforceable only via
  brief-scoping at spawn time.
- **Hard COMPLETE/INCOMPLETE gate** makes "done" computable from disk
  (the `complete` row), which is what manual sequencing across long
  gaps requires.
- **Escalation to the user is for design decisions, not continuation**
  (the ESCALATION RULE above).
- **Inline fast-path for small RDRs** is the PRECHECKS **Size gate**
  above — the `size` row of `$RDR_HOME/models/rdr-launch.toml`
  (profile=`small` + hard caps), not an ad-hoc "trivial surface" judgment
  (FALLBACK re-asks it). The red-before-green gate still applies inline.
- **The Phase 2 deviation Types** (SPEC-DEFECT / SPEC-UNDER /
  DEPENDENCY-LIMIT / TEST-FIXTURE / IMPL-DECISION / IMPL-GAP) are the same
  taxonomy the RDR process uses for post-mortem drift classification
  (`$RDR_HOME/README.md`, *Post-Mortem Process*). They are defined inline
  in the prompt above so this file stays standalone-pasteable; keep the
  two definitions in sync. In the RDR flow ([`$RDR_HOME/stages/`]($RDR_HOME/stages/README.md)),
  this prompt is **Stage 8 (Implement)** — the flow's terminus.

## Citations

- TiCoder, TSE 2024 — tests-from-spec-first, ranking by test-consistency.
  <https://www.seas.upenn.edu/~asnaik/assets/papers/tse24_ticoder.pdf>
- Dhuliawala et al., *Chain-of-Verification*, ACL Findings 2024 —
  independence beats CoT/few-shot on hallucination.
  <https://arxiv.org/abs/2309.11495>
- Addy Osmani, *How to write a good spec for AI agents*, 2025 —
  conformance testing + per-test spec citation pattern.
  <https://addyosmani.com/blog/good-spec/>
