#!/bin/sh
# §commit — commit an EXACT owned path-set to its OWN repo. No staging churn, no
# git-status read, repo-aware, parallel-safe (private index + compare-and-swap ref update).
# Sourced, not run: `. "$RDR_HOME/skills/rdr-commit.sh"` then call rdr_commit.
# Usage:  rdr_autocommit_on "$@" && { . "$RDR_HOME/skills/rdr-commit.sh"; rdr_commit "docs(rdr): … cli/$NNNN — <summary>" "$RDR_PATH" "$RDR_RECORDS/README.md"; }
# Preconditions: §seam-bind ran (RDR_AUTOCOMMIT + paths bound). Paths are ABSOLUTE; all in one repo per call.
# The gate (rdr_autocommit_on) stays inline in rdr-common §commit — it decides whether
# to source this file at all, so it cannot live in the file it gates.

rdr_commit() {
  SUBJECT="$1"; shift                                  # remaining args = the owned ABSOLUTE paths
  [ "$#" -ge 1 ] || return 0
  # Derive the owning repo from the FIRST path; assert every path lives in that same repo.
  REPO=$(cd "$(dirname "$1")" 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null) || {
    echo "stopped:commit-no-repo:$1" >&2; return 1; }
  for p in "$@"; do
    r=$(cd "$(dirname "$p")" 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null)
    [ "$r" = "$REPO" ] || { echo "stopped:commit-cross-repo:$p not in $REPO" >&2; return 1; }
  done
  TMPIDX="$REPO/.git/rdr-skillidx-$$-${NNNN:-x}"        # per-run PRIVATE index in the TARGET repo
  n=0
  while [ "$n" -lt 50 ]; do                            # CAS retry cap — generous; the retry is cheap
    PARENT=$(git -C "$REPO" rev-parse HEAD)
    GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" read-tree "$PARENT"        # seed full tree from HEAD
    GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" add -- "$@"                # stage ONLY my paths, in MY index
    TREE=$(GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" write-tree)
    if [ "$TREE" = "$(git -C "$REPO" rev-parse "$PARENT^{tree}")" ]; then  # no-op guard: my paths unchanged
      rm -f "$TMPIDX"; return 0                                            # → no empty commit, silent
    fi
    COMMIT=$(GIT_INDEX_FILE="$TMPIDX" git -C "$REPO" commit-tree "$TREE" -p "$PARENT" -m "$SUBJECT")
    if git -C "$REPO" update-ref HEAD "$COMMIT" "$PARENT" 2>/dev/null; then   # CAS: only if HEAD unmoved
      # Reconcile ONLY my paths in the REAL index so `git status` is clean afterward,
      # without disturbing the user's own staged work. `git reset -- <pathspec>` handles
      # both file and DIRECTORY args (evidence is passed as a dir); retry if index.lock is busy.
      r=0; while [ "$r" -lt 20 ]; do git -C "$REPO" reset -q HEAD -- "$@" 2>/dev/null && break; r=$((r+1)); done
      rm -f "$TMPIDX"
      echo "committed $(git -C "$REPO" rev-parse --short HEAD)  $SUBJECT"
      return 0
    fi
    n=$((n+1))                                          # CAS lost (a parallel run advanced HEAD): retry
    sleep "0.0$((n % 9))"                               # brief jittered backoff so racers don't re-collide
  done
  rm -f "$TMPIDX"
  echo "stopped:commit-contended — HEAD moved 50×; the paths are written, commit manually" >&2
  return 1
}
