# `rdr` — a read-only projector for RDR records

`rdr` reads RDR markdown and projects it as data. It never writes to a
record: **markdown remains the source of truth**, and everything here is a
view of it. If the model and a record disagree, the record is right and the
model has drift to fix.

This directory currently contains the plumbing and `internal/model`, the
machine-readable model of `TEMPLATE.md`. The line scanner that feeds
`inspect`, `index` and `lint` lands separately; until it does, those
subcommands print `stopped:not-implemented` and exit 2.

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

A divergence names itself and says what to update. Gaining, losing,
renaming or re-levelling a section, or flipping its class, all fail — each
was verified by mutating `TEMPLATE.md` and watching the test fail before
the file was restored.

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
      main.go              subcommand dispatch, flags, version
      internal/model/      the template model
        template.go        sections, grammars, markers, the value-continuation rule
        vocabulary.go      the four closed vocabularies; Method/Type/Profile parsing
        qualifier.go       the status qualifier grammars
        epoch.go           the four epoch tables and fingerprint detection
        alias.go           legacy names and the match kinds
        scaffold.go        filled-in template scaffolds
      testdata/            synthetic fixtures, one per epoch (see its README)
