---
authors: [NAME] <[EMAIL]>
state: prediscussion
discussion: [PR url, once open]
governs: [the JDRs and RDR clusters under this capability, or omit]
labels: [comma-separated]
---

# RFD [NUMBER] [The capability, as a noun phrase]

*[This is a position paper. It frames the desired experience and the tradeoffs
worth preserving. Names, syntax, commands, statuses and examples are
**non-normative**; the records own their exact contracts. The Principles below
are the exception — they are what this document commits to.]*

## Problem Statement

[What is wrong or missing today, in the user's terms. What it costs. No
mechanism — the point is the problem, and a problem statement that names its
solution has skipped the argument.]

## Background

[What exists now and why it is shaped that way. Prior art, and what it got
right. Non-normative.]

## Prior Art

[Conditional — delete the section when the capability rests on no external
work. What each source contributes and how it was checked, one bullet per
source: the claim taken from it, and the section of this document it
grounds. A source listed with no claim taken from it is a reading list, not
prior art.]

## Desired Experience

[The capability as the user meets it — journeys, in order, each with what the
user is trying to do and what they get. Examples are illustrative, and say so.
This is the section records cite for CONTEXT (`RFD NNNN §2`), and its numbers
freeze once cited; the prose under them stays free to evolve.]

### 1. [Journey]

### 2. [Journey]

## Principles

[**Required.** The citable surface. One line each, an RFC 2119 force word, a
rationale, derived from the narrative above rather than invented. A principle
earns its place by ruling options out. Ids are APPEND-ONLY: a principle that no
longer holds is marked superseded in place with a date and a pointer, and its
id never moves.]

- **P-1** [The commitment, in one line with MUST / SHOULD / MAY.]
  *(Derived from §[n].)*
- **P-2** [The commitment.] *(Derived from §[n].)*

<!-- Superseded, in place:
- **P-2** ~~[the old commitment]~~ — superseded 2026-09-16 by P-7; the id
  stays so every record that cited it still resolves.
-->

## Non-goals

[What this capability deliberately does not do, and the nearest thing it might
be confused with. A non-goal is as load-bearing as a principle for ruling
options out.]

## Open Questions

[What is genuinely undecided, as questions. A question that a seam's records
must agree on is a JDR entry, not an open question here — hoist it and cite the
entry.]

<!-- NOT IN AN RFD (rfd/README.md §What an RFD must not hold): a locked
     decision table or closed-question ledger (that is a JDR), a push-down or
     follow-on work list (the RDR index and the tracker), a reconcile
     scratchpad (git), or a transcript presented as contract. -->
