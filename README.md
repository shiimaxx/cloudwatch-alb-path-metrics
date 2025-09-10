# cloudwatch-alb-path-metrics

A Lambda function that aggregates ALB logs and publishes path-based CloudWatch custom metrics

## Overview

This project provides a Lambda function that analyzes Application Load Balancer (ALB) access logs and publishes path-based HTTP request performance metrics as Amazon CloudWatch custom metrics. The key feature is collecting metrics organized by request paths, enabling detailed monitoring of individual API endpoints or URL patterns.

## Motivation

ALB-level metrics serve as an effective starting point for SLI/SLO implementation. However, for systems where a single ALB handles all requests, this approach cannot provide meaningful SLIs for individual features or services since all requests are aggregated together.

This project provides a quick and easy solution for path-based metrics collection using only AWS services, enabling granular and actionable SLI/SLO monitoring for each API endpoint or service path.

## Features

### Core Functionality

- **Flexible Filtering**: Filter log entries for metrics conversion by HTTP method, host, and path
- **Path Grouping**: Group paths using regular expressions to control cardinality
- **Duration Metrics**: Measure processing time (seconds) from HTTP request reception to response delivery

### Deployment Model

- **S3 Triggered Execution**: Lambda function automatically executes when ALB logs are stored in S3

### Metrics Specification

| Metric | Description |
|--------|-------------|
| Duration | Time taken from receiving HTTP request to returning response.<br><br>**Reporting criteria:** There is a nonzero value<br><br>**Statistics:** The most useful statistics are Average and pNN.NN (percentiles). <br><br>**Dimentions** <br>- `Service`,`Method`,`Host`,`Path` |

## Configuration

### Environment Variables

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `NAMESPACE` | CloudWatch custom metrics namespace | Yes | `MyApplication/ALB` |
| `SERVICE` | Service name | Yes | `web-api` |
| `FILTER` | Log entry filtering conditions ([expr](https://github.com/expr-lang/expr) format) | No | `method == "GET" && status_code < 400` |
| `PATH_GROUP_REGEXES` | Regular expressions for path grouping (comma-separated) | No | `/api/users/\d+,/api/orders/\d+` |

### Filter Condition Examples

```bash
# Only GET requests
FILTER='method == "GET"'

# Exclude specific paths
FILTER='!contains(path, "/health") && !contains(path, "/metrics")'

# Filter by specific host
FILTER='host == "api.example.com"'

# Combine multiple conditions
FILTER='method == "POST" && contains(path, "/api/") && host == "api.example.com"'
```

### Path Grouping Examples

```bash
# Group paths containing user IDs or order IDs
PATH_GROUP_REGEXES='/api/users/\d+,/api/orders/\d+,/api/products/[^/]+'
```

This configuration performs the following transformations:
- `/api/users/123` → `/api/users/\d+`
- `/api/orders/456` → `/api/orders/\d+`
- `/api/products/abc-def` → `/api/products/[^/]+`

## Architecture

### System Components

```
S3 Bucket (ALB Logs) → Lambda Function → CloudWatch Custom Metrics
```

### Processing Flow

1. ALB outputs access logs to S3
2. S3 event trigger activates Lambda function
3. Lambda function retrieves and parses log files from S3
4. Log entries are filtered based on configured filter conditions
5. Paths are normalized according to path grouping settings
6. Duration metrics are calculated and published to CloudWatch

## Performance Optimization

### Cost Optimization

- **Cardinality Control**: Path grouping feature consolidates Path dimension values to avoid cardinality explosion
- **API Call Reduction**: Utilize CloudWatch `PutMetricData` API with Values and Counts to minimize API calls

### Metrics Aggregation

The Lambda function pre-aggregates metrics with the same dimension combinations and sends them to CloudWatch in batches for efficient metric publishing.

## License

This project is released under the license described in the [LICENSE](LICENSE) file.
