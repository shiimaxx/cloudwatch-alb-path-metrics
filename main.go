package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

	"github.com/julienschmidt/httprouter"
)

var s3Client *s3.Client
var cwClient *cloudwatch.Client

type PathPattern struct {
	Pattern    string
	MetricName string
	Router     *httprouter.Router
}

var pathPatterns []PathPattern

func loadPathPatterns() {
	pathPatternsEnv := os.Getenv("PATH_PATTERNS")
	if pathPatternsEnv == "" {
		return
	}

	pairs := strings.Split(pathPatternsEnv, ",")
	pathPatterns = make([]PathPattern, 0, len(pairs))

	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			fmt.Printf("Invalid PATH_PATTERNS format: %s\n", pair)
			continue
		}

		pattern := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])

		// 各パターン専用のルーターを作成
		router := httprouter.New()
		router.Handle("GET", pattern, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {})

		pathPatterns = append(pathPatterns, PathPattern{
			Pattern:    pattern,
			MetricName: name,
			Router:     router,
		})
	}
}

func normalizePath(path string) (string, bool) {
	if len(pathPatterns) == 0 {
		return path, true
	}

	for _, pp := range pathPatterns {
		handle, _, _ := pp.Router.Lookup("GET", path)
		if handle != nil {
			return pp.MetricName, true
		}
	}

	return "", false
}

func publishMetrics(ctx context.Context, t time.Time, path string, entries []string) error {
	var requestCount float64
	var successfulRequestCount float64
	latencies := make(map[float64]float64)
	var metricData []types.MetricDatum

	for _, entry := range entries {
		sp := strings.Split(entry, " ")
		requestProcessingTime, err := strconv.ParseFloat(sp[5], 64)
		if err != nil {
			fmt.Println("failed to parse request processing time:", err)
			continue
		}
		targetProcessingTime, err := strconv.ParseFloat(sp[6], 64)
		if err != nil {
			fmt.Println("failed to parse target processing time:", err)
			continue
		}
		responseProcessingTime, err := strconv.ParseFloat(sp[7], 64)
		if err != nil {
			fmt.Println("failed to parse response processing time:", err)
			continue
		}
		elbStatusCode, err := strconv.Atoi(sp[8])
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
		},
		Value: aws.Float64(requestCount),
		Unit:  types.StandardUnitCount,
	})

	metricData = append(metricData, types.MetricDatum{
		MetricName: aws.String("SuccessfullRequestCount"),
		Timestamp:  aws.Time(t),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("Path"),
				Value: aws.String(path),
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

func processLogEntry(ctx context.Context, entries []string) error {
	metrics := make(map[string]map[string][]string)

	for _, entry := range entries {
		if entry == "" {
			continue
		}

		sp := strings.Split(entry, " ")
		t, err := time.Parse(time.RFC3339Nano, sp[1])
		if err != nil {
			fmt.Println("failed to parse time:", err)
			continue
		}
		t = t.Truncate(time.Minute)
		tAsKey := t.Format(time.RFC3339Nano)

		request := sp[13]
		u, err := url.Parse(request)
		if err != nil {
			fmt.Println("failed to parse URL:", err)
			continue
		}

		normalizedPath, allowed := normalizePath(u.Path)
		if !allowed {
			continue
		}

		if _, ok := metrics[normalizedPath]; !ok {
			metrics[normalizedPath] = make(map[string][]string)
		}

		metrics[normalizedPath][tAsKey] = append(metrics[normalizedPath][tAsKey], entry)
	}

	for path, timeMap := range metrics {
		for t, entries := range timeMap {
			parsedTime, err := time.Parse(time.RFC3339Nano, t)
			if err != nil {
				fmt.Println("failed to parse time:", err)
				continue
			}
			publishMetrics(ctx, parsedTime, path, entries)
		}
	}

	return nil
}

func processS3Object(ctx context.Context, client *s3.Client, bucket, key string) error {
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

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, zr); err != nil {
		return fmt.Errorf("failed to read gzip content: %w", err)
	}

	processLogEntry(ctx, strings.Split(buf.String(), "\n"))

	return nil
}

func handler(ctx context.Context, s3Event events.S3Event) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}
	s3Client = s3.NewFromConfig(cfg)
	cwClient = cloudwatch.NewFromConfig(cfg)

	loadPathPatterns()

	for _, record := range s3Event.Records {
		if err := processS3Object(ctx, s3Client, record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
			fmt.Println("error processing object:", err)
		}
	}
	return "Hello, World!", nil
}

func main() {
	lambda.Start(handler)
}
