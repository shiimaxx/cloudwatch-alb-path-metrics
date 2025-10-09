# alb-logs-generator

## Overview

A command-line utility that emits synthetic AWS Application Load Balancer (ALB) access logs. The tool is intended for testing and validation workflows that need realistic log entries without using production traffic.

## Design Summary

- Each execution produces log lines covering a five-minute window, matching the cadence at which ALB delivers log files. The generator defaults the start timestamp to five minutes in the past.
- The total number of entries defaults to `rps * 300` (five minutes), where `--rps` expresses the average requests per second. Setting `--count` overrides this calculation, while optional jitter makes the per-second distribution look more natural.
- Log lines follow the ALB access log v2 schema. `ALBLogEntry` is built from two dedicated structs: `IrrelevantFields` bundles the background values the tool does not currently inspect, while `RelevantFields` carries the metrics-oriented fields (`request_processing_time`, `target_processing_time`, `response_processing_time`, `elb_status_code`, `target_status_code`, and `request`). `ALBLogEntry.String()` renders the combined data in the order expected by AWS.
- `IrrelevantFields` uses [`go-faker/faker/v4`](https://github.com/go-faker/faker) tags to synthesize the static pieces (client and target addresses, byte counts, IDs, user agents, SSL metadata, etc.) without custom logic. A separate `RelevantFieldGenerator` produces the `RelevantFields`, leaving room to enforce constraints across the latency timings and status codes as scenarios become more sophisticated.
- Request lines are assembled from caller-supplied host and path candidates so traffic patterns can be tailored per run. Weighted selections retain the existing support for HTTP verbs, status distributions, and latency jitter via `math/rand` seeded with `--seed` (default: `time.Now().UnixNano()`).
- CLI flags currently include `--rps` and `--count`; output always streams to stdout via a buffered `io.Writer` so large log sets do not accumulate in memory.
