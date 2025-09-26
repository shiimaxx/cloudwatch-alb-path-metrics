# cloudwatch-alb-path-metrics

A Lambda function that aggregates ALB logs and publishes path-based CloudWatch custom metrics

## Overview

This project provides a Lambda function that analyzes Application Load Balancer (ALB) access logs and publishes path-based HTTP request performance metrics as Amazon CloudWatch custom metrics. The key feature is collecting metrics organized by request paths, enabling detailed monitoring of individual API endpoints or URL patterns.

## Motivation

ALB-level metrics serve as an effective starting point for SLI/SLO implementations. However, when a single ALB fronts every request, those metrics remain coarse aggregates that mask per-feature or per-service behavior. Teams still need a way to observe how individual endpoints perform.

AWS-native tooling does not surface path-level metrics out of the box; you must parse access logs and normalize URLs yourself to keep metric cardinality under control. This project automates that workflow with a Lambda function that turns ALB logs into curated CloudWatch metrics for each endpoint while staying entirely within managed AWS services.

## Configuration

### Environment Variables

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `NAMESPACE` | CloudWatch custom metrics namespace | Yes | `MyApplication/ALB` |
| `SERVICE` | Service name | Yes | `web-api` |
| `INCLUDE_PATH_RULES` | JSON array describing host-aware path normalization rules | No | `[{"host":"example.com","path":"^/users/[0-9]+$","name":"/users/:id"}]` |

### Path Rules

Define path rules to group high-cardinality URLs into stable patterns before publishing metrics. Provide a JSON array via `INCLUDE_PATH_RULES`, ordered from the most specific rule to the most general. Each rule object supports the following keys:

- `host` (required): Exact host name comparison performed against the log entry.
- `path` (required): Regular expression applied to the request path.
- `name` (required): Normalized path string emitted in the `Path` dimension when both host and regex match.

```json
[
  {"host":"example.com","path":"^/users/[0-9]+$","name":"/users/:id"},
  {"host":"example.com","path":"^/articles/(?:[a-z0-9-]+)/comments$","name":"/article/:slug/comments"},
  {"host":"admin.example.com","path":"^/dashboard(?:/.*)?$","name":"/dashboard/*"}
]
```

This configuration performs the following transformations when both host and path match:

- `https://example.com/users/42` → `/users/:id`
- `https://example.com/articles/next-gen-observability/comments` → `/article/:slug/comments`
- `https://admin.example.com/dashboard/settings` → `/dashboard/*`

Log entries that do not match any rule are ignored.
