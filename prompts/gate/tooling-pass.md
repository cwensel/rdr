# Tooling Pass — RDR Mechanical Adherence Sweep

**Use when**: the **mechanical pre-step of the Finalization Gate**
([Stage 7]($PROCESS_ROOT/rdr/flow/07.0-finalize.md)), run on **every** RDR just before the
Gate's written responses — after whatever pre-lock rounds the RDR's profile
called for (a small RDR runs none) and after the Stage 6 spike/assumption
reconcile. It is a *post-mutation regression sweep*: the lenses and fix-passes
rewrite the draft and the reconcile flips assumptions, any of which can hollow a
section or disturb an evidence record an earlier stage had clean. For a small
RDR that ran no lens it is the *only* mechanical check; for any RDR it is the
conformance backstop when Refine/Resolve was skimped.

**Status**: Not yet implemented as a script. Run as the AI prompt below until
it is; it is the round most worth scripting (seconds, every RDR).

**Cost**: ~5 min via AI; seconds when scripted.

## Prompt

```text
Walk the RDR at {RDR_PATH} and run these mechanical checks against TEMPLATE.md
and README.md at the root of the rdr/ tree. Read this as a regression sweep:
the review rounds and reconcile just rewrote this document — confirm none of
them hollowed a section or disturbed an assumption's evidence.

Report findings only. Do not edit the RDR.

CHECK 1 — Template section coverage  (primary signal)
For each top-level and second-level section in TEMPLATE.md, mark the RDR's
corresponding section as: Present-substantive | Present-hollow | Missing.
Hollow = TBD, "see above," single-sentence placeholder (`_Draft placeholder._`),
or copy-paste of the template instructional text. A surviving `this is a seed
skeleton` header is an automatic Present-hollow on the Finalization Gate section
— name it. List every Present-hollow and Missing entry by section name. (This is
where post-rewrite regressions show up most.)

CHECK 2 — Method label vocabulary
For every Critical Assumption Evidence Record, confirm the Method is exactly
one of the eight sanctioned labels:
  Source Search | Spike | Prior Art | Derivation |
  Design Decision | Peer RDR | MVV Test | Docs Only
List every record whose Method is missing, paraphrased, or off-vocabulary —
watch for records ADDED or relabeled during the rounds.

CHECK 3 — Source Search self-reference  (rare; verify, don't hunt)
For every Source Search record, resolve the Evidence path. If it resolves to
{RDR_PATH} itself (or any path under this RDR's artifact directory), it is
self-reference and not Verified. List offenders. NOTE: with the structured
Evidence Record and a disciplined Resolve stage upstream, this check has not
fired in practice since those landed — a hit here means an evidence record
was disturbed after Resolve, so treat any offender as a real regression.

CHECK 4 — Docs Only on load-bearing claims
List every Docs Only record whose Evidence line lacks a Spike or Source Search
plan. These block lock per the Finalization Gate — and are a common artifact
of a fix-pass that added a claim without verifying it.

CHECK 5 — Symbol-resolution of Source Search / Spike anchors
For every Source Search (and any Spike citing source), confirm the Evidence is a
greppable `path::Symbol`, not a bare `file:line`. Grep the symbol: resolves in
the cited file → ok; elsewhere in the repo → note the move; resolves NOWHERE →
flag (the phantom / never-built / renamed class). Do NOT check line numbers. A
bare `file:line` with no symbol is itself a finding (rewrite as `path::Symbol`).

CHECK 6 — Status consistency  (grep-able)
List any assumption marked `Pending` or `Unverified` whose property is then
relied on as a settled fact in prose elsewhere in the RDR (a "settled-fact"
sentence depending on an unsettled assumption). Also flag any place a checklist
box and the Finalization Gate disagree about the same assumption's status — these
cannot both be right. (The checklist-vs-gate sub-check is transitional: once a
contract is single-sourced there is no second copy to disagree.)

Output: one bullet per finding, prefixed with the check ID (C1–C6) and
the section or assumption ID. End with a one-line verdict:
  PASS — no findings; proceed to the Gate's written responses.
  BLOCK — N findings, lock prohibited until resolved.
```

## Expected signal

- **PASS** — every TEMPLATE section is Present-substantive, every Method is in
  the vocabulary, no `Source Search` self-references, no load-bearing
  `Docs Only`, every cited symbol resolves on `main`, and no settled-fact prose
  leans on an unsettled assumption. Proceed to the Finalization Gate's written
  responses.
- **BLOCK** — at least one finding. Most findings are C1 (a section the rounds
  hollowed) or C2/C4 (an assumption a fix-pass disturbed). Fix the RDR and
  re-run. A C3 finding is rare and signals a disturbed evidence record — treat
  it seriously rather than as routine. A C5 "resolves nowhere" finding is the
  phantom-API class (cited code not on `main`); a C6 finding is an internal
  status contradiction. Both block lock.

## Out of scope for this pass

The README rule that *every external API call inside a Normative block must
have a corresponding Critical Assumption Evidence Record above* is
intentionally not mechanized here — it requires linking call-sites to records,
an analytical judgment. Verify it during 3amigo (Implementer persona) or
Action-items (Repeatability Probe). A future CHECK 7 can absorb it once the
matching heuristic is reliable.

## Source

The four checks are the mechanical share of the implementation-prompt review
lens, isolated from the analytical share so a script can absorb them — see
`_rdr/RDR-PROCESS-IMPROVEMENT.md` §D.2, which places this sweep
last, immediately before the Gate. CHECK 3 originates in the X4 triage
(`action-items/X4-triage-report.md`), which found 24/54 assumptions
self-referencing across RDRs 0001–0010 — RDRs authored *before* the structured
Evidence Record and the Resolve stage existed; both now prevent that failure
at the source, which is why C3 is a regression check, not a hunt. When this is
ported to a script, the script replaces this prompt verbatim.
