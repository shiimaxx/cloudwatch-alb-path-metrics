#!/bin/bash

GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o dist/bootstrap cmd/cloudwatch-alb-path-metrics/main.go
cd dist
zip cloudwatch-alb-path-metrics.zip bootstrap

# aws lambda create-function \
#   --function-name cloudwatch-alb-path-metrics \
#   --runtime provided.al2023 \
#   --handler bootstrap \
#   --architectures arm64 \
#   --zip-file fileb://cloudwatch-alb-path-metrics.zip

aws lambda update-function-code \
  --no-cli-pager \
  --function-name cloudwatch-alb-path-metrics \
  --zip-file fileb://cloudwatch-alb-path-metrics.zip

sleep 3

aws lambda update-function-configuration \
  --no-cli-pager \
  --function-name cloudwatch-alb-path-metrics \
  --environment Variables="
{
  INCLUDE_PATH_RULES=\"${INCLUDE_PATH_RULES}\",
	DRY_RUN=\"${DRY_RUN}\"
}"
