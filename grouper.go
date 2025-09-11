package main

import (
	"errors"
	"regexp"
	"strings"
)

// grouper represents a compiled path grouping pattern matcher
type grouper struct {
	patterns []compiledPattern
	enabled  bool
}

// compiledPattern holds a compiled regex pattern and its original string
type compiledPattern struct {
	regex   *regexp.Regexp
	pattern string
}

// newGrouper creates a new grouper instance with compiled regex patterns
func newGrouper(regexes string) (*grouper, error) {
	if regexes == "" {
		return &grouper{enabled: false}, nil
	}

	patterns := strings.Split(regexes, ",")
	compiledPatterns := make([]compiledPattern, 0, len(patterns))

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, errors.New("failed to compile regex pattern '" + pattern + "': " + err.Error())
		}

		compiledPatterns = append(compiledPatterns, compiledPattern{
			regex:   regex,
			pattern: pattern,
		})
	}

	return &grouper{
		patterns: compiledPatterns,
		enabled:  len(compiledPatterns) > 0,
	}, nil
}

// groupPath groups the given path using the compiled regex patterns
// Returns the first matching regex pattern string, or the original path if no match
func (g *grouper) groupPath(path string) string {
	if !g.enabled {
		return path
	}

	for _, cp := range g.patterns {
		if cp.regex.MatchString(path) {
			return cp.pattern
		}
	}

	return path
}
