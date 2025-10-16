package metrics

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// pathRuleConfig represents the JSON shape used to configure route normalization rules.
type pathRuleConfig struct {
	Host   string `json:"host"`
	Path   string `json:"path"`
	Route  string `json:"route"`
	Method string `json:"method,omitempty"`
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
	route  string
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

		if cfg.Path == "" {
			return nil, fmt.Errorf("path rule %d: path regex is required", idx)
		}

		if cfg.Route == "" {
			return nil, fmt.Errorf("path rule %d: route is required", idx)
		}

		method := strings.ToUpper(cfg.Method)

		regex, err := regexp.Compile(cfg.Path)
		if err != nil {
			return nil, fmt.Errorf("path rule %d: failed to compile regex: %w", idx, err)
		}

		compiled = append(compiled, compiledRule{
			host:   cfg.Host,
			method: method,
			route:  cfg.Route,
			regex:  regex,
		})
	}

	return &pathRules{
		enabled: true,
		rules:   compiled,
	}, nil
}

// normalize returns the normalized route for the provided entry if any rule matches.
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
			return rule.route, true
		}
	}

	return "", false
}
