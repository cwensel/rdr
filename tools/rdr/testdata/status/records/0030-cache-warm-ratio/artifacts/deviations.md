# Deviations — cache warm ratio

- **Type**: SPEC-UNDER — the RDR does not say whether a tier with no
  entries reads 0.0 or 1.0.
  Status: needs author decision → RESOLVED (an empty tier reads 1.0; there is
  nothing left to warm)
- **Type**: IMPL-DECISION — the gauge reuses the existing tier label.
  Status: mechanical translation
