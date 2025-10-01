package main

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

	batches, err := chunkMetricData(metrics, 3)
	require.NoError(t, err)

	expectedBatchCounts := []int{3, 3, 1}
	require.Lenf(t, batches, len(expectedBatchCounts), "expected %d batches", len(expectedBatchCounts))

	idx := 0
	for i, expected := range expectedBatchCounts {
		require.Lenf(t, batches[i], expected, "batch %d expected size %d", i, expected)
		for _, datum := range batches[i] {
			require.Equalf(t, metrics[idx].MetricName, datum.MetricName, "metric order mismatch at position %d", idx)
			idx++
		}
	}
}

func TestChunkMetricData_HandlesEmptyInput(t *testing.T) {
	batches, err := chunkMetricData([]types.MetricDatum{}, 5)
	require.NoError(t, err)
	assert.Empty(t, batches)
}

func TestChunkMetricData_InvalidSize(t *testing.T) {
	_, err := chunkMetricData(make([]types.MetricDatum, 1), 0)
	assert.Error(t, err)
}
