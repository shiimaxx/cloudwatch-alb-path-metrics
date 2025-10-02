package main

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	metricNameResponseTime       = "ResponseTime"
	metricNameRequestCount       = "RequestCount"
	metricNameFailedRequestCount = "FailedRequestCount"

	metricDimensionMethod = "Method"
	metricDimensionHost   = "Host"
	metricDimensionRoute  = "Route"
)

type metricKey struct {
	Method string
	Host   string
	Route  string
	Minute time.Time
}

type metricAggregate struct {
	responseTime       []float64
	requestCount       int
	failedRequestCount int
}

// MetricAggregator maintains per method/host/route aggregates convertible to CloudWatch MetricDatum values.
type MetricAggregator struct {
	metrics map[metricKey]*metricAggregate
}

// Record adds a single request observation to the aggregate identified by the normalized route.
func (m *MetricAggregator) Record(entry albLogEntry, route string) {
	if route == "" {
		return
	}

	minute := entry.timestamp.UTC().Truncate(time.Minute)
	key := metricKey{Method: entry.method, Host: entry.host, Route: route, Minute: minute}
	agg, ok := m.metrics[key]
	if !ok {
		agg = &metricAggregate{}
		m.metrics[key] = agg
	}

	agg.responseTime = append(agg.responseTime, entry.duration)
	agg.requestCount++
	if entry.status >= 500 && entry.status <= 599 {
		agg.failedRequestCount++
	}
}

// GetCloudWatchMetricData materializes the aggregates as CloudWatch metric data points.
func (m *MetricAggregator) GetCloudWatchMetricData() []types.MetricDatum {
	var metricData []types.MetricDatum

	for key, agg := range m.metrics {
		timestamp := key.Minute

		dimensions := []types.Dimension{
			{Name: aws.String(metricDimensionMethod), Value: aws.String(key.Method)},
			{Name: aws.String(metricDimensionHost), Value: aws.String(key.Host)},
			{Name: aws.String(metricDimensionRoute), Value: aws.String(key.Route)},
		}

		if len(agg.responseTime) > 0 {
			values := make([]float64, len(agg.responseTime))
			counts := make([]float64, len(agg.responseTime))
			copy(values, agg.responseTime)
			for i := range counts {
				counts[i] = 1.0
			}

			metricData = append(metricData, types.MetricDatum{
				MetricName: aws.String(metricNameResponseTime),
				Timestamp:  aws.Time(timestamp),
				Dimensions: cloneDimensions(dimensions),
				Values:     values,
				Counts:     counts,
				Unit:       types.StandardUnitSeconds,
			})
		}

		metricData = append(metricData, types.MetricDatum{
			MetricName: aws.String(metricNameRequestCount),
			Timestamp:  aws.Time(timestamp),
			Dimensions: cloneDimensions(dimensions),
			Value:      aws.Float64(float64(agg.requestCount)),
			Unit:       types.StandardUnitCount,
		})

		metricData = append(metricData, types.MetricDatum{
			MetricName: aws.String(metricNameFailedRequestCount),
			Timestamp:  aws.Time(timestamp),
			Dimensions: cloneDimensions(dimensions),
			Value:      aws.Float64(float64(agg.failedRequestCount)),
			Unit:       types.StandardUnitCount,
		})
	}

	return metricData
}

func cloneDimensions(source []types.Dimension) []types.Dimension {
	clone := make([]types.Dimension, len(source))
	copy(clone, source)
	return clone
}
