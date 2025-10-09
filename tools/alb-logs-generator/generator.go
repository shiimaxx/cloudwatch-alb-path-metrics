package main

import "time"

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
	irrelevant *IrrelevantGenerator
	relevant   *RelevantGenerator
}

// GenerateEntries returns skeleton log entries.
func (g *Generator) GenerateEntries(count int, start time.Time) ([]ALBLogEntry, error) {
	return nil, nil
}

// ResolveEntryCount resolves the number of entries to generate.
func ResolveEntryCount(count int, rps float64) (int, error) {
	return count, nil
}
