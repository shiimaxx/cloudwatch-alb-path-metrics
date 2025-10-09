package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/go-faker/faker/v4"
)

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

	fakerSource := rand.NewSource(*seedFlag)
	faker.SetRandomSource(faker.NewSafeSource(fakerSource))
	dataRand := rand.New(rand.NewSource(*seedFlag))

	entries := generateEntries(entryCount, startTime, dataRand)
	writer := bufio.NewWriter(os.Stdout)
	for i, entry := range entries {
		if _, err := writer.WriteString(entry.String()); err != nil {
			log.Fatalf("failed to write log entry: %v", err)
		}

		if i < len(entries)-1 {
			if err := writer.WriteByte('\n'); err != nil {
				log.Fatalf("failed to write newline: %v", err)
			}
		}
	}
	if err := writer.Flush(); err != nil {
		log.Fatalf("failed to flush output: %v", err)
	}
}
