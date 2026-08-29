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
# A marker states which project it describes (RDR_PROJECT_ANCHOR) and REFUSES under
# another, binding nothing. Stop here when it does: every check below reads a contract
# var, so continuing would report an unbound seam as a broken engine and a missing
# records dir - a cascade of wrong diagnoses over one real cause. Check 1b's
# membership rule is the same judgement reached independently, and it is what the
# workspace-scope marker's own guard implements; it still runs for an ANCHORLESS
# shared marker, which binds and so cannot refuse.
if . "$MARKER" 2>/tmp/rdrdoc.err; then
  pass "2 marker sources clean"
else
  fail "2 marker source error - $(head -1 /tmp/rdrdoc.err)"
  case "$(head -1 /tmp/rdrdoc.err)" in
    stopped:foreign-*)
      echo "        the marker describes another project, so no seam var is bound;"
      echo "        checks 3+ would report that as a broken install. Fix the bind first:"
      echo "        run \$rdr-init in Codex or /rdr-init in Claude here for a repo-local marker."
      echo "Verdict: $((nf)) FAIL - marker refused this project."
      exit 0;;
  esac
fi
m=""; for v in RDR_HOME RDR_RECORDS RDR_EVIDENCE RDR_ENV RDR_RESOURCES; do eval "[ -n \"\$$v\" ]" || m="$m $v"; done
[ -z "$m" ] && pass "3 five-var contract set" || fail "3 unset:$m - re-run \$rdr-init in Codex or /rdr-init in Claude to write the marker"
# 3b-legacy - a marker written before the rename still exports RDR_REPO. The
# skills no longer read it, so anchor checks silently report "not run" on a
# consumer that was correctly configured. Name the migration, do not guess it.
if [ -z "$RDR_SOURCE_REPO" ] && [ -n "$RDR_REPO" ]; then
  warn "3b RDR_REPO is the pre-rename name and is no longer read - rename it to RDR_SOURCE_REPO in $MARKER (value unchanged: $RDR_REPO), or re-run \$rdr-init --reconfigure in Codex / /rdr-init --reconfigure in Claude"
else
  [ -n "$RDR_SOURCE_REPO" ] && { [ -d "$RDR_SOURCE_REPO" ] && pass "3b source root - $RDR_SOURCE_REPO" || fail "3b RDR_SOURCE_REPO set but missing ($RDR_SOURCE_REPO) - fix the marker; a wrong root makes anchor checks report false findings"; } || { [ "$PROJECT" = "$RDR_HOME" ] && echo "  [INFO] 3b RDR_SOURCE_REPO unset - engine repo, the flow runs in consumers - n/a" || warn "3b RDR_SOURCE_REPO unset - source-anchor checks report \"not run\" (correct when the RDRs cite no path::Symbol anchors; else set it in the marker)"; }
fi
# 1b - an inherited shared marker. Workspace scope describes ONE project whose
# parts span sibling repos; its vars are single-valued. A repo that is NOT part of
# that project silently inherits its records, so a /rdr-seed here lands an RDR in
# the other project's dir with no error. Membership = this repo is the source root,
# or holds the records / evidence / seam data the marker names.
if [ "$SCOPE" = "workspace (shared)" ] && [ "$PROJECT" != "$RDR_HOME" ]; then
  mine=""
  for v in "$RDR_SOURCE_REPO" "$RDR_RECORDS" "$RDR_EVIDENCE" "$RDR_ENV"; do
    case "$v" in "$PROJECT"|"$PROJECT"/*) mine=1;; esac
  done
  # A *_ROOT naming this repo is NOT membership: a marker lists sibling repos as
  # conveniences, most with no RDR role. Only the four seam paths decide. With
  # RDR_SOURCE_REPO unset the source repo cannot prove itself, so say that rather than
  # accuse it - the fix is to set RDR_SOURCE_REPO, which 3b already asks for.
  if [ -z "$mine" ] && [ -z "$RDR_SOURCE_REPO" ]; then
    warn "1b shared marker (records=$RDR_RECORDS) - RDR_SOURCE_REPO unset, so whether this repo is the project's source root cannot be told. Set RDR_SOURCE_REPO in $MARKER, or write a repo-local marker if this is a different project"
    mine=skip
  fi
  [ "$mine" = skip ] || { [ -n "$mine" ] && pass "1b shared marker describes this project" \
    || warn "1b shared marker names another project (records=$RDR_RECORDS) - this repo is not in it; a /rdr-seed here writes there. Run \$rdr-init in Codex or /rdr-init in Claude to write a repo-local marker"; }
fi
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
# 9b - shared helpers a SKILL.md references must be PRESENT beside it. An upgrade that
# adds a helper leaves older skill dirs without the link; missing != broken, so 9 can't see it.
miss9=""
for f in rdr-common.md rdr-commit.sh rdr-commit-map.md; do
  [ -f "$RDR_HOME/skills/$f" ] || { miss9="$miss9 engine:$f"; continue; }
  for d in "$RDR_HOME"/skills/*/; do
    [ -f "${d}SKILL.md" ] || continue
    grep -q "$f" "${d}SKILL.md" "$RDR_HOME/skills/rdr-common.md" 2>/dev/null || continue
    [ -e "$d$f" ] || miss9="$miss9 $(basename "$d")/$f"
  done
done
[ -z "$miss9" ] && pass "9b shared helpers linked in every skill dir" || fail "9b missing helper links (engine upgrade added a shared file):$miss9 - relink each as: ln -s \"../<file>\" \"\$RDR_HOME/skills/<skill>/<file>\" (or reinstall the engine)"
seen10=
for base in "$PROJECT/.claude/skills" "$PROJECT/.codex/skills"; do
  [ -d "$base" ] || continue
  seen10=1
  b=$(find "$base"/rdr-* -type l ! -exec test -e {} ";" -print 2>/dev/null)
  [ -z "$b" ] && pass "10 consumer links resolve - $base" || { fail "10 broken consumer links in $base - repoint to \$RDR_HOME/skills/:"; echo "$b" | sed "s/^/        /"; }
  # 10b - a farm that mounts rdr-* must be COMPLETE: a newly shipped engine skill
  # is invisible to the consumer until linked (missing != broken, so check 10 can't see it).
  if ls "$base"/rdr-* >/dev/null 2>&1; then
    miss=""
    for d in "$RDR_HOME"/skills/*/; do
      s=$(basename "$d"); [ -f "${d}SKILL.md" ] || continue
      [ -e "$base/$s" ] || miss="$miss $s"
    done
    [ -z "$miss" ] && pass "10b farm complete - $base" || warn "10b engine skills not mounted in $base:$miss - add: ln -s \"\$RDR_HOME/skills/<name>\" \"$base/<name>\" (a bare /rdr-init re-run offers this)"
  fi
done
[ -n "$seen10" ] || echo "  [INFO] 10 no consumer skill links here (engine repo, or hand-driven consumer) - n/a"
# 11 - the projector binary rdr-init builds. Skills call it as "$RDR_HOME/bin/rdr";
# it is gitignored, so a fresh clone/plugin install has none until /rdr-init runs.
RDR_BIN="$RDR_HOME/bin/rdr"
if [ ! -x "$RDR_BIN" ]; then
  if command -v go >/dev/null 2>&1; then
    fail "11 projector not built ($RDR_BIN) - \$rdr-init in Codex or /rdr-init in Claude builds it, or: (cd \"\$RDR_HOME/tools/rdr\" && go build -o \"\$RDR_HOME/bin/rdr\" .)"
  else
    fail "11 projector not built and 'go' is not on PATH - install Go (go.dev/dl), then \$rdr-init in Codex or /rdr-init in Claude"
  fi
else
  ver=$("$RDR_BIN" version 2>/dev/null) || ver=""
  if [ -z "$ver" ]; then
    fail "11 projector present but 'rdr version' failed ($RDR_BIN) - rebuild: \$rdr-init in Codex or /rdr-init in Claude"
  else
    pass "11 projector built - $RDR_BIN ($ver)"
    # 11b - staleness. The binary is stamped with the engine revision it was built
    # from (-X main.version). A plugin upgrade or a git pull moves the engine and
    # leaves the old binary in place; it still answers, so this warns, never fails.
    built=$(echo "$ver" | awk '{print $2}')
    cur=""
    if [ -d "$RDR_HOME/.git" ] || git -C "$RDR_HOME" rev-parse --git-dir >/dev/null 2>&1; then
      cur=$(git -C "$RDR_HOME" rev-parse --short HEAD 2>/dev/null)
    elif [ -f "$RDR_HOME/.claude-plugin/plugin.json" ]; then
      cur=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$RDR_HOME/.claude-plugin/plugin.json" | head -1)
    fi
    if [ -z "$cur" ]; then
      warn "11b projector staleness unknown - engine has no git dir or plugin.json to compare against (built $built)"
    elif [ "$built" = "dev" ]; then
      warn "11b projector built outside the install path (version 'dev', engine at $cur) - \$rdr-init in Codex or /rdr-init in Claude stamps it"
    elif [ "$built" = "$cur" ]; then
      pass "11b projector matches the engine - $cur"
    else
      warn "11b projector stale - built $built, engine now $cur - rebuild: \$rdr-init in Codex or /rdr-init in Claude"
    fi

    # 11c - self-binding. The projector finds the marker itself and reads
    # $RDR_RECORDS from it, so a skill reaches it with no seam bound and no
    # exported prefix carried between calls. If that stops working the flow
    # still runs - every call site may still export - but it pays the baggage
    # again on every turn, silently. Checked with the environment CLEARED, so
    # a var this session happens to export cannot mask a broken bind.
    if [ -n "$RDR_RECORDS" ] && [ -d "$RDR_RECORDS" ]; then
      probe=$(cd "$PROJECT" 2>/dev/null && env -u RDR_RECORDS -u RDR_SOURCE_REPO "$RDR_BIN" index --status 2>&1 | tail -1)
      case "$probe" in
        *"no-records"*|*"stopped:"*)
          warn "11c projector does not self-bind the seam from $PROJECT - skills must export \$RDR_RECORDS per call ($probe)" ;;
        *)
          pass "11c projector self-binds the seam - no per-call export needed" ;;
      esac

      # 11d - the usage log. Off is the default; this exists so "is it actually
      # recording?" is never a guess, and so a marker that says on but resolves
      # nowhere is visible rather than silently writing nothing. Off stops being
      # neutral once autocommit is on: the log is the lint receipt §commit
      # demands, and with no log `rdr receipt` exits 2 and every record commit
      # proceeds unchecked - a gate closed without lint is no longer caught.
      case "$RDR_USAGE_LOG" in
        ""|0|false|off|no|OFF|FALSE|No|NO)
          case "$RDR_AUTOCOMMIT" in
            1|true|on|yes|TRUE|ON|Yes|YES) warn "11d usage log off while autocommit is on - no lint receipt, so §commit cannot refuse an unlinted record (rdr receipt exits 2, commits proceed with a note) - /rdr-init --usage-log turns it on" ;;
            *) echo "  [INFO] 11d usage log off - /rdr-init --usage-log turns it on (with autocommit off there is no commit to gate)" ;;
          esac ;;
        1|true|on|yes|TRUE|ON|Yes|YES)
          # Beside the marker, whichever scope resolved: the log follows the
          # seam rather than assuming a .rdr/ a workspace consumer never had.
          echo "  [INFO] 11d usage log on - $(dirname "$MARKER")/usage.jsonl" ;;
        *)
          if [ -d "$(dirname "$RDR_USAGE_LOG")" ]; then
            echo "  [INFO] 11d usage log on - $RDR_USAGE_LOG"
          else
            warn "11d usage log points at $RDR_USAGE_LOG whose directory does not exist"
          fi ;;
      esac
    fi
  fi
fi

# 12 - the routing binary and its model. `intrastate` is a DEPENDENCY (rdr-common
# §intrastate): the models under models/ are the authority for which stage and
# which lens come next, so a skill that cannot reach it has no routing answer and
# stops. Absent is therefore a FAIL, not the INFO it was while it was an
# accelerator.
#
# A model that stops LINTING still answers, just without the proof, so a broken
# lint stays a WARN - that distinction is the reason these are two findings.
#
# Resolution order is the skill's: $RDR_INTRASTATE, else PATH. A marker naming a
# binary that is not there FAILs on its own - it was configured on purpose, so
# silently falling through to PATH would hide a typo in the marker.
if [ -n "$RDR_HOME" ] && [ -f "$RDR_HOME/models/rdr-status.toml" ]; then
  IS=""
  if [ -n "$RDR_INTRASTATE" ]; then
    if [ -x "$RDR_INTRASTATE" ]; then IS="$RDR_INTRASTATE"
    else fail "12 RDR_INTRASTATE names $RDR_INTRASTATE, which is not executable - fix the path in $MARKER or unset it to fall back to PATH"; fi
  else
    IS=$(command -v intrastate 2>/dev/null)
  fi
  if [ -n "$IS" ]; then
    if lintout=$("$IS" lint --model "$RDR_HOME/models/rdr-status.toml" --as json 2>&1); then
      case "$lintout" in
        # Exit 0 still carries advisories, and one of them matters here:
        # coverage closed by a bare escape row is closed, not proved.
        *graph-coverage-closed-by-escape*)
          warn "12 routing model coverage is closed by an escape row rather than proved over its declared domains - a bare green is deliberately not available" ;;
        *) pass "12 routing model lints clean - every routing cell is claimed exactly once" ;;
      esac
    else
      warn "12 routing model fails graph-lint, so the navigator's coverage proof is broken - $(echo "$lintout" | head -c 200)"
    fi
  elif [ -z "$RDR_INTRASTATE" ]; then
    # Only when nothing was configured: a bad RDR_INTRASTATE already FAILed, and
    # repeating it would read as a second, separate finding.
    fail "12 intrastate not found - the routing models cannot be resolved, so every stage's next-step answer stops - \$rdr-init in Codex or /rdr-init in Claude installs it, or set RDR_INTRASTATE in $MARKER to a built binary"
  fi
fi
if [ "$nf" -gt 0 ]; then echo "Verdict: $nf FAIL, $nw WARN - fix the FAIL(s) above (usually \$rdr-init in Codex or /rdr-init in Claude), then re-run \$rdr-doctor in Codex or /rdr-doctor in Claude."
elif [ "$nw" -gt 0 ]; then echo "Verdict: 0 FAIL, $nw WARN - healthy; WARNs are advisory."
else echo "Verdict: all checks PASS - the seam is healthy."; fi
rm -f /tmp/rdrdoc.err
