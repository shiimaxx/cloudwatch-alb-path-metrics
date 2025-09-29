package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestChunkMetricData_SplitsIntoExpectedBatchSizes(t *testing.T) {
	metrics := make([]types.MetricDatum, 7)
	for i := range metrics {
		name := aws.String(fmt.Sprintf("metric-%d", i))
		metrics[i] = types.MetricDatum{MetricName: name}
	}

	batches, err := chunkMetricData(metrics, 3)
	if err != nil {
		t.Fatalf("chunkMetricData returned error: %v", err)
	}

	expectedBatchCounts := []int{3, 3, 1}
	if len(batches) != len(expectedBatchCounts) {
		t.Fatalf("expected %d batches, got %d", len(expectedBatchCounts), len(batches))
	}

	idx := 0
	for i, expected := range expectedBatchCounts {
		if len(batches[i]) != expected {
			t.Fatalf("batch %d expected size %d, got %d", i, expected, len(batches[i]))
		}

		for _, datum := range batches[i] {
			if datum.MetricName != metrics[idx].MetricName {
				t.Fatalf("metric order mismatch at position %d", idx)
			}
			idx++
		}
	}
}

func TestChunkMetricData_HandlesEmptyInput(t *testing.T) {
	batches, err := chunkMetricData([]types.MetricDatum{}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(batches) != 0 {
		t.Fatalf("expected no batches, got %d", len(batches))
	}
}

func TestChunkMetricData_InvalidSize(t *testing.T) {
	if _, err := chunkMetricData(make([]types.MetricDatum, 1), 0); err == nil {
		t.Fatalf("expected error for invalid chunk size")
	}
}
