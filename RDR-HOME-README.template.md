<!--
  RDR-HOME index template. /rdr-init (flow/00-bootstrap.md, step 4) copies this
  to $RDR_DIR/README.md when a consumer's RDR home has no index yet. It is the
  ONLY engine file vendored into a consumer — TEMPLATE.md and the prompts are
  read in place from $RDR_FLOW_HOME, never copied (see flow/skills/rdr-common.md
  on TEMPLATE drift). After the copy this is a project-local file: edit it freely;
  it does not track the engine.

  Fill {PROJECT} below. /rdr-finalize adds/updates the index row for an RDR and
  keeps its Status column current (this file is the authoritative status table —
  see flow/07.0-finalize.md). Leave the table header + legend in place; the flow
  reads this file as the index + Status/Priority table.
-->
# {PROJECT} — Recommendation Decisioning Records

Project-scoped RDRs for `{PROJECT}`. Draft new RDRs from the shared
`TEMPLATE.md` in the RDR engine (`$RDR_FLOW_HOME/TEMPLATE.md`); `/rdr-seed`
materializes a copy automatically. Rationale + the full stage flow live in the
engine README — this file is only the per-project index.

## Index

| ID | Title | Status | Priority |
| --- | --- | --- | --- |
<!-- /rdr-finalize adds a row per RDR here; first finalized RDR replaces this comment. -->

## Status legend

- **Draft** — during the planning/research phase
- **Final** — locked, ready for or during implementation
- **Implemented** — implementation complete
- **Reverted** — implemented then undone (document why)
- **Abandoned** — RDR not implemented
- **Superseded** — replaced by another RDR
- **Demoted** — judged not RDR-shaped; refiled as a plain issue (carry `Demoted [→ <issue link>]`)
