package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	metricNameDuration = "Duration"
	delimiter          = "\t" // Tab character - safe delimiter that won't appear in URLs
)

// MetricKey represents the key for aggregating metrics
type MetricKey struct {
	Method string
	Host   string
	Path   string
}

// String returns a string representation of the MetricKey for use as a map key
func (k MetricKey) String() string {
	return fmt.Sprintf("%s%s%s%s%s", k.Method, delimiter, k.Host, delimiter, k.Path)
}

// MetricValue represents a single metric data point with timestamp
type MetricValue struct {
	Duration  float64
	Timestamp time.Time
}

// MetricAggregator aggregates duration metrics by Method, Host, and Path
type MetricAggregator struct {
	metrics   map[string][]MetricValue
	namespace string
	service   string
}

// NewMetricAggregator creates a new MetricAggregator instance
func NewMetricAggregator(namespace, service string) *MetricAggregator {
	return &MetricAggregator{
		metrics:   make(map[string][]MetricValue),
		namespace: namespace,
		service:   service,
	}
}

// AddDuration adds a duration value with timestamp for the given metric key
func (m *MetricAggregator) AddDuration(key MetricKey, duration float64, timestamp time.Time) {
	keyStr := key.String()
	m.metrics[keyStr] = append(m.metrics[keyStr], MetricValue{
		Duration:  duration,
		Timestamp: timestamp,
	})
}

// GetCloudWatchMetricData returns CloudWatch MetricDatum for all aggregated metrics
func (m *MetricAggregator) GetCloudWatchMetricData() []types.MetricDatum {
	var metricData []types.MetricDatum

	for keyStr, metricValues := range m.metrics {
		// Parse the key back to get Method, Host, Path
		key, err := parseMetricKey(keyStr)
		if err != nil {
			// Skip invalid keys
			continue
		}

		// Create dimensions
		dimensions := []types.Dimension{
			{
				Name:  aws.String("Service"),
				Value: aws.String(m.service),
			},
			{
				Name:  aws.String("Method"),
				Value: aws.String(key.Method),
			},
			{
				Name:  aws.String("Host"),
				Value: aws.String(key.Host),
			},
			{
				Name:  aws.String("Path"),
				Value: aws.String(key.Path),
			},
		}

		// Create MetricDatum with Values and Counts for efficient API usage
		// Note: CloudWatch MetricDatum only supports single timestamp per datum,
		// so we use Values/Counts arrays and let CloudWatch handle timestamp
		unit := types.StandardUnitSeconds

		// Convert metric values to CloudWatch format
		values := make([]float64, len(metricValues))
		counts := make([]float64, len(metricValues))
		for i, mv := range metricValues {
			values[i] = mv.Duration
			counts[i] = 1.0 // Each duration represents 1 request
		}

		metricDatum := types.MetricDatum{
			MetricName: aws.String(metricNameDuration),
			Dimensions: dimensions,
			Values:     values,
			Counts:     counts,
			Unit:       unit,
		}

		metricData = append(metricData, metricDatum)
	}

	return metricData
}

// parseMetricKey parses a string key back into MetricKey
func parseMetricKey(keyStr string) (MetricKey, error) {
	parts := strings.SplitN(keyStr, delimiter, 3)
	if len(parts) != 3 {
		return MetricKey{}, fmt.Errorf("invalid key format: %s", keyStr)
	}

	return MetricKey{
		Method: parts[0],
		Host:   parts[1],
		Path:   parts[2],
	}, nil
}
