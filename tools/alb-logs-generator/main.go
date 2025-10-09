package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/go-faker/faker/v4"
)

const (
	defaultEntryCount = 10
	logTimeFormat     = time.RFC3339
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
	countFlag := flag.Int("count", defaultEntryCount, "number of log entries to emit")
	flag.Parse()

	if *countFlag <= 0 {
		log.Fatalf("count must be positive: %d", *countFlag)
	}

	rng := rand.New(rand.NewSource(*seedFlag))
	faker.SetRandomSource(faker.NewSafeSource(rand.NewSource(*seedFlag)))

	entries := generateEntries(*countFlag, time.Now().UTC(), rng)
	for _, entry := range entries {
		if _, err := fmt.Fprintln(os.Stdout, entry.String()); err != nil {
			log.Fatalf("failed to write log entry: %v", err)
		}
	}
}

func generateEntries(count int, anchor time.Time, rng *rand.Rand) []albLogEntry {
	entries := make([]albLogEntry, 0, count)
	start := anchor.Add(-time.Duration(count) * 500 * time.Millisecond)

	for i := range count {
		timestamp := start.Add(time.Duration(i)*500*time.Millisecond + time.Duration(rng.Intn(250))*time.Millisecond)
		template := newEntryTemplate()
		entries = append(entries, albLogEntry{
			Timestamp:   timestamp,
			ClientAddr:  fmt.Sprintf("%s:%d", template.ClientIP, template.ClientPort),
			TargetAddr:  fmt.Sprintf("%s:%d", template.TargetIP, template.TargetPort),
			RequestLine: buildRequestLine(template),
		})
	}

	return entries
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
