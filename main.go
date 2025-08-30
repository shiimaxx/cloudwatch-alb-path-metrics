package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func processS3Object(client *s3.Client, bucket, key string) {
	fmt.Printf("Processing object %s from bucket %s\n", key, bucket)
}

func handler(ctx context.Context, s3Event events.S3Event) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	for _, record := range s3Event.Records {
		processS3Object(s3Client, record.S3.Bucket.Name, record.S3.Object.Key)
	}
	return "Hello, World!", nil
}

func main() {
	lambda.Start(handler)
}
