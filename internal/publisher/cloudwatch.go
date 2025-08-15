package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/shiimaxx/cloudwatch-alb-path-metrics/pkg/config"
	"github.com/shiimaxx/cloudwatch-alb-path-metrics/pkg/models"
)

const (
	MaxMetricsPerRequest = 1000
	MaxRequestSizeBytes  = 1048576 // 1MB
)

type CloudWatchPublisher struct {
	client *cloudwatch.Client
	config *config.Config
}

func NewCloudWatchPublisher(client *cloudwatch.Client, cfg *config.Config) *CloudWatchPublisher {
	return &CloudWatchPublisher{
		client: client,
		config: cfg,
	}
}

func (p *CloudWatchPublisher) PublishMetrics(ctx context.Context, metricData []*models.MetricData) error {
	if len(metricData) == 0 {
		return nil
	}

	metricBatches := p.createMetricBatches(metricData)

	for i, batch := range metricBatches {
		input := &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(p.config.MetricNamespace),
			MetricData: batch,
		}

		_, err := p.client.PutMetricData(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to publish metrics batch %d: %w", i+1, err)
		}
	}

	return nil
}

func (p *CloudWatchPublisher) createMetricBatches(metricData []*models.MetricData) [][]types.MetricDatum {
	var batches [][]types.MetricDatum
	var currentBatch []types.MetricDatum
	
	for _, data := range metricData {
		metricDatums := p.convertToCloudWatchMetrics(data)
		
		for _, datum := range metricDatums {
			if len(currentBatch) >= MaxMetricsPerRequest {
				batches = append(batches, currentBatch)
				currentBatch = []types.MetricDatum{}
			}
			currentBatch = append(currentBatch, datum)
		}
	}
	
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}
	
	return batches
}

func (p *CloudWatchPublisher) convertToCloudWatchMetrics(data *models.MetricData) []types.MetricDatum {
	dimensions := []types.Dimension{
		{
			Name:  aws.String("path"),
			Value: aws.String(data.NormalizedRoute),
		},
	}
	
	if p.config.Environment != "" {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String("environment"),
			Value: aws.String(data.Environment),
		})
	}
	
	var metrics []types.MetricDatum
	
	requestsTotalMetric := types.MetricDatum{
		MetricName: aws.String("RequestsTotal"),
		Value:      aws.Float64(float64(data.RequestsTotal)),
		Unit:       types.StandardUnitCount,
		Timestamp:  aws.Time(data.Timestamp),
		Dimensions: dimensions,
	}
	metrics = append(metrics, requestsTotalMetric)
	
	requestsGoodMetric := types.MetricDatum{
		MetricName: aws.String("RequestsGood"),
		Value:      aws.Float64(float64(data.RequestsGood)),
		Unit:       types.StandardUnitCount,
		Timestamp:  aws.Time(data.Timestamp),
		Dimensions: dimensions,
	}
	metrics = append(metrics, requestsGoodMetric)
	
	if data.LatencyCount > 0 {
		latencyMetric := types.MetricDatum{
			MetricName: aws.String("Latency"),
			Value:      aws.Float64(data.AverageLatency()),
			Unit:       types.StandardUnitSeconds,
			Timestamp:  aws.Time(data.Timestamp),
			Dimensions: dimensions,
		}
		metrics = append(metrics, latencyMetric)
	}
	
	return metrics
}

func (p *CloudWatchPublisher) ValidateMetricData(metricData []*models.MetricData) error {
	for i, data := range metricData {
		if data.Service == "" {
			return fmt.Errorf("metric %d: service name is required", i)
		}
		if data.NormalizedRoute == "" {
			return fmt.Errorf("metric %d: normalized route is required", i)
		}
		if data.Timestamp.IsZero() {
			return fmt.Errorf("metric %d: timestamp is required", i)
		}
		if data.Timestamp.After(time.Now().Add(2 * time.Hour)) {
			return fmt.Errorf("metric %d: timestamp cannot be more than 2 hours in the future", i)
		}
		if data.Timestamp.Before(time.Now().Add(-14 * 24 * time.Hour)) {
			return fmt.Errorf("metric %d: timestamp cannot be more than 14 days in the past", i)
		}
	}
	
	return nil
}