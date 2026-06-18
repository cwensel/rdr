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

## Run it — ONE Bash call, verbatim

Run the script in a **single** Bash call and relay its stdout. Don't split checks
across calls or improvise the resolution: shell state (`$WS`, the sourced `RDR_*`
vars, the marker's `: "${WS:?…}"` guard) lives only inside the one shell that sets
them — piecemeal runs are what make it slow and flaky. POSIX `sh`, so identical
under bash (Linux) and zsh (macOS).

```sh
sh -c '
GC=$(git rev-parse --git-common-dir 2>/dev/null) || { echo "[FAIL] not in a git repo - run /rdr-doctor inside a workspace repo"; exit 0; }
GIT_COMMON=$(cd "$GC" && pwd -P) || { echo "[FAIL] git dir unreadable"; exit 0; }
WS=$(dirname "$(dirname "$GIT_COMMON")"); export WS          # export so the marker :? guard passes
PROJ=$(dirname "$GIT_COMMON")
echo "rdr-doctor - workspace: $WS"
nf=0; nw=0
fail(){ echo "  [FAIL] $1"; nf=$((nf+1)); }
warn(){ echo "  [WARN] $1"; nw=$((nw+1)); }
pass(){ echo "  [PASS] $1"; }

[ -f "$WS/.rdr-workspace" ] || { fail "1 marker absent - run /rdr-init from the consumer repo root"; echo "Verdict: 1 FAIL - no marker."; exit 0; }
pass "1 marker present - $WS/.rdr-workspace"
. "$WS/.rdr-workspace" 2>/tmp/rdrdoc.err && pass "2 marker sources clean" || fail "2 marker source error - $(head -1 /tmp/rdrdoc.err)"
m=""; for v in RDR_HOME RDR_RECORDS RDR_EVIDENCE RDR_ENV RDR_RESOURCES; do eval "[ -n \"\$$v\" ]" || m="$m $v"; done
[ -z "$m" ] && pass "3 five-var contract set" || fail "3 unset:$m - re-run /rdr-init to write the marker"
[ -d "$RDR_HOME/stages" ] && [ -d "$RDR_HOME/skills" ] && [ -d "$RDR_HOME/prompts" ] && [ -f "$RDR_HOME/TEMPLATE.md" ] && pass "4 engine resolves - $RDR_HOME" || fail "4 RDR_HOME is not an engine root ($RDR_HOME) - re-run /rdr-init"
[ -d "$RDR_RECORDS" ] && pass "5 records dir - $RDR_RECORDS" || fail "5 records dir missing ($RDR_RECORDS) - /rdr-init scaffolds it"
[ -d "$RDR_RECORDS" ] && { [ -f "$RDR_RECORDS/README.md" ] && pass "5b index README present" || warn "5b no index README - /rdr-init scaffolds it"; }
if [ -d "$RDR_EVIDENCE" ]; then pass "6 evidence root reachable - $RDR_EVIDENCE"
elif [ -d "$(dirname "$RDR_EVIDENCE")" ]; then warn "6 evidence root absent but creatable - $RDR_EVIDENCE (lenses mkdir -p on first write)"
else fail "6 evidence root parent missing ($RDR_EVIDENCE) - fix RDR_EVIDENCE in the marker"; fi
[ -f "$RDR_ENV" ] && [ -f "$RDR_RESOURCES" ] && pass "7 data files exist" || fail "7 missing RDR_ENV/RDR_RESOURCES - /rdr-init writes both"
[ -f "$RDR_ENV" ] && { grep -q "{EVIDENCE_DIR}" "$RDR_ENV" && grep -q "{ARTIFACT_DIR}" "$RDR_ENV" && grep -q "{SPIKE_DIR}" "$RDR_ENV" && pass "8 path-map names EVIDENCE_DIR/ARTIFACT_DIR/SPIKE_DIR" || fail "8 path-map missing a staging key - re-run /rdr-init"; }
b=$(find "$RDR_HOME/skills" -type l ! -exec test -e {} ";" -print 2>/dev/null)
[ -z "$b" ] && pass "9 engine skill symlinks resolve" || { fail "9 broken engine symlinks - reinstall the engine:"; echo "$b" | sed "s/^/        /"; }
seen10=
for base in "$PROJ/.claude/skills" "$PROJ/.codex/skills"; do
  [ -d "$base" ] || continue
  seen10=1
  b=$(find "$base"/rdr-* -type l ! -exec test -e {} ";" -print 2>/dev/null)
  [ -z "$b" ] && pass "10 consumer links resolve - $base" || { fail "10 broken consumer links in $base - repoint to \$RDR_HOME/skills/:"; echo "$b" | sed "s/^/        /"; }
done
[ -n "$seen10" ] || echo "  [INFO] 10 no consumer skill links here (engine repo, or hand-driven consumer) - n/a"
if [ "$nf" -gt 0 ]; then echo "Verdict: $nf FAIL, $nw WARN - fix the FAIL(s) above (usually /rdr-init), then re-run /rdr-doctor."
elif [ "$nw" -gt 0 ]; then echo "Verdict: 0 FAIL, $nw WARN - healthy; WARNs are advisory."
else echo "Verdict: all checks PASS - the seam is healthy."; fi
rm -f /tmp/rdrdoc.err
'
```

**Print the script's stdout verbatim inside a fenced block** — it is the report (pure
ASCII, one line per check + a Verdict). Do not reformat it into a table, re-run it to
"confirm", or paraphrase the lines — that's wasted turns. The first run is the answer;
if a line says FAIL, surface its fix as-is, don't improvise a repair.

**Corpora (optional, after a clean structural run).** If `arc` is present, list the
named corpora in `$RDR_RESOURCES` and note any that don't resolve as `[WARN]`
(degraded mode — research stages run hollow until built). Never a FAIL; skip if no `arc`.

## Review gate

- **Read-only?** No file/symlink written, nothing installed — the doctor diagnoses;
  `/rdr-init` (or the user) repairs.
- **Complete?** Every check ran; one FAIL never skips the rest.
- **Each FAIL names one fix?** No "might be" — the command.

## Next step (rdr-common §next-step)

- All PASS → `Next: /rdr-status`, or `/rdr-seed <idea>`.
- Any FAIL → run the named fix (usually `/rdr-init`), then re-run `/rdr-doctor`.
