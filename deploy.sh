#!/bin/bash

GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o dist/bootstrap main.go
cd dist
zip cloudwatch-alb-path-metrics.zip bootstrap
aws lambda update-function-code \
  --function-name cloudwatch-alb-path-metrics \
  --zip-file fileb://cloudwatch-alb-path-metrics.zip
