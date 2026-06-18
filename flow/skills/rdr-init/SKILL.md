---
name: rdr-init
argument-hint: (none — run from the root of the project that will use the flow)
description: Use ONCE to bind a project to the RDR flow so the /rdr-* skills can resolve their paths (e.g. "set up RDR for this project", "/rdr-init", "bootstrap the RDR flow here"). Stage 0 — writes the rdr-env.md / rdr-resources.md seam files, the worktree-invariant workspace marker, and the RDR home's index README (creating RDR_DIR if absent). Non-invasive to the consumer's source tree (no CLAUDE.md or root .gitignore edits); the only tracked file it adds is the index README inside RDR_DIR. Pairs with /rdr-seed (next).
---

# rdr-init — Stage 0 (Bootstrap)

One-time seam setup the other `/rdr-*` skills depend on, leaving the consumer's
**source tree** ignorant of RDR. Writes the path map (`rdr-env.md`), evidence
index (`rdr-resources.md`), the workspace marker (lets every skill resolve the
engine + paths from any cwd/worktree), and the RDR home's index `README.md`.

**Scope.** Creates the untracked seam + marker, the RDR home, and its index
README (the one tracked file it adds — inside `$RDR_DIR`, not the consumer's code;
the flow needs it as its index/Status table). Does **not** install the `/rdr-*`
symlink farm (consumer-owned) and never copies the engine's `TEMPLATE.md`/prompts
(read live from `$RDR_FLOW_HOME`).

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

   **Resolve `RDR_FLOW_HOME`** (the engine: `flow/`, `prompts/`, `skills/`,
   `TEMPLATE.md`) — first that binds, else `stopped:` and ask:
   - **plugin** — `[ -d "$CLAUDE_PLUGIN_ROOT/flow" ]` → `$CLAUDE_PLUGIN_ROOT`
     (write the resolved absolute path; the cache dir is version-stamped, so re-run
     `/rdr-init` after a plugin upgrade);
   - **sibling repo** — else `[ -d "$WS/rdr/flow" ]` → `$WS/rdr`.

2. **Run the stage** [`00-bootstrap.md`](00-bootstrap.md) — its *Paste this* block
   is the authoring contract. In order, it:
   - keeps `_rdr/` out of git (touching no project-level file),
   - writes `_rdr/rdr-env.md` (the PATH MAP) inferred from this project,
   - writes `_rdr/rdr-resources.md` (the EVIDENCE INDEX), **probing which named arc
     corpora actually resolve** — missing ones get a degraded-mode TODO (research
     stages run hollow until built; Propose/Refine still work on doc priors),
   - installs the WORKSPACE MARKER at `$WS/.rdr-workspace` from
     [`workspace.example`](workspace.example), filling the four contract vars
     (`RDR_FLOW_HOME` resolved per step 1),
   - scaffolds the RDR home: `mkdir -p "$RDR_DIR"` and, if no index yet, copies
     [`RDR-HOME-README.template.md`](RDR-HOME-README.template.md) →
     `$RDR_DIR/README.md` (the only engine file vendored in),
   - **offers** (never auto-installs) the SessionStart seam hook
     [`rdr-seam-context.sh.template`](rdr-seam-context.sh.template) — on yes: copy
     to `.claude/hooks/rdr-seam-context.sh` (`chmod +x`) + add a `SessionStart`
     entry to `.claude/settings.json`; opt-in (adds a `.claude/` footprint).

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
   `$RDR_DIR` is created by step 2's scaffold (with the index `README.md`), so it
   must bind here as a dir holding that README — unless you deliberately deferred it.

## Review gate (Stage `00-bootstrap.md`)

- **No footprint on the consumer's source?** No change to `CLAUDE.md` or the root
  `.gitignore`; `_rdr/` ignored. Only tracked adds: the RDR-home `README.md` (in
  `$RDR_DIR`) and, if accepted, the opt-in `.claude/` seam hook + settings entry.
- **Inferred values real?** Source paths exist; named docs/corpora present (missing
  corpora carry the degraded-mode TODO, not a silent dead reference).
- **Marker binds?** A fresh `. .rdr-workspace` exports the four contract vars;
  `RDR_FLOW_HOME` resolved plugin-first, else sibling.
- **RDR home scaffolded?** `$RDR_DIR` holds an index `README.md`; no other engine
  file vendored in.

## Next step (rdr-common §next-step)

- `Next: /rdr-seed <idea>` — start the first RDR now that the seam resolves.
- Re-run only to refresh values; the marker, once present, is left in place.
