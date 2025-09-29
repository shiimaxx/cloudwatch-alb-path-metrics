#!/bin/bash

GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o dist/bootstrap ./...
cd dist
zip cloudwatch-alb-path-metrics.zip bootstrap
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
  NAMESPACE=\"shiimaxx\",
  SERVICE=\"hello\",
  INCLUDE_PATH_RULES=\"${INCLUDE_PATH_RULES}\",
}"
