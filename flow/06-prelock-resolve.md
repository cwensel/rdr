# Stage 6 — Pre-Lock Resolve (the fix pass)

**Goal**: turn a Stage 5 lens's findings into edits to the RDR — one at a
time, grounded in the project's design contracts and the literature. This is
the **fix pass that runs after every Stage 5 lens** (and after any review that
emits a findings list). The most-run prompt in the flow. Named
`06-prelock-resolve` to bind it to the Stage 5 Pre-Lock lens it resolves.

## Paste this

**Same two-paste arg-header convention as Stage 5** (the TUI collapses a pasted
body to one line). No lens body to splice — the header is just RDR + lens.

**Paste 1 — arg header** (edit only the values):

```text
RDR: {RDR_PATH}
LENS: <3amigo | critique | repeatability | cove>
```

**Paste 2 — resolver + fix body** (verbatim):

```text
From the arg header above, bind for this session:
  - {RDR_PATH} = RDR:; <rdr-slug> = its filename stem; <lens> = LENS:.
  - {RDR_RESOURCES} = _rdr/rdr-resources.md.
  - {FLOW_DIR} = _rdr/<lens>/<rdr-slug>/ (highest iter-N if it has iter-* subfolders).

For the RDR at {RDR_PATH}, resolve each finding one by one. Read the findings
from {FLOW_DIR} — its consolidation / findings / diff file (e.g.
3amigo/<slug>/consolidation.md, cove/<slug>/findings.md,
repeatability/<slug>/diff.md). If you were handed findings inline instead, use
those.

Read {RDR_RESOURCES} first and pull in the design docs + corpora it names that
bear on these findings — principally its default-load principles doc (every
fix is checked against it) and the internal contract docs it lists. Use the
named corpora for research when a finding turns on complex design or external
behavior.

For heavy reads — corpus searches, dependency/source spelunking, several
round-output files — delegate to a sub-agent that returns finding + file:line
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
Do NOT verify them now — Stage 7 closes them. Just ensure none is silently
absorbed into the draft as if already true.

Be brief — terse reasoning, tight edits, no change-history narration (keep
rationale for decisions, not a log of edits). Ultrathink ONLY for
load-bearing or complex-design findings. Come to the user for tiebreakers —
genuine either/or design calls.

Output: per finding, one line — disposition (fixed / deferred-with-reason /
needs-tiebreaker) + section touched. Then the needs-verification list, then
any tiebreakers you need from me.
```

**Run when**: a Stage 5 lens — or a Stage 7 reconcile, or a Stage 8 readiness
check — produced a list of blockers/issues. **Produces**: edits to the RDR
draft + a running **needs-verification list** (see flag-as-you-go).

## Review gate

- **Did it stay brief and tight?** Edits shouldn't bloat the RDR or narrate
  the change. A draft that grew a change-log goes back.
- **Were principles actually consulted?** A fix that violates a principle
  (silently ignores an error, breaks determinism) is wrong even if it closes
  the finding. Spot-check against the named principle.
- **Is the needs-verification list honest?** This is Stage 7's safety valve.
  A fix that clearly touched a normative claim but left the list empty means
  flag-as-you-go wasn't followed — re-run. An empty list on a pure
  wording/proportionality round is fine.
- **Tiebreakers surfaced, not silently decided?**

Re-run for the subset of findings not yet resolved, or a single finding whose
fix you reject.

## Advance when

Every finding is dispositioned (fixed / deferred-with-stated-reason /
tiebroken-by-you), the needs-verification list is complete, and no fix
violates a project principle.

→ Back to [05-prelock.md](05-prelock.md) for the next lens, or — all lenses
done — on to [07-reconcile.md](07-reconcile.md).
