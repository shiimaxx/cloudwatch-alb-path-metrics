package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/go-faker/faker/v4"
)

const (
	defaultRPS     = 10.0
	logTimeFormat  = time.RFC3339
	windowDuration = 5 * time.Minute
	windowSeconds  = 300
)

type albLogEntry struct {
	Timestamp   time.Time
	ClientAddr  string
	TargetAddr  string
	RequestLine string
}

func (e albLogEntry) String() string {
	return fmt.Sprintf("%s %s %s %q", e.Timestamp.Format(logTimeFormat), e.ClientAddr, e.TargetAddr, e.RequestLine)
}

func main() {
	seedFlag := flag.Int64("seed", time.Now().UnixNano(), "seed for synthetic data generation")
	countFlag := flag.Int("count", 0, "number of log entries to emit (default: derived from --rps)")
	rpsFlag := flag.Float64("rps", defaultRPS, "average requests per second over the five-minute window")
	startFlag := flag.String("start", "", "start time (RFC3339) for the five-minute window; defaults to now minus five minutes")
	flag.Parse()

	if *countFlag == 0 && *rpsFlag <= 0 {
		log.Fatalf("rps must be positive when count is not specified: %.2f", *rpsFlag)
	}
	if *countFlag < 0 {
		log.Fatalf("count must be non-negative: %d", *countFlag)
	}

	startTime := resolveStartTime(*startFlag)
	entryCount := resolveEntryCount(*countFlag, *rpsFlag)

	faker.SetRandomSource(faker.NewSafeSource(rand.NewSource(*seedFlag)))

	entries := generateEntries(entryCount, startTime)
	for _, entry := range entries {
		if _, err := fmt.Fprintln(os.Stdout, entry.String()); err != nil {
			log.Fatalf("failed to write log entry: %v", err)
		}
	}
}

func generateEntries(count int, start time.Time) []albLogEntry {
	entries := make([]albLogEntry, 0, count)
	windowNanos := windowDuration.Nanoseconds()
	var step int64
	if count > 1 {
		step = windowNanos / int64(count-1)
	}

	for i := range count {
		offset := time.Duration(step * int64(i))
		timestamp := start.Add(offset)
		template := newEntryTemplate()
		next := albLogEntry{
			Timestamp:   timestamp,
			ClientAddr:  fmt.Sprintf("%s:%d", template.ClientIP, template.ClientPort),
			TargetAddr:  fmt.Sprintf("%s:%d", template.TargetIP, template.TargetPort),
			RequestLine: buildRequestLine(template),
		}
		entries = append(entries, next)
	}

	return entries
}

func resolveStartTime(startFlag string) time.Time {
	if startFlag == "" {
		return time.Now().UTC().Add(-windowDuration)
	}

	startTime, err := time.Parse(time.RFC3339, startFlag)
	if err != nil {
		log.Fatalf("invalid start timestamp %q: %v", startFlag, err)
	}
	return startTime
}

func resolveEntryCount(count int, rps float64) int {
	if count > 0 {
		return count
	}

	derived := int(math.Round(rps * windowSeconds))
	if derived <= 0 {
		log.Fatalf("derived entry count must be positive (rps=%.2f)", rps)
	}
	return derived
}

func newEntryTemplate() entryTemplate {
	var template entryTemplate
	if err := faker.FakeData(&template); err != nil {
		log.Fatalf("faker failed to populate entry template: %v", err)
	}
	return template
}

func buildRequestLine(template entryTemplate) string {
	return fmt.Sprintf("%s %s HTTP/1.1", template.Method, template.Path)
}

type entryTemplate struct {
	ClientIP   string `faker:"ipv4"`
	TargetIP   string `faker:"ipv4"`
	ClientPort int    `faker:"boundary_start=1024, boundary_end=65535"`
	TargetPort int    `faker:"oneof:80,443,8080,9000"`
	Method     string `faker:"oneof:GET,POST,PUT,PATCH,DELETE"`
	Path       string `faker:"oneof:/,/health,/login,/logout,/api/orders,/api/users,/static/css/main.css"`
}
