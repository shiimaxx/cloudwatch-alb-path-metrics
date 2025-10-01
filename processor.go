package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type MetricsProcessor struct {
	s3Client   *s3.Client
	rules      *PathRules
	aggregator *MetricAggregator
	publisher  *CloudWatchMetricPublisher
}

func (p *MetricsProcessor) HandleEvent(ctx context.Context, s3Event events.S3Event) error {
	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		if bucket == "" {
			return fmt.Errorf("missing bucket name in S3 event record")
		}

		key, err := url.QueryUnescape(record.S3.Object.Key)
		if err != nil {
			return fmt.Errorf("decode object key %q: %w", record.S3.Object.Key, err)
		}

		if err := p.streamObjectLines(ctx, bucket, key); err != nil {
			return fmt.Errorf("stream s3://%s/%s: %w", bucket, key, err)
		}
	}

	metricData := p.aggregator.GetCloudWatchMetricData()
	if len(metricData) == 0 {
		return nil
	}

	if err := p.publisher.Publish(ctx, metricData); err != nil {
		return fmt.Errorf("publish metrics: %w", err)
	}

	p.logMetrics(metricData)

	return nil
}

func (p *MetricsProcessor) streamObjectLines(ctx context.Context, bucket, key string) error {
	resp, err := p.s3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
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
		entry, route, matched := p.normalizeLogLine(line)
		if !matched {
			continue
		}
		p.aggregator.Record(*entry, route)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan gzip stream: %w", err)
	}

	return nil
}

func (p *MetricsProcessor) logMetrics(metricData []types.MetricDatum) {
	for _, data := range metricData {
		if data.MetricName == nil {
			continue
		}

		switch aws.ToString(data.MetricName) {
		case metricNameResponseTime:
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Values: %v, Counts: %v\n",
				aws.ToString(data.MetricName),
				data.Dimensions,
				data.Timestamp,
				data.Values,
				data.Counts,
			)
		case metricNameRequestCount, metricNameFailedRequestCount:
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Value: %v\n",
				aws.ToString(data.MetricName),
				data.Dimensions,
				data.Timestamp,
				aws.ToFloat64(data.Value),
			)
		}
	}
}

// normalizeLogLine returns the parsed entry and normalized route when the log line matches a rule.
func (p *MetricsProcessor) normalizeLogLine(line string) (*albLogEntry, string, bool) {
	if p.rules == nil || !p.rules.enabled {
		return nil, "", false
	}

	entry, err := parseALBLogLine(line)
	if err != nil {
		return nil, "", false
	}

	route, matched := p.rules.normalize(*entry)
	if !matched {
		return nil, "", false
	}

	return entry, route, true
}
