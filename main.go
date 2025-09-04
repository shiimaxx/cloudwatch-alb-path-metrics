package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client
var cwClient *cloudwatch.Client

var regexMatcher map[string]*pathPattern

type pathPattern struct {
	Method  string `json:"method"`
	Pattern string `json:"pattern"`
	Name    string `json:"name"`
	Re      *regexp.Regexp
}

func normalizePath(method, path string) (string, bool) {
	for name, p := range regexMatcher {
		if p.Method == method && p.Re.MatchString(path) {
			return name, true
		}
	}
	return "", false
}

func publishMetrics(ctx context.Context, t time.Time, path, serviceName string, records [][]string) error {
	var requestCount float64
	var successfulRequestCount float64
	latencies := make(map[float64]float64)
	var metricData []types.MetricDatum

	for _, record := range records {
		requestProcessingTime, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			fmt.Println("failed to parse request processing time:", err)
			continue
		}
		targetProcessingTime, err := strconv.ParseFloat(record[6], 64)
		if err != nil {
			fmt.Println("failed to parse target processing time:", err)
			continue
		}
		responseProcessingTime, err := strconv.ParseFloat(record[7], 64)
		if err != nil {
			fmt.Println("failed to parse response processing time:", err)
			continue
		}
		elbStatusCode, err := strconv.Atoi(record[8])
		if err != nil {
			fmt.Println("failed to parse elb status code:", err)
			continue
		}
		// targetStatusCode, err := strconv.Atoi(sp[9])
		// if err != nil {
		// 	fmt.Println("failed to parse target status code:", err)
		// 	continue
		// }

		requestCount++
		if elbStatusCode >= 200 && elbStatusCode <= 499 {
			successfulRequestCount++
		}
		latency := requestProcessingTime + targetProcessingTime + responseProcessingTime
		if _, ok := latencies[latency]; !ok {
			latencies[latency] = 0
		}
		latencies[latency]++
	}

	metricData = append(metricData, types.MetricDatum{
		MetricName: aws.String("RequestCount"),
		Timestamp:  aws.Time(t),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("Path"),
				Value: aws.String(path),
			},
			{
				Name:  aws.String("Service"),
				Value: aws.String(serviceName),
			},
		},
		Value: aws.Float64(requestCount),
		Unit:  types.StandardUnitCount,
	})

	metricData = append(metricData, types.MetricDatum{
		MetricName: aws.String("SuccessfulRequestCount"),
		Timestamp:  aws.Time(t),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("Path"),
				Value: aws.String(path),
			},
			{
				Name:  aws.String("Service"),
				Value: aws.String(serviceName),
			},
		},
		Value: aws.Float64(successfulRequestCount),
		Unit:  types.StandardUnitCount,
	})

	latencyValues := make([]float64, 0, len(latencies))
	latencyCounts := make([]float64, 0, len(latencies))
	for latency, count := range latencies {
		latencyValues = append(latencyValues, latency)
		latencyCounts = append(latencyCounts, count)
	}
	metricData = append(metricData, types.MetricDatum{
		MetricName: aws.String("Latency"),
		Timestamp:  aws.Time(t),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("Path"),
				Value: aws.String(path),
			},
			{
				Name:  aws.String("Service"),
				Value: aws.String(serviceName),
			},
		},
		Values: latencyValues,
		Counts: latencyCounts,
		Unit:   types.StandardUnitSeconds,
	})

	_, err := cwClient.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String("Shiimaxx"),
		MetricData: metricData,
	})
	if err != nil {
		return fmt.Errorf("failed to put metric data: %w", err)
	}

	return nil
}

func processLogEntry(ctx context.Context, reader *csv.Reader, serviceName string) error {
	metrics := make(map[string]map[string][][]string)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to read CSV: %w", err)
		}

		t, err := time.Parse(time.RFC3339Nano, record[1])
		if err != nil {
			fmt.Println("failed to parse time:", err)
			continue
		}
		t = t.Truncate(time.Minute)
		tAsKey := t.Format(time.RFC3339Nano)

		request := record[12]
		sp := strings.Split(request, " ")
		method := sp[0]
		u, err := url.Parse(sp[1])
		if err != nil {
			fmt.Println("failed to parse URL:", err)
			continue
		}

		normalizedPath, allowed := normalizePath(method, u.Path)
		if !allowed {
			continue
		}

		if _, ok := metrics[normalizedPath]; !ok {
			metrics[normalizedPath] = make(map[string][][]string)
		}

		metrics[normalizedPath][tAsKey] = append(metrics[normalizedPath][tAsKey], record)
	}

	for path, timeMap := range metrics {
		for t, records := range timeMap {
			parsedTime, err := time.Parse(time.RFC3339Nano, t)
			if err != nil {
				fmt.Println("failed to parse time:", err)
				continue
			}
			publishMetrics(ctx, parsedTime, path, serviceName, records)
		}
	}

	return nil
}

func processS3Object(ctx context.Context, client *s3.Client, bucket, key, serviceName string) error {
	fmt.Printf("Processing object %s from bucket %s\n", key, bucket)

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer out.Body.Close()

	zr, err := gzip.NewReader(out.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer zr.Close()

	reader := bufio.NewReader(zr)
	cr := csv.NewReader(reader)
	cr.Comma = ' '
	cr.ReuseRecord = true

	if err := processLogEntry(ctx, cr, serviceName); err != nil {
		return fmt.Errorf("failed to process log entry: %w", err)
	}

	return nil
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config, %v", err)
	}
	s3Client = s3.NewFromConfig(cfg)
	cwClient = cloudwatch.NewFromConfig(cfg)

	serviceName := os.Getenv("SERVICE")
	if serviceName == "" {
		return fmt.Errorf("SERVICE environment variable is required")
	}

	// [
	//   {"method":"GET", "pattern":"/api/v1/users/\d","name":"GetUser"},
	//   {"method":"POST", "pattern":"/api/v1/users","name":"CreateUser"}
	// ]
	var pp []pathPattern
	if pathPatterns := os.Getenv("PATH_PATTERNS"); pathPatterns != "" {
		if err := json.Unmarshal([]byte(pathPatterns), &pp); err != nil {
			return fmt.Errorf("failed to unmarshal path patterns: %w", err)
		}

		regexMatcher = make(map[string]*pathPattern)
		for _, p := range pp {
			p := p
			regex, err := regexp.Compile(p.Pattern)
			if err != nil {
				return fmt.Errorf("failed to compile regex pattern %s: %w", p.Pattern, err)
			}
			p.Re = regex
			regexMatcher[p.Name] = &p
		}
	}

	for _, record := range s3Event.Records {
		if err := processS3Object(ctx, s3Client, record.S3.Bucket.Name, record.S3.Object.Key, serviceName); err != nil {
			fmt.Println("error processing object:", err)
		}
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
