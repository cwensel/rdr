Seed a new RDR from: {IDEA}. See the rdr README for what an RDR is.

Read {RDR_ENV} for the project's RDR directory — the parent of {ARTIFACT_DIR}
(the RDR file and its artifact dir are siblings: file `<rdr-dir>/NNNN-slug.md`,
artifacts `<rdr-dir>/NNNN-slug/`). If {RDR_ENV} names no RDR directory yet, ask me
before writing.

**Claim the number atomically FIRST** (rdr-common §rdr-claim), before authoring
anything: in a retry-on-collision loop, take `max(NNNN)+1` and materialize
`<rdr-dir>/NNNN-RESERVED.md` **as a copy of TEMPLATE.md** under `set -C`
(noclobber); on collision, bump and retry. This both reserves the slot (so a
concurrent session picks NNNN+1) and gives you the canonical skeleton to fill.

**The TEMPLATE.md copy is the skeleton — do not author structure from scratch and
do not copy another RDR for "house style."** A neighbor RDR is a *finished*
document; copying it drifts the structure and leaks its chosen solution into your
Draft. Fill ONLY Metadata, Problem Statement, and Context (plus `[NUMBER]` /
`[TITLE]` in the H1) from the kata, editing in place. Leave every other section
**exactly as the template ships it** — its placeholders already are the Draft
placeholders. Do NOT invent a solution, assumptions, or research findings; those
are later stages. Once the slug is settled,
`git mv <rdr-dir>/NNNN-RESERVED.md <rdr-dir>/NNNN-slug.md`.

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

This is a skeleton, not a finished document. No ultrathink at seed time.
