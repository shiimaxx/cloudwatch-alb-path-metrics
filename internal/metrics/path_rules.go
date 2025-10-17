package metrics

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// pathRuleConfig represents the JSON shape used to configure path normalization rules.
type pathRuleConfig struct {
	Host    string `json:"host"`
	Pattern string `json:"pattern"`
	Name    string `json:"name"`
	Method  string `json:"method,omitempty"`
}

// pathRules holds the compiled rule set for host-aware path normalization.
type pathRules struct {
	enabled bool
	rules   []compiledRule
}

// compiledRule represents a single host/path matching rule compiled for runtime use.
type compiledRule struct {
	host   string
	method string
	name   string
	regex  *regexp.Regexp
}

// NewPathRules parses the JSON configuration string and returns a compiled rule set.
func NewPathRules(raw string) (*pathRules, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return &pathRules{enabled: false}, nil
	}

	var configs []pathRuleConfig
	if err := json.Unmarshal([]byte(trimmed), &configs); err != nil {
		return nil, fmt.Errorf("failed to parse path rules JSON: %w", err)
	}

	if len(configs) == 0 {
		return &pathRules{enabled: false}, nil
	}

	compiled := make([]compiledRule, 0, len(configs))

	for idx, cfg := range configs {
		if cfg.Host == "" {
			return nil, fmt.Errorf("path rule %d: host is required", idx)
		}

		if cfg.Pattern == "" {
			return nil, fmt.Errorf("path rule %d: pattern is required", idx)
		}

		if cfg.Name == "" {
			return nil, fmt.Errorf("path rule %d: name is required", idx)
		}

		method := strings.ToUpper(cfg.Method)

		regex, err := regexp.Compile(cfg.Pattern)
		if err != nil {
			return nil, fmt.Errorf("path rule %d: failed to compile pattern regex: %w", idx, err)
		}

		compiled = append(compiled, compiledRule{
			host:   cfg.Host,
			method: method,
			name:   cfg.Name,
			regex:  regex,
		})
	}

	return &pathRules{
		enabled: true,
		rules:   compiled,
	}, nil
}

// normalize returns the configured name for the provided entry if any rule matches.
func (pr *pathRules) normalize(entry albLogEntry) (string, bool) {
	if pr == nil || !pr.enabled {
		return "", false
	}

	for _, rule := range pr.rules {
		if entry.host != rule.host {
			continue
		}

		if rule.method != "" && !strings.EqualFold(entry.method, rule.method) {
			continue
		}

		if rule.regex.MatchString(entry.path) {
			return rule.name, true
		}
	}

	return "", false
}

// PathRuleConfig exposes the internal rule configuration structure for tests.
type PathRuleConfig = pathRuleConfig

// PathRules exposes the compiled rule type for tests.
type PathRules = pathRules
