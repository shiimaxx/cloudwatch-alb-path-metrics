#!/bin/bash

GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o dist/bootstrap main.go
cd dist
zip cloudwatch-alb-path-metrics.zip bootstrap
aws lambda update-function-code \
  --no-paginate \
  --function-name cloudwatch-alb-path-metrics \
  --zip-file fileb://cloudwatch-alb-path-metrics.zip
aws lambda update-function-configuration \
  --no-paginate \
  --function-name cloudwatch-alb-path-metrics \
  --environment Variables="{SERVICE=hello,PATH_PATTERNS='[{\"name\":\"/graphql\", \"expr\":\"method == \\\"POST\\\" && path matches \\\"/graphql\\\"\"}]'}"
