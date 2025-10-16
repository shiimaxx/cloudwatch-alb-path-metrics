package metrics

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const defaultMetricBatchSize = 20

// cloudWatchMetricPublisher sends metric data to CloudWatch using PutMetricData.
type cloudWatchMetricPublisher struct {
	client       *cloudwatch.Client
	namespace    string
	maxBatchSize int
	dryRun       bool
}

// Publish sends metric data to CloudWatch in batches that respect PutMetricData limits.
func (p *cloudWatchMetricPublisher) publish(ctx context.Context, data []types.MetricDatum) error {
	if len(data) == 0 {
		return nil
	}

	chunks, err := p.chunkMetricData(data)
	if err != nil {
		return fmt.Errorf("prepare metric batches: %w", err)
	}

	fmt.Printf("Publishing %d metrics in %d batches to CloudWatch namespace %q\n", len(data), len(chunks), p.namespace)

	if p.dryRun {
		fmt.Println("Dry run enabled, skipping actual publishing")
		return nil
	}

	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}

		input := &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(p.namespace),
			MetricData: chunk,
		}

		if _, err := p.client.PutMetricData(ctx, input); err != nil {
			return fmt.Errorf("put metric data: %w", err)
		}
	}

	return nil
}

// chunkMetricData splits the provided metric data into size-bounded batches.
func (p *cloudWatchMetricPublisher) chunkMetricData(data []types.MetricDatum) ([][]types.MetricDatum, error) {
	size := p.maxBatchSize
	if size <= 0 {
		return nil, fmt.Errorf("invalid max batch size %d", size)
	}

	if len(data) == 0 {
		return [][]types.MetricDatum{}, nil
	}

	batches := make([][]types.MetricDatum, 0, (len(data)+size-1)/size)
	for start := 0; start < len(data); start += size {
		end := min(start+size, len(data))
		batches = append(batches, data[start:end])
	}

	return batches, nil
}
