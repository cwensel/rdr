Seed a new RDR from: {IDEA}. See the rdr README for what an RDR is.

Read {RDR_ENV} for the project's RDR directory — the parent of {ARTIFACT_DIR}
(the RDR file and its artifact dir are siblings: file `<rdr-dir>/NNNN-slug.md`,
artifacts `<rdr-dir>/NNNN-slug/`). If {RDR_ENV} names no RDR directory yet, ask me
before writing.

**Stay on the current branch** — never `git branch`/`switch -c`/`checkout -b` or a
worktree: sessions share this branch, so branching hides the new RDR and breaks the
§rdr-claim number lock.

**Claim the number atomically FIRST** — rdr-common §rdr-claim, before authoring
anything. It reserves the slot and hands you the canonical TEMPLATE.md skeleton.

Fill it IN PLACE: ONLY Metadata, Problem Statement, and Context (plus `[NUMBER]` /
`[TITLE]` in the H1) from the kata. Leave every other section **exactly as the
template ships it** — its placeholders already are the Draft placeholders. Do NOT
invent a solution, assumptions, or research findings; those are later stages.

Fill the **Seam Lineage** Metadata field from the `kata-scope-review
§seam-accretion` emission on the originating kata (the locus `path::Symbol`/
`area:*`, the point-fix count, and the prior-id trail) — copy it verbatim, do
not re-derive (a re-derived count drifts). If the seam has no prior closed
point-fixes, write "no prior accretion".

For the **Profile** Metadata field, write a one-line provisional estimate from
the same design shape you judged to seed this as one RDR (`rdr/stages/README.md`
matrix: one internal contract → small; user-facing or locks a contract → mid;
locks an enum/hash/format/grammar/destructive-op → large; cross-RDR / spans
modules → foundational). **Floor it: if Seam Lineage carries ≥2 prior
point-fixes, the profile is `foundational` regardless** (the accretion floor —
escapable only by a written accretion disposition in Seam Lineage). It is an
early budget signal, provisional because the RDR is `Draft`; Stage 4 (Resolve)
overwrites it from the verified count. Don't labor over it. ≥2 independent
contracts across separate seams → flag a split, don't pick a profile.

State the Problem Statement as a USER OUTCOME first (who runs this, what they
want to accomplish, how they discover they need it), then the system-internal
requirement. Do not solve it here.

If {IDEA} is a plain description rather than a kata id, synthesize a
one-paragraph problem statement from it, flagged for my review.

Once the Draft file is at its final `<rdr-dir>/NNNN-slug.md` path, **add its row
to the RDR README index**: rdr-common §rdr-write, `--outcome readme`. Title and
Priority come from the RDR's own H1/Metadata. (A `Demoted` seed adds no row — it
runs no further stages.)

This is a skeleton, not a finished document. No ultrathink at seed time.
