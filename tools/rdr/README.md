# `rdr` — a read-only projector for RDR records

`rdr` reads RDR markdown and projects it as data. It never writes to a
record: **markdown remains the source of truth**, and everything here is a
view of it. If the model and a record disagree, the record is right and the
model has drift to fix.

`inspect` is live: it projects one record as an outline, a set of
addressable elements and a warnings channel, and resolves any element
ID back to the lines it names. `index --derived` reports the corpus's
labelling backlog. The remaining `index` facets and `lint` land
separately; until they do, they print `stopped:not-implemented` and
exit 2.

    rdr inspect 0055                       # one line per element, ids first
    rdr inspect 0055 --json                # the envelope: outline, elements, warnings, counts
    rdr inspect 0055 --select 0055:C4      # the bytes the id names
    rdr inspect path/to/0055-x.md --select outline
    rdr index --derived --records ../rdr/cli

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
| `0055:D-identity` | load-bearing decision | the template's decision class (`DecisionClasses`); else the label's slug |
| `0055:RT1` | round-trip invariant | an `RT1` / `INV-1` lead, or a unique list number; else ordinal |
| `0055:ALT2` | alternative | the `Alternative 2` scaffold ordinal |
| `0055:BR3` | briefly-rejected item | a unique list number; else ordinal |
| `0055:S5` | validation scenario | a unique list number; else ordinal |
| `0055:MVV` | minimum viable validation | none — one per record |
| `0055:F2` | failure mode | a unique list number; else ordinal |
| `0055:G-scope` | inlined gate response (epochs A, B) | the gate item (`GateItems`); else the heading's slug |
| `0055:§approach` | outline section | the canonical section's slug; scaffolds and legacy aliases slug their own heading |
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

A `NNNN-slug-postmortem.md` beside a record is not a record: `NNNN`
resolution and `index` skip it, and a file with no epoch fingerprint is
listed as skipped rather than silently dropped.

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

In practice only Status has an observed set: `Rejected` and `Deferred`,
both written as real terminal dispositions by frozen records that
`TEMPLATE.md` never listed. Method's observed set is empty, and that is a
finding rather than an oversight — every Method value in the corpus
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
| `unknown` | genuinely foreign — the only thing worth a warning |

`scaffold-instance` matters more than it looks: the filled-in scaffolds are
the single largest class of heading a name-only lookup cannot place, and
treating them as foreign would bury every real finding.

## Layout

    tools/rdr/
      main.go              subcommand dispatch, flags, inspect, index --derived
      internal/ident/      the element ID grammar, slugs, content hash
      internal/scan/       the line scanner: outline, elements, warnings
      internal/model/      the template model
        template.go        sections, grammars, markers, the value-continuation rule
        vocabulary.go      the four closed vocabularies; Method/Type/Profile parsing
        qualifier.go       the status qualifier grammars
        epoch.go           the four epoch tables and fingerprint detection
        alias.go           legacy names and the match kinds
        scaffold.go        filled-in template scaffolds
        template.go        also DecisionClasses and GateItems, the D- and G-key tables
      testdata/            synthetic fixtures, one per epoch (see its README)
