#!/usr/bin/env bash

curl http://localhost:9000/2015-03-31/functions/function/invocations -d "
{
  \"Records\":
    [
      {\"s3\":{\"bucket\":{\"name\":\"${BUCKET}\"},\"object\":{\"key\":\"${KEY}\"}}}
    ]
}"

