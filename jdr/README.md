# Joint Decision Registries (JDRs)

A **JDR** is the single normative home for decisions that two or more sibling
RDRs *jointly* own — decisions no one RDR solely owns, which therefore cannot
live inside any one of them.

This is the engine's JDR document class: the convention, the grammar, and
[`TEMPLATE.md`](TEMPLATE.md). Instances live in the consumer repo at
`jdr/<project>/NNNN-<seam-slug>.md`, the way records live at
`rdr/<project>/`.

## Why this exists

The cluster gate (`/rdr-cluster-reconcile`) types a cross-RDR finding one of
three ways. The third, **JOINT-DECISION**, is explicitly *not* a spec defect:
the siblings contradict over a decision that is jointly theirs. The prescribed
cure is to **hoist** it to a single normative home that the siblings **cite**
rather than restate, and let them proceed under recorded tolerance.

Restating a shared decision in each sibling is what makes them drift. The
engine states the general law:

> Two copies of one contract drift into self-contradiction — the most common
> internal-defect class — so the cure is deletion, not a lint to keep them in
> sync.
> — [`stages/README.md`](../stages/README.md), *Single-source each contract*

The failure mode is concrete: two RDRs each carry a copy of one shared rule,
their re-locks land a day apart, and the copies contradict. Every gate passes,
because each document is internally consistent. An implementer follows the
wrong copy and rebuilds the defect with a green conscience.

## What a JDR is *not*

- **Not an RDR.** An RDR is one decision, with alternatives, assumptions,
  lenses, and a finalize gate; it locks. A JDR is a **registry of many entries**
  with independent per-entry lifecycle. Seeding an umbrella *RDR* to fix
  cross-RDR drift reproduces the drift one level up.
- **Not a decision log.** In ADR practice "decision log" already means the whole
  directory of records. A JDR is a curated subset with normative force.
- **Not an RFD.** An RFD is a capability: intent, journeys, principles, human
  voice. A registry of locked rows inside one is the abuse this class exists to
  end.

## Identity — a question over a seam

**A JDR is one question, asked over one seam, whose entries are the decisions
every record on that seam is bound by.** It is not a cluster of records and it
is not a ledger of records.

| Term | Definition |
| --- | --- |
| Seam | a declared set of loci: `path::Symbol` anchors and path prefixes. Named at seed. May widen, never narrow. |
| Question | the one thing the seam's records must agree on. Two questions on one seam are two registries. |
| Identity | seam ∧ question. Fixed at seed. Never merged or split. |
| Member | a record whose anchors touch the seam. **Derived**, never declared. Non-transitive. |
| Bound | a member an entry names. Every bound record is a member; not every member is bound. |
| `cluster` field | a recorded view, "members as of iteration n." Provenance, not identity. |

**Why the cluster cannot be the identity.** The engine clusters twice, and both
sets are transitive: Stage 2's overlap graph over shared anchors and literals,
and Stage 7.1's closure over predecessors, peer evidence and shared concerns.
Transitivity merges — A shares with B, B with C, so A and C land in one
component with nothing to say to each other. Seam membership is a property of
each record *against the seam*: A touches S; B touches S and T; C touches T.
A and C share no registry. B is in two. Nothing merges.

This is also why a JDR is **append-only and never merges or splits after
seeding**. Once a record states `→ JDR cli/0001 §JD-5`, that anchor must never
move. If a later decision genuinely spans two registries it lands in the one
owning the seam it sits on, and the other cross-references it. Two registries
cross-referencing is cheap; a moved anchor is not.

**Compound titles are a smell worth checking.** If the topic question needs an
"and" to state, test whether the halves are causally chained (one registry) or
merely adjacent (two).

### Membership is derived, and the projector computes it

`recs` publishes the facts; a model routes on them. The intrastate guard
vocabulary has no set intersection, so the intersection is the projector's job
and the decision is the table's.

The facts are **per record**, in the shape the rollup facts already use
(`rdr-facts.toml`'s predecessor and cluster rollups read many peers and answer
once). The registry is not an operand: asking "is this record on registry X"
requires the caller to have already chosen X, and choosing X *is* the fire
routing this table decides. A per-record rollup keeps the decision on the
table's side.

| Fact | Meaning |
| --- | --- |
| `jdr_registries: set` | the registries this record's anchors or **Seam Lineage** locus touch — **0, 1 or more is the routing dimension** |
| `jdr_member: bool` | the set is non-empty |
| `jdr_has_citation: bool` | the record names at least one entry |
| `jdr_has_binding: bool` | at least one entry names the record |
| `jdr_cited: set` | the registry entries its `jdr` / `joint-decision-home` edges name |
| `jdr_bound_by: set` | the entries whose **Binds:** line names this record |
| `jdr_cite_only: bool` | every seam hit sits under a section other than Implementation Plan |

All of them go **absent** when no JDR root is bound — a consumer that has not
adopted the class has not written a non-member record, it has written one
nothing has looked at. `recs index --jdr-members <registry>` asks the same
question from the registry's side, and stops rather than answering an empty
set when no root is bound.

`jdr_registries` is a set rather than a count because §Fire routing's three
arms read off it directly: none seeds, one routes, **more than one stops and
asks**. A count would answer the same question; the set also names them, which
is what the stop has to print.

The table guards on the bools and carries the sets: a `set` is guarded only
by `contains`, a superset test with no negated form that costs 2^|elements|
cells, so the projector publishes a companion bool for each and the sets ride
along so a finding can name the entries.

**What is deliberately NOT a new fact**, because the engine already answers it:

| Question | Already answered by |
| --- | --- |
| is a joint decision still open against this record | `open_joint_decisions`, read off the Status field |
| which records cite this entry | `index --backlinks=jdr:<project>/NNNN:§<entry>` — `BuildGraph` keys backlinks by target verbatim, so a registry entry is a backlink key for free |
| do two records share an anchor | `index --anchor-intersect`, whose path rule `MatchesLocus` shares |
| which entries does a registry bind | the registry's own **Binds:** lines, read by the projector |

A second mechanism for any of these would be a copy that drifts, which is the
defect this whole document class exists to prevent.

A seam locus is a full `path::Symbol` or a path prefix, matched by
`scan.MatchesLocus`, which shares its path rule with the overlap graph's
`sameAnchor` so seam membership and the overlap graph cannot drift apart. The
path halves match when one is a component-aligned suffix of the other (authors
write both `corpus.go::F` and `internal/cli/corpus.go::F`), and a
receiver-qualified anchor resolves to its member.

A **prefix** locus matches at any component boundary, not just the path head:
one repo's `internal/cli/x.go` is another record's `a/b/internal/cli/x.go`, and
a head-anchored match would drop the second out of the seam silently — a
narrowing nobody declared, when a seam may only widen. The bound on that
tolerance is ambiguity: a **single-component** locus matches at the head only,
because `cli` alone would claim every `cli` directory in the tree, and an
ambiguous reference stays unresolved rather than guessed at.

**`area:*` is not a locus the projector understands** — a registry spells its
area as the listed loci it stands for, and the tests run over that list.

Three lint codes fall out, and [`models/jdr-membership.toml`](../models/jdr-membership.toml)
decides them:

- `jdr:cited-off-seam` — cited ∧ ¬member. Either the seam widens or the
  citation is wrong.
- `jdr:bound-off-seam` — bound ∧ ¬member. An entry binds a record outside the
  seam.
- `jdr:cite-only-member` — advisory. The anchors that hit the seam are all
  under sections other than Implementation Plan: the record cites the seam for
  contrast rather than modifying it. Deterministic, because an edge carries its
  line and the outline says which section owns that line.

### Routing a fire

When a `Joint-check` fires on records `{A, B}` over shared anchors `X`:

1. **Exactly one registry** has `X ∩ seam ≠ ∅` → route to it. Confirm the
   question matches; if it does not, seed a second registry on the same seam.
2. **None** → propose seeding one, seam named from `X`.
3. **More than one** → **stop and ask.** No longest-prefix tiebreak: a tiebreak
   is a guess, and the projector's rule is that an ambiguous reference stays
   unresolved.

A fire graded a **fork** seeds or appends. An overlap alone seeds nothing.

### Naming

`jdr/<project>/NNNN-<seam-slug>.md`, zero-padded, sequential. The slug names the
**seam**, never the member list — membership grows, filenames must not rot.

Must live **outside** `$RDR_RECORDS`: `§rdr-resolve` globs
`$RDR_RECORDS/NNNN-*.md` non-recursively, so a numbered file placed there would
be silently resolvable as an RDR.

## Citation

Records cite `JDR <project>/NNNN §<ID>` — anchor only, never restated mechanism
prose. The cluster gate treats a restatement as the defect. Prose anchors follow
the same form: `JDR cli/0001 §Principles`.

A record tolerated under a joint decision carries the
[`TEMPLATE.md`](../TEMPLATE.md) qualifier on its live Status value:

```
**Status**: Final [joint decision → JDR cli/0001 §JD-5]
```

It stays `Final` for every binary gate and **does not self-clear** — the home
owns the decision, so the qualifier is permanent until the home says otherwise.

**Stamp the qualifier when the entry resolves, not when it is filed.** The
qualifier asserts a sibling may proceed under tolerance; asserting that over an
entry still `open` would license implementation across an undecided contract.
The engine enforces the matching rule from the other side: `index --cycles`
treats an `open` entry as it treats a Draft home, so a Final homed on one is
reported as a home ahead of its lock.

A registry that took over anchors from an earlier home declares them in
`inherits`, so a legacy spelling still resolves. **An alias hit is not a
finding.** The edge carries the registry that answered it, so the citations
the migration has not reached stay countable.

**Text taken over is not the registry's to restyle.** A citation inside hoisted
text keeps the spelling its old home wrote: the migration rules that rewrite an
author's own citations skip it, for the same reason they skip a quotation. Two
declarations mark it, and a registry may use either — `inherits:` names the
anchors, so each named entry's block is covered; and a heading that opens
`Hoisted from …` covers its whole section, which is what carries entries
hoisted as bullets or ids that continue past the declared range.

The alias is **scoped and exclusive**. `inherits: RFD 0004 DX-1..DX-18` answers
a citation of RFD 0004's DX-13 and no other document's — an id is only an alias
for the home it was actually taken from. And exactly one registry may claim an
anchor: two claimants make the citation genuinely ambiguous, so it stays
unresolved rather than resolving to a guess.

## Entry lifecycle

Per entry, not per document.
[`models/jdr-entry.toml`](../models/jdr-entry.toml) is the state machine; lint
proves the transition table covers the state space.

| Status | Meaning |
| --- | --- |
| `open` | Identified, not yet decided. Blocks the records it binds. |
| `constraint` | Shipped code or a stated principle already determines it. Record it so nobody implements against it. Does not block. |
| `blank` | A detail the implementer fills (a code string, a test's home). Names an owner, not a negotiation. Does not block. |
| `deferred` | Cannot honestly be settled on paper; names the phase that settles it. Does not block seeding that phase; does block claiming the seam is closed. |
| `decided` | Resolved here. A **Resolved:** line names the landing records. Bound records may carry the tolerance qualifier. |
| `withdrawn` | Re-triaged as a single-record defect or a non-finding; the reason is kept so the anchor never dangles. |

**Never delete an entry — withdraw it.** Deleting breaks citations. A target
that no longer exists at all is kept alive by a `withdrawn` entry.

### Grade entries, or the registry over-gates

The cluster gate grades findings by severity (`blocks-impl` / `risks-impl` /
`cosmetic`). That axis asks *must an implementer confront this?* — true of a
thousand micro-decisions, so `blocks-impl` inflates. A registry that copies
severity straight into `open` will hold a cluster at NOT RECONCILED over
bookkeeping.

Grade every entry on a second axis before filing it:

- **Fork** → `open`. A genuine either/or with divergent consequences; wrong
  choice costs rework or ships a defect. **These gate implementation.**
- **Constraint** → `constraint`. Shipped code or a stated principle already
  determines it. Write it down; do not negotiate it.
- **Blank** → `blank`. The implementer fills it. Name the owner.

Check candidates against **shipped code, not sibling prose** — a gap between two
records often closes the moment you read the package they both describe.

The inverse error is real too: grade by *consequence*, never by how hard the
finding was to spot or how cheap the fix looks. An entry that is one line to fix
and silently unsound if skipped is a fork.

## Document state

`open` — entries still being decided · `settled` — no entry `open` ·
`superseded` — replaced (name the successor). Derived, not asserted.

## Structure

[`TEMPLATE.md`](TEMPLATE.md) is the file. In outline:

1. YAML frontmatter — `authors`, `state`, `rfd`, `seam`, `cluster`, `labels`,
   `inherits`
2. `# JDR <project>/NNNN <Title>` — a **question**, not a noun phrase
3. `## Problem statement` — what the members jointly decide
4. `## Principles` — **required.** See below.
5. `## D1…Dn` — the genuine forks, as `(a)/(b)/(c)` with trade-offs and a
   recommendation; each closes with a bolded **Resolved:** naming which records
   the decision lands in and what changes there
6. `## Interface record` — the `JD-n` entries, each stating its answer, the
   user-visible stake, and a short provenance parenthetical
7. `## What this does not decide` — local items that stay with their records

**Entry ids are per registry, and a registry that inherits decisions keeps the
ids it inherited**, so a `DX-13` hoisted out of an RFD stays `DX-13`.

**No work plan.** A JDR records decisions, not the sequence for executing them.
A "next steps" section is stale the moment the first record re-locks, nothing
updates it, and it duplicates what the Status lines and `/rdr-status` already
say authoritatively — the same two-copies-drift failure this document type
exists to prevent, turned on the document itself. Each **Resolved:** names its
landing records; that is the durable half, and it stays true after the work is
done.

**Keep instances short.** Rationale and decisions only. Convention lives here
and is never repeated in an instance; process history lives in git and is never
narrated in the document.

### Guiding principles — what makes a JDR steer instead of merely record

A registry of independently-argued entries drifts exactly like the sibling
records it replaced: ten locally-reasonable resolutions that do not add up to
one coherent behavior. The cure is a small set of standing commitments, stated
before the entries, that every resolution must satisfy.

- **Derive, never invent.** Each principle must be traceable to normative text
  the members already lock, or to a decision resolved in this registry. Quote
  the source. A principle with no anchor is the author legislating.
- **Make them decisive.** A principle earns its place by ruling options *out*.
  If every option on a fork satisfies it, it is decoration.
- **One line each.** A principle that needs a paragraph is an argument, and
  arguments belong in the decision that uses it.
- **A resolved fork may become a principle.** Resolutions compound rather than
  accumulating: once a fork fixes a promise at a strength, later entries bound
  themselves by it and cite it rather than re-arguing.

This is the section that answers "is a JDR a ledger?" — no. A ledger records; a
JDR records *and* constrains how its open entries may be closed.

## Ancestry

No canonical name exists for this artifact; it combines two long-standing
traditions.

- **Interface Control Documents** and their joint-ownership working groups
  (MIL-STD-490A; NASA SP-2016-6105) — a boundary jointly owned by parties whose
  own specs must not restate it.
- **IANA registries** (RFC 8126) — numbered entries, per-entry lifecycle, and
  specifications that cite entries rather than restating them.
- **arc42 §8 Cross-cutting Concepts** — rules stated once so sibling sections
  need not repeat them.
- **ViewPoints inter-viewpoint rules** (Finkelstein/Nuseibeh 1994; Easterbrook
  1996) and **boundary objects** (Star & Griesemer 1989) — the stance that
  cross-view inconsistency is *managed* at chosen checkpoints rather than
  eliminated. Stage 7.1 already rests on this literature.

Rejected names: `ICD`/`IRD` (aerospace-only recognition; `.icd` is an OpenCL
driver config), `ADR` (one-per-file, and collides with RDR), `SDR` (System
Design Review), `decision log` (inverts the established ADR meaning).
