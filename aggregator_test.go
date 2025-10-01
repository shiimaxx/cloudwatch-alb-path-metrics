package main

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricAggregator(t *testing.T) {
	aggregator := NewMetricAggregator()

	assert.NotNil(t, aggregator)
	assert.NotNil(t, aggregator.metrics)
	assert.Empty(t, aggregator.metrics)
}

func TestMetricAggregator_RecordAggregatesMetrics(t *testing.T) {
	aggregator := NewMetricAggregator()

	route := "/users/:id"
	time1 := time.Date(2024, 1, 1, 12, 0, 10, 0, time.UTC)
	time2 := time.Date(2024, 1, 1, 12, 0, 45, 0, time.UTC)
	time3 := time.Date(2024, 1, 1, 12, 1, 5, 0, time.UTC)

	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 200, duration: 0.12, timestamp: time1}, route)
	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 502, duration: 0.34, timestamp: time2}, route)
	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 200, duration: 0.56, timestamp: time3}, route)

	assert.Len(t, aggregator.metrics, 2)

	minute1Key := metricKey{Method: "GET", Host: "example.com", Route: route, Minute: time1.Truncate(time.Minute)}
	minute1Agg, ok := aggregator.metrics[minute1Key]
	assert.True(t, ok)
	assert.Equal(t, 1, minute1Agg.successCount)
	assert.Equal(t, 1, minute1Agg.failedCount)
	assert.Equal(t, []float64{0.12, 0.34}, minute1Agg.durations)

	minute2Key := metricKey{Method: "GET", Host: "example.com", Route: route, Minute: time3.Truncate(time.Minute)}
	minute2Agg, ok := aggregator.metrics[minute2Key]
	assert.True(t, ok)
	assert.Equal(t, 1, minute2Agg.successCount)
	assert.Equal(t, 0, minute2Agg.failedCount)
	assert.Equal(t, []float64{0.56}, minute2Agg.durations)
}

func TestMetricAggregator_GetCloudWatchMetricData(t *testing.T) {
	aggregator := NewMetricAggregator()

	route := "/check"
	time1 := time.Date(2024, 2, 1, 8, 0, 10, 0, time.UTC)
	time2 := time.Date(2024, 2, 1, 8, 0, 40, 0, time.UTC)
	time3 := time.Date(2024, 2, 1, 8, 1, 5, 0, time.UTC)

	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 200, duration: 0.5, timestamp: time1}, route)
	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 504, duration: 0.75, timestamp: time2}, route)
	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 200, duration: 0.6, timestamp: time3}, route)

	metricData := aggregator.GetCloudWatchMetricData()
	assert.Len(t, metricData, 6)

	minute1 := time1.Truncate(time.Minute)
	minute2 := time3.Truncate(time.Minute)

	var (
		minute1Response     *types.MetricDatum
		minute1RequestCount *types.MetricDatum
		minute1FailedCount  *types.MetricDatum
		minute2Response     *types.MetricDatum
		minute2RequestCount *types.MetricDatum
		minute2FailedCount  *types.MetricDatum
	)

	for i := range metricData {
		datum := metricData[i]
		ts := time.Time{}
		if datum.Timestamp != nil {
			ts = datum.Timestamp.UTC()
		}

		dims := make(map[string]string)
		for _, dim := range datum.Dimensions {
			dims[*dim.Name] = *dim.Value
		}
		assert.Equal(t, "GET", dims["Method"])
		assert.Equal(t, "api.example.com", dims["Host"])
		assert.Equal(t, route, dims["Route"])
		assert.NotContains(t, dims, "Service")

		switch ts {
		case minute1:
			switch *datum.MetricName {
			case metricNameResponseTime:
				minute1Response = &datum
			case metricNameRequestCount:
				minute1RequestCount = &datum
			case metricNameFailedRequestCount:
				minute1FailedCount = &datum
			default:
				require.Failf(t, "unexpected metric name for minute1", "metric: %s", *datum.MetricName)
			}
		case minute2:
			switch *datum.MetricName {
			case metricNameResponseTime:
				minute2Response = &datum
			case metricNameRequestCount:
				minute2RequestCount = &datum
			case metricNameFailedRequestCount:
				minute2FailedCount = &datum
			default:
				require.Failf(t, "unexpected metric name for minute2", "metric: %s", *datum.MetricName)
			}
		default:
			require.Failf(t, "unexpected timestamp", "%s", ts.String())
		}
	}

	require.NotNil(t, minute1Response)
	require.NotNil(t, minute1RequestCount)
	require.NotNil(t, minute1FailedCount)
	require.NotNil(t, minute2Response)
	require.NotNil(t, minute2RequestCount)
	require.NotNil(t, minute2FailedCount)

	assert.Equal(t, types.StandardUnitSeconds, minute1Response.Unit)
	assert.ElementsMatch(t, []float64{0.5, 0.75}, minute1Response.Values)
	assert.Equal(t, []float64{1, 1}, minute1Response.Counts)
	assert.Equal(t, minute1, minute1Response.Timestamp.UTC())

	assert.Equal(t, types.StandardUnitCount, minute1RequestCount.Unit)
	assert.Equal(t, float64(1), *minute1RequestCount.Value)
	assert.Equal(t, minute1, minute1RequestCount.Timestamp.UTC())

	assert.Equal(t, types.StandardUnitCount, minute1FailedCount.Unit)
	assert.Equal(t, float64(1), *minute1FailedCount.Value)
	assert.Equal(t, minute1, minute1FailedCount.Timestamp.UTC())

	assert.Equal(t, types.StandardUnitSeconds, minute2Response.Unit)
	assert.ElementsMatch(t, []float64{0.6}, minute2Response.Values)
	assert.Equal(t, []float64{1}, minute2Response.Counts)
	assert.Equal(t, minute2, minute2Response.Timestamp.UTC())

	assert.Equal(t, types.StandardUnitCount, minute2RequestCount.Unit)
	assert.Equal(t, float64(1), *minute2RequestCount.Value)
	assert.Equal(t, minute2, minute2RequestCount.Timestamp.UTC())

	assert.Equal(t, types.StandardUnitCount, minute2FailedCount.Unit)
	assert.Equal(t, float64(0), *minute2FailedCount.Value)
	assert.Equal(t, minute2, minute2FailedCount.Timestamp.UTC())
}

func TestMetricAggregator_GetCloudWatchMetricData_EmptyMetrics(t *testing.T) {
	aggregator := NewMetricAggregator()

	metricData := aggregator.GetCloudWatchMetricData()

	assert.Empty(t, metricData)
}
