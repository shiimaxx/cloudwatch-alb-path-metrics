package main

import (
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	defaultRPS     = 10.0
	logTimeFormat  = "2006-01-02T15:04:05.000000Z"
	windowDuration = 5 * time.Minute
	windowSeconds  = 300
)

func generateEntries(count int, start time.Time, rng *rand.Rand) []albLogEntry {
	entries := make([]albLogEntry, 0, count)
	windowNanos := windowDuration.Nanoseconds()
	var step int64
	if count > 1 {
		step = windowNanos / int64(count-1)
	}

	for i := range count {
		offset := time.Duration(step * int64(i))
		timestamp := start.Add(offset)
		template := newEntryTemplate(rng)
		entries = append(entries, buildLogEntry(template, timestamp))
	}

	return entries
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
