package main

import (
	"flag"
	"log"
	"time"
)

type cliConfig struct {
	rps   float64
	count int
}

func main() {
	cfg := parseFlags()

	count, err := ResolveEntryCount(cfg.count, cfg.rps)
	if err != nil {
		log.Fatalf("resolve entry count: %v", err)
	}

	gen := Generator{
		Irrelevant: NewIrrelevantGenerator(),
		Relevant:   NewRelevantGenerator(),
	}

	if _, err := gen.GenerateEntries(count, time.Now().UTC()); err != nil {
		log.Fatalf("generate entries: %v", err)
	}
}

func parseFlags() cliConfig {
	rps := flag.Float64("rps", defaultRPS, "requests per second")
	count := flag.Int("count", 0, "number of entries to generate")
	flag.Parse()
	return cliConfig{
		rps:   *rps,
		count: *count,
	}
}
