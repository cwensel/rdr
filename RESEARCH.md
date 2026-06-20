# RESEARCH — The Evidence Base Behind the RDR Process

The citation record for the RDR process — the template
([`TEMPLATE.md`](TEMPLATE.md)), the flow ([`stages/`](stages/README.md)), and the
prompts ([`prompts/`](prompts/README.md)). Read here *why* each part is shaped
the way it is, and follow the links to the primary sources.

1. **Methodology** — how the process was derived.
2. **Load-bearing citations** — the sources the process depends on, each mapped
   to the element it justifies.
3. **Wider bibliography** — the broader spec-driven-development (SDD) surface it
   draws on, so nothing is lost.

DOIs are linked as `https://doi.org/…`, preprints as `https://arxiv.org/abs/…`;
where a source is paywalled, an open alternative is given alongside.

---

## 1. Methodology — how this process was built

The RDR process is **mined from practice, not invented** — two halves, two
provenances:

- **The back half** (pre-lock lenses + implementation-launch orchestrator)
  adapts a spec-fitness prompt battery derived from a survey of three streams:
  software-inspection and requirements-engineering literature, industrial
  lightweight-formal-methods practice, and the spec-driven-development (SDD)
  tooling/discourse (GitHub Spec Kit, AWS Kiro, Tessl, Anthropic Agent Skills,
  practitioner blogs). The survey converged on a handful of high-ROI techniques
  — **premortem, perspective-based reading, example mapping, chain-of-verification,
  adversarial review, obstacle analysis** — that the pre-lock lenses
  ([`prompts/pre-lock/`](prompts/pre-lock/)) and review gates draw on. §2 maps
  each lens to its source.

- **The front half** (Seed → Propose → Refine → Resolve, [`stages/`](stages/README.md)
  stages 1–4) was mined from a real multi-RDR project's own AI session history:
  the recurring shape of how the author drove RDRs from idea to locked spec, and
  the ordering constraints (Resolve before Pre-Lock; Reconcile between rounds and
  lock), recovered from that corpus and cross-checked against the literature
  below. The mining record held project-specific transcripts and is **not**
  published — only the generalizable structure survives, here and in the stage
  files.

The result is a **filter cascade with review gates**: each stage removes a
defect class the next assumes gone, and each gate uses a *distinct* review
method rather than re-running the same check. The two load-bearing choices —
few phases, and review-method-per-gate as the lever rather than phase count —
are grounded in the literature (Piskala 2026; Porter et al. 1995; Lanubile &
Visaggio 2000).

### Internal evidence behind the template

The template was not designed up front; its current shape was driven by two
audits of the process running against itself, recorded in this project's commit
history (the audit documents were project-internal scratch and are not
published — the commits are the durable record):

- **The structured Evidence Record, the Normative/Illustrative split, and the
  applicable-only Cross-Cutting Concerns** replaced an earlier 3-label
  assumption checkbox and free-form code/verification tables. Driver: an audit
  of the first ten RDRs found **5 of 6 recurring authoring problems were
  template or vocabulary failures**, not author error — so the fix belonged in
  the template, not in guidance. The same audit's finding that load-bearing
  assumptions routinely cited their own RDR as "evidence" is why the
  **`Source Search` self-reference rule** exists (later broadened to any path
  under the RDR's artifact directory).
- **The Predecessors metadata, the Overrides metadata, and the Capability
  Dependencies table** were added as cross-RDR work began (Predecessors first,
  the other two later): RDRs were specifying future subsystems as if they
  existed and re-building capabilities a peer already owned. These slots force
  each load-bearing capability to be classed (exists now / this RDR /
  predecessor / deferred), and the (pre-existing) Existing Infrastructure Audit
  table to be filled, before design.
- **The Finalization Gate's mechanical pre-sweep** was relocated to last (just
  before lock) after **mining 16 real runs of the sweep**: ~82% of its findings
  were template-coverage issues a disciplined Refine already caught, and its
  self-reference check fired **zero** times once the Evidence Record and the
  Resolve stage existed. So the sweep earns its place as a *post-mutation
  regression guard* — confirming the review rounds didn't hollow a section —
  not as an opening defect hunt. (Porter et al. 1995 / Lanubile & Visaggio 2000:
  cut ceremony, not gates.)
- **The `Status: Draft [revised from Final …; re-verify …]` qualifier** was
  added when a Final RDR demoted by the cross-RDR cluster gate re-entered the
  flow as a bare Draft and re-verified every assumption from scratch — wasted
  work, since only the named cross-RDR defect had changed. The qualifier carries
  the scope on the live Status value (so the change-history scrub can't delete
  it, and the re-lock flip self-clears it).

---

## 2. Load-bearing citations, mapped to the process

Each entry lists what in the process it justifies, the full citation, and a
link. These are the sources the template/flow/prompts *depend on*; if you change
one of these elements, this is the reasoning you are changing.

### Overall shape — filter cascade, few phases, gate-method-not-phase-count

**Drives**: the [`prompts/`](prompts/README.md) lens set and the
[Stage 5 risk-profile matrix](stages/05-prelock.md); the written-response gate in [`TEMPLATE.md`](TEMPLATE.md).

- **Piskala (2026), *Spec-Driven Development: From Code to Contract in the Age of
  AI Coding Assistants*** — the "few phases, each producing an artifact that
  constrains the next, with a human review at each checkpoint" default, and the
  "minimum rigor that removes ambiguity" doctrine the flow's brevity rule
  restates. arXiv 2602.00180 — <https://arxiv.org/abs/2602.00180>
- **Porter, Votta & Basili (1995), *Comparing Detection Methods for Software
  Requirements Inspections: A Replicated Experiment*, IEEE TSE 21(7)** — the
  finding that *restructuring* an inspection (changing phase count, team size)
  changes cost, not defect yield; the **review method at each gate** is what
  catches defects. This is why the flow spends its economy on cutting prose, not
  gates. DOI 10.1109/32.391380 — <https://doi.org/10.1109/32.391380>
- **Lanubile & Visaggio (2000), *Are the Perspectives Really Different? — Further
  Experimentation on Scenario-Based Reading of Requirements*, Empirical Software
  Engineering 5(4):331–356** — companion evidence that *how* you read, not how
  many passes, drives yield; underpins the same ordering argument. DOI
  10.1023/A:1009848320066 — <https://doi.org/10.1023/A:1009848320066>

### Resolve-before-review ordering, and the 3amigo lens

**Drives**: [`stages/04-resolve-assumptions.md`](stages/04-resolve-assumptions.md) (resolve-before-review ordering) and the
[`prompts/pre-lock/1-3amigo.md`](prompts/pre-lock/1-3amigo.md) lens.

- **Basili, Green, Laitenberger, Lanubile, Shull, Sørumgård & Zelkowitz (1996),
  *The Empirical Investigation of Perspective-Based Reading*, Empirical Software
  Engineering 1(2)** — Perspective-Based Reading (PBR): build and verify a model
  of the artifact *before* hunting defects. This is why Stage 4 (Resolve) must
  precede Stage 5 (Pre-Lock), and the direct ancestor of the **3amigo** lens
  (read the draft as PM / Implementer / QA personas). DOI 10.1007/BF00368702 —
  <https://doi.org/10.1007/BF00368702>
- **Shull, Rus & Basili (2000), *How Perspective-Based Reading Can Improve
  Requirements Inspections*, IEEE Computer 33(7)** — the practitioner-facing
  case for PBR. DOI 10.1109/2.869376 — <https://doi.org/10.1109/2.869376>
  (open PDF: <https://www.cs.umd.edu/~basili/publications/journals/J79.pdf>)
- **Adzic (2011), *Specification by Example: How Successful Teams Deliver the
  Right Software*, Manning** — the BDD "Three Amigos" and Example Mapping
  practice the 3amigo lens combines with PBR.
  <https://www.manning.com/books/specification-by-example>

### Critique / premortem lens

**Drives**: [`prompts/pre-lock/2-critique.md`](prompts/pre-lock/2-critique.md).

- **Klein (2007), *Performing a Project Premortem*, Harvard Business Review** —
  the premortem technique the Critique lens is built on: imagine the failure has
  already happened, then work backward to its causes.
  <https://hbr.org/2007/09/performing-a-project-premortem>
- **Dhuliawala, Komeili, Xu, Raileanu, Li, Celikyilmaz & Weston (2024),
  *Chain-of-Verification Reduces Hallucination in Large Language Models*, ACL
  Findings 2024** — the *independence* discipline (a fresh context with no prior
  answers beats "now double-check yourself") that powers both the CoVe lens and
  the dual-model / fresh-context anti-sycophancy step in Critique. arXiv
  2309.11495 — <https://arxiv.org/abs/2309.11495> ; ACL Anthology —
  <https://aclanthology.org/2024.findings-acl.212/>
- **Mitani et al. (2025), *LLM-AQuA-DiVeR: LLM-Assisted Quality Assurance Through
  Dialogues on Verifiable Specification*, RAIE 2025 @ ICSE** — LLM-assisted spec
  QA via stakeholder dialogue; cited for the practical caveat the Critique lens
  encodes (an LLM tends to agree with a plausible spec, so run dual-model and
  diff rather than asking one model to critique its own draft).
  <https://conf.researchr.org/details/icse-2025/raie-2025-papers/6/>

### Propose selection — read prior art first, compare before committing

**Drives**: the **retrieval-first prior-art read + cite-check** and the
**scored Questions-Options-Criteria matrix** (`large`/`foundational`) in
[`prompts/stages/02-propose.prompt.md`](prompts/stages/02-propose.prompt.md). The
design treats Propose as owning *selection* (which approach wins, read from prior
art) and Resolve as owning *verification* (the expensive spikes) — a split of
depth, not stage. The case that forced it: a corpus run where a chosen approach
was overturned three stages late by a Resolve spike, root-caused to **prior art
misread at Propose** (citations inverted, not merely unread) — verification
working as designed, but firing far past the cheapest point to switch. Four
independent literatures converge on doing the alternatives-comparison early while
deferring only the irreversible *commitment*:

- **Boehm & Basili (2001), *Software Defect Reduction Top 10 List*, IEEE Computer
  34(1)** — the cost-of-change direction: defects caught after delivery cost
  ~100× a requirements/design-time catch, and ~40–50% of effort is avoidable
  rework. The magnitude is softenable under agile practice (Kuppuswami et al.
  2003, *XP and the Cost-of-Change Curve*, LNCS, DOI 10.1007/3-540-44870-5_8), but
  the direction — earlier is cheaper — holds. DOI 10.1109/2.962984 —
  <https://doi.org/10.1109/2.962984>
- **Jansson & Smith (1991), *Design Fixation*, Design Studies 12(1)** — committing
  to one early idea makes designers reproduce its features (even bad ones) and
  generate worse alternatives; the hazard the enumerate-2–4 rule and the matrix
  counter. DOI 10.1016/0142-694X(91)90003-F —
  <https://doi.org/10.1016/0142-694X(91)90003-F>
- **Dow, Glassco, Kass, Schwarz, Schwartz & Klemmer (2010), *Parallel Prototyping
  Leads to Better Design Results*, ACM TOCHI 17(4)** — controlled evidence that
  creating and comparing *multiple* options in parallel yields objectively better,
  more diverse outcomes than serial single-option iteration. Justifies the scored
  matrix over prose. DOI 10.1145/1879831.1879836 —
  <https://doi.org/10.1145/1879831.1879836>
- **Sobek & Ward (1996), *Principles from Toyota's Set-Based Concurrent
  Engineering*, ASME DTM** (with Ford & Sobek 2005, real-options framing, IEEE
  TEM 52(2), DOI 10.1109/TEM.2005.844466) — keep alternatives alive and converge
  *late*; the lean "last responsible moment" (Poppendieck & Cusumano 2012, IEEE
  Software 29(5), DOI 10.1109/MS.2012.107) is not a counter-argument but its basis:
  explore the set early, lock the irreversible choice late. DOI
  10.1115/96-DETC/DTM-1510 — <https://doi.org/10.1115/96-DETC/DTM-1510>
- **Deng, Brucks & Toubia (2026), *Examining and Addressing Barriers to Diversity
  in LLM-Generated Ideas*** — the load-bearing LLM finding: **LLMs fixate like
  humans, early outputs constrain later ideation**, and self-evaluation is
  unreliable (cf. Huang et al. 2023, *LLMs Cannot Self-Correct Reasoning Yet*,
  arXiv 2310.01798). The fix is grounding *before* generation: Nova (Hu et al.
  2024, arXiv 2410.14255) got 3.4× more novel ideas from retrieval-first ideation,
  and *Premise Order Matters* (Chen et al. 2024, arXiv 2402.08939) shows
  context-before-reasoning ordering is itself load-bearing. This is the direct
  justification for reading prior art **before** enumerating, not after. arXiv
  2602.20408 — <https://arxiv.org/abs/2602.20408>

### Repeatability lens

**Drives**: [`prompts/pre-lock/3-repeatability.md`](prompts/pre-lock/3-repeatability.md); the paired internal-silence
check is [`prompts/pre-lock/4-cove.md`](prompts/pre-lock/4-cove.md).

- **Böckeler (2025), *Understanding Spec-Driven-Development: Kiro, spec-kit, and
  Tessl*** (on martinfowler.com) — the non-determinism / regenerate-and-compare
  experiment: generate from the same spec several times and treat the
  *disagreements* as the spec's silences. This is the entire basis of the
  Repeatability lens (3 runs, ≥1 on a different base model, then diff).
  <https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html>

### Grounding lens & cove-as-codebase-oracle

**Drives**: [`prompts/pre-lock/0-grounding.md`](prompts/pre-lock/0-grounding.md)
(the Mid/Large grounding sweep) and Step 0 of
[`prompts/pre-lock/4-cove.md`](prompts/pre-lock/4-cove.md) (cove leads with it at
Foundational). The grounding lens checks an RDR's codebase claims against *source*
— catching a frame that is internally coherent but false against the code, which a
self-referential review cannot see.

- **Stroebl, Kapoor & Narayanan (2024), *The Limits of Inference Scaling Through
  Resampling*** (formerly *Inference Scaling fLaws*) — the verifier dichotomy:
  resampling against an *imperfect* verifier (an LLM opining) has a finite accuracy
  ceiling set by its false-positive rate, while an **oracle** (a deterministic
  check, false-positive rate ≈ 0) has no such ceiling. This is why the grounding
  pass must *read code* (oracle-class) rather than ask a model whether a claim
  "seems grounded" (imperfect). arXiv 2411.17501 —
  <https://arxiv.org/abs/2411.17501>
- **Cemri et al. (2025), *Why Do Multi-Agent LLM Systems Fail?* (MAST)** — the
  failure taxonomy in which system-design / specification issues are the largest
  category, and adding one high-level grounding/verification step was the single
  largest intervention measured; the empirical case for grounding early rather
  than relying on late, low-level checks. arXiv 2503.13657 —
  <https://arxiv.org/abs/2503.13657>
- **Zhu et al. (2025), *Where LLM Agents Fail and How They Can Learn From
  Failures*** — early, root-cause errors dominate and cascade downstream; fixing
  the earliest critical step yields far more than patching surface symptoms. The
  reason grounding keys off "locks a contract" (where a wrong frame is set), and
  the reason the accretion gate forces the root design decision before the next
  patch. arXiv 2509.25370 — <https://arxiv.org/abs/2509.25370>

### Accretion floor & blast-radius profile sizing

**Drives**: the **Seam Lineage** Metadata field and the accretion-floor Profile
rule in [`TEMPLATE.md`](TEMPLATE.md); the deterministic floor in the
[Stage 5 matrix](stages/05-prelock.md) and [README matrix](README.md); the Seed
fill ([`prompts/stages/01-seed.prompt.md`](prompts/stages/01-seed.prompt.md)) and the
Propose accretion gate ([`prompts/stages/02-propose.prompt.md`](prompts/stages/02-propose.prompt.md)).
Profile sized by contract count alone lets an RDR self-scope to one contract and
under-gate a seam that already carries prior point-fixes; the accretion axis floors
it instead.

- **Rabanser (2026), *Towards a Science of AI Agent Reliability*** — consistency /
  determinism is a first-class reliability axis ("an agent that fails predictably
  is more manageable than one that succeeds unpredictably"), and scrutiny should
  be allocated by consequence / blast-radius. The basis for making the accretion
  count a *mechanical* gate (read from a field, not re-judged each stage) and for
  sizing the profile by blast radius rather than contract count. arXiv 2602.16666 —
  <https://arxiv.org/abs/2602.16666>
- **Provenance-as-durable-artifact** — the accretion count is copied once from the
  `kata-scope-review §seam-accretion` emission into Seam Lineage, never re-derived:
  a hand-carried count drifts across re-derivations. This is the same
  "durable disposition, written once" principle the pre-lock loop already follows.

### Cost discipline — why grounding is breadth, not added volume

**Drives**: the choice to run a *lightweight* grounding Step-0 at Mid (not the full
cove persona pass), and the budget intent behind concentrating gates on
high-accretion loci.

- **Chen, Davis, Hanin, Bailis, Stoica, Zaharia & Zou (2024), *Are More LLM Calls
  All You Need? Towards the Scaling Properties of Compound AI Systems*, NeurIPS
  2024** — aggregating more *redundant* calls over a fixed task has non-monotone
  returns; the caution against uniformly piling review passes onto the common
  profile. (Grounding sidesteps it by adding verification *breadth* — running a
  check the RDR already requires — not redundant depth.) arXiv 2403.02419 —
  <https://arxiv.org/abs/2403.02419>
- **Demirer, Musolff & Yang (2026), *Writing Code vs. Shipping Code: Productivity
  Effects Across Generations of AI Coding Tools*, NBER w35275 / SSRN** — AI gains
  attenuate up the production chain (human review is the bottleneck), so heavier
  *upstream* gating pays only when concentrated where the risk is. The budget
  rationale for the accretion floor as a concentrator rather than blanket
  escalation. <https://www.nber.org/papers/w35275>

### Pairwise / cross-RDR consistency (Stage 7.1)

**Drives**: [`prompts/gate/pairwise.md`](prompts/gate/pairwise.md), dispatched at
[`stages/07.1-cluster-reconcile.md`](stages/07.1-cluster-reconcile.md).

- **Finkelstein, Gabbay, Hunter, Kramer & Nuseibeh (1994), *Inconsistency
  Handling in Multiperspective Specifications*, IEEE TSE 20(8)** — the viewpoints
  tradition: tolerate inconsistency between views during work, and reconcile it
  at *chosen* points rather than enforcing global consistency as a precondition.
  This is exactly how the flow defers cross-RDR drift to Stage 7.1 instead of
  gating every per-RDR edit on it. DOI 10.1109/32.310667 —
  <https://doi.org/10.1109/32.310667>
  (open PDF: <https://www.finkelstein.live/papers/tse94.esec.pdf>)
- **Nentwich, Capra, Emmerich & Finkelstein (2002), *xlinkit: A Consistency
  Checking and Smart Link Generation Service*, ACM TOIT 2(2)** — mechanized
  cross-document consistency checking; the conceptual ancestor of the pairwise
  cross-RDR check. DOI 10.1145/514183.514186 —
  <https://doi.org/10.1145/514183.514186>
- **Liu, Wang & Zhai (2025), *A Hybrid Framework for Inconsistency Detection in
  Diversity Requirements: Combining Multi-Graph Merging and LLM*, QRS 2025** —
  contemporary LLM-assisted cross-spec inconsistency detection; cited as the
  Pairwise lens's modern anchor. DOI 10.1109/QRS65678.2025.00014 —
  <https://doi.org/10.1109/QRS65678.2025.00014>

### Reconcile gate (Stage 6) — residual-risk closure

**Drives**: [`stages/06-reconcile.md`](stages/06-reconcile.md).

- **Letier & van Lamsweerde (2025), *Obstacle Analysis in Requirements
  Engineering: Retrospective and Emerging Challenges*, IEEE TSE 51(3)** —
  obstacle analysis: residual risk is closed only when it is *explicitly judged
  acceptable*, not silently. This is the rationale for making Reconcile a forced
  gate (a refuting spike re-opens the RDR; an MVV-critical spike cannot defer
  past lock) rather than a checkbox. DOI 10.1109/TSE.2025.3534318 —
  <https://doi.org/10.1109/TSE.2025.3534318>
  (open: <https://discovery.ucl.ac.uk/id/eprint/10204032/>)

### Evidence discipline & defect economics (the template's core)

**Drives**: the *Critical Assumptions* Evidence-Record model in [`TEMPLATE.md`](TEMPLATE.md), enforced mechanically by
[`prompts/gate/tooling-pass.md`](prompts/gate/tooling-pass.md).

- **Boehm (1984), *Software Engineering Economics*, IEEE TSE SE-10(1)** — the
  cost-of-late-defects argument that motivates spending tokens on verifying
  assumptions *before* code is generated. DOI 10.1109/TSE.1984.5010193 —
  <https://doi.org/10.1109/TSE.1984.5010193>
- **Haskins, Stecklein, Dick, Lovell, Moroney & Lempia (2004), *Error Cost
  Escalation Through the Project Life Cycle*, INCOSE International Symposium** —
  the empirical cost-escalation curve behind "verify load-bearing claims
  early." DOI 10.1002/j.2334-5837.2004.tb00608.x —
  <https://doi.org/10.1002/j.2334-5837.2004.tb00608.x>
- **Chillarege, Bhandari, Chaar, Halliday, Moebus, Ray & Wong (1992),
  *Orthogonal Defect Classification — A Concept for In-Process Measurements*,
  IEEE TSE 18(11)** — ODC; the conceptual basis for the **deviation taxonomy**
  (SPEC-DEFECT / SPEC-UNDER / DEPENDENCY-LIMIT / TEST-FIXTURE / IMPL-DECISION)
  defined in [`README.md`](README.md) and
  [`prompts/implementation/launch.md`](prompts/implementation/launch.md), which
  classifies implementation divergences so patterns can be analyzed across RDRs.
  DOI 10.1109/32.177364 — <https://doi.org/10.1109/32.177364>

### Implementation launch (Stage 8)

**Drives**: [`prompts/implementation/launch.md`](prompts/implementation/launch.md), dispatched at
[`stages/08-implement.md`](stages/08-implement.md).

- **Fakhoury, Naik, Sakkas, Chakraborty & Lahiri (2024), *LLM-based Test-driven
  Interactive Code Generation* (TiCoder), IEEE TSE** — tests-from-spec-first and
  ranking generations by test-consistency; the basis for the launch prompt's
  Phase 1 "write tests from the REQ-N quotes before any implementation."
  <https://www.seas.upenn.edu/~asnaik/assets/papers/tse24_ticoder.pdf>
  (DOI 10.1109/TSE.2024.3428972)
- **Osmani (2025), *How to write a good spec for AI agents*** — the
  conformance-testing + per-test spec-citation pattern the launch prompt's
  coverage artifact (`REQ-N × test-name`) implements.
  <https://addyosmani.com/blog/good-spec/>
  (O'Reilly Radar version: <https://www.oreilly.com/radar/how-to-write-a-good-spec-for-ai-agents/>)
- (Dhuliawala et al. 2024, above) also anchors Phase 3a's CoVe verifier and the
  structural independence between phase sub-agents.

### The RDR concept itself

- **Architecture Decision Records (ADRs)** — the conceptual ancestor. RDRs
  differ in that they evolve *during* planning and capture supporting evidence,
  whereas an ADR documents a decision already made.
  <https://adr.github.io>

---

## 3. Wider bibliography

The process does not strictly depend on the sources below, but they are the
research surface the prompt battery and flow were shaped against. Grouped by
theme; included so an adopter can go deeper and so the provenance is complete.

### Software inspection & requirements engineering

- Fagan (1976), *Design and code inspections to reduce errors in program
  development*, IBM Systems Journal — the origin of formal inspection.
- Porter, Votta & Basili — see §2 (load-bearing). Porter & Votta (1994), *An
  experiment to assess different defect detection methods for software
  requirements inspections*, ICSE, is the original of the 1995 TSE replication.
- Sabaliauskaite et al. (2002), perspective-based reading replication, ICSE.
- Fogelström & Gorschek (2007), inspection economics.
- Rigby (2013), *Convergent Contemporary Software Peer Review Practices*.
- van Lamsweerde & Letier (2000), *Handling Obstacles in Goal-Oriented
  Requirements Engineering*, IEEE TSE; *From System Goals to Software
  Architecture* (2004) — DOI 10.1109/RE.2004.25; *Goal-Oriented Requirements
  Engineering: A Guided Tour* (RE 2001).
- Easterbrook, Finkelstein, Kramer & Nuseibeh (1994), *Coordinating Conflicting
  Viewpoints by Managing Inconsistency*.
- Sindre & Opdahl (2005), *Eliciting Security Requirements with Misuse Cases*,
  Requirements Engineering Journal — DOI 10.1007/s00766-004-0194-4; McDermott &
  Fox (1999), *Using Abuse Case Models for Security Requirements Analysis*,
  ACSAC.
- Kazman et al. — SAAM (1994/1996) and ATAM (1999–) scenario-based architecture
  evaluation.
- Robertson & Robertson (2003), *Volere Requirements Specification Template*.

### LLMs for requirements & specification

- Bashir, Ferrari, Khan et al. (2025), *Requirements Ambiguity Detection and
  Explanation with LLMs: An Industrial Study*, ICSME — DOI
  10.1109/ICSME64153.2025.00063.
- Oghabi & Kahani (2025), *Multi-Label Ambiguity Detection in Software
  Requirements Using Language Models*, CASCON — DOI 10.1109/CASCON66301.2025.00076.
- Cheng, Husen, Lu, Racharak et al. (2026), *Generative AI for Requirements
  Engineering: A Systematic Literature Review*, SPE — DOI 10.1002/spe.70029.
- Zadenoori, Dąbrowski, Alhoshan et al. (2025), *LLMs for Requirements
  Engineering: A Systematic Literature Review*, arXiv 2509.11446.
- Huang, Wang, Huang & Arora (2025), *Prompt Engineering for Requirements
  Engineering: A Literature Review and Roadmap*, REW — arXiv 2507.07682.
- Marri (2026), *Constitutional Spec-Driven Development*, arXiv 2602.02584.
- Rehan (2026), *Test-Driven AI Agent Definition*, arXiv 2603.08806.
- Taghavi & Bhavani (2026), *Spec Kit Agents: Context-Grounded Agentic
  Workflows*, arXiv 2604.05278.

### Verification & adversarial review patterns

- Devil's Advocate / *Anticipatory Reflection for LLM Agents*, EMNLP Findings
  2024 — arXiv 2405.16334.
- DEBATE multi-agent verification — arXiv 2405.09935.
- Cross-Verification Collaboration Protocol (CVCP), MDPI Symmetry 2025 —
  <https://www.mdpi.com/2073-8994/17/10/1660>.
- SWE-Debate — arXiv 2507.23348; TestFlow (acceptance-test generation) —
  arXiv 2504.07244.
- Property-based testing: Hughes & Claessen QuickCheck (2000); Goldstein,
  Cutler, Dickstein & Pierce (2024), *Property-Based Testing in Practice*, ICSE
  — DOI 10.1145/3597503.3639581.
- Metamorphic testing: Chen et al. (2018), ACM Computing Surveys — DOI
  10.1145/3143561; Segura et al. (2016), IEEE TSE.

### Lightweight formal methods (industrial)

- Bornholt et al. (2021), *Using Lightweight Formal Methods to Validate a
  Key-Value Storage Node in Amazon S3*, SOSP '21 — DOI 10.1145/3477132.3483540.
- Newcombe et al. (2014/2015), *Use of Formal Methods at Amazon Web Services* —
  <https://6826.csail.mit.edu/2019/papers/formal-methods-amazon.pdf>.
- Konnov, Kuppe & Merz (2022), *Specification and Verification With the TLA+
  Trifecta*, ISoLA; Konnov, Kukovec & Tran (2019), *TLA+ Model Checking Made
  Symbolic* (Apalache), PACMPL.
- Reid et al. (2020), *Towards Making Formal Methods Normal: Meeting Developers
  Where They Are*, arXiv.
- Jackson, *Software Abstractions* (Alloy), MIT Press; Bögli et al. (2025), *A
  Systematic Literature Review on a Decade of Industrial TLA+ Practice*, LNCS —
  DOI 10.1007/978-3-031-76554-4_2.
- Hawblitzel et al. (2015), *IronFleet*, SOSP '15.

### Spec-driven-development tooling & practitioner discourse

- GitHub **Spec Kit** — <https://github.com/github/spec-kit>; launch post —
  <https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/>.
- AWS **Kiro** — <https://kiro.dev/docs/specs/>; best practices —
  <https://kiro.dev/docs/specs/best-practices/>; intro —
  <https://kiro.dev/blog/introducing-kiro/>.
- **Tessl** (spec-driven framework + registry) —
  <https://tessl.io/blog/tessl-launches-spec-driven-framework-and-registry/>;
  Guy Podjarny interview — <https://www.gv.com/news/guy-podjarny-interview-tessl-ai-software-development>.
- Anthropic **Agent Skills** —
  <https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills>;
  *Effective context engineering for AI agents* —
  <https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents>.
- **AGENTS.md** — <https://agents.md/>; **llms.txt** (Jeremy Howard) —
  <https://llmstxt.org/>.
- Martin Fowler / Thoughtworks *Exploring Gen AI* series —
  <https://martinfowler.com/articles/exploring-gen-ai.html>; Böckeler,
  *Context Engineering for Coding Agents* —
  <https://martinfowler.com/articles/exploring-gen-ai/context-engineering-coding-agents.html>.
- Simon Willison, *Vibe Engineering* —
  <https://simonwillison.net/2025/Oct/7/vibe-engineering/>.
- Andrej Karpathy, *2025 LLM Year in Review* —
  <https://karpathy.bearblog.dev/year-in-review-2025/>.
- Addy Osmani, *My LLM coding workflow going into 2026* —
  <https://addyosmani.com/blog/ai-coding-workflow/>.
- Factory.ai, *Using Linters to Direct Agents* —
  <https://factory.ai/news/using-linters-to-direct-agents>.
- Cognition (Devin), *Coding Agents 101* — <https://devin.ai/agents101>.

---

*This document travels with the process. When a stage or prompt adds a
load-bearing dependency, add it to §2.*
