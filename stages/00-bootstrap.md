# Stage 0 — Bootstrap (one-time per project; `/rdr-init`)

**Goal**: set a project up to run the RDR flow **without the project knowing
about RDR** — no versioned footprint, no `CLAUDE.md` edits, nothing in the
project's own `.gitignore`. The flow expects two data files — the evidence index
and the path map — defaulting to `.rdr/resources.md` and `.rdr/env.md` (no `rdr-`
prefix: the `.rdr/` dir already scopes them; a *pinned* seam in a shared dir uses
`rdr-resources.md` / `rdr-env.md` so the names stay self-describing). This stage
creates them, filled from the project itself, and ensures `.rdr/` stays out of git.
Run once; re-run only to refresh values.

> **Already have a pinned seam?** A project may keep its seam (and evidence) in
> a tracked external directory instead of gitignored `.rdr/` — e.g. a consumer
> that keeps `rdr-resources.md` / `rdr-env.md` in a sibling seam repo (such as a
> `flow/context/`), with evidence under that repo's `rdr/evidence/`. If a
> **workspace marker**
> (`.rdr-workspace` at the workspace parent, see the README *Where the seam
> lives*) already exports `$RDR_RESOURCES` / `$RDR_ENV`, **skip this stage** —
> the resolver finds them. Bootstrap only creates the fallback `.rdr/` seam (and
> a marker pointing at it) for a project that has none.

## Paste this

Run from the **root of the project that will use the flow** (not the worktree
the flow docs live in). Nothing to fill.

```text
Bootstrap this project to run the RDR flow, non-invasively. The project must
NOT gain any committed knowledge of the RDR process: do NOT edit CLAUDE.md, do
NOT edit the project's root .gitignore, do NOT add tracked files.

1. Keep .rdr/ out of git. Run `git check-ignore .rdr`. If it is already
   ignored (e.g. an `_*` rule), do nothing. If NOT ignored, create
   `.rdr/.gitignore` containing a single line `*` — this ignores .rdr's own
   contents from inside, touching no project-level file.

2. Write `.rdr/env.md` — the PATH MAP — inferring values from this project:
   - Output staging keys {SPIKE_DIR}, {EVIDENCE_DIR}, {ARTIFACT_DIR}. Both evidence
     and artifacts are **per-RDR-first and same-shaped** — each RDR owns one folder,
     evidence keyed by lens inside it:
       {ARTIFACT_DIR}=<RDR_RECORDS>/<rdr-slug>/artifacts/  (impl output, tracked beside the RDR)
       {EVIDENCE_DIR}=<RDR_EVIDENCE>/<rdr-slug>/evidence/<lens>/
       {SPIKE_DIR}=<RDR_EVIDENCE>/<rdr-slug>/evidence/spikes/
     The evidence keys hang under `RDR_EVIDENCE` (the marker var, step 4), the
     artifact key under `RDR_RECORDS`; **when RDR_EVIDENCE defaults to RDR_RECORDS the
     two are siblings** under one `<rdr-slug>/` folder (e.g.
     <RDR_RECORDS>/0038-policies/{artifacts/, evidence/3amigo/}). Each lens fills
     <lens> with its own name and writes its element files there. When a consumer
     points RDR_EVIDENCE at its own dir/repo, the same `<rdr-slug>/evidence/` shape
     lives there instead — keeping evidence isolated and self-identifying for
     commits. Find the RDR directory by locating existing RDR files or asking me if
     none exist yet.
   - Reuse-audit source paths: inspect the source tree and list the
     highest-traffic / most-central modules to check first for a "does the code
     already do this?" reuse audit (Stage 4 reads these).

3. Write `.rdr/resources.md` — the EVIDENCE INDEX — inferring from this
   project:
   - Domain priors: the product principles doc, the feature/scope surface, the
     user-mental-model docs, and any prior-art/competitor set this project
     compares itself against. (Propose/Refine read this section.)
   - Search corpora: any arc corpora available for this project (run
     `arc:config` / list corpora) tagged by purpose (docs/standards, dependency
     source, peer-tool source, literature, market). **Probe what actually
     resolves**, don't just copy a peer project's list: list the corpora arc
     reports present, and for any you name in this section, confirm it exists. If
     few/none exist (the common case for a fresh consumer), say so plainly and
     record a **degraded-mode TODO**: the research-touching stages (Resolve /
     Reconcile / Finalize) will run hollow until these corpora are built; Propose /
     Refine still work (they read only the doc-based Domain-priors section). Name
     the missing corpora so the user knows what to build.
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

4. Install the SEAM MARKER so every cwd/worktree resolves the seam. Resolve topology
   first (split form — never `cd "$(git …)"`, an empty capture makes `cd` a silent
   no-op):
   `GC=$(git rev-parse --git-common-dir) || stop; GC=$(cd "$GC" && pwd -P);`
   `PROJECT=$(dirname "$GC"); WS=$(dirname "$PROJECT")`.
   **Pick the marker by scope (default repo-local):**
   - **repo-local (default)** → `$PROJECT/.rdr/workspace`. Inside the already
     gitignored `.rdr/`, so no project-level `.gitignore` edit; anchored at
     `$PROJECT` so worktrees resolve it. Anchor the marker's paths on `$PROJECT`.
   - **workspace (`--workspace`)** → `$WS/.rdr-workspace`, shared above the repos,
     anchored on `$WS`. If it already exists, leave it (just verify it exports
     `RDR_HOME`; add a missing var rather than rewriting).
   **Copy the tracked template** `workspace.example` (symlinked beside this skill) to
   the chosen path and fill its blocks (set `WS`/`PROJECT` at the top to match the
   anchor). A repo-local marker **never reads or writes** the shared one.
   Fill the **five-var engine contract** (all required — the skills read only these):
   - **`RDR_HOME`** — the RDR engine (holds `stages/`, `prompts/`, `skills/`,
     `TEMPLATE.md`). Resolve by install shape, first that binds:
     **(a) plugin** — if `$CLAUDE_PLUGIN_ROOT` is set and `$CLAUDE_PLUGIN_ROOT/stages`
     exists, the engine ships inside the plugin: use that resolved absolute path
     (the cache dir is version-stamped — re-run `/rdr-init` after a plugin upgrade);
     **(b) sibling repo** — else the engine is a sibling checkout: `$WS/rdr` (assert
     `$WS/rdr/stages`). If neither binds, stop and ask for the engine path.
   - **`RDR_RECORDS`** — this project's RDR-instances directory (where `NNNN-slug.md`
     RDRs + their artifact subdirs live). This is the single source of truth for
     "where do my RDRs live" — the same directory `{ARTIFACT_DIR}` in `{RDR_ENV}`
     sits under. A default `.rdr/` project sets it under its own root; a pinned
     project points at its tracked RDR tree (e.g. `<repo>/rdr/cli`).
   - **`RDR_EVIDENCE`** — this project's evidence root (the base `{EVIDENCE_DIR}` /
     `{SPIKE_DIR}` hang under). Default to `$RDR_RECORDS` (evidence beside the
     records); point it elsewhere only if the consumer stages lens/spike output in
     its own dir or a separate repo. Set it even when defaulting, so the contract is
     explicit.
   - **`RDR_ENV` / `RDR_RESOURCES`** → the seam files just written (the `.rdr/`
     ones, or the pinned location if this project pins one).
   Plus any per-consumer repo-root conveniences (not load-bearing). If no template is
   reachable, hand-write an equivalent sourceable file exporting at least those five
   contract vars, anchored on the scope's root (`$PROJECT` repo-local, `$WS`
   workspace). Both are per-machine working config, untracked: repo-local under the
   gitignored `.rdr/`; workspace above the repos (a worktree-surviving location).

5. Scaffold the RDR home. The marker's `RDR_RECORDS` is where the `NNNN-slug.md`
   RDRs live and is the single source of truth for it; a brand-new consumer has
   no such directory and, crucially, no index README — the file the flow reads as
   the index + Status/Priority table (`/rdr-finalize` adds/updates each RDR's row
   and keeps its Status column current). Nothing else creates it, so
   do it here: `mkdir -p "$RDR_RECORDS"`, then if `$RDR_RECORDS/README.md` is absent, copy
   the engine template `$RDR_HOME/RDR-HOME-README.template.md` to it and fill
   `{PROJECT}`. This is the ONLY engine file vendored into the consumer —
   `TEMPLATE.md` and the prompts stay read-in-place from `$RDR_HOME` (copying
   them would drift; see `skills/rdr-common.md`). Leave an existing
   `README.md` untouched.

6. OFFER (do not auto-install) the SessionStart seam hook. The `/rdr-*` skills
   work without it — they run §seam-bind each call — but a consumer can pre-resolve
   the seam once per session so they take the fast path. This adds a `.claude/`
   footprint (a hook script + a `settings.json` entry), so it is opt-in: DESCRIBE
   it and let me say yes, don't write it silently. If I accept, copy the engine
   template `$RDR_HOME/rdr-seam-context.sh.template` to the consumer's
   `.claude/hooks/rdr-seam-context.sh` (`chmod +x`), and add a `SessionStart` entry
   to `.claude/settings.json` that runs it (use the update-config skill for the
   settings edit). If I decline, note it as an available later step and move on.

Report: which ignore path was taken (already-ignored vs wrote .rdr/.gitignore),
the two files written, whether a workspace marker was created or already present
(and whether it exports the five contract vars `RDR_HOME` / `RDR_RECORDS` /
`RDR_EVIDENCE` / `RDR_ENV` / `RDR_RESOURCES`), whether `$RDR_RECORDS` and its `README.md` index were
created or already present, how `RDR_HOME` resolved (plugin vs sibling),
which named corpora exist vs are missing (the degraded-mode TODO), whether the
SessionStart hook was offered/installed/declined, and any TODO I must resolve.
```

## Review gate

- **Zero footprint on the consumer's source tree?** Confirm no change to
  `CLAUDE.md`, the project's root `.gitignore`, or any tracked file in the
  consumer's code. The seam (`.rdr/`) is ignored; the one tracked file Stage 0
  may add is the RDR-home index `README.md`, and it lives inside `$RDR_RECORDS` (the
  RDR directory), never in the consumer's source.
- **Are the inferred values real?** Spot-check that source paths exist and the
  named docs/corpora are present — an inferred `{RDR_ENV}` pointing at a
  missing module is worse than a TODO.
- **Both seam files present and structured** per the README keys; `$RDR_RECORDS`
  exists and holds an index `README.md` (scaffolded from the engine template or
  already present).

## Advance when

`.rdr/resources.md` and `.rdr/env.md` exist at the project root, are
gitignored, and carry project-appropriate values (or explicit TODOs you accept);
`$RDR_RECORDS` exists and holds an index `README.md` (scaffolded from the engine
template, or already present).

→ Next: [01-seed.md](01-seed.md) — start the first RDR.
