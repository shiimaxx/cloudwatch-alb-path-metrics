package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client
var cwClient *cloudwatch.Client

func publishMetrics(ctx context.Context, path string, metrics map[time.Time][]string) error {
	for t, entries := range metrics {
		fmt.Println(t, entries)
	}

	_, err := cwClient.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Shiimaxx"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("RequestCount"),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("Path"),
						Value: aws.String(path),
					},
				},
				Value: aws.Float64(1.0),
				Unit:  types.StandardUnitCount,
			},
		},
	})
	if err != nil {
		fmt.Println("failed to put RequestCount:", err)
	}

	_, err = cwClient.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Shiimaxx"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("SuccessRequestCount"),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("Path"),
						Value: aws.String(path),
					},
				},
				Value: aws.Float64(1.0),
				Unit:  types.StandardUnitCount,
			},
		},
	})
	if err != nil {
		fmt.Println("failed to put SuccessRequestCount:", err)
	}

	_, err = cwClient.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Shiimaxx"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("RequestTime"),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("Path"),
						Value: aws.String(path),
					},
				},
				Values: []float64{123.0},
				Counts: []float64{1},
				Unit:   types.StandardUnitMilliseconds,
			},
		},
	})
	if err != nil {
		fmt.Println("failed to put RequestTime:", err)
	}

	return nil
}

func processLogEntry(ctx context.Context, entries []string) error {
	metrics := make(map[string]map[time.Time][]string)

	for _, entry := range entries {
		sp := strings.Split(entry, " ")
		t := sp[1]
		request := sp[13]
		u, err := url.Parse(request)
		if err != nil {
			fmt.Println("failed to parse URL:", err)
			continue
		}

		if _, ok := metrics[u.Path]; !ok {
			metrics[u.Path] = make(map[time.Time][]string)
		}

		tt, err := time.Parse(time.RFC3339Nano, t)
		if err != nil {
			fmt.Println("failed to parse time:", err)
			continue
		}
		_ = tt.Truncate(time.Minute)

		metrics[u.Path][tt] = append(metrics[u.Path][tt], entry)
	}

	for path, timeMap := range metrics {
		publishMetrics(ctx, path, timeMap)
	}

	return nil
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

	processLogEntry(ctx, strings.Split(buf.String(), "\n"))

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
