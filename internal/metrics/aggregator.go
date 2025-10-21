package metrics

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	metricNameTargetResponseTime = "TargetResponseTime"
	metricNameRequestCount       = "RequestCount"
	metricNameFailedRequestCount = "FailedRequestCount"

	metricDimensionMethod = "Method"
	metricDimensionHost   = "Host"
	metricDimensionPath   = "Path"

	maxMetricValues = 150
)

type metricKey struct {
	Method string
	Host   string
	Path   string
	Minute time.Time
}

type metricAggregate struct {
	targetResponseTime []float64
	requestCount       int
	failedRequestCount int
}

// metricAggregator maintains per method/host/path aggregates convertible to CloudWatch MetricDatum values.
type metricAggregator struct {
	metrics map[metricKey]*metricAggregate
}

// Record adds a single request observation to the aggregate identified by the rule name.
func (m *metricAggregator) Record(entry albLogEntry, name string) {
	if name == "" {
		return
	}

	minute := entry.timestamp.UTC().Truncate(time.Minute)
	key := metricKey{Method: entry.method, Host: entry.host, Path: name, Minute: minute}
	agg, ok := m.metrics[key]
	if !ok {
		agg = &metricAggregate{}
		m.metrics[key] = agg
	}

	// Ignore negative target processing times, which indicate no target was involved.
	if entry.targetProcessingTime >= 0 {
		agg.targetResponseTime = append(agg.targetResponseTime, entry.targetProcessingTime)
	}

	agg.requestCount++
	if entry.status >= 500 && entry.status <= 599 {
		agg.failedRequestCount++
	}
}

// GetCloudWatchMetricData materializes the aggregates as CloudWatch metric data points.
func (m *metricAggregator) GetCloudWatchMetricData() []types.MetricDatum {
	var metricData []types.MetricDatum

	for key, agg := range m.metrics {
		timestamp := key.Minute

		dimensions := []types.Dimension{
			{Name: aws.String(metricDimensionMethod), Value: aws.String(key.Method)},
			{Name: aws.String(metricDimensionHost), Value: aws.String(key.Host)},
			{Name: aws.String(metricDimensionPath), Value: aws.String(key.Path)},
		}

		valueIndex := make(map[float64]int, len(agg.targetResponseTime))
		var values []float64
		var counts []float64
		for _, v := range agg.targetResponseTime {
			if idx, ok := valueIndex[v]; ok {
				counts[idx]++
				continue
			}

			valueIndex[v] = len(values)
			values = append(values, v)
			counts = append(counts, 1.0)
		}

		for start := 0; start < len(values); start += maxMetricValues {
			end := min(start+maxMetricValues, len(values))
			metricData = append(metricData, types.MetricDatum{
				MetricName: aws.String(metricNameTargetResponseTime),
				Timestamp:  aws.Time(timestamp),
				Dimensions: dimensions,
				Values:     values[start:end],
				Counts:     counts[start:end],
				Unit:       types.StandardUnitSeconds,
			})
		}

		metricData = append(metricData, types.MetricDatum{
			MetricName: aws.String(metricNameRequestCount),
			Timestamp:  aws.Time(timestamp),
			Dimensions: dimensions,
			Value:      aws.Float64(float64(agg.requestCount)),
			Unit:       types.StandardUnitCount,
		})

		metricData = append(metricData, types.MetricDatum{
			MetricName: aws.String(metricNameFailedRequestCount),
			Timestamp:  aws.Time(timestamp),
			Dimensions: dimensions,
			Value:      aws.Float64(float64(agg.failedRequestCount)),
			Unit:       types.StandardUnitCount,
		})
	}

	return metricData
}

// MetricAggregator exposes the internal aggregator type for tests.
type MetricAggregator = metricAggregator

// metricNameResponseTime retains compatibility with existing tests that reference the older constant name.
const metricNameResponseTime = metricNameTargetResponseTime
