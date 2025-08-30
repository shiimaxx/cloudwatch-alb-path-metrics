package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, s3Event events.S3Event) (string, error) {
	for _, record := range s3Event.Records {
		fmt.Println(record.S3.Object.Key)
	}
	return "Hello, World!", nil
}

func main() {
	lambda.Start(handler)
}
