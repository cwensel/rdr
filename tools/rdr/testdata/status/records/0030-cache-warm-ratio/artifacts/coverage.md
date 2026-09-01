# Coverage — cache warm ratio

| Requirement | Test |
| --- | --- |
| `REQ-1` | `TestWarmRatioGaugePerTier` |
| `REQ-2` | `TestWarmRatioReadsOneAfterWarmup` |
| `REQ-3` | `TestGaugeOnExistingEndpoint` |

## REQ-MVV runner

`go test ./cache -run TestWarmRatioEndToEnd`

## REQ-MVV output

A caller hit the metrics endpoint after one warm cycle and read a
warm ratio of 1.0 for the hot tier.
