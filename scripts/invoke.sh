#!/usr/bin/env bash

aws lambda invoke \
  --function-name cloudwatch-alb-path-metrics \
  out \
  --log-type Tail \
  --query 'LogResult' \
  --output text \
  --cli-binary-format raw-in-base64-out \
  --payload "
{
  \"Records\":
  [
    {\"s3\":{\"bucket\":{\"name\":\"${BUCKET}\"},\"object\":{\"key\":\"${KEY}\"}}}
  ]
}" | base64 -d
