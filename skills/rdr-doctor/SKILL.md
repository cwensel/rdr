---
name: rdr-doctor
argument-hint: (none — run from any repo/worktree in the workspace)
description: Use to verify the RDR seam and engine are wired correctly (e.g. "check my RDR setup", "/rdr-doctor", "is the RDR flow installed right", "diagnose rdr"). Read-only health check — confirms the five-var contract binds, the engine layout resolves (stages/, prompts/, skills/, TEMPLATE.md), the evidence root is reachable, and no consumer/skill symlink is broken. Writes nothing; reports PASS/WARN/FAIL + the one fix per failure. Run after /rdr-init, after an engine upgrade, or when a skill reports a `stopped:` seam error.
---

# rdr-doctor — verify the seam + engine

Read-only health check. Runs the same resolution every `/rdr-*` skill's §seam-bind
does, but checks **every** invariant (never stops at the first failure) and reports
each with the one command that fixes it. The answer to "a skill said `stopped:…` —
what's wrong?" It **writes nothing** — sources the marker, stats paths. Safe anytime,
from any repo or worktree.

## Run it — one Bash call

All checks live in a **script file** shipped with the engine (`doctor.sh`, beside this
SKILL.md). Run that file — do **not** paste its body into a shell (pasting a long
quoted block is what made earlier runs drop their labels). Resolve the engine the same
two ways §seam-bind does (the doctor can't use the seam to find itself), then run it:

```sh
GC=$(git rev-parse --git-common-dir 2>/dev/null) && GC=$(cd "$GC" && pwd -P)
WS=$(dirname "$(dirname "$GC")")
for H in "$CLAUDE_PLUGIN_ROOT" "$CODEX_PLUGIN_ROOT" "$WS/rdr"; do [ -f "$H/skills/rdr-doctor/doctor.sh" ] && { sh "$H/skills/rdr-doctor/doctor.sh"; break; }; done
```

Under Codex, if `$CODEX_PLUGIN_ROOT` is unset, use this skill's filesystem-backed
source locator instead: run the `doctor.sh` that sits beside this `SKILL.md`.

That is the whole invocation — one `sh <file>`, no embedded quoting to mistranscribe.
It prints the full report (pure ASCII, one line per check + a Verdict). POSIX `sh`, so
identical under bash (Linux) and zsh (macOS).

**Print the script's stdout verbatim inside a fenced block** — it is the report (pure
ASCII, one line per check + a Verdict). Do not reformat it into a table, re-run it to
"confirm", or paraphrase the lines — that's wasted turns. The first run is the answer;
if a line says FAIL, surface its fix as-is, don't improvise a repair.

**Corpora (optional, after a clean structural run).** If `arc` is present, list the
named corpora in `$RDR_RESOURCES` and note any that don't resolve as `[WARN]`
(degraded mode — research stages run hollow until built). Never a FAIL; skip if no `arc`.

**Domain priors (optional, after a clean structural run).** Read `$RDR_RESOURCES`
and check its *Domain priors* section: if it is absent, empty, or names no
competitor/prior-art/standard for the project's problem space, report `[WARN]
domain priors thin — Propose enumerates approaches from the model prior, not
prior art`. This is the breadth gap: with no priors to reach for, Propose invents
candidate approaches from training memory and Resolve can only verify the winner
of a race the strongest alternatives may have sat out. Never a FAIL (the priors
are the user's to populate); the one fix is "populate Domain priors in
`$RDR_RESOURCES` with the competitors/prior art for this domain."

## Review gate

- **Read-only?** No file/symlink written, nothing installed — the doctor diagnoses;
  `/rdr-init` (or the user) repairs.
- **Complete?** Every check ran; one FAIL never skips the rest.
- **Each FAIL names one fix?** No "might be" — the command.

## Next step (rdr-common §next-step)

- All PASS → `Next: /rdr-status`, or `/rdr-seed <idea>`.
- Any FAIL → run the named fix (usually `/rdr-init`), then re-run `/rdr-doctor`.
