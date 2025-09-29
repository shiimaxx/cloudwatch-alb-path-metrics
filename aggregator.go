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
)

type metricKey struct {
	Method string
	Host   string
	Route  string
	Minute time.Time
}

type metricAggregate struct {
	durations    []float64
	successCount int
	failedCount  int
}

// MetricAggregator maintains per method/host/route aggregates convertible to CloudWatch MetricDatum values.
type MetricAggregator struct {
	metrics   map[metricKey]*metricAggregate
	namespace string
	service   string
}

// NewMetricAggregator creates a new MetricAggregator instance.
func NewMetricAggregator(namespace, service string) *MetricAggregator {
	return &MetricAggregator{
		metrics:   make(map[metricKey]*metricAggregate),
		namespace: namespace,
		service:   service,
	}
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

	agg.durations = append(agg.durations, entry.duration)
	if entry.status >= 500 && entry.status <= 599 {
		agg.failedCount++
	} else {
		agg.successCount++
	}
}

// GetCloudWatchMetricData materializes the aggregates as CloudWatch metric data points.
func (m *MetricAggregator) GetCloudWatchMetricData() []types.MetricDatum {
	var metricData []types.MetricDatum

	for key, agg := range m.metrics {
		totalRequests := agg.successCount + agg.failedCount
		if totalRequests == 0 {
			continue
		}

		timestamp := key.Minute

		dimensions := []types.Dimension{
			{Name: aws.String("Service"), Value: aws.String(m.service)},
			{Name: aws.String("Method"), Value: aws.String(key.Method)},
			{Name: aws.String("Host"), Value: aws.String(key.Host)},
			{Name: aws.String("Route"), Value: aws.String(key.Route)},
		}

		if len(agg.durations) > 0 {
			values := make([]float64, len(agg.durations))
			counts := make([]float64, len(agg.durations))
			copy(values, agg.durations)
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
			Value:      aws.Float64(float64(agg.successCount)),
			Unit:       types.StandardUnitCount,
		})

		metricData = append(metricData, types.MetricDatum{
			MetricName: aws.String(metricNameFailedRequestCount),
			Timestamp:  aws.Time(timestamp),
			Dimensions: cloneDimensions(dimensions),
			Value:      aws.Float64(float64(agg.failedCount)),
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
