# alb-logs-generator

## Overview

A command-line utility that emits synthetic AWS Application Load Balancer (ALB) access logs. The tool is intended for testing and validation workflows that need realistic log entries without using production traffic.

## Design Summary

- Each execution produces log lines covering a five-minute window, matching the cadence at which ALB delivers log files. The generator defaults the start timestamp to five minutes in the past.
- The total number of entries defaults to `rps * 300` (five minutes), where `--rps` expresses the average requests per second. Setting `--count` overrides this calculation, while optional jitter makes the per-second distribution look more natural.
- Log lines follow the ALB access log v2 schema. An `albLogEntry` struct models the fields (type, timestamp, ELB name, client and target addresses, processing times, status codes, request line, user agent, SSL data, matched rule, actions, target status, target list, ARN, redirect URL, error code, target response metrics, etc.), and its `String()` method formats the fields in the exact order expected by AWS.
- Synthetic values rely on [`go-faker/faker/v4`](https://github.com/go-faker/faker) for items such as IPv4 addresses and user agents. Additional helper functions supply weighted selections for HTTP methods, paths, status codes, and random-but-realistic latencies using `math/rand` seeded via `--seed` (default: `time.Now().UnixNano()`).
- CLI flags include `--output` (defaults to stdout), `--alb-name`, `--target`, `--rps`, `--count`, and `--seed`. Output streaming is handled through an `io.Writer` abstraction using a buffered writer so large log sets do not accumulate in memory.
