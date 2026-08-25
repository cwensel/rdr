# Tooling Pass — RDR Mechanical Adherence Sweep

**Use when**: the **mechanical pre-step of the Finalization Gate**
([Stage 7]($RDR_HOME/stages/07.0-finalize.md)), run on **every** RDR just before the
Gate's written responses — after whatever pre-lock rounds the RDR's profile
called for (a small RDR runs none) and after the Stage 6 spike/assumption
reconcile. It is a *post-mutation regression sweep*: the lenses and fix-passes
rewrite the draft and the reconcile flips assumptions, any of which can hollow a
section or disturb an evidence record an earlier stage had clean. For a small
RDR that ran no lens it is the *only* mechanical check; for any RDR it is the
conformance backstop when Refine/Resolve was skimped.

**Status**: Mostly scripted. C1, C2, C5, C6 and C10 are projections of `rdr`;
C3, C4 and C9 remain judgment on a narrowed read.

**Cost**: seconds via `rdr`; ~5 min when it is absent.

## Run first

```bash
[ -x "$RDR_HOME/bin/rdr" ] || echo "stopped:projector-not-built"
"$RDR_HOME/bin/rdr" lint --locking --records "$RDR_RECORDS" --repo "$RDR_SOURCE_REPO" {RDR_NUMBER}
"$RDR_HOME/bin/rdr" inspect --json --records "$RDR_RECORDS" --repo "$RDR_SOURCE_REPO" {RDR_NUMBER}
```

`lint` exit 1 = at least one blocking finding (printed with a leading `!`).
Exit 0 with findings = advisory only. Read the findings; do not re-derive them.

If the binary is absent, run every check as prose against the RDR, TEMPLATE.md
and README.md at the root of the rdr/ tree, per the rules below.

## Prompt

```text
Walk the RDR at {RDR_PATH}. Read this as a regression sweep: the review rounds
and reconcile just rewrote this document — confirm none of them hollowed a
section or disturbed an assumption's evidence.

Report findings only. Do not edit the RDR.

CHECK 1 — Template section coverage  (primary signal)
MISSING: `lint`'s `template:missing-section` findings ARE the Required-section
set difference, already computed with this check's exclusions (Conditional
sections omitted by design; a subsection under an absent parent; a gate
subsection under a `gate.md` pointer). They are `conformance` tier — advice for
the stage holding the file open, never blocking on their own. Judge whether one
is a genuine spine hole: that is a BLOCK.
HOLLOW stays prose: walk `outline[]` (`canonical`, `class`, `line_start`,
`line_end`) and read the ranges. Hollow = TBD, "see above," single-sentence
placeholder (`_Draft placeholder._`), or copy-paste of template instructional
text, from any template version. A surviving `this is a seed skeleton` header is
an automatic Present-hollow on the Finalization Gate section — name it. (At/after
lock that section holds only the pointer line to gate.md — that is the record,
not a hollow section.) A surviving verbatim template bracket — `[Conditional —
…]`, `[Resource]`, `[Capability]`, any bracketed template instruction — in a
non-Draft (non-locking) RDR is a BLOCK. List every hollow and missing section by
name.

CHECK 2 — Method label vocabulary
For each element field with `label == "Method"`, read `method.off_vocabulary[]`:
non-empty means an unsanctioned member, and it names itself. Gloss is already
stripped and `+` already split (`Spike (repro)` → members `["Spike"]`), so gloss
is never a finding. Flag only the off-vocabulary member, by name. An Evidence
Record with NO Method field is a finding too — the field is simply absent from
that element's `fields[]`. Watch for records ADDED or relabeled during the
rounds. (The eight labels are asserted against README.md §Verifying load-bearing
claims by the model's own tests; do not restate them.)

CHECK 3 — Source Search self-reference  (rare; verify, don't hunt)
The projection narrows the read only: `method.members` containing `Source
Search` names WHICH records to look at; whether the Evidence path is
self-referential stays judgment. Resolve each Evidence path — if it resolves to
{RDR_PATH} itself or any path under this RDR's artifact directory, it is
self-reference and not Verified. List offenders. NOTE: this has not fired since
the structured Evidence Record and the Resolve stage landed — a hit means an
evidence record was disturbed after Resolve, so treat it as a real regression.

CHECK 4 — Docs Only on load-bearing claims
Same shape: `method.members == ["Docs Only"]` names the candidates; whether the
claim is load-bearing is judgment that stays prose. List every Docs Only record
whose Evidence line lacks a Spike or Source Search plan. These block lock per
the Finalization Gate — and are a common artifact of a fix-pass that added a
claim without verifying it.

CHECK 5 — Symbol resolution of Source Search / Spike anchors
`edges[]` where `kind == "source-anchor"` IS this check; `lint --repo` reports
the failures as blocking `edge:unresolved`. The rule the tool applies is this
check's rule exactly — the SYMBOL resolves, never the line number, and a symbol
found anywhere in the repo counts (a move is a note, not a finding). `resolved`
is THREE-valued: true / false / ABSENT. ABSENT means no `--repo` was supplied
and nothing was looked for — report those SKIPPED, never as pass or fail. A bare
`file:line` with no symbol emits no source-anchor edge at all; that absence is
itself a finding (rewrite as `path::Symbol`). A stale line number alone, where
the symbol still resolves, is a NON-finding and does not block Final.
Corpus-wide: `rdr index --unresolved --records "$RDR_RECORDS" --repo "$RDR_SOURCE_REPO"`.

CHECK 6 — Status consistency  (projection-narrowed)
`metadata[]` Status carries `status.{value,qualifier,form,tier}`; each
assumption's Status field carries `status.value` — read those, never re-parse
the markdown. Then judge: list any assumption whose `status.value` is `Pending`
or `Unverified` whose property is then relied on as a settled fact in prose
elsewhere in the RDR. Also flag any place a checklist box and the Finalization
Gate (inline or `{ARTIFACT_DIR}/gate.md`) disagree about the same assumption's
status — these cannot both be right. (Transitional: once a contract is
single-sourced there is no second copy to disagree.) `rdr index --status` groups
the corpus for a cross-record question.

CHECK 9 — Evidence-field budget  (ADVISORY, never blocks)
TEMPLATE.md specifies the Evidence field as a single sentence naming a stable
anchor; the corpus median is 14 lines. Each Evidence field carries `line_start`
and `line_end`, so its length is `line_end - line_start + 1` — arithmetic, not a
read. List every field over **30 lines**, longest first, with its assumption ID
and count, then one total: `N fields over budget, M lines`. This is a REPORT:
over-budget fields alone never make the verdict BLOCK. Do NOT propose truncation
— the mass is usually real verification content (source-verified positions,
documented exceptions, consumer censuses), and cutting it blinds the grounding
sweep that reads those anchors. Ask one question per hit: has the prose outgrown
the *record* — is the load-bearing anchor still findable, and does the balance
belong in `{ARTIFACT_DIR}` with the field keeping the anchor and a pointer? The
author answers it per assumption at the Gate.

CHECK 10 — Linking: labelled contracts and resolvable citations
This check IS `rdr lint`. Report its findings; do not re-read for them.
  - `label:contracts` (advisory) — a normative block with no `**Cn**` label;
    the rewriting stage labels them C1..Cn in-pass. `label:contracts-required`
    is the same finding on a record postdating the rule (no grandfathering).
  - `peer-evidence:no-element` [blocks] — a `Method: Peer RDR` Evidence naming
    a bare record, not an ELEMENT (`cli/0055:C4`): a citation landing on a
    whole file names a document, not a reason.
  - `edge:unresolved` [blocks] — a typed reference (Predecessors, Overrides,
    Cluster, Peer-RDR Evidence, joint-decision home, source anchor) looked for
    and not found.
  - `edge:unresolved-terminal` — the same in a TERMINAL record: a data error in
    the citing record, non-blocking because that record is not the one locking.
    Fix the pointer text in the named range, nothing else.
Without `rdr`, check the Peer-RDR citations by hand and report the rest
SKIPPED — never as a pass.

Output: one bullet per finding, prefixed with the check ID (C1–C6, C9, C10)
and the section or assumption ID. End with a one-line verdict:
  PASS — no blocking finding; proceed to the Gate's written responses.
  BLOCK — N blocking findings, lock prohibited until resolved.
```

## Expected signal

- **PASS** — no blocking finding; proceed to the Gate's written responses. A run
  whose only findings are advisory — C9, and lint's conformance tier on a sound
  spine — is a PASS carrying a report.
- **BLOCK** — at least one. Most are C1 (a section the rounds hollowed) or
  C2/C4 (an assumption a fix-pass disturbed). Fix the RDR and re-run. C3 is rare
  and signals a disturbed evidence record — treat it seriously, not as routine.
  C5 "resolves nowhere" is the phantom-API class (cited code not on `main`); C6
  is an internal status contradiction.

## Out of scope for this pass

The README rule that *every external API call inside a Normative block must have
a corresponding Critical Assumption Evidence Record above* is intentionally not
mechanized here — it requires linking call-sites to records, an analytical
judgment. Verify it during 3amigo (Implementer persona) or Action-items
(Repeatability Probe). A future CHECK 7 can absorb it once the matching
heuristic is reliable. (C9 is taken — Evidence-field budget; C10 — linking.)

A future CHECK 8 (cheap regex guard, no script yet) can flag any unqualified
`ScheduleWakeup` / `wakeup` / `heartbeat` phrasing introduced into RDR skills or
prompts — i.e. a wakeup that does not name the external uncertainty it guards and
its self-clear (per `rdr-common.md` §no-heartbeat).

## Source

These checks are the mechanical share of the implementation-prompt review lens,
isolated from the analytical share so a script could absorb them — see
`.rdr/RDR-PROCESS-IMPROVEMENT.md` §D.2, which places this sweep last, immediately
before the Gate. Most now have: `rdr` is that script. C3 originates in the X4
triage (`action-items/X4-triage-report.md`), which found 24/54 assumptions
self-referencing across RDRs 0001–0010 — RDRs authored *before* the structured
Evidence Record and the Resolve stage existed; both now prevent that failure at
the source, which is why C3 is a regression check, not a hunt.
