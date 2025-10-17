package metrics

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
)

func TestMetricAggregator_RecordAggregatesMetrics(t *testing.T) {
	aggregator := &MetricAggregator{metrics: make(map[metricKey]*metricAggregate)}

	name := "/users/:id"
	time1 := parseTime(t, "2024-01-01T12:00:15Z")
	time2 := parseTime(t, "2024-01-01T12:00:45Z")
	time3 := parseTime(t, "2024-01-01T12:01:05Z")

	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 200, targetProcessingTime: 0.12, timestamp: time1}, name)
	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 502, targetProcessingTime: 0.34, timestamp: time2}, name)
	aggregator.Record(albLogEntry{method: "GET", host: "example.com", status: 200, targetProcessingTime: 0.56, timestamp: time3}, name)

	assert.Len(t, aggregator.metrics, 2)

	minute1Key := metricKey{Method: "GET", Host: "example.com", Path: name, Minute: time1.Truncate(time.Minute)}
	minute1Agg, ok := aggregator.metrics[minute1Key]
	assert.True(t, ok)
	assert.Equal(t, 2, minute1Agg.requestCount)
	assert.Equal(t, 1, minute1Agg.failedRequestCount)
	assert.Equal(t, []float64{0.12, 0.34}, minute1Agg.responseTime)

	minute2Key := metricKey{Method: "GET", Host: "example.com", Path: name, Minute: time3.Truncate(time.Minute)}
	minute2Agg, ok := aggregator.metrics[minute2Key]
	assert.True(t, ok)
	assert.Equal(t, 1, minute2Agg.requestCount)
	assert.Equal(t, 0, minute2Agg.failedRequestCount)
	assert.Equal(t, []float64{0.56}, minute2Agg.responseTime)
}

func TestMetricAggregator_GetCloudWatchMetricData(t *testing.T) {
	aggregator := &MetricAggregator{metrics: make(map[metricKey]*metricAggregate)}

	name := "/check"
	time1 := parseTime(t, "2024-02-01T08:00:15Z")
	time2 := parseTime(t, "2024-02-01T08:00:45Z")
	time3 := parseTime(t, "2024-02-01T08:01:05Z")

	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 200, targetProcessingTime: 0.5, timestamp: time1}, name)
	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 504, targetProcessingTime: 0.75, timestamp: time2}, name)
	aggregator.Record(albLogEntry{method: "GET", host: "api.example.com", status: 200, targetProcessingTime: 0.6, timestamp: time3}, name)

	metricData := aggregator.GetCloudWatchMetricData()
	assert.Len(t, metricData, 6)

	md0 := metricData[0]
	assert.Equal(t, metricNameResponseTime, *md0.MetricName)
	assert.Equal(t, "GET", *md0.Dimensions[0].Value)
	assert.Equal(t, "api.example.com", *md0.Dimensions[1].Value)
	assert.Equal(t, name, *md0.Dimensions[2].Value)
	assert.Equal(t, []float64{0.5, 0.75}, md0.Values)
	assert.Equal(t, []float64{1, 1}, md0.Counts)

	md1 := metricData[1]
	assert.Equal(t, metricNameRequestCount, *md1.MetricName)
	assert.Equal(t, "GET", *md1.Dimensions[0].Value)
	assert.Equal(t, "api.example.com", *md1.Dimensions[1].Value)
	assert.Equal(t, name, *md1.Dimensions[2].Value)
	assert.Equal(t, float64(2), *md1.Value)

	md2 := metricData[2]
	assert.Equal(t, metricNameFailedRequestCount, *md2.MetricName)
	assert.Equal(t, "GET", *md2.Dimensions[0].Value)
	assert.Equal(t, "api.example.com", *md2.Dimensions[1].Value)
	assert.Equal(t, name, *md2.Dimensions[2].Value)
	assert.Equal(t, float64(1), *md2.Value)

	md3 := metricData[3]
	assert.Equal(t, metricNameResponseTime, *md3.MetricName)
	assert.Equal(t, "GET", *md3.Dimensions[0].Value)
	assert.Equal(t, "api.example.com", *md3.Dimensions[1].Value)
	assert.Equal(t, name, *md3.Dimensions[2].Value)
	assert.Equal(t, []float64{0.6}, md3.Values)
	assert.Equal(t, []float64{1}, md3.Counts)

	md4 := metricData[4]
	assert.Equal(t, metricNameRequestCount, *md4.MetricName)
	assert.Equal(t, "GET", *md4.Dimensions[0].Value)
	assert.Equal(t, "api.example.com", *md4.Dimensions[1].Value)
	assert.Equal(t, name, *md4.Dimensions[2].Value)
	assert.Equal(t, float64(1), *md4.Value)

	md5 := metricData[5]
	assert.Equal(t, metricNameFailedRequestCount, *md5.MetricName)
	assert.Equal(t, "GET", *md5.Dimensions[0].Value)
	assert.Equal(t, "api.example.com", *md5.Dimensions[1].Value)
	assert.Equal(t, name, *md5.Dimensions[2].Value)
	assert.Equal(t, float64(0), *md5.Value)
}

func TestMetricAggregator_GetCloudWatchMetricData_EmptyMetrics(t *testing.T) {
	aggregator := &MetricAggregator{metrics: make(map[metricKey]*metricAggregate)}
	metricData := aggregator.GetCloudWatchMetricData()
	assert.Empty(t, metricData)
}

func TestMetricAggregator_GetCloudWatchMetricData_GroupsDuplicateResponseTimes(t *testing.T) {
	aggregator := &MetricAggregator{metrics: make(map[metricKey]*metricAggregate)}

	name := "/dup"
	time1 := parseTime(t, "2024-03-01T09:00:05Z")
	time2 := parseTime(t, "2024-03-01T09:00:35Z")
	time3 := parseTime(t, "2024-03-01T09:00:55Z")
	time4 := parseTime(t, "2024-03-01T09:00:59Z")

	aggregator.Record(albLogEntry{method: "POST", host: "api.example.com", status: 200, targetProcessingTime: 0.42, timestamp: time1}, name)
	aggregator.Record(albLogEntry{method: "POST", host: "api.example.com", status: 200, targetProcessingTime: 0.42, timestamp: time2}, name)
	aggregator.Record(albLogEntry{method: "POST", host: "api.example.com", status: 200, targetProcessingTime: 0.58, timestamp: time3}, name)
	aggregator.Record(albLogEntry{method: "POST", host: "api.example.com", status: 500, targetProcessingTime: 0.42, timestamp: time4}, name)

	metricData := aggregator.GetCloudWatchMetricData()

	var responseDatum *types.MetricDatum
	for i := range metricData {
		md := &metricData[i]
		if md.MetricName == nil || *md.MetricName != metricNameResponseTime {
			continue
		}
		responseDatum = md
		break
	}

	if responseDatum == nil {
		t.Fatalf("response time metric datum not found")
	}

	assert.Equal(t, []float64{0.42, 0.58}, responseDatum.Values)
	assert.Equal(t, []float64{3, 1}, responseDatum.Counts)
}
