package main

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricAggregator(t *testing.T) {
	namespace := "TestNamespace"
	service := "TestService"

	aggregator := NewMetricAggregator(namespace, service)

	assert.NotNil(t, aggregator)
	assert.Equal(t, namespace, aggregator.namespace)
	assert.Equal(t, service, aggregator.service)
	assert.NotNil(t, aggregator.metrics)
	assert.Empty(t, aggregator.metrics)
}

func TestMetricAggregator_RecordAggregatesMetrics(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	route := "/users/:id"
	time1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC)

	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 200, duration: 0.12, timestamp: time1}, route)
	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 502, duration: 0.34, timestamp: time2}, route)

	require.Len(t, aggregator.metrics, 1)

	key := metricKey{Method: "GET", Host: "example.com", Route: route}
	agg, ok := aggregator.metrics[key]
	require.True(t, ok)
	assert.Equal(t, 1, agg.successCount)
	assert.Equal(t, 1, agg.failedCount)
	assert.Equal(t, []float64{0.12, 0.34}, agg.durations)
	assert.Equal(t, time2, agg.latestRecorded)
}

func TestMetricAggregator_GetCloudWatchMetricData(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	route := "/check"
	time1 := time.Date(2024, 2, 1, 8, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 2, 1, 8, 1, 0, 0, time.UTC)

	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 200, duration: 0.5, timestamp: time1}, route)
	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 504, duration: 0.75, timestamp: time2}, route)

	metricData := aggregator.GetCloudWatchMetricData()
	require.Len(t, metricData, 3)

	metricsByName := make(map[string]types.MetricDatum)
	for _, datum := range metricData {
		metricsByName[*datum.MetricName] = datum
	}

	require.Contains(t, metricsByName, metricNameResponseTime)
	require.Contains(t, metricsByName, metricNameRequestCount)
	require.Contains(t, metricsByName, metricNameFailedRequestCount)

	response := metricsByName[metricNameResponseTime]
	assert.Equal(t, types.StandardUnitSeconds, response.Unit)
	assert.ElementsMatch(t, []float64{0.5, 0.75}, response.Values)
	assert.Equal(t, []float64{1, 1}, response.Counts)
	assert.Equal(t, time2, *response.Timestamp)

	requestCount := metricsByName[metricNameRequestCount]
	assert.Equal(t, types.StandardUnitCount, requestCount.Unit)
	assert.Equal(t, float64(1), *requestCount.Value)
	assert.Equal(t, time2, *requestCount.Timestamp)

	failedCount := metricsByName[metricNameFailedRequestCount]
	assert.Equal(t, types.StandardUnitCount, failedCount.Unit)
	assert.Equal(t, float64(1), *failedCount.Value)
	assert.Equal(t, time2, *failedCount.Timestamp)

	for _, datum := range metricData {
		dimensions := make(map[string]string)
		for _, dim := range datum.Dimensions {
			dimensions[*dim.Name] = *dim.Value
		}

		assert.Equal(t, "TestService", dimensions["Service"])
		assert.Equal(t, "GET", dimensions["Method"])
		assert.Equal(t, "api.example.com", dimensions["Host"])
		assert.Equal(t, route, dimensions["Route"])
	}
}

func TestMetricAggregator_GetCloudWatchMetricData_EmptyMetrics(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	metricData := aggregator.GetCloudWatchMetricData()

	assert.Empty(t, metricData)
}
