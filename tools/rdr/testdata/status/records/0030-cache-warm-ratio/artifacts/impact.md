# Impact — 0030-cache-warm-ratio

families: 2
rows: 3
records: 0021
literals: none
repo: testdata/status/repo @abc1234
convention: go

## TestCorpus0021 (2 tests, 1 files)
| test | file | arm |
|---|---|---|
| TestCorpus0021_REQ1_WarmRatio | internal/cache/warm_test.go | record:0021 |
| TestCorpus0021_REQ2_ColdStart | internal/cache/warm_test.go | record:0021 |

## warm_test.go (1 tests, 1 files)
| test | file | arm |
|---|---|---|
| TestWarmHelper | internal/cache/warm_test.go | record:0021 |
