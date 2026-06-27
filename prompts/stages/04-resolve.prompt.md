For the RDR at {RDR_PATH}, resolve all critical assumptions.

Read {RDR_RESOURCES} for the corpora, design docs, and anchors; read {RDR_ENV} for the
spike-output location ({SPIKE_DIR}) and the reuse-audit source paths. Two source
domains: {RDR_ENV}'s paths point at THIS project's own code; {RDR_RESOURCES}'s
corpora point at EXTERNAL source (dependency/peer/standard). A claim is verified
against whichever domain owns the behavior — own-code claims against project
source, external-behavior claims against the corpora. Use the corpora to
research against the literature, prior art, competition, and standards; confirm
all references and citations from that local research, not from memory.

FIRST, the reuse audit: for each behavior the approach introduces, check the
{RDR_ENV} reuse-audit paths for code that already does it. If the codebase already
provides it, that is a finding — fold reuse into the design or return to Stage
2/3 rather than verifying an assumption about building anew.

This stage is read-heavy: for each corpus search, reuse audit, or dependency
spelunk, delegate to a sub-agent that returns the verdict + evidence pointer
(file:line, or spike command + output path) as a §return-packet (rdr-common) — NOT raw hits or whole files.
Hold only the verdicts here.

**Per-assumption search budget (in the sub-agent brief).** Each delegated search
is bounded: ≤4 corpus queries per assumption, ≤6 opened hits; stop on the first
hit that verifies or falsifies — sufficiency, not exhaustiveness, is the bar.
Invoke `arc search semantic --corpus <C> --limit N --json "<q>"` (flag is
`--corpus`; an obsolete-flag retry counts against the budget). The sub-agent
returns query strings + accepted evidence pointers in its §return-packet
(rdr-common) — do not define a competing shape — and a `negative: <assumption> —
no corpus evidence in <corpora>` line when nothing lands. Persist accepted
citations + rejected branches to {EVIDENCE_DIR}research/; a SCOPED RE-ENTRY or
Draft re-run reuses that file and re-searches ONLY assumptions not already cited
there.

SCOPED RE-ENTRY: if the Status reads `Draft [revised from Final …; re-verify
<IDs>]`, this RDR was lock-audited and demoted by the 07.1 cluster gate for a
named defect. Re-verify ONLY the listed assumption IDs (plus any whose Evidence
anchor the demotion's edit touched); carry the rest forward as already Verified
— do NOT re-derive them. A bare `Draft` (no qualifier) is the cold path: verify
every assumption from scratch as below.

For each Critical Assumption in scope:
- Pick exactly one Method: Source Search | Spike | Prior Art | Derivation |
  Design Decision | Peer RDR | MVV Test | Docs Only.
- Produce concrete Evidence for it:
    Source Search → file:line in the actual source that owns the behavior:
      for a claim about THIS project's own code, the project source (the
      {RDR_ENV} reuse-audit paths are a fine starting point); for a claim about
      EXTERNAL behavior (a dependency, peer tool, or standard), the
      {RDR_RESOURCES} corpora. NOT this RDR or its artifact dir — citing the
      spec to verify the spec is self-reference, not verification. (The reuse
      audit above asks a different question of the same own-code source: does a
      capability already EXIST? Source Search asks: does the code BEHAVE as the
      claim says?)
    Spike → the command run against a live service/fixture + where output is
      captured under {SPIKE_DIR}. Actually run it; paste the output.
    Prior Art → named external system + section/page.
    Derivation → the math, inline.
    Peer RDR → the RDR id + section that owns the property.
    Docs Only → INSUFFICIENT for load-bearing claims; allowed only paired
      with a Spike or Source Search plan stated in the Evidence line.
- Set Status: Verified only when Method + Evidence actually support it.
  Otherwise leave Pending with a named verification plan.
- Confirm "If wrong" is non-empty and names how it surfaces to a user/test.

Verify EXACTNESS words too (all/every, first/nearest, byte-identical,
lossless, canonical, deterministic, stable order) — each needs an Evidence
Record or coverage by the Minimum Viable Validation. For byte-stable output,
run the determinism checklist (hash fn+lib, pre-image byte layout, encodings,
map order, whitespace, case folding, empty/null/absent, version marker).

Once the assumptions hold, author the **evidence-body** from them (replace its
`_Draft placeholder._`s):
- **Testing Strategy**: the test matrix the verified assumptions imply — the
  code paths reviewed + spike outputs that back each, named.
- **Performance Expectations**: what the evidence (spikes, derivation) shows;
  the determinism-checklist results where byte-stability is claimed.
Propose owns Investigation + Implementation Plan; do not re-author those here.

Then OVERWRITE THE **Profile** Metadata field — the routing latch every later
stage reads instead of re-deriving size. Seed wrote a provisional estimate; you
replace it from the now-verified contracts (don't trust the estimate — recount).
Count the *independent* load-bearing contracts in the Normative Contracts
section (a distinct type design, hash, wire format, taxonomy, or destructive-op
policy each count as one) and write the matching value (`rdr/stages/README.md`
matrix):
- one contract, no user-facing surface → `small` (skips Stage 5 — next is
  Reconcile, not Pre-Lock);
- one contract + user-facing surface, OR locks a contract → `mid`;
- locks an enum/hash/format/grammar/destructive op → `large`;
- cross-RDR producer / spans modules → `foundational`.
With the assumptions just verified, the count is evidence-grounded here, not
guessed. ≥2 independent contracts → the RDR spans more than one seam: flag for
splitting (back to Stage 2/3) rather than picking a profile. Report the profile
+ the contract count behind it. The field holds the value + one clause naming
the contract(s); strip any matrix/provenance prose the template or Seed left
behind — that guidance lives in the template comment and `rdr/stages/README.md`,
not the instance.

Be brief in results; ultrathink for complex design or any load-bearing
assumption; never trade brevity for a weaker verification. Report per
assumption: Status + Method + one-line Evidence, and flag any you could NOT
verify.
