# cloudwatch-alb-path-metrics

cloudwatch-alb-path-metrics is a Lambda function that generates path-based metrics from Application Load Balancer (ALB) access logs and publishes them as Amazon CloudWatch custom metrics.

## Motivation

When implementing SLI/SLO practices, it is reasonable to leverage the monitoring services provided by the cloud platform.
In typical AWS workloads, metrics from the ALB can be used to measure request availability and latency.

However, ALB’s built-in metrics are only available at the load balancer or target group level.
While these are useful for understanding overall service trends, they do not provide visibility at the request-path level, which is often more relevant to user experience.
If path-based metrics were available, teams could define SLIs aligned with Critical User Journeys (CUJs) and operate SLOs more effectively.

cloudwatch-alb-path-metrics is a simple solution designed for this purpose.
It parses ALB access logs, normalizes request paths, and publishes custom CloudWatch metrics for each path—enabling path-level SLI/SLO measurement entirely within AWS, without introducing additional middleware or external observability systems.

## Installation

```
aws iam create-policy \
  --policy-name cloudwatch-alb-path-metrics-alb-logs-bucket-access \
  --policy-document \
'{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowReadAlbLogs",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:GetObjectAttributes"
      ],
      "Resource": "arn:aws:s3:::<alb logs bucket name>/*"
    },
    {
      "Sid": "AllowListAlbLogBucket",
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::<alb logs bucket name>"
    }
  ]
}'


aws iam create-policy \
  --policy-name cloudwatch-alb-path-metrics-publish \
  --policy-document \
'{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowPutMetrics",
      "Effect": "Allow",
      "Action": "cloudwatch:PutMetricData",
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "cloudwatch:Namespace": "ALBAccessLog"
        }
      }
    },
    {
      "Sid": "AllowLambdaLogs",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    }
  ]
}'

aws iam create-role \
  --role-name CloudWatchALBPathMetrics \
  --max-session-duration 3600 \
  --assume-role-policy-document \
'{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}'

aws iam attach-role-policy \
  --role-name CloudWatchALBPathMetrics \
  --policy-arn arn:aws:iam::<account-id>:policy/cloudwatch-alb-path-metrics-alb-logs-bucket-access

aws iam attach-role-policy \
  --role-name CloudWatchALBPathMetrics \
  --policy-arn arn:aws:iam::<account-id>:policy/cloudwatch-alb-path-metrics-publish

aws iam attach-role-policy \
  --role-name CloudWatchALBPathMetrics \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

make

aws lambda create-function \
  --function-name cloudwatch-alb-path-metrics \
  --runtime provided.al2023 \
  --handler bootstrap \
  --architectures arm64 \
  --zip-file fileb://dist/cloudwatch-alb-path-metrics.zip \
  --role arn:aws:iam::<account-id>:role/CloudWatchALBPathMetrics \
  --environment Variables=\
"{
  INCLUDE_PATH_RULES='[{\"host\":\"example.com\",\"method":\"GET\",\"pattern\":\"^/users/[0-9]+$\",\"name\":\"/users/:id\"}]'
}"
```

## Configuration

### INCLUDE_PATH_RULES

INCLUDE_PATH_RULES defines which request paths should be published as metrics.
It accepts a JSON array describing host and path-matching rules.

Publishing metrics for every unique path in ALB access logs can easily lead to a high-cardinality explosion in the Path dimension,
which increases CloudWatch costs and reduces the usefulness of aggregated metrics.
To avoid this, the tool is designed to emit metrics only for a small set of important endpoints that represent your SLI targets.

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
- `https://example.com/articles/hello-world/comments` → `/article/:slug/comments`
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
