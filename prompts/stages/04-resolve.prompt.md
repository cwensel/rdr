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
RDR was lock-audited and demoted by the 07.1 cluster gate for a named defect;
`"routed-back"` means a later stage sent it back mid-flow on the same `re-verify`
scope, and this stage clears that qualifier as its first act (rdr-common
§rdr-write *Receiving a route-back*). The
scope set is `edges[]` where `kind=="reverify"`: each `to` is one of this record's
own `NNNN:A*` ids. Re-verify ONLY those; carry the rest forward as already
Verified — do NOT re-derive them. Anchors the demotion's edit touched are read by
id in one call (`--select <NNNN>:I-4`), never by line windows. Read each named
assumption by id, not the whole Critical Assumptions section:

```sh
"$RDR_HOME/bin/rdr" inspect --select <NNNN>:A2 <NNNN>
```

Third branch — `revised-from` with an EMPTY `reverify` set and a refine commit
after the demote (a route-back refine that rewrote the list): scope = the
assumptions whose lines that commit touched. Never `git show` it; ask:

```sh
sha=$(git -C "$RDR_RECORDS" log -1 --format=%h --grep='^docs(rdr): refine' -- "$RDR_PATH")
"$RDR_HOME/bin/rdr" inspect --touched-since "$sha^" --json <NNNN>   # .touched[].id, A* only
```

Use the id list as a value; write it into the qualifier so the next stage
routes without recomputing.

A `resolved:false` target names an assumption that does not exist — report it
rather than skipping silently; `resolved` absent means nothing looked, neither
sound nor broken. `.status.form` anything else (a bare `Draft`) is the cold path:
verify every assumption from scratch as below.

On any scoped re-entry the reuse audit and the {RDR_ENV} / {RDR_RESOURCES} reads
above are owed only for behaviours the in-scope assumptions introduce; otherwise
the close packet says `reuse audit: n/a — scoped re-entry`. A landing-order
precondition on a peer ("NNNN lands first") is answered by
`"$RDR_HOME/bin/rdr" status --tags <peer>` — `status=Final` and
`gate_stale=false` — never by reading the peer's files or history.

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
ask: resolve by another Method first. Every question the round carries has
walked rdr-common §ground-before-ask; an `apply` answer is a recommendation,
not a question. And nothing the round will rule on is written into the record beforehand: a
pre-applied recommended edit is a rewrite waiting on a different answer. An approved fixture → record it as a
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
replace it from the now-verified contracts. Judge the one durable contract on
two dispositions, written once, in the clause: is its surface user-facing
(`yes|no`), and what does it lock (`none` | `contract` | `format` — an
enum/hash/format/grammar/destructive op | `cross-rdr` — a producer other RDRs
consume, or one spanning modules). Then run rdr-common §rdr-write with
`--outcome profile`, passing `--tag floor=<emit.floor of --outcome floor>`
(§lens-row's call — `raise` lifts the count's tier by one, resolved there,
never re-read here), `--tag user_facing=<yes|no>` and `--tag locks=<…>`,
and apply `edit` as handed: the field becomes `<value> — <one clause naming the
contract>; user-facing <yes|no>; locks <…>`; drop any matrix/provenance prose
the template or Seed left (it lives in the template comment, not the instance).
The row counts the durable (non-`Transient`) fenced contracts for you:
`stopped:split-signal` means two or more — the RDR spans more than one seam, so
flag a split (back to Stage 2/3) rather than pick a profile;
`stopped:contracts-unlabelled` means the contracts are prose — label each
`**Cn**` over its fence first. Report `emit.profile`, the durable count behind
it, and `emit.why` when the floor applied.

Be brief in results; ultrathink for complex design or any load-bearing
assumption; never trade brevity for a weaker verification. Report per
assumption: Status + Method + one-line Evidence, and flag any you could NOT
verify; plus one line: author's-round items put / approved / rejected (fixtures
and questions, `none` if the round had nothing to ask). Close with
rdr-common §mechanical-gate.
