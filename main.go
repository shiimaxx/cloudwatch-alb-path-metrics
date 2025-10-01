package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func handler(ctx context.Context, s3Event events.S3Event) error {
	namespace := os.Getenv("NAMESPACE")
	if err := validateCloudWatchNamespace(namespace); err != nil {
		return fmt.Errorf("invalid CloudWatch namespace: %w", err)
	}

	rules, err := NewPathRules(os.Getenv("INCLUDE_PATH_RULES"))
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
