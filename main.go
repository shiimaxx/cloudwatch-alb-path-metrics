package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/csv"
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
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var s3Client *s3.Client
var cwClient *cloudwatch.Client

var filterProgram *vm.Program
var groupPatterns []*regexp.Regexp

func isRequestAllowed(method, path string) bool {
	if filterProgram == nil {
		return true
	}

	env := map[string]any{
		"method": method,
		"path":   path,
	}

	result, err := expr.Run(filterProgram, env)
	if err != nil {
		fmt.Printf("failed to evaluate filter expression: %v\n", err)
		return false
	}

	return result.(bool)
}

func getPathGroup(path string) (string, bool) {
	for _, pattern := range groupPatterns {
		if pattern.MatchString(path) {
			return pattern.String(), true
		}
	}
	return "", false
}

func normalizePath(method, path string) (string, bool) {
	fmt.Println("Evaluating path:", path, "with method:", method)

	if !isRequestAllowed(method, path) {
		fmt.Println("Request filtered out:", method, path)
		return "", false
	}

	groupName, found := getPathGroup(path)
	if !found {
		fmt.Println("No matching group pattern for path:", path)
		return "", false
	}

	fmt.Println("Matched group pattern:", groupName, "for path:", path)
	return groupName, true
}

func publishMetrics(ctx context.Context, t time.Time, group, serviceName string, records [][]string) error {
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
				Name:  aws.String("Group"),
				Value: aws.String(group),
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
				Name:  aws.String("Group"),
				Value: aws.String(group),
			},
			{
				Name:  aws.String("Group"),
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
				Name:  aws.String("Group"),
				Value: aws.String(group),
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

	if envFilter := os.Getenv("FILTER"); envFilter != "" {
		program, err := expr.Compile(envFilter, expr.AsBool())
		if err != nil {
			return fmt.Errorf("failed to compile filter expression: %w", err)
		}
		filterProgram = program
	}

	if envGroups := os.Getenv("GROUPS"); envGroups != "" {
		patterns := strings.Split(envGroups, ",")
		groupPatterns = make([]*regexp.Regexp, 0, len(patterns))

		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("failed to compile regex pattern %s: %w", pattern, err)
			}

			groupPatterns = append(groupPatterns, re)
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
