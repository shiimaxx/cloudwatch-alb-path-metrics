package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client
var cwClient *cloudwatch.Client

func processLogEntry(entries []string) {
	for _, entry := range entries {
		sp := strings.Split(entry, " ")
		t := sp[1]
		requestProcessingTime := sp[5]
		targetProcessingTime := sp[6]
		responseProcessingTime := sp[7]
		elbStatusCode := sp[8]
		targetStatusCode := sp[9]
		request := sp[12]

		fmt.Printf("Time: %s,ELB Status: %s, Target Status: %s, Request Time: %s, Target Time: %s, Response Time: %s, Request: %s\n",
			t, elbStatusCode, targetStatusCode, requestProcessingTime, targetProcessingTime, responseProcessingTime, request)
	}
}

func processS3Object(ctx context.Context, client *s3.Client, bucket, key string) error {
	fmt.Printf("Processing object %s from bucket %s\n", key, bucket)

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer out.Body.Close()

	zr, err := gzip.NewReader(out.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer zr.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, zr); err != nil {
		return fmt.Errorf("failed to read gzip content: %w", err)
	}

	processLogEntry(strings.Split(buf.String(), "\n"))

	return nil
}

func handler(ctx context.Context, s3Event events.S3Event) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}
	s3Client = s3.NewFromConfig(cfg)
	cwClient = cloudwatch.NewFromConfig(cfg)

	for _, record := range s3Event.Records {
		if err := processS3Object(ctx, s3Client, record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
			fmt.Println("error processing object:", err)
		}
	}
	return "Hello, World!", nil
}

func main() {
	lambda.Start(handler)
}
