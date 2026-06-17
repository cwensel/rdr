Paste 1 — arg header (edit only the values):

RDR: {RDR_PATH}
LENS: <3amigo | critique | repeatability | cove>

Paste 2 — resolver + fix body (verbatim):

From the arg header above, bind for this session:
  - {RDR_PATH} = RDR:; <rdr-slug> = its filename stem; <lens> = LENS:.
  - {RDR_ENV} / {RDR_RESOURCES} = the seam files. Resolve their location via the
    workspace marker (see the flow README *Where the seam lives*):
    `WS=$(dirname "$(dirname "$(cd "$(git rev-parse --git-common-dir)" && pwd
    -P)")")` then `. "$WS/.rdr-workspace"` → `$RDR_ENV` / `$RDR_RESOURCES`. Keep
    this in the same shell as any command reading `$RDR_ENV`/`$FLOW_DIR` (state
    dies between calls — re-run, don't carry); a direct source exits 1 without
    `$WS`. Only if no marker and no {RDR_ENV} exist does `_rdr/` apply.
  - {FLOW_DIR} = the lens-output dir {RDR_ENV} resolves to for <lens>/<rdr-slug>
    (highest iter-N if it has iter-* subfolders).

For the RDR at {RDR_PATH}, resolve each finding one by one. Read the findings
from {FLOW_DIR} — its consolidation / findings / diff file (e.g.
3amigo/<slug>/consolidation.md, cove/<slug>/findings.md,
repeatability/<slug>/diff.md). If you were handed findings inline instead, use
those.

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

ORIGIN ANCHOR (anti-plank). On the first pass, the findings *are* the
originating concerns — keep them as a ledger. Every finding you act on traces to
a ledger entry. A finding that traces to none is **net-new scope** — do not let
it quietly expand this RDR (the "scope-expansion wormhole"). **On a re-run,
delta-scope to the still-open ledger entries** — do not author a fresh full
critique of the rewritten draft; critiquing your own edits is exactly the
critique-on-critique drift this guards against.

DISPOSITION (every finding exits exactly one way — no silent drops):
  - **fixed** — grounded, in-scope, edited into the draft.
  - **dismissed-with-cite** — failed the grounding gate (code absent, principle
    forbids, or already-decided); record the one-line reason + the source.
  - **charted-to-successor** — real but net-new scope: record it durably, then
    dismiss it from this loop citing where it landed. Durable means a `Charted:`
    line in this lens's {FLOW_DIR} folder (one line: finding + why out-of-scope +
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

Be brief — terse reasoning, tight edits, no change-history narration (keep
rationale for decisions, not a log of edits).

TIEBREAKER-REDUCTION GATE. **Ultrathink** before applying any load-bearing /
cross-subsystem / structural / principle-touching / intent-conflicting finding,
and use that reasoning + the grounding evidence to **collapse the fork yourself** —
most apparent either/ors dissolve once the evidence is on the table. Escalate to
me ONLY when the evidence is genuinely indeterminate or design intent truly
conflicts, not as the default for a hard call. (Cross-model repeatability
independence is the one fork you can't collapse alone — the lens handles it.)

Output: per finding, one line — disposition (fixed / dismissed-with-cite /
charted-to-successor / needs-tiebreaker) + the origin-ledger entry it traces to +
section touched. Then the needs-verification list, then any tiebreakers you
genuinely could not collapse with reasoning + evidence.
