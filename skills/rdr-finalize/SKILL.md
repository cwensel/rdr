---
name: rdr-finalize
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: Use to run the Finalization Gate and lock an RDR to Final (e.g. "finalize RDR 46", "lock RDR 46", "$rdr-finalize 0046" in Codex, "/rdr-finalize 0046" in Claude). Runs Stage 7 of the RDR flow as one gated prompt — mechanical sweep + five written gate responses; READY locks (Status → Final) in the same pass, NOT READY flips nothing and names the return stage. Pairs with $rdr-implement in Codex or /rdr-implement in Claude (next) and $rdr-reconcile or /rdr-reconcile (back).
---

# rdr-finalize — Stage 7 (Finalize / Lock)

Confirm nothing regressed, run the Finalization Gate, and flip Status → **Final**.
After this the RDR is immutable — the implementation prompt treats it as a contract.
This stage is **one gated prompt**: the verdict is the gate, not a human pause —
READY locks in the same pass, NOT READY flips nothing.

## Usage

```
Codex: $rdr-finalize <NNNN>
Claude: /rdr-finalize <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_ENV`, `RDR_PATH`, `{EVIDENCE_DIR}`.
2. Run [`07-finalize.prompt.md`](07-finalize.prompt.md). It runs the mechanical
   Tooling sweep, confirms no cluster re-entry note survives, writes the
   Finalization Gate's five responses, then acts on the verdict:
   - **READY** → set Status → Final, write the gate's five responses into the RDR,
     update the README index row. Then, if the autocommit gate is on (§commit), run
     **§commit** with subject `docs(rdr): finalize cli/NNNN <slug> (Gate PASS)` over
     `$RDR_PATH` + `$RDR_RECORDS/README.md` — a **standalone** commit, never a `fixup!`
     (RDR commits ARE the design history; per the no-fixup doctrine we record the lock
     as its own real subject, we do not defer-squash it).
   - **NOT READY** → flip nothing; report the named blockers and the return stage.
   - A single gate item genuinely in doubt → stop per §stop-packet, don't guess.
   - The Tooling sweep's checks live verbatim in [`tooling-pass.md`](tooling-pass.md)
     (CHECK 1 template coverage, CHECK 2 **Method-label vocabulary** against the
     sanctioned set, CHECK 3 source-search self-reference, CHECK 4 Docs-Only on
     load-bearing). **Run those CHECK blocks verbatim — do not re-derive them.** This
     is where a paraphrased Method label or a placeholder gate response is caught.
   - If you lint the RDR `.md`, use the repo's `.markdownlint.json` from the repo
     root (it sets `line_length: 120`, `tables: false`) — running markdownlint from
     inside the engine repo (`rdr/`) floods spurious MD013. An RDR-`.md`-only status flip usually
     needs no lint at all.

## Review gate (Stage `07.0-finalize.md`)

- The mechanical sweep ran and PASSED (a BLOCK is a real regression — fix before
  locking).
- Any `## Refinement Context (cluster re-entry)` note is gone.
- The five gate responses are *written*, not rubber-stamped.
- Readiness says READY, no open blockers; MVV genuinely in scope; README index
  updated. On NOT READY the prompt flips nothing — return to the named stage.

## Next step (rdr-common §next-step)

- The lock commit (standalone `docs(rdr): finalize …`, **never** `fixup!`) is part of the READY action above — see Usage / rdr-common §commit.
- Locked → `Next: /rdr-implement NNNN`. (If ≥2 related RDRs are now Final and none
  implemented → `/rdr-cluster-reconcile <cluster>` first.)
- NOT READY → the named earlier stage (`/rdr-reconcile NNNN` for an open
  spike/assumption; `/rdr-resolve` / `/rdr-refine` / `/rdr-propose` per the blocker).
- `/rdr-status NNNN` to re-orient.
