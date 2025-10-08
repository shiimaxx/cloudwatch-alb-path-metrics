package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	defaultSeed := time.Now().UnixNano()
	outputPath := flag.String("output", "", "path to write logs (defaults to stdout)")
	seed := flag.Int64("seed", defaultSeed, "seed for random log generation")
	flag.Parse()

	rng := rand.New(rand.NewSource(*seed))

	var (
		out     io.Writer = os.Stdout
		closeFn func() error
		isFile  bool
	)

	if *outputPath != "" && *outputPath != "-" {
		file, err := os.Create(*outputPath)
		if err != nil {
			log.Fatalf("failed to create output file: %v", err)
		}
		out = file
		isFile = true
		closeFn = file.Close
	}

	if _, err := fmt.Fprintf(out, "ALB placeholder log line (seed=%d, sample=%d)\n", *seed, rng.Int63()); err != nil {
		log.Fatalf("failed to write log line: %v", err)
	}

	if isFile && closeFn != nil {
		if err := closeFn(); err != nil {
			log.Printf("warning: failed to close output file: %v", err)
		}
	}
}
