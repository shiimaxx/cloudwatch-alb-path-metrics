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
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type MetricsProcessor struct {
	s3Client   *s3.Client
	rules      *pathRules
	aggregator *MetricAggregator
	publisher  *CloudWatchMetricPublisher
}

func (p *MetricsProcessor) HandleEvent(ctx context.Context, s3Event events.S3Event) error {
	if p == nil {
		return fmt.Errorf("metrics processor is nil")
	}

	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		if bucket == "" {
			return fmt.Errorf("missing bucket name in S3 event record")
		}

		key, err := url.QueryUnescape(record.S3.Object.Key)
		if err != nil {
			return fmt.Errorf("decode object key %q: %w", record.S3.Object.Key, err)
		}

		if err := p.streamObjectLines(ctx, bucket, key); err != nil {
			return fmt.Errorf("stream s3://%s/%s: %w", bucket, key, err)
		}
	}

	if p.aggregator == nil {
		return fmt.Errorf("metric aggregator is nil")
	}

	metricData := p.aggregator.GetCloudWatchMetricData()
	if len(metricData) == 0 {
		return nil
	}

	if p.publisher == nil {
		return fmt.Errorf("metric publisher is nil")
	}

	if err := p.publisher.Publish(ctx, metricData); err != nil {
		return fmt.Errorf("publish metrics: %w", err)
	}

	p.logMetrics(metricData)

	return nil
}

func (p *MetricsProcessor) streamObjectLines(ctx context.Context, bucket, key string) error {
	if p.s3Client == nil {
		return fmt.Errorf("s3 client is nil")
	}

	resp, err := p.s3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
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
		entry, route, ok := normalizeLogLine(line, p.rules)
		if !ok {
			continue
		}
		if p.aggregator != nil {
			p.aggregator.Record(*entry, route)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan gzip stream: %w", err)
	}

	return nil
}

func (p *MetricsProcessor) logMetrics(metricData []types.MetricDatum) {
	for _, data := range metricData {
		if data.MetricName == nil {
			continue
		}

		switch aws.ToString(data.MetricName) {
		case metricNameResponseTime:
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Values: %v, Counts: %v\n",
				aws.ToString(data.MetricName),
				data.Dimensions,
				data.Timestamp,
				data.Values,
				data.Counts,
			)
		case metricNameRequestCount, metricNameFailedRequestCount:
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Value: %v\n",
				aws.ToString(data.MetricName),
				data.Dimensions,
				data.Timestamp,
				aws.ToFloat64(data.Value),
			)
		}
	}
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	namespace := os.Getenv("NAMESPACE")
	if err := validateCloudWatchNamespace(namespace); err != nil {
		return fmt.Errorf("invalid CloudWatch namespace: %w", err)
	}

	rules, err := newPathRules(os.Getenv("INCLUDE_PATH_RULES"))
	if err != nil {
		return fmt.Errorf("parse path rules: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	processor := &MetricsProcessor{
		s3Client:   s3.NewFromConfig(cfg),
		rules:      rules,
		aggregator: &MetricAggregator{metrics: make(map[metricKey]*metricAggregate)},
		publisher: &CloudWatchMetricPublisher{
			client:       cloudwatch.NewFromConfig(cfg),
			namespace:    namespace,
			maxBatchSize: defaultMetricBatchSize,
		},
	}

	return processor.HandleEvent(ctx, s3Event)
}

func validateCloudWatchNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("NAMESPACE environment variable is required")
	}

	if len(namespace) > 255 {
		return fmt.Errorf("namespace must be at most 255 characters")
	}

	if strings.HasPrefix(namespace, "AWS/") {
		return fmt.Errorf("namespace must not start with 'AWS/'")
	}

	for i := 0; i < len(namespace); i++ {
		b := namespace[i]
		if b < 32 || b > 126 {
			return fmt.Errorf("namespace must contain only printable ASCII characters; found 0x%X at position %d", b, i+1)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
