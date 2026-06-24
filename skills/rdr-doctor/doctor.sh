#!/bin/sh
# rdr-doctor — read-only RDR seam/engine health check. POSIX sh; identical under
# bash (Linux) and zsh (macOS). Run it as a FILE (sh "$THIS"), never by pasting its
# body into a shell — that is what made earlier runs drop labels. Writes nothing.
GC=$(git rev-parse --git-common-dir 2>/dev/null) || { echo "[FAIL] not in a git repo - run \$rdr-doctor in Codex or /rdr-doctor in Claude inside a workspace repo"; exit 0; }
GIT_COMMON=$(cd "$GC" && pwd -P) || { echo "[FAIL] git dir unreadable"; exit 0; }
PROJECT=$(dirname "$GIT_COMMON"); WS=$(dirname "$PROJECT"); export PROJECT WS
echo "rdr-doctor - project: $PROJECT"
nf=0; nw=0
fail(){ echo "  [FAIL] $1"; nf=$((nf+1)); }
warn(){ echo "  [WARN] $1"; nw=$((nw+1)); }
pass(){ echo "  [PASS] $1"; }

# nearest marker wins: repo-local (inside .rdr/) overrides shared workspace
if   [ -f "$PROJECT/.rdr/workspace" ]; then MARKER="$PROJECT/.rdr/workspace"; SCOPE="repo-local"
elif [ -f "$WS/.rdr-workspace" ];   then MARKER="$WS/.rdr-workspace";  SCOPE="workspace (shared)"
else fail "1 no marker - run \$rdr-init in Codex or /rdr-init in Claude in this repo (looked in $PROJECT/.rdr and $WS)"; echo "Verdict: 1 FAIL - no marker."; exit 0; fi
pass "1 marker present - $MARKER  [$SCOPE]"
. "$MARKER" 2>/tmp/rdrdoc.err && pass "2 marker sources clean" || fail "2 marker source error - $(head -1 /tmp/rdrdoc.err)"
m=""; for v in RDR_HOME RDR_RECORDS RDR_EVIDENCE RDR_ENV RDR_RESOURCES; do eval "[ -n \"\$$v\" ]" || m="$m $v"; done
[ -z "$m" ] && pass "3 five-var contract set" || fail "3 unset:$m - re-run \$rdr-init in Codex or /rdr-init in Claude to write the marker"
[ -d "$RDR_HOME/stages" ] && [ -d "$RDR_HOME/skills" ] && [ -d "$RDR_HOME/prompts" ] && [ -f "$RDR_HOME/TEMPLATE.md" ] && pass "4 engine resolves - $RDR_HOME" || fail "4 RDR_HOME is not an engine root ($RDR_HOME) - re-run /rdr-init"
[ -d "$RDR_RECORDS" ] && pass "5 records dir - $RDR_RECORDS" || fail "5 records dir missing ($RDR_RECORDS) - \$rdr-init in Codex or /rdr-init in Claude scaffolds it"
[ -d "$RDR_RECORDS" ] && { [ -f "$RDR_RECORDS/README.md" ] && pass "5b index README present" || warn "5b no index README - \$rdr-init in Codex or /rdr-init in Claude scaffolds it"; }
if [ -d "$RDR_EVIDENCE" ]; then pass "6 evidence root reachable - $RDR_EVIDENCE"
elif [ -d "$(dirname "$RDR_EVIDENCE")" ]; then warn "6 evidence root absent but creatable - $RDR_EVIDENCE (lenses mkdir -p on first write)"
else fail "6 evidence root parent missing ($RDR_EVIDENCE) - fix RDR_EVIDENCE in the marker"; fi
[ -f "$RDR_ENV" ] && [ -f "$RDR_RESOURCES" ] && pass "7 data files exist" || fail "7 missing RDR_ENV/RDR_RESOURCES - \$rdr-init in Codex or /rdr-init in Claude writes both"
[ -f "$RDR_ENV" ] && { grep -q "{EVIDENCE_DIR}" "$RDR_ENV" && grep -q "{ARTIFACT_DIR}" "$RDR_ENV" && grep -q "{SPIKE_DIR}" "$RDR_ENV" && pass "8 path-map names EVIDENCE_DIR/ARTIFACT_DIR/SPIKE_DIR" || fail "8 path-map missing a staging key - re-run /rdr-init"; }
b=$(find "$RDR_HOME/skills" -type l ! -exec test -e {} ";" -print 2>/dev/null)
[ -z "$b" ] && pass "9 engine skill symlinks resolve" || { fail "9 broken engine symlinks - reinstall the engine:"; echo "$b" | sed "s/^/        /"; }
seen10=
for base in "$PROJECT/.claude/skills" "$PROJECT/.codex/skills"; do
  [ -d "$base" ] || continue
  seen10=1
  b=$(find "$base"/rdr-* -type l ! -exec test -e {} ";" -print 2>/dev/null)
  [ -z "$b" ] && pass "10 consumer links resolve - $base" || { fail "10 broken consumer links in $base - repoint to \$RDR_HOME/skills/:"; echo "$b" | sed "s/^/        /"; }
done
[ -n "$seen10" ] || echo "  [INFO] 10 no consumer skill links here (engine repo, or hand-driven consumer) - n/a"
if [ "$nf" -gt 0 ]; then echo "Verdict: $nf FAIL, $nw WARN - fix the FAIL(s) above (usually \$rdr-init in Codex or /rdr-init in Claude), then re-run \$rdr-doctor in Codex or /rdr-doctor in Claude."
elif [ "$nw" -gt 0 ]; then echo "Verdict: 0 FAIL, $nw WARN - healthy; WARNs are advisory."
else echo "Verdict: all checks PASS - the seam is healthy."; fi
rm -f /tmp/rdrdoc.err
