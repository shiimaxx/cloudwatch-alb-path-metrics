package main

import (
	"fmt"
	"math"
	"time"
)

const (
	defaultRPS     = 10.0
	windowDuration = 5 * time.Minute
)

// RelevantInput bundles the context required to generate relevant fields.
type RelevantInput struct {
	Timestamp        time.Time
	IrrelevantFields IrrelevantFields
}

// Generator orchestrates irrelevant and relevant field generators.
type Generator struct {
	Irrelevant *IrrelevantGenerator
	Relevant   *RelevantGenerator
}

// GenerateEntries returns skeleton log entries.
func (g *Generator) GenerateEntries(count int, start time.Time) ([]ALBLogEntry, error) {
	return nil, nil
}

// ResolveEntryCount resolves the number of entries to generate.
func ResolveEntryCount(count int, rps float64) (int, error) {
	if count < 0 {
		return 0, fmt.Errorf("count must be non-negative: %d", count)
	}

	if count > 0 {
		return count, nil
	}

	if rps <= 0 {
		return 0, fmt.Errorf("rps must be positive when count is zero: %.2f", rps)
	}

	calculated := int(math.Round(rps * windowDuration.Seconds()))
	if calculated <= 0 {
		calculated = 1
	}

	return calculated, nil
}
