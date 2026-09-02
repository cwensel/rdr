---
name: rdr-finalize
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: 'Use to run the Finalization Gate and lock an RDR to Final. Runs Stage 7: mechanical sweep plus written gate responses. Trigger for finalize/lock RDR, $rdr-finalize, or /rdr-finalize.'
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

1. Read [`rdr-common.md`](rdr-common.md) **whole, with the Read tool** (it exceeds the 30KB
   Bash cap — `cat` truncates and costs a retry; never `sed`/`grep` §-slices); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_ENV`, `RDR_PATH`, `{EVIDENCE_DIR}`, `{ARTIFACT_DIR}`
   (= `<RDR_RECORDS>/<RDR_SLUG>/artifacts/`).
2. Run [`07-finalize.prompt.md`](07-finalize.prompt.md). It runs the mechanical
   Tooling sweep, confirms no cluster re-entry note survives, writes the
   Finalization Gate's judgement responses to `{ARTIFACT_DIR}/gate.md`, then
   acts on the verdict:
   - **READY** → write responses 1, 2, 3 and 5 to `{ARTIFACT_DIR}/gate.md`
     (overwrite on re-lock), then run rdr-common **§rdr-write** twice and apply
     each `emit.edit` as handed — never retype the edits as prose:
     `--outcome lock` replaces those four sub-sections with the one-line
     pointer to gate.md and sets Status → Final (`### Cross-Cutting Concerns`
     STAYS in the record, because peers cite it as `cli/NNNN:G-cross-cutting`);
     `--outcome readme` **flips this RDR's README index row to Final** in place
     (add it only if a pre-seed RDR has none). A `stopped:gate-stale` from a
     probe run before gate.md was (re)written is not the lock — re-run after.
     Then, if the autocommit gate is on (§commit), run
     **§commit** with subject `docs(rdr): finalize cli/NNNN <slug> (Gate PASS)` over
     `$RDR_PATH` + `$RDR_RECORDS/README.md` + `{ARTIFACT_DIR}/gate.md` — a
     **standalone** commit, never a `fixup!`
     (RDR commits ARE the design history; per the no-fixup doctrine we record the lock
     as its own real subject, we do not defer-squash it).
   - **Sweep BLOCK** → mechanical findings (hollow-but-fillable sections such as
     `## References`, surviving template text including old-template blocks,
     anchor rewrites) are fixed in-pass and the sweep re-run — never routed to
     Refine, whose contract excludes conformance. Only substantive findings make
     the RDR NOT READY.
   - **NOT READY** → flip nothing; report the named blockers and the return
     stage. A finding re-reported unchanged after its named stage ran is a
     routing failure — stop per §stop-packet (`stopped:finalize-routing-loop:…`).
   - A single gate item genuinely in doubt → stop per §stop-packet, don't guess.
   - The Tooling sweep's checks live verbatim in [`tooling-pass.md`](tooling-pass.md)
     (CHECK 1 template coverage, CHECK 2 **Method-label vocabulary** against the
     sanctioned set, CHECK 3 source-search self-reference, CHECK 4 Docs-Only on
     load-bearing). **Run those CHECK blocks verbatim — do not re-derive them.** This
     is where a paraphrased Method label or a placeholder gate response is caught.
   - Linting the RDR `.md` is optional — an RDR-`.md`-only status flip usually
     needs none. If you do, honor the **consumer repo's** `.markdownlint.json`
     when it has one (don't impose engine-repo defaults), and never run
     markdownlint from inside the engine repo (`rdr/`) — with no config there it
     floods spurious MD013.

## Review gate (Stage `07.0-finalize.md`)

- The mechanical sweep ran and PASSED (a BLOCK is a real regression — fix before
  locking).
- Any `## Refinement Context (cluster re-entry)` note is gone.
- The four judgement responses are *written* in `{ARTIFACT_DIR}/gate.md`, not
  rubber-stamped, and `### Cross-Cutting Concerns` is still in the record.
- Readiness says READY, no open blockers; MVV genuinely in scope; the RDR's
  README index row flipped to Final (the row seed created). On NOT READY the
  prompt flips nothing — return to the named stage.

## Next step (rdr-common §next-step)

- The lock commit (standalone `docs(rdr): finalize …`, **never** `fixup!`) is part of the READY action above — see Usage / rdr-common §commit.
- Locked → `Next: /rdr-implement NNNN`;
  `"$RDR_HOME/bin/rdr" status --json --filter related_final_unimplemented NNNN`
  reading `1+` → `/rdr-cluster-reconcile NNNN` first.
- NOT READY → the named earlier stage (`/rdr-reconcile NNNN` for an open
  spike/assumption; `/rdr-resolve` / `/rdr-refine` / `/rdr-propose` per the blocker).
- `/rdr-status NNNN` to re-orient.
