package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func handler(ctx context.Context, s3Event events.S3Event) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	cwClient := cloudwatch.NewFromConfig(cfg)

	namespace := strings.TrimSpace(os.Getenv("NAMESPACE"))
	if namespace == "" {
		return fmt.Errorf("NAMESPACE environment variable is required")
	}

	service := strings.TrimSpace(os.Getenv("SERVICE"))
	if service == "" {
		return fmt.Errorf("SERVICE environment variable is required")
	}

	rules, err := newPathRules(os.Getenv("INCLUDE_PATH_RULES"))
	if err != nil {
		return fmt.Errorf("parse path rules: %w", err)
	}

	aggregator := NewMetricAggregator(namespace, service)
	publisher := NewCloudWatchMetricPublisher(cwClient, namespace)

	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		if bucket == "" {
			return fmt.Errorf("missing bucket name in S3 event record")
		}

		key, err := url.QueryUnescape(record.S3.Object.Key)
		if err != nil {
			return fmt.Errorf("decode object key %q: %w", record.S3.Object.Key, err)
		}

		if err := streamObjectLines(ctx, s3Client, bucket, key, rules, aggregator); err != nil {
			return fmt.Errorf("stream s3://%s/%s: %w", bucket, key, err)
		}
	}

	metricData := aggregator.GetCloudWatchMetricData()
	if len(metricData) == 0 {
		return nil
	}

	if err := publisher.Publish(ctx, metricData); err != nil {
		return fmt.Errorf("publish metrics: %w", err)
	}

	for _, data := range metricData {
		if *data.MetricName == metricNameResponseTime {
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Values: %v, Counts: %v\n",
				aws.ToString(data.MetricName),
				data.Dimensions,
				data.Timestamp,
				data.Values,
				data.Counts,
			)
		}

		if *data.MetricName == metricNameRequestCount || *data.MetricName == metricNameFailedRequestCount {
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Value: %v\n",
				aws.ToString(data.MetricName),
				data.Dimensions,
				data.Timestamp,
				aws.ToFloat64(data.Value),
			)
		}
	}

	return nil
}

func streamObjectLines(ctx context.Context, client *s3.Client, bucket, key string, rules *pathRules, aggregator *MetricAggregator) error {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		line := scanner.Text()
		entry, route, ok := normalizeLogLine(line, rules)
		if !ok {
			continue
		}
		if aggregator != nil {
			aggregator.Record(*entry, route)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan gzip stream: %w", err)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
