# Requirements — cache warm ratio

- **[REQ-1]** "Expose a warm-ratio gauge per cache tier."
- **[REQ-2]** "The gauge reads 1.0 once warm-up completes."
- **[REQ-3]** "Publish the gauge on the existing metrics endpoint, no new port."
- [REQ-MVV] "A caller reads a warm ratio of 1.0 after one warm cycle."
