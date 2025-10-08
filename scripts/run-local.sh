#!/usr/bin/env bash

container build . -t $(basename $(pwd))
credentials=$(aws configure export-credentials)
container run \
  --rm \
  -e AWS_REGION=ap-northeast-1 \
  -e AWS_ACCESS_KEY_ID=$(echo "$credentials" | jq -r '.AccessKeyId') \
  -e AWS_SECRET_ACCESS_KEY=$(echo "$credentials" | jq -r '.SecretAccessKey') \
  -e AWS_SESSION_TOKEN=$(echo "$credentials" | jq -r '.SessionToken') \
  -p 9000:8080 \
  cloudwatch-alb-path-metrics

