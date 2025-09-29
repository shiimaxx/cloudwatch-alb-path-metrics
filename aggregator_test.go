package main

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricKey_String(t *testing.T) {
	key := MetricKey{
		Method: "GET",
		Host:   "example.com",
		Path:   "/api/users",
	}

	want := "GET\texample.com\t/api/users"
	assert.Equal(t, want, key.String())
}

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

func TestMetricAggregator_AddDuration(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	key1 := MetricKey{Method: "GET", Host: "example.com", Path: "/api/users"}
	key2 := MetricKey{Method: "POST", Host: "example.com", Path: "/api/users"}

	timestamp1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timestamp2 := time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC)
	timestamp3 := time.Date(2023, 1, 1, 12, 2, 0, 0, time.UTC)

	aggregator.AddDuration(key1, 0.1, timestamp1)
	aggregator.AddDuration(key1, 0.2, timestamp2)
	aggregator.AddDuration(key2, 0.3, timestamp3)

	assert.Len(t, aggregator.metrics, 2)
	assert.Contains(t, aggregator.metrics, key1.String())
	assert.Contains(t, aggregator.metrics, key2.String())

	metricValues1 := aggregator.metrics[key1.String()]
	assert.Len(t, metricValues1, 2)
	assert.Equal(t, 0.1, metricValues1[0].Duration)
	assert.Equal(t, timestamp1, metricValues1[0].Timestamp)
	assert.Equal(t, 0.2, metricValues1[1].Duration)
	assert.Equal(t, timestamp2, metricValues1[1].Timestamp)

	metricValues2 := aggregator.metrics[key2.String()]
	assert.Len(t, metricValues2, 1)
	assert.Equal(t, 0.3, metricValues2[0].Duration)
	assert.Equal(t, timestamp3, metricValues2[0].Timestamp)
}

func TestMetricAggregator_GetCloudWatchMetricData(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	key := MetricKey{Method: "GET", Host: "example.com", Path: "/api/users"}
	timestamp1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timestamp2 := time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC)
	timestamp3 := time.Date(2023, 1, 1, 12, 2, 0, 0, time.UTC)

	aggregator.AddDuration(key, 0.1, timestamp1)
	aggregator.AddDuration(key, 0.2, timestamp2)
	aggregator.AddDuration(key, 0.3, timestamp3)

	metricData := aggregator.GetCloudWatchMetricData()

	require.Len(t, metricData, 1)

	metric := metricData[0]

	assert.Equal(t, metricNameDuration, *metric.MetricName)
	assert.Equal(t, types.StandardUnitSeconds, metric.Unit)
	require.Len(t, metric.Dimensions, 4)

	dimensionsMap := make(map[string]string)
	for _, dim := range metric.Dimensions {
		dimensionsMap[*dim.Name] = *dim.Value
	}

	assert.Equal(t, "TestService", dimensionsMap["Service"])
	assert.Equal(t, "GET", dimensionsMap["Method"])
	assert.Equal(t, "example.com", dimensionsMap["Host"])
	assert.Equal(t, "/api/users", dimensionsMap["Path"])

	require.Len(t, metric.Values, 3)
	require.Len(t, metric.Counts, 3)

	assert.Contains(t, metric.Values, 0.1)
	assert.Contains(t, metric.Values, 0.2)
	assert.Contains(t, metric.Values, 0.3)

	// All counts should be 1.0
	for _, count := range metric.Counts {
		assert.Equal(t, 1.0, count)
	}
}

func TestMetricAggregator_GetCloudWatchMetricData_MultipleKeys(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	key1 := MetricKey{Method: "GET", Host: "example.com", Path: "/api/users"}
	key2 := MetricKey{Method: "POST", Host: "example.com", Path: "/api/orders"}

	timestamp1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timestamp2 := time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC)
	timestamp3 := time.Date(2023, 1, 1, 12, 2, 0, 0, time.UTC)

	aggregator.AddDuration(key1, 0.1, timestamp1)
	aggregator.AddDuration(key1, 0.2, timestamp2)
	aggregator.AddDuration(key2, 0.3, timestamp3)

	metricData := aggregator.GetCloudWatchMetricData()

	require.Len(t, metricData, 2)

	// Check that we have metrics for both keys
	metricsByPath := make(map[string]types.MetricDatum)
	for _, metric := range metricData {
		for _, dim := range metric.Dimensions {
			if *dim.Name == "Path" {
				metricsByPath[*dim.Value] = metric
			}
		}
	}

	require.Contains(t, metricsByPath, "/api/users")
	require.Contains(t, metricsByPath, "/api/orders")

	usersMetric := metricsByPath["/api/users"]
	assert.Len(t, usersMetric.Values, 2)

	ordersMetric := metricsByPath["/api/orders"]
	assert.Len(t, ordersMetric.Values, 1)
}

func TestMetricAggregator_GetCloudWatchMetricData_EmptyMetrics(t *testing.T) {
	aggregator := NewMetricAggregator("TestNamespace", "TestService")

	metricData := aggregator.GetCloudWatchMetricData()

	assert.Empty(t, metricData)
}

func TestParseMetricKey(t *testing.T) {
	tests := []struct {
		name     string
		keyStr   string
		expected MetricKey
		hasError bool
	}{
		{
			name:   "valid key",
			keyStr: "GET\texample.com\t/api/users",
			expected: MetricKey{
				Method: "GET",
				Host:   "example.com",
				Path:   "/api/users",
			},
			hasError: false,
		},
		{
			name:     "too few parts",
			keyStr:   "GET\texample.com",
			expected: MetricKey{},
			hasError: true,
		},
		{
			name:     "empty key",
			keyStr:   "",
			expected: MetricKey{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMetricKey(tt.keyStr)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
