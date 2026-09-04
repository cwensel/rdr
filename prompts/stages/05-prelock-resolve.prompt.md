Paste 1 — arg header (edit only the values):

RDR: {RDR_PATH}
LENS: <3amigo | critique | repeatability | cove>

Paste 2 — resolver + fix body (verbatim):

From the arg header above, bind for this session:
  - {RDR_PATH} = RDR:; <rdr-slug> = its filename stem; <lens> = LENS:.
  - {RDR_ENV} / {RDR_RESOURCES} = the seam files. Bind them with
    `eval "$("$RDR_HOME/bin/rdr" env)"` → `$RDR_ENV` / `$RDR_RESOURCES`. It
    applies nearest-wins (a repo-local `.rdr/workspace` beats the shared
    `$WS/.rdr-workspace`), so **never** source `$WS/.rdr-workspace` directly —
    under a repo-local marker that binds another project's seam. Keep it in the
    same shell as any command reading `$RDR_ENV`/`$EVIDENCE_DIR` (state dies
    between calls — re-run, don't carry).
  - {EVIDENCE_DIR} = the lens-output dir, from
    `eval "$("$RDR_HOME/bin/rdr" paths --lens <lens> --next-iter <NNNN>)"`.
    Read the findings from `$PRIOR_DIR` — the pass that just ran wrote there
    (the highest iteration on disk; the base after a first pass). `$ITER_DIR`
    is where a re-run would write, `$ITER_BUCKET` the loop's pass tag.

For the RDR at {RDR_PATH}, resolve each finding one by one. If you were handed
the findings inline (a return packet's ledger — the norm under `--auto`), that
IS the origin ledger: work from it, and open the file below only for the
adjudication prose behind a row. Otherwise read the findings from
{EVIDENCE_DIR} — its consolidation / findings / diff file (e.g.
<slug>/evidence/3amigo/consolidation.md, <slug>/evidence/cove/findings.md,
<slug>/evidence/repeatability/diff.md).

CRITIQUE LEDGER (when <lens> = critique): `critique.md` opens with a `C-N` ledger —
use it as the origin ledger as-is, don't rebuild one. Each row's **Origin**
(`§1`/`§2`/`§3`/`premortem`/`AT-N`) points into the prose below, where the reasoning
that justifies the row lives — read it before dispositioning, and rank
premortem/AT-origin rows equal to the rest. A defect the prose raises but no row
indexes still gets resolved (say so; it's a ledger bug, not a skip). `C-N` is a
per-file row handle; across files the key is the element id — dual-model runs
diff `LC_ALL=C comm` over `"$R" anchors --record <NNNN>` of `critique.md` vs
`critique-modelB.md` (`$R="$RDR_HOME/bin/rdr"`; two findings on one id collapse).

Each finding passes the GROUNDING GATE before it can edit the draft, gets a
durable DISPOSITION, and is anchored to an origin concern. Do these in order; the
loop + cap that decide whether to re-run the lens live in the skill, not here.

GROUNDING GATE (before any edit — the anti-flapping core). Ground each finding
against three sources before actioning it; one that fails any ground is wrong —
dismiss-with-reason, never silently drop:
  1. **Code on `main`** — confirm any cited symbol/behavior (signature, missing
     default, rule, call site) exists as described; if it doesn't, the finding is
     wrong — don't edit to satisfy it. (Delegate the lookup; want
     verdict + `path::Symbol`, not raw hits.)
  2. **{RDR_RESOURCES}** — read it first; check every fix against its default-load
     principles doc + the internal contract docs, and use its named corpora to
     research any finding turning on external behavior or complex design.
  3. **The RDR's own decided text** — a finding re-litigating an already-adjudicated
     option ("chose X, declined Y") is a **re-raise**, not a defect: dismiss citing
     the decision, and make that decision explicit in the draft if it was implicit
     (the #1 flapping cause in the corpus is re-litigating settled calls).

COMPUTE, DON'T ARGUE. A fix claiming agreement with something pinned — a fixture,
count, census, or another clause — writes the comparison's **executed result**,
not reasoning about it. "Satisfies X by construction", "the two agree" are
arguments; run the subtraction or diff and write what it returned. Can't execute
it → book an assumption, don't close it. Also check the clause against fixes an
earlier lens in this row already made to it: the row edits a shared draft and
nobody re-reads it whole, so two lenses can pin one field two ways. Both shipped
defects. Limit: this catches a false claim inside a sound frame, not a wrong
frame — a count re-measured in the wrong unit returns correct and wrong. Findings
that keep *widening* one enumeration (6→7→9→…) are that signal: question the unit.

ORIGIN ANCHOR (anti-plank). On the first pass, the findings *are* the
originating concerns — the loose files under `$EVIDENCE_DIR` are the ledger
(critique ships one; other lenses' findings files are it). Every finding you act
on traces to a ledger entry; one that traces to none is **net-new scope** — do
not let it quietly expand this RDR (the "scope-expansion wormhole"). On a
re-run the trace is a diff, not a re-read (`$PRIOR_DIR` is this pass, from the
same `paths` eval):
  ```sh
  R="$RDR_HOME/bin/rdr"; L=$(mktemp); N=$(mktemp)
  "$R" anchors --record <NNNN> "$EVIDENCE_DIR"/*.md > "$L"   # origin ledger
  "$R" anchors --record <NNNN> "$PRIOR_DIR"/*.md > "$N"      # this pass
  LC_ALL=C comm -13 "$L" "$N"   # net-new scope
  LC_ALL=C comm -12 "$L" "$N"   # still-open
  ```
**Delta-scope to the still-open set** — do not author a fresh full critique of
the rewritten draft; critiquing your own edits is exactly the
critique-on-critique drift this guards against.

DISPOSITION (every finding exits exactly one way — no silent drops):
  - **fixed** — grounded, in-scope, edited into the draft.
  - **dismissed-with-cite** — failed the grounding gate (code absent, principle
    forbids, or already-decided); record the one-line reason + the source.
  - **charted-to-successor** — real but net-new scope: record it durably, then
    dismiss it from this loop citing where it landed. Durable means a `Charted:`
    line in this lens's {EVIDENCE_DIR} folder (one line: finding + why out-of-scope +
    suggested successor); for a substantial follow-up, say so in the
    needs-from-me output so the driver can `/rdr-seed` it. Never edit the current
    RDR to absorb net-new scope — that is the scope-expansion wormhole.

REPEATABILITY DIFF (when <lens> = repeatability): a diff finding is a
DISAGREEMENT, not yet a defect — the runs diverged because the RDR underspecified
or said it twice. Disposition each as one of: **pin** the contract (state the one
shape — the common right fix); **cut** the prose that let a weaker model invent
(un-pinned prose is the leak vector — a cross-model `[boundary split]` is the
strongest signal, resolve it first); **single-source** a contract the RDR states
in two places that drifted; **leave non-normative** when the divergence is a
legitimate impl detail (identity/format/naming RDRs underspecify *by design* — the
RDR *class* decides; don't specify it away, that's the over-spec trap); or
**tiebreaker** when the call is genuinely yours. Resolve findings that land in the
same RDR section as one edit, not strictly one-by-one. An "unhealthy" diff
(identical-but-confidently-wrong, no GUESS markers) is a *rerun the lens on
another model* disposition, not an edit.

For heavy reads — corpus searches, dependency/source spelunking, several
round-output files — delegate to a sub-agent that returns finding + `path::Symbol`
+ verdict, NOT raw hits or whole files. Keep the design judgment here.

FLAG-AS-YOU-GO (load-bearing): the review rounds disturb assumptions. As you
resolve findings, whenever a fix TOUCHES or ADDS a load-bearing claim (a
normative signature, a wire/byte format, an external-behavior claim, an exactness
word like all/every/exact/canonical/deterministic), record it in a running
"needs (re)verification" list:
  - Fix invalidates an assumption previously Verified → flip it back to
    Status: Pending with a one-line reason.
  - Fix introduces a NEW load-bearing claim → add a Critical Assumption A-N
    (Status: Pending) with a one-line verification plan (Method + how).
  - A finding explicitly calls for a spike → note the spike to run.
Do NOT verify them now — Stage 6 closes them. Just ensure none is silently
absorbed into the draft as if already true.

AMENDMENT SWEEP: rdr-common §amendment-sweep, before each fix's disposition closes.

Be brief — terse reasoning, tight edits, no change-history narration (keep
rationale for decisions, not a log of edits). Edit mechanics: replace the exact
bytes you just projected with `--select NNNN:<id>`, never text recalled from
squeezed/grepped output (whitespace drifts and the replace misses); portable
tools only — no GNU-only `sed` addresses (`N,+Mp`) or `cat -A`.

TIEBREAKER-REDUCTION GATE. **Ultrathink** before applying any load-bearing /
cross-subsystem / structural / principle-touching / intent-conflicting finding,
and use that reasoning + the grounding evidence to **collapse the fork yourself** —
most apparent either/ors dissolve once the evidence is on the table. Still
indeterminate → rdr-common §ground-before-ask, then §strong-consult; escalate to me ONLY on its
NEEDS_DECISION or when design intent truly conflicts, not as the default for
a hard call. (Cross-model repeatability
independence is the one fork you can't collapse alone — the lens handles it.)

Output: per finding, one line — disposition (fixed / dismissed-with-cite /
charted-to-successor / needs-tiebreaker) + the origin-ledger entry it traces to +
section touched. Then the needs-verification list, then any tiebreakers you
genuinely could not collapse with reasoning + evidence.
