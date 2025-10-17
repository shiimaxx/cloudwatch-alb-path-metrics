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
| `INCLUDE_PATH_RULES` | JSON array describing host-aware path normalization rules | No | `[{"host":"example.com","pattern":"^/users/[0-9]+$","name":"/users/:id"}]` |

The function publishes metrics under the CloudWatch namespace `cloudwatch-alb-path-metrics`.

### Path Rules

Define path rules to group high-cardinality URLs into stable patterns before publishing metrics. Provide a JSON array via `INCLUDE_PATH_RULES`, ordered from the most specific rule to the most general. Each rule object supports the following keys:

- `host` (required): Exact host name comparison performed against the log entry.
- `pattern` (required): Regular expression applied to the request path.
- `name` (required): Logical name emitted in the `Path` dimension when both host and regex match.
- `method` (optional): HTTP method to match (case-insensitive). When omitted, the rule matches any method.

```json
[
  {"host":"example.com","pattern":"^/users/[0-9]+$","name":"/users/:id","method":"GET"},
  {"host":"example.com","pattern":"^/articles/(?:[a-z0-9-]+)/comments$","name":"/article/:slug/comments"},
  {"host":"admin.example.com","pattern":"^/dashboard(?:/.*)?$","name":"/dashboard/*","method":"POST"}
]
```

This configuration performs the following transformations when both host and path match:

- `https://example.com/users/42` → `/users/:id`
- `https://example.com/articles/next-gen-observability/comments` → `/article/:slug/comments`
- `https://admin.example.com/dashboard/settings` → `/dashboard/*`

Log entries that do not match any rule are ignored to prevent Path dimension cardinality from exploding.

## Metrics

| Name | Unit | Value |
|------|------|-------|
| `TargetResponseTime` | Seconds | `target_processing_time` field in the ALB access log |
| `RequestCount` | Count | Always 1 for each processed request |
| `FailedRequestCount` | Count | 1 for requests with 5xx responses, otherwise omitted |

## Dimensions

| Name | Description | Example |
|------|-------------|---------|
| `Method` | HTTP method extracted from the ALB log entry | `GET` |
| `Host` | Request host used to route traffic | `api.example.com` |
| `Path` | Normalized logical path name after applying `INCLUDE_PATH_RULES` | `/users/:id` |

## Development

To build and run the project locally using Docker, use the following commands:
```
INCLUDE_PATH_RULES='[{\"host\":\"example.com\",\"method\":\"GET\",\"pattern\":\"^/users/[0-9]+$\",\"name\":\"/users/:id\"}]' script/deploy.sh
```

Invoke the function with a test event:
```
BUCKET=my-alb-logs-bucket KEY=path/to/logfile.log script/invoke.sh
```

Another option is to run the function locally using Docker:
```
# Build and run the Docker image
docker build -t cloudwatch-alb-path-metrics .
INCLUDE_PATH_RULES='[{"host":"example.com","method":"GET","pattern":"^/users/[0-9]+$","name":"/users/:id"}]' scripts/run-local.sh

# Invoke the function with a test event
BUCKET=my-alb-logs-bucket KEY=path/to/logfile.log scripts/invoke-local.sh
```
