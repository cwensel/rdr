# Stage 0 — Bootstrap (one-time per project)

**Goal**: set a project up to run the RDR flow **without the project knowing
about RDR** — no versioned footprint, no `CLAUDE.md` edits, nothing in the
project's own `.gitignore`. The flow expects two files: `rdr-resources.md` (the
evidence index) and `rdr-env.md` (the path map), defaulting to an `_rdr/` folder
at the project root. This stage creates them, filled from the project itself,
and ensures `_rdr/` stays out of git. Run once; re-run only to refresh values.

> **Already have a pinned seam?** A project may keep its seam (and evidence) in
> a tracked external directory instead of gitignored `_rdr/` — e.g. a consumer
> that keeps `rdr-resources.md` / `rdr-env.md` in a sibling seam repo (such as a
> `flow/context/`), with evidence under that repo's `rdr/evidence/`. If a
> **workspace marker**
> (`.rdr-workspace` at the workspace parent, see the README *Where the seam
> lives*) already exports `$RDR_RESOURCES` / `$RDR_ENV`, **skip this stage** —
> the resolver finds them. Bootstrap only creates the fallback `_rdr/` seam (and
> a marker pointing at it) for a project that has none.

## Paste this

Run from the **root of the project that will use the flow** (not the worktree
the flow docs live in). Nothing to fill.

```text
Bootstrap this project to run the RDR flow, non-invasively. The project must
NOT gain any committed knowledge of the RDR process: do NOT edit CLAUDE.md, do
NOT edit the project's root .gitignore, do NOT add tracked files.

1. Keep _rdr/ out of git. Run `git check-ignore _rdr`. If it is already
   ignored (e.g. an `_*` rule), do nothing. If NOT ignored, create
   `_rdr/.gitignore` containing a single line `*` — this ignores _rdr's own
   contents from inside, touching no project-level file.

2. Write `_rdr/rdr-env.md` — the PATH MAP — inferring values from this project:
   - Output staging keys {SPIKE_DIR}, {FLOW_DIR}, {ARTIFACT_DIR}. Default
     {SPIKE_DIR}=_rdr/spikes/<rdr-slug>/, {FLOW_DIR}=_rdr/<lens>/<rdr-slug>/
     — lens-first, then a per-RDR folder under it (e.g.
     _rdr/3amigo/0038-policies/ holds persona-1-pm.md, persona-2-implementer.md,
     persona-3-qa.md, consolidation.md). Each lens fills <lens> with its own
     name and writes its element files there; both gitignored scratch.
     {ARTIFACT_DIR}= wherever this project keeps RDRs,
     beside the RDR (tracked). Find the RDR directory by locating existing RDR
     files or asking me if none exist yet.
   - Reuse-audit source paths: inspect the source tree and list the
     highest-traffic / most-central modules to check first for a "does the code
     already do this?" reuse audit (Stage 4 reads these).

3. Write `_rdr/rdr-resources.md` — the EVIDENCE INDEX — inferring from this
   project:
   - Domain priors: the product principles doc, the feature/scope surface, the
     user-mental-model docs, and any prior-art/competitor set this project
     compares itself against. (Propose/Refine read this section.)
   - Search corpora: any arc corpora available for this project (run
     `arc:config` / list corpora) tagged by purpose (docs/standards, dependency
     source, peer-tool source, literature, market).
   - Design docs: the authoritative contract docs every fix is checked against.
   - External anchors: standards / canonical references cited by name.
   - (Optional) Alt-model roster: alternate base models for the dual-model
     (critique) and repeatability lenses, each with the exact command to
     relaunch the CLI on it (e.g. `ollama launch claude --model
     kimi-k2.6:cloud`). Omit if only one model is available.

   Keep both files DATA-ONLY: a one-line role banner (pointing to the flow
   README for rationale) + the section tables above + a one-line maintenance
   comment. No explanatory prose — the rationale lives in the README, not the
   per-project file. Fill concrete values where you can verify them from the
   project; leave a clearly-marked TODO only where a value needs my judgment.
   Flag the filled files for my review at the end.

4. Install the WORKSPACE MARKER so every repo/worktree resolves the seam without
   probing _rdr/. Find the workspace parent (the dir holding this project) with
   `WS=$(dirname "$(dirname "$(cd "$(git rev-parse --git-common-dir)" && pwd
   -P)")")`. If `$WS/.rdr-workspace` already exists, leave it. Otherwise **copy
   the tracked template** `flow/context/workspace.example` (in this workspace's
   flow repo — resolve via `$WS/flow/context/workspace.example`) to
   `$WS/.rdr-workspace`, then fill its repo-root and seam blocks for this
   machine: keep the repo roots that exist, point `RDR_ENV` / `RDR_RESOURCES` at
   the seam files just written (the `_rdr/` ones, or the pinned location if this
   project pins one). If no flow repo / template is reachable, hand-write an
   equivalent sourceable file exporting at least `RDR_ENV` / `RDR_RESOURCES`
   anchored on `$WS`. The marker lives ABOVE the repos (not inside one) so it
   survives worktrees; it is per-machine working config, not tracked.

Report: which ignore path was taken (already-ignored vs wrote _rdr/.gitignore),
the two files written, whether a workspace marker was created or already present,
and any TODO I must resolve.
```

## Review gate

- **Zero project footprint?** Confirm no change to `CLAUDE.md`, the project's
  root `.gitignore`, or any tracked file. `git status` should show nothing
  staged (the `_rdr/` tree is ignored).
- **Are the inferred values real?** Spot-check that source paths exist and the
  named docs/corpora are present — an inferred `rdr-env.md` pointing at a
  missing module is worse than a TODO.
- **Both files present and structured** per the README keys.

## Advance when

`_rdr/rdr-resources.md` and `_rdr/rdr-env.md` exist at the project root, are
gitignored, and carry project-appropriate values (or explicit TODOs you accept).

→ Next: [01-seed.md](01-seed.md) — start the first RDR.
