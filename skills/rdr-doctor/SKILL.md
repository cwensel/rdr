---
name: rdr-doctor
metadata:
  argument-hint: (none — run from any repo/worktree in the workspace)
description: 'Use to verify an RDR setup. Read-only health check for the seam, five-var contract, engine layout, evidence root, and skill links. Trigger for diagnose RDR, check RDR setup, $rdr-doctor, or /rdr-doctor.'
---

# rdr-doctor — verify the seam + engine

Read-only health check. Runs the same resolution every `/rdr-*` skill's §seam-bind
does, but checks **every** invariant (never stops at the first failure) — the seam,
the engine layout, the skill links, and the `rdr` projector binary — and reports
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

**Close-summary golden self-test (optional, after a clean structural run).** Run
`sh close-summary-check.sh --self-test` (beside this SKILL.md): it validates the
§next-step close-packet contract against golden fixtures — the good ones (both
`/rdr-*` Claude and `$rdr-*` Codex surfaces) must PASS, the gate-only sample must be
rejected. A green self-test means the contract guard still binds; a FAIL means a recent
edit regressed the close packet or the fixtures. Advisory, never a seam FAIL; skip if
the script is absent.

**Projector (checks 11/11b).** The script checks `$RDR_HOME/bin/rdr` exists, runs,
and reports the engine revision it was stamped with. Absent or unrunnable is a
**FAIL** — the skills read records through it, and `/rdr-init` builds it. A binary
whose stamp trails the engine (a plugin upgrade or a `git pull` moved the engine and
left the old build) is a **WARN**, not a FAIL: it still answers, it is just behind.
A `dev` stamp means someone built it by hand rather than through `/rdr-init`.
**11d** reports the usage log; off is a **WARN** only when autocommit is on, because
the log is the lint receipt `§commit` refuses a record without — off means unlinted
records commit unchecked.

**12** resolves `intrastate` and lints `models/rdr-status.toml`, the navigator's
routing. A missing binary is a **FAIL** — it is a dependency (rdr-common
§intrastate), and without it every stage's next-step answer stops. A failing
lint is a **WARN**: the model still answers, just without the coverage proof.
Closed-by-escape warns too — closed is not proved.

## Review gate

- **Read-only?** No file/symlink written, nothing installed — the doctor diagnoses;
  `/rdr-init` (or the user) repairs. Checks 11/11b only *read* the binary
  (`rdr version`) — the doctor never builds it.
- **Complete?** Every check ran; one FAIL never skips the rest.
- **Each FAIL names one fix?** No "might be" — the command.

## Next step (rdr-common §next-step)

- All PASS → `Next: /rdr-status`, or `/rdr-seed <idea>`.
- Any FAIL → run the named fix (usually `/rdr-init`), then re-run `/rdr-doctor`.
