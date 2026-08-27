# Fixtures

Every file here is **synthetic**: hand-written for this repo, describing an
invented subject (a frame serialization format, a cache eviction policy).
None of it is drawn from any real record.

That is a hard rule, not a preference. This repo is public and
consumer-agnostic, while the records the model was built against live in a
private repo. A fixture copied from a real record would publish that
record's content — its paths, its identifiers, its design — through the
back door of a test file. So the fixtures reproduce the template's SHAPE,
which is what the model reads, and invent the content that fills it.

Four fixtures spanning the shapes the corpus contains. Each is named for
the SHAPE it exercises. They were once `epoch-a` … `epoch-d`, one per
template epoch, back when the reader carried four template tables; the
epochs are gone and the names went with them, because a fixture named for
a generation says nothing about what it tests. All four are read against
the ONE current template, which is the point: an old shape must still
classify clean through level-variance, the alias table and the scaffold
patterns.

| file | shape | exercises |
| --- | --- | --- |
| `legacy-shape.md` | oldest | no Profile or Seam Lineage, no Evidence Records, gate responses inlined; the legacy `#### Dependency Source Verification` heading |
| `assumptions-nested.md` | + apparatus | Profile, Seam Lineage, Load-Bearing Decisions; Evidence Records; gate still inlined; a compound Method and the legacy `**Related**` field |
| `gate-inline.md` | + gate pointer | the gate.md pointer replacing the inlined body; a `Transient` contract marker and a `normative` fence; the recognised-but-unmapped `### Premortem` heading |
| `current-shape.md` | current | Critical Assumptions at `##`; a joint-decision status qualifier; a wrapped metadata value; a filled-in `Alternative 2` scaffold |

Each fixture carries at least one alias or qualifier form, so the
tolerant-reading paths are exercised rather than only the happy path.
`TestFixturesKeepTheirShapeSignals` pins the signals that make each
fixture its shape, so an edit that flattens the set fails loudly.

`variants/` holds one fixture per known parser failure — the ways the
corpus departs from the template that an exact-literal parser silently
undercounts (README §The resilience contract). Each is an otherwise
conformant record whose tally the scanner tests pin by hand.

| file | exercises |
| --- | --- |
| `heading-level.md` | Critical Assumptions at `###`; the joint-decision Status form |
| `label-variants.md` | `**Status:**` with the colon inside the bold; `Evidence — …`, `If wrong (…)`, `Evidence (plan)` prefix labels; an author's `Note` inside an assumption; a glossed and a compound Method |
| `status-parenthetical.md` | `Implemented (\`main\` sha)`; Evidence Record statuses as `**Verified**`, `REFUTED (…)`, `Verified as narrowed — …`, `Resolved — …`, and the template legend left unfilled |
| `wrapped-metadata.md` | the em-dash `Deferred — …` Status paragraph with a guidance comment under it; Predecessors, Overrides and Seam Lineage wrapped, with a nested labelled bullet; the legacy `Related`, the recognised `Release scope`, and an unknown `Referenced by` |
| `author-structure.md` | author sub-headings under Approach; a foreign `## Appendix` with sub-headings; prose-named labels in Consequences, Failure Modes and Cross-Cutting Concerns; a `Risk` under a Step; `Rejected (…)` |
| `addressable-text.md` | bold paragraph leads a `§` citation can name, and mid-paragraph emphasis that is not one; a `### Load-Bearing Decisions` heading with author-numbered `**D1**` / `**D6**` bullets |

`lint/` is a five-record corpus for the linking rules — the fixtures are
read together, because a citation is only resolvable against the record it
names. Each is dated after the labelling rule landed.

| file | exercises |
| --- | --- |
| `0010-frame-header-labelled.md` | the conforming case: `**C1**` / `**C2**` labels, a Peer-RDR Evidence citing `0011-…:C1` by element ID. Lints clean at a lock gate |
| `0011-frame-header-unlabelled.md` | two unlabelled contracts on a post-rule record — advisory when grandfathered by date, blocking at lock when not |
| `0012-frame-header-dangling.md` | a TERMINAL record citing a record that does not exist (`0099`) and an element that does not (`0010:C9`): both come back as non-blocking fix pointers with line ranges |
| `0013-frame-header-order-v2.md` | ownership transfer: overrides `0010:C1` BY ID while leaving `0010:C2` and the whole of 0010 untouched, so the backlink query is the review worklist |
| `0014-frame-header-debug-flag.md` | `Demoted [→ 0013-…]`, for the `moved-to` edge — and a terminal status that is not `Implemented` |
