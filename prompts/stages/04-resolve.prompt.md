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

**Per-assumption search budget — paste this line into EVERY Source Search brief,
whatever the method (corpus search, own-code `rg`/code search, spike):** ≤4
queries per assumption, ≤6 opened hits, stop on the first hit that verifies or
falsifies — sufficiency, not exhaustiveness, is the bar; a retry of a wrong flag
or regex counts against it. For a corpus the call is `arc search semantic
--corpus <C> --limit N --json "<q>"` (flag is `--corpus`). The sub-agent
returns query strings + accepted evidence pointers in its §return-packet
(rdr-common) — do not define a competing shape — and a `negative: <assumption> —
no corpus evidence in <corpora>` line when nothing lands. Persist accepted
citations + rejected branches to {EVIDENCE_DIR}research/; a SCOPED RE-ENTRY or
Draft re-run reuses that file and re-searches ONLY assumptions not already cited
there.

SCOPED RE-ENTRY. Ask the projector, don't parse the Status line:

```sh
"$RDR_HOME/bin/rdr" inspect --json --filter metadata,edges <NNNN>
```

`metadata[]` where `label=="Status"` → `.status.form == "revised-from"` means this
RDR was lock-audited and demoted by the 07.1 cluster gate for a named defect. The
scope set is `edges[]` where `kind=="reverify"`: each `to` is one of this record's
own `NNNN:A*` ids. Re-verify ONLY those (plus any whose Evidence anchor the
demotion's edit touched — a diff signal, not a projection); carry the rest forward
as already Verified — do NOT re-derive them. Read each named assumption by id, not
the whole Critical Assumptions section:

```sh
"$RDR_HOME/bin/rdr" inspect --select <NNNN>:A2 <NNNN>
```

A `resolved:false` target names an assumption that does not exist — report it
rather than skipping silently; `resolved` absent means nothing looked, neither
sound nor broken. `.status.form` anything else (a bare `Draft`) is the cold path:
verify every assumption from scratch as below.

For each Critical Assumption in scope:
- Pick exactly one Method: Source Search | Spike | Prior Art | Derivation |
  Design Decision | Peer RDR | MVV Test | Docs Only.
- If the claim can be shown as a concrete value — an exactness word (the sweep
  below) **in the assumption or the normative clause it backs** is the prime
  cue; the same word loose in prose is not, or the cue fires on nearly every
  RDR — render 1–3 **fixtures**: the exact expected value itself (a wire record,
  byte layout, count, field set), read from a spike run or the source, never
  invented. Hold them for the round below. Most assumptions render none; that is
  the expected case, not a gap to fill.
- Produce concrete Evidence for it:
    Source Search → file:line in the actual source that owns the behavior:
      for a claim about THIS project's own code, the project source (the
      {RDR_ENV} reuse-audit paths are a fine starting point); for a claim about
      EXTERNAL behavior (a dependency, peer tool, or standard), the
      {RDR_RESOURCES} corpora. NOT this RDR or its artifact dir — citing the
      spec to verify the spec is self-reference, not verification. (The reuse
      audit above asks a different question of the same own-code source: does a
      capability already EXIST? Source Search asks: does the code BEHAVE as the
      claim says?) A claim about call ORDER, REACHABILITY of a site, or
      emission side-effects is never attested by opening the symbol it names —
      an unconditional consumer reads true while the conditional producer
      upstream decides. Method is Spike or MVV Test, with the ordering,
      reachability, or output diff itself as Evidence — Evidence answering the
      claim's own predicate, not a proxy question. A normative clause
      asserting one of these carries its own assumption, so a sweep has
      something to falsify.
    Spike → the command run against a live service/fixture + where output is
      captured under {SPIKE_DIR}. Actually run it; paste the output.
    Prior Art → named external system + section/page.
    Derivation → the math, inline.
    Peer RDR → the RDR id + section that owns the property.
    Docs Only → INSUFFICIENT for load-bearing claims; allowed only paired
      with a Spike or Source Search plan stated in the Evidence line.
- Set Status: Verified only when Method + Evidence actually support it.
  Otherwise leave Pending with a named verification plan — unless the MVV
  depends on it (the MVV, or a normative fixture the MVV consumes, rests on
  this assumption): an MVV-critical assumption resolves NOW or the stage is
  NOT READY.
- Confirm "If wrong" is non-empty and names how it surfaces to a user/test.

Then ONE consolidated **author's round** — Stage 4's single user interaction,
one round like Stage 6's accept/defer tiebreakers, never a drip per assumption:
present every rendered fixture for approve/reject — plus any other question only
the author can settle, so the round is the stage's one interaction whether or
not a fixture exists. **Everything it surfaces carries the
grounding it takes to rule on it** — the source anchor, spike output, or prior
decision (peer RDR, RFD clause, standing contract) that constrains the answer.
You hold these from the Method work above; withheld, the author re-derives what
you just verified or answers blind. Can't name the grounding → not ready to
ask: resolve by another Method first. An approved fixture → record it as a
normative fixture in the RDR body: named on the assumption's Evidence line
and, where a Testing Strategy scenario covers it, in that scenario's Expected
— citing the spike artifact under {SPIKE_DIR} that produced it (TEMPLATE.md's
Normative/Illustrative rule gives it its lock-time semantics). Rejected → the
assumption is genuinely open: resolve by another Method or revise the claim.
A revised or narrowed clause runs rdr-common §amendment-sweep.

Verify EXACTNESS words too — each needs an Evidence Record (prefer a named
normative fixture from the round above) or coverage by the Minimum Viable
Validation. `rdr lint`'s `prose:exactness` names the terms of art inside the
normative fences; the QUANTIFIERS (all/every, first/nearest) are yours to read,
because only you can tell "all rows" the commitment from "all three" the
sentence. For byte-stable output, run the determinism checklist (hash fn+lib,
pre-image byte layout, encodings, map order, whitespace, case folding,
empty/null/absent, version marker).

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
Exclude `Transient`-marked contracts (scheduled deletion by a named sibling —
TEMPLATE.md Normative Contracts) from this recount and from the ≥2 split
signal: the marker is a recorded lifespan disposition; sizing stays on the
durable contracts.
With the assumptions just verified, the count is evidence-grounded here, not
guessed. ≥2 independent contracts → the RDR spans more than one seam: flag for
splitting (back to Stage 2/3) rather than picking a profile.
**Then apply the accretion floor — it outranks the count.** Read `Seam Lineage`
(do not re-derive): ≥2 closed prior point-fixes → write `foundational`
regardless of what you just counted, escapable only by the written accretion
disposition already in that field. The count may raise the profile; it may
never lower one the floor holds — un-flooring an accreting seam routes it past
the very lenses the floor buys. Report the profile + the contract count behind
it, and the floor's disposition when it applied. The field holds the value + one
clause naming the contract(s); strip any matrix/provenance prose the template or Seed left
behind — that guidance lives in the template comment and `rdr/stages/README.md`,
not the instance.

Be brief in results; ultrathink for complex design or any load-bearing
assumption; never trade brevity for a weaker verification. Report per
assumption: Status + Method + one-line Evidence, and flag any you could NOT
verify; plus one line: author's-round items put / approved / rejected (fixtures
and questions, `none` if the round had nothing to ask). Close with
rdr-common §mechanical-gate.
