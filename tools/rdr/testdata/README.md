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

One fixture per template epoch:

| file | epoch | exercises |
| --- | --- | --- |
| `epoch-a.md` | A | no Profile or Seam Lineage, no Evidence Records, gate responses inlined; the legacy `#### API Verification` and `#### Dependency Source Verification` headings |
| `epoch-b.md` | B | Profile, Seam Lineage, Load-Bearing Decisions; Evidence Records; gate still inlined; a compound Method and the legacy `**Related**` field |
| `epoch-c.md` | C | the gate.md pointer replacing the inlined body; a `Transient` contract marker and a `normative` fence; the legacy `### Premortem` heading |
| `epoch-d.md` | D | Critical Assumptions at `##`; a joint-decision status qualifier; a wrapped metadata value; a filled-in `Alternative 2` scaffold |

Each fixture carries its epoch's fingerprint, so `DetectEpoch` places it,
and at least one alias or qualifier form, so the tolerant-reading paths are
exercised rather than only the happy path.
