package metrics

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkMetricData_SplitsIntoExpectedBatchSizes(t *testing.T) {
	metrics := make([]types.MetricDatum, 7)
	for i := range metrics {
		name := aws.String(fmt.Sprintf("metric-%d", i))
		metrics[i] = types.MetricDatum{MetricName: name}
	}

	size := 3
	publisher := &cloudWatchMetricPublisher{maxBatchSize: size}

	batches, err := publisher.chunkMetricData(metrics)
	require.NoError(t, err)

	wantBatchCounts := []int{3, 3, 1}
	assert.Len(t, batches, len(wantBatchCounts))

	for i, want := range wantBatchCounts {
		assert.Len(t, batches[i], want)

		for j, datum := range batches[i] {
			assert.Equal(t, metrics[i*size+j].MetricName, datum.MetricName)
		}
	}
}

func TestChunkMetricData_HandlesEmptyInput(t *testing.T) {
	publisher := &cloudWatchMetricPublisher{maxBatchSize: 5}
	batches, err := publisher.chunkMetricData([]types.MetricDatum{})
	assert.NoError(t, err)
	assert.Empty(t, batches)
}

func TestChunkMetricData_InvalidSize(t *testing.T) {
	publisher := &cloudWatchMetricPublisher{maxBatchSize: 0}
	_, err := publisher.chunkMetricData(make([]types.MetricDatum, 1))
	assert.Error(t, err)
}
