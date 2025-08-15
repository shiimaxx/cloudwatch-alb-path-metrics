package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/shiimaxx/cloudwatch-alb-path-metrics/internal/metrics"
	"github.com/shiimaxx/cloudwatch-alb-path-metrics/internal/publisher"
	appConfig "github.com/shiimaxx/cloudwatch-alb-path-metrics/pkg/config"
)

type Handler struct {
	s3Client            *s3.Client
	cloudwatchClient    *cloudwatch.Client
	config              *appConfig.Config
	aggregator          *metrics.Aggregator
	publisher           *publisher.CloudWatchPublisher
}

func NewHandler() (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	appCfg := appConfig.Load()

	s3Client := s3.NewFromConfig(cfg)
	cloudwatchClient := cloudwatch.NewFromConfig(cfg)

	aggregator := metrics.NewAggregator(appCfg)
	cwPublisher := publisher.NewCloudWatchPublisher(cloudwatchClient, appCfg)

	return &Handler{
		s3Client:         s3Client,
		cloudwatchClient: cloudwatchClient,
		config:           appCfg,
		aggregator:       aggregator,
		publisher:        cwPublisher,
	}, nil
}

func (h *Handler) HandleS3Event(ctx context.Context, event events.S3Event) error {
	log.Printf("Processing %d S3 records", len(event.Records))

	for _, record := range event.Records {
		if err := h.processS3Record(ctx, record.S3); err != nil {
			log.Printf("Failed to process S3 record %s/%s: %v", 
				record.S3.Bucket.Name, record.S3.Object.Key, err)
			return err
		}
	}

	return nil
}

func (h *Handler) processS3Record(ctx context.Context, s3Record events.S3Entity) error {
	bucketName := s3Record.Bucket.Name
	objectKey := s3Record.Object.Key

	log.Printf("Processing S3 object: s3://%s/%s", bucketName, objectKey)

	logData, err := h.downloadS3Object(ctx, bucketName, objectKey)
	if err != nil {
		return fmt.Errorf("failed to download S3 object: %w", err)
	}

	if len(logData) == 0 {
		log.Printf("S3 object is empty, skipping: s3://%s/%s", bucketName, objectKey)
		return nil
	}

	metricData, err := h.aggregator.AggregateFromLogData(logData)
	if err != nil {
		return fmt.Errorf("failed to aggregate metrics: %w", err)
	}

	if len(metricData) == 0 {
		log.Printf("No metrics generated from S3 object: s3://%s/%s", bucketName, objectKey)
		return nil
	}

	if err := h.publisher.ValidateMetricData(metricData); err != nil {
		return fmt.Errorf("metric data validation failed: %w", err)
	}

	if err := h.publisher.PublishMetrics(ctx, metricData); err != nil {
		return fmt.Errorf("failed to publish metrics: %w", err)
	}

	log.Printf("Successfully processed %d metric data points from s3://%s/%s", 
		len(metricData), bucketName, objectKey)

	return nil
}

func (h *Handler) downloadS3Object(ctx context.Context, bucket, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	result, err := h.s3Client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	data := make([]byte, 0)
	buffer := make([]byte, 1024*1024) // 1MB buffer

	for {
		n, err := result.Body.Read(buffer)
		if n > 0 {
			data = append(data, buffer[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to read S3 object: %w", err)
		}
	}

	return data, nil
}

func main() {
	handler, err := NewHandler()
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	lambda.Start(handler.HandleS3Event)
}
