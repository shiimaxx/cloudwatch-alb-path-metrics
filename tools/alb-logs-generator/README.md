# alb-logs-generator

## Overview

A command-line utility that emits synthetic AWS Application Load Balancer (ALB) access logs. The tool is intended for testing and validation workflows that need realistic log entries without using production traffic.

## Design Summary

- Each execution produces log lines covering a five-minute window, matching the cadence at which ALB delivers log files. The start timestamp can be supplied via `--start`; otherwise the generator defaults to five minutes in the past.
- The total number of entries defaults to `rps * 300` (five minutes), where `--rps` expresses the average requests per second. Setting `--count` overrides this calculation, while optional jitter makes the per-second distribution look more natural.
- Log lines follow the ALB access log v2 schema. An `albLogEntry` struct models the fields (type, timestamp, ELB name, client and target addresses, processing times, status codes, request line, user agent, SSL data, matched rule, actions, target status, target list, ARN, redirect URL, error code, target response metrics, etc.), and its `String()` method formats the fields in the exact order expected by AWS.
- Synthetic values rely on [`go-faker/faker/v4`](https://github.com/go-faker/faker) for items such as IPv4 addresses and user agents. Additional helper functions supply weighted selections for HTTP methods, paths, status codes, and random-but-realistic latencies using `math/rand` seeded via `--seed` (default: `time.Now().UnixNano()`).
- CLI flags include `--output` (defaults to stdout), `--alb-name`, `--target`, `--rps`, `--count`, `--start`, and `--seed`. Output streaming is handled through an `io.Writer` abstraction using a buffered writer so large log sets do not accumulate in memory.

## ToDo

- [x] Quote the trailing string columns (`actions_executed`, `redirect_url`, `error_reason`, `target_port_list`, `target_status_code_list`, `classification`, `classification_reason`) to match the quoting style of real ALB access logs.
- [ ] Build the request line host/port using the listener (client-facing) port instead of the target port so generated URLs mirror actual ALB logs.
- [ ] Output `-` for target-related fields when the action is `redirect`, `fixed-response`, `authenticate`, or `waf`, because no target is contacted in those cases in production logs.
- [ ] Ensure `request_creation_time` plus the three processing time fields align with the entry timestamp, preserving the timing invariant observed in ALB logs.
- [ ] Format `target_port_list` and `target_status_code_list` as quoted lists (or `"-"` when untouched) to mimic the structure of AWS’ real log lines.

