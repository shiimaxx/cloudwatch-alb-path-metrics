# alb-logs-generator

## Overview

A command-line utility that emits synthetic AWS Application Load Balancer (ALB) access logs. The tool is intended for testing and validation workflows that need realistic log entries without using production traffic.

## Design Summary

- Each execution produces log lines covering a five-minute window, matching the cadence at which ALB delivers log files. The start timestamp can be supplied via `--start`; otherwise the generator defaults to five minutes in the past.
- The total number of entries defaults to `rps * 300` (five minutes), where `--rps` expresses the average requests per second. Setting `--count` overrides this calculation, while optional jitter makes the per-second distribution look more natural.
- Log lines follow the ALB access log v2 schema. An `albLogEntry` struct models the fields (type, timestamp, ELB name, client and target addresses, processing times, status codes, request line, user agent, SSL data, matched rule, actions, target status, target list, ARN, redirect URL, error code, target response metrics, etc.), and its `String()` method formats the fields in the exact order expected by AWS.
- Synthetic values rely on [`go-faker/faker/v4`](https://github.com/go-faker/faker) for items such as IPv4 addresses and user agents. Additional helper functions supply weighted selections for HTTP methods, paths, status codes, and random-but-realistic latencies using `math/rand` seeded via `--seed` (default: `time.Now().UnixNano()`).
- CLI flags include `--output` (defaults to stdout), `--alb-name`, `--target`, `--rps`, `--count`, `--start`, and `--seed`. Output streaming is handled through an `io.Writer` abstraction using a buffered writer so large log sets do not accumulate in memory.

## Incremental Implementation Plan

1. **Skeleton CLI**: Wire up `main.go` with basic flag parsing (`--output`, `--seed`) and emit a single placeholder log line to validate wiring via `go run`.
2. **Randomization & Entry Skeleton**: Initialize the faker-based random source, define the `albLogEntry` structure with essential fields (timestamp, client/target addresses, request line), and generate a small fixed set (e.g., 10) of entries for smoke testing.
3. **Five-Minute Window Logic**: Introduce `--start`, `--rps`, and `--count` handling. Compute the five-minute range (300 seconds), assign timestamps across that range with optional jitter, and confirm entry counts through manual runs.
4. **Complete ALB Schema**: Populate the remaining ALB log fields, ensure formatting order matches the ALB documentation, and expand helper functions for processing times, status codes, byte counts, actions, and other metadata.
5. **Refinement & Docs**: Add robust error handling, wrap output in a `bufio.Writer`, update usage notes in `README.md`, run `go fmt`, and capture sample output for verification.

