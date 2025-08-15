package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, s3Event events.S3Event) (string, error) {
	return "Hello, World!", nil
}

func main() {
	lambda.Start(handler)
}
