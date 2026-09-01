# Coverage — cache metrics surface

| Requirement | Test |
| --- | --- |
| `REQ-1` | `TestHitRateGaugePerTier` |
| `REQ-2` | `TestEvictionCounterPerTier` |
| `REQ-3` | `TestGaugesOnExistingEndpoint` |

## REQ-MVV output (recorded after Phase 2)

A caller hit the metrics endpoint after one warm cycle and read a
non-zero hit-rate gauge for the hot tier.
