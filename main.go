package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const defaultCloudWatchNamespace = "ALBAccessLog"

func handler(ctx context.Context, s3Event events.S3Event) error {
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
			namespace:    defaultCloudWatchNamespace,
			maxBatchSize: defaultMetricBatchSize,
		},
	}

	return processor.HandleEvent(ctx, s3Event)
}

func main() {
	lambda.Start(handler)
}
