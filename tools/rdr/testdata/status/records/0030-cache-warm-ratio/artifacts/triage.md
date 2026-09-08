# Triage — cache warm ratio

## Verdict table

| # | Source | Location | Verdict | odc-type | odc-trigger | Outcome |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | cove | cache.go:41 | drop:superseded-at-HEAD | — | — | — |
| 2 | grounding | cache.go:58 | IN-SCOPE (C1: gauge label) | — | — | fixed:0a1b2c3 |
| 3 | 3amigo | cache.go:72 | IN-SCOPE (C2: warmup race) | — | — | held:contract (C2 names no unit) |
| 4 | critique | cache.go:90 | OUT-OF-SCOPE | — | — | kata zz9q |
