package metrics

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Processor struct {
	s3Client   *s3.Client
	rules      *pathRules
	aggregator *metricAggregator
	publisher  *cloudWatchMetricPublisher
	debug      bool
}

func NewProcwessor(s3Client *s3.Client, cwClient *cloudwatch.Client, rules *pathRules, dryRun, debug bool) *Processor {
	return &Processor{
		s3Client:   s3Client,
		rules:      rules,
		aggregator: &metricAggregator{metrics: make(map[metricKey]*metricAggregate)},
		publisher: &cloudWatchMetricPublisher{
			client:       cwClient, 
			namespace:    "ALBAccessLog",
			maxBatchSize: defaultMetricBatchSize,
			dryRun:       dryRun,
		},
		debug: debug,
	}
}

func (p *Processor) HandleEvent(ctx context.Context, s3Event events.S3Event) error {
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

	if p.debug {
		p.logMetrics(metricData)
	}

	return nil
}

func (p *Processor) streamObjectLines(ctx context.Context, bucket, key string) error {
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

func (p *Processor) logMetrics(metricData []types.MetricDatum) {
	expandDimensions := func(dimensions []types.Dimension) (r []string) {
		for _, d := range dimensions {
			r = append(r, fmt.Sprintf("%s=%s", aws.ToString(d.Name), aws.ToString(d.Value)))
		}
		return r
	}

	for _, data := range metricData {
		switch aws.ToString(data.MetricName) {
		case metricNameTargetResponseTime:
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Values: %v, Counts: %v\n",
				aws.ToString(data.MetricName),
				expandDimensions(data.Dimensions),
				data.Timestamp,
				data.Values,
				data.Counts,
			)
		case metricNameRequestCount, metricNameFailedRequestCount:
			fmt.Printf("Metric: %s, Dimensions: %v, Timestamp: %v, Value: %v\n",
				aws.ToString(data.MetricName),
				expandDimensions(data.Dimensions),
				data.Timestamp,
				aws.ToFloat64(data.Value),
			)
		}
	}
}

// normalizeLogLine returns the parsed entry and normalized route when the log line matches a rule.
func (p *Processor) normalizeLogLine(line string) (*albLogEntry, string, bool) {
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
