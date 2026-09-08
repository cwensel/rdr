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

`touched/hunks.diff` is a synthetic unified diff read against
`current-shape.md`'s element table by `inspect --touched-since`'s
goldens — an interior edit, a pure deletion and a blank-length add —
so the overlap rule is pinned without a git repository in the tree.

`status/` is a ten-record corpus for the fact table and `rdr status`,
and it is the only fixture tree that ships more than records: the facts
are exact-path PROBES, so the paths have to exist. `records/` holds the
records and each one's artifact folder; `evidence/` is a synthetic
`$RDR_EVIDENCE`. `facts.golden` pins every record's whole fact vector —
regenerate it with `UPDATE_GOLDEN=1 go test -run TestStatusGolden`, and
read the diff before you do.

A golden over the LIVE corpus was the alternative and is the wrong one:
it would fail on every record edit, cannot run where the consumer repo is
absent, and would publish that repo's content into this public one.

| file | shape | exercises |
| --- | --- | --- |
| `0020-cache-eviction-policy.md` | front-half Draft | CAs all `Pending` (`ca=all-pending`, Refine is the open `~`); a `Profile` with a rationale tail — the PROSE case `--tags` refuses to render; a `Seam Lineage` at `2nd point-fix` escaped by a nested-bullet `Accretion disposition:` (`seam_lineage=2+`, `accretion_disposition=true` — the floor's escaped cell) |
| `0021-cache-warmup-order.md` | foundational, mid-row | `cove` and `3amigo` done, `critique` single-model only, so the dual-model diff is still owed; a labelled `**C1**`; a joint-decision Status; a declared `Cluster`; `artifacts/gate.md` written; a `Seam Lineage` at `3rd point-fix` with no disposition — the floor holds and Profile is at it |
| `0022-cache-metrics-surface.md` | scoped re-entry | `Draft [revised from Final …; re-verify …]`; a `Refuted` assumption beside a `Verified` one; a Stage-9 `artifacts/status.md` capsule header stating `IN-PROGRESS` (the current layout); `Seam Lineage: no prior accretion` (`seam_lineage=0`) |
| `0023-cache-shard-count.md` | terminal, legacy evidence | `Implemented`, with its 3amigo output still in the pre-migration FILE shape (`evidence/3amigo/<slug>.md`) — the warning probe's case, without which the record reads as never lensed |
| `0024-cache-persistence.md` | parked | `Deferred [revisit when …]` — neither in flight nor terminal; a prose Normative Contracts closed by `Determinacy: n/a — …` (`determinacy=na`, the mid record that owes no lite run) |
| `0026-cache-hash-identity.md` | foundational, complete signals | critique on two base models plus the diff; run-1 stamped `variant: full`; prose contracts (`contracts=0`, `contracts_prose=true`) closed by a bold `**Determinacy:** fired — …` line (`determinacy=fired`, exercised without changing a foundational route). 0020 carries no line and is the `unjudged` sentinel |
| `0027-cache-tier-labels.md` | legacy artifact layout | capsule and `req-list.md` flat under `<slug>/`, `gate.md` under `<slug>/artifacts/` — the pre-move shape; one vector reads `gate_written=true` AND `impl_capsule=true`, so the legacy leg is per path, not per folder |
| `0028-cache-flush-hook.md` | both layouts | a `COMPLETE` capsule and a 1-REQ ledger under `artifacts/` beside an `INCOMPLETE` capsule and a 12-REQ ledger flat — reads `COMPLETE` and `0-10`: canonical wins |
| `0029-cache-size-report.md` | neither | only `artifacts/gate.md` — `impl_capsule=false`, `impl_state` and the ledger facts absent, `gate_written=true` |
| `0030-cache-warm-ratio.md` | complete, small | the table's one COMPLETE cell and inline-size cell; a `triage.md` ledger with only closed IN-SCOPE rows (`impl_findings_open=0`) |
| `0036-cache-warm-archive.md` | Implemented, pre-convention | no capsule in either layout, `Status: Implemented` — the landing flow's own terminal assertion, read only because no capsule exists to read instead |
| `0037-cache-warm-digest.md` | predecessor reads Implemented | `Predecessors: 0036`; `predecessors_state=complete` off the Status word alone, `predecessors_incomplete` empty |
| `0038-cache-warm-lead.md` | cluster-order lead | `Cluster: 0039`, no Predecessors; 0039 names 0038 in its own Predecessors, so 0038 reads `related_final_unordered=0` — the order is declared |
| `0039-cache-warm-follow.md` | cluster-order follow | `Cluster: 0038`, `Predecessors: 0038`; 0038 has no capsule, so `predecessors_state=incomplete`, and `related_final_unordered=1+` with `cluster_unordered=[0038]` — Stage 8's cluster-order halt |
| `0040-cache-warm-left.md` | cluster-order tie, first by number | `Cluster: 0041`, no Predecessors, no Overrides — no explicit edge either way; same Priority (Medium) as 0041, so the lower number builds first: `related_final_unordered=0` |
| `0041-cache-warm-right.md` | cluster-order tie, second by number | `Cluster: 0040`, no Predecessors, no Overrides; same Priority (Medium) as 0040, so it reads `related_final_unordered=1+` with `cluster_unordered=[0040]` — the symmetric pair that used to deadlock, now ordered by number |
| `0042-cache-warm-late.md` | cluster-order, outranked by Priority | `Cluster: 0043`, Priority Low; 0043 is High with the higher number, and Priority outranks number, so 0043 builds first: `related_final_unordered=1+` with `cluster_unordered=[0043]` |
| `0043-cache-warm-urgent.md` | cluster-order, wins on Priority | `Cluster: 0042`, Priority High; despite the higher number, High outranks 0042's Low, so it builds first: `related_final_unordered=0` |

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
