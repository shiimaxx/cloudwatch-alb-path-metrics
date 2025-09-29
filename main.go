package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func handler(ctx context.Context, s3Event events.S3Event) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	namespace := strings.TrimSpace(os.Getenv("NAMESPACE"))
	if namespace == "" {
		return fmt.Errorf("NAMESPACE environment variable is required")
	}

	service := strings.TrimSpace(os.Getenv("SERVICE"))
	if service == "" {
		return fmt.Errorf("SERVICE environment variable is required")
	}

	rules, err := newPathRules(os.Getenv("INCLUDE_PATH_RULES"))
	if err != nil {
		return fmt.Errorf("parse path rules: %w", err)
	}

	aggregator := NewMetricAggregator(namespace, service)

	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		if bucket == "" {
			return fmt.Errorf("missing bucket name in S3 event record")
		}

		key, err := url.QueryUnescape(record.S3.Object.Key)
		if err != nil {
			return fmt.Errorf("decode object key %q: %w", record.S3.Object.Key, err)
		}

		if err := streamObjectLines(ctx, s3Client, bucket, key, rules, aggregator); err != nil {
			return fmt.Errorf("stream s3://%s/%s: %w", bucket, key, err)
		}
	}

	metricData := aggregator.GetCloudWatchMetricData()
	if len(metricData) == 0 {
		return nil
	}

	output := metricsEnvelope{
		Namespace:  namespace,
		MetricData: convertMetricData(metricData),
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("marshal metric output: %w", err)
	}

	fmt.Println(string(encoded))

	return nil
}

func streamObjectLines(ctx context.Context, client *s3.Client, bucket, key string, rules *pathRules, aggregator *MetricAggregator) error {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		line := scanner.Text()
		entry, route, ok := normalizeLogLine(line, rules)
		if !ok {
			continue
		}
		if aggregator != nil {
			aggregator.Record(*entry, route)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan gzip stream: %w", err)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}

type metricsEnvelope struct {
	Namespace  string              `json:"namespace"`
	MetricData []metricDatumOutput `json:"metric_data"`
}

type metricDatumOutput struct {
	MetricName string            `json:"metric_name"`
	Timestamp  time.Time         `json:"timestamp"`
	Dimensions []dimensionOutput `json:"dimensions"`
	Unit       string            `json:"unit"`
	Value      *float64          `json:"value,omitempty"`
	Values     []float64         `json:"values,omitempty"`
	Counts     []float64         `json:"counts,omitempty"`
}

type dimensionOutput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func convertMetricData(metricData []types.MetricDatum) []metricDatumOutput {
	outputs := make([]metricDatumOutput, 0, len(metricData))
	for _, datum := range metricData {
		outputs = append(outputs, metricDatumOutput{
			MetricName: derefString(datum.MetricName),
			Timestamp:  derefTime(datum.Timestamp),
			Dimensions: convertDimensions(datum.Dimensions),
			Unit:       string(datum.Unit),
			Value:      copyFloatPointer(datum.Value),
			Values:     append([]float64(nil), datum.Values...),
			Counts:     append([]float64(nil), datum.Counts...),
		})
	}
	return outputs
}

func convertDimensions(dimensions []types.Dimension) []dimensionOutput {
	if len(dimensions) == 0 {
		return nil
	}
	result := make([]dimensionOutput, len(dimensions))
	for i, dim := range dimensions {
		result[i] = dimensionOutput{Name: derefString(dim.Name), Value: derefString(dim.Value)}
	}
	return result
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func copyFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
