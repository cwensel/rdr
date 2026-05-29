# Tooling Pass — RDR Mechanical Adherence Sweep

**Use when**: the **mechanical pre-step of the Finalization Gate**
([Stage 8](../../flow/08.0-finalize.md)), run on **every** RDR just before the
Gate's written responses — after whatever pre-lock rounds the RDR's profile
called for (a small RDR runs none) and after the spike/assumption reconcile. It
is a *post-mutation regression sweep*: the lenses and fix-passes rewrite the
draft and the reconcile flips assumptions, any of which can hollow a section or
disturb an evidence record an earlier stage had clean. For a small RDR that ran
no lens it is the *only* mechanical check; for any RDR it is the conformance
backstop when Refine/Resolve was skimped.

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
Hollow = TBD, "see above," single-sentence placeholder, or copy-paste of the
template instructional text. List every Present-hollow and Missing entry by
section name. (This is where post-rewrite regressions show up most.)

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

Output: one bullet per finding, prefixed with the check ID (C1/C2/C3/C4) and
the section or assumption ID. End with a one-line verdict:
  PASS — no findings; proceed to the Gate's written responses.
  BLOCK — N findings, lock prohibited until resolved.
```

## Expected signal

- **PASS** — every TEMPLATE section is Present-substantive, every Method is in
  the vocabulary, no `Source Search` self-references, no load-bearing
  `Docs Only`. Proceed to the Finalization Gate's written responses.
- **BLOCK** — at least one finding. Most findings are C1 (a section the rounds
  hollowed) or C2/C4 (an assumption a fix-pass disturbed). Fix the RDR and
  re-run. A C3 finding is rare and signals a disturbed evidence record — treat
  it seriously rather than as routine.

## Out of scope for this pass

The README rule that *every external API call inside a Normative block must
have a corresponding Critical Assumption Evidence Record above* is
intentionally not mechanized here — it requires linking call-sites to records,
an analytical judgment. Verify it during 3amigo (Implementer persona) or the
Repeatability lens. A future CHECK 5 can absorb it once the
matching heuristic is reliable.

## Source

The four checks are the mechanical share of the implementation-prompt review
lens, isolated from the analytical share so a script can absorb them. The sweep
runs last, immediately before the Gate. CHECK 3 (Source-Search self-reference)
originates in an early audit that found nearly half of all load-bearing
assumptions self-referencing across RDRs authored *before* the structured
Evidence Record and the Resolve stage existed; both now prevent that failure
at the source, which is why C3 is a regression check, not a hunt. When this is
ported to a script, the script replaces this prompt verbatim.
