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

    rdr inspect 55                         # sections (§, nested) then elements, one line each: id, range, byte size, label
    rdr inspect --grep 'RetryBudget' 0055  # which elements hold the literal (case-sensitive): id, range, hit lines, first match; a miss is no-match at exit 0
    rdr inspect --touched-since abc123 --json 0055   # which ids the diff from that rev to the working tree overlaps: .touched[].{id,kind,line_start,line_end}, the hunks beside them
    rdr inspect --json 0055                # the envelope: outline, elements, anchors, metadata, fields, edges, warnings, coverage, counts
    rdr inspect --json --filter metadata,counts 0055   # only those keys — one call, ~4% of the envelope
    rdr inspect --select 0055:C4 0055      # the bytes the id names
    rdr inspect --select edges 0055        # the typed relations alone
    rdr inspect --select 0055:A1 --select 0055:C4 0055   # several, in order; text is the bytes in sequence, --json an array
    rdr inspect 0055 0056 0057             # a named set: each record's projection in turn, --json with identity per row
    rdr inspect cli/0055:C4                # the corpus's own citation spelling, as printed by lint, reaches the bytes
    rdr index --derived --records ../rdr/cli
    rdr index --coverage --records ../rdr/cli
    rdr index --unresolved --records ../rdr/cli --repo ../src
    rdr index --cluster-of 0130 --records ../rdr/cli
    rdr impact 0113 --literal '.dml.sql'   # the predecessor tests the record's changes will turn red: rows by family, from its override/predecessor edges and the retired literals

Go stdlib only — no third-party dependencies, by design. RDR markdown is
line-oriented (headings, fences, bullet trees with bold labels), so a line
scanner is both smaller and more maintainable than a generic AST adapter,
and a static binary gives `rdr-doctor` one thing to check for.

    go build ./...
    go vet ./...
    go test ./...

## What this tool is for

`rdr` exists to make the **skills in this repo** cheaper and more
reliable, and that is the only measure of a change to it. Before it
existed, every turn of the flow that needed a structural fact about a
record — its Status, which sections are filled, how many assumptions are
Verified, which peers it cites — answered by having a model re-read the
markdown. Records run ~1,200 lines at the median and past 5,000 at the
tail, so a structural question cost a whole-file read, and the ad-hoc
parses were wrong in known ways: a section that is `###` in most records
and `##` in a few, Evidence labels that vary, Status values carrying
qualifiers. A parser keyed on the template literal silently undercounts,
and a skipped check must never read as a passed one.

So the tool answers those questions deterministically, and the skills
call it instead of reading. The saving is real but it is **not uniform**,
and the shape matters more than the headline:

    --select NNNN:C1      ~500 B    the contract-quoting path
    status (no arg)       ~2 KB     the worklist WITH each record's facts
    status NNNN --tags    ~1 KB     one record's whole position, as argv
    lint --locking        ~3 KB
    inspect --json        LARGER than the record it read

The full envelope costs more than the file. The saving lives in
`--select` and the index facets — the narrow reads — which is why the
skills are written to ask narrow questions. `$RDR_USAGE_LOG` measures
this per invocation; it is opt-in and off by default.

Consumption is deliberately **not a hard dependency**. The projection is
an accelerator, not a prerequisite: a caller without the binary degrades
to the old hand read rather than stopping, and the sites that must not
fail — the commit receipt among them — check for it and proceed with a
note. A skill spawning a sub-agent that has no access to this file says
the same thing the long way, by pasting the three reads it wants.

### What it refuses to do

- **It never writes to a record.** Markdown is the source of truth; the
  projection is derived and never authoritative. If the two disagree,
  the record is right and the model has drift to fix.
- **It makes no semantic judgment.** Contradiction, hollow-vs-thin
  prose, whether a claim is any good — none of that lives here. The
  projector reads structure; lint applies mechanical rules.
- **It never guesses.** Resolution is exact, `resolved` is three-valued,
  and an ambiguous citation stays unresolved rather than taking a first
  or longest match. A tiebreak is a guess by another name.

### Reading a finding before acting on it

**Findings are the intended output, not a defect.** The conformance
rules are forward-only: a terminal record's content is never amended
(structure migrates — §Identifiers), and a live record carries findings
until the stage already rewriting it fixes them in-pass. A Final record
exiting non-zero under `--locking` is an ordinary state, not a broken lock.

The corpus audit that closed this tool's build-out is the cautionary
tale, and it is worth restating because the mistake is easy to repeat.
Of the record-to-record findings examined, **a clear majority were
tool-grammar gaps, a minority were real record errors, and several were
correctly unresolved.** Bulk-editing records to satisfy the tool would
have rewritten a large number of *correct* citations. The discipline
that follows:

**Classify before editing anything.** Sort each finding into
tool-grammar gap, record error, or correctly-unresolved, and fix the
class — never the symptom. A high finding count is evidence about the
grammar as often as about the records.

**A false finding is strictly worse than an absent one.** An unchecked
edge that reads as broken sends a consumer chasing it. This is why a
wrong `--repo` root is worse than none, why receiver-qualified anchors
match their member, and why `REQ-N` and an author's own `G-<slug>`
resolve to the document rather than asserting an element the target
cannot have.

**The rule's stated rationale may be narrower than its purpose.** A
comment gives one reason a rule exists; TEMPLATE.md and the commit that
introduced it give the rule. Read those before concluding a finding is
spurious — the guard in a neighbouring rule is usually that rule's own
plumbing, not shared doctrine.

Known-open work is tracked outside this repo. Source anchors span more
than one repo: over one corpus audit, 142 of 173 unresolved edges were
source anchors, nearly all naming third-party codebases cited for
contrast, which want a multi-root `--repo` and an `external` verdict
rather than `false`. The remainder of the mechanical tooling pass is
still an AI prompt where it could be a script. The lint receipt
(§receipt) closes the gap where a stage could close a gate without lint
ever running; it is built and enforced at commit, pending confirmation
over a full stage run.

## The same-commit rule

**A `TEMPLATE.md` change that adds, removes, renames or re-levels a
section, or changes a class, field, label or qualifier grammar, ships in
the same commit as the marker or sidecar entry that carries it.**

`TEMPLATE.md` is the only schema an RDR has and it is prose, so every
consumer that needed its section names, classes, vocabularies and
qualifier grammars restated them by hand — and they drifted. Four
template generations became tribal knowledge because nothing checked.

The first answer was a hand-maintained Go table with a test standing
between it and the file. That worked, and it was still one fact stated
twice. **The binary now reads `TEMPLATE.md` at startup**, so the table it
used to check no longer exists to disagree with: section names, levels,
nesting, classes, the Keys column, the metadata and Evidence Record field
sets, the Status/Type/Profile vocabularies, the decision classes and the
gate items all come out of the document. Method comes out of this
README's *Verifying load-bearing claims*, which declares itself
authoritative precisely so the guidance does not ship inside the
template.

Where the template can carry a fact, it carries it — as a bracket marker
beside the section it describes, deleted by the author with the rest of
the guidance:

| marker | says |
| --- | --- |
| `[Required — …` / `[Conditional — …` | the section's omission rule |
| `[Conditional scaffold — …` | this block is a per-instance slot, or (on a parent) that a child it names is |
| `[Retained at lock — …` | this gate sub-section stays in the record when the rest move to `gate.md` |
| `[Gate key: <key>]` | the id a gate response is cited by — `cli/NNNN:G-<key>`, stable across a reworded heading |

Three things have no home in the template and live in
`models/rdr-template.toml`, which ships beside it: the element-kind map
(a reader fact — the template never names the ids a projector mints), the
observed vocabulary tiers (corpus facts, which exist *because* the
template never listed them), and the status lifecycle (stated in the
template only as English inside comments, and parsing a set out of prose
is guessing).

The element-kind map is what the **projector** reads to find a kind's
section, not merely what answers "does the template key this kind". A
kind the map does not name projects from no section list at all, and
that is a statement rather than an omission: `G` is keyed by the gate
item, `JC` by a line anywhere in the record, `§` by the heading itself.
So retargeting a kind — or pointing the binary at a second document
family — is a table edit, not a code edit; nothing in `internal/scan`
writes a section name of its own.

What still binds, mechanically:

| test | asserts |
| --- | --- |
| `TestSidecarNamesNoUnknownSection` | every section the sidecar names still exists in `TEMPLATE.md` — rename one without updating the other and this fails |
| `TestProjectedSectionsComeFromTheSidecar` | each list-projected kind reads the section `[elements]` names it — a section name reintroduced as a Go literal drifts from the table and this fails |
| `TestGateKeysComeFromMarkers` | every Finalization Gate sub-section declares a `[Gate key: …]`, and the set matches what the reader projects |
| `TestBulletGateProjectsTheSameItems` | a gate written as a labelled list projects the same keys as one written as sub-headings, and the two shapes never both fire |
| `TestMethodVocabularyMatchesREADME` | the eight Method labels match this README's authoritative list — still two documents, so still a real check |
| `TestLoadReadsWhatTheTemplateStates` | the loader returns the values `TEMPLATE.md` visibly writes |
| `TestSidecarDeclaresWhatTheReaderNeeds` | every minted element kind has a section, and the lifecycle sets are the ones lint and `rdr status` depend on |
| `TestSidecarRefusesRatherThanSkips` | an unrecognised sidecar shape is an error naming it, never a silent default |
| `TestObservedSectionsResolve` | the one label table left in Go — the corpus-derived `Observed` tier — still names sections the template has |
| `TestFixturesKeepTheirShapeSignals` | the fixture set still spans the shapes the corpus contains |

A missing or unreadable template is `stopped:no-template` and exit 2 — not
a zero schema. A zero schema does not fail; it succeeds wrongly,
classifying every value off-vocabulary and every heading
unknown-to-template, so `lint` would report a missing install as a
corpus-wide defect. `receipt` is the one exemption: it reads only the
usage log, and `§commit` refuses a commit without it.

## Identifiers

Every element of a record has one ID, in one grammar:

    [<project>/]<NNNN>:<KIND><key>

| id | element | key is |
| --- | --- | --- |
| `0055:A3` | assumption | the `A3` label, as written (`A4b` / `A1.b` → `A4b`, `A1b`) |
| `0055:C4` | normative contract (a ```` ```normative ```` block) | a `**C4**` / `##### C4` label on the line above the fence; else the block's document ordinal |
| `0055:L-3` | a clause inside a contract fence | the label as written at column zero of a definition line (`L-3  …`, `REQ-2 (…)`, `NC-1:`; any 1–3 capitals but `D-`/`G-`, which are the decision and gate grammars); `parent` names the contract. Labels are unique per record: one defined twice mints nothing, warns `clause:duplicate`, and `--select` refuses it as `ambiguous-element` naming both |
| `0055:D-identity` | load-bearing decision | the template's decision class (`DecisionClasses`); else the author's own number (`D-6`); else the label's slug |
| `0055:RT1` | round-trip invariant | an `RT1` / `INV-1` lead, or a unique list number; else ordinal |
| `0055:ALT2` | alternative | the `Alternative 2` scaffold ordinal |
| `0055:BR3` | briefly-rejected item | a unique list number; else ordinal |
| `0055:S5` | validation scenario | a unique list number, or a table row's lead (`T-5`); else ordinal |
| `0055:MVV` | minimum viable validation | none — one per record |
| `0055:F2` | failure mode | a unique list number; else ordinal |
| `0055:JC2` | `Joint-check:` line, parsed (`joint`: verdict, targets, home, `open`) | document ordinal |
| `0055:G-scope` | inlined gate response | the gate item (`GateItems`); else the heading's slug. The namespace is CLOSED to those five keys — a citation of a record's own `G-a` guard table names the document, as `REQ-N` does |
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

A derived ID means one of two things, and **the template decides which**.
Where the section an element is projected from shows its items carrying a
key — a numbered list (`1. **Scenario**:`), a labelled lead (`- **A1
[Statement]**`, `**C1**`) — a derived ID is a **backlog**: the slot is
empty and writing the key pins the ID against an insertion. Where the
section shows an unnumbered bullet (`- **[Alternative N]**:` under
Briefly Rejected) or plain prose (Failure Modes), there is no key to
write: the ordinal *is* the identity, no edit improves it, and `index
--derived` reports the count in a `structural:` tail rather than as a
backlog column.

That fact lives in one place — `Section.Keys` in the template table,
bound to TEMPLATE.md's own body by the same-commit test — so a template
that starts numbering Briefly Rejected changes one row and the reader
follows. `Grammar` cannot answer it: Testing Strategy, Briefly Rejected
and Failure Modes are all `GrammarProse`, and only the first shows a
numbered list.

Sections work the same way, and only one of their derivation causes is
work: a **legacy alias** is a reformat in the waiting, retired when the
record is migrated to the canonical heading. A scaffold instance (`###
Alternative 2: …`), the author's own sub-heading, and a heading
recognised with no canonical home are the author's text — permanent, and
no template edit labels them. So `§` reports 39 of 6,190 as backlog, not
1,400.

Both are minted IDs naming real bytes; only the first names work.
Counting them together reported 976 pending edits corpus-wide against a
real backlog of zero, and made "stop minting them" look like the way to
clear the queue — which would strip the IDs off 976 elements and unanchor
the 236 edges written from them.

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

A contract's clauses carry their own labels inside the fence — `L-1  …`,
`REQ-2 (…)` — and the corpus cites them one grain below the contract
(`cli/0112 §Normative Contracts L-3`); `--select 0112:L-3` answers the
clause's lines, not the 500-line contract's. The label is the clause's
name within the record, so a second contract restarts nowhere: the next
fence continues the numbering or takes another letter.

A cross-record reference to a load-bearing element — an assumption, a
contract, a scenario, a decision — is written as an ID: `0055:C4`,
`cli/0055:A3`. The grammar reads the colon form, the spaced form
(`cli/0055 A5`) and `§Section Name` alike, so older citations keep
resolving; the colon form is what new ones use. For a clause the colon
form (`cli/0112:L-3`, `cli/0113:REQ-12a`) is the one the edge grammar
resolves to the clause; the prose spellings (`cli/0112 L-3`, `cli/0112
§Normative Contracts L-3`) remain document and section references by
design, so no existing citation is asked to change. A filename or heading-text
reference is a `mentions` edge: fine for context, wrong for a claim, since
it is exactly what a reword or a move breaks.

`Method: Peer RDR` Evidence names an element, never a bare record. A
citation that resolves to a whole 4,000-line record has named a document,
not a reason — `rdr lint` reports it as `peer-evidence:no-element`.

**Content is never amended; structure is migrated.** A terminal record's
prose, verdicts and decisions are frozen. Its *structure* may be brought
to the current `TEMPLATE.md` by tooling, ids and content bytes preserved.
Migratable, and nothing else: heading level and name; element labels
(the label written is the id the projector already derives, so no
citation moves); citation form; Status/Method/Type spelling; the gate
pointer (inline responses move to `artifacts/gate.md`); evidence-tree
location. After migration, A, C, D, S, RT and ALT carry written ids; BR
and F carry none. There is one schema — the current template — and a
section a record never had stays absent, which is not a finding on a
terminal record. Delivery is `lint` fix fields applied by a
script, never a write verb: this tool still writes no record. Live
records get labels the same way, in-pass.

## Lint

`rdr lint [<NNNN>] [--locking]` is the conformance authority: one pass,
three severities, and a rule about which records each may speak about.
With no argument it lints the whole records dir. It exits 0 on PASS —
findings or not — and 1 when a finding blocks a lock.

There is ONE reading, and it is the strict one: every record judged
against the current TEMPLATE.md, terminal ones included, each mechanical
finding carrying the bytes that would repair it. `--locking` is the only
lint flag, and it changes delivery, never what is found.

| tier | scope | blocks? |
| --- | --- | --- |
| `parse` | every record | never |
| `conformance` | every record | never |
| `resolution` | every record | at a lock gate, on a live record |

**parse** republishes the scanner's warnings channel. On a terminal record
a parse warning is a projector bug — the file cannot have changed, so the
scanner is what is wrong — and the fix is a fixture.

**conformance** is migration advice: unlabelled contracts, a Required
section the current template carries and this record does not, a heading
written at a non-canonical level, a citation in a non-canonical form, and
TEMPLATE.md's own `[Required — …]` / `[Conditional — …]` guidance left
standing where the author's words belong (`placeholder:survived`). It
is phrased for the stage already holding the file open, and it never
blocks — on ANY record. Terminal records get it too, because a frozen
record's CONTENT is never amended but its STRUCTURE may be migrated
(§Identifiers), so the advice is actionable and withholding it only hid
what a migration costs. Two exclusions keep it honest: a subsection whose parent is absent is not
separately missing, and a gate subsection under a `gate.md` pointer is not
missing at all — at lock those responses move out of the
record on purpose. Cross-Cutting Concerns is the exception it keeps: it
states a policy peer RDRs cite, so a locked gate holds the pointer AND
that one subsection; a pointer without it — a record locked before the
template retained it — is `gate:cross-cutting-missing`.

**resolution** judges what a record EMITS: every typed edge resolves, every
`Method: Peer RDR` Evidence names an element, and contracts are labelled on
a record written after the rule landed. This is the tier that applies to
terminal records too, because it is not about their shape. The DELIVERY
differs — on a frozen record the finding is a fix pointer carrying the line
range to open, and it does not block, because that record is not the one
locking. Correcting the reference text in that range is the one sanctioned
content amendment to a locked RDR: the pointer only, never prose.

The label rule's boundary is the record's own `Date`, not a template
fingerprint. A fingerprint would beg the question — the signal placing a
record in a "labels its contracts" era is the presence of labelled
contracts, so a new record that labelled nothing would fingerprint as
legacy and escape the check the rule exists to apply. `Date` is written by
Seed on every record and says when it entered the flow, which is what the
grandfathering rule actually asks.

### Patches: the migration, priced

Lint attaches a machine-applicable `patch` — `{line_start, line_end, op,
text}`, where `op` is `replace`, `prepend` or `insert` — to each finding
whose repair is computed rather than judged. It changes no verdict:
everything it adds is advisory, and a corpus that blocks nothing exits 0.

This was `--strict`, an opt-in beside a narrower default that spoke only
about live records. The narrower reading was a kindness aimed at the
wrong thing: a terminal record's STRUCTURE may be migrated
(§Identifiers), so the advice was always actionable, and withholding it
only hid the cost. What the flag bought — a quieter report — is the
tier's job, and the tier already does it. Delivery is still a patch field
read by a script; this tool writes no record.

| rule | patched | withheld when |
| --- | --- | --- |
| `heading:level` | the heading, re-levelled | the promotion would capture a following sibling |
| `label:missing` | the id the projector already derived | the section labels nothing, the fence is indented or fenced, the label would not read back |
| `citation:form` | the line, every citation on it at once | the target is a section, the text is fenced or repeats in range |
| `section:legacy-name` | never | always — see below |
| `gate:inline` | never | always — a cross-file move |
| `gate:cross-cutting-missing` | never | always — the Gate re-answers the retained item; the fix names the gate.md holding the prior text |
| `placeholder:survived` | never | always — a block above authored content and one standing in for missing content need different repairs, and telling them apart is judgment |

**The patch set is a fixpoint, applied bottom-up.** A second pass over
the result proposes nothing: conformance is reached in ONE pass, and
there is no iterate-until-clean loop to run or bridge between. Applying
a set leaves the graph — every element id and every resolved edge —
unchanged. That was 911 patches when the rule landed; the 144-record
corpus has since been migrated past all of them and proposes none, so
what remains are the findings whose repair is withheld by design. Two properties an
applier depends on: patches are applied bottom-up by `line_start`, and
findings that repair the same line SHARE one patch, so it must
deduplicate by identity before applying.

**A rule earns a patch only where its repair is exact**, and the corpus
taught which those are. Re-levelling moves no id, because a canonical
section takes the canonical slug at any level — but it can change what the
section CONTAINS, so a promotion that would swallow a sibling is reported
and not patched. A renamed legacy heading is never patched at all: the
section's id is a slug of its heading text, so a rename moves the id, and
the canonical name carries a canonical level that can re-parent the body
under it. A label is never written into a section that labels nothing,
because there the projector is reading elements BY POSITION — a tolerance,
not the record's claim — and writing those ids down converts a guess into
an authored fact. Section citations are never rewritten, because a `§Name`
citation is a bounded fragment while the id is the slug of the whole
heading.

Everything withheld is still REPORTED. The findings are the work list for
the hand pass, with lint as its checker.

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
| `TestStrictPatchesPreserveTheGraph` | applying every patch moves no element id, section id or edge |
| `TestStrictPatchesAreIdempotent` | a second pass over the result proposes nothing |
| `TestConformanceAdvisesAndNeverBlocks` | a terminal record gets conformance findings and still verdicts PASS, locking or not |

### What the scanner reads, and what it does not judge

Contracts are read document-wide: records put ```` ```normative ```` blocks
under their own `#####` sub-headings or beside the design they specify,
and a block is a contract wherever it sits (`section` says where).
Contracts written as prose are not addressable; that shows as a zero
count, not a warning, because older records wrote them that way.

Assumptions are read from wherever Critical Assumptions lives — `##`,
`###`, or a legacy alias — in all four corpus forms (`**A1 [S]**`,
`**A1 — S.**`, `**A1** S`, `**A1 S**`). A section that labels its
assumptions also carries other bullets (a whole cohort copied the
template's Method-vocabulary legend in verbatim); those are not
assumptions. Only a label-free section — the older checkbox list — has
its bullets read as assumptions by position, derived.

Gate responses exist only while the gate is inlined; a `gate.md` pointer
means they live outside the record. The exception is Cross-Cutting
Concerns, which a locked record keeps — it is the one gate item other
RDRs cite, addressed as `cli/NNNN:G-cross-cutting`, and an element that
is not projected cannot be cited.

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
elements even when the heading is not the canonical name — a record
writing `### Decisions` over `- **D1**` bullets has those decisions
whatever it called the heading. The classification is
unchanged (the heading is still reported recognised-and-unmapped for that
and the template table is never bent for one record); only the elements
underneath become addressable. Author-numbered decision bullets key as
written (`D-6`), which is the spelling the citation grammar already reads.

A `NNNN-slug-postmortem.md` beside a record is not a record: `NNNN`
resolution and `index` skip it, and a file that is not a record is
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
| `joint-decision-home` | `Final [joint decision → <home §anchor>: …]`, and a `Joint-check: fired … (home: <ref>)` line (from the `JC` element; `OPEN` mints nothing) | element |
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

Resolution is paid for **only by the facets that can show a verdict**.
`--unresolved` queries the verdict itself and `--backlinks` carries it on
every row, so both resolve; with `--repo` that adds a walk of the source
tree, every symbol tested per file — present or absent — so the cost is
one traversal and not one per citation. `--cluster-of` reads no verdict — it walks the edge
graph, three edge kinds and a direction test — and so resolves nothing:
14.1s to 0.89s over the reference corpus, byte-identical output. The rule
is `inspect`'s rule at corpus scale: resolve when a facet can show it,
never because the corpus happened to be in hand.

    rdr index [--json]            # the graph: records, elements, edges, derived backlinks
    rdr index --status            # records grouped by status
    rdr index --backlinks         # the reverse edge set, transposed — never re-parsed
    rdr index --backlinks=0055:C4 # who cites this contract — typed edges and mentions
    rdr index --backlinks=0055    # who cites this record or anything in it
    rdr index --cluster-of N      # Stage 7.1's membership rule, as a query
    rdr index --cluster-of N[,M] --closure --final-unimplemented   # 7.1 step 1: the rule to a fixpoint, scoped; out_of_scope[] says what was dropped and why
    rdr index --topo[=N,M,…]      # build order over predecessor edges: Kahn, ties by Priority then number; cycles[] and external[] apart
    rdr index --anchor-intersect  # in-flight pairs sharing code anchors, uncited first
    rdr index --literal-intersect # in-flight pairs whose contracts share a literal, uncited first
    rdr index --anchor-intersect --record 0113   # only the pairs touching one record; also --literal-intersect, --open-joint
    rdr index --json --filter records,elements   # only the named graph keys
    rdr index --unresolved        # typed edges whose target was looked for and not found
    rdr index --readme[=PATH]     # the README index table checked against the records
    rdr index --derived           # the labelling backlog per record, structural ids apart
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

`--literal-intersect` is the same scan over CONTRACTS: two in-flight
records naming one error code, flag, field or sentinel inside their
```normative fences. It is a separate arm of the same check rather than a
widening of the one above, because the couplings differ — a shared anchor
is two records editing one function, a shared literal is two records
specifying one surface — and a reader acts on them differently.

`--record NNNN` scopes either arm (and `--open-joint`) to the rows touching
one record, in any spelling a record is named by (§Naming a record). The
propose stage asks "does THIS record appear in a pair", and one session
answered it by filtering every pair with inline python, twice. The corpus
is still scanned once — a pair needs both sides — and the JSON carries
`record` so a reader knows the scope it is looking at. A record that does
not resolve is a stop, never an empty scope: no rows for a misspelt number
would read as "nothing intersects", the arm's false clear.

The obvious query is not this one. Two contracts with the same content
`hash` finds NOTHING (0 of 316 on the reference corpus): the hash is
exact-text identity, and no two authors write a contract the same way.
The question is a shared token inside otherwise-different text, which on
the same corpus reports 36 pairs of which 4 are uncited. Literals shorter
than three bytes are dropped as words rather than decisions, and
`TEMPLATE.md`'s own literals are subtracted so a record that kept its
guidance does not link to every other record that did. A frequency
ceiling for corpus vocabulary was tried and removed — it changed no pair
here, and it made `--all` lose a fire the narrower scope reported.

`--filter` is inspect's flag at corpus scale, with the same semantics:
the named top-level graph keys (`records`, `elements`, `edges`,
`backlinks`), `schema` carried unasked, and an unknown key a stop rather
than an empty answer. It earns its place on size — the graph is the
largest thing this tool emits, 5.6 MB on the reference corpus, and a
caller wanting `elements` was paying for `edges` and `backlinks` too.
Because only `edges` and `backlinks` can carry a `resolved` verdict, a
filter that keeps neither also skips the repo walk that decides one.

`--readme` is a check, not a generator: it names each row that disagrees
with its record (status, title, priority, a missing or extra row) and the
author decides which side is wrong. Nothing here writes.

`--cluster-of` is the 7.1 prompt's own definition — "mutual
`**Predecessors**:`, Peer-RDR citations, or a shared Cross-Cutting
Concern" — evaluated over `edges[]` instead of by reading every
candidate. Mutual is strict: a one-way predecessor is the ordinary
build-order dependency every record has several of, and it resolves by
implementing one first. A declared `Cluster` field earns membership on
its own — it is the author's assertion, not something the graph has to
confirm. Each member reports the relation that earned it and its status,
so 7.1's Final-and-unimplemented scope is a filter, not a read.

Neither the query nor the field is the membership. The query cannot see a
peer that cites nobody; the field is written at Propose and frozen at
Final, so it cannot see a later joiner. They are lossy in opposite
directions, which is why 7.1 unions them and judges rather than taking
either as the answer — and why TEMPLATE.md tells an author to declare the
field when peers will not cite each other.

Checked against seven clusters 7.1 actually reconciled, the typed rule
reproduced three exactly and missed members in the rest — records joined
to the seed only by dense prose cross-reference. So two in-flight records
that each mention the other are also reported, as `mutual-mentions` with
`candidate: true`: a lead to confirm, not an assertion the records make.
Two historical members had no citation in either direction; no rule over
the records recovers a membership the records never state, and a declared
`Cluster` field is the fix.

**What changed since a rev, as ids.** `inspect --touched-since REV` is
the re-entry scope: the diff from REV to the working tree (`git diff
-U0`, issued by the binary) read as post-image hunks, intersected with
the record's ranges — every element and section a hunk overlaps, a
nested clause and its contract both, a pure deletion touching the id
that ends before it or begins after it. The answer is the id list a
stage scopes by, with the hunks beside it so an empty `touched: []` is
evidence, not silence; a rev git cannot diff is `stopped:no-diff`, never
an empty set. A record renamed since REV has no history at that path
and diffs as a whole-file add: every id is touched, which is the honest
answer. Two stages read it — Stage 4's demotion route-back and
pre-lock's `lens_stale` re-entry — and neither intersects hunks with
line ranges by hand any more.

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
  another level is the same section (`level-variant`).
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
conforms to the template and near zero over the whole corpus, and a rise
after a `TEMPLATE.md` change is the drift alarm — it points at the
template entry the same-commit rule required and did not get. `rdr index
--coverage` reports it per record and in total, with warnings by code,
and lists every unknown heading and author label that recurs across
three or more records: one record's invention is the author's, the same
text across records is a convention or an unmodelled template addition.

The contract's tests run over every synthetic fixture
and one variant per known failure (`testdata/README.md`):

| test | asserts |
| --- | --- |
| `TestZeroSilentDrop` | every non-blank line lies inside the outline, nodes nest without overlap; every labelled bullet outside a fence is accounted for; `counts.fields` sums to the fields recorded; `coverage.unclassified` is exactly the warned non-blank lines |
| `TestConformantFixturesHaveFullCoverage` | a conformant record has no warnings and rate zero |
| `TestHeadingLevelVariant` | Critical Assumptions at `###` reads every assumption |
| `TestLabelVariants` | clause-extended labels match by prefix; a colon inside the bold is a label; an author's sub-field is recorded, not warned |
| `TestStatusForms` | the parenthetical, dash and joint-decision lifecycle forms; every Evidence Record Status form; the placeholder |
| `TestWrappedMetadata` | wrapped values join, a guidance comment ends them, a nested bullet is not part of the value; legacy and author labels classify; an unknown one warns and is still recorded |
| `TestAuthorStructure` | author sub-headings are not findings; a foreign section warns once; prose-named labels are observed, the author's are recorded |
| `TestIndexCoverage` | the corpus rate, warnings by code, and the recurrence table with its threshold |

## One template, read never judge

There is one template, and it is the file: `model.Template()` returns
`TEMPLATE.md` as the binary read it at startup. Every record is read
against it whatever its age.

The reader used to carry four **epoch** tables and a fingerprint that
placed each record in the generation that produced it, so a frozen record
was judged by its own template rather than today's. That machinery is
gone. It bought a kinder verdict on old records and cost a second
template model to keep in step; the same kindness now comes from the
lint tier a finding lands in, which is where it belonged.

What the epochs actually varied still exists in the corpus, and the
reader still absorbs it — but as TOLERANCE, not as a second table:

| the old variation | how it reads now |
| --- | --- |
| Critical Assumptions at `###` rather than `##` | `level-variant` — the same section |
| gate responses inlined rather than a `gate.md` pointer | projected as `G-<item>` elements |
| gate responses written as a labelled LIST rather than sub-headings | the same `G-<item>` elements, keyed by the template's `[Gate key: …]` markers |
| pre-Evidence-Record verification headings | the alias table maps them onto Critical Assumptions |
| `Profile` / `Seam Lineage` / `Load-Bearing Decisions` absent | a Required section the record does not carry: a conformance finding, never blocking |

Each of those is a **reformat in the waiting**: the tolerance holds the
record readable and its elements citable until the record is migrated,
and it retires when the corpus is uniform — not before. The alias table
is load-bearing exactly this way; see §Tolerant matching.

## Canonical vs. observed vocabularies

Each closed vocabulary carries **two** tiers, and the split is doctrine
rather than convenience.

- **Canonical** — what a NEW record may write. Straight from `TEMPLATE.md`
  and `README.md`.
- **ObservedAccepted** — additionally present in the frozen corpus, and
  legitimate practice the template simply never listed. The reader accepts
  it with no warning, and a gate or lint consumer treats it as valid.

The tier exists because the corpus was frozen under the template that
wrote it, and a reader that called those values off-vocabulary would have
been unfixably wrong. Structural migration (§Identifiers) ends that: once
the corpus is on one schema the tier retires, like the epoch tables
before it.

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
against the template, and report **how** it matched. Only `MatchUnknown` is a
finding; everything else is a legitimate way a conformant record of some
records write a template-drawn thing.

| kind | meaning |
| --- | --- |
| `exact` | name and level both match |
| `level-variant` | same section, written at another level |
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

## A field, not a regex

`status.open_joint_decisions` names the joint decisions a record is
still waiting on, and only those. It is present on every lifecycle
status, empty list included, so "none open" is distinguishable from
"this does not apply".

It exists because the qualifier is prose, and prose lies to a regex. A
record that has FINISHED answering its joint decisions writes them down:

    Final [all joint decisions answered — JDR 0001 §D8 (§JD-9), §D9 (§JD-19), …]

Scraping `JD-\d+` out of that reports five open decisions on a record
with none. Only the routing form means one is open:

    Final [joint decision → JDR 0001 §JD-18: conforming-view enforcer]

The projector already knew the difference — `status.form` is
`joint-decision` for the second and `bracketed` for the first — but a
caller had to know to check it, and a caller in a hurry greps the text.
On the corpus that mistake was twelve false positives across three
records, each one a record reported as blocked when it was finished.

The general rule this stands for: **when a consumer has to parse a
projected string, the projection is missing a field.** The tool holds
the grammar; re-deriving it at the call site is where the flow's own
doctrine — a skipped check must never read as a passed one — quietly
inverts into a passed check reading as a failed one.

## The seam binds itself

Every var this binary reads is written in a marker file the flow already
maintains, so it reads the marker rather than waiting to be told.

    cd anywhere/in/the/project
    rdr status                     # no --records, no exports, no seam bound

From the working directory it walks up for the project root, applies the
flow's own nearest-marker-wins rule (a repo-local `.rdr/workspace` beats
the shared `../.rdr-workspace`), sources the marker with `sh` — markers
are plain assignments that expand `$WS`/`$PROJECT` internally, so they
need a real shell, not a regex — and reads `RDR_RECORDS`,
`RDR_SOURCE_REPO`, `RDR_EVIDENCE` and `RDR_HOME` back out. In a git
worktree, `.git` is a FILE rather than a directory; it is followed (no
`git` process spawned) to the main checkout for the marker lookup, but a
repo-local marker's records bind to the worktree being edited, never the
main checkout beside it. A marker written before that distinction existed
refuses with `stopped:marker-binds-main-checkout` rather than silently
handing back the wrong tree.

The binding order is **flag, then environment, then marker**. A flag is
someone spelling out a path; an exported var is a decision someone made;
the marker is only what fills the gap that would otherwise be an error.
No marker binds nothing at all — discovery, never invention — and the
caller fails exactly as it did before. `--records` and `--repo` bind
`$RDR_RECORDS` and `$RDR_SOURCE_REPO` for every other reader of those
vars too — the fact table's roots (§Facts) included — so naming a
directory on the command line moves the whole seam, not just the record
lookup.

This exists because shell state dies between agent tool calls. A skill
that needed `$RDR_RECORDS` had to re-run a fifteen-line resolver or carry
an `export RDR_HOME=… RDR_RECORDS=… RDR_EVIDENCE=…` prefix on every
invocation — bytes on each call, a fresh chance to get a path wrong, and
neither carrying anything the marker did not already hold. `rdr-doctor`
check 11c watches that this keeps working, with the environment cleared
so an exported var cannot mask a broken bind.

`$RDR_EVIDENCE` and `$RDR_HOME` joined that list when the fact table
landed (§Facts). Neither is a records path: the first roots the exact-path
probes a fact declares, the second is where the fact table itself lives.
`rdr env` publishes every seam var the marker set, including the three this
tool never opens — `$RDR_ENV`, `$RDR_RESOURCES`, `$RDR_AUTOCOMMIT` — which
were the only reason `§seam-bind` still carried a shell resolver.

It is the one seam read that does **not** let the environment win. A flag
names one directory for one call, so `--records` outranks everything; but
re-publishing an inherited `RDR_*` would let a leak survive the `eval` and
outlive the turn that made it, where sourcing the marker overwrote it. In a
workspace whose repos bind disjoint seams — one repo-local marker beside a
shared one — that leak reads records from one project against another's
evidence, and every lens fact comes back false with nothing to show it. So
`env` answers from the marker, and reports `$RDR_MARKER` / `$RDR_PROJECT`
for the caller's own guard.

## Paths

`rdr paths` answers the two questions six skill and prompt sites used to
build by hand: where does this lens write, and which iteration is next.

It exists because a restatement is a copy that can go stale alone, and two
of those six did — they named `<lens>/<slug>/` after the migration moved it
to `<slug>/evidence/<lens>/`. That failure is silent by construction: a lens
directory that cannot exist reads exactly like a lens that never ran.

So it answers from the SAME declarations the fact evaluator reads —
`models/rdr-facts.toml`'s roots plus its `[iteration]` block — and one source
cannot disagree with itself. The convention is data rather than a compiled-in
regex for the reason the roots are: this binary is pointed at more than one
seam, and the reference workspace alone binds two disjoint ones — a shared
marker whose records and evidence sit in different repos, and a repo-local
marker where `RDR_EVIDENCE = RDR_RECORDS` collapses them under one
`<slug>/`. Both resolve correctly with no branch, and a third tree is a TOML
edit.

`--next-iter` LISTS the directory; it never matches a guessed name, which is
the glob the probe rule bans. Loose files are iteration 1 (§evidence), so a
first pass writes the base itself and no `iter-1` is ever invented. Next is
1 + the HIGHEST segment found, never the lowest absent: the reference corpus
really holds `iter-3` with no `iter-2`, and reusing that number would file a
later report under an earlier one. The segments found travel with the answer,
with a note when they are not contiguous, so a gap stays visible.

An unbound root is a stated absence and exit 1, never a path rooted at `/`
that a caller might `mkdir -p`, and an unfilled `{key}` is refused rather
than collapsed to the parent of every cluster. It creates no directory and
writes nothing.

Two more values ride `--next-iter` so a loop never counts for itself.
`PRIOR_DIR` is where the last pass wrote — the highest segment, or the base
when only loose files are there — so "diff this pass against the ledger"
reads one variable. `ITER_BUCKET` is ITER against the tree's declared `cap`
(`[iteration.tree.<name>] cap` in the fact table): `"1".."cap"` verbatim,
`"over"` past it. It is the `iter` tag `models/rdr-loop.toml` routes on, so
the cap a stage doc states and the one the loop enforces are one number in
one file; a tree that declares no cap gets no bucket.

## Anchors

`rdr anchors --record NNNN FILE...` prints, sorted and unique, the element
ids the files cite that the record's projection mints — the same outline,
element and anchor ids `inspect` lists. Findings ledgers anchor rows to
those ids, so reconciling a re-run against its origin ledger is `comm -13`
(net-new) and `comm -12` (still-open) over two of these, and the 3amigo
hotspots are `sort | uniq -c` over three. The prompts used to say "reconcile
by passage anchor" and "set intersection on those ids" in prose, which a
model then performed by re-reading both files.

Membership is exact: a token shaped like an id that names nothing minted
contributes nothing, and a peer record's id is a citation, not an anchor.
`--unresolved` prints those shaped-but-unminted tokens instead, which is
what a ledger row points at after a reword. Output is byte-ordered so two
outputs `comm` without a locale in the way (`LC_ALL=C comm` in the prompts
makes that explicit).

## Impact

`rdr impact <record> [--literal TOKEN]...` predicts which predecessor
tests a locked record's contract changes will turn red, so Stage 8's
implementer meets the list up front instead of one red test at a time
inside its loop. The launch precheck proves the baseline green, so every
predecessor test that goes red is this change's doing; the per-test call
— regression to fix, or a contract the record retires — was made mid-loop
with no list, and Phase 0 now writes `<art>/impact.md` from this verb's
stdout (`prompts/implementation/launch.md`).

It predicts by FILE from two sources: the records the RDR states it
overrides or succeeds — its own Metadata edges, read the way `status`
reads a record and never resolved — and the literals the change retires,
which the caller names with `--literal` because deciding what a REQ
retires is judgement. A test file is predicted when a test in it pins a
set member by name or its body carries a literal; every test in a
predicted file is then a row (arm: `record:NNNN` and/or `literal:<tok>`),
grouped by family. How a test's name pins a record, and which files are
tests, is a convention of the SOURCE repo and lives in
`models/rdr-impact.toml` — `detect` file, `glob`, three regexes — chosen
by whichever convention's detect file the repo holds; a second language
is a table there, not a branch here.

Only files the glob names are opened, each once, under the resolver's
own skip-dirs and size cap; a source file that carries a literal is not a
test and is never read (the read count is what the test pins). A
directory holding its own `.git` (a nested clone or worktree) is a
border the walk does not cross, so a consumer repo that carries
worktrees is counted once. An unbound
`--repo` or a repo no convention detects is a `stopped:` line, never
`rows: 0` — a tree nothing looked at must not read as a tree with no
impact. It never writes; the caller redirects stdout to `impact.md`.

## Facts

`models/rdr-facts.toml` declares the signals `rdr-status` reads, and the
binary evaluates them. It exists because those signals were prose in a
skill: a table of directory shapes and projection paths that a model
re-derived every run, describing a tree nothing checked it against. The
corpus already holds the predictable result — a path documented in two
skills that exists nowhere in the evidence tree.

A fact is a **probe** or a **field**.

A probe asks whether an exact path exists, under one of two roots: the
per-RDR evidence tree (`$RDR_EVIDENCE/<slug>/evidence/`) or the record's
artifact folder (`$RDR_RECORDS/<slug>/artifacts/`, with the flat `<slug>/`
read second, per path, while pre-move records remain). It names ONE path. No globs, no
patterns, no first match — a load error, not a path that quietly matches
nothing. The reason is the one this whole tool is built on: a lens that
ran, read as un-run, sends a consumer to redo work that is already done,
and that is the same class of error as a skipped check reading as a
passed one.

A field is a projection path. The projector has already split a Status
from its qualifier, named the qualifier's grammar, normalised an
assumption's `**Verified**` and `REFUTED (…)` onto a vocabulary label and
placed it in a tier. A fact reads what it published. Where a fact would
otherwise have to parse a rendered string — the `Profile` value carries a
rationale tail, and §lens-row says match the leading word — that is the
projection missing a field, so the fact carries the word and a second
carries the sentence.

**Absent is not false**, exactly as with `resolved`. A probe whose root is
bound answers true or false: the tool looked. A probe whose root is
UNBOUND is omitted from the output entirely, because "no evidence root is
configured" and "the lens did not run" are different answers and only one
of them should route. This matters more than it looks: the consuming
navigator is a three-valued kernel, where an omitted key leaves a rule
undecided and `false` decides it.

The fact NAMES are a contract. `models/rdr-status.toml` matches on them,
neither binary calls the other, and the skill composes the two in one
call — so a rename is a breaking change to a file in another repo. Every
fact declares one of the kinds that side accepts (`enum`, `bool`, `int`,
`set`, `scalar`) and every value crosses as a string. A `scalar` may also
declare `prose = true`, which says its value is free text an author wrote
rather than a token — see §status for what that changes and why the
declaration lives in the table.

One source is neither a probe nor a field. **Stage 7.1's output is keyed
by the CLUSTER**, not by the slug — `cluster-reconcile/0122-0123-0130-0131-0132/`
— because one run reconciles a set and writes one directory for all of
it. In the current shape the key is the members' numbers joined, so the
key IS the membership, and `cluster-member` reads that directory rather
than naming a path. That is not the guess the probe rule bans: nothing is
predicted, the tree is asked what it holds. The rejected alternative was a
table of hand-authored keys, which makes a derived name a maintained one.

Overlap there is not ambiguity — where a re-run widened a cluster both
directories persist, one being the earlier iteration of the other, and
both answer the only question asked: 7.1 ran.

Two things the table deliberately does NOT declare, both recorded in it:

- **The topical cluster epoch.** Before 2026-06-29 a 7.1 directory was
  named for its subject, not its members — `dml-purpose`,
  `final-cluster-2026-05-28`. Those are excluded by SHAPE rather than read
  and filtered, because `final-cluster-2026-06-22` contains the four-digit
  run `2026` and a rule that pulled numbers out of a name would mint a
  membership claim for a record 2026 that no run ever made. Their real
  membership survives only in prose, and whether they are even the same
  object is open: `final-cluster-2026-05-28` has no whole-set critique and
  no pairwise scan, and predates the stage file by three weeks.
- **A legacy-shape probe that changes no routing.** The corpus migration
  moved directory-shaped evidence under each record and left file-shaped
  evidence where it was, so pre-migration output still sits at
  `3amigo/<slug>.md` and `critique/<slug>-critique.md`. Every such record
  is terminal, so nothing routes on it today — but a claim about the
  corpus that nothing checks is a claim that quietly stops being true, and
  the failure it guards is precisely a lens that ran reading as un-run.

The table is found at `$RDR_HOME/models/rdr-facts.toml`, or beside the
binary (`$RDR_HOME/bin/rdr` → `../models/`) when no marker is bound.
Reading it needs a TOML parser and the stdlib has none, so `toml.go` reads
the subset the table uses — table headers, string/int/bool values, lists,
comments. Every line outside that subset is REFUSED with its line number.
A parser that skips what it does not understand turns a typo into a
missing fact, and a missing fact reads as absent when it was only
misspelled.

## status — the navigator's read, in one call

`rdr status NNNN` evaluates the fact table over one record. It is the
verb the table was written for: before it, answering "where is this
record and what runs next" cost an `inspect --json --filter`, a
`--select §decision-rationale` byte read, an `ls` per lens folder, a
`status.md` read, and then a model re-deriving a prose signal table over
the results. Each of those is a TURN, which re-sends the conversation.

Three renderings of ONE evaluation:

    rdr status 0055                # one fact per line — the cheap human read
    rdr status --json 0055         # the neutral vector (§Facts)
    rdr status --tags 0055         # `--tag k=v` argv for a resolver
    rdr status --checklist 0055    # the stage checklist, three-valued (`?` = nothing looked)
    rdr status 0122 0123 0130      # a NAMED SET — one row per record
    rdr status                     # the Draft+Final worklist, each row with its facts
    rdr status --argv              # the worklist + Deferred, one tab-separated argv line each (bin/rdr-next)

### Three arities, two costs

Naming several records answers the question two stages actually ask.
Stage 8's predecessor precheck (`prompts/implementation/launch.md`) and
Stage 7.1's Final-and-unimplemented filter each hold a LIST of records,
and both used to construct N paths and read N `status.md` headers by
hand — the choreography this verb exists to end, just at a different
scale.

A named set resolves each argument BY NAME, so it reads only those files
and never enumerates the records dir. That is the whole reason it is an
arity rather than a corpus facet: on the reference corpus one record is
47ms and the worklist scan is 2.0s, so a set that fell through to a scan
would be a silent 40× regression on a call Stage 8 makes every run.

`--tags` stays single-record — it renders ONE resolver's argv, and a set
has no record to name — and refuses with the worklist's own words.

**An unresolvable argument is a `skipped[]` row, not a refusal.** This is
the three-valued discipline (§Facts) carried up to the set: a predecessor
whose record or capsule is missing must read as "nothing looked", and a
caller has to tell that from "looked, not COMPLETE" — Stage 8 halts on
the second and reports the first. Refusing the whole call would collapse
them, since a set that answers nothing says nothing about any member.

### `--filter` — why a set is affordable to read

The vector is 48 facts, about 7KB of JSON per record, so a five-member
cluster costs ~44KB of a caller's context to answer one word per record.
`--filter` keeps only the facts named, and drops the row's summary with
them — `Profile` alone carries the field's whole rationale tail, which is
most of the payload. Measured on that cluster: 43,888 bytes to 1,569, a
28× reduction.

    rdr status --json --filter impl_state 0122 0123 0130 0131 0132

Identity survives filtering (`record`, `path`), because a row a caller
cannot attribute to a record is not an answer. A name the table does not
declare is REFUSED with the list of what could have been asked for —
the same rule `--filter` follows on `inspect` and `index`, because a
filter that silently answers nothing reads as "the fact is absent", which
is a claim about the record rather than about the request. Filtering
happens AFTER evaluation: it is a question about the output, never an
instruction to look at less.

With no argument it is the worklist, and it absorbed `index --in-flight`,
which answered the same question without the facts. That is also 16×
cheaper: the old facet ran the full edge resolver it never used (13.8s on
the 144-record reference corpus; the worklist is 0.86s WITH the facts).
`index --status`, which groups every record, stayed on `index` — that is
a question about the corpus, not about what to do next.

It never writes. Not the records, not the evidence, and not a resolver's
owned-state artifact: that file's format belongs to the other side of the
seam, and a navigator that writes is no longer derivable-from-disk.

### Why `--tags` will not render prose

The composition this exists for is one Bash call:

    rdr status --tags NNNN >/dev/null || exit 2    # a refusal splatted into argv is lost
    intrastate flow resolve --model "$RDR_HOME/models/rdr-status.toml" \
      $(rdr status --tags NNNN)

An **unquoted** `$(…)` splits its output on IFS whitespace and then globs
the words. It does not split on lines, and quotes inside the output are
literal characters rather than syntax. So a value carrying a space does
not arrive as one argument — it arrives as several, and the first of them
is `k=<head>`: a well-formed tag with a silently truncated value. The
resolver refuses the leftover words *usually*; a tail that happens to
parse would be accepted, and the caller would route on words the record
never said.

The glob half is not theoretical either. Under `sh` and `bash`, with
files `k=a` and `k=b` present, the word `k=[ab]` expands to TWO arguments,
`k=a` and `k=b` — the value replaced by a filename.

Every routing fact is safe by construction: an enum, a bool, an int, and
a set rendered as a COMPACT JSON array (`["0131","0132"]`, no space after
the comma) are each exactly one shell word. The exception is prose. On
the reference corpus 91 records of 144 carry a `Profile` rationale tail —
`mid — one contract plus the metrics surface` — which shatters into eight
words.

So a fact the table declares `prose = true` is not rendered as a tag at
all, and is read through `--json`, where it is a JSON string and nothing
splits it. That is a DECLARATION, not a discovery: which facts carry
prose is a property of the fact, and deciding it from the value in hand
would make `--tags` succeed on one record and fail on the next. The table
already draws the same line one level in — the Status QUALIFIER is prose
and is deliberately not a fact, while its `form` is.

Omitting a prose fact is safe here, and only here, because the omission
is declared rather than conditional: an omitted key is load-bearing on
the other side (that kernel is three-valued, and absent leaves a rule
undecided), but no routing rule can match on prose anyway, so a resolver
never had it to lose. A non-prose fact that still would not survive means
the table is wrong about itself, and that is a `stopped:unsafe-tag`
refusal naming the fact — never a silent skip.

| test | what it pins |
| --- | --- |
| `TestStatusGolden` | every fixture's whole fact vector, over a synthetic corpus — so it fails when the evaluator changes, not when a record does |
| `TestStatusTagsRenderShellSafeArgv` | every rendered word is ONE shell word, asserted directly rather than trusting the renderer |
| `TestStatusTagsOmitProseAndJSONKeepsIt` | the prose fact is absent from the tags AND still reachable in `--json` — omitting it is only correct because it is not lost |
| `TestUnsafeTagWordCatchesWhatTheShellWouldRewrite` | the rule itself, including the set renderings it must NOT refuse |
| `TestStatusWorklistIsTheInFlightSet` | Draft and Final, never terminal, never parked — and the facts travel with the row |

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

The corpus's own citation spelling is a name too: `cli/0055` is record
0055 when the records dir is called `cli` — the prefix is read off the
bound dir, never spelled in Go, so another dir's prefix still names
another dir — and `cli/0055:C4` or `0055:C4` as the positional selects
that element. Lint prints citations in exactly this form, so a finding's
id pastes back as a working call.

**`--select` repeats, and answers in order.** `--select A20 --select A21`
used to return A21 alone with no word said (Go's flag package keeps the
last value), and sessions fell back to one call per element — 64 selects
in one refine pass. Now each select is answered as it would be alone:
text element selects print their bytes in sequence, JSON is an array of
the single forms with a named facet wrapped as `{"select": name, name:
value}`. A select that names nothing is still `stopped:no-such-element`
for the whole call, never a partial answer.

**A named set is one call.** `inspect 0097 0108 0110` is each record's
projection in the order given, resolved by name so only those files are
read — the arity `status` has, for the same caller: 7.1's critique agent
held a list of members and tried it twice before falling back. Text is
each record's own rendering in turn (the summary heads itself with the
number; element bytes print as the single form prints them); `--json` is
`{records: [{record, path, value}], skipped: []}`, the form to read when
the selects span records. A member that does not resolve is a `skipped`
row with its reason, not a refusal of the whole call — the discipline
§status states. A citation per member (`inspect cli/0112:A3 cli/0113:A8`)
reads several records' elements in one call. The usage log names the
arity (`records:text`, `records:select:element`) and every select class
(`select:element,element`), so a chain replaced by one call is measurable.

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

Bytes are not the only cost. Deciding `resolved` is the one thing
`inspect` does beyond reading the record — reading the records its edges
name, and a walk of `--repo` — and only `edges[]` can show the verdict.
So resolution runs only when the projection can carry it: the whole
envelope, `--select edges`, or a `--filter` naming `edges`. Every other
facet, including `--select <id>` and the text summary, skips it
(`showsEdges`) and costs ~35ms.

**The corpus a single record resolves against is its EDGE TARGETS, not
the directory.** A verdict is decided by the target's own projection, so
every record the document does not name was parsed and discarded.
Reading the whole dir to answer a question about 15 of its 144 records
cost 1.85s of a 2.10s call — the largest single cost in the tool, paid by
every `lint` and every default `inspect --json`. Narrowing it is
byte-identical by construction, `resolved` verdicts included, and takes
`inspect --json` from 2.09s to 0.56s over the reference corpus.

What remains is the repo walk (~0.22s), and it is the floor: a single
record resolves through `ResolveAll`, not `Resolve`, because priming the
symbol cache is what makes that walk a single pass. The per-symbol path
walked the tree once per distinct symbol cited — 1.45s where one primed
walk is 0.56s, for identical verdicts. The corpus path and the
single-record path pay the same walk once each; neither pays it per
citation.

Because the narrowed set can legitimately be EMPTY — a record every one
of whose targets is missing — the resolver is told that a records dir was
read (`NewResolverOver`) rather than inferring it from a non-empty map.
Absent means nothing looked; a target the dir does not hold is `false`.
Conflating the two would turn every dangling reference into an unchecked
one: a skipped check reading as a pass.
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

Turn it on and every invocation appends one line recording what was
asked and what it cost. This exists because the consumer-integration
pass had to argue the tool's value from *estimated* byte counts: nothing
recorded what the binary was actually asked for or how much it emitted.
Now the numbers accumulate over ordinary use — no dashboard, no session
instrumentation, no second tool to run.

`RDR_USAGE_LOG` is a seam var like any other — environment first, then
the marker — so a project opts in **once**, at `/rdr-init --usage-log`,
and no call site has to know a path:

| value | |
| --- | --- |
| unset, or `off`/`false`/`0`/`no` | no log — the default |
| `true`/`on`/`1`/`yes` | beside the marker that said so |
| a path | that file |

**Beside the marker** follows the seam's own scope instead of assuming
one. A repo-local marker sits at `$PROJECT/.rdr/workspace`, so the log
joins it in that already self-ignoring directory — the counterpart of
the sibling codebase's `$REPO/.retrofit/`. A workspace marker sits at
`$WS/.rdr-workspace`, above sibling repos that deliberately have no
`.rdr/`; the log goes there, at `$WS/usage.jsonl`, inside no repo and
needing no ignore rule.

Creating a `.rdr/` for the log would be wrong twice over: it invents
seam structure a workspace consumer opted out of, and it scatters one
shared setting into per-repo files nobody ignored. One marker, one log.

It is beside the project on purpose, not in a user-wide state directory:
the log is about *these* records and means nothing away from them. A
truthy setting with no marker to anchor to writes nothing rather than
inventing a location.

    rdr inspect --select 0142:C1 0142        # with the marker's RDR_USAGE_LOG="true"

    {"bytes_out":490,"cmd":"inspect","elapsed_ms":551,"exit":0,"facet":"select:element","target":"0142","ts":"2026-08-24T20:38:11-07:00"}

A filtered envelope logs as `"facet":"json:metadata,counts"`, so the
log can say what `--filter` saved rather than spelling both `json`.

That line is the whole argument for the tool, measured rather than
estimated: quoting one contract costs **490 bytes** where reading the
record costs **156,350**. It also keeps the tool honest in the other
direction — `inspect --json` on the same record emits *more* than the
file does. The saving is in `--select` and the index facets, never in
the full envelope, and the log says so.

## Open joint decisions

    rdr index --open-joint          # in-flight records; --all for every record; --json for the rows

One row per open joint decision, from either place a record states one: a
body `Joint-check: … (home: OPEN)` line (`signal: joint-check`, with the
`JC` element id, line, targets and home) or a Status line in joint-decision
form (`signal: status`, with the qualifier). 7.1's first question, answered
without opening a member body or grepping one — a grep for `home: OPEN`
once lost the only open line to `| head` and reported none.

## Cycles

    rdr index --cycles              # --json for the rows

Four shapes the flow cannot make progress through, and nothing from the
relations that are symmetric by design (`cluster`, `peer-evidence`,
`cross-cutting-owner`, `mentions` — a cycle there is the corpus working):

| finding | over | what it means |
| --- | --- | --- |
| `ownership-cycle` | `predecessor` ∪ `overrides` ∪ `moved-to` | two records each claim to stand in for the other; no authority is named. `lint --locking` blocks on the pair the record is half of (`ownership:mutual`) |
| `home-cycle` | Joint-check `home` → **Draft** record | deferrals wait on each other — a deadlock until one arc is broken. A home on a Final record is a ruling that exists, never an arc |
| `home-ahead-of-lock` | one Final record | its Joint-check home is a Draft record; a ruling change there strands the lock |
| `open-at-lock` | one Final record | it locked carrying an OPEN Joint-check |

Strongly connected components (Tarjan), stable order, so the same corpus
prints the same report. Found on first run: one mutual `overrides`
(0106↔0112), three mutual `predecessor` pairs, no live home cycle, and
four Final records whose homes sit on the two Drafts still in flight.

## §receipt — was it linted since it was last written?

    rdr receipt 0143          # 0: prints the lint's log line; 1: stopped:no-lint-receipt; 2: no log bound

The log's first real audit found a record carried through four stages in
a day with two `inspect` calls and no `lint`, every gate closed. The
skills say "lint at stage exit"; nothing checked. `receipt` is the check:
the newest `lint` line whose target covers the record (its number in any
spelling, its path, or a whole-dir lint) at or after the record's mtime
(`--since` overrides). A lint that exited 1 still counts — it ran. §commit
in `rdr-commit.sh` runs it for every `NNNN-*.md` path it is handed and
refuses the commit on 1; on 2 it proceeds with a note, because a project
that never turned the log on did not opt into this either. That makes the
commit the choke point: a gate closed without lint is caught where the
history is written, not read about later. `rdr-doctor` 11d WARNs when the
log is off while autocommit is on — the one configuration where the refusal
is silently unenforced.

| field | |
| --- | --- |
| `ts` | RFC3339 with offset |
| `cmd` | `inspect` \| `index` \| `lint` |
| `facet` | which query ran (`json`, `select:element`, `status:tags`, `locking`, …) |
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
      env.go               `rdr env`: publishes the bound seam, marker-authoritative
      paths.go             `rdr paths`: the evidence dir and the iteration, from the same table the facts read
      status.go            the navigator's read: facts evaluated, rendered three ways
      impact.go            `rdr impact`: the predecessor tests a record's changes will turn red, by the convention models/rdr-impact.toml declares
      corpus.go            the corpus facets: graph, status, backlinks-to, anchor intersection, README drift
      internal/ident/      the element ID grammar, slugs, content hash
      internal/edge/       the typed relation model: kinds and reference grammars
      internal/scan/       the line scanner: outline, elements, warnings
        fields.go          the labelled-bullet pass: metadata, element and section fields; coverage
        edges.go           the typed-edge pass: every stated relation, by kind
        resolve.go         resolution against the records dir and the repo; backlinks; cluster derivation
        corpus.go          corpus queries: record summaries, the graph document, anchor intersection, README drift
      internal/toml/       the TOML subset the data tables use; refuses, never skips
      internal/model/      the template model, read from TEMPLATE.md at startup
        parse.go           the reader: heading tree, class and gate markers, vocabularies
        schema.go          Load and Bind: the schema this process reads records against
        sidecar.go         models/rdr-template.toml: element kinds, observed tiers, lifecycle
        template.go        section and class types, markers, the value-continuation rule
        fields.go          per-section label sets, prefix matching, the Evidence Record Status
        vocabulary.go      the four closed vocabularies; Method/Type/Profile parsing
        qualifier.go       the status qualifier grammars
        template_table.go  the loaded table: sections, fields, vocabularies
        alias.go           legacy names and the match kinds
        scaffold.go        filled-in template scaffolds
      testdata/            synthetic fixtures spanning the corpus shapes, plus variants/ (see its README)
