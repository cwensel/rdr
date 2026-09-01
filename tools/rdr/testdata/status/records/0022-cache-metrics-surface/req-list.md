# Requirements — cache metrics surface

- **[REQ-1]** "Expose a hit-rate gauge per cache tier."
- **[REQ-2]** "Expose an eviction-count counter per cache tier."
- **[REQ-3]** "Publish the gauges on the existing metrics endpoint, no new port."
- [REQ-MVV] "A caller can read the hit-rate gauge after one warm cycle."
