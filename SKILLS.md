# RDR Skills — Overview

The RDR flow ships as a suite of `/rdr-*` skills. Each one drives a single
**stage** of taking a Recommendation Decisioning Record from an idea to
implemented, spec-verified code — plus two read-only navigators that tell you
where you are and whether your setup is healthy.

This page is the map: a one-line blurb for every skill and how it maps to the
[workflow](README.md#workflow) and the [`stages/`](stages/README.md) recipe.
For the full description, run a skill with no arguments or read its
`skills/<name>/SKILL.md`.

## The flow at a glance

```
        setup            ┌──────────────── author the draft ────────────────┐   ┌─ lock ─┐  ┌ build ┐
  init ──► doctor ──►  seed ─► propose ─► refine ─► resolve ─► prelock ─► reconcile ─► finalize ─► implement
  (Stage 0)           (1)     (2)        (3)       (4)        (5)        (6)          (7)         (8)
                                                                                       │
                                                              cluster-reconcile (7.1) ─┘  (per cluster, optional)

  status ─ read-only navigator, run any time to ask "where is this RDR, what's next?"
```

## Setup & navigation

| Skill | What it does |
| --- | --- |
| **`/rdr-init`** | **Stage 0.** Run **once per project** to bind it to the flow — writes the per-project seam and the RDR home index README so the other skills can resolve their paths. Installing the plugin alone does *not* set this up; init does. |
| **`/rdr-doctor`** | Read-only health check. Confirms the seam binds, the engine resolves, the evidence root is reachable, and no symlink is broken — reports PASS/WARN/FAIL with the one fix per failure. Run after `init`, after an engine upgrade, or when a skill reports a `stopped:` seam error. |
| **`/rdr-status`** | Read-only navigator. Derives an RDR's position in the flow purely from disk and tells you what to run next. Omit the number to list in-flight RDRs. Writes nothing. |

## Authoring a record — Stages 1–6

| Skill | Stage | What it does |
| --- | --- | --- |
| **`/rdr-seed`** | 1 — Seed | Start a brand-new RDR from an idea, kata, or one-line description. Allocates the next number and writes a template-conformant Draft skeleton. |
| **`/rdr-propose`** | 2 — Propose | Move a seeded RDR from a problem statement to a chosen approach, with alternatives weighed and pre-mortemed. Also the front-half resume point — its freshness check re-validates a seed that sat idle. |
| **`/rdr-refine`** | 3 — Refine | Remove internal contradiction, redundancy, change-history narration, and bloat — make the draft internally consistent *before* its claims are verified. |
| **`/rdr-resolve`** | 4 — Resolve | Verify the RDR's critical assumptions against reality: research, spikes, and a reuse audit, flipping assumptions to **Verified** with a Method + Evidence. |
| **`/rdr-prelock`** | 5+6 — Pre-Lock | Run **one** pre-lock review lens against the draft *and* resolve its findings in the same pass. Lens is `grounding \| 3amigo \| critique \| repeatability \| cove`, picked by the RDR's risk profile. Loops until the lens converges. |
| **`/rdr-reconcile`** | 6 — Reconcile | Close every open spike and round-disturbed assumption: force each to a terminal state (VERIFIED / DOWNGRADED / ACCEPTED / BLOCKER) and write it into the RDR. Gates the lock. |

## Locking & building — Stages 7–8

| Skill | Stage | What it does |
| --- | --- | --- |
| **`/rdr-finalize`** | 7 — Finalize | Run the Finalization Gate and lock the RDR to **Final** — mechanical sweep + five written gate responses. READY locks in the same pass; NOT READY flips nothing and names the return stage. |
| **`/rdr-cluster-reconcile`** | 7.1 — Cluster Reconcile | **Per cluster, not per RDR** — most RDRs skip it. For a set of related RDRs that are all Final and none yet implemented: a whole-set critique + pairwise contradiction scan before any of the cluster implements. Routes Final→Draft on a defect. |
| **`/rdr-implement`** | 8 — Implement | Implement a locked (Final) RDR into working, spec-verified code. Re-entrant across days; terminates at COMPLETE/INCOMPLETE. A contract-level spec defect routes back to the RDR — it never patches over the spec. |

## How this maps to the process

Each skill is one stage of the [workflow in the README](README.md#workflow):
seeding and proposing live in **Create → Research → Decide**; refine, resolve,
prelock, and reconcile are the **Pre-Lock Review** and assumption-closing work;
finalize is the **Lock**; implement is **Implement**. The
[`stages/`](stages/README.md) directory is the same arc as a runnable recipe —
one parameterized prompt per stage, each with a review gate and an advance-when
condition — so a record can be driven end to end without reconstructing the
prompts from memory.

After an RDR is implemented, lived with, and code becomes the source of truth,
the [Post-Mortem Process](README.md#post-mortem-process) closes the loop —
comparing plan vs. reality to improve the authoring process itself.
