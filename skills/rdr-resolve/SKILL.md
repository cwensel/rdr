---
name: rdr-resolve
metadata:
  argument-hint: <NNNN> [--commit | --no-commit]
description: 'Use to verify critical RDR assumptions against source, spikes, prior art, and reuse audits. Runs Stage 4 before pre-lock review. Trigger for resolve assumptions, $rdr-resolve, or /rdr-resolve.'
---

# rdr-resolve — Stage 4 (Resolve Assumptions)

Verify every Critical Assumption against source, live spikes, prior art, and
standards so the pre-lock rounds critique a *true* document. The load-bearing
front-half stage. Running a pre-lock round before this one wastes the round.

## Usage

```
Codex: $rdr-resolve <NNNN>
Claude: /rdr-resolve <NNNN>
```

1. Read [`rdr-common.md`](rdr-common.md) **whole, with the Read tool** (it exceeds the 30KB
   Bash cap — `cat` truncates and costs a retry; never `sed`/`grep` §-slices); run **§seam-bind** + **§rdr-resolve**
   to bind `$RDR_ENV`, `$RDR_RESOURCES`, `RDR_PATH`, `RDR_SLUG`, `{SPIKE_DIR}`.
2. **Run the prompt** [`04-resolve.prompt.md`](04-resolve.prompt.md)
   with those bound. Have a live spike target reachable. The prompt is read-heavy —
   delegate corpus searches / reuse audits / dependency spelunks to a sub-agent that
   returns *verdict + evidence pointer* (§return-packet), not raw hits (rdr-common §delegation — spawn,
   then wait by ending the turn).
   - **Fan-out heuristic (mined):** group the **Source-Search** assumptions into one
     sub-agent (multi-pass, one verdict each) and give each **Spike** assumption its
     own sub-agent (it needs the live target). Don't spawn one agent per source-search
     claim — they share the same corpus/source reads. Hold only the verdicts in the
     main context; author the `Status: Verified` edits here, not in the sub-agent.
3. **Self-detected re-entry (no flag).** The prompt asks the projector, not the
   `Status:` string: `status --tags <NNNN>` (rdr-common §rdr-resolve) →
   `status_form=revised-from` — or `routed-back`, which also clears its qualifier
   here (rdr-common §rdr-write *Receiving a route-back*) — scopes the run to the
   `reverify=[…]` ids (+ anchors the demotion touched), carrying the rest forward
   as Verified; any other form is the cold path. The record itself is one `inspect --json --filter
   path,metadata <NNNN>` call — never `edges` here. Do not pass a resume
   flag — the RDR's state drives it. "Did refine (or any earlier stage) already run?"
   is answered by `git log --oneline -5 -- "$RDR_PATH"` (stage-named subjects) or
   `recs status --tags NNNN` — never by §lens-row, which only names the next lens.

## Review gate (what the human checks — Stage `04-resolve-assumptions.md`)

- Every load-bearing assumption actually `Verified`, Evidence not self-reference and
  not bare `Docs Only`.
- Spikes actually ran (command + captured output under `{SPIKE_DIR}`).
- One consolidated author's round carried every fixture **and** every question
  only the user can settle, each with its grounding; approved fixtures named as
  normative fixtures in the RDR body — an unapproved fixture is not Evidence.
  Most RDRs render no fixtures; the round still runs if it has a question.
  Delegated, the round is in `author-round.md` and the packet is `NEEDS_DECISION`
  (prompt §author's round), never a summary.
- The reuse audit ran against the `$RDR_ENV` reuse-audit paths (or the close
  packet says `reuse audit: n/a — scoped re-entry`).
- No research finding contradicts the approach — if one does, that's **not** a
  citation fix.

## Next step (rdr-common §next-step)

- If autocommit is on, run **§commit** for `resolve` first: `rdr_commit "docs(rdr): resolve cli/NNNN — <summary>" "$RDR_PATH"`,
  then, only if a spike wrote, `rdr_commit "chore(rdr): cli/NNNN spike evidence" "{SPIKE_DIR}"` (the rdr-commit-map.md row, inlined).
- Research refuted the approach or surfaced existing capability → **back** to
  `/rdr-propose NNNN` (or `/rdr-refine NNNN`); rework, then re-run this. Apply
  §rdr-write `--outcome return --tag blocker_class=<class>`: its Status qualifier
  is what marks the backward edge. Unwritten, the record still reads as
  already-proposed and Stage 2 refuses — the packet is not a marker.
- Assumptions verified → forward. This stage **sets the `Profile` Metadata
  field** (the routing latch) via §rdr-write `--outcome profile`, fed
  `--outcome floor`'s answer, which raises the count's tier by one (the prompt owns the call). Then **run
  §lens-row's call** for the next pointer — it reads the field this stage just
  wrote, and the first lens is profile-dependent (`foundational` leads with
  `cove`, not `grounding`), so an inferred row is how a lens gets skipped.

  `emit.next` is the pointer; append `NNNN`. `stopped:no-profile` means the
  latch was not written — fix that here, do not default the row.
- `/rdr-status NNNN` to re-orient.
