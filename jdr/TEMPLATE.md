---
authors: [NAME] <[EMAIL]>
state: open
rfd: [RFD NNNN, the governing capability — omit the key if none]
seam:
  - [path/prefix/]
  - [path/file.go::Symbol]
cluster: [NNNN, NNNN — members as of iteration [n]; a VIEW, not the identity]
labels: [comma-separated]
inherits: [RFD NNNN DX-1..DX-18 — legacy anchors this registry took over; omit if none]
---

# JDR [PROJECT]/[NUMBER] [The question this registry answers]

> One question over one seam. Entries are append-only and never merge, split
> or delete — a cited anchor never moves. Convention lives in
> `$RDR_HOME/jdr/README.md` and is never repeated here; process history lives
> in git and is never narrated here.

*[Identifiers and transcripts below are **non-normative**; the records own
exact contracts. Source: [the cluster gate run, or the fire that seeded this].]*

## Problem statement

[What the records on this seam jointly decide, and why no one of them can own
it. Name the seam in prose and say what goes wrong if each record answers
separately — the concrete drift, not the abstract risk.]

## Principles

[**Required.** Standing commitments every resolution below must satisfy.
Derive, never invent: each traceable to normative text the members already
lock, or to a decision resolved here — quote the source. One line each. A
principle earns its place by ruling options *out*; if every option on a fork
satisfies it, delete it.]

1. **[Principle]** — [rationale, with the source quoted or cited.]
2. **[Principle]** — [rationale.]

---

## D1 — [The fork, as a question]

[The conflict: what each record says, where, and why the two cannot both
stand. Cite `<project>/NNNN:<El>` anchors rather than restating their text.]

- **(a) [Option].** [Consequence. Which principle rules it out, if one does.]
- **(b) [Option] — recommended.** [Consequence, cost, and what it dissolves.]

**Resolved: ([x]).** [The decision, stated once, in the words the records will
cite rather than restate.]

*Lands in **[NNNN]**: [what changes there]. **[NNNN]** cites the decision
rather than restating it.*

<!-- Repeat D2…Dn. Most consequential first: a real fork buried under a small
     question is the defect that makes a registry unusable. A fork with no
     Resolved: line yet is `open` and blocks the records it binds. -->

## Interface record

[The `JD-n` entries: the settled surface records cite. Each states its answer,
the user-visible stake, and a short provenance parenthetical. Entry ids are per
registry and append-only; an id inherited from an earlier home keeps its
number.]

- **JD-1** `[status]` — [the answer, one statement.] [User-visible stake.]
  *(Provenance: [the fire, gate run, or decision that produced it].)*
  **Binds:** [NNNN, NNNN.]
- **JD-2** `[status]` — [the answer.] *(Provenance: …)* **Binds:** [NNNN.]

<!-- Status is one of open | constraint | blank | deferred | decided |
     withdrawn, per $RDR_HOME/models/jdr-entry.toml. `deferred` names the
     phase that settles it; `blank` names the owner who fills it;
     `withdrawn` keeps its reason so the anchor never dangles. Never delete
     an entry. -->

## What this does not decide

[Local items that stay with their records, so a reader does not go looking.
Name the record that owns each.]
