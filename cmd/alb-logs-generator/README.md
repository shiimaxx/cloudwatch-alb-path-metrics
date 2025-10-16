# alb-logs-generator

Generate dummy ALB logs 100req/s for 5 minutes and upload to S3

```
go run main.go --count $(expr 100 \* 300) | gzip --stdout | aws s3 cp - s3://your-test-bucket/test.log.gz
```

