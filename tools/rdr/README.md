# `rdr` — a read-only projector for RDR records

`rdr` reads RDR markdown and projects it as data. It never writes to a
record: **markdown remains the source of truth**, and everything here is a
view of it. If the model and a record disagree, the record is right and the
model has drift to fix.

`inspect` is live: it projects one record as an outline, a set of
addressable elements, the classified fields of its Metadata block and of
every element, the typed edges it states, and a warnings channel, and
resolves any element ID back to the lines it names. `index --derived`
reports the corpus's labelling backlog; `index --coverage` is the drift
alarm; `index --unresolved`, `--backlinks` and `--cluster-of` query the
edge graph. `lint` is the conformance authority (§Lint).

Flags precede the positional argument — Go's flag parser stops at the
first non-flag word.

    rdr inspect 55                         # one line per element, ids first
    rdr inspect --json 0055                # the envelope: outline, elements, anchors, metadata, fields, edges, warnings, coverage, counts
    rdr inspect --json --filter metadata,counts 0055   # only those keys — one call, ~4% of the envelope
    rdr inspect --select 0055:C4 0055      # the bytes the id names
    rdr inspect --select edges 0055        # the typed relations alone
    rdr index --derived --records ../rdr/cli
    rdr index --coverage --records ../rdr/cli
    rdr index --unresolved --records ../rdr/cli --repo ../src
    rdr index --cluster-of 0130 --records ../rdr/cli

Go stdlib only — no third-party dependencies, by design. RDR markdown is
line-oriented (headings, fences, bullet trees with bold labels), so a line
scanner is both smaller and more maintainable than a generic AST adapter,
and a static binary gives `rdr-doctor` one thing to check for.

    go build ./...
    go vet ./...
    go test ./...

## The same-commit rule

**A `TEMPLATE.md` change that adds, removes, renames or re-levels a
section, or changes a class, field, label or qualifier grammar, ships in
the same commit as its model update and a synthetic fixture.**

This is the rule the whole package exists to enforce. `TEMPLATE.md` is the
only schema an RDR has and it is prose, so every consumer that needed its
section names, classes, vocabularies and qualifier grammars restated them
by hand — and they drifted. Four template generations became tribal
knowledge because no rule like this existed.

It is enforced mechanically, not by good intentions:

| test | asserts |
| --- | --- |
| `TestEpochDMatchesTemplateFile` | every section name, level, class and position in `EpochDTable` matches `TEMPLATE.md`, read from the file at test time |
| `TestStatusVocabularyMatchesTemplate` | the canonical Status set matches `TEMPLATE.md`'s Status line |
| `TestTypeVocabularyMatchesTemplate` | the canonical Type set matches its Type line |
| `TestMethodVocabularyMatchesREADME` | the eight Method labels match `README.md`'s *Verifying load-bearing claims*, which declares itself authoritative |
| `TestFixturesDetectTheirEpoch` | each synthetic fixture still fingerprints as its epoch |
| `TestDecisionClassesMatchTemplate` | the D-key classes match the Load-Bearing Decisions bullets |
| `TestGateItemsMatchTemplate` | the G-keys match the Finalization Gate sub-sections |
| `TestSectionFieldsMatchTemplate` | each section's canonical `- **Label**:` set matches the bullets `TEMPLATE.md` writes under it, at any indent |
| `TestAssumptionStatusVocabularyMatchesTemplate` | the Evidence Record's Status set matches its `Verified \| Pending \| Unverified` line |

A divergence names itself and says what to update. Gaining, losing,
renaming or re-levelling a section, or flipping its class, all fail — each
was verified by mutating `TEMPLATE.md` and watching the test fail before
the file was restored.

## Identifiers

Every element of a record has one ID, in one grammar:

    [<project>/]<NNNN>:<KIND><key>

| id | element | key is |
| --- | --- | --- |
| `0055:A3` | assumption | the `A3` label, as written (`A4b` / `A1.b` → `A4b`, `A1b`) |
| `0055:C4` | normative contract (a ```` ```normative ```` block) | a `**C4**` / `##### C4` label on the line above the fence; else the block's document ordinal |
| `0055:D-identity` | load-bearing decision | the template's decision class (`DecisionClasses`); else the author's own number (`D-6`); else the label's slug |
| `0055:RT1` | round-trip invariant | an `RT1` / `INV-1` lead, or a unique list number; else ordinal |
| `0055:ALT2` | alternative | the `Alternative 2` scaffold ordinal |
| `0055:BR3` | briefly-rejected item | a unique list number; else ordinal |
| `0055:S5` | validation scenario | a unique list number; else ordinal |
| `0055:MVV` | minimum viable validation | none — one per record |
| `0055:F2` | failure mode | a unique list number; else ordinal |
| `0055:G-scope` | inlined gate response (epochs A, B) | the gate item (`GateItems`); else the heading's slug. The namespace is CLOSED to those five keys — a citation of a record's own `G-a` guard table names the document, as `REQ-N` does |
| `0055:§approach` | outline section | the canonical section's slug; scaffolds and legacy aliases slug their own heading |
| `0055:§the-values` | bold paragraph lead | the lead's own slug — addressable text, not structure; matched exactly, never by prefix |
| `cli/0055:C4` | any of the above, across records dirs | `--project` supplies the prefix; omitted inside one dir |

An ID is **as written** when the author labelled the element and
**derived** when the projector minted it from the element's ordinal.
Derived is metadata, not a defect: on a terminal record an ordinal is as
stable as a label because the file never changes again, and every
legacy record is fully addressable at zero edit cost. On a live record a
derived ID holds until a sibling is inserted before it — which is what
labels are for, and what `index --derived` counts down.

Two rules keep IDs unique when a live record is part-way through
labelling: a label always wins its number (first writer; a repeated
label is a `<kind>:duplicate` warning and the later element yields), and
a derived ordinal a label has already claimed takes the first free
number above the element count, with a `<kind>:collision` warning.
Author numbering is never overridden.

Every element and section carries `line_start` / `line_end` (1-based,
inclusive) so an ID round-trips to bytes — `--select 0055:C4` prints
them — and a `hash` (`ident.Hash`: SHA-256 over the trimmed non-blank
lines, first 8 hex digits) that separates *same ID, same content* from
*same ID, changed*. Reflowing or re-indenting an element leaves its hash
alone; editing a word does not.

Ownership transfer is by ID, never by renumbering: when responsibility
moves between records, the element keeps its ID and the edge extraction
(a separate change) records `overrides` / `moved-to` from the old home.

### The citation form, and what authors owe

Reading IDs is half of it; the other half is an authoring rule, because a
reference is only resolvable if the target was given an identity and the
citation reaches for it.

Contracts carry `**C1**`, `**C2**` … in document order, on the line above
the fence (TEMPLATE.md §Normative Contracts). The number is the contract's
name for life: never reused, never renumbered — a deleted `C2` leaves a
gap, because somewhere a peer cites it.

A cross-record reference to a load-bearing element — an assumption, a
contract, a scenario, a decision — is written as an ID: `0055:C4`,
`cli/0055:A3`. The grammar reads the colon form, the spaced form
(`cli/0055 A5`) and `§Section Name` alike, so older citations keep
resolving; the colon form is what new ones use. A filename or heading-text
reference is a `mentions` edge: fine for context, wrong for a claim, since
it is exactly what a reword or a move breaks.

`Method: Peer RDR` Evidence names an element, never a bare record. A
citation that resolves to a whole 4,000-line record has named a document,
not a reason — `rdr lint` reports it as `peer-evidence:no-element`.

**None of this is retroactive.** Terminal records are never amended, and
an ordinal-derived ID on a frozen file is exactly as stable as a written
label, so the legacy corpus is fully citable at zero edit cost. Live
records get labels the fix-forward way: `rdr lint` advises, and the stage
already rewriting the file labels them in-pass.

## Lint

`rdr lint [<NNNN>] [--locking]` is the conformance authority: one pass,
three severities, and a rule about which records each may speak about.
With no argument it lints the whole records dir. It exits 0 on PASS —
findings or not — and 1 when a finding blocks a lock.

| tier | scope | blocks? |
| --- | --- | --- |
| `parse` | every record | never |
| `conformance` | LIVE records only | never |
| `resolution` | every record | at a lock gate, on a live record |

**parse** republishes the scanner's warnings channel. On a terminal record
a parse warning is a projector bug — the file cannot have changed, so the
scanner is what is wrong — and the fix is a fixture.

**conformance** is migration advice for a record that is going to be
rewritten anyway: unlabelled contracts, a Required section the current
template carries and this record's epoch predates. It is phrased for the
stage already holding the file open, and it never blocks. Terminal records
never generate it, because advice no one is permitted to act on is noise.
Two exclusions keep it honest: a subsection whose parent is absent is not
separately missing, and a gate subsection under a `gate.md` pointer is not
missing at all — from epoch C on, lock moves those responses out of the
record on purpose.

**resolution** judges what a record EMITS: every typed edge resolves, every
`Method: Peer RDR` Evidence names an element, and contracts are labelled on
a record written after the rule landed. This is the tier that applies to
terminal records too, because it is not about their shape. The DELIVERY
differs — on a frozen record the finding is a fix pointer carrying the line
range to open, and it does not block, because that record is not the one
locking. Correcting the reference text in that range is the one sanctioned
amendment to a locked RDR: the pointer only, never prose or structure.

The label rule's boundary is the record's own `Date`, not an epoch
fingerprint. A fingerprint would beg the question — the signal placing a
record in a "labels its contracts" epoch is the presence of labelled
contracts, so a new record that labelled nothing would fingerprint as
legacy and escape the check the rule exists to apply. `Date` is written by
Seed on every record and says when it entered the flow, which is what the
grandfathering rule actually asks.

The stability properties are tests, not intentions:

| test | asserts |
| --- | --- |
| `TestStabilityUnderProseEdits` | editing prose elsewhere changes no element ID or hash |
| `TestStabilityUnderReorder` | swapping two unrelated sections changes no ID or hash |
| `TestDerivedIDChangesOnlyWithOwnContent` | editing a contract moves only its own hash, never its ID |
| `TestLabelledContractIsAsWritten` | a `**C4**` is honoured; bold prose mentioning `C2` is not a label; collisions resolve as above |
| `TestLineRangesRoundTrip` | every ID selects its own bytes, local and project-qualified |
| `TestOutlineNestsAndCovers` | no non-blank line lies outside the outline |
| `TestSelectRoundTripsToBytes` | `--select` prints exactly the record's lines |
| `TestJSONIsDeterministic` | same bytes, same JSON |

### What the scanner reads, and what it does not judge

Contracts are read document-wide: records put ```` ```normative ```` blocks
under their own `#####` sub-headings or beside the design they specify,
and a block is a contract wherever it sits (`section` says where).
Contracts written as prose are not addressable; that shows as a zero
count, not a warning, because older epochs wrote them that way.

Assumptions are read from wherever Critical Assumptions lives — `##`,
`###`, or a legacy alias — in all four corpus forms (`**A1 [S]**`,
`**A1 — S.**`, `**A1** S`, `**A1 S**`). A section that labels its
assumptions also carries other bullets (a whole cohort copied the
template's Method-vocabulary legend in verbatim); those are not
assumptions. Only a label-free section — epoch A's checkbox list — has
its bullets read as assumptions by position, derived.

Gate responses exist only while the gate is inlined; a `gate.md` pointer
means they live outside the record.

Bold paragraph leads are indexed as ANCHORS, in their own list rather than
in the outline: a lead governs no lines and does not nest, so putting it
in the outline would break the coverage and nesting invariants that make
the outline worth having. It is not an element either — no class, no
fields, no lifecycle — only a piece of text with a name, recorded so a
citation reaching for it by that name lands somewhere. A lead at column
zero opening a paragraph is one; indented bold (a field, an assumption
label) already has an owner, mid-paragraph bold is emphasis, and a
contract's `**C4**` label is that contract's, not a second identity for
the same bytes.

A section whose heading ALIASES to a template section is read for its
elements even when the record's own epoch table has no such section — an
epoch A record writing `### Decisions` over `- **D1**` bullets has those
decisions whatever its template offered. The epoch classification is
unchanged (the heading is still reported recognised-and-unmapped for that
epoch, and no epoch table is bent for one record); only the elements
underneath become addressable. Author-numbered decision bullets key as
written (`D-6`), which is the spelling the citation grammar already reads.

A `NNNN-slug-postmortem.md` beside a record is not a record: `NNNN`
resolution and `index` skip it, and a file with no epoch fingerprint is
listed as skipped rather than silently dropped.

## Typed edges

An RDR states its relations in a dozen syntaxes, none of them
machine-checked. `edges[]` types every one of them, so "who cites this
contract?" is a lookup rather than an LLM reading two files.

Each edge is `{from, to, kind, resolved, line, line_end, evidence,
field}`. `from` is an element ID, or the record when the relation is the
document's own; `to` is what the kind's target class says it is.

| kind | read from | target |
| --- | --- | --- |
| `predecessor` | `- **Predecessors**:` | record or element |
| `overrides` | `- **Overrides**:` | record or element |
| `cluster` | `- **Cluster**:` | record |
| `moved-to` | `Demoted [→ …]` | issue or record |
| `joint-decision-home` | `Final [joint decision → <home §anchor>: …]` | element |
| `reverify` | `Draft [revised from Final …; re-verify A2,A4]` | this record's own assumptions |
| `peer-evidence` | a `Method: Peer RDR` assumption's Evidence | element |
| `transient-deleted-by` | the `Transient — scheduled deletion by …` marker | record |
| `cross-cutting-owner` | a Cross-Cutting Concerns citation | record |
| `source-anchor` | `path::Symbol`, anywhere in the body | symbol |
| `artifact` | `{SPIKE_DIR}` / `{ARTIFACT_DIR}` / `{EVIDENCE_DIR}` paths | path |
| `issue` | `_issues/NNNN`, `kata ahg1`, `kata #71` | tracker id |
| `rfd` | `RFD 0004`, `rfd/0004/…` | RFD |
| `mentions` | every other record reference in prose | record or element |

**Mentions is a kind, not a fallback.** A bare `cli/NNNN` carries no
stated relation, and there are thousands of them. Folding them into the
typed kinds would make every typed answer wrong; dropping them would lose
the only trace of most cross-references. `Kind.Typed()` separates them.

**A bare four-digit number is not a record** — except inside
`Predecessors`, `Overrides` and `Cluster`, whose whole value is a record
list. In prose, four digits is `RFC 6962`, `2000 rows × 1000 elements` or
`walker.go:1306` at least as often as a record, and a false edge is worse
than an absent one. Even in a record list, a `NNNN-DD-DD` is a date.

**A `REQ-N` names no element.** The implementation prompt mints REQ ids
per run into a `req-list.md` artifact, over every clause of the record;
they are not the record's C-numbers. The citation targets the document.

**Nor does a `G-` key outside the gate's closed set.** Every `G-` element
is minted from a Finalization Gate sub-heading, so the five `GateItems`
keys are the whole namespace. Records also coin `G-a`…`G-j` for their own
guard tables and `G-faithful` for a mode; reading those as gate responses
asserts an element the target cannot have under any spelling and reports
a correct citation broken forever. They target the document, for the same
reason `REQ-N` does.

**A record-shaped filename segment is not a citation.** The `NNNN-slug`
form is the one grammar with no marker of its own, and a path in the
RDR's own evidence tree matches it one segment in
(`{EVIDENCE_DIR}research/a5-0118-clause-spans.md`). A four-digit run whose
preceding hyphen follows an alphanumeric is mid-token, and is declined.

### Resolution

`resolved` is three-valued: `true`, `false`, or **absent**. Absent means
nothing looked — no records dir was given, or a source anchor was read
with no `--repo` to grep. An unchecked edge must never read as broken (a
finding a consumer chases) nor as sound (a skipped check reading as a
pass, which is the failure this flow exists to prevent).

**Resolution is exact; the parser never guesses.** A record must exist,
and a citation reaching inside it must land on an element that exists in
that record's projection. A section citation that names no section is
reported, and so is a number whose filename slug names a different
record.

A section citation may be a whole-word PREFIX of the target's heading
rather than the whole of it — in either direction, because the corpus
clips both ways: `§Normative` for `Normative Contracts`, `§Safety
boundary` for `Safety boundary (normative)`, and `§Failure-Modes residual
chartered it as a` where the citation grammar's word window ran past the
heading into the sentence about it. Such a citation resolves **only when
it stands in that relation to exactly one heading of the target**. Two
candidates and it stays unresolved — no first match, no longest match, no
tiebreak, because a tiebreak is the guess by another name. `§Semantic`
against a target with five `Semantic *` headings is still an
under-specified reference, i.e. record data, which is what an earlier
unguarded prefix rule got wrong and why it was reversed.

Bold paragraph leads (`**The values.**`) are addressable by their exact
name, in the `§` namespace, and by nothing else: a heading is named by
the template and repeated across the corpus, so a prefix of one
identifies it, while a lead is a sentence written once and a prefix of a
sentence is not a citation of it.

That is deliberate, and it is the division of labour: **structural
variants are absorbed generically** by the template model's aliases and
prefix matching, while **a dangling cross-document reference is record
data, not a parser tolerance case**. A dangling edge on a terminal record
is reported with its line range, and the fix is a minimal pointer
correction to the reference text — the one sanctioned amendment to a
terminal RDR. Nothing else about the record changes.

`source-anchor` resolution is tooling-pass CHECK 5's rule exactly: the
SYMBOL is what resolves, never the line number, and a symbol found
anywhere in the repo counts (the anchor moved; the claim stands).

A **receiver-qualified** anchor (`views.go::TableVertex.LiveConstraints`)
resolves to its member. No language writes the qualifier adjacent to the
member where it is declared — Go has `func (v *TableVertex)
LiveConstraints()` — so matching the dotted string whole reports a live
method as missing. That is a FALSE finding, strictly worse than the absent
verdict a skipped check gives, because a consumer chases it. The qualifier
is the author saying where the symbol lived; a move is a note, not a
finding. The member still matches as a whole word, so `Codec.Encode` does
not resolve out of `EncodeAll`.

An unmapped reference form — a citation shape no grammar reads — lands in
`warnings[]` as `edge:unmapped-reference`. It is **not** an unclassified
line: the rate measures structural drift against TEMPLATE.md, and a line
whose structure is fully read but whose reference is untyped is not that.

### Queries over the graph

`rdr index` projects the whole records dir once — sub-second over a
143-record corpus, byte-deterministic, no cache to go stale — and answers
the corpus-level questions the flow used to answer by opening every file.
With `--repo`, symbol resolution adds one walk of the source tree (~13s
over a 110MB checkout citing ~1.2k symbols); every symbol is tested per
file, so the cost is one traversal, not one per citation:

    rdr index [--json]            # the graph: records, elements, edges, derived backlinks
    rdr index --status            # records grouped by status
    rdr index --in-flight         # the worklist: Draft and Final (rdr-status no-arg mode)
    rdr index --backlinks         # the reverse edge set, transposed — never re-parsed
    rdr index --backlinks=0055:C4 # who cites this contract — typed edges and mentions
    rdr index --backlinks=0055    # who cites this record or anything in it
    rdr index --cluster-of N      # Stage 7.1's membership rule, as a query
    rdr index --anchor-intersect  # in-flight pairs sharing code anchors, uncited first
    rdr index --unresolved        # typed edges whose target was looked for and not found
    rdr index --readme[=PATH]     # the README index table checked against the records
    rdr index --derived           # the unlabelled-element backlog per record
    rdr index --coverage          # the drift alarm (§The resilience contract)

`--anchor-intersect` is the after-propose scan: two in-flight records
citing the same `path::Symbol` with no edge of any kind between them are
proposing to change one function without knowing of each other, which is
the shape that otherwise surfaces as a joint decision five gate iterations
later. Anchors match on the symbol with the path as a component-aligned
suffix (`uniqueid.go::f` is `internal/validate/uniqueid.go::f`); the
template's own `path::Symbol` is ignored; `--all` widens the scan past
in-flight records. On the consumer corpus's propose snapshots it fires
on the pair the flow missed.

`--readme` is a check, not a generator: it names each row that disagrees
with its record (status, title, priority, a missing or extra row) and the
author decides which side is wrong. Nothing here writes.

`--cluster-of` is the 7.1 prompt's own definition — "mutual
`**Predecessors**:`, Peer-RDR citations, or a shared Cross-Cutting
Concern" — evaluated over `edges[]` instead of by reading every
candidate. Mutual is strict: a one-way predecessor is the ordinary
build-order dependency every record has several of, and it resolves by
implementing one first. A declared `Cluster` field is authoritative on
its own. Each member reports the relation that earned it and its status,
so 7.1's Final-and-unimplemented scope is a filter, not a read.

Checked against seven clusters 7.1 actually reconciled, the typed rule
reproduced three exactly and missed members in the rest — records joined
to the seed only by dense prose cross-reference. So two in-flight records
that each mention the other are also reported, as `mutual-mentions` with
`candidate: true`: a lead to confirm, not an assertion the records make.
Two historical members had no citation in either direction; no rule over
the records recovers a membership the records never state, and a declared
`Cluster` field is the fix.

| test | asserts |
| --- | --- |
| `TestEveryEdgeSyntaxMaps` | every syntax in the table maps to its kind, on the fixtures |
| `TestUnmappedFormWarns` | a form no grammar reads becomes a warning with its line |
| `TestEdgesAreNotDuplicated` | a citation two passes reach is one edge |
| `TestMetadataIsReadByFieldNotByLine` | no date's year is ever read as a record |
| `TestPlaceholderFieldsMintNoEdges` | a seed's candidates are mentions, never declared relations |
| `TestUnresolvedIsAFinding` | citing `0001 A9` when 0001 has only A1 is caught |
| `TestStaleSlugIsUnresolved` | a number and slug naming two records is caught |
| `TestSectionCitationResolvesExactly` | an under-specified section citation is reported, not guessed |
| `TestUncheckedIsNotUnresolved` | with no corpus and no repo, `resolved` is absent |
| `TestSymbolResolution` / `TestSymbolIsAWholeWord` | CHECK 5's rule; `Encode` does not resolve out of `EncodeAll` |
| `TestUnresolvedEdgeCarriesARange` | a wrapped field's finding names its whole range |
| `TestReverseEdgesDerive` | backlinks transpose the forward set, reading no file |
| `TestClusterRuleIsAQuery` | 7.1 membership, including that a one-way predecessor is not a member |
| `TestIndexClusterCandidateTier` | mutual mentions between in-flight records are a candidate, one-way is nothing |
| `TestEdgesAreDeterministic` | same record, same edge bytes |
| `TestIndexGraphIsDeterministic` | the corpus graph carries every facet and two builds are the same bytes |
| `TestAnchorIntersectFiresOnUncitedOverlap` | the uncited pair leads, spellings merge, the placeholder is ignored, implemented records need `--all` |
| `TestIndexBacklinksToTarget` | `--backlinks=NNNN[:elem]` gathers typed edges and mentions for one target |
| `TestReadmeDriftIsACheck` | every drift class is named; an escaped pipe stays in its cell |
| `TestSummaryReadsMetadataNotBody` | in-flight is Draft or Final; terminal is the never-amended set |

## The resilience contract

The corpus is not template-uniform, and every ad-hoc parser before this
one silently undercounted because of it: Critical Assumptions at `###`
in most records, Evidence labels extended into clauses, Status values
carrying parentheticals and paragraphs, Metadata values wrapped across
lines. A projector keyed on exact literals produces confident, wrong
JSON — the worst outcome, because a downstream gate reads its silence
as PASS. So the contract is stated before the parsing, and tested:

- **Sections match by name, level-insensitively.** A section written at
  another epoch's level is the same section (`level-variant`).
- **Labels match by prefix.** A bullet label is the field followed by a
  boundary, not the whole bold text: `Evidence — the two channel
  reductions`, `If wrong (a refusal is owed)`, `Evidence (plan)` are the
  Evidence and If-wrong fields (`prefix`). The longest vocabulary label
  wins, so `Status / sentinel errors` is never read as `Status`.
- **Status normalises to `{value, qualifier, raw}`.** The lifecycle
  Status keeps its qualifier grammar (`form`); the Evidence Record's
  Status matches its vocabulary by prefix under any emphasis or case
  (`**Verified**`, `REFUTED`, `Verified as narrowed — …`). The template
  legend left unfilled is a `placeholder`, never `Verified`.
- **Nothing is dropped.** Every labelled bullet is a metadata field, a
  field of the element it sits in, a field of its section, or a warning.
  A label in no vocabulary is recorded as the author's (`author`) — not
  a finding. A heading the template does not know, nested inside a
  section it does, is the author's sub-structure
  (`author-subsection`) — not a finding. Only a foreign top-level
  section and an unknown Metadata label warn.
- **Read, never judge.** A parse warning on a terminal record is always
  a projector bug to fix with a fixture, never a reason to edit the
  record.

The **unclassified-line rate** is the property as a number: the share
of a record's non-blank lines that lie inside a warning's range
(`coverage` in the envelope). It is exactly zero on a record that
conforms to its epoch and near zero over the whole corpus, and a rise
after a `TEMPLATE.md` change is the drift alarm — it points at the
epoch entry the same-commit rule required and did not get. `rdr index
--coverage` reports it per record and in total, with warnings by code,
and lists every unknown heading and author label that recurs across
three or more records: one record's invention is the author's, the same
text across records is a convention or an unmodelled template addition.

The contract's tests run over every synthetic fixture, the four epochs
and one variant per known failure (`testdata/README.md`):

| test | asserts |
| --- | --- |
| `TestZeroSilentDrop` | every non-blank line lies inside the outline, nodes nest without overlap; every labelled bullet outside a fence is accounted for; `counts.fields` sums to the fields recorded; `coverage.unclassified` is exactly the warned non-blank lines |
| `TestConformantFixturesHaveFullCoverage` | a record conformant to its epoch has no warnings and rate zero |
| `TestHeadingLevelVariant` | Critical Assumptions at `###` in an epoch D record reads every assumption |
| `TestLabelVariants` | clause-extended labels match by prefix; a colon inside the bold is a label; an author's sub-field is recorded, not warned |
| `TestStatusForms` | the parenthetical, dash and joint-decision lifecycle forms; every Evidence Record Status form; the placeholder |
| `TestWrappedMetadata` | wrapped values join, a guidance comment ends them, a nested bullet is not part of the value; legacy and author labels classify; an unknown one warns and is still recorded |
| `TestAuthorStructure` | author sub-headings are not findings; a foreign section warns once; prose-named labels are observed, the author's are recorded |
| `TestIndexCoverage` | the corpus rate, warnings by code, and the recurrence table with its threshold |

## The four epochs

Drift across the record corpus is **epochal, not chaotic**. Within an epoch
conformance is near-total; the differences between epochs are additive,
each generation's section set containing its predecessor's.

| epoch | what it introduced | fingerprint |
| --- | --- | --- |
| A | the original template | a metadata block, no `Profile`, no Evidence Record `Method` field, gate responses inlined |
| B | `Profile`, `Seam Lineage`, `Load-Bearing Decisions` | any of those three present |
| C | the Finalization Gate body externalised | a `gate.md` pointer replaces the inlined responses |
| D | Critical Assumptions promoted to `##`; joint-decision qualifiers | Critical Assumptions at `##`, or a joint-decision status qualifier |

`DetectEpoch` reads the signals newest-first, because a record showing a
later epoch's signal also shows every earlier one.

### Why a fingerprint and not a `Template-version:` stamp

A stamp was considered and rejected. It would be one more metadata field to
drift; it is copyable-wrong from a neighbouring record, which is exactly
how metadata already spreads; and it would be absent from every record
already frozen, which is most of them — so the fingerprint would have to
exist anyway as the fallback. The fingerprint separates the four
generations cleanly with no cooperation from the author.

Add a stamp only if a future epoch turns out to be fingerprint-ambiguous
against its predecessor. Until then it buys nothing and costs a field.

## Canonical vs. observed vocabularies

Each closed vocabulary carries **two** tiers, and the split is doctrine
rather than convenience.

- **Canonical** — what a NEW record may write. Straight from `TEMPLATE.md`
  and `README.md`.
- **ObservedAccepted** — additionally present in the frozen corpus, and
  legitimate practice the template simply never listed. The reader accepts
  it with no warning, and a gate or lint consumer treats it as valid.

The doctrinal basis is that **terminal records are never amended**. An
`Implemented`, `Rejected`, `Abandoned` or `Superseded` record is frozen at
the template epoch that produced it, forever. Nobody can go back and
correct it, so a reader that called its values off-vocabulary would be
permanently and unfixably wrong about it. The model's job is to **read
those records, never to judge them**.

In practice only Status has an observed set: `Rejected`, written as a real
terminal disposition by frozen records that `TEMPLATE.md` never listed.
`Deferred` was in that set too, and its promotion out shows what the tier
is for. A frozen record had improvised it for a state the template could
not spell — parked with a revisit trigger, which is not `Abandoned` — so
the reader accepted it while the template gained the word; it is Canonical
now, and reading did not change. That is the tier working as intended: it
holds a value the corpus needs, visibly, until the template answers for it.
Method's observed set is empty, and that is a finding rather than an
oversight — every Method value in the corpus
resolves to one of the eight sanctioned labels once compounds are split and
parenthetical glosses are stripped.

There is deliberately **no third "tolerated but wrong" tier**. A value that
is neither sanctioned nor legitimate practice is a typo, and a typo is what
the gate exists to report; naming it in the model would launder it.

### Compound Methods

Records combine methods: `Source Search + Spike`, `Peer RDR + Source
Search`, `Source Search + Peer RDR + Spike`. A compound is **valid iff
every member is in vocabulary**, and a compound with one bad member is
off-vocabulary *on that member*, which `MethodValue.OffVocabulary` names so
a consumer can report the member rather than rejecting the value.

Two parsing details are load-bearing rather than cosmetic:

- **Glosses are excised before the value is split.** A parenthetical gloss
  is free text that frequently contains the separator itself — `Source
  Search (carrier + decode package boundary)` is ONE method with a gloss.
  Splitting first would manufacture off-vocabulary members out of
  conformant records.
- **A member's label ends at its first clause boundary.** Records continue
  a Method into prose (`Source Search, plus the Stage-4 I/O round`), and
  that prose is a note about the method, not part of its name.

## Wrapped values

Metadata values wrap. Most records wrap at least one — `Seam Lineage`,
`Predecessors` and `Overrides` most often, being long values a formatter
reflowed. Truncating at the newline loses the back half, which is where the
accretion trail and the override list live.

`ValueContinues` owns the predicate for where a value ends; the scanner does
the line joining. A value continues onto an indented line that is not
itself a bullet. It ends at a blank or unindented line, at the next
`- **Label**:` bullet, and at an HTML comment — the template's own guidance
comments are copied into instances verbatim and sit indented directly under
the field they annotate, so they are guidance, not value.

## Tolerant matching

`LookupSection` and `LookupField` classify an observed heading or field
against an epoch, and report **how** it matched. Only `MatchUnknown` is a
finding; everything else is a legitimate way a conformant record of some
epoch writes a template-drawn thing.

| kind | meaning |
| --- | --- |
| `exact` | name and level both match |
| `level-variant` | same section, written at another epoch's level |
| `case-variant` | differs only by case |
| `scaffold-instance` | a template scaffold with its placeholder filled in (`Alternative 2: …` for `Alternative 1: [Name]`) |
| `legacy-alias` | a recognised predecessor name, mapped to its canonical section |
| `recognized-unmapped` | recognised as something authors wrote, with no canonical home |
| `author-subsection` | unknown, but nested inside a recognised section: the author's own structure |
| `unknown` | genuinely foreign — the only thing worth a warning |

Bullet labels (`LookupLabel`) match `exact`, `case-variant` or `prefix`
against the section's field set — its canonical labels from
`TEMPLATE.md` and the labels its prose names (`SectionFields`) — and are
otherwise `author`. Metadata labels additionally pass through the field
alias table (`legacy-alias`, `recognized-unmapped`).

`scaffold-instance` matters more than it looks: the filled-in scaffolds are
the single largest class of heading a name-only lookup cannot place, and
treating them as foreign would bury every real finding.

## The seam binds itself

Every var this binary reads is written in a marker file the flow already
maintains, so it reads the marker rather than waiting to be told.

    cd anywhere/in/the/project
    rdr index --in-flight          # no --records, no exports, no seam bound

From the working directory it walks up for the project root, applies the
flow's own nearest-marker-wins rule (a repo-local `.rdr/workspace` beats
the shared `../.rdr-workspace`), sources the marker with `sh` — markers
are plain assignments that expand `$WS`/`$PROJECT` internally, so they
need a real shell, not a regex — and reads `RDR_RECORDS` and
`RDR_SOURCE_REPO` back out.

The binding order is **flag, then environment, then marker**. A flag is
someone spelling out a path; an exported var is a decision someone made;
the marker is only what fills the gap that would otherwise be an error.
No marker binds nothing at all — discovery, never invention — and the
caller fails exactly as it did before.

This exists because shell state dies between agent tool calls. A skill
that needed `$RDR_RECORDS` had to re-run a fifteen-line resolver or carry
an `export RDR_HOME=… RDR_RECORDS=… RDR_EVIDENCE=…` prefix on every
invocation — bytes on each call, a fresh chance to get a path wrong, and
neither carrying anything the marker did not already hold. `rdr-doctor`
check 11c watches that this keeps working, with the environment cleared
so an exported var cannot mask a broken bind.

`§seam-bind` is still the authority for what this tool does *not* read —
`$RDR_EVIDENCE`, `$RDR_ENV`, `$RDR_RESOURCES`, `$RDR_AUTOCOMMIT`.

## Naming a record, and reading part of one

A record is named by **number, slug or path**, and the number needs no
padding: `55`, `055` and `0055` are the same record, read decimally —
never as octal, the bug that once turned `0106` into `0070` and returned
a different real record without erroring. `--records` defaults to
`$RDR_RECORDS`, and a *relative* `--records` resolves against it rather
than the process's cwd, because a stage prompt runs from wherever the
harness happened to be.

Each of those is one fewer round-trip. A tool that answers `open 3: no
such file` to `3` costs a turn to diagnose and a turn to retry, and a
turn re-sends the whole conversation — far more than the bytes at stake.

**`--filter` keeps only the top-level keys you name**, comma-separated:

    rdr inspect --json --filter metadata,counts 0142
    rdr inspect --json --filter path 0142

`--select` answers *"give me exactly one facet"*. `--filter` answers the
other question, because the envelope is lopsided — on a large record
`elements` and `edges` are ~80% of it, and `inspect --json` is bigger
than the record it read on a quarter of the corpus. A caller wanting
status and counts paid the whole envelope for a few KB of answer, or
spent a second invocation to avoid it. On 0142:

| | bytes | |
| --- | --- | --- |
| `--json` | 160,092 | the whole envelope |
| `--filter metadata,counts` | 5,643 | −96.5% |
| `--filter warnings,coverage` | 1,242 | −99.2% |
| `--filter path` | 140 | what `§rdr-resolve` needs |

`schema`, `record` and `path` come back unasked: ~120 bytes that answer
"which record is this" so no caller spends a turn re-establishing it.

Keys are read off the marshalled envelope, so a filter can never name a
key the envelope lacks and no second list can drift from the struct. An
unknown key is `stopped:no-such-facet` **listing the valid keys** — never
an empty result, because a consumer handed `{}` for `--filter elments`
would conclude the record has no elements: a skipped check reading as a
passed one, which is the failure this flow exists to prevent.

## The usage log

`rdr` writes nothing — with one opt-in exception, off by default.

Set `$RDR_USAGE_LOG` to a path and every invocation appends one line
recording what was asked and what it cost. This exists because the
consumer-integration pass had to argue the tool's value from *estimated*
byte counts: nothing recorded what the binary was actually asked for or
how much it emitted. Now the numbers accumulate over ordinary use — no
dashboard, no session instrumentation, no second tool to run.

    RDR_USAGE_LOG=$RDR_EVIDENCE/usage.jsonl rdr inspect --select 0142:C1 --records "$RDR_RECORDS" 0142

    {"bytes_out":490,"cmd":"inspect","elapsed_ms":551,"exit":0,"facet":"select:element","target":"0142","ts":"2026-08-24T20:38:11-07:00"}

That line is the whole argument for the tool, measured rather than
estimated: quoting one contract costs **490 bytes** where reading the
record costs **156,350**. It also keeps the tool honest in the other
direction — `inspect --json` on the same record emits *more* than the
file does. The saving is in `--select` and the index facets, never in
the full envelope, and the log says so.

| field | |
| --- | --- |
| `ts` | RFC3339 with offset |
| `cmd` | `inspect` \| `index` \| `lint` |
| `facet` | which query ran (`json`, `select:element`, `in-flight`, `locking`, …) |
| `target` | the positional argument, when there was one |
| `bytes_out` | what was actually emitted, counted on the way out |
| `elapsed_ms` | wall time |
| `exit` | the exit code, failures included |

**Format.** One JSON object per line, keys sorted (records are marshalled
from a map, so the byte order is stable and the log diffs cleanly),
snake_case, optional fields omitted when empty. The field set is
**append-only**: a key may join, an existing one never changes meaning
or disappears, and a consumer must tolerate keys it does not know.

**No record content is ever logged.** A projection of prose must not leak
the prose into a log that outlives it; the log carries sizes and
identifiers, never bodies.

**`ts` is a deliberate exception** to the determinism rule the sibling
codebase holds to — that the same input yields the same bytes forever.
That rule is right for build artifacts and wrong for a cost log: a
record of *when* work happened is worthless without a clock. The clock is
confined to the log, and the projection never reads one.

**Failures are silent.** An unwritable path, a full disk, an
over-long line: none of them change what the tool prints or what it
exits with. Measurement is subordinate to the projection — a log that
could break an answer would be worse than no log.

## Layout

    tools/rdr/
      main.go              subcommand dispatch, flags, inspect, the per-record index facets
      usagelog.go          the opt-in usage log: $RDR_USAGE_LOG, one JSONL line per invocation
      seam.go              marker discovery: the records dir and source root, bound without a shell
      corpus.go            the corpus facets: graph, status, backlinks-to, anchor intersection, README drift
      internal/ident/      the element ID grammar, slugs, content hash
      internal/edge/       the typed relation model: kinds and reference grammars
      internal/scan/       the line scanner: outline, elements, warnings
        fields.go          the labelled-bullet pass: metadata, element and section fields; coverage
        edges.go           the typed-edge pass: every stated relation, by kind
        resolve.go         resolution against the records dir and the repo; backlinks; cluster derivation
        corpus.go          corpus queries: record summaries, the graph document, anchor intersection, README drift
      internal/model/      the template model
        template.go        sections, grammars, markers, the value-continuation rule
        fields.go          per-section label sets, prefix matching, the Evidence Record Status
        vocabulary.go      the four closed vocabularies; Method/Type/Profile parsing
        qualifier.go       the status qualifier grammars
        epoch.go           the four epoch tables and fingerprint detection
        alias.go           legacy names and the match kinds
        scaffold.go        filled-in template scaffolds
        template.go        also DecisionClasses and GateItems, the D- and G-key tables
      testdata/            synthetic fixtures, one per epoch, plus variants/ (see its README)
