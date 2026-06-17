---
name: rdr-init
argument-hint: (none — run from the root of the project that will use the flow)
description: Use ONCE to bind a project to the RDR flow — bootstrap the seam so the /rdr-* skills can resolve their paths (e.g. "set up RDR for this project", "/rdr-init", "bootstrap the RDR flow here"). Runs Stage 0 — writes the project's rdr-env.md / rdr-resources.md seam files and installs the worktree-invariant workspace marker. Non-invasive: no CLAUDE.md edits, no tracked files, nothing in the project's .gitignore. Pairs with /rdr-seed (next — start the first RDR).
---

# rdr-init — Stage 0 (Bootstrap)

Bind a project to the RDR flow **without the project knowing about RDR**. This is
the one-time seam setup the other `/rdr-*` skills depend on: it writes the path
map (`rdr-env.md`), the evidence index (`rdr-resources.md`), and the workspace
marker that lets every skill resolve `$RDR_FLOW_HOME` (the engine) and `$RDR_ENV`
(this project's paths) from any cwd or worktree.

**Scope: seam-binding only.** This skill does **not** install the `/rdr-*` skill
symlink farm under the consumer's `.claude/skills/` — that is a separate, tracked
step the consumer owns (it would add a versioned footprint, which Stage 0 avoids).
`/rdr-init` only creates the (untracked) seam + marker.

## Usage

```
/rdr-init
```

Run from the **root of the project that will use the flow** (not a worktree, not
the engine repo).

1. **Inverse seam-bind.** Unlike every other `/rdr-*` skill, this one does **not**
   run `rdr-common` §seam-bind — §seam-bind *requires* the marker, and creating it
   is this skill's job. Derive only the workspace root from git topology (the same
   first lines §seam-bind uses), then check for an existing seam:

   ```sh
   GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" && pwd -P)
   WS=$(dirname "$(dirname "$GIT_COMMON")")        # workspace root (holds sibling repos)
   PROJECT=$(dirname "$GIT_COMMON")                # this project's root
   if [ -f "$WS/.rdr-workspace" ]; then
     . "$WS/.rdr-workspace"
     # Marker present. Verify the four-var contract is complete — older markers
     # may pre-date the engine split (no RDR_FLOW_HOME) or the RDR_DIR var; if any
     # is missing, ADD it and re-export rather than rewriting the whole marker.
     for v in RDR_FLOW_HOME RDR_DIR RDR_ENV RDR_RESOURCES; do
       eval "[ -n \"\$$v\" ]" || echo "marker missing: $v"
     done
   fi
   ```

   If the marker already exists and exports all four contract vars, the seam is set
   up — report it and **stop** rather than overwrite. Only write/amend when the seam
   (or one of the contract vars) is missing.

2. **Run the stage** [`00-bootstrap.md`](00-bootstrap.md) — its *Paste this* block
   is the authoring contract. In order, it:
   - keeps `_rdr/` out of git (touching no project-level file),
   - writes `_rdr/rdr-env.md` (the PATH MAP) inferred from this project,
   - writes `_rdr/rdr-resources.md` (the EVIDENCE INDEX) inferred from this project,
   - installs the WORKSPACE MARKER at `$WS/.rdr-workspace` from the template
     [`workspace.example`](workspace.example) (symlinked beside this skill),
     filling `RDR_FLOW_HOME` (**required** — the engine repo), the consumer repo
     root(s), and `RDR_ENV` / `RDR_RESOURCES`.

   Fill concrete values where verifiable from the project; leave a marked TODO only
   where a value needs the user's judgment. **Author in the main context** — do not
   delegate the file writes to a sub-agent.

3. **Verify the seam resolves.** After writing, source the new marker and confirm
   the full four-var contract binds — this is exactly what every later `/rdr-*`
   skill's §seam-bind will check:

   ```sh
   . "$WS/.rdr-workspace"
   [ -n "$RDR_FLOW_HOME" ] && [ -d "$RDR_FLOW_HOME/flow" ] || echo "stopped:RDR_FLOW_HOME-unset-or-missing"
   [ -n "$RDR_DIR" ]       && [ -d "$RDR_DIR" ]            || echo "stopped:RDR_DIR-unset-or-missing"
   [ -n "$RDR_ENV" ]       && [ -f "$RDR_ENV" ]            || echo "stopped:RDR_ENV-unset-or-missing"
   [ -n "$RDR_RESOURCES" ] && [ -f "$RDR_RESOURCES" ]      || echo "stopped:RDR_RESOURCES-unset-or-missing"
   ```
   (`$RDR_DIR` may legitimately not exist yet for a brand-new project — if so,
   create it, or note it as the dir `/rdr-seed` will populate.)

## Review gate (Stage `00-bootstrap.md`)

- **Zero project footprint?** No change to `CLAUDE.md`, the project's root
  `.gitignore`, or any tracked file (`git status` shows nothing staged — `_rdr/`
  is ignored).
- **Are the inferred values real?** Source paths exist; named docs/corpora are
  present. An `rdr-env.md` pointing at a missing module is worse than a TODO.
- **Marker complete?** `$WS/.rdr-workspace` exports `RDR_FLOW_HOME` (engine),
  `RDR_ENV`, `RDR_RESOURCES`; a fresh `. .rdr-workspace` binds them.

## Next step (rdr-common §next-step)

- `Next: /rdr-seed <idea>` — start the first RDR now that the seam resolves.
- Re-run only to refresh values; the marker, once present, is left in place.
