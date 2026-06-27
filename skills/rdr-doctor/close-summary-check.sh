#!/bin/sh
# close-summary-check — regression guard for the §next-step close packet
# (skills/rdr-common.md). Asserts a stage close summary carries the decision
# fields and is not a gate-only footer; enforces per-surface command spelling.
# POSIX sh; identical under bash (Linux) and zsh (macOS). Writes nothing.
#
#   sh close-summary-check.sh <summary-file>   # validate one summary
#   sh close-summary-check.sh --self-test      # run fixtures/ (good PASS, bad FAIL)
#
# The required-field set below is the literal §next-step contract. If a field is
# renamed in rdr-common.md without mirroring it here, the SKILL's binding check
# (grep of §next-step labels) diverges and the dry-run catches it.
FIELDS='Outcome Gate RDR delta Deviations Continue check Next'

# check_one FILE SURFACE  (SURFACE = claude | codex | any) -> prints reason, returns 1 on fail
check_one() {
  f=$1; surface=$2; why=
  for label in Outcome Gate "RDR delta" Deviations "Continue check" Next; do
    grep -qE "^${label}:" "$f" || why="${why} missing-field:${label};"
  done
  # gate-only guard: RDR delta must name content OR use the explicit escape
  delta=$(grep -E '^RDR delta:' "$f" | head -1 | sed 's/^RDR delta:[[:space:]]*//')
  if [ -z "$delta" ]; then
    why="${why} RDR-delta-empty;"
  elif printf '%s' "$delta" | grep -qiE '^(none|n/a)\b' && ! printf '%s' "$delta" | grep -qiE 'none owed because'; then
    why="${why} RDR-delta-bare-none(must say 'none owed because <reason>');"
  fi
  # per-surface command spelling on the Next line
  next=$(grep -E '^Next:' "$f" | head -1)
  case $surface in
    claude) printf '%s' "$next" | grep -q '/rdr-' || why="${why} Next-missing-/rdr-*(Claude);";;
    codex)  printf '%s' "$next" | grep -q '\$rdr-' || why="${why} Next-missing-\$rdr-*(Codex);";;
    any)    printf '%s' "$next" | grep -qE '(/rdr-|\$rdr-)' || why="${why} Next-missing-rdr-command;";;
  esac
  [ -z "$why" ] && return 0
  echo "$why"; return 1
}

if [ "$1" = "--self-test" ]; then
  DIR=$(dirname "$0")/fixtures; nf=0; np=0
  for g in "$DIR"/*.good.md; do
    [ -f "$g" ] || continue
    case $(basename "$g") in claude-*) s=claude;; codex-*) s=codex;; *) s=any;; esac
    r=$(check_one "$g" "$s") && { echo "  [PASS] $(basename "$g")"; np=$((np+1)); } \
      || { echo "  [FAIL] $(basename "$g") -$r  (expected PASS)"; nf=$((nf+1)); }
  done
  for b in "$DIR"/bad-*.md; do
    [ -f "$b" ] || continue
    if check_one "$b" any >/dev/null; then echo "  [FAIL] $(basename "$b") wrongly PASSED (expected FAIL)"; nf=$((nf+1))
    else echo "  [PASS] $(basename "$b") correctly rejected"; np=$((np+1)); fi
  done
  [ "$nf" -eq 0 ] && echo "Verdict: self-test green - $np checks, contract guard binds." \
    || echo "Verdict: $nf FAIL of $((np+nf)) - the close-packet guard regressed."
  exit 0
fi

[ -n "$1" ] && [ -f "$1" ] || { echo "usage: sh close-summary-check.sh <summary-file> | --self-test"; exit 2; }
case $(basename "$1") in claude-*) s=claude;; codex-*) s=codex;; *) s=any;; esac
r=$(check_one "$1" "$s") && echo "PASS - close packet complete ($1)" \
  || { echo "FAIL - $1 -$r"; exit 1; }
