# §commit — per-stage owned path-sets and subjects

Reference for the commit call in **rdr-common §commit**, read only when autocommit
is on (the gate decides; off means this file is never read). Find your stage's row,
commit exactly those paths. Engine-generic: `cli/NNNN` is the consumer's own slug
grammar — match the subject grammar already in the consumer's log.

**What each stage commits** (owned path-set + subject — never `git add -A`):

| Stage | Doc commit (`$RDR_PATH` [+ `$RDR_RECORDS/README.md`]) | Evidence commit (separate) |
|-------|---|---|
| seed | `docs(rdr): seed cli/NNNN <slug>` (`$RDR_PATH` + `$RDR_RECORDS/README.md` — the new index row) | — |
| propose | `docs(rdr): propose cli/NNNN — <summary>` | `chore(rdr): cli/NNNN propose evidence` (if the hardened premortem wrote) |
| refine | `docs(rdr): refine cli/NNNN — <summary>` | — |
| resolve | `docs(rdr): resolve cli/NNNN — <summary>` | `chore(rdr): cli/NNNN spike evidence` (if a spike wrote) |
| prelock (non-repeatability lens) | `docs(rdr): prelock cli/NNNN — <lens> pass` | `chore(rdr): cli/NNNN <lens> evidence` |
| reconcile | `docs(rdr): reconcile cli/NNNN — <summary>` | `chore(rdr): cli/NNNN reconcile evidence` |
| finalize | `docs(rdr): finalize cli/NNNN <slug> (Gate PASS)` | `chore(rdr): cli/NNNN tooling-pass evidence` (`$ITER_DIR/tooling-pass.md` the sweep wrote) |
| cluster-reconcile | (commits at finalize) | `chore(rdr): cli/NNNN cluster-reconcile <cluster>` |
| implement | (code-repo `feat(...)` commit — its own contract) | artifact files only, if gated on |

Two commits, never one: the doc/README commit is the **design history** (`docs(rdr):`,
real subject — *never* `fixup!`; per the no-fixup doctrine, RDR commits ARE the history);
the evidence subtree is a separate `chore(rdr):` commit so the `docs(rdr)` log stays
readable. Match the subject grammar already in the consumer's log
(`docs(rdr): <stage> cli/NNNN — <one-line>`).

**The `repeatability` lens is the one exception to "commit at the lens."** Its run file(s)
each land in a *fresh* session that writes only `evidence/repeatability/run-N.md` and does
**not** touch the RDR doc. So each run session commits just its own run file
(`chore(rdr): cli/NNNN repeatability run-N`) — that keeps the loop tight (a fresh session's
owned set is one file, unambiguous). The **doc commit is deferred to the diff session**,
the first point the RDR doc actually changes (`docs(rdr): prelock cli/NNNN — repeatability`).
**The diff session's evidence commit covers the whole `repeatability/` dir**, not just
`diff.md` (`chore(rdr): cli/NNNN repeatability evidence`) — so it is **self-healing**: any
run whose own session did not commit it (autocommit was off then, or the runs predate
enabling it) is swept in here, alongside `diff.md`. The no-op guard makes this idempotent —
already-committed runs add nothing. So however the per-run commits went, the lens lands
fully committed at the diff, never leaving a straggler for the human to chase.

