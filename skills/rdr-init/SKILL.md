---
name: rdr-init
argument-hint: "[--interactive | --defaults | --workspace | --reconfigure]   # run from the consumer project root"
description: >
  Use ONCE to bind a project to the RDR flow so the /rdr-* skills can resolve
  their paths (e.g. "set up RDR for this project", "/rdr-init", "bootstrap the
  RDR flow here"). Stage 0 — writes the seam data files (default .rdr/env.md /
  .rdr/resources.md), the seam marker, and the RDR home's index README (creating
  RDR_RECORDS if absent). Defaults to a repo-local seam (this repo's own RDR env);
  `--workspace` writes/joins a shared seam several sibling repos inherit. Bare
  run is smart: infers locations, asks only when it genuinely can't, discloses
  the choices; `--interactive` forces the questions, `--defaults` is silent,
  `--reconfigure` changes an existing seam. Non-invasive to the consumer's
  source tree (no CLAUDE.md or root .gitignore edits); the only tracked file it
  adds is the index README inside RDR_RECORDS. Pairs with /rdr-seed (next).
---

# rdr-init — Stage 0 (Bootstrap)

One-time seam setup the other `/rdr-*` skills depend on, leaving the consumer's
**source tree** ignorant of RDR. Writes the path map + evidence index (default
`.rdr/env.md` / `.rdr/resources.md`), the seam marker (lets every skill resolve the
engine + paths from any cwd/worktree), and the RDR home's index `README.md`.

**Scope.** Creates the untracked seam + marker, the RDR home, and its index
README (the one tracked file it adds — inside `$RDR_RECORDS`, not the consumer's code;
the flow needs it as its index/Status table). Does **not** install the `/rdr-*`
symlink farm (consumer-owned) and never copies the engine's `TEMPLATE.md`/prompts
(read live from `$RDR_HOME`).

## Usage

```
/rdr-init                 # repo-local (default); smart — infer locations, ask only if ambiguous, disclose
/rdr-init --interactive   # force the questions (records/evidence, tracked vs gitignored)
/rdr-init --defaults      # silent: repo-local defaults, no questions (scripted)
/rdr-init --workspace     # write/join the SHARED $WS seam several sibling repos inherit
/rdr-init --reconfigure   # seam already exists: change its location choices (interactive; migrates if RDRs exist)
```

**Scope is repo-local by default** — binds **only this repo**: `$PROJECT/.rdr/workspace`,
with records + evidence inside it. That's the common single-repo case. `--workspace`,
run from the **parent**, instead writes/joins a shared `$WS/.rdr-workspace` that every
sibling repo under `$WS` inherits (the multi-repo arrangement, e.g. `newcoinc/`'s
retrofit/process/flow). Because the seam is a path map, the shared scope also lets
`RDR_RECORDS` / `RDR_EVIDENCE` point at **their own repos**, separate from the code repo.
Repo-local **overrides** a shared marker via §seam-bind's nearest-wins, and **never
touches** it.

**Modes** — the no-flag run is the common path; flags override its prompting:

| Mode | When it asks | Use for |
| --- | --- | --- |
| *(bare)* **smart** | only on a genuine fork — a location it can't infer, **or** a shared workspace marker already covers this repo (offer to inherit vs keep repo-local) | the default — repo-local, fast, never guesses silently |
| `--interactive` | always — records/evidence/tracked placement | deliberate first-run setup |
| `--defaults` | never | CI / scripted setup |
| `--workspace` | only an un-inferable location | sibling repos that deliberately **share** one seam |
| `--reconfigure` | always, against the **existing** marker | changing placement after the fact (see *Reconfigure* below) |

**Location defaults**: seam at gitignored `.rdr/`; `RDR_RECORDS` under the project;
`RDR_EVIDENCE=$RDR_RECORDS`. Every mode **discloses** the resolved scope + locations +
overrides in its closing report.

**Run from inside the project that will use the flow** — never globally. `/rdr-init`
writes a *per-project* seam (the marker + `.rdr/` files), so it requires a consumer
git repo as cwd. It is the same skill whether the engine is a marketplace plugin
(user or project scope) or a clone sitting beside your repos — that only changes
where the *engine* lives, never that init runs **in the project**.
The Step-1 preflight enforces this: it hard-stops outside a git repo, inside the
engine repo, or inside the installed plugin dir. Not a worktree; the project root.

1. **Inverse seam-bind.** Unlike every other `/rdr-*` skill, this one does **not**
   run `rdr-common` §seam-bind — §seam-bind *requires* the marker, and creating it
   is this skill's job. **Preflight first** (this skill writes a per-project seam, so
   it must run *inside the consumer project* — never globally, never in the engine).
   Hard-stop if either guard trips; do not proceed to write anything:

   ```sh
   # Guard 1 — must be inside a git repo (a project), not $HOME / ~/.claude / loose cwd.
   # Resolve in two steps: an empty $(git …) would make `cd ""` a silent no-op, so test
   # the git exit first, then the cd.
   GC=$(git rev-parse --git-common-dir 2>/dev/null) || { echo "stopped:not-in-a-project — cd into the project that will use the flow, then /rdr-init"; exit 1; }
   GIT_COMMON=$(cd "$GC" && pwd -P) || { echo "stopped:not-in-a-project — git dir unreadable"; exit 1; }
   PROJECT=$(dirname "$GIT_COMMON")                # this project's root
   WS=$(dirname "$PROJECT")                         # workspace root (holds sibling repos)
   export PROJECT WS                                # so a sourced marker can anchor on either
   # Guard 2 — refuse to run inside the engine itself (it is not a consumer).
   if [ -d "$PROJECT/stages" ] && [ -d "$PROJECT/skills" ] && [ -f "$PROJECT/TEMPLATE.md" ]; then
     echo "stopped:in-engine-repo — /rdr-init binds a CONSUMER project, not the RDR engine; cd into your project"; exit 1
   fi
   # Guard 3 — refuse if cwd is the installed plugin dir (engine shipped via marketplace).
   case "$PROJECT" in *"/.claude/plugins/"*|*"/.codex/plugins/"*|*"/.agents/plugins/"*) echo "stopped:in-plugin-dir — /rdr-init runs in your project, not the installed engine"; exit 1;; esac

   # Detect BOTH scopes — repo-local (inside .rdr/, this repo's own env) and workspace.
   [ -f "$PROJECT/.rdr/workspace" ] && LOCAL_MARKER=1 || LOCAL_MARKER=
   [ -f "$WS/.rdr-workspace" ]      && WS_MARKER=1    || WS_MARKER=
   ```

   **Branch on marker state × mode** (scope defaults to **repo-local**):
   - **A repo-local marker already exists** (`$PROJECT/.rdr/workspace`), no
     `--reconfigure` → do not rewrite seam/env/resources. Verify + report the
     existing five-var contract, then still offer the SessionStart seam hook if
     the active harness hook config is absent.
   - **`--workspace`** → write/join the shared `$WS/.rdr-workspace` (create it, or if it
     exists and is complete, stop and report this repo inherits it). For sharing siblings.
   - **No repo-local marker, but a shared workspace marker exists** (the `intrastate`
     case) → the **scope fork**. Default is still repo-local, but a shared seam already
     covers this repo, so surface it:
     - **`--defaults`** → take the repo-local default silently (own seam).
     - **bare / `--interactive`** → **ask once**: *"A shared workspace seam at
       `$WS/.rdr-workspace` already covers this repo. Give `<repo>` its own repo-local
       RDR env (default), or inherit the shared one?"* Own = write the repo-local marker;
       inherit = write nothing, stop, report it binds to the shared seam.
   - **Neither marker exists** → normal first-run: write a **repo-local** marker
     (`--workspace` overrides to shared). Prompting for *locations* follows the mode (step 2).
   - **`--reconfigure`** → see *Reconfigure* below.

   **Where the new marker goes, by scope:** repo-local (default) →
   `$PROJECT/.rdr/workspace` — **inside the already gitignored `.rdr/`**, so no
   project-level `.gitignore` edit, anchored at `$PROJECT` so worktrees resolve it.
   Workspace (`--workspace`) → `$WS/.rdr-workspace`, shared above the repos. A repo-local
   marker **overrides** a shared one via §seam-bind's nearest-wins — it never reads or
   rewrites the shared marker.

   **Resolve `RDR_HOME`** (the engine root: `stages/`, `prompts/`, `skills/`,
   `TEMPLATE.md`) — first that binds, else `stopped:` and ask:
   - **Claude plugin install** — `[ -d "$CLAUDE_PLUGIN_ROOT/stages" ]` →
     `$CLAUDE_PLUGIN_ROOT` (write the resolved absolute path; the cache dir is
     version-stamped, so re-run `/rdr-init` after a plugin upgrade);
   - **Codex plugin install** — use `$CODEX_PLUGIN_ROOT` if the harness provides it;
     otherwise use the filesystem-backed source locator for this `SKILL.md`, walk up
     to the plugin root, and accept it only if it contains `stages/`, `prompts/`,
     `skills/`, and `TEMPLATE.md` (write that resolved absolute path; re-run
     `/rdr-init` after a plugin upgrade);
   - **cloned beside your repos** — else `[ -d "$WS/rdr/stages" ]` → `$WS/rdr` (the
     `rdr` repo sits next to this one).

2. **Decide the three locations, per mode**, then run the stage. The choices are:
   *seam* (`RDR_ENV`/`RDR_RESOURCES` — gitignored `.rdr/` vs a tracked dir),
   *records* (`RDR_RECORDS`), and *evidence* (`RDR_EVIDENCE` — beside records vs its
   own dir/repo).
   - **`--defaults`** → take all three defaults, no questions.
   - **bare (smart)** → infer from repo signals (an existing `rdr/cli/`-style tree ⇒
     records there; an existing seam dir ⇒ reuse it); take the default for anything
     clear; **ask only** for a value with no inferable answer (e.g. no RDR dir exists
     yet and none is implied). Never guess silently on a genuine fork.
   - **`--interactive`** → ask all three regardless of inference, defaults pre-filled.
   Once decided, **run** [`00-bootstrap.md`](00-bootstrap.md) — its *Paste this* block
   is the authoring contract. In order, it:
   - keeps `.rdr/` out of git (touching no project-level file),
   - writes `.rdr/env.md` (the PATH MAP) inferred from this project,
   - writes `.rdr/resources.md` (the EVIDENCE INDEX), **probing which named arc
     corpora actually resolve** — missing ones get a degraded-mode TODO (research
     stages run hollow until built; Propose/Refine still work on doc priors),
   - installs the MARKER from [`workspace.example`](workspace.example), filling the
     five contract vars (`RDR_HOME` per step 1; `RDR_EVIDENCE` defaults to
     `$RDR_RECORDS`) — at `$PROJECT/.rdr/workspace` (repo-local default, auto-ignored by
     `.rdr/.gitignore`) or `$WS/.rdr-workspace` (`--workspace`, shared),
   - scaffolds the RDR home: `mkdir -p "$RDR_RECORDS"` and, if no index yet, copies
     [`RDR-HOME-README.template.md`](RDR-HOME-README.template.md) →
     `$RDR_RECORDS/README.md` (the only engine file vendored in),
   - **offers** (never auto-installs) the SessionStart seam hook
     [`rdr-seam-context.sh.template`](rdr-seam-context.sh.template) — on yes:
     Claude Code uses `.claude/hooks/rdr-seam-context.sh` + `.claude/settings.json`;
     Codex uses `.codex/hooks/rdr-seam-context.sh` + `.codex/hooks.json`
     (`SessionStart`, matcher `startup|resume|clear|compact`) and requires `/hooks`
     trust review.

   Fill concrete values where verifiable from the project; leave a marked TODO only
   where a value needs the user's judgment. **Author in the main context** — do not
   delegate the file writes to a sub-agent.

3. **Verify the seam resolves.** After writing, source the new marker and confirm
   the full five-var contract binds — this is exactly what every later `/rdr-*`
   skill's §seam-bind will check:

   ```sh
   . "${RDR_MARKER:-$WS/.rdr-workspace}"   # the marker just written (repo-local or workspace)
   [ -n "$RDR_HOME" ]      && [ -d "$RDR_HOME/stages" ]     || echo "stopped:RDR_HOME-unset-or-missing"
   [ -n "$RDR_RECORDS" ]   && [ -d "$RDR_RECORDS" ]         || echo "stopped:RDR_RECORDS-unset-or-missing"
   [ -n "$RDR_EVIDENCE" ]                                   || echo "stopped:RDR_EVIDENCE-unset"
   [ -n "$RDR_ENV" ]       && [ -f "$RDR_ENV" ]             || echo "stopped:RDR_ENV-unset-or-missing"
   [ -n "$RDR_RESOURCES" ] && [ -f "$RDR_RESOURCES" ]       || echo "stopped:RDR_RESOURCES-unset-or-missing"
   ```
   `$RDR_RECORDS` is created by step 2's scaffold (with the index `README.md`), so it
   must bind here as a dir holding that README — unless you deliberately deferred it.
   (`/rdr-doctor` runs this same contract check, plus symlink/layout checks, anytime
   after — point the user there if a later skill reports a `stopped:` seam error.)

4. **Disclose the choices** (every mode, even `--defaults`). End by stating the scope,
   the three resolved locations, and the one-line override, so the decision is never silent:

   ```text
   Scope:    repo-local | workspace (shared) — marker at <RDR_MARKER>
   Engine:   <RDR_HOME>                      — plugin, or the rdr repo cloned beside your repos
   Seam:     .rdr/  (gitignored)            — to track: point RDR_ENV/RDR_RESOURCES at a tracked dir
   Records:  <RDR_RECORDS>                  — change: /rdr-init --reconfigure
   Evidence: <RDR_EVIDENCE> (= records)     — change: /rdr-init --reconfigure
   ```

## Reconfigure — `/rdr-init --reconfigure`

Changes an **existing** marker's locations after the fact (the only mode that
re-decides a complete seam). Always interactive: re-ask the three location choices
(records, evidence, seam tracked-vs-gitignored) with the *current* values pre-filled,
then rewrite **only** the changed vars in `$WS/.rdr-workspace` (leave `RDR_HOME` and
unchanged vars alone). It changes *where things live*, never RDR content.

**Migration guard.** If RDRs already exist under the old `RDR_RECORDS` (or evidence
under the old `RDR_EVIDENCE`) and that location is changing, do **not** silently
re-point — the existing files would orphan. Either: offer to `git mv` / move them to
the new location and report what moved, or, if a move is unsafe (cross-repo,
uncommitted changes), **stop** with `stopped:reconfigure-needs-migration` naming the
dirs to move by hand. Re-run `/rdr-doctor` after to confirm the new layout binds.

## Review gate (Stage `00-bootstrap.md`)

- **Ran inside a consumer project?** Step-1 preflight passed — a git repo cwd, not
  the engine repo, not the installed plugin dir. If any guard tripped, nothing was
  written; that is correct.
- **No footprint on the consumer's source?** No change to `CLAUDE.md`, `AGENTS.md`,
  or root `.gitignore`; `.rdr/` ignored. Only tracked adds: the RDR-home
  `README.md` and, if accepted, the opt-in `.claude/` or `.codex/` hook config.
- **Inferred values real?** Source paths exist; named docs/corpora present (missing
  corpora carry the degraded-mode TODO, not a silent dead reference).
- **Scope honored?** Repo-local (default) wrote `$PROJECT/.rdr/workspace` and **left any
  shared `$WS/.rdr-workspace` untouched**; `--workspace` wrote/kept `$WS`'s. The report
  names which.
- **Marker binds?** Sourcing the resolved marker exports the five contract vars;
  `RDR_HOME` resolved plugin-first, else sibling; `RDR_EVIDENCE` set (defaults to
  `$RDR_RECORDS`).
- **RDR home scaffolded?** `$RDR_RECORDS` holds an index `README.md`; no other engine
  file vendored in.
- **Choices disclosed / mode honored?** The report names the three resolved locations +
  overrides. `--defaults` asked nothing; `--interactive`/`--reconfigure` asked; bare
  asked only an un-inferable value. `--reconfigure` migrated existing RDRs or stopped
  for a manual move — never orphaned them.

## Next step (rdr-common §next-step)

- `Next: /rdr-seed <idea>` — start the first RDR now that the seam resolves.
- `/rdr-doctor` — verify the full five-var contract + engine layout in one read-only
  pass (use it if any later skill reports a `stopped:` seam error).
- A bare re-run on an existing seam is idempotent: it verifies + reports and may
  offer missing SessionStart hook setup, but does not rewrite seam/env/resources.
  Use `--reconfigure` to change where records/evidence/seam live.
